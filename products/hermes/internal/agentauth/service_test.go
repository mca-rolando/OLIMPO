package agentauth

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/database"
	"github.com/mca-rolando/HermesDDNS/internal/model"
)

func TestIssueAndAuthenticateAgentCredential(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "agentauth.db"))
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{Name: "COR-P-AGENT", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc := Service{DB: db, Now: func() time.Time { return now }}

	credential, key, err := svc.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if credential.Status != model.CredentialStatusActive {
		t.Fatalf("unexpected status %q", credential.Status)
	}
	if credential.CredentialID != key.ID {
		t.Fatalf("credential id mismatch: got %q want %q", credential.CredentialID, key.ID)
	}

	ctx, err := svc.Authenticate(key.Plaintext, "198.51.100.40")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Device.ID != device.ID || ctx.Credential.ID != credential.ID {
		t.Fatalf("unexpected auth context: %#v", ctx)
	}
	if ctx.Credential.LastUsedAt == nil || !ctx.Credential.LastUsedAt.Equal(now) {
		t.Fatalf("unexpected LastUsedAt: %v", ctx.Credential.LastUsedAt)
	}
	if ctx.Credential.LastUsedIP != "198.51.100.40" {
		t.Fatalf("unexpected LastUsedIP %q", ctx.Credential.LastUsedIP)
	}
}

func TestAgentCredentialRejectsWrongExpiredAndRevokedKeys(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "agentauth.db"))
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{Name: "COR-P-AGENT-2", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc := Service{DB: db, Now: func() time.Time { return now }}
	credential, key, err := svc.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Authenticate(key.Plaintext+"x", "198.51.100.41"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("wrong key: expected ErrBadAuth, got %v", err)
	}

	expired := now.Add(-time.Minute)
	if err := db.Model(&model.DeviceIdentityCredential{}).Where("id = ?", credential.ID).Update("expires_at", expired).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(key.Plaintext, "198.51.100.41"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("expired key: expected ErrBadAuth, got %v", err)
	}

	if err := db.Model(&model.DeviceIdentityCredential{}).Where("id = ?", credential.ID).Updates(map[string]any{
		"expires_at": nil,
		"status":     model.CredentialStatusRevoked,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(key.Plaintext, "198.51.100.41"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("revoked key: expected ErrBadAuth, got %v", err)
	}
}

func TestIssueAgentCredentialRejectsSecondActiveCredential(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "agentauth.db"))
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{Name: "COR-P-AGENT-3", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	svc := Service{DB: db}
	if _, _, err := svc.IssueCredential(device.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.IssueCredential(device.ID); !errors.Is(err, ErrActiveCredentialExists) {
		t.Fatalf("expected ErrActiveCredentialExists, got %v", err)
	}
}

func TestRevokeAgentCredentialAllowsReplacement(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "agentauth.db"))
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{Name: "COR-P-AGENT-4", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	svc := Service{DB: db}
	credential, oldKey, err := svc.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeCredential(device.ID, credential.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(oldKey.Plaintext, "198.51.100.42"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("revoked key: expected ErrBadAuth, got %v", err)
	}

	if _, _, err := svc.IssueCredential(device.ID); err != nil {
		t.Fatalf("replacement credential should be issuable after revoke: %v", err)
	}
}
