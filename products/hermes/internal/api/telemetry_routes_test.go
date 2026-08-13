package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/telemetry"
)

func TestAgentHeartbeatRequiresIdentityAndStoresTelemetry(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", bytes.NewBufferString(`{}`))
	unauthorizedReq.Header.Set("Content-Type", "application/json")
	unauthorizedRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("heartbeat without identity expected %d, got %d", http.StatusUnauthorized, unauthorizedRec.Code)
	}

	body := bytes.NewBufferString(`{
		"agent_version":"26.08-02-test",
		"system_hostname":"COR-P-API",
		"platform":"unifi-os",
		"architecture":"arm64",
		"os_version":"UniFi OS",
		"kernel_version":"6.x",
		"firmware_version":"4.x",
		"boot_id":"boot-test-1",
		"uptime_seconds":12345,
		"cpu_count":4,
		"load_1":0.25,
		"load_5":0.20,
		"load_15":0.15,
		"memory_total_bytes":8589934592,
		"memory_available_bytes":6442450944,
		"disk_total_bytes":137438953472,
		"disk_available_bytes":107374182400
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", body)
	req.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.90:57000"
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("heartbeat expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret_hash") || strings.Contains(rec.Body.String(), agentKey.Plaintext) {
		t.Fatalf("heartbeat response must not expose credentials: %s", rec.Body.String())
	}

	var response struct {
		Status                 string                       `json:"status"`
		DeviceID               uint                         `json:"device_id"`
		NextHeartbeatSeconds   int                          `json:"next_heartbeat_seconds"`
		OnlineThresholdSeconds int                          `json:"online_threshold_seconds"`
		Telemetry              model.AgentTelemetrySnapshot `json:"telemetry"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "ok" || response.DeviceID != device.ID || response.NextHeartbeatSeconds != telemetry.DefaultHeartbeatIntervalSeconds {
		t.Fatalf("unexpected heartbeat response: %#v", response)
	}
	if response.Telemetry.DeviceID != device.ID || response.Telemetry.LastIP != "198.51.100.90" || response.Telemetry.AgentVersion != "26.08-02-test" {
		t.Fatalf("unexpected telemetry response: %#v", response.Telemetry)
	}

	var stored model.AgentTelemetrySnapshot
	if err := db.Where("device_id = ?", device.ID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SystemHostname != "COR-P-API" || stored.CPUCount != 4 || stored.MemoryAvailableBytes != 6442450944 {
		t.Fatalf("unexpected stored telemetry: %#v", stored)
	}

	var storedDevice model.Device
	if err := db.First(&storedDevice, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedDevice.AgentVersion != "26.08-02-test" || storedDevice.LastIP != "198.51.100.90" || storedDevice.LastSeenAt == nil {
		t.Fatalf("heartbeat did not refresh Device presence: %#v", storedDevice)
	}
}

func TestHeartbeatCannotSelectAnotherDevice(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	other := model.Device{Name: "OTHER-HEARTBEAT", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	// device_id is deliberately not part of the heartbeat contract. Even if a
	// client sends it as an unknown JSON field, identity comes only from hagent_.
	body := bytes.NewBufferString(fmt.Sprintf(`{"device_id":%d,"agent_version":"26.08-02"}`, other.ID))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", body)
	req.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("heartbeat expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var own model.AgentTelemetrySnapshot
	if err := db.Where("device_id = ?", device.ID).First(&own).Error; err != nil {
		t.Fatal(err)
	}
	var otherCount int64
	if err := db.Model(&model.AgentTelemetrySnapshot{}).Where("device_id = ?", other.ID).Count(&otherCount).Error; err != nil {
		t.Fatal(err)
	}
	if own.DeviceID != device.ID || otherCount != 0 {
		t.Fatalf("heartbeat identity was not bound to authenticated Device: own=%#v otherCount=%d", own, otherCount)
	}
}

func TestAgentStatusAPISupportsFleetDashboard(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	neverSeen := model.Device{Name: "ZZ-NEVER-SEEN", Status: "active"}
	if err := db.Create(&neverSeen).Error; err != nil {
		t.Fatal(err)
	}

	heartbeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", bytes.NewBufferString(`{"agent_version":"26.08-02"}`))
	heartbeatReq.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	heartbeatReq.Header.Set("Content-Type", "application/json")
	heartbeatRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(heartbeatRec, heartbeatReq)
	if heartbeatRec.Code != http.StatusOK {
		t.Fatalf("heartbeat expected %d, got %d: %s", http.StatusOK, heartbeatRec.Code, heartbeatRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/agent-status", device.ID), nil)
	statusRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK || !strings.Contains(statusRec.Body.String(), `"state":"online"`) {
		t.Fatalf("device Agent status expected online: %d %s", statusRec.Code, statusRec.Body.String())
	}

	neverReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/agent-status", neverSeen.ID), nil)
	neverRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(neverRec, neverReq)
	if neverRec.Code != http.StatusOK || !strings.Contains(neverRec.Body.String(), `"state":"never_seen"`) {
		t.Fatalf("never-seen Agent status unexpected: %d %s", neverRec.Code, neverRec.Body.String())
	}

	fleetReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent-status", nil)
	fleetRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(fleetRec, fleetReq)
	if fleetRec.Code != http.StatusOK {
		t.Fatalf("fleet status expected %d, got %d: %s", http.StatusOK, fleetRec.Code, fleetRec.Body.String())
	}
	var fleet []telemetry.AgentStatus
	if err := json.Unmarshal(fleetRec.Body.Bytes(), &fleet); err != nil {
		t.Fatal(err)
	}
	if len(fleet) != 2 {
		t.Fatalf("expected two fleet statuses, got %d: %s", len(fleet), fleetRec.Body.String())
	}
}

func TestAgentHeartbeatRejectsImpossibleCapacityMetrics(t *testing.T) {
	_, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", bytes.NewBufferString(`{"memory_total_bytes":100,"memory_available_bytes":101}`))
	req.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid capacity expected %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}
