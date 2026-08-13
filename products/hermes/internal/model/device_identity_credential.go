package model

import (
	"time"

	"gorm.io/gorm"
)

type DeviceIdentityCredential struct {
	gorm.Model

	DeviceID     uint   `gorm:"index;not null" json:"device_id"`
	CredentialID string `gorm:"uniqueIndex;not null" json:"credential_id"`
	SecretHash   string `gorm:"not null" json:"-"`

	Status string `gorm:"index;not null" json:"status"`

	ActivatedAt *time.Time `json:"activated_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`

	LastUsedAt *time.Time `json:"last_used_at"`
	LastUsedIP string     `json:"last_used_ip"`
}
