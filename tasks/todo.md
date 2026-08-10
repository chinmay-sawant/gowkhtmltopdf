# Fixture 56 renderer fidelity

Scope: repair the Go renderer for `fixture-56-architecture-diagram` without
changing `testdata/golden/fixture-56-architecture-diagram.html` or its CSS.

## Baseline findings

- [x] Capture all 13 Gowkhtmltopdf PDF pages and 13 browser-rendered HTML reference pages.
- [ ] Keep the hero, pipeline, TOC, and domain content aligned with the HTML reference.
- [x] Preserve CSS grid/flex column widths so package/detail text wraps instead of clipping.
- [ ] Keep diagram nodes and flow rows in the same arrangement as the HTML reference.
- [x] Preserve light/dotted borders, colored rails, card fills, and section heading rules.
- [x] Preserve inline `code`, `mark`, `kbd`, and semantic-element visual treatments.
- [x] Render `meter` and `progress` values as their intended visual bars.
- [x] Respect `details` open/collapsed state in print layout.
- [ ] Preserve list markers and remove stray/clipped glyphs at page boundaries.
- [ ] Reduce pagination drift caused by text metrics, wrapping, and fragmented sections.
- [ ] Keep the DAG, divergence, security, and colophon composition stable across the final pages.

## Implementation and validation

- [x] Add focused Go regression coverage at the correct layout/paint seam.
- [ ] Re-render the fixture and compare all pages against the HTML reference.
- [x] Run the approved Go checks after the renderer fix.

## Renderer evidence

- Go seam fixes cover embedded `var()` substitution, fallback-first modern
  declarations, border side longhands, inline chrome measurement/paint,
  flexible grid overflow, explicit flex minimums, native value widgets, and
  details disclosure state.
- The regenerated PDF has 14 pages versus the 13-page browser reference. The
  remaining pagination and final-page composition drift is intentionally still
  open for a follow-up pass.
- HTML hash remains `1113b02b4cd1e641b6748dce0a6f67eb0a0558f41f99ed90f87244907b4d58c6`;
  CSS hash remains `db120ba8bcaf10fb625ea61d7013164a9288c84fcfe00b3aca3216d52d4ed9ab`.

## Guardrails

- HTML and CSS fixture files are read-only for this task.
- No Git commands.
- Preserve the existing layout, display-list, paint, pagination, and PDF architecture.
- A fresh 4-5 subagent wave was attempted but the session thread limit prevented new agents; investigation continues at the existing seams.
