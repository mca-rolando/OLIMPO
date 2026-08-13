# Changelog

## 26.08-02 — 2026-08-13

### Added
- DDNS credential lifecycle states and a safe server-side rotation state machine with request, stage, validation, grace, completion, rollback, and reconciliation.
- Permanent Device Identity Credentials using the `hagent_` namespace, separate from `hddns_` DDNS credentials.
- One-time `henroll_` Agent enrollment tokens with expiration, single-use consumption, confirmation, revocation, and replay protection.
- Agent-authenticated identity and credential-lifecycle API endpoints.
- Agent heartbeat and current operational telemetry snapshots with per-device and fleet status APIs.
- Current UDM Network Identity snapshots for WAN identity, public/private/CGNAT classification, upstream/double-NAT context, and LAN VLAN/subnet summaries.
- Automatic server-side reconciliation of expired DDNS credential grace periods.
- Docker/BIND end-to-end release tests covering DDNS, Agent enrollment/identity, heartbeat, Network Identity, credential rotation, and persistent-data upgrades.
- CI execution of the release E2E suite with full Git history for historical upgrade testing.

### Changed
- DDNS authentication now enforces credential lifecycle state and records successful credential usage metadata.
- The legacy administrative rotation endpoint now requests a safe rotation instead of minting a replacement plaintext key server-side.
- Agent identity is derived only from the authenticated `hagent_` credential rather than caller-supplied Device identifiers.
- Network-context persistence stores one current snapshot per UDM and replaces current WAN/LAN child rows transactionally.
- Fleet Network Identity retrieval uses bounded queries instead of per-device lookups.
- Release metadata, Compose build defaults, and example environment metadata now identify 26.08-02.

### Fixed
- Live BIND updates now remove stale A and AAAA records before publishing the active address family, preventing obsolete cross-family DNS answers.
- Network Identity validation now rejects snapshots containing more than one default-route WAN.

### Validated
- `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...`.
- Live Docker E2E with HermesDDNS + SQLite + authoritative BIND, including `good`/`nochg` behavior and A/AAAA transitions.
- Complete server-side Agent enrollment, heartbeat, Network Identity, and DDNS credential-rotation path through grace.
- Real 26.08-01 to 26.08-02 upgrade using persistent SQLite/BIND data, preserving Device, DDNS credential, Host, DNS state, and audit history.

## 26.08-01 — 2026-08-12

### Added
- Official HermesDDNS fork foundation and branding.
- New module path `github.com/mca-rolando/HermesDDNS`.
- `hermesddns` and `hermesctl` executables.
- Build/version metadata and health endpoints.
- Device-oriented SQLite schema.
- Hashed DDNS API keys.
- Automatic host provisioning for authenticated devices.
- Device ownership enforcement for hostnames.
- DynDNS-compatible update endpoints with `good <IP>` and `nochg <IP>`.
- BIND9 `nsupdate` adapter.
- Bootstrap administration API for Domains, Devices, and Logs.
- Docker Compose foundation and BIND setup.
- Architecture/specification documentation and conceptual UI storyboards.
- Manual DDNS credential rotation primitive with configurable grace period.
- Secure-by-default administrative API startup requirement.
- Device-prefix hostname auto-provisioning policy.

### Changed
- TheBBCloud naming is removed from active application code.
- Plaintext per-host username/password authentication is replaced by Device + hashed API-key authentication.

### Preserved
- Original upstream code and UI under `legacy/` and `docs/legacy-ui/` for traceability during the fork transition.
