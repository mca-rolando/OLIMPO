package networkcontext

import (
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"gorm.io/gorm"
)

var (
	ErrDeviceNotFound     = errors.New("device not found")
	ErrNoWAN              = errors.New("at least one WAN interface is required")
	ErrInvalidInterface   = errors.New("invalid WAN interface")
	ErrDuplicateInterface = errors.New("duplicate WAN interface")
	ErrInvalidWANRole     = errors.New("invalid WAN role")
	ErrInvalidIPAddress   = errors.New("invalid IP address")
	ErrInvalidPublicIPv4  = errors.New("observed public IPv4 must be a public IPv4 address")
	ErrInvalidNetworkName = errors.New("invalid network name")
	ErrInvalidVLAN        = errors.New("invalid VLAN ID")
	ErrInvalidCIDR        = errors.New("invalid IPv4 CIDR")
	ErrInvalidGateway     = errors.New("gateway IPv4 is outside the network CIDR")
)

const (
	WANRolePrimary   = "primary"
	WANRoleSecondary = "secondary"
	WANRoleOther     = "other"

	AddressScopePublic  = "public"
	AddressScopePrivate = "private"
	AddressScopeCGNAT   = "cgnat"
	AddressScopeSpecial = "special"
	AddressScopeUnknown = "unknown"

	NATStateDirect         = "direct"
	NATStateDoubleNAT      = "double_nat"
	NATStateCGNAT          = "cgnat"
	NATStatePublicMismatch = "public_mismatch"
	NATStateUnknown        = "unknown"

	PublicIPSourceAgentProbe = "agent_probe"
	PublicIPSourceServerPeer = "server_peer"
)

type WANInput struct {
	InterfaceName string
	Role          string
	DefaultRoute  bool
	IPv4          string
	GatewayIPv4   string
	IPv6          string
	PublicIPv4    string
}

type SegmentInput struct {
	Name        string
	VLANID      *int
	IPv4CIDR    string
	GatewayIPv4 string
	Purpose     string
}

type ReportInput struct {
	WANs     []WANInput
	Networks []SegmentInput
}

type DeviceNetworkContext struct {
	Device   model.Device                   `json:"device"`
	Reported bool                           `json:"reported"`
	Snapshot *model.NetworkIdentitySnapshot `json:"snapshot"`
	WANs     []model.NetworkWAN             `json:"wans"`
	Networks []model.NetworkSegment         `json:"networks"`
}

type Service struct {
	DB  *gorm.DB
	Now func() time.Time
}

func (s *Service) Report(deviceID uint, serverObservedIP string, input ReportInput) (DeviceNetworkContext, error) {
	if err := validateInput(input); err != nil {
		return DeviceNetworkContext{}, err
	}

	var result DeviceNetworkContext
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Where("id = ? AND status = ?", deviceID, "active").First(&device).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeviceNotFound
			}
			return err
		}

		now := s.now()
		observed := strings.TrimSpace(serverObservedIP)
		var snapshot model.NetworkIdentitySnapshot
		err := tx.Where("device_id = ?", deviceID).First(&snapshot).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			snapshot = model.NetworkIdentitySnapshot{DeviceID: deviceID}
		}
		snapshot.ReportedAt = now
		snapshot.ServerObservedIP = observed
		if snapshot.ID == 0 {
			if err := tx.Create(&snapshot).Error; err != nil {
				return fmt.Errorf("create network identity snapshot: %w", err)
			}
		} else if err := tx.Save(&snapshot).Error; err != nil {
			return fmt.Errorf("update network identity snapshot: %w", err)
		}

		if err := tx.Unscoped().Where("device_id = ?", deviceID).Delete(&model.NetworkWAN{}).Error; err != nil {
			return fmt.Errorf("replace WAN identity: %w", err)
		}
		if err := tx.Unscoped().Where("device_id = ?", deviceID).Delete(&model.NetworkSegment{}).Error; err != nil {
			return fmt.Errorf("replace network segments: %w", err)
		}

		wans := make([]model.NetworkWAN, 0, len(input.WANs))
		for _, in := range input.WANs {
			wan, err := buildWAN(deviceID, observed, in)
			if err != nil {
				return err
			}
			if err := tx.Create(&wan).Error; err != nil {
				return fmt.Errorf("create WAN identity: %w", err)
			}
			wans = append(wans, wan)
		}

		networks := make([]model.NetworkSegment, 0, len(input.Networks))
		for _, in := range input.Networks {
			network := model.NetworkSegment{
				DeviceID:    deviceID,
				Name:        strings.TrimSpace(in.Name),
				VLANID:      in.VLANID,
				IPv4CIDR:    strings.TrimSpace(in.IPv4CIDR),
				GatewayIPv4: strings.TrimSpace(in.GatewayIPv4),
				Purpose:     strings.TrimSpace(in.Purpose),
			}
			if err := tx.Create(&network).Error; err != nil {
				return fmt.Errorf("create network segment: %w", err)
			}
			networks = append(networks, network)
		}

		result = DeviceNetworkContext{
			Device:   device,
			Reported: true,
			Snapshot: &snapshot,
			WANs:     wans,
			Networks: networks,
		}
		return nil
	})
	if err != nil {
		return DeviceNetworkContext{}, err
	}
	sortContext(&result)
	return result, nil
}

func (s *Service) Context(deviceID uint) (DeviceNetworkContext, error) {
	var device model.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeviceNetworkContext{}, ErrDeviceNotFound
		}
		return DeviceNetworkContext{}, err
	}

	var snapshot model.NetworkIdentitySnapshot
	if err := s.DB.Where("device_id = ?", deviceID).First(&snapshot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeviceNetworkContext{Device: device, Reported: false, WANs: []model.NetworkWAN{}, Networks: []model.NetworkSegment{}}, nil
		}
		return DeviceNetworkContext{}, err
	}

	var wans []model.NetworkWAN
	if err := s.DB.Where("device_id = ?", deviceID).Order("default_route desc, interface_name asc").Find(&wans).Error; err != nil {
		return DeviceNetworkContext{}, err
	}
	var networks []model.NetworkSegment
	if err := s.DB.Where("device_id = ?", deviceID).Order("name asc").Find(&networks).Error; err != nil {
		return DeviceNetworkContext{}, err
	}
	return DeviceNetworkContext{Device: device, Reported: true, Snapshot: &snapshot, WANs: wans, Networks: networks}, nil
}

// ListContexts performs four bounded queries and groups child rows in memory,
// avoiding an N+1 query pattern for fleet/ARGUS consumers.
func (s *Service) ListContexts() ([]DeviceNetworkContext, error) {
	var devices []model.Device
	if err := s.DB.Order("name asc").Find(&devices).Error; err != nil {
		return nil, err
	}
	var snapshots []model.NetworkIdentitySnapshot
	if err := s.DB.Find(&snapshots).Error; err != nil {
		return nil, err
	}
	var wans []model.NetworkWAN
	if err := s.DB.Order("default_route desc, interface_name asc").Find(&wans).Error; err != nil {
		return nil, err
	}
	var networks []model.NetworkSegment
	if err := s.DB.Order("name asc").Find(&networks).Error; err != nil {
		return nil, err
	}

	snapshotByDevice := make(map[uint]*model.NetworkIdentitySnapshot, len(snapshots))
	for i := range snapshots {
		snapshotByDevice[snapshots[i].DeviceID] = &snapshots[i]
	}
	wansByDevice := make(map[uint][]model.NetworkWAN)
	for _, wan := range wans {
		wansByDevice[wan.DeviceID] = append(wansByDevice[wan.DeviceID], wan)
	}
	networksByDevice := make(map[uint][]model.NetworkSegment)
	for _, network := range networks {
		networksByDevice[network.DeviceID] = append(networksByDevice[network.DeviceID], network)
	}

	result := make([]DeviceNetworkContext, 0, len(devices))
	for _, device := range devices {
		snapshot := snapshotByDevice[device.ID]
		deviceWANs := wansByDevice[device.ID]
		deviceNetworks := networksByDevice[device.ID]
		if deviceWANs == nil {
			deviceWANs = []model.NetworkWAN{}
		}
		if deviceNetworks == nil {
			deviceNetworks = []model.NetworkSegment{}
		}
		result = append(result, DeviceNetworkContext{
			Device:   device,
			Reported: snapshot != nil,
			Snapshot: snapshot,
			WANs:     deviceWANs,
			Networks: deviceNetworks,
		})
	}
	return result, nil
}

func buildWAN(deviceID uint, serverObservedIP string, in WANInput) (model.NetworkWAN, error) {
	local := strings.TrimSpace(in.IPv4)
	observed := strings.TrimSpace(in.PublicIPv4)
	source := ""
	if observed != "" {
		source = PublicIPSourceAgentProbe
	} else if in.DefaultRoute && isPublicIPv4String(serverObservedIP) {
		observed = strings.TrimSpace(serverObservedIP)
		source = PublicIPSourceServerPeer
	}

	scope := classifyIPv4(local)
	wan := model.NetworkWAN{
		DeviceID:           deviceID,
		InterfaceName:      strings.TrimSpace(in.InterfaceName),
		Role:               normalizedRole(in.Role),
		DefaultRoute:       in.DefaultRoute,
		IPv4:               local,
		GatewayIPv4:        strings.TrimSpace(in.GatewayIPv4),
		IPv6:               strings.TrimSpace(in.IPv6),
		ObservedPublicIPv4: observed,
		PublicIPSource:     source,
		AddressScope:       scope,
		NATState:           NATStateUnknown,
	}

	if scope == AddressScopeCGNAT {
		wan.NATState = NATStateCGNAT
		wan.UpstreamNAT = true
		wan.CGNAT = true
	}

	if local == "" || observed == "" {
		return wan, nil
	}
	localAddr, localOK := parseIPv4(local)
	observedAddr, observedOK := parseIPv4(observed)
	if !localOK || !observedOK {
		return wan, nil
	}
	match := localAddr == observedAddr
	wan.PublicIPMatch = &match

	switch {
	case scope == AddressScopePublic && match:
		wan.NATState = NATStateDirect
	case scope == AddressScopePublic && !match:
		wan.NATState = NATStatePublicMismatch
	case scope == AddressScopePrivate && isPublicIPv4(observedAddr):
		wan.NATState = NATStateDoubleNAT
		wan.UpstreamNAT = true
		wan.DoubleNAT = true
	case scope == AddressScopeCGNAT:
		wan.NATState = NATStateCGNAT
		wan.UpstreamNAT = true
		wan.CGNAT = true
	}
	return wan, nil
}

func validateInput(input ReportInput) error {
	if len(input.WANs) == 0 {
		return ErrNoWAN
	}
	seenInterfaces := make(map[string]struct{}, len(input.WANs))
	for _, wan := range input.WANs {
		name := strings.TrimSpace(wan.InterfaceName)
		if name == "" {
			return ErrInvalidInterface
		}
		key := strings.ToLower(name)
		if _, exists := seenInterfaces[key]; exists {
			return ErrDuplicateInterface
		}
		seenInterfaces[key] = struct{}{}
		if role := normalizedRole(wan.Role); role != WANRolePrimary && role != WANRoleSecondary && role != WANRoleOther {
			return ErrInvalidWANRole
		}
		for _, value := range []string{wan.IPv4, wan.GatewayIPv4} {
			if strings.TrimSpace(value) != "" {
				addr, err := netip.ParseAddr(strings.TrimSpace(value))
				if err != nil || !addr.Is4() {
					return ErrInvalidIPAddress
				}
			}
		}
		if strings.TrimSpace(wan.IPv6) != "" {
			addr, err := netip.ParseAddr(strings.TrimSpace(wan.IPv6))
			if err != nil || !addr.Is6() {
				return ErrInvalidIPAddress
			}
		}
		if strings.TrimSpace(wan.PublicIPv4) != "" && !isPublicIPv4String(wan.PublicIPv4) {
			return ErrInvalidPublicIPv4
		}
	}

	for _, network := range input.Networks {
		if strings.TrimSpace(network.Name) == "" {
			return ErrInvalidNetworkName
		}
		if network.VLANID != nil && (*network.VLANID < 1 || *network.VLANID > 4094) {
			return ErrInvalidVLAN
		}
		cidr := strings.TrimSpace(network.IPv4CIDR)
		gateway := strings.TrimSpace(network.GatewayIPv4)
		if cidr == "" && gateway == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil || !prefix.Addr().Is4() {
			return ErrInvalidCIDR
		}
		prefix = prefix.Masked()
		if gateway != "" {
			gatewayAddr, err := netip.ParseAddr(gateway)
			if err != nil || !gatewayAddr.Is4() {
				return ErrInvalidIPAddress
			}
			if !prefix.Contains(gatewayAddr) {
				return ErrInvalidGateway
			}
		}
	}
	return nil
}

func normalizedRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return WANRoleOther
	}
	return role
}

func classifyIPv4(value string) string {
	addr, ok := parseIPv4(strings.TrimSpace(value))
	if !ok {
		if strings.TrimSpace(value) == "" {
			return AddressScopeUnknown
		}
		return AddressScopeSpecial
	}
	if isCGNAT(addr) {
		return AddressScopeCGNAT
	}
	if addr.IsPrivate() {
		return AddressScopePrivate
	}
	if isPublicIPv4(addr) {
		return AddressScopePublic
	}
	return AddressScopeSpecial
}

func parseIPv4(value string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil || !addr.Is4() {
		return netip.Addr{}, false
	}
	return addr, true
}

func isPublicIPv4String(value string) bool {
	addr, ok := parseIPv4(value)
	return ok && isPublicIPv4(addr)
}

func isPublicIPv4(addr netip.Addr) bool {
	if !addr.Is4() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsMulticast() || addr.IsUnspecified() || isCGNAT(addr) {
		return false
	}
	for _, prefix := range specialIPv4Prefixes() {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

func isCGNAT(addr netip.Addr) bool {
	return netip.MustParsePrefix("100.64.0.0/10").Contains(addr)
}

func specialIPv4Prefixes() []netip.Prefix {
	return []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"),
	}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Stable ordering makes API responses and tests deterministic even after an
// atomic delete/recreate of the current snapshot rows.
func sortContext(context *DeviceNetworkContext) {
	sort.Slice(context.WANs, func(i, j int) bool {
		if context.WANs[i].DefaultRoute != context.WANs[j].DefaultRoute {
			return context.WANs[i].DefaultRoute
		}
		return context.WANs[i].InterfaceName < context.WANs[j].InterfaceName
	})
	sort.Slice(context.Networks, func(i, j int) bool { return context.Networks[i].Name < context.Networks[j].Name })
}
