# Target Monorepo Layout v1.0

## Recommended hierarchy

```text
OLIMPO/
├── AGENTS.md
├── README.md
├── CHANGELOG.md
├── docs/
│   ├── adr/
│   ├── architecture/
│   ├── design/
│   ├── governance/
│   └── migration/
├── platform/
│   └── olimpo/
│       ├── AGENTS.md
│       ├── README.md
│       ├── VERSION
│       ├── CHANGELOG.md
│       ├── docs/
│       └── future implementation/
├── products/
│   ├── hermes/
│   │   ├── AGENTS.md
│   │   ├── README.md
│   │   ├── VERSION
│   │   ├── CHANGELOG.md
│   │   └── imported HERMES tree
│   ├── argus/
│   │   ├── AGENTS.md
│   │   ├── README.md
│   │   ├── VERSION
│   │   ├── CHANGELOG.md
│   │   └── imported ARGUS tree
│   └── metis/
│       ├── AGENTS.md
│       ├── README.md
│       ├── CHANGELOG.md
│       └── imported METIS tree
├── packages/
│   ├── design-system/
│   ├── contracts/
│   ├── events/
│   ├── entities/
│   ├── auth/
│   ├── sdk/
│   └── observability/
├── schemas/
│   ├── api/
│   ├── entities/
│   └── events/
├── deployment/
│   ├── olimpo/
│   ├── hermes/
│   ├── argus/
│   └── metis/
├── migration/
│   └── provenance/
└── .github/
    ├── CODEOWNERS
    └── workflows/
```

The `future implementation/` label is explanatory only; an actual placeholder directory is unnecessary. Product code should use its native coherent structure beneath its product root rather than being forced into a uniform internal layout.

## Refinements from the initial concept

- Root `docs/` owns ecosystem architecture, cross-product ADRs, governance, Design System specification, and migration records. Product-specific architecture remains with its product.
- Root has no global `VERSION`; the current `VERSION` should eventually become `platform/olimpo/VERSION` because products version independently. Root `CHANGELOG.md` records ecosystem/repository governance, while each product keeps its own changelog.
- Product deployment sources remain inside each product when tightly coupled to its build. Root `deployment/<product>/` should contain only ecosystem-level orchestration or operator overlays and must not become a mandatory giant deployment.
- `migration/provenance/` is reserved for generated SHA maps, tag maps, source-ref inventories, tool versions, and signed-off validation manifests from the approved migration run.
- Shared directories begin only when a reviewed consumer and ownership model exists. Empty speculative packages should not be created during migration.

## Shared package boundaries

| Package | Appropriate contents | Excluded contents |
|---|---|---|
| `design-system` | Semantic tokens, accessible components, AppShell, Light/Dark/System themes, top-right theme selector, navigation patterns, kiosk/NOC primitives, WCAG 2.2 AA and 1920x1080 validation assets | Product workflows, domain screens, tenant branding that changes semantic colors |
| `contracts` | Versioned cross-product interface definitions, compatibility metadata, error/idempotency conventions | Internal persistence models and product domain services |
| `events` | Tenant-aware envelope types, schema validation, correlation/causation primitives, safe codecs and test fixtures | Product event production rules, broker deployment, ARGUS health correlation, METIS workflow logic |
| `entities` | Canonical reference types, tenant-qualified mapping contracts, identifiers and provenance formats | HERMES domains/records, ARGUS observations, METIS WorkItems/SLA models |
| `auth` | OIDC protocol types, token-validation building blocks, tenant-context interfaces, security test utilities | Central runtime authorization dependency, tenant policy decisions, credentials or secrets |
| `sdk` | Generated/pinned clients for versioned APIs and test fakes | Direct database access or hidden service-to-service shortcuts |
| `observability` | Attribute naming, redaction interfaces, trace/correlation propagation, baseline instrumentation helpers | One mandatory telemetry backend or cross-tenant query privilege |

Shared packages are build-time dependencies with independent compatibility policy. Products pin versions or workspace revisions and must be able to build from a declared lock state. A shared package cannot require another product or the OLIMPO Control Plane at runtime.

Machine-readable schemas under `schemas/` are the contract source of truth when applicable; language packages may be generated from them. Ownership, versioning, compatibility, and consumer tests must be explicit to avoid two competing sources.

## AGENTS.md hierarchy

- Root `AGENTS.md`: ecosystem invariants, tenant isolation, MSP-first architecture, Git/history safety, repository conventions, cross-product impact review, shared contract and Design System governance, and documentation policy.
- `platform/olimpo/AGENTS.md`: OLIMPO Control Plane authority, control-plane/data-plane boundary, graceful degradation, canonical mapping, event/automation, and platform-role constraints.
- `products/hermes/AGENTS.md`: DDNS authority, DNS safety, agent/credential lifecycle, release compatibility, HERMES tests and deployment.
- `products/argus/AGENTS.md`: read-only and outbound-only UniFi invariants, appliance safety, monitoring authority, data freshness, agent resource bounds, and NOC behavior.
- `products/metis/AGENTS.md`: ITSM authority, WorkItem/workspace/SLA invariants, connector direction, internal-note confidentiality, and METIS tests/deployment.

Instructions apply from root to the file being changed. A nested file may add stricter rules but may not weaken root security, tenant isolation, product autonomy, shared-contract governance, or Git safety. Product rules decide product domain behavior; root rules decide repository and cross-product behavior. Conflicts stop work pending human or ADR resolution.

## Independent operations

The eventual task runner may expose `build hermes`, `test hermes`, and corresponding ARGUS, METIS, and OLIMPO Control Plane commands, but each delegates to product-native tooling. A complete repository build is useful for release qualification and shared-package changes, not mandatory for routine isolated product work.

Deployable units remain separate. Supported combinations may include ARGUS only, ARGUS plus METIS, HERMES plus ARGUS, or all products. Repository layout does not decide shared versus dedicated customer topology and never implies a shared database.
