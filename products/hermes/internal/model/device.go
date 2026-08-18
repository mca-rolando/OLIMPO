package model

import (
	"gorm.io/gorm"
	"time"
)

type Device struct {
	gorm.Model
	Name         string     `gorm:"uniqueIndex;not null" json:"name" validate:"required,min=3,max=128"`
	DisplayName  string     `json:"display_name"`
	Type         string     `json:"type"`
	Status       string     `gorm:"not null;default:active" json:"status"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	LastIP       string     `json:"last_ip"`
	AgentVersion string     `json:"agent_version"`
}
