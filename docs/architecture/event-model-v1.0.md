# Common Event and Correlation Model v1.0

> **Tenant isolation is a security boundary. Tenant data, identity, credentials, events, integrations, automation, operational state, and audit context must not cross tenant boundaries without explicit authorized platform-level behavior.**

> **The OLIMPO ecosystem must support managed-service-provider operation without making any individual customer the architectural owner of the platform.**

## Envelope

Events are immutable facts expressed through a transport-independent OLIMPO abstraction. A conceptual envelope is:

```json
{
  "schema_version": "1.0",
  "tenant_id": "TENANT-HYPOTHETICAL-A",
  "event_id": "01HYPOTHETICALULID00000000",
  "source": "ARGUS",
  "type": "argus.site.offline",
  "timestamp": "2026-08-17T14:30:00Z",
  "severity": "critical",
  "correlation_id": "CORR-HYPOTHETICAL-001",
  "causation_id": "EVENT-HYPOTHETICAL-PARENT",
  "entity_refs": [{"type": "site", "canonical_id": "SITE-HYPOTHETICAL-MIA-04"}],
  "payload": {}
}
```

Required fields for a tenant-owned event are schema version, trustworthy tenant ID, globally unique event ID, registered source, namespaced type, UTC timestamp, severity, correlation ID, entity references, and payload. Causation ID is required for derived events/actions and null only for a root fact. Severity values are `critical`, `warning`, `informational`, and `unknown`; presentation may map `healthy` as state rather than incident severity.

The producer derives tenant context from an authenticated workload binding, tenant-owned resource, trusted adapter configuration, or another validated server-side relationship. A client-supplied `tenant_id` is never authoritative by itself. The producer rejects disagreement between claimed, authenticated, resource, and integration scope.

Initial event namespaces may include `argus.site.offline`, `argus.device.offline`, `argus.vpn.down`, `hermes.client.offline`, `hermes.dns.update.failed`, `hermes.public_ip.changed`, `metis.ticket.created`, `metis.ticket.assigned`, `metis.incident.resolved`, and `metis.sla.breached`.

## Flow and delivery

```mermaid
sequenceDiagram
  participant P as Product producer
  participant L as Local outbox/queue
  participant B as OLIMPO event abstraction
  participant C as Consumer/correlation
  participant D as Dead-letter handling
  P->>L: Persist domain fact and event intent
  L->>B: Publish envelope
  B-->>L: Acknowledge durable acceptance
  B->>C: Deliver at least once
  C->>C: Validate and deduplicate event_id
  alt transient failure
    C-->>B: Retry with backoff and jitter
  else terminal/exhausted
    B->>D: Preserve envelope and reason
  end
```

At-least-once delivery means duplicates are normal. Consumers persist an idempotency result keyed by tenant, event ID, and action scope. Event subjects or streams, partition keys, retention, replay authorization, dead-letter records, telemetry, and operator tools preserve tenant context. Ordering is guaranteed only where a future transport contract explicitly provides a partition/order key; consumers use timestamps, domain versions, and monotonic source sequences when supplied, and must not assume global ordering.

Retries use bounded exponential backoff with jitter and explicit maximum age/attempt policy. Poison or expired events enter a dead-letter facility with reason, attempts, schema information, and replay authorization. Replay preserves the original event ID and adds replay metadata outside the immutable business payload. Operators can inspect, repair mapping/schema issues, and replay safely.

## Schema evolution and consumer duties

Envelope and payload versions are separate. Compatible evolution adds optional fields and enum values defensively. Breaking changes use a new major schema/type version and overlap through a published deprecation window. Producers validate before publish; consumers validate, ignore unknown optional fields, reject unsupported major versions visibly, protect sensitive data, deduplicate, authorize effects, and record outcomes.

## Explainable correlation

Correlation occurs within one tenant boundary by default. Correlation indexes, windows, topology state, caches, keys, outputs, and incident candidates include tenant scope. Events from different tenants must never be correlated. An explicitly designed platform health function may aggregate minimum necessary signals across tenants only with platform authorization, audit, and output controls that prevent customer data disclosure to another tenant.

Correlation evaluates canonical entity, topology, time window, severity, maintenance state, and prior conditions. It may combine gateway, switch, access-point, and camera failures at one site into one parent outage, or combine HERMES DDNS failure with ARGUS gateway-unreachable and VPN-down evidence into a possible connectivity outage.

Each correlation output records rule/version, contributing event IDs, entity mappings, window, maintenance context, confidence or deterministic reason, suppressed candidates, and operator overrides. Correlation never destroys source events. False-positive review and rule rollback are supported.

```mermaid
flowchart LR
  G[Gateway offline] --> W[Site/time correlation window]
  S[Switches offline] --> W
  A[APs offline] --> W
  C[Cameras offline] --> W
  W --> O[Explainable parent site outage]
  O --> I[One incident candidate]
```
