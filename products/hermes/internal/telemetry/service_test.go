package telemetry

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/database"
	"github.com/mca-rolando/HermesDDNS/internal/model"
)

func TestReportStoresSnapshotAndPreservesPartialTelemetry(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "telemetry.db"))
	if err != nil {
		t.Fatal(err)
	}
	device := model.Device{Name: "COR-P-TELEMETRY", Status: "active", AgentVersion: "bootstrap-version"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 13, 17, 0, 0, 0, time.UTC)
	service := &Service{DB: db, Now: func() time.Time { return now }}
	version := "26.08-02"
	hostname := "COR-P-TELEMETRY"
	memoryTotal := uint64(8 * 1024 * 1024 * 1024)
	memoryAvailable := uint64(6 * 1024 * 1024 * 1024)
	load := 0.75
	cpuCount := 4

	first, err := service.Report(device.ID, "198.51.100.10", HeartbeatInput{
		AgentVersion:         &version,
		SystemHostname:       &hostname,
		CPUCount:             &cpuCount,
		Load1:                &load,
		MemoryTotalBytes:     &memoryTotal,
		MemoryAvailableBytes: &memoryAvailable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.DeviceID != device.ID || first.LastIP != "198.51.100.10" || first.AgentVersion != version {
		t.Fatalf("unexpected first snapshot: %#v", first)
	}

	now = now.Add(time.Minute)
	zeroLoad := 0.0
	second, err := service.Report(device.ID, "198.51.100.11", HeartbeatInput{Load1: &zeroLoad})
	if err != nil {
		t.Fatal(err)
	}
	if second.Load1 != 0 || second.AgentVersion != version || second.MemoryTotalBytes != memoryTotal {
		t.Fatalf("partial heartbeat erased previous telemetry: %#v", second)
	}
	if second.LastIP != "198.51.100.11" || !second.LastHeartbeatAt.Equal(now) {
		t.Fatalf("heartbeat presence was not refreshed: %#v", second)
	}

	var count int64
	if err := db.Model(&model.AgentTelemetrySnapshot{}).Where("device_id = ?", device.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one current snapshot per Device, got %d", count)
	}

	var storedDevice model.Device
	if err := db.First(&storedDevice, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedDevice.LastSeenAt == nil || storedDevice.LastIP != "198.51.100.11" || storedDevice.AgentVersion != version {
		t.Fatalf("Device presence not synchronized: %#v", storedDevice)
	}
}

func TestReportRejectsInvalidCPUAndOversizedCounters(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "bounds.db"))
	if err != nil {
		t.Fatal(err)
	}
	device := model.Device{Name: "COR-P-BOUNDS", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	service := &Service{DB: db}

	zeroCPU := 0
	if _, err := service.Report(device.ID, "192.0.2.20", HeartbeatInput{CPUCount: &zeroCPU}); !errors.Is(err, ErrInvalidCPUCount) {
		t.Fatalf("expected invalid cpu count, got %v", err)
	}

	tooLarge := uint64(^uint64(0))
	if _, err := service.Report(device.ID, "192.0.2.20", HeartbeatInput{UptimeSeconds: &tooLarge}); !errors.Is(err, ErrInvalidCounter) {
		t.Fatalf("expected invalid counter, got %v", err)
	}
}

func TestReportRejectsInvalidCapacityMetrics(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "invalid.db"))
	if err != nil {
		t.Fatal(err)
	}
	device := model.Device{Name: "COR-P-INVALID", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	service := &Service{DB: db}

	memoryTotal := uint64(100)
	memoryAvailable := uint64(101)
	_, err = service.Report(device.ID, "192.0.2.10", HeartbeatInput{
		MemoryTotalBytes:     &memoryTotal,
		MemoryAvailableBytes: &memoryAvailable,
	})
	if !errors.Is(err, ErrInvalidMemoryMetrics) {
		t.Fatalf("expected invalid memory metrics, got %v", err)
	}

	diskTotal := uint64(200)
	diskAvailable := uint64(201)
	_, err = service.Report(device.ID, "192.0.2.10", HeartbeatInput{
		DiskTotalBytes:     &diskTotal,
		DiskAvailableBytes: &diskAvailable,
	})
	if !errors.Is(err, ErrInvalidDiskMetrics) {
		t.Fatalf("expected invalid disk metrics, got %v", err)
	}
}

func TestStatusUsesHeartbeatNotGenericDeviceLastSeen(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "status.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 13, 18, 0, 0, 0, time.UTC)
	genericSeen := now
	device := model.Device{Name: "COR-P-STATUS", Status: "active", LastSeenAt: &genericSeen, LastIP: "203.0.113.8"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	service := &Service{DB: db, Now: func() time.Time { return now }}

	status, err := service.Status(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != AgentStateNeverSeen || status.Online || status.Telemetry != nil {
		t.Fatalf("generic Device last_seen must not imply Agent online: %#v", status)
	}

	if _, err := service.Report(device.ID, "198.51.100.20", HeartbeatInput{}); err != nil {
		t.Fatal(err)
	}
	status, err = service.Status(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != AgentStateOnline || !status.Online || status.HeartbeatAgeSeconds == nil || *status.HeartbeatAgeSeconds != 0 {
		t.Fatalf("expected online Agent status: %#v", status)
	}

	now = now.Add(time.Duration(DefaultOnlineThresholdSeconds+1) * time.Second)
	status, err = service.Status(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != AgentStateOffline || status.Online {
		t.Fatalf("expected offline Agent status after threshold: %#v", status)
	}
}

func TestListStatusesReturnsFleetWithoutNPlusOneContract(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "fleet.db"))
	if err != nil {
		t.Fatal(err)
	}
	first := model.Device{Name: "A-UDM", Status: "active"}
	second := model.Device{Name: "B-UDM", Status: "active"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	service := &Service{DB: db}
	if _, err := service.Report(first.ID, "192.0.2.1", HeartbeatInput{}); err != nil {
		t.Fatal(err)
	}

	statuses, err := service.ListStatuses()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 || statuses[0].Device.Name != "A-UDM" || statuses[0].State != AgentStateOnline || statuses[1].State != AgentStateNeverSeen {
		t.Fatalf("unexpected fleet statuses: %#v", statuses)
	}
}
