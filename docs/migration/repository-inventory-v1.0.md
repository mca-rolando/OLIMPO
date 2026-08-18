# Source Repository Inventory v1.0

Inspection date: 2026-08-17  
Inspection mode: local and read-only; no credential files were read

## Summary

| Product | Path | Current/default branch | HEAD | Version | Tracked size | Tags |
|---|---|---|---|---|---:|---:|
| OLIMPO architecture baseline | `/home/rolando/projects/olimpo/git` | `architecture/olimpo-monorepo`; remote default not locally configured, `main` tracks `origin/main` | `c272aac24bc2ce16e664f6b6d082bc7fa9e55d64` | `0.1.0-dev` | 30 files / 100,607 bytes | 0 |
| HERMES | `/home/rolando/projects/hermes/git/HermesDDNS` | `main`; `origin/main` | `bcf0d8db8b340718277dbba3b44a585493f26aa5` | `26.08-02` | 126 files / 14,738,475 bytes | 12 |
| ARGUS | `/home/rolando/projects/argus/git` | `main`; `origin/main` | `400c5f174c835046304f3e1cb6f75331c9ab9f44` | `26.08-01-dev` | 75 files / 6,821,674 bytes | 0 |
| METIS | `/home/rolando/projects/metis/git` | `main`; `origin/main` | `e9ead7d839c597c21ec77e3d22ead19872e258dc` | No `VERSION` file | 27 files / 732,128 bytes | 0 |

Tracked size is the sum of HEAD blob sizes and excludes `.git`, ignored files, and working-tree-only files. Git object-store measurements were approximately 252 KiB loose for OLIMPO, 13.06 MiB packed for HERMES, 5.59 MiB packed plus 84 KiB loose for ARGUS, and 1.35 MiB loose for METIS.

## OLIMPO

- Remote: `origin = git@github.com:mca-rolando/OLIMPO.git`.
- Local branches: `architecture/olimpo-monorepo` at `c272aac…` (active planning branch); `main` at `c272aac…`; `docs/architecture-decisions-v1` at `c931f2b…` (historical baseline).
- Remote branches: `origin/main` at `c272aac…`; `origin/docs/msp-multitenancy-baseline` at `8c95da0…` (merged topic history).
- Recent history: `c272aac` merged the MSP baseline, `8c95da0` adopted MSP-first multi-tenancy, and `c931f2b` established the architecture baseline.
- Structure/tooling: documentation-only root with `AGENTS.md`, `README.md`, `CHANGELOG.md`, `VERSION`, architecture/design/governance/ADR documentation, and a workflow placeholder. No application, Docker, deployment, or active CI implementation exists.
- Product architecture: shared control plane, tenant-aware identity/entities/events/integrations/automation/audit, product autonomy, and the OLIMPO Design System.

## HERMES

- Remote: `origin = git@github.com:mca-rolando/HermesDDNS.git`.
- Local branch: `main` at `bcf0d8d…`.
- Important remote branches: `origin/main` and `origin/release/26.08-02` at `bcf0d8d…`; `origin/release/26.08-01` at `076ddfa…`; historical `origin/master`, `origin/v2-0-0`, and `origin/v2-0-1/2`; multiple Dependabot branches.
- Tags: lightweight `0.1`, `0.2`, `0.3`, `1.0.0`, `1.1.0` through `1.1.4`; annotated `26.08-01`, `26.08-02`, and `pre-hermes-26.08-01`.
- Latest commits are the `26.08-02` release series, culminating in `bcf0d8d` (`HermesDDNS 26.08-02: finalize release metadata`).
- Language/tooling: Go module; 54 tracked Go files; server and CLI commands; unit tests; shell build/smoke/E2E tooling; Dockerfile, Compose, BIND setup, and GitHub Actions CI running Go test/vet/build plus E2E.
- Structure: `cmd/`, `internal/`, `deployment/`, `docs/`, `legacy/`, `scripts/`, and `tests/e2e/`. Legacy upstream source and attribution are intentionally retained.
- Product architecture: self-hosted DDNS, BIND `nsupdate`, SQLite persistence, device identities, hashed credentials, enrollment, current telemetry/network context, and credential rotation. HERMES remains authoritative for DDNS state.
- Potential overlap: agent identity, telemetry envelopes, API contracts, audit conventions, and eventual Design System usage may align with shared contracts; DDNS models, credential lifecycle, DNS update logic, BIND integration, SQLite schema, and agent domain behavior remain HERMES-owned.

## ARGUS

- Remote: `origin = git@github.com:mca-rolando/Argus.git`.
- Branches: local and remote `main` only at `400c5f1…`; no tags.
- Recent commits establish the scaffold and strengthen repository/UniFi API governance.
- Version: `26.08-01-dev`; changelog explicitly states no stable production release.
- Language/tooling direction: planned Python/FastAPI API and worker, React/TypeScript web UI, Python appliance-safe agent, PostgreSQL, optional Redis, NGINX, Docker Compose, and systemd resources. Most runtime paths currently contain `.gitkeep` only.
- Structure: `apps/api`, `apps/web`, `apps/worker`, `agent`, `packages/python-common`, `deployment`, `tests`, scripts, security policy, 18 diagrams, 22 storyboards, and DOCX/PDF specification assets. GitHub workflows contain only `.gitkeep`.
- Product architecture: read-only, outbound-only UniFi observability with appliance-safe collection, site-centered health, strict Network/Protect interface governance, NOC/kiosk UI, and independent monitoring authority.
- Potential overlap: Design System/AppShell, common event envelope, entity references, auth protocol helpers, SDK clients, and observability conventions may be shared after contract review. UniFi acquisition, health evaluation, incident correlation, spool, resource-safety policy, and monitoring data remain ARGUS-owned.

## METIS

- Remote: `origin = git@github.com:mca-rolando/metis.git`.
- Branches: `main` at `e9ead7d…`; local backup branches `backup/local-pre-baseline-20260817-003546` and `backup/remote-pre-baseline-20260817-003546` at `df59279…`; matching remote backup branch.
- No tags and no `VERSION` file. The changelog contains an Unreleased architecture baseline.
- Recent commits include the current architecture baseline, a deliberate repository-content clearing commit, and preserved pre-baseline history on backup refs.
- Language/tooling: documentation-first; Markdown specification/invariants, PNG architecture/storyboard assets, and two review/check shell scripts. No CI, Docker, deployment, or production implementation exists.
- Product architecture: API-first ITSM/ESM, WorkItems, workspaces, service catalog, incidents, requests, SLA/OLA, automation, integrations, audit, and a planned modular-monolith deployment. METIS remains authoritative for ITSM lifecycle.
- Potential overlap: Design System/AppShell, event/entity contracts, identity protocol types, SDK clients, and observability conventions may align. WorkItem, workspace, SLA/OLA, catalog, approval, assignment, and ITSM automation logic remain METIS-owned.

## Branch classification

| Repository/ref | Classification | Recommendation |
|---|---|---|
| OLIMPO `architecture/olimpo-monorepo` | Active | Continue as planning branch; migration execution should use a separate reviewed migration branch/clone. |
| OLIMPO `main` | Active default | Import base; do not rewrite remotely during planning. |
| OLIMPO merged documentation refs | Historical/topic | Preserve in source/bundle; no permanent monorepo branch required unless review policy requires it. |
| HERMES `main` | Active | Import rewritten history. |
| HERMES `release/26.08-02` | Release | Import as a namespaced historical release ref or document that it equals the release tag/main. |
| HERMES `release/26.08-01` | Historical release | Preserve through tag and source bundle; import branch only if release maintenance remains possible. |
| HERMES `master`, `v2-*` | Historical | Retain in source/bundle; import namespaced refs only after human review establishes continuing value. |
| HERMES Dependabot refs | Automated/likely obsolete | Do not create permanent monorepo branches; retain in source repository and pre-migration bundle. |
| ARGUS `main` | Active | Import rewritten history. |
| METIS `main` | Active | Import rewritten history. |
| METIS backup refs | Backup/historical | Preserve in source/bundle; do not make permanent monorepo branches unless needed to recover the cleared baseline. |

This classification is a planning recommendation, not permission to delete or archive any ref.
