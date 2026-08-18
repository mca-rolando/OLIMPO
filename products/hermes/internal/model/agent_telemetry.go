package model

import (
	"time"

	"gorm.io/gorm"
)

// AgentTelemetrySnapshot stores the latest heartbeat/telemetry reported by a
// Hermes UDM Agent. It is intentionally a one-row-per-Device snapshot rather
// than an unbounded time-series table.
type AgentTelemetrySnapshot struct {
	gorm.Model
	DeviceID             uint      `gorm:"uniqueIndex;not null" json:"device_id"`
	LastHeartbeatAt      time.Time `gorm:"index;not null" json:"last_heartbeat_at"`
	LastIP               string    `json:"last_ip"`
	AgentVersion         string    `json:"agent_version"`
	SystemHostname       string    `json:"system_hostname"`
	Platform             string    `json:"platform"`
	Architecture         string    `json:"architecture"`
	OSVersion            string    `json:"os_version"`
	KernelVersion        string    `json:"kernel_version"`
	FirmwareVersion      string    `json:"firmware_version"`
	BootID               string    `json:"boot_id"`
	UptimeSeconds        uint64    `json:"uptime_seconds"`
	CPUCount             int       `json:"cpu_count"`
	Load1                float64   `json:"load_1"`
	Load5                float64   `json:"load_5"`
	Load15               float64   `json:"load_15"`
	MemoryTotalBytes     uint64    `json:"memory_total_bytes"`
	MemoryAvailableBytes uint64    `json:"memory_available_bytes"`
	DiskTotalBytes       uint64    `json:"disk_total_bytes"`
	DiskAvailableBytes   uint64    `json:"disk_available_bytes"`
}
