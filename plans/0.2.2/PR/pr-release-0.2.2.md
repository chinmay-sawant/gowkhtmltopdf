## Summary

Promote **v0.2.2** from unreleased `master` work to the current release: bump `VERSION`, move changelog notes into a dated 0.2.2 section, drop “unreleased 0.2.2” language from user docs / the documentation site / issue-dossier evidence, and commit the GitHub release body that will be attached to tag `v0.2.2`.

No converter, CLI flag, or library behavior change in this PR — the engine already shipped in #45–#47.

---

## Motivation / context

- Plans: [`plans/0.2.2/README.md`](plans/0.2.2/README.md), [`plans/0.2.2/PR/release-v0.2.2.md`](plans/0.2.2/PR/release-v0.2.2.md)
- CONTRIBUTING requires `VERSION` + a dated `CHANGELOG.md` section before tagging `v0.2.2`
- After #45–#49, guides and the site still said “unreleased 0.2.2 on master / not in 0.2.1”
- Issues: see **Related issues**

---

## Changes

### Release metadata

- `VERSION` `0.2.1` → `0.2.2`
- `CHANGELOG.md`: move Unreleased 0.2.2 notes into **0.2.2 (2026-08-15)**; leave an empty `## Unreleased` for later work
- Add [`plans/0.2.2/PR/release-v0.2.2.md`](plans/0.2.2/PR/release-v0.2.2.md) (GitHub release body covering everything since tag `v0.2.1`)

### Docs honesty

- README, overview, CLI, library API, getting started, fidelity, deferred, samples, landscape-2026, and the SebastiaanKlippert comparison treat 0.2.2 as current
- Opt-in `--pdf-version` / `--pdf-profile` (and `WithPDFVersion` / `WithPDFProfile`) are shipped, not “unreleased”
- Default remains **unclaimed PDF 1.4**; a version flag is still not a conformance claim

### Documentation site

- Frontend content JSON + landing page + dossier copy
- Dossier `issues.json` evidence strings no longer say “unreleased 0.2.2 on master”
- Rebuilt `docs/` Pages assets

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | None in the converter. Docs, `VERSION` stamp, and site copy only |
| **API / CLI** | Docs only. Flags and builders already shipped in #45–#47 |
| **Dependencies** | None |
| **Binary size / build time** | Unchanged. Tag `v0.2.2` will stamp `internal/cli.Version` via the release workflow |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None in this PR | Engine contracts unchanged. Callers that read `Get("pdfprofile")` still get canonical tokens (already shipped in #47) |

---

## Test plan

- [x] `VERSION` is `0.2.2`
- [x] `CHANGELOG.md` has dated **0.2.2 (2026-08-15)** and an empty `## Unreleased`
- [x] User-facing `documentation/` and README no longer say “unreleased 0.2.2”
- [x] Frontend content + `frontend/public/data/issues.json` no longer say “unreleased 0.2.2”
- [x] `cd frontend && npm run build` copied a clean `docs/` tree
- [ ] `make test` (no Go behavior change)
- [ ] `make lint`
- [ ] After merge: tag `v0.2.2` and paste [`plans/0.2.2/PR/release-v0.2.2.md`](plans/0.2.2/PR/release-v0.2.2.md) as the GitHub release body

### Commands

```sh
test "$(tr -d '[:space:]' < VERSION)" = "0.2.2"
rg -n 'unreleased 0\.2\.2' README.md documentation frontend/src frontend/public/data docs/data || true
cd frontend && npm run build
```

---

## Screenshots / sample output

```
VERSION=0.2.2
CHANGELOG: ## 0.2.2 (2026-08-15)
frontend build: copied build output → docs/
```

---

## Related issues

- Relates to #29 (newer PDF versions and compliance epic)
- Relates to #31 (PDF 1.7 — shipped in #45)
- Relates to #32 (PDF 2.0 — shipped in #46)
- Relates to #33 (PDF/A-4 + PDF/UA-2 — shipped in #46)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/0.2.2/PR/pr-release-0.2.2.md`

---

## Follow-ups (out of scope)

- Tag `v0.2.2` and publish the GitHub Release (binaries via `.github/workflows/release.yml`)
- Close #29 / #31 / #32 / #33 from the tag if they are still open after merge

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in the diff
- [ ] Public API / CLI docs match shipped 0.2.2 (not “unreleased”)
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
