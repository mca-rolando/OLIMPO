# OLIMPO

OLIMPO is the shared control, integration, identity, automation, and user-experience layer for the MCA application ecosystem. The initial suite includes **HERMES** for DDNS and dynamic endpoints, **ARGUS** for UniFi multi-site observability, and **METIS** for IT service management.

> **Products remain operationally autonomous; OLIMPO provides shared control-plane capabilities, not mandatory data-plane dependency.**

If OLIMPO is unavailable, HERMES continues DDNS operations, ARGUS continues collecting and evaluating monitoring data, and METIS continues ticketing and ITSM functions. OLIMPO coordinates and enriches the suite; it does not replace product domain logic or become its mandatory data plane.

## Why OLIMPO exists

OLIMPO provides a consistent suite experience and governed integration layer: a shared design system and application shell, application/service registry, canonical cross-product identities, versioned events and APIs, explainable correlation, controlled automation, shared identity policy, global navigation and search, notification routing, maintenance context, audit, and ecosystem health.

OLIMPO does not automatically own product data. HERMES remains authoritative for DDNS state, ARGUS for monitoring observations, and METIS for ticket lifecycle. OLIMPO owns or maintains only control-plane records such as canonical mappings, integration configuration, event/automation state, cached summaries, and audit evidence.

## Status

Current version: **0.1.0-dev**. This repository is in its architecture and governance bootstrap phase. It intentionally contains no production backend, frontend, database, container, or deployment implementation.

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

## Repository structure

`docs/architecture` contains normative platform specifications, `docs/design` contains shared experience guidance, and `docs/adr` records architectural decisions. `.github/workflows` currently documents future automation only.

## Security posture

The baseline requires least privilege, server-side authorization, secure defaults, encrypted transport, auditable changes, short-lived service identities, and external secrets management. No real credentials belong in this repository. See [Identity and security](docs/architecture/identity-security-v1.0.md).

## Future implementation direction

Subject to ADR review, the leading candidates are TypeScript and React with OLIMPO-owned shared packages for the frontend, Python 3 and FastAPI for backend services, PostgreSQL for durable control-plane state, and NATS JetStream behind an internal event abstraction. Redis is optional only where a measured need exists. APIs should be OpenAPI-first and machine-readable contracts versioned.

Licensed under the [MIT License](LICENSE).
