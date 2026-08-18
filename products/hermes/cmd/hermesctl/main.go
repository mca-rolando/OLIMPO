package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/buildinfo"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version":
		info := buildinfo.Current()
		fmt.Printf("HermesDDNS %s\nCommit: %s\nBuild: %s\n", info.Version, info.Commit, info.BuildTime)
	case "status":
		base := strings.TrimRight(envOr("HERMES_URL", "http://127.0.0.1:8080"), "/")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(base + "/health")
		if err != nil {
			fmt.Fprintf(os.Stderr, "HermesDDNS status: ERROR: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var body any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		pretty, _ := json.MarshalIndent(body, "", "  ")
		fmt.Printf("HermesDDNS status: HTTP %d\n%s\n", resp.StatusCode, pretty)
		if resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
func usage() { fmt.Println("Usage: hermesctl <version|status>") }
func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
