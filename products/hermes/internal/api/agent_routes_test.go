package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
)

func TestAdminIssuesAndListsAgentCredential(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)

	issueReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/agent-credentials", device.ID), nil)
	issueRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(issueRec, issueReq)
	if issueRec.Code != http.StatusCreated {
		t.Fatalf("issue expected %d, got %d: %s", http.StatusCreated, issueRec.Code, issueRec.Body.String())
	}
	if !strings.Contains(issueRec.Body.String(), "hagent_") {
		t.Fatalf("issued response must contain one-time agent key: %s", issueRec.Body.String())
	}
	if strings.Contains(issueRec.Body.String(), "secret_hash") {
		t.Fatalf("response must never expose stored hash: %s", issueRec.Body.String())
	}

	var issued struct {
		Credential model.DeviceIdentityCredential `json:"credential"`
		AgentKey   string                         `json:"agent_key"`
	}
	if err := json.Unmarshal(issueRec.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.Credential.DeviceID != device.ID || issued.AgentKey == "" {
		t.Fatalf("unexpected issue response: %#v", issued)
	}

	var stored model.DeviceIdentityCredential
	if err := db.First(&stored, issued.Credential.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !security.VerifyAPIKey(issued.AgentKey, stored.SecretHash) {
		t.Fatal("issued agent key must verify against stored hash")
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/agent-credentials", device.ID), nil)
	duplicateRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(duplicateRec, duplicateReq)
	if duplicateRec.Code != http.StatusConflict {
		t.Fatalf("second active credential expected %d, got %d: %s", http.StatusConflict, duplicateRec.Code, duplicateRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/agent-credentials", device.ID), nil)
	listRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list expected %d, got %d: %s", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "secret_hash") || strings.Contains(listRec.Body.String(), issued.AgentKey) {
		t.Fatalf("agent credential list must not expose secrets: %s", listRec.Body.String())
	}
}

func TestAgentMeRequiresBearerAndTracksUsage(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	credential, key, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent/me", nil)
	unauthRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token expected %d, got %d", http.StatusUnauthorized, unauthRec.Code)
	}

	wrongReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent/me", nil)
	wrongReq.Header.Set("Authorization", "Bearer hagent_0000000000000000.invalid")
	wrongRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(wrongRec, wrongReq)
	if wrongRec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token expected %d, got %d", http.StatusUnauthorized, wrongRec.Code)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+key.Plaintext)
	meReq.RemoteAddr = "198.51.100.80:55000"
	meRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("agent me expected %d, got %d: %s", http.StatusOK, meRec.Code, meRec.Body.String())
	}
	var me struct {
		Device     model.Device                   `json:"device"`
		Credential model.DeviceIdentityCredential `json:"credential"`
	}
	if err := json.Unmarshal(meRec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode agent me response: %v: %s", err, meRec.Body.String())
	}
	if me.Device.ID != device.ID || me.Credential.ID != credential.ID {
		t.Fatalf("agent me response does not identify authenticated device/credential: %s", meRec.Body.String())
	}
	if strings.Contains(meRec.Body.String(), "secret_hash") {
		t.Fatalf("agent me response must not expose stored hash: %s", meRec.Body.String())
	}

	var stored model.DeviceIdentityCredential
	if err := db.First(&stored, credential.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.LastUsedAt == nil || stored.LastUsedIP != "198.51.100.80" {
		t.Fatalf("agent usage not recorded: %#v", stored)
	}
}

func TestAgentCredentialRotationLifecycleAPI(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	rotation, err := server.Credentials.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent/credential-rotations/current", nil)
	currentReq.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	currentRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(currentRec, currentReq)
	if currentRec.Code != http.StatusOK {
		t.Fatalf("current rotation expected %d, got %d: %s", http.StatusOK, currentRec.Code, currentRec.Body.String())
	}
	if !strings.Contains(currentRec.Body.String(), `"next_action":"generate_and_stage_candidate"`) {
		t.Fatalf("unexpected current rotation response: %s", currentRec.Body.String())
	}

	candidateKey, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	stageBody := bytes.NewBufferString(fmt.Sprintf(`{"key_id":%q,"secret_hash":%q}`, candidateKey.ID, candidateKey.Hash))
	stageReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agent/credential-rotations/%d/stage", rotation.ID), stageBody)
	stageReq.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	stageReq.Header.Set("Content-Type", "application/json")
	stageRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(stageRec, stageReq)
	if stageRec.Code != http.StatusCreated {
		t.Fatalf("stage expected %d, got %d: %s", http.StatusCreated, stageRec.Code, stageRec.Body.String())
	}
	if strings.Contains(stageRec.Body.String(), candidateKey.Plaintext) {
		t.Fatalf("Hermes response must not contain candidate plaintext: %s", stageRec.Body.String())
	}

	var candidate model.DDNSCredential
	if err := db.Where("device_id = ? AND key_id = ?", device.ID, candidateKey.ID).First(&candidate).Error; err != nil {
		t.Fatal(err)
	}
	if candidate.Status != model.CredentialStatusPending {
		t.Fatalf("staged candidate expected pending, got %q", candidate.Status)
	}

	validateReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agent/credential-rotations/%d/start-validation", rotation.ID), nil)
	validateReq.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	validateRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(validateRec, validateReq)
	if validateRec.Code != http.StatusOK {
		t.Fatalf("start validation expected %d, got %d: %s", http.StatusOK, validateRec.Code, validateRec.Body.String())
	}
	if !strings.Contains(validateRec.Body.String(), `"status":"validating"`) {
		t.Fatalf("rotation must enter validating: %s", validateRec.Body.String())
	}

	ddnsReq := httptest.NewRequest(http.MethodGet, "/nic/update?hostname=cor-p-api.ddns.example.com&myip=203.0.113.99", nil)
	ddnsReq.SetBasicAuth(device.Name, candidateKey.Plaintext)
	ddnsReq.RemoteAddr = "198.51.100.81:56000"
	ddnsRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(ddnsRec, ddnsReq)
	if ddnsRec.Code != http.StatusOK {
		t.Fatalf("candidate DDNS validation expected %d, got %d: %s", http.StatusOK, ddnsRec.Code, ddnsRec.Body.String())
	}

	var storedRotation model.DDNSCredentialRotation
	if err := db.First(&storedRotation, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedRotation.Status != model.RotationStatusGrace || storedRotation.ConfirmedAt == nil {
		t.Fatalf("successful candidate use must advance rotation to grace: %#v", storedRotation)
	}
}

func TestAgentCannotStageAnotherDevicesRotation(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	other := model.Device{Name: "OTHER-UDM", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	otherKey, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.Create(&model.DDNSCredential{
		DeviceID:    other.ID,
		KeyID:       otherKey.ID,
		SecretHash:  otherKey.Hash,
		Status:      model.CredentialStatusActive,
		ActivatedAt: &now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	rotation, err := server.Credentials.RequestRotation(other.ID, 30)
	if err != nil {
		t.Fatal(err)
	}

	candidate, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	body := bytes.NewBufferString(fmt.Sprintf(`{"key_id":%q,"secret_hash":%q}`, candidate.ID, candidate.Hash))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agent/credential-rotations/%d/stage", rotation.ID), body)
	req.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-device stage expected %d, got %d: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestRevokedAgentCredentialCannotAuthenticate(t *testing.T) {
	_, server, device, _ := newCredentialAPITestServer(t)
	credential, key, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	revokeReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/agent-credentials/%d/revoke", device.ID, credential.ID), nil)
	revokeRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke expected %d, got %d: %s", http.StatusOK, revokeRec.Code, revokeRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+key.Plaintext)
	meRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked credential expected %d, got %d: %s", http.StatusUnauthorized, meRec.Code, meRec.Body.String())
	}
}
