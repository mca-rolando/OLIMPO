# HermesDDNS 26.08-02 — Stabilization Block 1

This block intentionally adds no new product feature. It hardens three release-critical behaviors before end-to-end and upgrade testing.

## DNS address-family transitions

Hermes stores one current address family per managed hostname. `NSUpdate.Upsert` now removes both `A` and `AAAA` records before adding the current address record. Wildcard companion records receive the same treatment. This prevents a stale IPv4 record from surviving an IPv4-to-IPv6 transition, or a stale IPv6 record from surviving the reverse transition.

Non-address record types continue to delete only their own record type.

## Network identity default route

A network identity report may contain zero or one WAN marked `default_route`. Reports containing more than one default WAN are rejected as ambiguous. This prevents Hermes from applying the server-observed public address to multiple WANs and producing misleading NAT classification.

## Automatic credential grace reconciliation

Credential rotations in `grace` are now reconciled automatically by the HermesDDNS server. The reconciler runs once at process startup and then every 60 seconds. Expired previous DDNS credentials are changed from `grace` to `revoked`, and their rotations are changed from `grace` to `completed`.

The existing administrative reconcile endpoint remains available for explicit operator use.

## Validation added

Unit coverage now includes:

- A-to-AAAA and AAAA-to-A stale-record protection, including wildcard records.
- rejection of multiple WAN default routes.
- automatic completion of an expired DDNS credential grace period.

The next stabilization block is reserved for Docker/BIND end-to-end testing and a real 26.08-01 to 26.08-02 database upgrade test.

# HermesDDNS 26.08-02 — Stabilization Block 2

This block adds release-validation infrastructure rather than product features. Its purpose is to prove the 26.08-02 server behavior against a real containerized BIND instance and to prove that an existing 26.08-01 installation can be started by 26.08-02 without losing its operational data.

## Live server / SQLite / BIND E2E

`tests/e2e/server-e2e.sh` builds the current HermesDDNS container and runs it with isolated temporary Docker volumes for SQLite and BIND. The test publishes only a temporary localhost HTTP port; DNS port 53 remains inside the container so a workstation already using `systemd-resolved` can run the suite without a host port conflict.

The test performs real `/nic/update` calls and validates BIND with `dig` inside the container. It covers initial `good`, repeated `nochg`, A-to-AAAA and AAAA-to-A transitions, and confirms that the opposite address family does not remain stale.

The same test then exercises the complete server-side Agent path available in 26.08-02: one-time enrollment, permanent Agent identity, enrollment confirmation, heartbeat/current telemetry, network identity, multiple-default-route rejection, credential rotation request, Agent-generated candidate registration, validation, DDNS confirmation, and grace operation.

## Real 26.08-01 to 26.08-02 upgrade

`tests/e2e/upgrade-26.08-01-to-26.08-02.sh` uses the local Git ref `26.08-01` to build the historical release source. It creates a Device, DDNS credential, Host, BIND record, and audit history with that old container. The old container is then removed while its SQLite and BIND directories are preserved.

The current 26.08-02 image starts against those same directories. The test verifies that the original Device and DDNS credential still work, the Host/DNS state remains valid, and audit history is preserved and extended. It then exercises the new Agent enrollment/identity, heartbeat telemetry, network identity, and credential rotation areas, proving that the new AutoMigrate schema is usable after the real upgrade.

## Continuous integration

GitHub Actions now runs the E2E suite after the normal Go test/vet/build job. The E2E checkout uses full Git history so the `26.08-01` release ref is available to the upgrade test.

Run the entire stabilization suite locally with:

```bash
tests/e2e/run.sh
```

# HermesDDNS 26.08-02 — Release Closing Block

The release-closing block freezes the 26.08-02 release metadata after both stabilization blocks passed. It does not add product behavior.

The closing block updates the root `VERSION`, Docker Compose build default, example environment header, README release status/roadmap, and CHANGELOG so the repository consistently identifies 26.08-02 as the current release. Historical 26.08-01 references used by compatibility comments, architecture examples, import history, and the real upgrade E2E remain intentionally unchanged.

Before tagging or publishing 26.08-02, the closing commit must again pass formatting/diff checks, Go tests, race detection, vet/build, and the full `tests/e2e/run.sh` release suite.
