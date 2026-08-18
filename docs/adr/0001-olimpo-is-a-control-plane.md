# ADR 0001: OLIMPO Is a Control Plane

- Status: Accepted
- Date: 2026-08-17

## Context

The OLIMPO ecosystem needs shared identity, integration, entity correlation, automation, governance, audit, and experience without moving product domain responsibilities into a central system or making any customer the platform owner.

## Decision

OLIMPO is the suite control plane. It coordinates and enriches HERMES, ARGUS, METIS, and future products through versioned capabilities and contracts. It does not implement DDNS, monitoring evaluation, or ticket lifecycle domain logic, and it is not the mandatory transport for routine product work.

## Consequences

OLIMPO can evolve shared capabilities consistently and new products can join through capability discovery. Product authority and local operation remain clear. Some views are eventually consistent, adapters and reconciliation are required, and cross-product transactions must expose partial failure rather than assume one database transaction.

## Alternatives considered

- **Centralize all product domain logic in OLIMPO:** rejected because it would erase product ownership boundaries and create a suite-wide failure domain.
- **Integrate products only through independent point-to-point links:** rejected because contracts, identity, audit, and user experience would fragment as the ecosystem grows.
