# Changelog

All notable changes to OLIMPO will be documented here. The project follows
[Semantic Versioning](https://semver.org/) once release artifacts exist.

## [Unreleased]

### Added

- Initial repository governance and documentation baseline.
- Architecture specifications for the control plane, integrations, entities,
  events, automation, identity, resilience, observability, and audit.
- Design-system, application-shell, theme, and kiosk guidance.
- Initial architecture decision records.
- ADR 0007 and an MSP-first, multi-tenant architecture baseline with tenant isolation as a security boundary.
- Suite compatibility policy and initial alignment matrix for OLIMPO, HERMES, ARGUS, and METIS.
- ADR 0008 accepting an autonomous-products monorepo and proposed ADR 0009 for the history-preserving import strategy.
- Repository inventory, target layout, history/tag/release preservation, rollback, versioning, boundary, and phased migration plans.

### Changed

- Clarified independent project ownership and replaced customer-specific architectural ownership language with vendor-neutral platform, operator, tenant, organization, and site terminology.
- Made identity, authorization, entities, events, correlation, integrations, automation, observability, audit, and resilience explicitly tenant-aware.
- Clarified the planned transition from the current architecture repository to the OLIMPO Ecosystem monorepo without claiming that product imports have occurred.

[Unreleased]: https://github.com/mca-rolando/OLIMPO/commits/main
