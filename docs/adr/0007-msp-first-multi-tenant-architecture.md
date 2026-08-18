# ADR 0007 — Adopt an MSP-First Multi-Tenant Architecture

- Status: Accepted
- Date: 2026-08-17

## Context

OLIMPO, HERMES, ARGUS, and METIS are independent projects. Their architecture must not assign ownership to an employer, customer, tenant, hosting organization, or future commercial brand. The intended direction includes outsourced and managed network administration in which a platform operator may serve multiple independent customers. A single-customer model would embed assumptions that are expensive and unsafe to remove later, particularly in identity, authorization, mappings, events, credentials, automation, and audit.

The existing autonomy decision remains mandatory:

> **Products remain operationally autonomous; OLIMPO provides shared control-plane capabilities, not mandatory data-plane dependency.**

## Decision

OLIMPO is MSP-first and multi-tenant by design. The primary long-term operating model includes a platform owner or managed service provider operating OLIMPO for multiple isolated tenants. No individual customer is the architectural owner of the platform.

> **Tenant isolation is a security boundary. Tenant data, identity, credentials, events, integrations, automation, operational state, and audit context must not cross tenant boundaries without explicit authorized platform-level behavior.**

> **The OLIMPO ecosystem must support managed-service-provider operation without making any individual customer the architectural owner of the platform.**

Customer-specific functionality is tenant-scoped rather than globally assumed. A tenant may contain one or more organizations, and organizations contain sites and locations. Tenant-owned resources carry trustworthy tenant context wherever necessary for isolation; identifiers are not added to objects without customer context. Single-customer deployments remain possible by operating one tenant with the same tenant-aware contracts.

Products remain operationally autonomous and authoritative for their domains. This decision does not select shared versus dedicated databases, processes, clusters, regions, or deployments. Physical deployment topology requires later evaluation and ADR approval.

## Security implications

No tenant data leakage is a first-class security requirement. Server-side authorization, tenant-aware data access, cache and search namespaces, event routing, correlation state, integrations, secret references, automation targets, background jobs, notifications, maintenance windows, feature flags, health projections, deep links, audit, and future files must preserve tenant scope. Cross-tenant access is denied by default. Explicit platform-level behavior uses minimum necessary data, strong authorization, and complete audit. PostgreSQL Row-Level Security is a possible future defense in depth, not selected here.

## Data-model implications

Platform, Tenant, Organization, Site, and Location are distinct concepts. Canonical mappings for customer-context entities include `tenant_id`; `organization_id`, `site_id`, and `location_id` appear only when that narrower scope applies. Product-native identifiers are unique only inside their documented tenant and source scope. Tenant-aware indexes, uniqueness constraints, idempotency keys, and test fixtures are required in future designs.

## Identity implications

Identity configuration is tenant-aware. Different tenants may use Microsoft Entra ID, another OIDC provider, or controlled local identity where supported. Entra remains a preferred enterprise provider, not a single global company boundary. Platform/MSP privileges are distinct from tenant privileges; a user can have different capability mappings per tenant. Bootstrap and platform emergency access remain controlled and strongly audited.

## Product implications

- HERMES should associate relevant agents, clients, domains, records, credentials, and events with tenant context while remaining authoritative for DDNS state.
- ARGUS should associate sites, consoles, devices, VPNs, observations, alerts, and integrations with tenant context while remaining authoritative for monitoring state.
- METIS should associate tickets, incidents, requests, requesters, queues, technicians, SLA, service catalog, assets, and integrations with tenant/customer context while remaining authoritative for ITSM lifecycle. OLIMPO may provide tenant or service-tier context without owning SLA logic.
- Future ecosystem applications must declare tenant scope and compatibility through versioned contracts.

These are required alignment directions, not claims of current implementation.

## Consequences

The model supports MSP and single-customer operation without separate core contracts, permits tenant-specific identity and capabilities, and makes isolation review explicit. It adds authorization, data-model, testing, operations, and audit complexity. Every cross-cutting change needs suite impact review, and every product must adopt compatible tenant-aware contracts before multi-tenant integration is considered complete.

## Alternatives considered

- **Single-customer-only architecture:** rejected because it makes one customer a global assumption and creates unsafe future migration work.
- **Separate deployment per customer only:** not required as the sole model. It can improve physical isolation but multiplies operations and does not remove the need for tenant-aware contracts, platform authorization, or safe MSP oversight.
- **Multi-tenant shared platform:** selected as the logical architecture because it preserves future commercial flexibility and consistent governance. Shared physical deployment is permitted but not mandated; dedicated or hybrid topologies may be chosen later without changing the logical tenant model.
