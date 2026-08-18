# OLIMPO Design System v1.0

## Direction and ownership

OLIMPO defines the shared visual language for HERMES, ARGUS, METIS, and future suite applications. The intended character is clean, modern, information-dense where useful, and enterprise-oriented, inspired by the usability and visual clarity of modern UniFi management experiences. OLIMPO must not copy proprietary CSS, assets, or another product pixel-for-pixel. All tokens, components, documentation, and code will be owned and maintained by OLIMPO.

The accessibility target is WCAG 2.2 AA. Primary dashboards are validated at 1920x1080 and across documented responsive sizes. Keyboard operation, visible focus, logical reading order, text resizing, reduced motion, contrast, accessible names, and assistive-technology semantics are acceptance criteria.

## Foundations

Semantic tokens cover color, typography, spacing, size, radius, border, elevation, motion, focus, and responsive breakpoints. Components consume semantic tokens rather than raw product colors. A future machine-readable token format should generate web variables and documentation from one versioned source.

The type scale prioritizes legibility and tabular-number alignment. Spacing follows a consistent base grid. Motion communicates state without blocking work and honors reduced-motion preference. Focus indicators remain visible in every theme.

Semantic status is suite-wide: green means healthy/success, amber warning, red critical/failure, blue informational, and gray unknown/disabled. Every state also includes text, icon, pattern, or shape; color alone never conveys status. Product accents—HERMES blue/cyan, ARGUS azure/blue, and METIS indigo/violet-blue—cannot redefine semantics.

## Components and patterns

The shared inventory includes buttons, links, icons, tooltips, menus, breadcrumbs, tabs, sidebar and top bar; text, selection, date/time, validation, and search inputs; tables, pagination, filters, bulk actions; cards, KPI tiles, alerts, notifications, dialogs, drawers; charts with accessible alternatives; skeletons and progress indicators; and intentional loading, empty, error, offline, stale, unauthorized, and partial-data states.

Tables preserve headers, keyboard access, responsive alternatives, sorting/filter semantics, density controls, and non-color status. Forms pair labels and instructions with fields, place validation near the cause, preserve input when recovery is safe, and summarize errors. Destructive actions state impact and require proportionate confirmation. Charts use redundant encodings and data/table alternatives.

## Governance

Shared components are documented in Storybook or an equivalent showcase with states, accessibility notes, usage guidance, and visual tests. Changes use semantic versioning, accessibility review, cross-theme screenshots, migration notes, and deprecation windows. Products contribute through review but do not fork core patterns casually. Exceptions are documented, measured, and either product-specific or promoted into the system.

Future package naming may use `@mca/olimpo`, with separate tokens, components, icons, and shell packages if justified. Package structure remains an architecture recommendation until implementation review.

See [theme requirements](light-dark-theme-v1.0.md), [application shell](application-shell-v1.0.md), and [kiosk layout](kiosk-layout-v1.0.md).
