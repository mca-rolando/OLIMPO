package model

import (
	"time"

	"gorm.io/gorm"
)

// NetworkIdentitySnapshot records when Hermes last received the Device's
// network identity. WANs and LAN segments are stored in normalized child rows
// and replaced atomically on each complete report.
type NetworkIdentitySnapshot struct {
	gorm.Model
	DeviceID         uint      `gorm:"uniqueIndex;not null" json:"device_id"`
	ReportedAt       time.Time `gorm:"index;not null" json:"reported_at"`
	ServerObservedIP string    `json:"server_observed_ip"`
}

// NetworkWAN describes identity-level WAN information. It intentionally does
// not contain health, latency, packet-loss, tunnel, or traffic metrics; those
// belong to ARGUS rather than HermesDDNS.
type NetworkWAN struct {
	gorm.Model
	DeviceID           uint   `gorm:"index;not null" json:"device_id"`
	InterfaceName      string `gorm:"not null" json:"interface_name"`
	Role               string `json:"role"`
	DefaultRoute       bool   `json:"default_route"`
	IPv4               string `json:"ipv4"`
	GatewayIPv4        string `json:"gateway_ipv4"`
	IPv6               string `json:"ipv6"`
	ObservedPublicIPv4 string `json:"observed_public_ipv4"`
	PublicIPSource     string `json:"public_ip_source"`
	AddressScope       string `json:"address_scope"`
	NATState           string `json:"nat_state"`
	UpstreamNAT        bool   `json:"upstream_nat"`
	DoubleNAT          bool   `json:"double_nat"`
	CGNAT              bool   `json:"cgnat"`
	PublicIPMatch      *bool  `json:"public_ip_match"`
}

// NetworkSegment is the deliberately small LAN inventory Hermes keeps for
// identity/context purposes: name, VLAN, IPv4 subnet, gateway, and purpose.
type NetworkSegment struct {
	gorm.Model
	DeviceID    uint   `gorm:"index;not null" json:"device_id"`
	Name        string `gorm:"not null" json:"name"`
	VLANID      *int   `json:"vlan_id"`
	IPv4CIDR    string `json:"ipv4_cidr"`
	GatewayIPv4 string `json:"gateway_ipv4"`
	Purpose     string `json:"purpose"`
}
