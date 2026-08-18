# OLIMPO

OLIMPO is an independent project and the shared control, integration, identity, automation, and user-experience layer for the OLIMPO ecosystem. It coordinates **HERMES** for DDNS and dynamic endpoints, **ARGUS** for multi-site observability, **METIS** for IT service management, and future ecosystem applications.

In this documentation, **OLIMPO Ecosystem** means the complete software ecosystem and target monorepo. **OLIMPO Control Plane** means the independently deployable shared control-plane product.

> **Products remain operationally autonomous; OLIMPO provides shared control-plane capabilities, not mandatory data-plane dependency.**

> **Tenant isolation is a security boundary. Tenant data, identity, credentials, events, integrations, automation, operational state, and audit context must not cross tenant boundaries without explicit authorized platform-level behavior.**

> **The OLIMPO ecosystem must support managed-service-provider operation without making any individual customer the architectural owner of the platform.**

> **Repository consolidation does not imply runtime consolidation.**

If OLIMPO is unavailable, HERMES continues DDNS operations, ARGUS continues collecting and evaluating monitoring data, and METIS continues ticketing and ITSM functions. OLIMPO coordinates and enriches the suite; it does not replace product domain logic or become its mandatory data plane.

## Why OLIMPO exists

OLIMPO provides a consistent suite experience and governed integration layer: a shared design system and application shell, application/service registry, canonical cross-product identities, versioned events and APIs, explainable correlation, controlled automation, shared identity policy, global navigation and search, notification routing, maintenance context, audit, and ecosystem health.

OLIMPO does not automatically own product data. HERMES remains authoritative for DDNS state, ARGUS for monitoring observations, and METIS for ticket lifecycle. OLIMPO owns or maintains only control-plane records such as canonical mappings, integration configuration, event/automation state, cached summaries, and audit evidence.

OLIMPO is MSP-first and multi-tenant by design. A platform operator can manage multiple isolated tenants, their organizations and sites, and the applications enabled for each tenant. No customer organization is globally assumed or made the platform's architectural owner, and single-customer deployment remains possible without changing tenant-aware contracts.

## Status

Current version: **0.1.0-dev**. This repository is in its architecture and governance bootstrap phase. It intentionally contains no production backend, frontend, database, container, or deployment implementation.

The current repository contains the OLIMPO architecture and governance baseline. The accepted target state is a monorepo containing the OLIMPO Control Plane, HERMES, ARGUS, METIS, and carefully bounded shared assets. Product source histories have not yet been imported; migration remains in the planning state and requires separate human approval. Each product will remain independently buildable, testable, deployable, versionable, releasable, observable, and recoverable.

## Documentation

- [Documentation index](docs/README.md)
- [Architecture and functional specification](docs/architecture/architecture-functional-spec-v1.0.md)
- [Control-plane model](docs/architecture/control-plane-v1.0.md)
- [Integration model](docs/architecture/integration-model-v1.0.md)
- [Common entity model](docs/architecture/common-entity-model-v1.0.md)
- [Event model](docs/architecture/event-model-v1.0.md)
- [Automation model](docs/architecture/automation-model-v1.0.md)
- [Identity and security](docs/architecture/identity-security-v1.0.md)
- [Observability and audit](docs/architecture/observability-audit-v1.0.md)
- [Autonomy and resilience](docs/architecture/autonomy-resilience-v1.0.md)
- [Design system](docs/design/design-system-v1.0.md)
- [Architecture decisions](docs/adr/0001-olimpo-is-a-control-plane.md)
- [Suite compatibility governance](docs/governance/suite-compatibility-policy-v1.0.md)
- [Monorepo migration planning](docs/migration/README.md)
- [Monorepo boundaries](docs/governance/monorepo-boundaries-v1.0.md)
- [Versioning and release policy](docs/governance/versioning-release-policy-v1.0.md)

## Repository structure

`docs/architecture` contains normative platform specifications, `docs/design` contains shared experience guidance, and `docs/adr` records architectural decisions. `.github/workflows` currently documents future automation only.

## Security posture

The baseline requires least privilege, server-side authorization, secure defaults, encrypted transport, auditable changes, short-lived service identities, and external secrets management. No real credentials belong in this repository. See [Identity and security](docs/architecture/identity-security-v1.0.md).

## Future implementation direction

Subject to ADR review, the leading candidates are TypeScript and React with OLIMPO-owned shared packages for the frontend, Python 3 and FastAPI for backend services, PostgreSQL for durable control-plane state, and NATS JetStream behind an internal event abstraction. The package namespace remains unresolved. Redis is optional only where a measured need exists. APIs should be OpenAPI-first and machine-readable contracts versioned.

Licensed under the [MIT License](LICENSE).
