package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const keyPrefix = "hddns_"

type GeneratedKey struct {
	ID        string
	Plaintext string
	Hash      string
}

func GenerateAPIKey() (GeneratedKey, error) {
	idRaw := make([]byte, 8)
	secretRaw := make([]byte, 32)
	if _, err := rand.Read(idRaw); err != nil {
		return GeneratedKey{}, fmt.Errorf("generate key id: %w", err)
	}
	if _, err := rand.Read(secretRaw); err != nil {
		return GeneratedKey{}, fmt.Errorf("generate key secret: %w", err)
	}

	id := hex.EncodeToString(idRaw)
	secret := base64.RawURLEncoding.EncodeToString(secretRaw)
	plain := keyPrefix + id + "." + secret
	return GeneratedKey{ID: id, Plaintext: plain, Hash: HashAPIKey(plain)}, nil
}

func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func VerifyAPIKey(plain, expectedHash string) bool {
	actual := HashAPIKey(plain)
	if len(actual) != len(expectedHash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}
