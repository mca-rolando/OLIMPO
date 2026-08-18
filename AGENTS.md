# Repository Guidance for Agents

## Purpose and governing principle

OLIMPO is the shared control, integration, identity, automation, and user-experience layer for the MCA application ecosystem. HERMES owns DDNS behavior, ARGUS owns monitoring behavior, and METIS owns ITSM behavior.

> **Products remain operationally autonomous; OLIMPO provides shared control-plane capabilities, not mandatory data-plane dependency.**

Architectural boundaries take precedence over implementation convenience. Never make routine product domain operation synchronously dependent on OLIMPO and never create a distributed monolith.

## Repository rules

- Write all repository-facing content exclusively in English.
- Prefer complete files and coherent, reviewable changes over fragments.
- Keep architecture, contracts, and documentation synchronized with behavior.
- Use ADRs for consequential or difficult-to-reverse decisions; update superseded ADRs rather than silently contradicting them.
- Do not perform destructive Git operations without explicit approval. Do not commit, push, publish, deploy, or create external resources automatically.
- During documentation-only work, make no real external calls and do not introduce production code, services, databases, or deployments.
- Never access real credentials. Never write secrets, tokens, private endpoints, or sensitive configuration to the repository, tests, logs, examples, or reports.

## Product and data boundaries

- HERMES remains authoritative for DDNS domain state.
- ARGUS remains authoritative for monitoring observations and health evaluation.
- METIS remains authoritative for ticket and incident lifecycle.
- OLIMPO may own canonical cross-product identifiers, mappings, integration and automation state, cached summaries, and audit records.
- Integrate through stable, versioned contracts. Prefer asynchronous exchange where appropriate; user-driven synchronous operations must fail gracefully.

## API and event governance

- Design APIs OpenAPI-first, version them explicitly, validate inputs, and enforce authorization server-side.
- Version event envelopes and payload schemas. Preserve unique event, correlation, and causation identifiers.
- Assume duplicate and out-of-order delivery. Consumers must be idempotent and tolerant of backward-compatible additions.
- Define deprecation windows, compatibility matrices, migration guidance, and consumer-driven contract tests before breaking changes.

## Security and identity

- Apply least privilege, secure defaults, defense in depth, and zero trust between services where practical.
- Use Microsoft Entra ID through OIDC/OAuth 2.0 as the primary enterprise identity direction, while preserving controlled bootstrap and emergency access.
- Never rely on frontend permissions for enforcement.
- Store only references and metadata for secrets; secret values belong in an approved external secrets manager.
- Prefer short-lived scoped service credentials, rotation, revocation, encryption in transit, and auditable administrative actions.

## UI/UX governance

- Use OLIMPO-owned semantic design tokens and shared components; do not copy proprietary CSS, assets, or layouts.
- Support native Light, Dark, and System themes. Dark Mode is intentionally designed, not inverted. Place quick theme control in the upper-right application area.
- Validate the primary dashboard and kiosk experience at 1920x1080 and test responsive behavior at smaller supported sizes.
- Target WCAG 2.2 AA. Status must never be communicated by color alone.
- Keep the shell, navigation, typography, forms, tables, dialogs, status semantics, and accessibility patterns consistent while allowing restrained product accents.

## Quality and testing

Future implementation changes require proportionate unit, component, API-contract, event-contract, integration, resilience, security, accessibility, visual-regression, and end-to-end tests. Critical cross-product workflows must be testable with local fakes and fixtures, never production dependencies.

Run relevant formatting, link, schema, and test checks before handoff. Document assumptions, failure behavior, operational impact, compatibility impact, and security implications. Update architecture documents and ADRs when boundaries or decisions change.
