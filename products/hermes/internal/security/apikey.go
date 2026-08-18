package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	ddnsKeyPrefix       = "hddns_"
	agentKeyPrefix      = "hagent_"
	enrollmentKeyPrefix = "henroll_"
)

type GeneratedKey struct {
	ID        string
	Plaintext string
	Hash      string
}

func GenerateDDNSKey() (GeneratedKey, error) {
	return generateKey(ddnsKeyPrefix)
}

func GenerateAgentKey() (GeneratedKey, error) {
	return generateKey(agentKeyPrefix)
}

func GenerateEnrollmentKey() (GeneratedKey, error) {
	return generateKey(enrollmentKeyPrefix)
}

// GenerateAPIKey is retained for compatibility with 26.08-01 code.
// New code should explicitly use GenerateDDNSKey or GenerateAgentKey.
func GenerateAPIKey() (GeneratedKey, error) {
	return GenerateDDNSKey()
}

func generateKey(prefix string) (GeneratedKey, error) {
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
	plain := prefix + id + "." + secret

	return GeneratedKey{
		ID:        id,
		Plaintext: plain,
		Hash:      HashAPIKey(plain),
	}, nil
}

func ParseAgentKeyID(key string) (string, bool) {
	return parseKeyID(key, agentKeyPrefix)
}

func ParseEnrollmentKeyID(key string) (string, bool) {
	return parseKeyID(key, enrollmentKeyPrefix)
}

func parseKeyID(key, prefix string) (string, bool) {
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(key, prefix)
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	id := strings.ToLower(parts[0])
	if len(id) != 16 {
		return "", false
	}
	if _, err := hex.DecodeString(id); err != nil {
		return "", false
	}
	secret, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(secret) != 32 {
		return "", false
	}
	return id, true
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

	return subtle.ConstantTimeCompare(
		[]byte(actual),
		[]byte(expectedHash),
	) == 1
}
