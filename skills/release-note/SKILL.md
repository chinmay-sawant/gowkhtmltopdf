---
name: release-note
description: >
  Cut a gowkhtmltopdf release: bump VERSION, write CHANGELOG and GitHub
  release notes from the previous tag, drop “unreleased” language from
  the generic user docs and site, stamp cli.Version, open the chore
  release PR. Use when the user says “release notes”, “draft release”,
  “cut 0.x.y”, “promote unreleased”, or runs /release-note.
---

# Release notes

Promote the next semver. Do **not** tag or push `v*` until the user
asks. Default integration branch is **`master`**.

Previous tag is `v` + the old `VERSION` (for example `v0.2.1`). New
version is the one they named (for example `0.2.2`).

## 1. Collect the delta

Cover **every** change since the previous tag (commits, merged PRs,
closed issues). Do not invent features that are only in plans.

Write two bodies:

| File | Role |
|------|------|
| `plans/<ver>/PR/release-v<ver>.md` | GitHub Release body (same shape as `plans/0.2.1/PR/release-v0.2.1.md`) |
| `plans/<ver>/PR/pr-release-<ver>.md` | PR body from `skills/PR/PR_TEMPLATE.md` |

Honesty that always stays true unless the engine change says otherwise:

- Default output is **unclaimed PDF 1.4**
- `--pdf-version` / `WithPDFVersion` is a version, **not** a claim
- `--pdf-profile` / `WithPDFProfile` is the claim

## 2. Version files (must change together)

These three must agree. `make test` fails if they do not
(`TestCLIVersionMatchesVERSIONFile` in `api_test.go` and
`internal/cli/cli_test.go`).

| File | What to set |
|------|-------------|
| [`VERSION`](../../VERSION) | Bare semver only (`0.2.2`), no `v` |
| [`CHANGELOG.md`](../../CHANGELOG.md) | Move `## Unreleased` into dated `## <ver> (YYYY-MM-DD)`. Leave an empty `## Unreleased` on top |
| [`internal/cli/help.go`](../../internal/cli/help.go) | Default `var Version = "<ver>"` (ldflags overwrite this on tagged builds) |

Do **not** change `LibraryVersion` in `api.go` — that is the wkhtmltopdf
settings-surface id (`0.12.7-dev`), not the project release.

## 3. Generic docs — drop “unreleased \<ver\>”

Search and rewrite so the new version is **current**, not “on master /
not in the previous tag”. Keep product honesty (version ≠ profile).

**Always touch when the release changes user-visible status:**

| File |
|------|
| [`README.md`](../../README.md) |
| [`documentation/overview.md`](../../documentation/overview.md) |
| [`documentation/getting-started.md`](../../documentation/getting-started.md) (`--version` example + `VERSION (currently …)`) |
| [`documentation/cli.md`](../../documentation/cli.md) |
| [`documentation/library-api.md`](../../documentation/library-api.md) (`go get …@v<ver>`) |
| [`documentation/fidelity.md`](../../documentation/fidelity.md) |
| [`documentation/deferred.md`](../../documentation/deferred.md) (shipped rows say **Shipped in \<ver\>**, not Unreleased) |
| [`documentation/samples.md`](../../documentation/samples.md) |
| [`output/README.md`](../../output/README.md) |

**Site (same wording as the guides):**

| File |
|------|
| `frontend/src/pages/LandingPage.jsx` |
| `frontend/src/data/content/page-overview.json` |
| `frontend/src/data/content/page-about.json` |
| `frontend/src/data/content/page-getting-started.json` |
| `frontend/src/data/content/page-cli.json` |
| `frontend/src/data/content/page-library-api.json` |
| `frontend/src/data/content/page-architecture.json` |
| `frontend/src/data/content/page-compatibility.json` |
| `frontend/src/data/content/page-dossier.json` |
| `frontend/public/data/issues.json` (evidence strings that name the version) |

Then rebuild Pages:

```sh
cd frontend && npm run build
```

That copies into `docs/` (`docs/index.html`, `docs/data/issues.json`,
hashed `docs/assets/*.js`). Commit the rebuilt tree with the source.

**Index the release notes:**

| File |
|------|
| `plans/<ver>/README.md` |
| [`plans/README.md`](../../plans/README.md) |

## 4. Leave custom / one-off files alone

Do **not** rewrite these unless the user names them:

- `documentation/comparison-with-others/*` (landscape, vendor comparisons)
- Historical PR bodies under `plans/**/PR/` from earlier releases
- `CONTRIBUTING.md` process language (`## Unreleased` as the next-release slot)
- Benchmark snapshot dates that record a **past** measured binary
  (`README.md` performance table, `testdata/golden/benchmarks/`)

## 5. Verify

```sh
test "$(tr -d '[:space:]' < VERSION)" = "<ver>"
# must match internal/cli.Version default
make test
make lint
```

Search leftover “unreleased” on the generic set only:

```sh
rg -n -i 'unreleased' README.md documentation/overview.md \
  documentation/getting-started.md documentation/cli.md \
  documentation/library-api.md documentation/fidelity.md \
  documentation/deferred.md documentation/samples.md \
  output/README.md frontend/src frontend/public/data/issues.json
```

`CHANGELOG.md` may keep an empty `## Unreleased` heading.

## 6. Branch, PR, then stop

Branch `chore/release-<ver>` from `master`. Commit. Push. Open the PR:

```sh
gh pr create \
  --base master \
  --head "$(git branch --show-current)" \
  --title "chore(release): promote v<ver> and drop unreleased language" \
  --body-file plans/<ver>/PR/pr-release-<ver>.md \
  --assignee "@me" \
  --label documentation \
  --label enhancement
```

After merge (only if the user asks): tag must match `VERSION` with a `v`
prefix (`git tag v<ver> && git push origin v<ver>`). Paste
`plans/<ver>/PR/release-v<ver>.md` as the GitHub Release body.
See [CONTRIBUTING.md — Cutting a release](../../CONTRIBUTING.md).
