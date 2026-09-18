# Next 100 props fixture authoring (v0.2.7)

## Goal

One golden HTML fixture documenting the next 100 CSS properties targeted after
the next-72 set. Same pattern as fixtures 60/61/62/64:

- property name
- plain-language description of the expected effect
- a visible Effect cell that applies that property

This commit lands the authoring contract and the JSON. The HTML itself is
`testdata/golden/fixture-66-next-100-props.html` and should be added with the
first 88.x implementation batch (plus a `fixturePageBounds` row).

## Fonts (required)

- `font-family: "Liberation Sans", "Liberation Serif", "Liberation Mono", sans-serif;`
- PDFs generated with: `--font-path testdata/fonts/implemented-audit`
- License: SIL OFL 1.1 in `testdata/fonts/implemented-audit/LICENSE.txt`

## Images (required, local only for golden reliability)

- `logo.png` (relative)
- `assets/asteria-lake.png` (relative)
- plus small data-URI PNG in the masthead (same pixel as fixture-60/61/62/64)
- Do not depend on live internet URLs

## File header (mandatory)

```html
<!DOCTYPE html>
<!--
  fixture-66-next-100-props
  Proves: visual audit of the next 100 CSS properties targeted after next-72,
  each with description + effect demo. Fonts: Liberation via --font-path
  testdata/fonts/implemented-audit.
  Expected: multi-page; envelope in fixturePageBounds.
-->
```

## Layout pattern

Print A4 stylesheet copied from fixture-64 (blue masthead `#1f4b99`,
audit table, zebra rows). Four columns:

| # | Property | What it should do | Effect (live CSS) |

Meta line under the property name uses `kind · group · Chrome yes/no (detail)`:

- `kind · group` from the webref mapping
- Chrome support from MDN `browser-compat-data`, frozen into
  `next-100-properties.json` as `chrome_support` / `chrome_label` / `chrome_detail`
- Effect cells for `chrome_support: no` also show a red "Chrome: no render expected" badge
- Effect cells for `honest_defer: true` show a gowk Unsupported panel, not a fake live demo

Descriptions use the `With \`prop: value\` ...` form from the JSON.

## Needles

- `NEXT-100-PROPS`
- `Liberation Sans`

## Property list source

`plans/0.2.7/next-100-properties.json`

Do not copy any name from `next-72-properties.json`.

## Output path

- `testdata/golden/fixture-66-next-100-props.html`

## Implementation ledger

Phase-wise batches live in `88-canonical-0.2.7-next-100.md` and `phases/phase-88.*.md`.
Full `make test` / `make golden` only in `phases/phase-88.9-closure-integration.md`.
