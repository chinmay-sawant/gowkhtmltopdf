---
name: diagnose-fixture-picture
description: >
  Diagnose gowkhtmltopdf fixture PDFs by screenshot and Effect-cell geometry
  (implemented-props 60/61/62 or any golden output PDF that looks wrong). Classifies
  placeholder authorship vs wrong-element demos vs invalid/initial values vs engine
  bugs, runs a 3-agent analysis council, fixes with exclusive ownership, then verifies
  with 4 parallel "is this picture right" agents. Use when Effect (live) looks empty
  or identical across rows, user pastes prop rows from a PDF, asks if a fixture is a
  placeholder, or runs /diagnose-fixture-picture. Not for structural golden failures
  (page envelope / needles) - that is diagnose-golden-fixture.
---

# Diagnose a fixture picture

Picture-first loop for print fixtures. Trust crops and measured paint, not memory
of CSS. No git commands unless the user asks.

## Thinking order (do not skip)

1. **Picture before theory.** Fresh PDF + page PNG + Effect-column crop.
2. **Measure, then look.** Word `(x,y)`, vertical rules, fill rects via PyMuPDF.
   Vision alone misreads 8pt text; coords do not.
3. **Classify authorship before blaming the engine.** Most "empty" Effect cells
   are fixture mistakes, not missing Go.
4. **One red probe.** Smallest HTML that still shows the bad geometry or the
   missing style field. Print `ResolvedStyle` and `Op` positions.
5. **Fix the class.** Hand-edit the fixture row and the generator branch so regen
   cannot wipe the demo. Touch engine code only when the probe proves a consumer
   or parse bug.
6. **Picture again.** Re-crop the same rows. Ask "is this picture right?" against
   the Expected column, not against vibes.

## Authorship classes (check in this order)

| Class | Smell | Fix home |
|-------|-------|----------|
| Placeholder | Effect text is `list/content demo`, `applied`, or identical across many props | Rewrite Effect HTML |
| Wrong element | Item prop (`grid-column`, `list-style`, `content`) on the container | Move prop onto child / `::before` / real `ul` |
| Invalid or initial | `auto` / illegal keyword where the initial look is invisible (`grid-auto-flow:auto`, `grid-template-areas:auto`) | Use a distinctive legal value |
| Engine | Parse stores wrong units, paint missing, layout ignores field | Package test + owning `.go` file |

`display:grid` with A/B children is **not** proof of every `grid-*` name. Same
stacked geometry for every row means the named property is not demonstrated.

## Inputs

- Fixture HTML under `testdata/golden/` (often `fixture-60/61/62-implemented-props-*.html`)
- PDF under `output/` (regenerate if stale)
- Fonts for 60/61/62: `--font-path testdata/fonts/implemented-audit`
- Optional good PDF when the user supplies a prior sample (regression branch)

## Phase 1 - Fresh pictures

```bash
make build
./bin/gowkhtmltopdf --allow-local-files --font-path testdata/fonts/implemented-audit \
  -o output/<fixture>.pdf testdata/golden/<fixture>.html
```

Render pages and Effect crops (150-200 DPI). Helper:

`python3 skills/diagnose-fixture-picture/scripts/render_fixture_pages.py output/<fixture>.pdf /tmp/fixture_pics`

Completion: page PNGs exist; you can name the page that holds the suspect rows.

## Phase 2 - Measure the Effect cell

For each suspect row index:

1. Read the fixture HTML Effect markup (inline styles and classes).
2. Extract words/draws in the Effect band: text, `x,y`, rule color/width, fills.
3. Assign one authorship class from the table above, or `engine`.

Completion: a table `Row | Property | Class | Evidence` with at least one
measured number or HTML quote per row.

## Phase 3 - Analysis council (3 parallel explore agents)

Spawn **three** read-only agents **in one turn**. Disjoint jobs, shared evidence
from Phase 2:

| Role | Job | Must return |
|------|-----|-------------|
| Analyst | Expected (`What it should do`) vs Actual (crop + HTML + paint metrics). Trace parse → `ResolvedStyle` → consumer with `file:line` when class is `engine`. | Defect table with class |
| Interpreter | Rank 3-5 falsifiable hypotheses from the Analyst table. Each needs a prediction ("if X, changing Y flips the crop"). Prefer authorship hypotheses first. | Ranked list |
| Critic | Attack the ranking. Reject any hypothesis without a prediction. Call out vision-only claims, wrong-element misses, and engine blame without a probe. | Surviving hypotheses only |

Do not fix during council. Collate Critic survivors into the fix list.

## Phase 4 - Red probe (orchestrator)

For each surviving `engine` hypothesis, write a throwaway or package test that
prints style fields and text/line ops. For authorship-only rows, skip to Phase 5
with a rewritten demo sketch.

Completion: one command already run that is red on the bug class (or an explicit
authorship-only verdict).

## Phase 5 - Fix (up to 4 parallel fix agents)

Exclusive file ownership. Typical split when several rows land together:

1. Fixture HTML (`testdata/golden/fixture-6*.html`)
2. Style parse (`style_values.go`, `style_properties.go`, `style_paint_props.go`, `style_cascade.go`)
3. Layout consumer (one of `grid.go` / `multicol.go` / `flex.go` / …)
4. Paint / chrome (`layout_chrome.go`, `outline.go`, `inline_paint.go`, …)

Rules: minimal edit; keep `paint_flow.go` from growing; add or extend a package
test for engine fixes; update generator when hand-editing 60/61/62 demos.

## Phase 6 - Picture council (4 parallel verify agents)

After regenerate + re-crop, spawn **four** read-only agents **in one turn**. Each
owns a disjoint row slice (or page). Question for every row:

> Is this picture right for the Expected column?

Each agent returns `Row | yes/no | what the crop shows | leftover risk`.
Any `no` re-enters Phase 2 for that row only.

Also run:

```bash
go test ./internal/layout -count=1 -short
go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/<fixture>' -count=1
```

Update `fixturePageBounds` only when page count actually moved.

## Regression branch (good vs current)

When the user gives a known-good PDF: render the same page from both, crop the
broken region, measure the delta (geometry / pagination / paint / reflow), then
enter Phase 3 with that metric table as Analyst input. Still no worktrees.

## Output shape

```
Rows: …
Classes: placeholder=N wrong-element=N invalid/initial=N engine=N
Council: surviving hypotheses …
Fix: file:line …
Picture council: all yes | remaining nos …
Proof: test commands + PDF path
```

## Related

- `diagnose-golden-fixture` - page envelope, needles, structural golden red
- `debug-html-template` - template symptom pick list, wait for user choice
