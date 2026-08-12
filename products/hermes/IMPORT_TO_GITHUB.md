# Import HermesDDNS 26.08-01 into the existing private GitHub repository

Target repository:

```text
https://github.com/mca-rolando/HermesDDNS.git
```

This procedure intentionally preserves the `.git` directory and therefore keeps the history already present in the private repository. Do **not** initialize a new Git repository inside the release folder and do not force-push.

## Recommended safe workflow

Assume the existing private clone is `/path/to/HermesDDNS` and this release was extracted to `/tmp/HermesDDNS-26.08-01`.

First inspect the existing clone:

```bash
cd /path/to/HermesDDNS
git status
git remote -v
git branch --show-current
```

The working tree should be clean before continuing. Confirm `origin` points to the private MCA repository. If necessary:

```bash
git remote set-url origin https://github.com/mca-rolando/HermesDDNS.git
git remote -v
```

Create a recovery tag for the exact pre-Hermes state and push it:

```bash
git tag -a upstream-baseline-20260812 -m "Pre-HermesDDNS refactor baseline"
git push origin upstream-baseline-20260812
```

Create a release branch:

```bash
git switch -c release/hermes-26.08-01
```

Replace the working tree with the Hermes release **without touching `.git`**:

```bash
rsync -a --delete \
  --exclude='.git/' \
  /tmp/HermesDDNS-26.08-01/ \
  /path/to/HermesDDNS/
```

Review everything Git sees, including deletions from the old upstream tree:

```bash
cd /path/to/HermesDDNS
git status
git diff --stat
git add -A
git diff --cached --stat
git diff --cached
```

Commit the first official Hermes release:

```bash
git commit -m "HermesDDNS 26.08-01: foundation and DDNS core"
```

Tag it and push the branch and tag:

```bash
git tag -a 26.08-01 -m "HermesDDNS 26.08-01"
git push -u origin release/hermes-26.08-01
git push origin 26.08-01
```

After review, merge it into the repository's normal branch. If the original branch is `main`:

```bash
git switch main
git merge --ff-only release/hermes-26.08-01
git push origin main
```

If the normal branch has another name, replace `main` with the value returned by `git branch --show-current` before creating the release branch.

## Keep upstream history available

The active module is now HermesDDNS. If you still want to track the original open-source project for occasional comparison, keep it as an `upstream` remote rather than as `origin`:

```bash
git remote add upstream https://github.com/benjaminbear/docker-ddns-server.git
# If upstream already exists, do not add it again.
git remote -v
```

Future upstream changes can be inspected with `git fetch upstream`; they should not be blindly merged because HermesDDNS has intentionally diverged in authentication, data model, repository layout, and deployment architecture.

## Important

Never commit `.env`, API keys, htpasswd secrets, TSIG keys, SQLite runtime databases, backups, or production configuration. The supplied `.gitignore` excludes the primary runtime/secret paths.
