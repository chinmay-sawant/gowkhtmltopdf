## Summary

Fixes visual Effect-cell demos on fixtures 61 and 62, and a few layout bugs those demos exposed: column-rule px width, transform restamp after pagination, and vertical-rl shrink-to-fit width. Sample PDFs and the diagnose-fixture-picture skill are updated with the same work.

## Motivation / context

- Branch: `chore/fix-fixtures`
- Goal: make Implemented CSS audit fixtures checkable by eye against Chrome-style expectations, without placeholder Effect cells
- Related issues: none opened for this batch

## Changes

### Layout engine

- `column-rule` / outline px lengths go through `borderPaintWidth` instead of treating `1px` as `1pt`
- After pagination, `restampBoxTransforms` rebakes transform origins so scale/rotate chips stay with their boxes on later pages
- `writing-mode: vertical-rl` / `vertical-lr` intrinsic width uses one glyph column (line-height / font-size), matching Chrome shrink-to-fit for narrow vertical columns

### Fixture 61 (implemented props B)

- Multicol rows 23-32: taller cells and distinct longhand demos
- Live `content` / counter demos via `::before`
- Grid props 73-85: item placement, auto-flow, named areas, explicit tracks instead of identical A/B stubs

### Fixture 62 (implemented props C)

- Overflow-clip-margin rows: directional protrusion demos
- Padding sides: fixed-width blue shell + orange chip so each side band is visible
- Opacity, place-items/place-self, clip-margin-top, skip-ink, text-shadow, decorations
- Transform demos (rotate, scale, transform, transform-box, transform-origin, translate) use centered stages; table no longer forces a sparse page break before writing-mode
- Regenerated `output/fixture-62-implemented-props-c.pdf`

### Tooling / samples

- Removed unused `scripts/gen-implemented-prop-fixtures.py` (committed HTML is the source of truth)
- Replaced parallel fixture-audit / pdf-regression skills with `skills/diagnose-fixture-picture`
- Regenerated sample PDFs under `output/` after fixture-61 fixes

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None expected |
| **Memory** | None expected |
| **Behavior / correctness** | Column-rule paint width, post-pagination transforms, vertical-rl fit-content width |
| **API / CLI** | None |
| **Dependencies** | None |
| **Binary size / build time** | None material |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] Targeted layout tests for column-rule, transform center/restamp, vertical-rl width
- [x] `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-62'`
- [ ] `make test` (full suite at review time)
- [ ] `make lint`
- [ ] `make golden` if reviewing layout/paint changes end to end
- [ ] Spot-check fixture-61/62 Effect cells vs Chrome for transform and writing-mode rows

### Commands

```sh
go test ./internal/layout/ -run 'ColumnRule|Transform|VerticalRL|TableCellTransformed' -count=1
go test ./internal/convert/ -run 'TestGoldenCorpusAllFixtures/fixture-6[12]' -count=1
make test
make lint
```

## Screenshots / sample output

Chrome vs ours writing-mode probe (same HTML): yellow vertical-rl box width ~16.5pt (Chrome) vs ~17.6pt (ours after fix); previously ~40pt wide.

Transform Effect cells on fixture-62 use `.xform-stage` / `.xform-chip`; chips stay inside the dashed stage after pagination restamp.

## Related issues

- None for this batch

## Commits on branch (`master..HEAD`)

- `3069edc` fix(layout): column-rule px width and fixture-61 multicol demos
- `f8d35f5` chore(output): regenerate sample PDFs after fixture-61 fixes
- `e9035bd` fix(fixtures): show live content and counter none demos
- `0cac57c` fix(fixtures): distinct live demos for grid props 73-85
- `31e2eac` chore(skills): replace picture audit skills with diagnose-fixture-picture
- `9497c77` fix(fixtures): live overflow-clip-margin demos on fixture-62
- `9e52853` fix(fixtures): distinct padding Effect demos on fixture-62
- `c006af9` chore: remove unused implemented-props fixture generator
- `daf657c` fix(fixtures): contain transforms and live decoration demos on fixture-62
- `66e5fb5` fix(fixtures): skip-ink contrast, writing-mode page break, clearer shadow
- `8dde162` fix(fixtures): opacity, place-items/self, clip-top, and scale demos
- `93ae738` fix(fixtures): self-framed transform chips and end-table packing
- `041b9f7` fix(layout): restamp transforms after pagination for centered stages
- `371a05f` fix(layout): size vertical-rl columns like Chrome shrink-to-fit

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled (none for this batch)
- [x] Filled body under `plans/PR/pr-fix-fixtures.md`

## Follow-ups (out of scope)

- Full per-glyph vertical typesetting (upright Latin stacking) beyond the current -90deg lite path
- Engine clip of transformed paint inside `overflow:hidden` table cells

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
