# Changelog

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
