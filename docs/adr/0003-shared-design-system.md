# ADR 0003: OLIMPO Owns the Shared Design System

- Status: Accepted
- Date: 2026-08-17

## Context

Users need a coherent suite while products retain recognizable identity. Copying third-party interface assets would introduce legal, maintenance, and accessibility risk.

## Decision

OLIMPO owns semantic design tokens, shared components, the application shell, Light/Dark/System themes, accessibility patterns, and kiosk layouts. The direction may be inspired by modern enterprise network-management clarity but must not copy proprietary CSS, assets, or pixel layouts. The target is WCAG 2.2 AA and primary validation includes 1920x1080. Product accents are allowed but cannot change suite semantic-status meanings.

## Consequences

Products share navigation and interaction patterns and can reuse tested accessibility behavior. The shared system needs versioning, documentation, visual regression, migration support, and contribution governance; product-specific divergence requires explicit review.

## Alternatives considered

- **Allow each product to maintain an unrelated design system:** rejected because it would create inconsistent navigation, accessibility, semantics, and maintenance.
- **Clone an existing proprietary interface:** rejected because of ownership, legal, accessibility, and long-term maintenance risks.
