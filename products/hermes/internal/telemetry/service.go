package telemetry

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"gorm.io/gorm"
)

var (
	ErrDeviceNotFound       = errors.New("device not found")
	ErrInvalidMemoryMetrics = errors.New("memory available bytes cannot exceed memory total bytes")
	ErrInvalidDiskMetrics   = errors.New("disk available bytes cannot exceed disk total bytes")
	ErrInvalidCPUCount      = errors.New("invalid cpu count")
	ErrInvalidLoad          = errors.New("invalid load average")
	ErrInvalidCounter       = errors.New("telemetry counter exceeds supported storage range")
)

const (
	DefaultHeartbeatIntervalSeconds = 60
	DefaultOnlineThresholdSeconds   = 180

	AgentStateNeverSeen = "never_seen"
	AgentStateOnline    = "online"
	AgentStateOffline   = "offline"
)

// HeartbeatInput uses pointers so a presence-only heartbeat can update the
// heartbeat timestamp without erasing telemetry fields reported previously.
type HeartbeatInput struct {
	AgentVersion         *string
	SystemHostname       *string
	Platform             *string
	Architecture         *string
	OSVersion            *string
	KernelVersion        *string
	FirmwareVersion      *string
	BootID               *string
	UptimeSeconds        *uint64
	CPUCount             *int
	Load1                *float64
	Load5                *float64
	Load15               *float64
	MemoryTotalBytes     *uint64
	MemoryAvailableBytes *uint64
	DiskTotalBytes       *uint64
	DiskAvailableBytes   *uint64
}

type AgentStatus struct {
	Device              model.Device                  `json:"device"`
	State               string                        `json:"state"`
	Online              bool                          `json:"online"`
	HeartbeatAgeSeconds *int64                        `json:"heartbeat_age_seconds"`
	Telemetry           *model.AgentTelemetrySnapshot `json:"telemetry"`
}

type Service struct {
	DB  *gorm.DB
	Now func() time.Time
}

func (s *Service) Report(deviceID uint, callerIP string, input HeartbeatInput) (model.AgentTelemetrySnapshot, error) {
	if err := validateInput(input); err != nil {
		return model.AgentTelemetrySnapshot{}, err
	}

	var result model.AgentTelemetrySnapshot

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Where("id = ? AND status = ?", deviceID, "active").First(&device).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeviceNotFound
			}
			return err
		}

		var snapshot model.AgentTelemetrySnapshot
		newSnapshot := false
		if err := tx.Where("device_id = ?", deviceID).First(&snapshot).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			newSnapshot = true
			snapshot = model.AgentTelemetrySnapshot{
				DeviceID:     deviceID,
				AgentVersion: device.AgentVersion,
			}
		}

		now := s.now()
		snapshot.LastHeartbeatAt = now
		snapshot.LastIP = strings.TrimSpace(callerIP)
		applyHeartbeat(&snapshot, input)
		if err := validateSnapshot(snapshot); err != nil {
			return err
		}

		if newSnapshot {
			if err := tx.Create(&snapshot).Error; err != nil {
				return fmt.Errorf("create agent telemetry snapshot: %w", err)
			}
		} else if err := tx.Save(&snapshot).Error; err != nil {
			return fmt.Errorf("update agent telemetry snapshot: %w", err)
		}

		deviceUpdates := map[string]any{
			"last_seen_at": now,
			"last_ip":      snapshot.LastIP,
		}
		if input.AgentVersion != nil {
			version := strings.TrimSpace(*input.AgentVersion)
			if version != "" {
				deviceUpdates["agent_version"] = version
			}
		}
		if err := tx.Model(&model.Device{}).Where("id = ?", deviceID).Updates(deviceUpdates).Error; err != nil {
			return fmt.Errorf("update device heartbeat presence: %w", err)
		}

		result = snapshot
		return nil
	})
	if err != nil {
		return model.AgentTelemetrySnapshot{}, err
	}

	return result, nil
}

func (s *Service) Status(deviceID uint) (AgentStatus, error) {
	var device model.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AgentStatus{}, ErrDeviceNotFound
		}
		return AgentStatus{}, err
	}

	var snapshot model.AgentTelemetrySnapshot
	if err := s.DB.Where("device_id = ?", deviceID).First(&snapshot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AgentStatus{Device: device, State: AgentStateNeverSeen, Online: false}, nil
		}
		return AgentStatus{}, err
	}

	return s.statusFrom(device, &snapshot), nil
}

// ListStatuses uses one Device query and one snapshot query so fleet dashboards
// do not need an N+1 request/query pattern for large UDM estates.
func (s *Service) ListStatuses() ([]AgentStatus, error) {
	var devices []model.Device
	if err := s.DB.Order("name asc").Find(&devices).Error; err != nil {
		return nil, err
	}

	var snapshots []model.AgentTelemetrySnapshot
	if err := s.DB.Find(&snapshots).Error; err != nil {
		return nil, err
	}
	byDevice := make(map[uint]*model.AgentTelemetrySnapshot, len(snapshots))
	for i := range snapshots {
		byDevice[snapshots[i].DeviceID] = &snapshots[i]
	}

	statuses := make([]AgentStatus, 0, len(devices))
	for _, device := range devices {
		statuses = append(statuses, s.statusFrom(device, byDevice[device.ID]))
	}
	return statuses, nil
}

func (s *Service) statusFrom(device model.Device, snapshot *model.AgentTelemetrySnapshot) AgentStatus {
	if snapshot == nil {
		return AgentStatus{Device: device, State: AgentStateNeverSeen, Online: false}
	}

	age := s.now().Sub(snapshot.LastHeartbeatAt)
	if age < 0 {
		age = 0
	}
	ageSeconds := int64(age / time.Second)
	online := age <= time.Duration(DefaultOnlineThresholdSeconds)*time.Second
	state := AgentStateOffline
	if online {
		state = AgentStateOnline
	}
	return AgentStatus{
		Device:              device,
		State:               state,
		Online:              online,
		HeartbeatAgeSeconds: &ageSeconds,
		Telemetry:           snapshot,
	}
}

func applyHeartbeat(snapshot *model.AgentTelemetrySnapshot, input HeartbeatInput) {
	applyString := func(dst *string, src *string) {
		if src != nil {
			*dst = strings.TrimSpace(*src)
		}
	}
	applyString(&snapshot.AgentVersion, input.AgentVersion)
	applyString(&snapshot.SystemHostname, input.SystemHostname)
	applyString(&snapshot.Platform, input.Platform)
	applyString(&snapshot.Architecture, input.Architecture)
	applyString(&snapshot.OSVersion, input.OSVersion)
	applyString(&snapshot.KernelVersion, input.KernelVersion)
	applyString(&snapshot.FirmwareVersion, input.FirmwareVersion)
	applyString(&snapshot.BootID, input.BootID)

	if input.UptimeSeconds != nil {
		snapshot.UptimeSeconds = *input.UptimeSeconds
	}
	if input.CPUCount != nil {
		snapshot.CPUCount = *input.CPUCount
	}
	if input.Load1 != nil {
		snapshot.Load1 = *input.Load1
	}
	if input.Load5 != nil {
		snapshot.Load5 = *input.Load5
	}
	if input.Load15 != nil {
		snapshot.Load15 = *input.Load15
	}
	if input.MemoryTotalBytes != nil {
		snapshot.MemoryTotalBytes = *input.MemoryTotalBytes
	}
	if input.MemoryAvailableBytes != nil {
		snapshot.MemoryAvailableBytes = *input.MemoryAvailableBytes
	}
	if input.DiskTotalBytes != nil {
		snapshot.DiskTotalBytes = *input.DiskTotalBytes
	}
	if input.DiskAvailableBytes != nil {
		snapshot.DiskAvailableBytes = *input.DiskAvailableBytes
	}
}

func validateInput(input HeartbeatInput) error {
	if input.CPUCount != nil && (*input.CPUCount < 1 || *input.CPUCount > 4096) {
		return ErrInvalidCPUCount
	}
	for _, load := range []*float64{input.Load1, input.Load5, input.Load15} {
		if load != nil && (math.IsNaN(*load) || math.IsInf(*load, 0) || *load < 0 || *load > 1000000) {
			return ErrInvalidLoad
		}
	}
	for _, counter := range []*uint64{
		input.UptimeSeconds,
		input.MemoryTotalBytes,
		input.MemoryAvailableBytes,
		input.DiskTotalBytes,
		input.DiskAvailableBytes,
	} {
		if counter != nil && *counter > math.MaxInt64 {
			return ErrInvalidCounter
		}
	}
	return nil
}

func validateSnapshot(snapshot model.AgentTelemetrySnapshot) error {
	if snapshot.CPUCount < 0 || snapshot.CPUCount > 4096 {
		return ErrInvalidCPUCount
	}
	if snapshot.Load1 < 0 || snapshot.Load5 < 0 || snapshot.Load15 < 0 ||
		snapshot.Load1 > 1000000 || snapshot.Load5 > 1000000 || snapshot.Load15 > 1000000 {
		return ErrInvalidLoad
	}
	if snapshot.MemoryTotalBytes > 0 && snapshot.MemoryAvailableBytes > snapshot.MemoryTotalBytes {
		return ErrInvalidMemoryMetrics
	}
	if snapshot.DiskTotalBytes > 0 && snapshot.DiskAvailableBytes > snapshot.DiskTotalBytes {
		return ErrInvalidDiskMetrics
	}
	return nil
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
