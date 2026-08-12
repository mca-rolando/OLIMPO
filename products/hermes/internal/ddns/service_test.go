package ddns

import (
	"path/filepath"
	"testing"

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
	key, err := security.GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	cred := model.DDNSCredential{DeviceID: device.ID, KeyID: key.ID, SecretHash: key.Hash, Status: "active"}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatal(err)
	}

	fake := &fakeDNS{}
	svc := Service{DB: db, DNS: fake, AllowedDomains: []string{"ddns.example.com"}, DefaultTTL: 300, AutocreatePolicy: "any"}
	auth, err := svc.Authenticate("COR-P-TEST", key.Plaintext)
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

	mk := func(name string) (AuthContext, string) {
		dev := model.Device{Name: name, Status: "active"}
		if err := db.Create(&dev).Error; err != nil {
			t.Fatal(err)
		}
		key, err := security.GenerateAPIKey()
		if err != nil {
			t.Fatal(err)
		}
		cred := model.DDNSCredential{DeviceID: dev.ID, KeyID: key.ID, SecretHash: key.Hash, Status: "active"}
		if err := db.Create(&cred).Error; err != nil {
			t.Fatal(err)
		}
		auth, err := svc.Authenticate(name, key.Plaintext)
		if err != nil {
			t.Fatal(err)
		}
		return auth, key.Plaintext
	}
	a, _ := mk("A")
	b, _ := mk("B")
	if _, err := svc.Update(a, "shared.ddns.example.com", "203.0.113.1", "203.0.113.1", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Update(b, "shared.ddns.example.com", "203.0.113.2", "203.0.113.2", "test"); err != ErrBadAuth {
		t.Fatalf("expected ErrBadAuth, got %v", err)
	}
}
