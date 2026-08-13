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
