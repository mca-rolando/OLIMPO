package security

import "testing"

func TestGenerateAndVerifyAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if key.Plaintext == "" || key.Hash == "" || key.ID == "" {
		t.Fatal("generated key fields must not be empty")
	}
	if !VerifyAPIKey(key.Plaintext, key.Hash) {
		t.Fatal("generated API key must verify")
	}
	if VerifyAPIKey(key.Plaintext+"x", key.Hash) {
		t.Fatal("modified API key must not verify")
	}
}
