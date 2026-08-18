# Monorepo Boundaries v1.0

> **Products remain operationally autonomous; OLIMPO provides shared control-plane capabilities, not mandatory data-plane dependency.**

> **Tenant isolation is a security boundary. Tenant data, identity, credentials, events, integrations, automation, operational state, and audit context must not cross tenant boundaries without explicit authorized platform-level behavior.**

> **Repository consolidation does not imply runtime consolidation.**

## Product boundaries

| Product | Domain authority | Required independent boundary |
|---|---|---|
| OLIMPO Control Plane | Canonical cross-product mappings, control-plane integrations/automation, shared policy metadata, cached summaries, and platform audit | Build, tests, deployment, version, release, data store, observability, recovery, and failure handling |
| HERMES | DDNS domains, clients/agents, records, credentials, and DDNS operation | Same independent boundaries; OLIMPO outage cannot stop routine DDNS |
| ARGUS | Monitoring observations, health evaluation, alerts, and monitoring operational state | Same independent boundaries; collection and evaluation continue without OLIMPO |
| METIS | Tickets, incidents, requests, workflows, SLA/OLA, and ITSM lifecycle | Same independent boundaries; ITSM continues without OLIMPO |

One repository does not authorize direct database access, in-process calls across deployable services, shared credentials, implicit trust, lockstep release, or one mandatory deployment. Shared code cannot bypass versioned APIs/events, tenant authorization, service identities, data ownership, or audit.

## Repository ownership

Path ownership and review rules should require product-owner review for product paths and cross-product review for shared packages/schemas. Changes to shared contracts identify affected products and run their compatibility tests. A root dependency graph is advisory for impact selection, not a license to couple runtime internals.

## Data and security

Repository visibility is not a runtime security mechanism and does not weaken runtime isolation. HERMES, ARGUS, METIS, and the OLIMPO Control Plane retain separate data authority. Tenant context, RBAC, service identities, APIs, events, secrets, caches, background jobs, and audit are enforced within each deployable boundary. Shared test fixtures use synthetic data and cannot contain credentials or customer information.

## CI/CD

CI uses path and dependency impact:

- `products/hermes/**` runs HERMES build, unit, integration, security, and release-contract checks.
- `products/argus/**` runs ARGUS checks, including agent/appliance-safety and NOC checks when relevant.
- `products/metis/**` runs METIS architecture/implementation checks.
- `platform/olimpo/**` runs OLIMPO Control Plane checks.
- Shared package or schema changes run the package checks plus every declared consumer's contract and build tests.
- Root governance/documentation changes run Markdown, link, policy, and affected-contract validation without deploying products.

Each product publishes and deploys only through its own explicitly approved workflow. A suite validation workflow may aggregate evidence but cannot become the only way to build or release a product.

## MSP alignment

The monorepo makes tenancy changes visible across products but does not prove adoption. HERMES needs a future tenant model for relevant agents, domains, records, credentials, and events. ARGUS needs alignment from its current organization boundary to the accepted Platform/Tenant/Organization/Site hierarchy. METIS needs vendor-neutral per-tenant identity and customer scope rather than a single organization assumption. Each alignment requires product-owned review, ADRs where necessary, migration design, and cross-tenant negative tests.

## Design System

The OLIMPO Design System is a shared package direction for all four products. It preserves Light, Dark, and System modes, the upper-right theme selector, shared AppShell/navigation patterns, semantic colors, WCAG 2.2 AA, 1920x1080 dashboard validation, and kiosk/NOC capability. Product branding may use restrained accents but cannot alter semantic status meaning or accessibility. Adoption is independently versioned and must not create an OLIMPO Control Plane runtime dependency.
