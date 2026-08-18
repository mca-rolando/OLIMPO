package model

import (
	"gorm.io/gorm"
	"time"
)

type UpdateLog struct {
	gorm.Model
	DeviceID     uint      `gorm:"index" json:"device_id"`
	HostID       *uint     `gorm:"index" json:"host_id"`
	CredentialID *uint     `gorm:"index" json:"credential_id"`
	Operation    string    `json:"operation"`
	Status       string    `gorm:"index" json:"status"`
	ResponseCode string    `json:"response_code"`
	Message      string    `json:"message"`
	SentIP       string    `json:"sent_ip"`
	CallerIP     string    `json:"caller_ip"`
	UserAgent    string    `json:"user_agent"`
	RequestedAt  time.Time `json:"requested_at"`
	CompletedAt  time.Time `json:"completed_at"`
}
