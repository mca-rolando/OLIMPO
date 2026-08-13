package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	CredentialStatusPending = "pending"
	CredentialStatusActive  = "active"
	CredentialStatusGrace   = "grace"
	CredentialStatusRevoked = "revoked"
	CredentialStatusExpired = "expired"
)

type DDNSCredential struct {
	gorm.Model

	DeviceID   uint   `gorm:"index;not null" json:"device_id"`
	KeyID      string `gorm:"uniqueIndex;not null" json:"key_id"`
	SecretHash string `gorm:"not null" json:"-"`

	Status string `gorm:"index;not null" json:"status"`

	ActivatedAt *time.Time `json:"activated_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	GraceUntil  *time.Time `json:"grace_until"`

	LastUsedAt *time.Time `json:"last_used_at"`
	LastUsedIP string     `json:"last_used_ip"`

	ReplacesCredentialID *uint `gorm:"index" json:"replaces_credential_id"`
}
