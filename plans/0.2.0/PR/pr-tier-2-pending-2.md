## Summary

Ships the next Tier 2 pending slice after #17: print-scoped `position: sticky`,
image-mode `@font-face`, go-text OpenType shaping path, headers/TOC/HF GoTo
polish, evidence-backed subplans for remaining flex/grid work, and sticky
continuation-page layout fixes (fixture-31) so page breaks keep row chrome,
section borders, and white cell backgrounds.

---

## Motivation / context

- Plans: `plans/phases/subplans-tier-2/` (sticky-print, image-mode-fontface,
  shaping-gotext-typesetting, nested-hf-v0.3.0, flex-grid-full, phase-17–20 pending)
- Branch: `feature/tier-2-pending-2` from `master` @ #17 merge
- Constraint: **stdlib-only** (no CGO / no real browser sticky overflow scroll)

---

## Changes

### Print sticky (phase 17)

- Page content box as sticky scrollport; clamp + continuation clones
- Reserve flow under sticky bar; page-leading section chrome stays put while
  thead-style rows shift
- Fixture-31 + layout regressions for overlap, spacing, borders, orphans, and
  Row 28 white background on continuation pages

### Fonts / image / HF

- Image-mode `@font-face` wiring
- go-text OpenType shaping amendment + path
- Nested HF / GoTo polish for phase 20 edges

### Docs & fixtures

- Subplans for remaining flex/grid and phases 17–20 pending work
- Golden fixtures 29–31 (float-beside-table, orphans heuristic, sticky-top)
- Regenerated sample PDFs under `output/`

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Sticky print clones; continuation row fills/borders; HF links; image `@font-face` |
| **Performance** | Unchanged hot path; sticky runs once per paint after rect splits |
| **Memory** | Negligible (sticky clone ops per continuation page) |
| **API / CLI** | Unchanged flags |
| **Dependencies** | None new for sticky; go-text path per amendment |
| **Binary size / build time** | Sample PDFs refreshed |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `go test ./internal/layout/ -count=1`
- [x] Sticky fixture-31 regressions (`TestStickyFixture31*`, Row 28 white bg, no page-1 orphans)
- [x] Regenerated `output/fixture-31-sticky-top.pdf` (page 1 clean end; page 2 sticky + white Row 28)
- [ ] CI `go test ./...` + lint + `CGO_ENABLED=0` static build

### Commands

```sh
go test ./internal/layout/ -count=1 -run Sticky
go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-31-sticky-top.html output/fixture-31-sticky-top.pdf
```

---

## Screenshots / sample output

- `output/fixture-31-sticky-top.pdf` — sticky bar on continuation; Row 28 white cell; page 1 ends at Row 27 without empty shells

---

## Related issues

- Relates to #17 (prior Tier 2 pending merge)
- Relates to #16 (Tier 2 core)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body under `plans/PR/pr-tier-2-pending-2.md`

---

## Follow-ups (out of scope)

- Full flex/grid deepening (`plans/phases/subplans-tier-2/flex-grid-full.md`) on `feature/pending-tier-2-flex-grid`
- Overflow-scroll sticky (unsupported; print scrollport only)

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed
