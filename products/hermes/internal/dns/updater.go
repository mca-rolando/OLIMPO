package dns

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Updater interface {
	Upsert(hostname, zone, recordType, target string, ttl int, wildcard bool) error
	Delete(hostname, zone, recordType string, wildcard bool) error
}

type NSUpdate struct {
	Binary      string
	Server      string
	TSIGKeyFile string
}

func (n NSUpdate) Upsert(hostname, zone, recordType, target string, ttl int, wildcard bool) error {
	return n.run(buildUpsertScript(n.Server, hostname, zone, recordType, target, ttl, wildcard))
}

func buildUpsertScript(server, hostname, zone, recordType, target string, ttl int, wildcard bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "server %s\n", server)
	fmt.Fprintf(&b, "zone %s\n", zone)

	deleteTypes := []string{recordType}
	if normalized := strings.ToUpper(strings.TrimSpace(recordType)); normalized == "A" || normalized == "AAAA" {
		// Hermes stores one current address family per hostname. Remove both
		// address record types so an A <-> AAAA transition cannot leave a
		// stale record in BIND.
		deleteTypes = []string{"A", "AAAA"}
	}
	for _, deleteType := range deleteTypes {
		fmt.Fprintf(&b, "update delete %s.%s %s\n", hostname, zone, deleteType)
		if wildcard {
			fmt.Fprintf(&b, "update delete *.%s.%s %s\n", hostname, zone, deleteType)
		}
	}

	fmt.Fprintf(&b, "update add %s.%s %d %s %s\n", hostname, zone, ttl, recordType, target)
	if wildcard {
		fmt.Fprintf(&b, "update add *.%s.%s %d %s %s\n", hostname, zone, ttl, recordType, target)
	}
	b.WriteString("send\n")
	return b.String()
}

func (n NSUpdate) Delete(hostname, zone, recordType string, wildcard bool) error {
	var b strings.Builder
	fmt.Fprintf(&b, "server %s\n", n.Server)
	fmt.Fprintf(&b, "zone %s\n", zone)
	fmt.Fprintf(&b, "update delete %s.%s %s\n", hostname, zone, recordType)
	if wildcard {
		fmt.Fprintf(&b, "update delete *.%s.%s %s\n", hostname, zone, recordType)
	}
	b.WriteString("send\n")
	return n.run(b.String())
}

func (n NSUpdate) run(script string) error {
	f, err := os.CreateTemp("", "hermes-nsupdate-*.txt")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(script); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	args := []string{}
	if n.TSIGKeyFile != "" {
		args = append(args, "-k", n.TSIGKeyFile)
	}
	args = append(args, name)
	cmd := exec.Command(n.Binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nsupdate failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if strings.TrimSpace(stdout.String()) != "" {
		return fmt.Errorf("nsupdate unexpected output: %s", strings.TrimSpace(stdout.String()))
	}
	return nil
}
