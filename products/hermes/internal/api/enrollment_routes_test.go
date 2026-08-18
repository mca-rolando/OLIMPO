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
)

func TestEnrollmentAdminAPIAndSingleUseExchange(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)

	createReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/devices/%d/enrollments", device.ID),
		bytes.NewBufferString(`{"ttl_minutes":10}`),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create enrollment expected %d, got %d: %s", http.StatusCreated, createRec.Code, createRec.Body.String())
	}
	if strings.Contains(createRec.Body.String(), "secret_hash") {
		t.Fatalf("enrollment hash must not be returned: %s", createRec.Body.String())
	}

	var created struct {
		Enrollment      model.AgentEnrollment `json:"enrollment"`
		EnrollmentToken string                `json:"enrollment_token"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Enrollment.Status != model.EnrollmentStatusPending || !strings.HasPrefix(created.EnrollmentToken, "henroll_") {
		t.Fatalf("unexpected create response: %s", createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/enrollments", device.ID), nil)
	listRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), created.Enrollment.TokenID) {
		t.Fatalf("list enrollment failed: %d %s", listRec.Code, listRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/enrollments/%d", device.ID, created.Enrollment.ID), nil)
	getRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get enrollment expected %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}

	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", nil)
	unauthorizedRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("enroll without token expected %d, got %d", http.StatusUnauthorized, unauthorizedRec.Code)
	}

	exchangeReq := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", nil)
	exchangeReq.Header.Set("Authorization", "Bearer "+created.EnrollmentToken)
	exchangeReq.RemoteAddr = "198.51.100.60:55123"
	exchangeRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(exchangeRec, exchangeReq)
	if exchangeRec.Code != http.StatusCreated {
		t.Fatalf("exchange expected %d, got %d: %s", http.StatusCreated, exchangeRec.Code, exchangeRec.Body.String())
	}
	if strings.Contains(exchangeRec.Body.String(), "secret_hash") || strings.Contains(exchangeRec.Body.String(), "hddns_") {
		t.Fatalf("exchange must not expose stored hashes or a DDNS plaintext secret: %s", exchangeRec.Body.String())
	}

	var exchanged struct {
		Enrollment      model.AgentEnrollment          `json:"enrollment"`
		Device          model.Device                   `json:"device"`
		AgentCredential model.DeviceIdentityCredential `json:"agent_credential"`
		AgentKey        string                         `json:"agent_key"`
		DDNS            struct {
			Username       string   `json:"username"`
			UpdatePath     string   `json:"update_path"`
			AllowedDomains []string `json:"allowed_domains"`
		} `json:"ddns_configuration"`
	}
	if err := json.Unmarshal(exchangeRec.Body.Bytes(), &exchanged); err != nil {
		t.Fatal(err)
	}
	if exchanged.Enrollment.Status != model.EnrollmentStatusIssued || exchanged.Device.ID != device.ID {
		t.Fatalf("unexpected exchange identity: %s", exchangeRec.Body.String())
	}
	if !strings.HasPrefix(exchanged.AgentKey, "hagent_") || exchanged.AgentCredential.ID == 0 {
		t.Fatalf("exchange did not issue Agent identity: %s", exchangeRec.Body.String())
	}
	if exchanged.DDNS.Username != device.Name || exchanged.DDNS.UpdatePath != "/nic/update" || len(exchanged.DDNS.AllowedDomains) != 1 {
		t.Fatalf("unexpected DDNS bootstrap configuration: %#v", exchanged.DDNS)
	}

	var stored model.AgentEnrollment
	if err := db.First(&stored, created.Enrollment.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.EnrollmentStatusIssued || stored.UsedIP != "198.51.100.60" || stored.AgentCredentialID == nil {
		t.Fatalf("unexpected stored enrollment after exchange: %#v", stored)
	}

	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", nil)
	replayReq.Header.Set("Authorization", "Bearer "+created.EnrollmentToken)
	replayRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed enrollment token expected %d, got %d: %s", http.StatusUnauthorized, replayRec.Code, replayRec.Body.String())
	}
}

func TestAgentEnrollmentConfirmationIsAuthenticatedAndIdempotent(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)

	enrollment, token, err := server.Enrollments.Create(device.ID, 15)
	if err != nil {
		t.Fatal(err)
	}
	exchanged, err := server.Enrollments.Exchange(token.Plaintext, "198.51.100.70")
	if err != nil {
		t.Fatal(err)
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enrollment/confirm", nil)
	badRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("confirmation without Agent identity expected %d, got %d", http.StatusUnauthorized, badRec.Code)
	}

	body := bytes.NewBufferString(`{"agent_version":"26.08-02-test"}`)
	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enrollment/confirm", body)
	confirmReq.Header.Set("Authorization", "Bearer "+exchanged.AgentKey.Plaintext)
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmReq.RemoteAddr = "198.51.100.71:56000"
	confirmRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("confirmation expected %d, got %d: %s", http.StatusOK, confirmRec.Code, confirmRec.Body.String())
	}
	if !strings.Contains(confirmRec.Body.String(), `"status":"completed"`) || !strings.Contains(confirmRec.Body.String(), `"next_action":"enrollment_complete"`) {
		t.Fatalf("unexpected confirmation response: %s", confirmRec.Body.String())
	}

	var stored model.AgentEnrollment
	if err := db.First(&stored, enrollment.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.EnrollmentStatusCompleted || stored.CompletedAt == nil {
		t.Fatalf("expected completed enrollment: %#v", stored)
	}

	var storedDevice model.Device
	if err := db.First(&storedDevice, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedDevice.AgentVersion != "26.08-02-test" || storedDevice.LastIP != "198.51.100.71" || storedDevice.LastSeenAt == nil {
		t.Fatalf("confirmation did not update Device state: %#v", storedDevice)
	}

	// Retry the same confirmation with the same permanent identity.
	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enrollment/confirm", bytes.NewBufferString(`{"agent_version":"26.08-02-test"}`))
	retryReq.Header.Set("Authorization", "Bearer "+exchanged.AgentKey.Plaintext)
	retryReq.Header.Set("Content-Type", "application/json")
	retryRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("idempotent confirmation expected %d, got %d: %s", http.StatusOK, retryRec.Code, retryRec.Body.String())
	}
}

func TestRevokeIssuedEnrollmentRevokesReturnedAgentKey(t *testing.T) {
	_, server, device, _ := newCredentialAPITestServer(t)

	enrollment, token, err := server.Enrollments.Create(device.ID, 15)
	if err != nil {
		t.Fatal(err)
	}
	exchanged, err := server.Enrollments.Exchange(token.Plaintext, "198.51.100.80")
	if err != nil {
		t.Fatal(err)
	}

	revokeReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/devices/%d/enrollments/%d/revoke", device.ID, enrollment.ID),
		nil,
	)
	revokeRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke issued enrollment expected %d, got %d: %s", http.StatusOK, revokeRec.Code, revokeRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+exchanged.AgentKey.Plaintext)
	meRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("Agent identity from revoked enrollment expected %d, got %d: %s", http.StatusUnauthorized, meRec.Code, meRec.Body.String())
	}
}

func TestExpiredEnrollmentTokenIsRejectedByAPI(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)

	now := time.Date(2026, 8, 13, 17, 0, 0, 0, time.UTC)
	nowFunc := func() time.Time { return now }
	server.AgentAuth.Now = nowFunc
	server.Enrollments.Now = nowFunc

	enrollment, token, err := server.Enrollments.Create(device.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token.Plaintext)
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired token expected %d, got %d: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var stored model.AgentEnrollment
	if err := db.First(&stored, enrollment.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.EnrollmentStatusExpired {
		t.Fatalf("expected expired enrollment, got %q", stored.Status)
	}
}
