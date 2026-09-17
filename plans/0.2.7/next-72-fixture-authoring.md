# Next 72 props fixture authoring (v0.2.7)

## Goal
One golden HTML fixture documenting the next 72 CSS properties targeted after
the v0.2.6 Implemented set (354). Same pattern as fixtures 60/61/62:
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
- plus small data-URI PNG in the masthead (same pixel as fixture-60/61/62)
- Do not depend on live internet URLs

## File header (mandatory)
```html
<!DOCTYPE html>
<!--
  fixture-64-next-72-props
  Proves: visual audit of the next 72 CSS properties targeted for v0.2.7,
  each with description + effect demo. Fonts: Liberation via --font-path
  testdata/fonts/implemented-audit.
  Expected: multi-page; envelope in fixturePageBounds.
-->
```

## Layout pattern
Print A4 stylesheet copied from fixture-60/61/62 (blue masthead `#1f4b99`,
audit table, zebra rows). Four columns:

| # | Property | What it should do | Effect (live CSS) |

Meta line under the property name uses `kind · group · Chrome yes/no (detail)`:
- `kind · group` from the webref mapping (same as fixture-60/61/62)
- Chrome support from MDN `browser-compat-data` (`css/properties`), frozen into
  `next-72-properties.json` as `chrome_support` / `chrome_label` / `chrome_detail`
- Effect cells for `chrome_support: no` also show a red "Chrome: no render expected" badge

Descriptions use the `With \`prop: value\` ...` form. Chrome-unsupported rows add a
sentence that Chrome will not paint a visible change.

## Needles
- `NEXT-72-PROPS`
- `Liberation Sans`

## Property list source
`plans/0.2.7/next-72-properties.json`

## Output path
- `testdata/golden/fixture-64-next-72-props.html`
