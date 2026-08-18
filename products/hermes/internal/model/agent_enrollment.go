package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	EnrollmentStatusPending   = "pending"
	EnrollmentStatusIssued    = "issued"
	EnrollmentStatusCompleted = "completed"
	EnrollmentStatusRevoked   = "revoked"
	EnrollmentStatusExpired   = "expired"
)

// AgentEnrollment represents the short-lived, one-time bootstrap authorization
// used to establish a permanent DeviceIdentityCredential on a Device.
type AgentEnrollment struct {
	gorm.Model

	DeviceID   uint   `gorm:"index;not null" json:"device_id"`
	TokenID    string `gorm:"uniqueIndex;not null" json:"token_id"`
	SecretHash string `gorm:"not null" json:"-"`
	Status     string `gorm:"index;not null" json:"status"`

	ExpiresAt   time.Time  `gorm:"index;not null" json:"expires_at"`
	IssuedAt    *time.Time `json:"issued_at"`
	CompletedAt *time.Time `json:"completed_at"`
	RevokedAt   *time.Time `json:"revoked_at"`

	UsedIP            string `json:"used_ip"`
	AgentCredentialID *uint  `gorm:"index" json:"agent_credential_id"`
}
