package enrollment

import (
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/agentauth"
	"github.com/mca-rolando/HermesDDNS/internal/database"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"gorm.io/gorm"
)

func TestCreateExchangeConfirmEnrollmentSingleUse(t *testing.T) {
	db, svc, device, now := newEnrollmentTestService(t)

	enrollment, token, err := svc.Create(device.ID, 15)
	if err != nil {
		t.Fatal(err)
	}
	if enrollment.Status != model.EnrollmentStatusPending {
		t.Fatalf("expected pending enrollment, got %q", enrollment.Status)
	}
	if enrollment.SecretHash == "" || token.Plaintext == "" {
		t.Fatal("enrollment token must be generated and stored only as a hash")
	}

	if _, err := svc.Exchange(wrongEnrollmentSecret(token.ID), "198.51.100.10"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("wrong enrollment secret: expected ErrBadAuth, got %v", err)
	}

	exchanged, err := svc.Exchange(token.Plaintext, "198.51.100.11")
	if err != nil {
		t.Fatal(err)
	}
	if exchanged.Enrollment.Status != model.EnrollmentStatusIssued || exchanged.Enrollment.IssuedAt == nil {
		t.Fatalf("expected issued enrollment: %#v", exchanged.Enrollment)
	}
	if exchanged.Enrollment.UsedIP != "198.51.100.11" {
		t.Fatalf("unexpected enrollment UsedIP %q", exchanged.Enrollment.UsedIP)
	}
	if exchanged.Enrollment.AgentCredentialID == nil || *exchanged.Enrollment.AgentCredentialID != exchanged.Credential.ID {
		t.Fatalf("issued credential not linked to enrollment: %#v", exchanged.Enrollment)
	}
	if exchanged.AgentKey.Plaintext == "" || exchanged.Credential.SecretHash == "" {
		t.Fatal("exchange must issue a permanent Agent identity key")
	}

	if _, err := svc.Exchange(token.Plaintext, "198.51.100.12"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("enrollment token replay: expected ErrBadAuth, got %v", err)
	}

	auth, err := svc.AgentAuth.Authenticate(exchanged.AgentKey.Plaintext, "198.51.100.13")
	if err != nil {
		t.Fatal(err)
	}
	if auth.Device.ID != device.ID || auth.Credential.ID != exchanged.Credential.ID {
		t.Fatalf("unexpected Agent auth context: %#v", auth)
	}

	*now = now.Add(2 * time.Minute)
	completed, err := svc.Confirm(device.ID, exchanged.Credential.ID, "198.51.100.14", "26.08-02-test")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != model.EnrollmentStatusCompleted || completed.CompletedAt == nil {
		t.Fatalf("expected completed enrollment: %#v", completed)
	}

	var storedDevice model.Device
	if err := db.First(&storedDevice, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedDevice.AgentVersion != "26.08-02-test" || storedDevice.LastIP != "198.51.100.14" || storedDevice.LastSeenAt == nil {
		t.Fatalf("confirmation did not update Device presence: %#v", storedDevice)
	}

	// Confirmation is intentionally idempotent so an Agent can retry after a
	// lost HTTP response without changing identity state.
	if _, err := svc.Confirm(device.ID, exchanged.Credential.ID, "198.51.100.15", "26.08-02-test"); err != nil {
		t.Fatalf("repeated confirmation must be idempotent: %v", err)
	}
}

func TestExpiredEnrollmentCannotExchangeAndNewEnrollmentCanBeCreated(t *testing.T) {
	db, svc, device, now := newEnrollmentTestService(t)

	enrollment, token, err := svc.Create(device.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(6 * time.Minute)

	if _, err := svc.Exchange(token.Plaintext, "198.51.100.20"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("expired token: expected ErrBadAuth, got %v", err)
	}

	var stored model.AgentEnrollment
	if err := db.First(&stored, enrollment.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.EnrollmentStatusExpired {
		t.Fatalf("expired enrollment must be persisted as expired, got %q", stored.Status)
	}

	if _, _, err := svc.Create(device.ID, 5); err != nil {
		t.Fatalf("a replacement enrollment should be allowed after expiry: %v", err)
	}
}

func TestRevokePendingEnrollmentBlocksExchange(t *testing.T) {
	_, svc, device, _ := newEnrollmentTestService(t)

	enrollment, token, err := svc.Create(device.ID, 15)
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := svc.Revoke(device.ID, enrollment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Status != model.EnrollmentStatusRevoked || revoked.RevokedAt == nil {
		t.Fatalf("unexpected revoked enrollment: %#v", revoked)
	}
	if _, err := svc.Exchange(token.Plaintext, "198.51.100.30"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("revoked token: expected ErrBadAuth, got %v", err)
	}

	if _, _, err := svc.Create(device.ID, 15); err != nil {
		t.Fatalf("replacement enrollment should be allowed after revoke: %v", err)
	}
}

func TestRevokeIssuedEnrollmentAlsoRevokesAgentCredential(t *testing.T) {
	_, svc, device, _ := newEnrollmentTestService(t)

	enrollment, token, err := svc.Create(device.ID, 15)
	if err != nil {
		t.Fatal(err)
	}
	exchanged, err := svc.Exchange(token.Plaintext, "198.51.100.40")
	if err != nil {
		t.Fatal(err)
	}

	revoked, err := svc.Revoke(device.ID, enrollment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Status != model.EnrollmentStatusRevoked {
		t.Fatalf("expected revoked enrollment, got %q", revoked.Status)
	}
	if _, err := svc.AgentAuth.Authenticate(exchanged.AgentKey.Plaintext, "198.51.100.41"); !errors.Is(err, agentauth.ErrBadAuth) {
		t.Fatalf("Agent key issued by revoked enrollment must be unusable: %v", err)
	}
}

func TestEnrollmentGuardsOpenEnrollmentActiveIdentityAndDeviceScope(t *testing.T) {
	_, svc, device, _ := newEnrollmentTestService(t)

	enrollment, _, err := svc.Create(device.ID, 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Create(device.ID, 15); !errors.Is(err, ErrEnrollmentInProgress) {
		t.Fatalf("second open enrollment: expected ErrEnrollmentInProgress, got %v", err)
	}

	other := model.Device{Name: "OTHER-ENROLL", Status: "active"}
	if err := svc.DB.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(other.ID, enrollment.ID); !errors.Is(err, ErrEnrollmentNotFound) {
		t.Fatalf("cross-device get: expected ErrEnrollmentNotFound, got %v", err)
	}
	if _, err := svc.Revoke(other.ID, enrollment.ID); !errors.Is(err, ErrEnrollmentNotFound) {
		t.Fatalf("cross-device revoke: expected ErrEnrollmentNotFound, got %v", err)
	}

	if _, err := svc.Revoke(device.ID, enrollment.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.AgentAuth.IssueCredential(device.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Create(device.ID, 15); !errors.Is(err, agentauth.ErrActiveCredentialExists) {
		t.Fatalf("active Agent identity: expected ErrActiveCredentialExists, got %v", err)
	}
}

func TestEnrollmentGetReconcilesExpiry(t *testing.T) {
	_, svc, device, now := newEnrollmentTestService(t)
	enrollment, _, err := svc.Create(device.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(2 * time.Minute)

	got, err := svc.Get(device.ID, enrollment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.EnrollmentStatusExpired {
		t.Fatalf("expected reconciled expired status, got %q", got.Status)
	}
}

func newEnrollmentTestService(t *testing.T) (*gorm.DB, *Service, model.Device, *time.Time) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "enrollment.db"))
	if err != nil {
		t.Fatal(err)
	}
	device := model.Device{Name: "COR-P-ENROLL", DisplayName: "Enrollment Test UDM", Type: "UDM-SE", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	nowFunc := func() time.Time { return now }
	agentSvc := &agentauth.Service{DB: db, Now: nowFunc}
	svc := &Service{DB: db, AgentAuth: agentSvc, Now: nowFunc}
	return db, svc, device, &now
}

func wrongEnrollmentSecret(tokenID string) string {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = 0xA5
	}
	return "henroll_" + tokenID + "." + base64.RawURLEncoding.EncodeToString(secret)
}
