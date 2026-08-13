---
name: debug-html-template
description: >
  Debug an HTML-to-PDF template when the rendered page wraps, misaligns,
  overlaps, or ignores CSS. Diagnose only: walk the symptom to the CSS,
  split measure vs paint and parse vs apply, then table the possible
  fixes and wait for the user to pick one. Engine only when measure or
  parse/apply actually lied; look-only gaps belong in the template.
  Alert if a template change is required. Use when the user says a
  template "looks wrong", "wrapped", "not aligned", "CSS not applied",
  "debug this HTML/PDF", or runs /debug-html-template. Do not use for
  API, CLI, or non-visual converter failures.
---

# Debug an HTML template

Visual PDF bug. **Diagnose, then stop.** Do not implement until the user
picks a row from the solutions table.

The template CSS is valid until a dump proves otherwise. Do not start
by rewriting the HTML. Engine work is for a lied measure/parse/apply
path — not to invent spacing a browser would not draw.

## 1. Pin the fragment

- **Open the HTML** (the `.html` file, plus any linked `.css`). Read the
  markup and the `<style>` / stylesheet that apply to the fragment. Do
  not guess structure from the PDF alone.
- **Open the PDF** on the page the user named, then the pages around it.
- Prefer **Python** to get coordinates, not just a screenshot:
  `pdfplumber` (`page.chars`, `extract_words`, `extract_text`) or
  `pypdf` for page count/boxes. Record each token's `x0`, `x1`, `top`,
  `bottom`. Wrap = same phrase, different `top`. Align = box/`rect`
  vs word baseline. (PDF `y` is often bottom-up; say which origin
  you used.)
- Collect **only** the CSS that applies to that fragment: element, parents,
  shared groups (`display: flex` on a comma list), UA-ish defaults.
- Quote the used values: `display`, `width`/`height`, `margin`, `gap`,
  `flex*`, `white-space`, `vertical-align`, `font-size`, `line-height`.

## 2. Name the symptom

Pick one primary:

| Symptom | What you must observe |
|---|---|
| wrap | a phrase splits across Y; second token X resets |
| align | boxes and text share a line but centers/baselines disagree |
| overlap / clip | two fragments share space they should not |
| missing | CSS requests paint (bg, width, pseudo) and ops never appear |

Wrap and align can both be true. Name and prove them separately.

## 3. Split the engine, not the markup

For the used properties, read **two** paths:

1. **Measure vs paint.** Intrinsic width/height (`measureCellMinMax`,
   flex base/min, shrink-to-fit) vs the box that is actually emitted
   (`collectInlineBlock`, flex used width, `build`).
   If paint honors `width`/`margin` and measure does not, the item is
   too narrow and text wraps.
2. **Parse vs apply.** Does `set*Value` store the declaration (keyword
   *and* length/%), and does emit/line-metrics read that field?
   If parse drops the value, or emit only handles keywords, alignment
   and offsets will look "almost right".

Author CSS that a browser would honor is an **engine gap**, not a
template bug.

## 4. Tight loop

Copy the fragment into the smallest `layoutHTML` + `sheet` case that
still shows the symptom. Dump display-list ops (`OpText`, `OpFillRect`):
`Text`, `X`, `Y`, `W`, `H`. Cross-check those numbers against the
Python PDF word/char boxes from step 1 when a rendered file exists.

Pass/fail on geometry, not screenshots:

- wrap: the two words share `Y` (epsilon) and the second `X` is greater
- align: box bottom vs text baseline (or box center vs text center)
  matches the CSS (`vertical-align` length, `middle`, etc.)

Run that probe once so the table is evidence-backed. Do **not** turn it
into a committed fix, and do **not** edit engine or template yet.

## 5. Report solutions — then wait

This skill ends at the table. No patches, no HTML edits, no PDF regen.

Lead with: used CSS, which path lied (measure, parse, or apply).

Then a table. Mark the recommended row using the rules below.

| Where | Change | Why | Risk |
|---|---|---|---|
| Engine | … | closes a real measure/parse/apply gap | … |
| Template (only if needed) | … | document-local look | local only |

## Recommendations

Put these in the table. Do not skip the side-effect column.

- **Engine** when a browser would honor the CSS and we do not (measure
  ignored a width, parse dropped a length, emit skipped a keyword).
  That fix should help every template.
- **Template** when the CSS is already honored and the user wants a
  *look* (gutter under a header, extra air, page-1 matching page-2).
  Write it as margin/padding/gap in that document.
- Do **not** recommend a global engine convention that invents space
  CSS does not ask for. Side effect: every other document with the
  same pattern grows, pagination can shift a row, print drifts from
  the browser.
- Do **not** treat a page-edge lead (keep ink out of the page margin)
  as the same problem as an in-flow sibling packed against the
  previous box. Keep the lead; do not copy it into normal flow.
- If you still list an invented-engine row, say the side effect in
  **Risk** and do not mark it recommended.

**Template alert.** If any row requires changing the HTML/CSS template,
say it in a standalone line the user cannot miss:

`ALERT: this needs a template modification — the engine cannot honor the current CSS as written.` / or `ALERT: a template change is the recommended fix; the engine already honors this CSS.`

If the CSS is invalid, that alert is mandatory and the template row is
the only honest fix.

Do not implement. Ask which row to take. Implement **only** after an
explicit go-ahead on a named row.

## 6. After go-ahead only

- Engine row: close that measure or parse/apply gap.
- Template row: edit only what the row named, after the alert was shown.
- Keep a layout-ops regression test. Do not assert internals.
- Re-run the layout package (or the files you touched).
- Regenerate the PDF after the test is green. Confirm the original page,
  then nearby pages that share the same CSS pattern.
