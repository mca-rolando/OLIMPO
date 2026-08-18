# ADR 0006: Store Secrets in an External Secrets Provider

- Status: Accepted
- Date: 2026-08-17

## Context

Integrations need credentials, but storing plaintext secrets in OLIMPO's normal application database or repository creates unacceptable exposure and rotation risk.

## Decision

Secret values reside in an approved external provider such as HashiCorp Vault, AWS Secrets Manager, or an equivalent enterprise service. OLIMPO stores opaque references and non-secret metadata for ownership, scope, rotation, expiration, health, and revocation. Authorized workloads retrieve values using scoped workload identity; values are never returned to browsers or placed in logs, traces, events, source control, or reports. The provider selection requires a later ADR.

## Consequences

Secret access and rotation are isolated and auditable, and OLIMPO is not a password vault. Runtime operation gains a provider dependency that requires caching/lease, outage, bootstrap, recovery, and break-glass design without persisting plaintext fallback values.

## Alternatives considered

- **Store encrypted secret values in the standard OLIMPO database:** rejected because it would make OLIMPO responsible for vault-grade key custody, access, rotation, and recovery.
- **Commit environment-specific credentials or distribute shared static keys:** rejected because this prevents safe rotation, expands exposure, and violates least privilege.
