# fix(lint): clear findings in the hot packages

## Summary

- Clear the six golangci-lint findings that make the `test + lint` job fail on `master`, so the job can go green.
- None of the findings is related to recent feature work: all six arrived with PR #83 (`fix/viewport-percentage-height`).
- The `buildBlock` fix also returns `internal/layout/layout.go` to 2362 lines, the count recorded in `scripts/file-size-allowlist.txt`, removing the hidden `size-check` failure that sat behind the golangci-lint failure.

---

## Motivation / context

- Plans: none. This is a CI unblock, not a planned change.
- Issues: see **Related issues**.

`master`'s `ci` workflow is red. The failing job is `test + lint`, and its Lint step stops on six findings in `internal/layout`: one `funlen` on `buildBlock`, four `wsl` and one `lll` in `viewport_height_test.go`. Because `make lint` runs golangci-lint first, it hides a second pre-existing problem: `internal/layout/layout.go` was 2372 lines against the 2362 recorded in `scripts/file-size-allowlist.txt`, so `size-check` would have failed next. The `buildBlock` extraction removes exactly the 10 lines PR #83 added to the file, so the ledger matches again with no allowlist edit.

---

## Changes

### `internal/layout/layout.go`: funlen on `buildBlock`

- `buildBlock` was 61 counted lines against the limit of 60. The two adjacent auto-sized control rules (meter/progress intrinsic height, textarea rows * line-height) moved into a new `autoSizedControlContentBottom` helper in `internal/layout/layout_widgets.go`, next to `applyCheckboxAutoSize`.
- Behaviour is identical. The helper runs the same two branches in the same order on the same values: `buildBlock` passes the `boxStyle` pointer it already holds, and `textareaAutoContentBottom` only reads through it.
- `buildBlock`'s existing `//nolint:cyclop,wsl` directive loses `wsl`, which `nolintlint` flagged as unused after the extraction. `cyclop` stays with its original reason.
- The extraction removes 10 lines from `layout.go` (2372 to 2362), matching the recorded count in `scripts/file-size-allowlist.txt`. No allowlist change is needed.

### `internal/layout/viewport_height_test.go`: four wsl and one lll

- Four `wsl` findings ("only one cuddle assignment allowed before if statement", lines 36, 62, 68, 89): a blank line now separates the assignment run from the flagged `if`.
- One `lll` finding (line 86, 124 characters): the `layoutHTMLAtViewport` call is wrapped across two lines.
- No assertion changed.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. The helper inlines to the same calls. |
| **Memory** | None. |
| **Behavior / correctness** | None. The extracted code runs the same branches in the same order on the same values. |
| **API / CLI** | None. Every change is internal and unexported. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make test GO_TEST_FLAGS='-count=1'` (full suite through the Makefile caps; AGENTS.md forbids bare `go test ./...` on this host)
- [x] `golangci-lint cache clean && golangci-lint run ./...` (exit 0, no findings)
- [x] `go build ./...`
- [x] `make golden`
- [x] `bash scripts/check-file-size.sh`
- [x] `make claim-scan`
- [x] `gofmt -l` on the three touched files
- [ ] `make run` wall time vs baseline: not applicable, no runtime path changed

### Commands

```sh
golangci-lint cache clean && golangci-lint run ./...   # exit 0, no findings
go build ./...                                          # ok
make test GO_TEST_FLAGS='-count=1'                      # exit 0, all packages ok
make golden                                             # exit 0, golden corpus passes
gofmt -l internal/layout/layout.go internal/layout/layout_widgets.go internal/layout/viewport_height_test.go  # empty
bash scripts/check-file-size.sh                         # clean, 2 allowlisted over-limit files
make claim-scan                                         # clean
```

---

## Screenshots / sample output

Not applicable: no rendering output changes.

---

## Related issues

- None. No issue was filed for this work, so no ticket ID is claimed.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`bug`)
- [x] Related issues filled with real ticket IDs (none exist; stated plainly)
- [x] Filled body committed under `plans/PR/pr-lint-hot-packages.md`

---

## Follow-ups (out of scope)

- `internal/layout/inline_paint.go` (2021 lines) and `internal/layout/layout.go` (2362 lines) remain over the 2000-line soft limit. That is recorded, pre-existing drift.
- `internal/layout`'s shared `*go-text/typesetting/font.Face` lazy caches are unsynchronised, so `go test -race ./internal/layout/` flakes intermittently on `master`. CI's race job passes; fixing the face sharing is a separate change.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 3 | 31 | 13 |
| `.md` | 1 | 130 | 0 |
| **Total** | **4** | **161** | **13** |
