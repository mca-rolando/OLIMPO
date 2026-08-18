# ADR 0004: Use a Versioned Common Event Model

- Status: Accepted
- Date: 2026-08-17

## Context

Cross-product correlation and automation require stable meaning across independently released producers and consumers. Future transport may provide at-least-once delivery.

## Decision

OLIMPO defines a transport-independent, versioned event envelope containing unique event ID, source, namespaced type, timestamp, severity, correlation and causation identifiers, entity references, and versioned payload. Consumers are idempotent and tolerate duplicates and reordering. Schema changes favor additive compatibility; breaking changes use new major versions and deprecation overlap. Retry, dead-letter, replay, and audit behavior are part of the contract.

## Consequences

Events can be traced, correlated, replayed, and tested independently of a chosen backbone. Producers and consumers carry validation and compatibility responsibilities, and schema governance becomes a required release practice.

## Alternatives considered

- **Expose transport-native messages directly to every product:** rejected because it would hard-couple domain code to a backbone and fragment envelope semantics.
- **Assume exactly-once, globally ordered delivery:** rejected because that guarantee is impractical across all failure and replay scenarios and would hide required consumer safeguards.
