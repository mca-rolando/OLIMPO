# HermesDDNS

HermesDDNS is a centrally managed, self-hosted Dynamic DNS platform designed for UniFi Dream Machine (UDM) fleets. It is derived from Benjamin Bärthlein's `docker-ddns-server` and retains the original MIT attribution while introducing a new device-oriented architecture, hashed DDNS credentials, automatic hostname provisioning, audit logging, release metadata, and a path toward managed UDM agents and MCA-Secure-Downloads integration.

> Current release: **26.08-01** — Foundation / Core Milestone.

> Development branch **26.08-02** currently includes hardened DDNS credential lifecycle, permanent Agent identity, one-time Agent enrollment/bootstrap, Agent heartbeat/current operational telemetry APIs, and current UDM network identity snapshots with WAN public/private/CGNAT/Double-NAT classification plus VLAN/subnet context. The published release remains 26.08-01 until the 26.08-02 release is formally closed.

## What 26.08-01 implements

- HermesDDNS branding and repository/module ownership.
- `hermesddns` server and initial `hermesctl` CLI.
- Version/build metadata and `/health` endpoint.
- New SQLite data model: Domain, Device, Host, DDNSCredential, UpdateLog.
- High-entropy DDNS API keys stored only as SHA-256 hashes.
- DynDNS-compatible endpoints: `/update`, `/nic/update`, `/v2/update`, `/v3/update`.
- Automatic host creation on the first valid device update.
- Device-to-host ownership enforcement.
- `good <ip>` and `nochg <ip>` responses.
- BIND9 updates through `nsupdate`.
- Minimal administrative REST API for bootstrap/testing, including manual credential rotation primitives.
- Docker Compose deployment foundation.
- Architecture & Functional Specification v1.0 source documentation and UI storyboards.

## Quick start (development)

```bash
cp .env.example .env
vi .env

docker compose build
docker compose up -d
curl http://127.0.0.1:8080/health
```

### Bootstrap a managed domain

If `HERMES_ADMIN_LOGIN` is configured, add `-u 'admin:password'` to the API calls.

```bash
curl -X POST http://127.0.0.1:8080/api/v1/domains \
  -H 'Content-Type: application/json' \
  -d '{"name":"ddns.example.com","default_ttl":300}'
```

### Create a UDM device identity

```bash
curl -X POST http://127.0.0.1:8080/api/v1/devices \
  -H 'Content-Type: application/json' \
  -d '{"name":"COR-P-TEST","display_name":"Test UDM","type":"UDM-SE"}'
```

The response includes the DDNS API key **once**. The database stores only the key hash. By default, automatic host creation is restricted to the device name or a device-name prefix such as `cor-p-test-wan2`; set `HERMES_AUTOCREATE_POLICY=any` only if you intentionally want unrestricted first-claim behavior within managed zones.

### Test DDNS update

```bash
curl -u 'COR-P-TEST:<APIKEY>' \
  'http://127.0.0.1:8080/nic/update?hostname=cor-p-test.ddns.example.com&myip=203.0.113.10'
```

Expected first response:

```text
good 203.0.113.10
```

Repeat with the same address:

```text
nochg 203.0.113.10
```

## CLI

```bash
hermesctl version
hermesctl status
```

`hermesctl update`, backup/restore, doctor, and release-channel management remain subsequent milestones. The 26.08-02 development branch now implements secure UDM Agent enrollment, Agent identity, heartbeat/current telemetry, network identity snapshots, and agent-delivered/confirmed DDNS credential rotation APIs; the actual persistent UDM Agent executable/service remains a subsequent milestone.

## Repository layout

```text
cmd/                  Executables (server and CLI)
internal/             Hermes application/core packages
deployment/docker/    Container runtime foundation
docs/                  Architecture, storyboards, legacy UI reference
legacy/                Original upstream source retained for traceability
scripts/               Build and smoke-test helpers
tests/                 End-to-end test area
```

## Security notes

- Administrative API authentication is required by default. For isolated development only, `HERMES_ALLOW_INSECURE_ADMIN=true` can disable that startup requirement.
- Do not expose port 8080 directly to the public Internet in production. Put Hermes behind HTTPS (Caddy is the planned front end).
- `HERMES_TRUST_PROXY_HEADERS` is disabled by default. Enable it only behind a trusted reverse proxy.
- API keys are generated from 256 bits of random secret material and stored only as SHA-256 hashes.
- 26.08-01 supports localhost `nsupdate`; TSIG configuration is represented in the runtime configuration and will be hardened in the next security milestone.

## Upstream attribution

HermesDDNS is a fork/derivative of:

- `benjaminbear/docker-ddns-server` (Benjamin Bärthlein)
- inspired upstream by `dprandzioch/docker-ddns`

See [LICENSE](LICENSE). The legacy upstream source is retained under `legacy/original-source/` during the initial fork phase for comparison and migration purposes.

## Roadmap

1. **26.08-01** — Foundation and first end-to-end DDNS core.
2. Hardened DNS/TSIG, key lifecycle and rotation primitives.
3. REST API expansion and redesigned administration UI.
4. Hermes UDM Agent: enrollment, heartbeat, managed DDNS configuration, key rotation.
5. MCA-Secure-Downloads release/update/rollback integration.
