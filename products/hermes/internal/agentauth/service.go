package agentauth

import (
	"errors"
	"fmt"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
	"gorm.io/gorm"
)

var (
	ErrBadAuth                = errors.New("bad agent authentication")
	ErrDeviceNotFound         = errors.New("device not found")
	ErrActiveCredentialExists = errors.New("active agent credential already exists")
	ErrCredentialNotFound     = errors.New("agent credential not found")
)

type Context struct {
	Device     model.Device
	Credential model.DeviceIdentityCredential
}

type Service struct {
	DB  *gorm.DB
	Now func() time.Time
}

func (s *Service) IssueCredential(deviceID uint) (model.DeviceIdentityCredential, security.GeneratedKey, error) {
	var credential model.DeviceIdentityCredential
	var generated security.GeneratedKey

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		credential, generated, err = s.IssueCredentialTx(tx, deviceID)
		return err
	})
	if err != nil {
		return model.DeviceIdentityCredential{}, security.GeneratedKey{}, err
	}

	return credential, generated, nil
}

// IssueCredentialTx creates a Device identity credential using an existing
// transaction. The caller owns transaction commit/rollback.
func (s *Service) IssueCredentialTx(tx *gorm.DB, deviceID uint) (model.DeviceIdentityCredential, security.GeneratedKey, error) {
	var device model.Device
	if err := tx.Where("id = ? AND status = ?", deviceID, "active").First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeviceIdentityCredential{}, security.GeneratedKey{}, ErrDeviceNotFound
		}
		return model.DeviceIdentityCredential{}, security.GeneratedKey{}, err
	}

	var activeCount int64
	if err := tx.Model(&model.DeviceIdentityCredential{}).
		Where("device_id = ? AND status = ?", deviceID, model.CredentialStatusActive).
		Count(&activeCount).Error; err != nil {
		return model.DeviceIdentityCredential{}, security.GeneratedKey{}, err
	}
	if activeCount != 0 {
		return model.DeviceIdentityCredential{}, security.GeneratedKey{}, ErrActiveCredentialExists
	}

	key, err := security.GenerateAgentKey()
	if err != nil {
		return model.DeviceIdentityCredential{}, security.GeneratedKey{}, err
	}

	now := s.now()
	credential := model.DeviceIdentityCredential{
		DeviceID:     deviceID,
		CredentialID: key.ID,
		SecretHash:   key.Hash,
		Status:       model.CredentialStatusActive,
		ActivatedAt:  &now,
	}
	if err := tx.Create(&credential).Error; err != nil {
		return model.DeviceIdentityCredential{}, security.GeneratedKey{}, fmt.Errorf("create agent credential: %w", err)
	}

	return credential, key, nil
}

func (s *Service) Authenticate(apiKey, callerIP string) (Context, error) {
	credentialID, ok := security.ParseAgentKeyID(apiKey)
	if !ok {
		return Context{}, ErrBadAuth
	}

	var credential model.DeviceIdentityCredential
	if err := s.DB.Where(
		"credential_id = ? AND status = ?",
		credentialID,
		model.CredentialStatusActive,
	).First(&credential).Error; err != nil {
		return Context{}, ErrBadAuth
	}

	now := s.now()
	if credential.ExpiresAt != nil && !credential.ExpiresAt.After(now) {
		return Context{}, ErrBadAuth
	}
	if !security.VerifyAPIKey(apiKey, credential.SecretHash) {
		return Context{}, ErrBadAuth
	}

	var device model.Device
	if err := s.DB.Where("id = ? AND status = ?", credential.DeviceID, "active").First(&device).Error; err != nil {
		return Context{}, ErrBadAuth
	}

	credential.LastUsedAt = &now
	credential.LastUsedIP = callerIP
	if err := s.DB.Model(&credential).Updates(map[string]any{
		"last_used_at": now,
		"last_used_ip": callerIP,
	}).Error; err != nil {
		return Context{}, fmt.Errorf("update agent credential usage: %w", err)
	}

	return Context{Device: device, Credential: credential}, nil
}

func (s *Service) RevokeCredential(deviceID, credentialID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		return s.RevokeCredentialTx(tx, deviceID, credentialID)
	})
}

// RevokeCredentialTx revokes an active Device identity credential inside an
// existing transaction. The caller owns transaction commit/rollback.
func (s *Service) RevokeCredentialTx(tx *gorm.DB, deviceID, credentialID uint) error {
	now := s.now()
	updated := tx.Model(&model.DeviceIdentityCredential{}).
		Where(
			"id = ? AND device_id = ? AND status = ?",
			credentialID,
			deviceID,
			model.CredentialStatusActive,
		).
		Updates(map[string]any{
			"status":     model.CredentialStatusRevoked,
			"revoked_at": now,
		})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return ErrCredentialNotFound
	}
	return nil
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
