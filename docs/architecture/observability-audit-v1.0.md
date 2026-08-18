# Observability and Audit Model v1.0

## Observability standards

Every future component emits structured logs, metrics, distributed traces where useful, liveness/health and readiness signals, and stable request, correlation, causation, and event identifiers. Instrumentation should be OpenTelemetry-compatible and vendor-neutral at the application boundary.

```mermaid
flowchart LR
  H[HERMES telemetry] --> C[Product-local collection]
  A[ARGUS telemetry] --> C2[Product-local collection]
  M[METIS telemetry] --> C3[Product-local collection]
  O[OLIMPO telemetry] --> OC[OLIMPO collection]
  C --> B[Authorized observability backends]
  C2 --> B
  C3 --> B
  OC --> B
  B --> D[Dashboards / alerts]
  B --> X[Cross-product correlation by IDs]
  O -. outage does not stop .-> C
```

Logs use timestamps, severity, service/version/environment, operation, result, latency, and relevant IDs. Metrics have bounded-cardinality labels. Traces sample intentionally and propagate context across authorized boundaries. Health checks distinguish process liveness, dependency readiness, and degraded optional capabilities; they reveal no sensitive configuration.

Telemetry excludes passwords, tokens, cookies, authorization headers, secret references that reveal provider paths, and unnecessary personal data. Redaction occurs before export. Access, retention, regional handling, and deletion follow classification and policy.

Tenant-owned telemetry preserves tenant context through authorized collection and querying, while exporters minimize identifiers and prevent tenant data from entering another tenant's dashboard. Logs, traces, health projections, alert routes, and bounded-cardinality metrics use tenant-aware access controls and storage/query boundaries. Platform aggregate health may use minimum necessary signals under explicit platform authorization; it must not expose customer detail across tenants.

## Ecosystem health

The OLIMPO dashboard can summarize HERMES, ARGUS, METIS, integrations (for example `12/12 Healthy`), event pipeline, automations, notifications, identity connectivity, and recent significant events. Status includes evidence, source, last observation, and staleness. Unknown is not reported as healthy. Product-local telemetry remains operational if OLIMPO is down.

## Immutable-oriented audit

Audit records carry tenant or explicit platform scope. Tenant audit records, indexes, exports, retention jobs, and access policy are isolated by tenant. Platform-level access records the authorizing capability, purpose, affected tenant set, and outcome. Cross-tenant administrative activity and denied leakage attempts are high-risk audit events. Audit context is preserved through event replay, automation retry, background work, and product reconciliation.

Audit records answer who/what/when/where, source, target, result, correlation ID, request ID, actor and service identity, policy/automation version, and appropriate redacted before/after metadata. Examples include a user changing the ARGUS-to-METIS automation and ARGUS-caused creation of `METIS-10482` linked to its originating event.

Audit is append-only through normal application interfaces, tamper-evident where feasible, access-controlled, exportable, time-synchronized, and governed by retention/legal policy. Corrections append a linked record. High-risk actions generate timely security alerts. Audit ingestion failure causes defined behavior: consequential administration fails closed or durably queues evidence; routine product domain work continues locally and later forwards its audit evidence.

Operational logs support diagnosis and may be sampled or rotated. Audit records establish accountability and must not be silently sampled, edited, or treated as general debug storage. Both avoid secret values.

## SLO and alert direction

Service-level objectives, recovery objectives, and alert thresholds are not fixed in this phase. They must be defined per capability, recognizing that control-plane availability and each product's essential domain availability are separate. Alerts should be actionable, deduplicated, severity-consistent, and linked to runbooks and correlated evidence.
