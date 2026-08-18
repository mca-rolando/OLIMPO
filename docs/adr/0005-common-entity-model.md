# ADR 0005: Use Canonical Cross-Product Entity IDs

- Status: Accepted
- Date: 2026-08-17

## Context

Products can use different names and identifiers for the same organization, site, location, user, team, device, network, application, service, or integration, preventing dependable correlation.

## Decision

OLIMPO owns immutable canonical entity IDs and mappings to product-native references. Products remain authoritative for their native entity lifecycle and domain attributes. Mappings retain provenance, freshness, history, tombstones, and audited merge/split decisions. Ambiguous automatic matches require authorized review; canonical IDs are never reassigned.

## Consequences

Deep links, search, correlation, and automation gain stable identity without centralizing all product data. Synchronization, stewardship, conflict resolution, privacy scoping, and reconciliation must be designed and operated explicitly.

## Alternatives considered

- **Choose one product's identifier as the suite-wide identifier:** rejected because it would privilege one domain, leak its lifecycle into others, and impede future applications.
- **Match entities only by display name at query time:** rejected because names are mutable, ambiguous, and unsuitable for durable correlation or audit.
