# Implemented props fixture authoring (60/61/62)

## Goal
Three golden HTML fixtures, each documenting ~118-120 Implemented CSS properties with:
- property name
- plain-language description of the expected effect
- a visible Effect cell that applies that property

## Fonts (required)
- `font-family: "Liberation Sans", "Liberation Serif", "Liberation Mono", sans-serif;`
- PDFs generated with: `--font-path testdata/fonts/implemented-audit`
- License: SIL OFL 1.1 in `testdata/fonts/implemented-audit/LICENSE.txt`

## Images (required, local only for golden reliability)
- `logo.png` (relative)
- `assets/asteria-lake.png` (relative)
- plus small data-URI PNG/SVG if needed
- Do not depend on live internet URLs (golden/CI has no network guarantee)

## File header (mandatory)
```html
<!DOCTYPE html>
<!--
  fixture-6N-implemented-props-x
  Proves: visual audit of Implemented CSS properties (slice X of 3), each with
  description + effect demo. Fonts: Liberation via --font-path testdata/fonts/implemented-audit.
  Expected: multi-page; envelope in fixturePageBounds.
-->
```

## Layout pattern
Use a print A4 stylesheet. For each property, one row/card:

| # | Property | What it should do | Effect (live CSS) |

Effect cell must actually set the property (inline style or a dedicated class named after the property). Alignment: consistent 4-column table, zebra rows, clear borders.

## Needles to embed as visible text
- Fixture A: `IMPLEMENTED-PROPS-A` and `Liberation Sans`
- Fixture B: `IMPLEMENTED-PROPS-B` and `Liberation Sans`
- Fixture C: `IMPLEMENTED-PROPS-C` and `Liberation Sans`

## Property list source
`plans/0.2.6/catalog/implemented-fixture-split.json` → fixtures.A/B/C.properties

## Output paths
- `testdata/golden/fixture-60-implemented-props-a.html`
- `testdata/golden/fixture-61-implemented-props-b.html`
- `testdata/golden/fixture-62-implemented-props-c.html`
