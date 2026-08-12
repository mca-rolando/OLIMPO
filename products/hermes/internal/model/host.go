package model

import (
	"gorm.io/gorm"
	"time"
)

type Host struct {
	gorm.Model
	DomainID   uint      `gorm:"uniqueIndex:idx_domain_hostname;not null" json:"domain_id"`
	DeviceID   uint      `gorm:"index;not null" json:"device_id"`
	Hostname   string    `gorm:"uniqueIndex:idx_domain_hostname;not null" json:"hostname" validate:"required,hostname"`
	IPAddress  string    `json:"ip_address"`
	RecordType string    `gorm:"not null;default:A" json:"record_type"`
	TTL        int       `gorm:"not null" json:"ttl" validate:"min=20,max=86400"`
	Status     string    `gorm:"not null;default:active" json:"status"`
	LastUpdate time.Time `json:"last_update"`
	Domain     Domain    `json:"domain,omitempty"`
	Device     Device    `json:"device,omitempty"`
}

func (h Host) FQDN() string {
	if h.Domain.Name == "" {
		return h.Hostname
	}
	return h.Hostname + "." + h.Domain.Name
}
