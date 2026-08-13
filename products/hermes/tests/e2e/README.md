# HermesDDNS end-to-end stabilization tests

These tests exercise the real HermesDDNS container, SQLite database, BIND9, `nsupdate`, DynDNS-compatible endpoint, server-side Agent APIs, and release-to-release database migration path.

## Full run

From the repository root:

```bash
tests/e2e/run.sh
```

The test suite intentionally does **not** publish host TCP/UDP port 53. Each test publishes only a temporary localhost HTTP port and runs `dig` inside the Hermes container against its own BIND instance. This avoids conflicts with `systemd-resolved` or another local DNS service.

The suite requires Docker, Git, curl, Python 3, and the local Git tag/ref `26.08-01`. The upgrade test uses `git archive` to build the actual historical source from that ref, creates persistent data under the old version, then starts the current code against the same persistent SQLite and BIND volumes.

## Coverage

`server-e2e.sh` verifies:

- `/health` startup.
- Device creation and one-time DDNS key return.
- real `/nic/update` `good` and `nochg` responses.
- real BIND `A` record creation through `nsupdate`.
- live A-to-AAAA and AAAA-to-A transitions without stale records.
- one-time `henroll_` enrollment exchange.
- permanent `hagent_` identity binding and enrollment confirmation.
- heartbeat/current telemetry and online Agent status.
- network identity reporting and Double-NAT classification.
- rejection of multiple WAN default routes.
- Agent-driven `hddns_` candidate staging, validation, DDNS confirmation, and transition to grace.
- previous and replacement DDNS credentials both working during grace.
- DDNS audit logging.

`upgrade-26.08-01-to-26.08-02.sh` verifies:

- actual `26.08-01` source starts and creates persistent SQLite/BIND state.
- the current code opens and migrates that same database.
- pre-upgrade Device, DDNS credential, Host, DNS record, and audit logs remain usable.
- all new 26.08-02 schema areas are exercised through Agent enrollment, identity credential, heartbeat telemetry, network identity, and credential rotation APIs.

Temporary containers and Docker volumes are removed automatically. Docker build images are retained to make repeated runs faster.
