# ADR 0008 — Adopt the OLIMPO Monorepo

- Status: Accepted
- Date: 2026-08-17

## Context

The OLIMPO Ecosystem comprises four independently deployable products: the OLIMPO Control Plane, HERMES, ARGUS, and METIS. Cross-cutting tenancy, identity, contracts, events, accessibility, and governance benefit from one review surface, but repository fragmentation increases coordination and compatibility cost.

The existing autonomy and tenant-isolation decisions remain mandatory:

> **Products remain operationally autonomous; OLIMPO provides shared control-plane capabilities, not mandatory data-plane dependency.**

> **Tenant isolation is a security boundary. Tenant data, identity, credentials, events, integrations, automation, operational state, and audit context must not cross tenant boundaries without explicit authorized platform-level behavior.**

## Decision

The OLIMPO Ecosystem will use a single source-code repository containing the OLIMPO Control Plane, HERMES, ARGUS, METIS, shared packages, shared schemas, shared architecture, and ecosystem governance.

> **Repository consolidation does not imply runtime consolidation.**

One repository does not mean one application. Each product remains independently buildable, testable, deployable, versionable, releasable, observable, and recoverable. Products retain their domain ownership, data ownership, service identities, authorization enforcement, runtime failure boundaries, and ability to operate without a mandatory OLIMPO Control Plane data-plane dependency.

Independent product versions and releases continue. Repository-wide validation may exist, but routine product work and releases must not require unrelated products to build or deploy. Shared packages contain cross-product primitives and generated clients only where ownership and compatibility are explicit; they must not absorb DDNS, monitoring, or ITSM domain logic or create mandatory runtime coupling.

## Security implications

Shared source visibility does not weaken tenant or runtime security boundaries. Service-to-service authentication, least privilege, tenant-aware authorization, secret isolation, API/event authorization, and independent data ownership remain enforced at runtime. Path ownership, supply-chain controls, affected-product tests, and release permissions provide repository-level defense in depth.

## Operational implications

CI and release automation will be path- and dependency-aware. Each product keeps an isolated build/test entry point, deployable artifact set, operational telemetry, backup/restore procedure, and rollback path. A customer may deploy any supported product combination; no giant suite deployment is mandatory.

## Consequences

Cross-product contracts and governance become easier to discover and review, atomic compatibility changes become possible, and the Design System can be shared deliberately. The repository gains more complex CI, ownership, release, access, and history-management requirements. Broad changes can affect several products and therefore require explicit impact analysis.

## Alternatives considered

- **Separate repositories:** preserves strong repository isolation but retains cross-cutting coordination, discovery, and atomic-change friction.
- **Monolithic single application:** rejected because it would collapse product, data, deployment, and failure boundaries.
- **Git submodules:** rejected as the primary model because they retain multi-repository coordination while adding pointer, checkout, and release complexity.
- **Monorepo with autonomous products:** accepted because it improves shared governance and compatibility without requiring runtime consolidation.
