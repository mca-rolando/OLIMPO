package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	RotationStatusRequested  = "requested"
	RotationStatusStaged     = "staged"
	RotationStatusValidating = "validating"
	RotationStatusGrace      = "grace"
	RotationStatusCompleted  = "completed"
	RotationStatusRolledBack = "rolled_back"
)

type DDNSCredentialRotation struct {
	gorm.Model

	DeviceID             uint  `gorm:"index;not null" json:"device_id"`
	PreviousCredentialID uint  `gorm:"index;not null" json:"previous_credential_id"`
	NewCredentialID      *uint `gorm:"index" json:"new_credential_id"`

	Status       string `gorm:"index;not null" json:"status"`
	GraceMinutes int    `gorm:"not null" json:"grace_minutes"`

	RequestedAt         time.Time  `json:"requested_at"`
	StagedAt            *time.Time `json:"staged_at"`
	ValidationStartedAt *time.Time `json:"validation_started_at"`
	ConfirmedAt         *time.Time `json:"confirmed_at"`
	GraceUntil          *time.Time `json:"grace_until"`
	CompletedAt         *time.Time `json:"completed_at"`
	RolledBackAt        *time.Time `json:"rolled_back_at"`
	LastError           string     `json:"last_error"`
}
