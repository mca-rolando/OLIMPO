package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddress      string
	DatabasePath       string
	AdminLogin         string
	AllowedDomains     []string
	DefaultTTL         int
	AllowWildcard      bool
	TrustProxyHeaders  bool
	AllowInsecureAdmin bool
	AutocreatePolicy   string
	DNSServer          string
	NSUpdateBinary     string
	TSIGKeyFile        string
	APIPrefix          string
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddress:      envOr("HERMES_LISTEN", ":8080"),
		DatabasePath:       envOr("HERMES_DATABASE", "data/hermes.db"),
		AdminLogin:         os.Getenv("HERMES_ADMIN_LOGIN"),
		AllowedDomains:     splitCSV(os.Getenv("HERMES_DOMAINS")),
		DefaultTTL:         intEnv("HERMES_DEFAULT_TTL", 300),
		AllowWildcard:      boolEnv("HERMES_ALLOW_WILDCARD", false),
		TrustProxyHeaders:  boolEnv("HERMES_TRUST_PROXY_HEADERS", false),
		AllowInsecureAdmin: boolEnv("HERMES_ALLOW_INSECURE_ADMIN", false),
		AutocreatePolicy:   envOr("HERMES_AUTOCREATE_POLICY", "device-prefix"),
		DNSServer:          envOr("HERMES_DNS_SERVER", "127.0.0.1"),
		NSUpdateBinary:     envOr("HERMES_NSUPDATE_BINARY", "/usr/bin/nsupdate"),
		TSIGKeyFile:        os.Getenv("HERMES_TSIG_KEY_FILE"),
		APIPrefix:          "/api/v1",
	}

	if cfg.AdminLogin == "" && !cfg.AllowInsecureAdmin {
		return cfg, fmt.Errorf("HERMES_ADMIN_LOGIN is required unless HERMES_ALLOW_INSECURE_ADMIN=true")
	}
	if cfg.AutocreatePolicy != "device-prefix" && cfg.AutocreatePolicy != "any" {
		return cfg, fmt.Errorf("HERMES_AUTOCREATE_POLICY must be device-prefix or any")
	}
	if len(cfg.AllowedDomains) == 0 {
		return cfg, fmt.Errorf("HERMES_DOMAINS must contain at least one managed DNS zone")
	}
	if cfg.DefaultTTL < 20 || cfg.DefaultTTL > 86400 {
		return cfg, fmt.Errorf("HERMES_DEFAULT_TTL must be between 20 and 86400 seconds")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitCSV(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(item), "."))
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func intEnv(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func boolEnv(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
