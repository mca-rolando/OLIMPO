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
