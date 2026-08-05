## Summary

Deepens report-engine Flexbox (Stage A) and CSS Grid (Stage B lite) toward the
Tier 2 flex-grid-full subplan: independent gaps, reverse directions, space-*
justify, align-content/self, flex shorthand, grid rows/`fr`/row-span, and
default `align-items: stretch` so spanning cells match HTML height. Adds
fixture-32 as proof without editing fixtures 25/28.

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

- Independent `row-gap` / `column-gap` (no longer collapse to a single gap)
- `flex-direction: row-reverse` / `column-reverse`
- `justify-content: space-around` / `space-evenly`
- `align-content` for wrapped multi-line flex
- Column path grow/shrink/justify/align parity with row
- Shorthand `flex: grow shrink basis`, `align-self`, content-based min-size floor

### Grid Stage B lite (`internal/layout/grid.go`, `style.go`)

- Consume `grid-template-rows` when container height is definite
- `grid-row` / start / end / `span` with 2D occupancy
- `fr` track distribution (+ lite minmax); independent row/column gaps
- `justify-items` / `align-items` / self variants
- **Stretch fix:** default `align-items: stretch` sizes items (including
  `grid-row: span 2`) to the grid area so fixture-32 “Tall span-2” matches HTML

### Paint honesty (`internal/layout/paint.go`)

- Tighten `isSectionWashRGB` so chromatic grid washes (e.g. `#f3e5f5`) are not
  clipped as neutral section greys

### Fixtures & docs

- New golden: `testdata/golden/fixture-32-flex-grid-full.html`
- Wire into `internal/convert/golden_test.go`, `testdata/golden/README.md`,
  compatibility matrix, and flex-grid-full checklist progress
- Regenerated sample PDFs under `output/` (including fixture-32)
- Unit coverage expansions in `flex_test.go` / `grid_test.go`

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Deeper flex/grid layouts; span-2 cells stretch to row tracks |
| **Performance** | Extra measure pass for stretch grid items (local to grid containers) |
| **Memory** | Negligible |
| **API / CLI** | Unchanged |
| **Dependencies** | None |
| **Binary size / build time** | Sample PDF refresh only |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `go test ./internal/layout -run 'Flex|Grid' -count=1`
- [x] `TestGridRowSpan` / `TestGridRowSpanStretchMatchesFixture32`
- [x] Regenerated `output/fixture-32-flex-grid-full.pdf` (Tall span-2 height matches HTML)
- [ ] CI `go test ./...` + lint + `CGO_ENABLED=0` static build

### Commands

```sh
go test ./internal/layout -run 'Flex|Grid' -count=1
go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-32-flex-grid-full.html \
  output/fixture-32-flex-grid-full.pdf
```

---

## Screenshots / sample output

- `output/fixture-32-flex-grid-full.pdf` — Grid rows + row span: Tall span-2
  fills both `1fr` rows (+ gap); B/C and D/E flush with tall cell edges

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

## Follow-ups (out of scope for this PR)

- Flex cross-axis `align-items: stretch` still packs as start (matrix `[~]`)
- True shared-track subgrid / full CSS Grid L3 masonry / Chrome layout-test parity
- Multi-pass flex intrinsic sizing beyond cyclic `%` → auto subset
- **Next:** Phase 21 site corpus stress against Stage A/B/C-lite layout
- CI `make test` on PR merge (layout + convert proof recorded on subplan 2026-08-05)

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or unrelated generated artifacts committed
