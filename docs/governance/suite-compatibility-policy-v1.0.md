# Suite Compatibility Policy v1.0

## Governing rule

> **Any cross-cutting architectural decision adopted by OLIMPO must be evaluated for compatibility impact across HERMES, ARGUS, METIS, and future ecosystem products.**

This policy applies even when only OLIMPO documentation or implementation changes. It does not transfer product authority to OLIMPO: each product remains independently owned, released, and operationally autonomous.

## Cross-cutting areas

Reviews cover tenancy, identity, RBAC, security, common entities, events, integrations, automation, audit, secrets, the OLIMPO Design System, deep linking, and compatibility/versioning. A proposal must state contract impact, adoption requirements, compatibility window, migration and rollback guidance, test impact, security implications, and whether product autonomy changes.

## Required impact declaration

Every proposal or ADR for a cross-cutting change includes:

```text
Affected products:

[ ] OLIMPO
[ ] HERMES
[ ] ARGUS
[ ] METIS
[ ] Future ecosystem products
```

For each checked product, record one of: **Defined in OLIMPO**, **Requires adoption in product**, **Not applicable**, or **Future**, with evidence. Do not claim adoption without reviewing that product. Breaking changes require a compatibility matrix, consumer-driven contract tests, a deprecation window, migration guidance, and independently deployable upgrade sequencing.

## Tenancy-specific review

Changes must prove that tenant context is trustworthy, server-side authorization is enforced, mappings and integration credentials cannot collide across tenants, and events, correlation, automation, audit, and background processing cannot leak or act across tenant boundaries. Explicit platform-level exceptions must be minimized, strongly authorized, audited, and incapable of exposing one tenant's customer data to another.
