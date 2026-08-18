# Monorepo Migration Backup and Rollback Plan v1.0

## Before execution

Create `/home/rolando/projects/olimpo/migration-work/` with access limited to the migration operator. Record OLIMPO and source HEADs, all local/remote refs, tags including object type and peeled target, remote URLs, working-tree status, object counts, and tool versions. Do not read or copy credential files.

Create portable full-ref bundles from fresh local mirrors, conceptually:

```text
git bundle create <backup-path>/<repository>.bundle --all
git bundle verify <backup-path>/<repository>.bundle
```

Record bundle checksums outside the repositories and test restoration into disposable directories. A bundle is a safety snapshot, not a substitute for retaining the original GitHub repositories.

## Migration rollback

The migration occurs in a disposable OLIMPO clone on an unpushed migration branch. Before any push, rollback means deleting or quarantining only that disposable workspace and starting from verified bundles/fresh clones. Authoritative source working repositories require no rollback because they are never modified.

After an approved migration branch is pushed but before merge, close or abandon the branch/PR without force-pushing and create a corrected branch from the unchanged target base. After merge, use a reviewed forward revert or corrective migration; never rewrite protected shared history. Product deployments remain unchanged until separately approved, so source consolidation rollback is independent of runtime rollback.

## Recovery concepts

- Restore a source mirror with `git clone <bundle> <recovery-path>` and verify all expected refs and tag targets.
- Restore the OLIMPO pre-migration state from its bundle or recorded starting commit into a new recovery branch.
- Compare `source-refs.json`, tag maps, commit maps, and tree manifests before resuming.
- Keep old repositories active through the validation period; they remain the release/source fallback until explicit cutover acceptance.

## Stop conditions

Stop before push if any required ref, HERMES release tag, author/timestamp/message, release tree, provenance mapping, version/changelog, build, or independent-boundary validation fails. Stop if a source repository status or HEAD changes unexpectedly. Stop if the migration requires credentials, source mutation, force-push, tag overwrite, or an unapproved GitHub operation.
