package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/config"
	"github.com/mca-rolando/HermesDDNS/internal/credential"
	"github.com/mca-rolando/HermesDDNS/internal/database"
	"github.com/mca-rolando/HermesDDNS/internal/ddns"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
	"gorm.io/gorm"
)

type apiFakeDNS struct {
	calls int
}

func (f *apiFakeDNS) Upsert(hostname, zone, recordType, target string, ttl int, wildcard bool) error {
	f.calls++
	return nil
}

func (f *apiFakeDNS) Delete(hostname, zone, recordType string, wildcard bool) error {
	return nil
}

func TestCredentialRotationAdminAPIRequestsWithoutIssuingSecret(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)

	body := bytes.NewBufferString(`{"grace_minutes":45}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/credential-rotations", device.ID), body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "api_key") || strings.Contains(rec.Body.String(), "hddns_") {
		t.Fatalf("rotation request must not issue or return a DDNS secret: %s", rec.Body.String())
	}

	var response struct {
		Rotation model.DDNSCredentialRotation `json:"rotation"`
		Next     string                       `json:"next_action"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Rotation.Status != model.RotationStatusRequested {
		t.Fatalf("expected requested rotation, got %q", response.Rotation.Status)
	}
	if response.Rotation.GraceMinutes != 45 {
		t.Fatalf("expected 45 grace minutes, got %d", response.Rotation.GraceMinutes)
	}
	if response.Next != "agent_stage_candidate" {
		t.Fatalf("unexpected next action %q", response.Next)
	}

	var activeCount int64
	if err := db.Model(&model.DDNSCredential{}).
		Where("device_id = ? AND status = ?", device.ID, model.CredentialStatusActive).
		Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}
	if activeCount != 1 {
		t.Fatalf("expected current credential to remain active, count=%d", activeCount)
	}
}

func TestLegacyRotateRouteUsesSafeLifecycleRequest(t *testing.T) {
	_, server, device, _ := newCredentialAPITestServer(t)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/credentials/rotate", device.ID), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "api_key") || strings.Contains(rec.Body.String(), "hddns_") {
		t.Fatalf("legacy route must no longer return a DDNS secret: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"requested"`) {
		t.Fatalf("legacy route must create a requested lifecycle rotation: %s", rec.Body.String())
	}
}

func TestCredentialRotationAdminAPIListsGetsAndRollsBack(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)

	rotation, err := server.Credentials.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/credential-rotations", device.ID), nil)
	listRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list expected %d, got %d: %s", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	var listed []model.DDNSCredentialRotation
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != rotation.ID {
		t.Fatalf("rotation %d not present in list: %s", rotation.ID, listRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/credential-rotations/%d", device.ID, rotation.ID), nil)
	getRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get expected %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}

	rollbackReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/credential-rotations/%d/rollback", device.ID, rotation.ID), bytes.NewBufferString(`{"reason":"operator cancelled"}`))
	rollbackReq.Header.Set("Content-Type", "application/json")
	rollbackRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rollbackRec, rollbackReq)
	if rollbackRec.Code != http.StatusOK {
		t.Fatalf("rollback expected %d, got %d: %s", http.StatusOK, rollbackRec.Code, rollbackRec.Body.String())
	}

	var stored model.DDNSCredentialRotation
	if err := db.First(&stored, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.RotationStatusRolledBack {
		t.Fatalf("expected rolled_back, got %q", stored.Status)
	}
	if stored.LastError != "operator cancelled" {
		t.Fatalf("unexpected rollback reason %q", stored.LastError)
	}
}

func TestSuccessfulDDNSUpdateConfirmsValidatingRotation(t *testing.T) {
	db, server, device, oldKey := newCredentialAPITestServer(t)

	rotation, err := server.Credentials.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}
	newKey, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := server.Credentials.StageCandidate(rotation.ID, device.ID, newKey.ID, newKey.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Credentials.StartValidation(rotation.ID, device.ID); err != nil {
		t.Fatal(err)
	}

	url := "/nic/update?hostname=cor-p-api.ddns.example.com&myip=203.0.113.44"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.SetBasicAuth(device.Name, newKey.Plaintext)
	req.RemoteAddr = "198.51.100.27:45000"
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("DDNS expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "good 203.0.113.44\n" {
		t.Fatalf("unexpected DDNS response %q", rec.Body.String())
	}

	var storedRotation model.DDNSCredentialRotation
	if err := db.First(&storedRotation, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedRotation.Status != model.RotationStatusGrace {
		t.Fatalf("expected rotation grace after successful DDNS validation, got %q", storedRotation.Status)
	}
	if storedRotation.ConfirmedAt == nil || storedRotation.GraceUntil == nil {
		t.Fatalf("confirmed rotation must have confirmation and grace timestamps: %#v", storedRotation)
	}

	var oldCredential model.DDNSCredential
	if err := db.Where("device_id = ? AND key_id = ?", device.ID, oldKey.ID).First(&oldCredential).Error; err != nil {
		t.Fatal(err)
	}
	if oldCredential.Status != model.CredentialStatusGrace || oldCredential.GraceUntil == nil {
		t.Fatalf("old credential must enter grace: %#v", oldCredential)
	}

	var replacement model.DDNSCredential
	if err := db.First(&replacement, candidate.ID).Error; err != nil {
		t.Fatal(err)
	}
	if replacement.Status != model.CredentialStatusActive || replacement.ConfirmedAt == nil {
		t.Fatalf("replacement must remain active and confirmed: %#v", replacement)
	}
}

func TestReconcileEndpointCompletesExpiredGrace(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)

	rotation, err := server.Credentials.RequestRotation(device.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	newKey, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := server.Credentials.StageCandidate(rotation.ID, device.ID, newKey.ID, newKey.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Credentials.StartValidation(rotation.ID, device.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := server.Credentials.ConfirmCredentialUse(candidate.ID); err != nil {
		t.Fatal(err)
	}

	expired := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&model.DDNSCredentialRotation{}).Where("id = ?", rotation.ID).Update("grace_until", expired).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.DDNSCredential{}).Where("id = ?", rotation.PreviousCredentialID).Update("grace_until", expired).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credential-rotations/reconcile", nil)
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reconcile expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"completed":1`) {
		t.Fatalf("expected one completed rotation: %s", rec.Body.String())
	}

	var stored model.DDNSCredentialRotation
	if err := db.First(&stored, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.RotationStatusCompleted {
		t.Fatalf("expected completed rotation, got %q", stored.Status)
	}
}

func newCredentialAPITestServer(t *testing.T) (*gorm.DB, *Server, model.Device, security.GeneratedKey) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{Name: "COR-P-API", DisplayName: "API Test UDM", Type: "UDM-SE", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	active := model.DDNSCredential{
		DeviceID:    device.ID,
		KeyID:       key.ID,
		SecretHash:  key.Hash,
		Status:      model.CredentialStatusActive,
		ActivatedAt: &now,
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatal(err)
	}

	fake := &apiFakeDNS{}
	ddnsService := &ddns.Service{
		DB:               db,
		DNS:              fake,
		AllowedDomains:   []string{"ddns.example.com"},
		DefaultTTL:       300,
		AutocreatePolicy: "any",
	}
	cfg := config.Config{
		APIPrefix:      "/api/v1",
		AllowedDomains: []string{"ddns.example.com"},
		DefaultTTL:     300,
	}
	server := New(db, cfg, ddnsService)
	server.Credentials = &credential.Service{DB: db}

	return db, server, device, key
}
