package model

import (
	"gorm.io/gorm"
	"time"
)

type DDNSCredential struct {
	gorm.Model
	DeviceID    uint       `gorm:"index;not null" json:"device_id"`
	KeyID       string     `gorm:"uniqueIndex;not null" json:"key_id"`
	SecretHash  string     `gorm:"not null" json:"-"`
	Status      string     `gorm:"index;not null" json:"status"`
	ActivatedAt time.Time  `json:"activated_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	GraceUntil  *time.Time `json:"grace_until"`
}
