package ddns

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/dns"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
	"gorm.io/gorm"
)

var (
	ErrBadAuth = errors.New("bad authentication")
	ErrNotFQDN = errors.New("hostname is not a managed FQDN")
	ErrBadIP   = errors.New("invalid IP address")
)

type Service struct {
	DB               *gorm.DB
	DNS              dns.Updater
	AllowedDomains   []string
	DefaultTTL       int
	AllowWildcard    bool
	AutocreatePolicy string
}

type AuthContext struct {
	Device     model.Device
	Credential model.DDNSCredential
}

type Result struct {
	Code    string
	IP      string
	Changed bool
	Created bool
	Host    model.Host
}

func (s *Service) Authenticate(username, apiKey, callerIP string) (AuthContext, error) {
	var device model.Device
	if err := s.DB.Where("name = ? AND status = ?", username, "active").First(&device).Error; err != nil {
		return AuthContext{}, ErrBadAuth
	}

	var creds []model.DDNSCredential
	now := time.Now().UTC()
	if err := s.DB.Where("device_id = ? AND status IN ?", device.ID, []string{model.CredentialStatusActive, model.CredentialStatusGrace}).Find(&creds).Error; err != nil {
		return AuthContext{}, ErrBadAuth
	}

	for _, cred := range creds {
		if cred.ExpiresAt != nil && !cred.ExpiresAt.After(now) {
			continue
		}
		if cred.Status == model.CredentialStatusGrace && (cred.GraceUntil == nil || !cred.GraceUntil.After(now)) {
			continue
		}
		if !security.VerifyAPIKey(apiKey, cred.SecretHash) {
			continue
		}

		cred.LastUsedAt = &now
		cred.LastUsedIP = callerIP
		if err := s.DB.Model(&cred).Updates(map[string]any{"last_used_at": now, "last_used_ip": callerIP}).Error; err != nil {
			return AuthContext{}, fmt.Errorf("update credential usage: %w", err)
		}

		return AuthContext{Device: device, Credential: cred}, nil
	}

	return AuthContext{}, ErrBadAuth
}

func (s *Service) Update(auth AuthContext, fqdn, requestedIP, callerIP, userAgent string) (Result, error) {
	requestedAt := time.Now().UTC()
	hostLabel, zone, err := s.splitManagedFQDN(fqdn)
	if err != nil {
		s.writeLog(auth, nil, "reject", "error", "notfqdn", err.Error(), requestedIP, callerIP, userAgent, requestedAt)
		return Result{}, err
	}

	ip := strings.TrimSpace(requestedIP)
	if ip == "" {
		ip = callerIP
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		s.writeLog(auth, nil, "reject", "error", "badagent", ErrBadIP.Error(), ip, callerIP, userAgent, requestedAt)
		return Result{}, ErrBadIP
	}
	recordType := "AAAA"
	if parsed.To4() != nil {
		recordType = "A"
	}
	ip = parsed.String()

	var domain model.Domain
	if err := s.DB.Where("name = ? AND enabled = ?", zone, true).First(&domain).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			domain = model.Domain{Name: zone, DefaultTTL: s.DefaultTTL, Enabled: true, Wildcard: s.AllowWildcard}
			if err := s.DB.Create(&domain).Error; err != nil {
				return Result{}, err
			}
		} else {
			return Result{}, err
		}
	}

	var host model.Host
	err = s.DB.Preload("Domain").Where("domain_id = ? AND hostname = ?", domain.ID, hostLabel).First(&host).Error
	created := false
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if !s.canAutocreate(auth.Device.Name, hostLabel) {
			err := fmt.Errorf("device %s is not authorized to auto-create hostname %s", auth.Device.Name, fqdn)
			s.writeLog(auth, nil, "reject", "error", "badauth", err.Error(), ip, callerIP, userAgent, requestedAt)
			return Result{}, ErrBadAuth
		}
		host = model.Host{
			DomainID:   domain.ID,
			DeviceID:   auth.Device.ID,
			Hostname:   hostLabel,
			IPAddress:  "",
			RecordType: recordType,
			TTL:        domain.DefaultTTL,
			Status:     "active",
			Domain:     domain,
		}
		created = true
	} else if err != nil {
		return Result{}, err
	} else if host.DeviceID != auth.Device.ID {
		err := fmt.Errorf("hostname %s belongs to another device", fqdn)
		s.writeLog(auth, &host, "reject", "error", "badauth", err.Error(), ip, callerIP, userAgent, requestedAt)
		return Result{}, ErrBadAuth
	}

	if !created && host.IPAddress == ip && host.RecordType == recordType {
		now := time.Now().UTC()
		host.LastUpdate = now
		_ = s.DB.Model(&auth.Device).Updates(map[string]any{"last_seen_at": now, "last_ip": ip}).Error
		_ = s.DB.Model(&host).Update("last_update", now).Error
		s.writeLog(auth, &host, "no_change", "success", "nochg", "IP address unchanged", ip, callerIP, userAgent, requestedAt)
		return Result{Code: "nochg", IP: ip, Changed: false, Host: host}, nil
	}

	if err := s.DNS.Upsert(hostLabel, zone, recordType, ip, host.TTL, domain.Wildcard); err != nil {
		s.writeLog(auth, &host, "dns_update", "error", "dnserr", err.Error(), ip, callerIP, userAgent, requestedAt)
		return Result{}, fmt.Errorf("dns update: %w", err)
	}

	now := time.Now().UTC()
	host.IPAddress = ip
	host.RecordType = recordType
	host.LastUpdate = now
	host.Domain = domain
	if created {
		if err := s.DB.Create(&host).Error; err != nil {
			return Result{}, err
		}
	} else if err := s.DB.Save(&host).Error; err != nil {
		return Result{}, err
	}
	_ = s.DB.Model(&auth.Device).Updates(map[string]any{"last_seen_at": now, "last_ip": ip}).Error
	operation := "update"
	if created {
		operation = "create"
	}
	s.writeLog(auth, &host, operation, "success", "good", "DNS record updated", ip, callerIP, userAgent, requestedAt)
	return Result{Code: "good", IP: ip, Changed: true, Created: created, Host: host}, nil
}

func (s *Service) canAutocreate(deviceName, hostLabel string) bool {
	if s.AutocreatePolicy == "any" {
		return true
	}
	base := strings.ToLower(strings.TrimSpace(deviceName))
	label := strings.ToLower(strings.TrimSpace(hostLabel))
	return label == base || strings.HasPrefix(label, base+"-")
}

func (s *Service) splitManagedFQDN(fqdn string) (string, string, error) {
	fqdn = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(fqdn)), ".")
	for _, zone := range s.AllowedDomains {
		zone = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(zone)), ".")
		suffix := "." + zone
		if strings.HasSuffix(fqdn, suffix) {
			label := strings.TrimSuffix(fqdn, suffix)
			if label == "" || strings.Contains(label, ".") {
				return "", "", ErrNotFQDN
			}
			return label, zone, nil
		}
	}
	return "", "", ErrNotFQDN
}

func (s *Service) writeLog(auth AuthContext, host *model.Host, operation, status, response, message, sentIP, callerIP, userAgent string, requestedAt time.Time) {
	entry := model.UpdateLog{
		DeviceID:     auth.Device.ID,
		CredentialID: &auth.Credential.ID,
		Operation:    operation,
		Status:       status,
		ResponseCode: response,
		Message:      message,
		SentIP:       sentIP,
		CallerIP:     callerIP,
		UserAgent:    userAgent,
		RequestedAt:  requestedAt,
		CompletedAt:  time.Now().UTC(),
	}
	if host != nil && host.ID != 0 {
		entry.HostID = &host.ID
	}
	_ = s.DB.Create(&entry).Error
}
