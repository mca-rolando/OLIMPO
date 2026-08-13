package ddns

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/database"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
)

type fakeDNS struct{ calls int }

func (f *fakeDNS) Upsert(hostname, zone, recordType, target string, ttl int, wildcard bool) error {
	f.calls++
	return nil
}
func (f *fakeDNS) Delete(hostname, zone, recordType string, wildcard bool) error { return nil }

func TestUpdateCreatesThenReturnsNoChange(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	device := model.Device{Name: "COR-P-TEST", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	cred := model.DDNSCredential{DeviceID: device.ID, KeyID: key.ID, SecretHash: key.Hash, Status: model.CredentialStatusActive}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatal(err)
	}

	fake := &fakeDNS{}
	svc := Service{DB: db, DNS: fake, AllowedDomains: []string{"ddns.example.com"}, DefaultTTL: 300, AutocreatePolicy: "any"}
	auth, err := svc.Authenticate("COR-P-TEST", key.Plaintext, "203.0.113.10")
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.Update(auth, "cor-p-test.ddns.example.com", "203.0.113.10", "203.0.113.10", "test")
	if err != nil {
		t.Fatal(err)
	}
	if first.Code != "good" || !first.Created || fake.calls != 1 {
		t.Fatalf("unexpected first result: %#v calls=%d", first, fake.calls)
	}
	second, err := svc.Update(auth, "cor-p-test.ddns.example.com", "203.0.113.10", "203.0.113.10", "test")
	if err != nil {
		t.Fatal(err)
	}
	if second.Code != "nochg" || fake.calls != 1 {
		t.Fatalf("unexpected second result: %#v calls=%d", second, fake.calls)
	}
}

func TestDeviceCannotTakeAnotherDeviceHost(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeDNS{}
	svc := Service{DB: db, DNS: fake, AllowedDomains: []string{"ddns.example.com"}, DefaultTTL: 300, AutocreatePolicy: "any"}

	mk := func(name, callerIP string) (AuthContext, string) {
		dev := model.Device{Name: name, Status: "active"}
		if err := db.Create(&dev).Error; err != nil {
			t.Fatal(err)
		}
		key, err := security.GenerateDDNSKey()
		if err != nil {
			t.Fatal(err)
		}
		cred := model.DDNSCredential{DeviceID: dev.ID, KeyID: key.ID, SecretHash: key.Hash, Status: model.CredentialStatusActive}
		if err := db.Create(&cred).Error; err != nil {
			t.Fatal(err)
		}
		auth, err := svc.Authenticate(name, key.Plaintext, callerIP)
		if err != nil {
			t.Fatal(err)
		}
		return auth, key.Plaintext
	}
	a, _ := mk("A", "203.0.113.1")
	b, _ := mk("B", "203.0.113.2")
	if _, err := svc.Update(a, "shared.ddns.example.com", "203.0.113.1", "203.0.113.1", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Update(b, "shared.ddns.example.com", "203.0.113.2", "203.0.113.2", "test"); !errors.Is(err, ErrBadAuth) {
		t.Fatalf("expected ErrBadAuth, got %v", err)
	}
}

func TestAuthenticateCredentialLifecycle(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	tests := []struct {
		name       string
		status     string
		expiresAt  *time.Time
		graceUntil *time.Time
		wrongKey   bool
		wantOK     bool
	}{
		{name: "active", status: model.CredentialStatusActive, wantOK: true},
		{name: "active with future expiry", status: model.CredentialStatusActive, expiresAt: &future, wantOK: true},
		{name: "active expired", status: model.CredentialStatusActive, expiresAt: &past, wantOK: false},
		{name: "grace valid", status: model.CredentialStatusGrace, graceUntil: &future, wantOK: true},
		{name: "grace expired", status: model.CredentialStatusGrace, graceUntil: &past, wantOK: false},
		{name: "grace without deadline", status: model.CredentialStatusGrace, wantOK: false},
		{name: "pending", status: model.CredentialStatusPending, wantOK: false},
		{name: "revoked", status: model.CredentialStatusRevoked, wantOK: false},
		{name: "expired status", status: model.CredentialStatusExpired, wantOK: false},
		{name: "wrong key", status: model.CredentialStatusActive, wrongKey: true, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatal(err)
			}

			device := model.Device{Name: "AUTH-TEST", Status: "active"}
			if err := db.Create(&device).Error; err != nil {
				t.Fatal(err)
			}

			key, err := security.GenerateDDNSKey()
			if err != nil {
				t.Fatal(err)
			}
			cred := model.DDNSCredential{
				DeviceID:   device.ID,
				KeyID:      key.ID,
				SecretHash: key.Hash,
				Status:     tt.status,
				ExpiresAt:  tt.expiresAt,
				GraceUntil: tt.graceUntil,
			}
			if err := db.Create(&cred).Error; err != nil {
				t.Fatal(err)
			}

			plaintext := key.Plaintext
			if tt.wrongKey {
				plaintext += "-wrong"
			}

			_, err = (&Service{DB: db}).Authenticate("AUTH-TEST", plaintext, "198.51.100.25")
			if tt.wantOK && err != nil {
				t.Fatalf("expected authentication success, got %v", err)
			}
			if !tt.wantOK && !errors.Is(err, ErrBadAuth) {
				t.Fatalf("expected ErrBadAuth, got %v", err)
			}
		})
	}
}

func TestAuthenticateRecordsCredentialUsage(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{Name: "USAGE-TEST", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	cred := model.DDNSCredential{
		DeviceID:   device.ID,
		KeyID:      key.ID,
		SecretHash: key.Hash,
		Status:     model.CredentialStatusActive,
	}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatal(err)
	}

	before := time.Now().UTC()
	auth, err := (&Service{DB: db}).Authenticate("USAGE-TEST", key.Plaintext, "198.51.100.42")
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC()

	if auth.Credential.LastUsedAt == nil {
		t.Fatal("authenticated credential must include LastUsedAt")
	}
	if auth.Credential.LastUsedIP != "198.51.100.42" {
		t.Fatalf("unexpected LastUsedIP in auth context: %q", auth.Credential.LastUsedIP)
	}
	if auth.Credential.LastUsedAt.Before(before) || auth.Credential.LastUsedAt.After(after) {
		t.Fatalf("LastUsedAt is outside expected range: %v", auth.Credential.LastUsedAt)
	}

	var stored model.DDNSCredential
	if err := db.First(&stored, cred.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.LastUsedAt == nil {
		t.Fatal("stored credential must have LastUsedAt")
	}
	if stored.LastUsedIP != "198.51.100.42" {
		t.Fatalf("unexpected stored LastUsedIP: %q", stored.LastUsedIP)
	}
}
