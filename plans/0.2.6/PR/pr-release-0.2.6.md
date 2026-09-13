## Summary

Cut v0.2.6: add the GitHub Release body at `plans/0.2.6/PR/release-v0.2.6.md`, date the CHANGELOG 0.2.6 section to the actual release day, fix the one stale compatibility-matrix line the release note contradicts, drop the "in flight for 0.2.6" WASM wording from the site, and rebuild the committed browser artifact at the release commit.

The version stamp (`VERSION`, `internal/cli.Version`, bindings, CHANGELOG facts, docs, samples) already landed on master in the #70 review wave. This PR carries only the release cut artifacts.

---

## Motivation / context

- Release process: `RELEASE.md` cut steps.
- Release note: `plans/0.2.6/PR/release-v0.2.6.md`.
- Predecessor release: `plans/0.2.5/PR/release-v0.2.5.md`.
- Issues: see **Related issues** (none open).

---

## Changes

### Release note

- `plans/0.2.6/PR/release-v0.2.6.md`: GitHub Release body covering the v0.2.5 to v0.2.6 delta (204 commits, 10 merged PRs): print CSS coverage at 354 implemented / 0 partial / 464 unsupported of 818 webref properties, the browser WASM adapter, warm-path performance and memory recovery, layout and table fixes, the 57-63 fixture additions, and honest known limits.

### Release bookkeeping

- `CHANGELOG.md`: the 0.2.6 section is dated 2026-09-13 (was 2026-08-29; the cut slipped).
- `plans/README.md` and `plans/0.2.6/README.md`: the 0.2.6 row now reads complete and links the release note.

### Docs and artifact accuracy

- `documentation/compatibility-matrix.md` group B: `background-clip`, `background-origin`, and `background-size` are Implemented; the three `-webkit-*` aliases stay Unsupported because their remaps are not registered (`plans/0.2.6/catalog/mapping.json`).
- `frontend/index.html`: removed the three "(in flight for 0.2.6)" WASM descriptions and rebuilt `docs/` with `npm --prefix frontend run build` (only `docs/index.html` changed).
- `frontend/public/wasm/gowkhtmltopdf.wasm` and `docs/wasm/gowkhtmltopdf.wasm`: rebuilt by `make wasm-test` at this release tree (`vcs.revision=f631908`, clean tree, `main.wasmVersion=0.2.6`). The previous committed artifact embedded `vcs.modified=true` from PR #71's tree.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. |
| **Memory** | None. |
| **Behavior / correctness** | None in the engine. The committed browser artifact now reports the release revision with a clean-tree build. |
| **API / CLI** | No changes. |
| **Dependencies** | None added or removed. |
| **Binary size / build time** | WASM artifact changes 8 bytes of embedded build metadata (30,029,485 to 30,029,477). |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

Run on the frozen tree before tagging, all exit 0:

- [x] `bash scripts/check_versions.sh` (versions aligned: 0.2.6)
- [x] `make test`
- [x] `make golden`
- [x] `make claim-scan` (clean)
- [x] `make lint` (golangci-lint v1.64.8, size gate, frontend lint)
- [x] `make wasm-test` (contract tests, frontend lint/build/tests, Chrome live-demo smoke for PDF, PNG, JPEG)
- [x] `make build` plus the version-stamp assertion (0.2.6)
- [x] `CGO_ENABLED=0 go build ./...`
- [x] `race` job is covered by CI on the identical code tree (`c904ba7`), and the #70 prep logs three `make test-race` reruns on 2026-09-13 after the strip fix, JPEG restore, and final tree.

### Commands

```sh
bash scripts/check_versions.sh
make test
make golden
make claim-scan
make lint
make wasm-test
make build
CGO_ENABLED=0 go build ./...
```

---

## Screenshots / sample output

```
claim-scan: clean
versions aligned: 0.2.6
ok  github.com/chinmay-sawant/gowkhtmltopdf/internal/convert  7.743s
size-check: clean (3 allowlisted over-limit files)
Live demo browser smoke passed for PDF, PNG, JPEG, validation, and mobile layout.
version stamp: OK (0.2.6)
CGO_ENABLED=0 build: OK
```

---

## Related issues

- Closes: none (no open issues on the repo as of 2026-09-13)
- Relates to #62 (0.2.6 CSS and layout tranche), #70 (review, perf, and stamp wave), #71 and #72 (mitered-border fix and artifact rebuild)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`, `enhancement`)
- [x] Related issues filled
- [x] Filled body committed under `plans/0.2.6/PR/pr-release-0.2.6.md`

---

## Follow-ups (out of scope)

- `plans/0.2.6/perf-improve/wave-2-50pct/phase-wise-checklist.md` stays open (1/43 rows closed).
- `testdata/golden/README.md` still lacks rows for fixture-59 and fixture-63.
- `AGENTS.md` plans partition stops at `plans/0.2.4/`.
- `RELEASE.md` version-source table has stale line references (`api.go:23`, `Makefile:76`, header lines 105/106).
- `documentation/comparison-with-others/landscape-2026.md` still calls v0.2.4 the current release.

---

## Reviewer checklist

- [ ] Release note matches the shipped delta and the machine-readable catalog
- [ ] No unrelated changes in diff
- [ ] No public API or CLI changes
- [ ] PR has assignee and labels
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.html` | 2 | 6 | 6 |
| `.md` | 6 | 327 | 11 |
| `.wasm` | 2 | Binary | Binary |
| **Total** | **10** | **333** | **17** |
