# OLIMPO Documentation

This directory is the documentation source of truth for the OLIMPO control plane. Normative language uses **MUST**, **SHOULD**, and **MAY** in their usual standards sense.

## Architecture

- [Umbrella architecture and functional specification](architecture/architecture-functional-spec-v1.0.md)
- [Control-plane boundaries and capabilities](architecture/control-plane-v1.0.md)
- [Integration, registry, deep links, search, notifications, and maintenance](architecture/integration-model-v1.0.md)
- [Canonical entity identity and ownership](architecture/common-entity-model-v1.0.md)
- [Versioned events, delivery, and correlation](architecture/event-model-v1.0.md)
- [Automation execution and safeguards](architecture/automation-model-v1.0.md)
- [Identity, access, service identity, secrets, and application security](architecture/identity-security-v1.0.md)
- [Telemetry, health, and immutable-oriented audit](architecture/observability-audit-v1.0.md)
- [Autonomy, failure modes, and recovery](architecture/autonomy-resilience-v1.0.md)

## Experience design

- [Design system](design/design-system-v1.0.md)
- [Light, Dark, and System themes](design/light-dark-theme-v1.0.md)
- [Application shell](design/application-shell-v1.0.md)
- [Kiosk and NOC layout](design/kiosk-layout-v1.0.md)

## Architecture decisions

ADRs [0001](adr/0001-olimpo-is-a-control-plane.md) through [0006](adr/0006-external-secrets-management.md) establish the initial control-plane, autonomy, design-system, event, entity, and secrets decisions. Accepted ADRs are immutable except for status and links to later superseding decisions.
