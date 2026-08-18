package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mca-rolando/HermesDDNS/internal/telemetry"
)

type agentHeartbeatRequest struct {
	AgentVersion         *string  `json:"agent_version" validate:"omitempty,max=128"`
	SystemHostname       *string  `json:"system_hostname" validate:"omitempty,max=255"`
	Platform             *string  `json:"platform" validate:"omitempty,max=128"`
	Architecture         *string  `json:"architecture" validate:"omitempty,max=64"`
	OSVersion            *string  `json:"os_version" validate:"omitempty,max=255"`
	KernelVersion        *string  `json:"kernel_version" validate:"omitempty,max=255"`
	FirmwareVersion      *string  `json:"firmware_version" validate:"omitempty,max=255"`
	BootID               *string  `json:"boot_id" validate:"omitempty,max=255"`
	UptimeSeconds        *uint64  `json:"uptime_seconds"`
	CPUCount             *int     `json:"cpu_count" validate:"omitempty,min=1,max=4096"`
	Load1                *float64 `json:"load_1" validate:"omitempty,gte=0,lte=1000000"`
	Load5                *float64 `json:"load_5" validate:"omitempty,gte=0,lte=1000000"`
	Load15               *float64 `json:"load_15" validate:"omitempty,gte=0,lte=1000000"`
	MemoryTotalBytes     *uint64  `json:"memory_total_bytes"`
	MemoryAvailableBytes *uint64  `json:"memory_available_bytes"`
	DiskTotalBytes       *uint64  `json:"disk_total_bytes"`
	DiskAvailableBytes   *uint64  `json:"disk_available_bytes"`
}

func (s *Server) agentHeartbeat(c echo.Context) error {
	auth, ok := agentContext(c)
	if !ok || s.Telemetry == nil {
		return agentUnauthorized(c)
	}

	var req agentHeartbeatRequest
	if err := c.Bind(&req); err != nil && !errors.Is(err, io.EOF) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	snapshot, err := s.Telemetry.Report(auth.Device.ID, s.callerIP(c), telemetry.HeartbeatInput{
		AgentVersion:         req.AgentVersion,
		SystemHostname:       req.SystemHostname,
		Platform:             req.Platform,
		Architecture:         req.Architecture,
		OSVersion:            req.OSVersion,
		KernelVersion:        req.KernelVersion,
		FirmwareVersion:      req.FirmwareVersion,
		BootID:               req.BootID,
		UptimeSeconds:        req.UptimeSeconds,
		CPUCount:             req.CPUCount,
		Load1:                req.Load1,
		Load5:                req.Load5,
		Load15:               req.Load15,
		MemoryTotalBytes:     req.MemoryTotalBytes,
		MemoryAvailableBytes: req.MemoryAvailableBytes,
		DiskTotalBytes:       req.DiskTotalBytes,
		DiskAvailableBytes:   req.DiskAvailableBytes,
	})
	if err != nil {
		return s.telemetryError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status":                   "ok",
		"device_id":                auth.Device.ID,
		"server_time":              snapshot.LastHeartbeatAt,
		"next_heartbeat_seconds":   telemetry.DefaultHeartbeatIntervalSeconds,
		"online_threshold_seconds": telemetry.DefaultOnlineThresholdSeconds,
		"telemetry":                snapshot,
	})
}

func (s *Server) getAgentStatus(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	if s.Telemetry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "telemetry service unavailable"})
	}

	status, err := s.Telemetry.Status(deviceID)
	if err != nil {
		return s.telemetryError(c, err)
	}
	return c.JSON(http.StatusOK, status)
}

func (s *Server) listAgentStatuses(c echo.Context) error {
	if s.Telemetry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "telemetry service unavailable"})
	}
	statuses, err := s.Telemetry.ListStatuses()
	if err != nil {
		return s.telemetryError(c, err)
	}
	return c.JSON(http.StatusOK, statuses)
}

func (s *Server) telemetryError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, telemetry.ErrDeviceNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, telemetry.ErrInvalidMemoryMetrics),
		errors.Is(err, telemetry.ErrInvalidDiskMetrics),
		errors.Is(err, telemetry.ErrInvalidCPUCount),
		errors.Is(err, telemetry.ErrInvalidLoad),
		errors.Is(err, telemetry.ErrInvalidCounter):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}
