## Summary

Add PDF operation checks and pixel-level visual tests for the numbered fixture PDFs. The visual suite now has references for all 65 body fixtures and a serial comparison test. Add a focused Ghostscript licensing page and link to it from the README. Phase 94 checks remain until the pinned comparison and closure gates pass.

## Motivation / context

Phase 94 and Phase 95 of `plans/0.2.7/validator-test/` track the PDF fixture checks. Operation tests compare measured PDF content. The visual test renders PDFs to page images and compares every RGB pixel. Keep both until the pinned corpus comparison passes because pixel checks do not cover PDF structure or link targets.

## Changes

### PDF fixture checks

- Add a PDF operation reader and per-fixture checks for page boxes, text, strokes, fills, and image placements (`internal/pdf/page_ops.go:62-80`, `internal/convert/fixturetests/output_ops_test.go:57-110`).
- Move fixture checks into `internal/convert/fixturetests/` and share fixture request setup. Keep fixture conversions serial because shared font and subset state can make exact output comparisons nondeterministic.
- Make equal-priority Unicode aliases choose the lowest code point in `Font.reverseCmap()` (`internal/pdf/shape_gotext.go:513-545`).

### Pixel comparison

- Pin raster comparison to Ghostscript 10.08.0, `png16m`, 150 DPI, and smoothing level 4 (`internal/convert/fixturetests/pixel_regression_test.go:16-20`).
- Add a serial corpus test and an inventory check. The inventory expects 65 body PDFs, including both fixture 29 files, and excludes fixture 36 header and footer files (`internal/convert/fixturetests/pixel_regression_corpus_test.go:31-73`, `:118-210`).
- Add reviewed references for fixtures 02-64 under `output/validated/`. Fixture 01 retains its earlier approved reference.
- Keep fixture 01 saved-candidate and mutation checks separate from the full corpus conversion (`internal/convert/fixturetests/pixel_regression_cases_test.go:103-192`).

### Plans and instructions

- Update the output guide, Phase 94 and Phase 95 checklists, and the v0.2.7 plan index with current evidence and open gates.
- Add the fixture inspection script and fixture-test authoring guide in the branch history.

### Ghostscript licensing

- Add `documentation/ghostscript-licensing.md` with the repo's test-only Ghostscript use, MIT boundary, and separate distribution and server-use cases.
- Keep the README note short and link it to the licensing page.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No application runtime change. The opt-in corpus test converts and rasterizes 65 PDFs serially. |
| **Memory** | No application memory change. Test raster pages use temporary storage. |
| **Behavior / correctness** | Equal-priority Unicode aliases now resolve deterministically. Fixture checks add structural and visual regression coverage. |
| **API / CLI** | No public API or CLI change. |
| **Dependencies** | No Go dependency added. Ghostscript remains a separately installed test tool. |
| **Binary size / build time** | No product binary size change. |

## Breaking changes / migration

| Item | Migration |
|-----------|-----------|
| None | - |

## Test plan

- [x] `GOCACHE=/tmp/gowkhtmltopdf-publish-cache make test` (passes, including `TestVisualReferenceInventory`)
- [x] `GOCACHE=/tmp/gowkhtmltopdf-publish-cache make golden` (passes)
- [x] `make claim-scan` (passes with the licensing page and README link)
- [x] `GOCACHE=/tmp/gowkhtmltopdf-publish-cache make build` (passes)
- [x] `make size-check` (passes)
- [x] `golangci-lint run --new-from-rev=537df27 ./internal/convert/fixturetests ./internal/pdf` (passes for the new and edited Go code)
- [ ] `make lint` (still reports `paralleltest` warnings in the deliberately serial fixture operation tests)
- [ ] Pinned 65-fixture pixel comparison (not run; this host has Ghostscript 9.55.0, while the comparator requires 10.08.0)

### Commands

```sh
GOCACHE=/tmp/gowkhtmltopdf-publish-cache make test
GOCACHE=/tmp/gowkhtmltopdf-publish-cache make golden
make claim-scan
GOCACHE=/tmp/gowkhtmltopdf-publish-cache make build
make size-check
GOCACHE=/tmp/gowkhtmltopdf-publish-cache make lint
```

The ordinary test run skips the pixel comparison when `GOWKHTMLTOPDF_VISUAL_GS` is unset. It still runs the reference inventory check.

## Screenshots / sample output

The 65 approved reference PDFs are under `output/validated/`. Candidates for fixtures 01-64 were reviewed as contact sheets rendered with Ghostscript 9.55.0 at 48 DPI. That review does not replace the pinned 10.08.0 comparison.

## Related issues

- No related issue was found. The work is tracked in `plans/0.2.7/validator-test/95-canonical-pixel-regression-tests.md`.

## PR metadata checklist (author)

- [x] Self-assigned with `--assignee "@me"`
- [x] Labels selected: `enhancement` and `documentation`
- [x] Related issue search completed; no matching ticket found
- [x] Filled body committed under `plans/PR/pr-pixel-regression-fixture-corpus.md`

## Follow-ups (out of scope)

- Run the full corpus comparison with Ghostscript 10.08.0.
- Add a `make visual-golden` target and a CI job that installs the pinned renderer.
- Resolve the `paralleltest` lint findings without running shared-state fixture conversions in parallel.
- Close Phase 94 only after the pinned comparison and required final gates pass.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct keywords
- [ ] No secrets or unintended generated artifacts are committed; PDF references and samples are intentional
- [ ] Diff-stat-by-extension table included below

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 75 | 4977 | 2 |
| `.md` | 19 | 1433 | 5 |
| `.pdf` | 68 | Binary | Binary |
| `.py` | 1 | 116 | 0 |
| **Total** | **163** | **6526** | **7** |
