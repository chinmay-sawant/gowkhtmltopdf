## Summary

Stamp 0.2.7 and record the release note for everything that landed on master after v0.2.6. The engine work is already merged. This PR dates the changelog, bumps the version files together, and drops "current release is 0.2.6" plus "unreleased next-72" from the guides and the site.

---

## Motivation / context

- Release process: `RELEASE.md` cut steps and `skills/release-note/SKILL.md`.
- Release note: `plans/0.2.7/PR/release-v0.2.7.md`.
- Predecessor: `plans/0.2.6/PR/release-v0.2.6.md` (tagged 2026-09-13).
- Delta covered by the note: 84 commits from `v0.2.6` (`5896872`) to `e974d30`, six merged PRs (#74, #76, #77, #78, #79, #81).
- Issues: see **Related issues** (none open).

---

## Changes

### Release note

- `plans/0.2.7/PR/release-v0.2.7.md`: GitHub Release body. Catalog 407 implemented / 0 partial / 411 unsupported of 818. 53 of 72 next-72 names implemented, 19 left unsupported on purpose. Public `html`, `css`, `layout`, `screen`, and `markup` packages. Chrome flex cases 1-40. `text-decoration: none` on links. Fixture operation checks and 65 visual references. Next-100 stays planned.

### Version stamp

- `VERSION`, `internal/cli.Version`, `bindings/c/include/gowkhtmltopdf.h` `GOWKHTMLTOPDF_VERSION`, `bindings/python` `pyproject.toml` and `__version__`, and the Python test expectation all read `0.2.7`.
- `LibraryVersion` stays `0.12.7-dev`.
- `CHANGELOG.md`: new `## 0.2.7 (2026-10-01)` section. Empty `## Unreleased` left on top.

### Docs and site

- Current-release lines in `README.md`, `documentation/overview.md`, `documentation/getting-started.md`, `documentation/cli.md`, `documentation/library-api.md`, `documentation/python.md`, and the matching frontend pages now say 0.2.7.
- The 2026-09-13 benchmark table still names the 0.2.6 binary it measured (`8aab63a`).
- `documentation/deferred.md`: the 19 next-72 leftovers are a shipped 0.2.7 decision. Next-100 is named as planned.
- `documentation/compatibility-matrix.md`: the VF/palette rows match the Implemented (lite) table. They are not Partial.
- Showcase and compatibility copy no longer call the next-72 wave unreleased.
- `plans/README.md` and `plans/0.2.7/README.md` link the release note.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None in this PR. The 40-case allocation drop (64.3 MB to 28.0 MB warm bytes) already landed in #76. The 500-page snapshot was not remeasured. |
| **Memory** | None in this PR. |
| **Behavior / correctness** | None in this PR. Link underlines, flex geometry, and the 53 CSS properties already landed on master. |
| **API / CLI** | No new flags. Version strings report 0.2.7. `html.Parse`, `css.Apply`, and `layout.Lay` are already public on master. |
| **Dependencies** | None added or removed. |
| **Binary size / build time** | The committed browser WASM is rebuilt from this tree with `wasmVersion` 0.2.7. `frontend/public/wasm/gowkhtmltopdf.wasm` goes from 30,029,477 bytes to 30,753,471. The old file was the 0.2.6 artifact. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None in the API | `Document` / `ImageDocument` stay. `LibraryVersion` stays `0.12.7-dev`. |
| Link underlines (already on master, #74) | An `a[href]` with `text-decoration: none` no longer draws a line. Pass `--print-link-underline` to force one. |

---

## Test plan

Run on this branch before tagging. All of these exited 0 on 2026-10-01.

- [x] `make check-versions` (`versions aligned: 0.2.7`)
- [x] `make test`
- [x] `make golden`
- [x] `make claim-scan` (`claim-scan: clean`)
- [x] `make lint` (`size-check: clean`, 2 allowlisted over-limit files)
- [x] `make wasm-test` (frontend smoke and live-demo browser smoke for PDF, PNG, JPEG)
- [x] `make build` plus the version-stamp assertion (`0.2.7`)
- [x] `CGO_ENABLED=0 go build ./...`
- [x] `make python-binding-test` (rebuilds the c-shared library at 0.2.7, 44 Python tests OK)

### Commands

```sh
make check-versions
make test
make golden
make claim-scan
make lint
make wasm-test
make build
test "$(./bin/gowkhtmltopdf --version | sed -n 's/^Version:[[:space:]]*//p' | tr -d '[:space:]')" = "$(tr -d '[:space:]' < VERSION)"
CGO_ENABLED=0 go build ./...
CGO_ENABLED=1 make c-shared
make python-binding-test
```

---

## Screenshots / sample output

```
versions aligned: 0.2.7
claim-scan: clean
make test: exit 0
make golden: exit 0
size-check: clean (2 allowlisted over-limit files)
make lint: exit 0
Live demo browser smoke passed for PDF, PNG, JPEG, validation, and mobile layout.
make wasm-test: exit 0
version stamp: OK (0.2.7)
CGO_ENABLED=0 build: OK
python-binding-test: 44 tests OK
```

---

## Related issues

- Closes: none (no open issues on the repo as of 2026-10-01)
- Relates to #74 (link underline), #76 (next-72 CSS and Chrome flex cases 1-40), #77 (showcase fixtures 57-64), #78 (site names the next-72 properties), #79 (fixture operation and pixel checks), #81 (public html, css, and layout packages)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`, `enhancement`)
- [x] Related issues filled
- [x] Filled body committed under `plans/0.2.7/PR/pr-release-0.2.7.md`

---

## Follow-ups (out of scope)

- Do not tag or push `v0.2.7` until asked. After merge, the tag must match `VERSION`.
- Next-100 (`plans/0.2.7/88-canonical-0.2.7-next-100.md`) is planned. Fixture-66 is not in this release.
- The pinned visual corpus run and its CI job stay open (`plans/0.2.7/validator-test/`).
- The README performance table is still the 2026-09-13 capture of the 0.2.6 binary.
- Local uncommitted `output/*.pdf` drift and uncommitted edits in `internal/layout/layout.go` and `internal/layout/layout_height_context.go` were left out of this branch's commit.

---

## Reviewer checklist

- [ ] Release note matches the shipped delta and the catalog counts (407 / 0 / 411, 53 of 72)
- [ ] Version files agree (`VERSION`, CLI, C header, Python package)
- [ ] `LibraryVersion` is still `0.12.7-dev`
- [ ] Benchmark snapshot dates that name the 0.2.6 binary were left alone
- [ ] No secrets or generated scratch artifacts committed
- [ ] PR has assignee and labels
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 1 | 1 | 1 |
| `.h` | 1 | 1 | 1 |
| `.html` | 1 | 1 | 1 |
| `.js` | 6 | 13 | 13 |
| `.json` | 6 | 12 | 12 |
| `.jsx` | 1 | 3 | 3 |
| `.md` | 17 | 436 | 22 |
| `.py` | 2 | 2 | 2 |
| `.toml` | 1 | 1 | 1 |
| `.wasm` | 2 | Binary | Binary |
| No extension | 1 | 1 | 1 |
| **Total** | **39** | **471** | **57** |

