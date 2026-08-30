# Honesty gates (phases 57-84)

Agents keep failing this ledger by flipping `catalog/mapping.json` to `implemented` and checking `[x]` without engine work. That is forbidden.

## Hard rules

1. **Code first.** A property is Implemented only when an apply arm writes `ResolvedStyle` **and** a layout/paint/pagination consumer reads that field for the claimed values.
2. **Mapping last.** Edit `mapping.json` / `coverage-summary.json` only after the consumer exists and tests pass.
3. **Matrix agrees.** `documentation/compatibility-matrix.md` must describe the same subset the code has. If the matrix still says Partial / non-goal, you may not mark Implemented.
4. **No fake proofs.** These citations never close an Implemented row:
   - `applyIgnoredGroup` / style fallthrough (`style_cascade.go` apply loop with no matching `case`)
   - Tests named `*Rejected*`, `*Ignored*`, or assertions that a value is dropped
   - Unrelated packages (SVG `<img>` raster for CSS `fill`, PDF tagging for `speak`, overflow clip for `clip-path`)
   - `python3 scripts/css-catalog-map.py --check` alone (it only checks apply-arm inventory consistency)
5. **Empty `code_path` is a smell.** Implemented rows need a real path into `internal/`.
6. **Do not mass-flip.** One property family per change set. Recount after each honest batch.

## Required flip packet (paste into the phase proof)

Before `[x]` on any “flip to Implemented” row, the commit or agent report must include:

```text
PROPERTY: <name>
APPLY: internal/layout/<file>.go:<line> case "<name>":
FIELD: ResolvedStyle.<Field>
CONSUMER: internal/layout/<file>.go:<line> (reads the field)
TEST: go test ./internal/layout -run <TestName>  (exit 0)
MATRIX: documentation/compatibility-matrix.md section updated
MAPPING: engine_status implemented; code_path set; notes name the subset
```

If any line is missing, leave the row `[ ]` and leave `engine_status` as `partial` or `unsupported`.

## How to add support (landing order)

Same as `AGENTS.md`, repeated here so agents do not skip it:

1. Parser only if needed (new unit/function/at-rule/selector).
2. `ResolvedStyle` field + `initialStyle` (`internal/layout/style.go`).
3. `inheritableProps` if inherited (`style_cascade.go`).
4. `case "prop":` in the right `apply*Group` (`style_properties.go` / `style_paint_props.go`).
5. Consumer in layout/paint/pagination.
6. Package test (+ golden if paint/pagination changes).
7. Matrix row, then mapping flip + recount.

## 2026-08-28 incident

Phases 69-77 were marked complete with all 247 former Ignored names set to Implemented. Code still rejected 3D, ignored animation, and had no apply arms for vendor prefixes / SVG fill / scroll-* / speech / etc. Catalog was reverted to **166 Implemented / 9 Partial / 643 Unsupported**. Do not repeat that pattern.
