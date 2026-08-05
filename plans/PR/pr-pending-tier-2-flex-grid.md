## Summary

Closes the Tier 2 flex-grid-full workstream on a stdlib-only report engine:
Stage A flex deepen, Stage B grid (rows/`fr`/areas/dense/`minmax`), Stage C
lite (cyclic `%`, subgrid copy-inherit, masonry pack), plus pagination fixes
so CI no longer hangs and fixture-31/33 match HTML.

---

## Motivation / context

- Plans: `plans/phases/subplans-tier-2/flex-grid-full.md`
- Branch: `feature/pending-tier-2-flex-grid` from `master` @ #18 merge
- Constraint: **stdlib-only** layout (no browser embed); honesty over Chrome
  pixel parity
- Prior PR #18 called out this workstream as the next Tier 2 pending slice

---

## Changes

### Flex Stage A (`internal/layout/flex.go`, `style.go`)

- Independent `row-gap` / `column-gap`
- `flex-direction: row-reverse` / `column-reverse`
- `justify-content: space-around` / `space-evenly`
- `align-content` for wrapped multi-line flex
- Column path grow/shrink/justify/align parity with row
- Shorthand `flex: grow shrink basis`, `align-self`, content-based min-size floor
- Percentage flex-basis: definite CB resolves `%`; indefinite/cyclic → auto
  (content) — fixture-33
- Row `align-items: stretch` sizes auto-height items to the flex line cross
  size (fixture-33 50%/50% boxes fill 36pt)

### Grid Stage B + Stage C lite (`internal/layout/grid.go`, `style.go`, `layout.go`)

- `grid-template-rows` / `fr` / row span + default stretch into grid areas
  (fixture-32 Tall span-2)
- `grid-template-areas` + `grid-area` name placement; `grid-auto-flow: dense`
  — fixture-34
- Full `minmax()` subset (lengths / `%` / `fr` / `auto` / min-/max-content)
- Intrinsic measure lite + cyclic `%` width/height honesty
- `display: subgrid` copy-inherit parent templates (no shared track sizing)
- One-axis masonry shortest-stack pack (both axes → dense fallback) —
  fixture-35

### Pagination / paint fixes (`internal/layout/paint.go`)

- **splitCrossingRects:** rebuild fragments in a new slice + epsilon page math
  (fixes `TestTenPageTableReportPerformance` 10m hang from float-edge infinite
  mid-slice inserts)
- Page-break same-row chrome: tight `minY` snap so fixture-31 Row 28 white
  fill stays above the baseline (not section gray)
- Tighten `isSectionWashRGB` so chromatic grid washes are not clipped as
  section greys

### Fixtures & docs

- New goldens only (25/28 untouched): fixtures **32–35**
- Compatibility matrix §2.7 / §2.8; Phase 17 markers; flex-grid-full Stage
  A/B complete **2026-08-05**, Stage C `[~]` with listed gaps
- Landscape 2026 comparison notes under `documentation/comparison-with-others/`
- Regenerated sample PDFs under `output/`

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Deeper flex/grid; cyclic `%`; areas/dense/`minmax`; Row 28 white; fixture-33 stretch |
| **Performance** | Fixes pathological paint hang; local measure passes for stretch/grid |
| **Memory** | Negligible |
| **API / CLI** | Unchanged |
| **Dependencies** | None |
| **Binary size / build time** | Sample PDF refresh |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make lint`
- [x] `go test ./internal/layout ./internal/convert -count=1`
- [x] `go test ./internal/convert -run TestTenPageTableReportPerformance` (~0.2s)
- [x] Sticky Row 28 white + Flex stretch + Grid areas/dense/minmax unit tests
- [x] CI `test + lint` + `CGO_ENABLED=0` static build green on #19
- [x] `make samples` regenerated output fixtures 32–35 (+ suite)

### Commands

```sh
make lint
go test ./internal/layout ./internal/convert -count=1 -timeout 120s
make samples
```

---

## Screenshots / sample output

- `output/fixture-32-flex-grid-full.pdf` — Tall span-2 fills both `1fr` rows
- `output/fixture-33-flex-cyclic-basis.pdf` — 50%/50% row boxes stretch to 36pt
- `output/fixture-34-grid-areas-dense.pdf` — named areas + dense packing
- `output/fixture-35-grid-minmax-intrinsic.pdf` — minmax / subgrid / masonry lite
- `output/fixture-31-sticky-top.pdf` — page-2 Row 28 white (not section gray)

---

## Related issues

- Relates to #18 (Tier 2 pending-2; called out flex-grid follow-up)
- Relates to #16 (Tier 2 core)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body under `plans/PR/pr-pending-tier-2-flex-grid.md`

---

## Follow-ups (out of scope)

- True shared-track subgrid / full CSS Grid L3 masonry / Chrome layout-test parity
- Multi-pass flex intrinsic sizing beyond cyclic `%` → auto subset
- **Next:** Phase 21 site corpus stress against Stage A/B/C-lite layout

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or unrelated generated artifacts committed
