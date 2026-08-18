# History, Tag, and Release Preservation v1.0

## Preservation objective

Preserve, as far as technically practical, commit authors, author/committer timestamps, messages, parent topology, file contents, attribution, required branches, tags, and release provenance. Authoritative source repositories remain available and unchanged throughout migration and validation.

## Recommended history preparation

Use fresh mirror clones or clones restored from verified bundles under `/home/rolando/projects/olimpo/migration-work/`. Run the reviewed `git filter-repo` script only against disposable preparation clones. Prefix HERMES, ARGUS, and METIS histories with `products/hermes/`, `products/argus/`, and `products/metis/`, then import the prepared refs into a fresh OLIMPO migration clone.

Path rewriting changes commit IDs because every affected tree changes. It can preserve author/committer identity, timestamps, messages, parent topology, and content, but it cannot preserve original commit SHAs. Do not describe rewritten commits as SHA-preserved.

## Provenance records

The approved migration produces:

```text
migration/provenance/hermes-sha-map.csv
migration/provenance/argus-sha-map.csv
migration/provenance/metis-sha-map.csv
migration/provenance/olimpo-sha-map.csv        # if OLIMPO paths are rewritten
migration/provenance/tag-map.csv
migration/provenance/source-refs.json
migration/provenance/migration-manifest.md
```

Each SHA CSV uses at least:

```text
source_repository,source_commit,imported_commit,path_prefix,source_refs,migration_run
```

Generate maps from `git filter-repo` commit-map output and independently verify sampled roots, merges, release commits, and tips. The human-readable manifest records tool version, commands, starting refs, object counts, warnings, validation results, and reviewers.

## HERMES 26.08-02

Local verification found:

- Source tag ref: `refs/tags/26.08-02`.
- Object type: annotated tag.
- Tag object: `5b531a5c2f66f80b60a18c102046a02cde29271b`.
- Peeled commit: `bcf0d8db8b340718277dbba3b44a585493f26aa5`.
- Tagger date: 2026-08-13 16:43:14 -0400.
- Tag message: `HermesDDNS 26.08-02`.
- The peeled commit exactly equals local `main`, `origin/main`, and `origin/release/26.08-02` (ahead/behind `0/0`).
- It is 11 commits ahead of `origin/release/26.08-01` and 15 commits ahead of `origin/master`; those older refs are ancestors of the newer history, while the release commit is not their ancestor.

During migration, map source tag `26.08-02` to `hermes-v26.08-02`. Record both original tag object and commit plus rewritten tag object and peeled commit. Verify the imported tree under `products/hermes/` equals the source release tree after applying only the prefix transformation. Verify the imported commit is reachable from imported HERMES history and the tag message/tagger metadata are preserved where tooling permits.

The original HERMES repository and its GitHub Release remain historical authorities and are not deleted. Migration documentation links the original release. Recreating a historical release in OLIMPO is optional and requires later approval; reference-only is safer by default because it avoids confusing original publication provenance.

## Tag collision plan

Import tags using `<product>-v<source-version>`. Maintain `tag-map.csv` with source repository, source ref, source tag object type/ID, peeled source commit, target tag, target object type/ID, peeled imported commit, and notes. Preserve lightweight versus annotated form. Resolve any target-name collision before import; never overwrite a target tag.

## Branch plan

Import active mainline histories. Create permanent namespaced branches only for active maintenance or legally/operationally important release lines, for example `history/hermes/release-26.08-02`, after review. Do not import automated dependency branches or backup branches as permanent heads by default; preserve them in source repositories, bundles, and `source-refs.json`. No source ref is deleted.

## GitHub Release policy

No GitHub API or release operation is part of planning or local import. Before archival, capture original release URLs, tags, notes, artifact names, sizes, and checksums using an approved read-only process. Historical source releases remain intact. Future monorepo releases follow the product-prefixed policy in [versioning and release policy](../governance/versioning-release-policy-v1.0.md).
