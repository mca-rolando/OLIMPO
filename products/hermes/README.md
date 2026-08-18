# HermesDDNS

HermesDDNS is a centrally managed, self-hosted Dynamic DNS platform designed for UniFi Dream Machine (UDM) fleets. It is derived from Benjamin Bärthlein's `docker-ddns-server` and retains the original MIT attribution while introducing a new device-oriented architecture, hashed DDNS credentials, automatic hostname provisioning, audit logging, release metadata, and a path toward managed UDM agents and MCA-Secure-Downloads integration.

> Current release: **26.08-02** — Managed Agent Foundation / Credential Lifecycle Milestone.

## What 26.08-02 implements

- Everything delivered in 26.08-01: HermesDDNS branding, `hermesddns`/`hermesctl`, SQLite device/host model, hashed DDNS keys, DynDNS-compatible update endpoints, automatic host provisioning, ownership enforcement, BIND9 `nsupdate`, and the bootstrap administration API.
- Hardened DDNS credential authentication with lifecycle states: `pending`, `active`, `grace`, `revoked`, and `expired`.
- Safe DDNS credential rotation state machine with request, stage, validation, grace, completion, rollback, and automatic grace reconciliation.
- Separate credential namespaces for DDNS (`hddns_`), permanent Agent identity (`hagent_`), and one-time enrollment (`henroll_`).
- Secure one-time UDM Agent enrollment/bootstrap with replay protection, confirmation, expiration, and revocation.
- Agent-authenticated lifecycle APIs that never trust a caller-supplied Device ID to establish Agent identity.
- Agent heartbeat and current operational telemetry snapshots with fleet status APIs.
- Current UDM Network Identity snapshots covering WAN identity, public/private/CGNAT classification, upstream/double-NAT context, and LAN VLAN/subnet summaries.
- Validation that a network snapshot has at most one default-route WAN.
- Stale A/AAAA protection: live DNS updates remove both address record types before publishing the current address family.
- End-to-end release validation against live SQLite + BIND, including DDNS `good`/`nochg`, Agent lifecycle, Network Identity, credential rotation, and a real persistent-data upgrade from 26.08-01 to 26.08-02.

The **persistent Hermes UDM Agent executable/service is not part of 26.08-02**. This release provides the server-side identity, enrollment, telemetry, network-context, and credential-lifecycle foundation that the Agent will consume.

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

`hermesctl update`, backup/restore, doctor, and release-channel management remain subsequent milestones. Release 26.08-02 provides the server-side primitives for secure UDM Agent enrollment, identity, heartbeat/current telemetry, network identity snapshots, and agent-delivered/confirmed DDNS credential rotation; the persistent UDM Agent executable/service remains a subsequent milestone.

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
- 26.08-02 continues to use localhost `nsupdate`; TSIG configuration is represented in the runtime configuration and remains scheduled for a later security-hardening milestone.

## Upstream attribution

HermesDDNS is a fork/derivative of:

- `benjaminbear/docker-ddns-server` (Benjamin Bärthlein)
- inspired upstream by `dprandzioch/docker-ddns`

See [LICENSE](LICENSE). The legacy upstream source is retained under `legacy/original-source/` during the initial fork phase for comparison and migration purposes.

## Roadmap

1. **26.08-01** — Foundation and first end-to-end DDNS core.
2. **26.08-02** — Server-side managed-Agent foundation: hardened credential lifecycle, Agent identity/enrollment, heartbeat/telemetry, Network Identity, DNS consistency fixes, and release E2E/upgrade validation.
3. **Next milestone** — Persistent Hermes UDM Agent: installation/service lifecycle, local network collection, managed `inadyn` configuration, heartbeat delivery, and automatic DDNS credential rotation execution.
4. Administration UI expansion, hardened DNS/TSIG operations, and fleet workflows.
5. MCA-Secure-Downloads release/update/backup/rollback integration.
