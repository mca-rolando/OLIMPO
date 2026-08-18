# ADR 0002: Products Remain Operationally Autonomous

- Status: Accepted
- Date: 2026-08-17

## Context

A central dependency could turn convenience integrations into a suite-wide outage and create a distributed monolith.

## Decision

> **Products remain operationally autonomous; OLIMPO provides shared control-plane capabilities, not mandatory data-plane dependency.**

HERMES continues DDNS operations, ARGUS continues data collection/evaluation, and METIS continues ITSM when OLIMPO is unavailable. Routine domain operations cannot synchronously require OLIMPO. Products use local queues/outboxes, cached safe policy, idempotency, graceful degradation, and recovery reconciliation. Asynchronous integration is preferred; explicit user-driven synchronous operations are allowed with bounded, visible failure.

## Consequences

Availability faults remain contained and products can upgrade independently. The system accepts eventual consistency and requires duplicate/out-of-order handling, cache expiry, replay, conflict resolution, and more rigorous resilience testing.

## Alternatives considered

- **Require OLIMPO synchronously for routine product operations:** rejected because an OLIMPO outage would become a HERMES, ARGUS, and METIS outage.
- **Disable all shared behavior during disconnection without buffering:** rejected because it would lose integration facts and prevent controlled reconciliation.
