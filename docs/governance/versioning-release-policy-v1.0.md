# Product Versioning and Release Policy v1.0

## Independent versions

The monorepo has no mandatory global suite version. Each product owns its version scheme, changelog, release readiness, artifacts, compatibility statement, and rollback instructions.

| Product | Current observed baseline | Direction |
|---|---|---|
| OLIMPO Control Plane | `0.1.0-dev` | Semantic versioning direction remains acceptable pending product release policy. |
| HERMES | `26.08-02` | Preserve the existing `YY.MM-RR` calendar sequence. |
| ARGUS | `26.08-01-dev` | Preserve its documented independent `YY.MM-RR` scheme. |
| METIS | No `VERSION` file; Unreleased architecture baseline | Decide its independent scheme before its first release. |

Shared packages may have their own semantic versions when published or may use reviewed workspace revisions while private. A shared-package change does not force an unrelated product release.

## Tag names

Future monorepo product release tags use lowercase product prefixes:

```text
olimpo-v0.1.0
hermes-v26.08-02
argus-v26.09-01
metis-v26.10-01
design-system-v1.0.0
contracts-v1.0.0
```

The `olimpo-` prefix means OLIMPO Control Plane, not the entire ecosystem. Repository-governance milestones should use GitHub milestones or clearly separate tags only if a future policy defines them. Existing source tags are never renamed in source repositories; imported equivalents are created only during the approved migration with a tag map.

Annotated tags are preferred for releases because they carry a tagger, date, message, and optional signature. Imported annotated tags must be recreated or rewritten as annotated tags pointing at the imported commit and record their original tag object and peeled commit. Lightweight historical tags remain lightweight unless a migration manifest explicitly records an intentional conversion.

## GitHub Releases

Every future GitHub Release title and tag identifies its product. Release notes include product path, version, supported compatibility, artifacts, migration/rollback notes, and affected shared-package versions. Release workflows are product-scoped and cannot publish artifacts for another product implicitly.

Historical releases in source repositories remain authoritative and must not be deleted or rewritten. The default recommendation is to reference them from `docs/migration/` and the corresponding imported tag rather than recreate them immediately. If human review later requires a mirrored OLIMPO GitHub Release, mark it as a historical mirror, link the original repository/release, reproduce checksums and notes faithfully, and never imply the monorepo hosted the original publication.

## Compatibility

Products declare compatible API, event, entity, schema, Design System, and shared-package versions. Breaking changes require a deprecation period, migration guidance, consumer tests, and independently deployable sequencing. Lockstep suite releases are prohibited unless a particular cross-product migration is explicitly approved and still provides safe failure and rollback boundaries.
