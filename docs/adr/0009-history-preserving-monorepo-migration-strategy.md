# ADR 0009 — History-Preserving Monorepo Migration Strategy

- Status: Proposed
- Date: 2026-08-17

## Context

HERMES, ARGUS, and METIS have independent Git histories, branches, tags, release evidence, and attribution. Their files must enter product subdirectories without modifying authoritative source repositories. HERMES release `26.08-02` is a protected historical asset.

## Proposed decision

After human approval, use fresh mirror clones or bundles in an isolated migration workspace. For each product, use `git filter-repo --to-subdirectory-filter products/<product>` in a disposable clone, preserve original refs in the source bundle, generate old-to-new commit maps, rename imported tags into product namespaces, and fetch/merge the rewritten history into a fresh OLIMPO migration clone with unrelated histories allowed. Validate HERMES first before repeating the procedure.

The current OLIMPO content should be classified before relocation: ecosystem architecture, ADRs, governance, and migration documentation remain under root `docs/`; OLIMPO Control Plane-specific implementation, version, changelog, and instructions move under `platform/olimpo/`. This relocation also rewrites affected OLIMPO commits if historical path rewriting is selected and therefore requires its own provenance record.

This proposal is not authorization to execute the migration.

## SHA behavior and provenance

Adding a directory prefix changes every rewritten commit tree, so imported product commit SHAs will change. Annotated tag object IDs also change when their target changes. Authors, timestamps, messages, parent topology, file contents, and attribution are preserved as far as the tool permits. Machine-readable CSV maps and a human-readable manifest must record original repository URL, original commit, imported commit, path prefix, source ref, and migration run identifier.

## Alternatives considered

- **`git filter-repo` path rewriting:** recommended. It provides explicit path and tag rewriting with commit maps, but changes SHAs and requires careful validation.
- **`git subtree add`:** can import history under a prefix and is simpler operationally, but provenance mapping and tag/branch preservation are less explicit and later subtree semantics are unnecessary for a one-time consolidation.
- **Direct unrelated-history merges:** preserve original commit SHAs until a path move commit, but historical files occupy repository root and collide; moving only at the tip weakens path-history ergonomics and complicates four roots.
- **Archive or copy:** preserves current files only, not useful commit history, authorship, branch relationships, or release provenance; rejected.

## Risks and controls

Risks include incomplete refs, tag collisions, accidental source mutation, lost release evidence, oversized imports, and incorrect SHA claims. Controls are fresh mirrors, portable bundles, immutable inventories, no source pushes, isolated migration clones, deterministic scripts, provenance maps, tag verification, commit-count/tree comparisons, build tests, and a reviewable migration branch.

## Open decisions before acceptance

- Whether to rewrite the existing OLIMPO Control Plane paths or begin its subdirectory only from the migration commit.
- Which historical and release branches receive namespaced refs in the monorepo.
- Whether historical GitHub releases are referenced only or recreated as clearly marked mirrors.
- Exact `git filter-repo` version and reviewed command script.
- Validation-period duration and archive criteria.
