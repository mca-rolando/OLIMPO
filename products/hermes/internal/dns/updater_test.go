package dns

import (
	"strings"
	"testing"
)

func TestBuildUpsertScriptRemovesBothAddressFamiliesBeforeA(t *testing.T) {
	script := buildUpsertScript("127.0.0.1", "router", "example.com", "A", "203.0.113.10", 60, false)

	want := []string{
		"update delete router.example.com A",
		"update delete router.example.com AAAA",
		"update add router.example.com 60 A 203.0.113.10",
	}
	for _, line := range want {
		if !strings.Contains(script, line+"\n") {
			t.Fatalf("script missing %q:\n%s", line, script)
		}
	}
	if strings.Index(script, want[0]) > strings.Index(script, want[2]) || strings.Index(script, want[1]) > strings.Index(script, want[2]) {
		t.Fatalf("address-family deletes must happen before add:\n%s", script)
	}
}

func TestBuildUpsertScriptRemovesBothAddressFamiliesBeforeAAAAWithWildcard(t *testing.T) {
	script := buildUpsertScript("127.0.0.1", "router", "example.com", "AAAA", "2001:db8::10", 120, true)

	want := []string{
		"update delete router.example.com A",
		"update delete *.router.example.com A",
		"update delete router.example.com AAAA",
		"update delete *.router.example.com AAAA",
		"update add router.example.com 120 AAAA 2001:db8::10",
		"update add *.router.example.com 120 AAAA 2001:db8::10",
	}
	for _, line := range want {
		if !strings.Contains(script, line+"\n") {
			t.Fatalf("script missing %q:\n%s", line, script)
		}
	}
}

func TestBuildUpsertScriptKeepsNonAddressRecordDeletionScoped(t *testing.T) {
	script := buildUpsertScript("127.0.0.1", "router", "example.com", "TXT", "value", 60, false)
	if !strings.Contains(script, "update delete router.example.com TXT\n") {
		t.Fatalf("TXT delete missing:\n%s", script)
	}
	if strings.Contains(script, "update delete router.example.com A\n") || strings.Contains(script, "update delete router.example.com AAAA\n") {
		t.Fatalf("non-address upsert must not delete A/AAAA:\n%s", script)
	}
}
