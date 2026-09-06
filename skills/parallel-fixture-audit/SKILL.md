---
name: parallel-fixture-audit
description: >
  Audit and fix gowkhtmltopdf implemented-CSS fixtures 60,61,62 (356 rules) by rendering all PDF pages to screenshots, scanning What it should do vs Effect (live) with 4-5 parallel sub-agents, collating defects, then fixing all broken items via parallel sub-agents. Use when user says screenshots of fixtures 60 61 62, 300+ rules, defect live, use 4-5 sub-agents, or runs /parallel-fixture-audit.
---

# Parallel fixture audit — find and fix

Run the 60/61/62 implemented-CSS audit as a parallel find-then-fix loop. Do not run sub-agents sequentially.

## Inputs

- Fixtures: `testdata/golden/fixture-60-implemented-props-a.html` (118), `fixture-61-implemented-props-b.html` (118), `fixture-62-implemented-props-c.html` (120) = 356 rules, catalog `plans/0.2.6/catalog/implemented-fixture-split.json`.
- PDFs: `output/fixture-60-implemented-props-a.pdf`, `output/fixture-61-implemented-props-b.pdf`, `output/fixture-62-implemented-props-c.pdf` (regenerate with `--allow-local-files --font-path testdata/fonts/implemented-audit` if stale).
- Screenshots: render every PDF page to PNG before scanning.

## Phase 1 — Screenshots (all pages)

Render fresh screenshots, not cached:

```bash
python3 << 'PY'
import fitz
from pathlib import Path
srcs=[Path("output/fixture-60-implemented-props-a.pdf"),Path("output/fixture-61-implemented-props-b.pdf"),Path("output/fixture-62-implemented-props-c.pdf")]
out=Path("/tmp/fixtures_screenshots"); out.mkdir(exist_ok=True, parents=True)
dpi=150; m=fitz.Matrix(dpi/72, dpi/72)
for s in srcs:
    d=fitz.open(s)
    for i in range(d.page_count):
        pix=d[i].get_pixmap(matrix=m, colorspace=fitz.csRGB, alpha=False)
        pix.save((out/f"{s.stem}-p{i+1:02d}.png").as_posix())
PY
```

Expect `9 + 10 + 9 = 28` PNGs `1241x1754` at 150 DPI under `/tmp/fixtures_screenshots/`. Verify counts before scanning.

## Phase 2 — Parallel find (4-5 agents, parallel not sequence)

Spawn **4-5** `Task` sub-agents **in one turn** (one `Task` call per message in same assistant turn). Each agent is read-only and owns a disjoint slice — never two agents on same package/file.

Suggested slices (cover all 356):

| Agent | Slice | File ownership (read-only) |
|-------|-------|----------------------------|
| 1 | Flex / Align / Gap / Order / Place | flex layout, gap, place logic |
| 2 | Background / Border / Radius / Outline / Box-Shadow | paint, border_image, box_shadow, outline |
| 3 | Typography / Text / Font / Color | inline_paint, style_text_props |
| 4 | Layout / Sizing / Position / Overflow / Transform | layout, transform, overflow_clip |
| 5 | Grid / Multicol / Table / SVG / Misc | grid, multicol, layout_tables, pseudo_content |

Each agent must:

1. Read its slice from the catalog and fixture HTML.
2. For each property trace: parse (`style_cascade.go:normalizeVendorPrefix`, `style_properties.go:apply*`) → `ResolvedStyle` (`style.go`) → layout/paint consumer (`flex.go`, `layout_chrome.go`, `background_image.go`, `inline_paint.go`, etc.) with `file:line` citations.
3. Compare Expected (`What it should do` column) vs Actual (`Effect (live)` cell and code reasoning) and mark `pass` / `partial` (fixture authorship) / `broken` (engine).
4. Return markdown table: `Property | Fixture:Row | Expected | Actual | file:line | Severity`.

Do not launch agents sequentially — emit all `Task` calls as parallel messages in one turn. Verify sub-agents produced turns before re-spawning.

## Phase 3 — Collate

Merge the 5 tables. Prioritize `broken` (engine ignores value, paints wrong) over `partial` (fixture puts item prop on container, duplicate `gap:20px`+`gap:4px` last-wins, equal heights indistinguishable). Keep `file:line` citations. Report counts: total, pass, partial, broken.

## Phase 4 — Parallel fix (4-5 agents, parallel not sequence)

Delegate fixes to **4-5** parallel `Task` fix agents, each with exclusive file ownership (no two agents edit same file). Do not run `git` commands unless user explicitly asks.

Example ownership:

- Agent 1: `style_values.go`, `style_properties.go`, `style.go`, `style_cascade.go` (place-content swap, max-height:%, grid line -1)
- Agent 2: `inline_paint.go`, `style_text_props.go`, `inline_collect.go` (wavy/dashed, emphasis, vertical-align, tab-size)
- Agent 3: `layout.go`, `transform.go`, `filter.go`, `layout_chrome.go` (translate:%, box-sizing min/max)
- Agent 4: `overflow_clip.go`, `background_image.go`, `border_image.go`, `outline.go`, `style_advanced_props.go` (overflow-clip-margin 1-4 values, bg clip, border-image width)
- Agent 5: `grid.go`, `multicol.go`, `layout_tables.go`, `pseudo_content.go`, `style_paint_props.go` (grid auto, caption-side logical, content:url, fill:none)

Each fix agent: read → minimal edit → `go build ./internal/layout` check → summary `file:line`. Keep `file size < 2000 lines`; extract rather than grow `paint_flow.go`.

## Phase 5 — Verify and regenerate

1. `go build ./internal/layout ./internal/convert`
2. `go test ./internal/layout -count=1 -short` (note `TestFixture60BackgroundImagesStayWithTheirRows` is pre-existing spill, also fails on clean HEAD)
3. `go test ./internal/convert -run TestGoldenCorpusAllFixtures -count=1` (update `fixturePageBounds` in `internal/convert/golden_test.go` if page count changes, e.g. 61 `maxPages 9->11`)
4. Regenerate `output/fixture-60/61/62.pdf` with `--allow-local-files --font-path testdata/fonts/implemented-audit` and re-render screenshots for next audit.

## Phase 6 — Commit (only if user asks)

If user says commit/push and only Go (or Go + 60/61/62 HTML/PDF): stage exactly `internal/layout/*.go`, `internal/convert/golden_test.go`, and `testdata/golden/fixture-60/61/62.html` + `output/fixture-60/61/62.pdf` — not all `output/*.pdf`. No em dashes in messages.

## Output shape

Report:

- screenshot counts and paths
- 5-agent defect tables with `file:line`
- collated priority fix list
- after-fix verification (build/test/golden) and regenerated PDF page counts

Do not write `TODO.md` or `todos.json`; use response todos.
