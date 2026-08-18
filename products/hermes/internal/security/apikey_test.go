package security

import (
	"strings"
	"testing"
)

func TestGenerateAndVerifyDDNSKey(t *testing.T) {
	key, err := GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}

	if key.Plaintext == "" || key.Hash == "" || key.ID == "" {
		t.Fatal("generated DDNS key fields must not be empty")
	}

	if !strings.HasPrefix(key.Plaintext, ddnsKeyPrefix) {
		t.Fatalf("DDNS key must start with %q", ddnsKeyPrefix)
	}

	if !VerifyAPIKey(key.Plaintext, key.Hash) {
		t.Fatal("generated DDNS key must verify")
	}

	if VerifyAPIKey(key.Plaintext+"x", key.Hash) {
		t.Fatal("modified DDNS key must not verify")
	}
}

func TestGenerateAndVerifyAgentKey(t *testing.T) {
	key, err := GenerateAgentKey()
	if err != nil {
		t.Fatal(err)
	}

	if key.Plaintext == "" || key.Hash == "" || key.ID == "" {
		t.Fatal("generated agent key fields must not be empty")
	}

	if !strings.HasPrefix(key.Plaintext, agentKeyPrefix) {
		t.Fatalf("agent key must start with %q", agentKeyPrefix)
	}

	if !VerifyAPIKey(key.Plaintext, key.Hash) {
		t.Fatal("generated agent key must verify")
	}

	if VerifyAPIKey(key.Plaintext+"x", key.Hash) {
		t.Fatal("modified agent key must not verify")
	}
}

func TestGenerateAPIKeyCompatibility(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(key.Plaintext, ddnsKeyPrefix) {
		t.Fatalf("legacy GenerateAPIKey must generate DDNS keys with prefix %q", ddnsKeyPrefix)
	}
}

func TestParseAgentKeyID(t *testing.T) {
	key, err := GenerateAgentKey()
	if err != nil {
		t.Fatal(err)
	}

	id, ok := ParseAgentKeyID(key.Plaintext)
	if !ok {
		t.Fatal("generated agent key must parse")
	}
	if id != key.ID {
		t.Fatalf("parsed id: got %q want %q", id, key.ID)
	}

	invalid := []string{
		"",
		"hddns_" + key.ID + ".invalid",
		"hagent_nothex.invalid",
		"hagent_" + key.ID,
		"hagent_" + key.ID + ".short",
	}
	for _, item := range invalid {
		if _, ok := ParseAgentKeyID(item); ok {
			t.Fatalf("invalid agent key parsed successfully: %q", item)
		}
	}
}

func TestGenerateAndVerifyEnrollmentKey(t *testing.T) {
	key, err := GenerateEnrollmentKey()
	if err != nil {
		t.Fatal(err)
	}

	if key.Plaintext == "" || key.Hash == "" || key.ID == "" {
		t.Fatal("generated enrollment key fields must not be empty")
	}
	if !strings.HasPrefix(key.Plaintext, enrollmentKeyPrefix) {
		t.Fatalf("enrollment key must start with %q", enrollmentKeyPrefix)
	}
	if !VerifyAPIKey(key.Plaintext, key.Hash) {
		t.Fatal("generated enrollment key must verify")
	}

	id, ok := ParseEnrollmentKeyID(key.Plaintext)
	if !ok || id != key.ID {
		t.Fatalf("generated enrollment key must parse: id=%q ok=%v want=%q", id, ok, key.ID)
	}

	invalid := []string{
		"",
		"hagent_" + key.ID + ".invalid",
		"henroll_nothex.invalid",
		"henroll_" + key.ID,
		"henroll_" + key.ID + ".short",
	}
	for _, item := range invalid {
		if _, ok := ParseEnrollmentKeyID(item); ok {
			t.Fatalf("invalid enrollment key parsed successfully: %q", item)
		}
	}
}
