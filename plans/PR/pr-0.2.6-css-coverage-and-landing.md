## Summary

Ships the v0.2.6 catalog-driven print CSS program (phases 48-85) through the Go engine: 356 Implemented / 0 Partial / 462 Unsupported across 818 webref properties, with layout/paint depth, golden fixtures 57-62, and correctness fixes for table breaks and Type0 fonts. Also refreshes the docs-site landing into a minimal HTML-to-PDF story. 40 commits, ~265 files on `feature/06-high-fun-extended-CSS-support-Fixing-issues`.

## Motivation / context

- Plans: `plans/0.2.6/` (canonical ledger `48-canonical-0.2.6-css-coverage.md`, phases 48-85, catalog honesty gates).
- The old landing buried the product purpose behind sandbox controls and mixed proof grids. That rewrite is a late slice (`09d5399`); most of the branch is engine work, not frontend.
- Catalog end state is recorded in `plans/0.2.6/catalog/coverage-summary.json` (356 / 0 / 462 / 0). `VERSION` stays `0.2.5`; `CHANGELOG.md` already has a 0.2.6 section.
- Issues: no single GitHub ticket for the whole tranche. See **Related issues**.

## Changes

### Engine / CSS coverage (primary)

- Close v0.2.6 catalog-driven CSS coverage through phases 48-85: map 818 webref properties, promote Partials, triage Unsupported honestly (`plans/0.2.6/`, `scripts/css-catalog-map.py --check`).
- Selectors and cascade: `:is()` / `:where()`, of-type nth, attribute `i` flag, `@import` under FetchSub ACL (`internal/css`, `internal/convert/prepare`).
- Values and box: `clamp()`, `hsl()` / `hsla()`, logical margin/padding/inset/size/borders, `vmin` / `vmax`, elliptical `border-radius`.
- Paint: multi-layer `background-image` + linear/radial gradients, outline, multi-layer `box-shadow` (incl. inset), `border-image`, overflow clip, image filters, text-shadow and decoration longhands (`internal/layout`).
- Generated content / lists: counters, quotes, `list-style-position: inside`, `list-style-image`.
- Paged media: `@page :first` / `:left` / `:right` and named pages; lite `@top-*` / `@bottom-*` into empty CLI header/footer slots; `page: ident` sibling breaks (`page_margin*.go`, `hf_geometry.go`).
- Layout deepen: flex/place/grid shorthands, `repeat(auto-fit/auto-fill)`, `inline-grid`, column-rule, vertical writing-mode mapping, individual 2D transforms, live `-webkit-*` remaps when the unprefixed base exists.
- Correctness: table page-break seals and cell clip on fixture-60; Type0 cmap skip when the face has no glyph for a non-Latin rune (`c55014a`); grid/flex replaced-image paint; default text header/footer size 8.5 pt when unset.
- Refactor / lint: split oversized layout/css modules, pass `ResolvedStyle` by pointer, sentinel errors, `make lint` cleanup on layout/pdf (`0bc2103`, `a6c6644`, `21e9aee`).
- Tooling: `make test` defaults to `-p 2 -parallel 2`; adds `test-unit` / `test-quick` / `test-serial` / `test-race`; CI race uses `make test-race`.

### Golden fixtures and docs

- Fixtures 57-62: Implemented probe, Unsupported safe degrade, Apex Digital landing, three Implemented property audits; envelopes in `internal/convert/golden_test.go`; corpus size 62.
- Sync `documentation/compatibility-matrix.md`, `documentation/deferred.md`, and `CHANGELOG.md` 0.2.6 notes with catalog honesty.

### Frontend / docs site (secondary)

- Unify `/` around one HTML-to-PDF story: rewrite `LandingPage.jsx` and `landing.css` into hero + pipeline + fit + proof + samples + close.
- Drop the interactive terminal sandbox and multi-control CLI builder; keep a static CLI/Go tabbed example with copy.
- Rebuild `docs/` from Vite so hashed JS/CSS and `index.html` match CI's dirty-docs gate.
- No route, library, or CLI contract changes from the landing slice. Class rename `landing-page` -> `landing-minimal` is internal CSS only.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Pointer `ResolvedStyle` avoids large style copies; pseudo `content` fast-path for static strings; PDF writer buffering and flate pool cap. Multi-layer backgrounds, shadows, filters, and gradients cost more paint when templates use them. Test parallelism capped to reduce swap thrash. |
| **Memory** | Style pointer path reduces layout copy pressure. Landing uses one `useState` instead of five plus two `useMemo`s. |
| **Behavior / correctness** | Large print-CSS expansion (356 Implemented). Templates using new selectors, logicals, backgrounds, shadows, and page rules render closer to author intent. Fixture-56 locked at 21 pages. Default HF text size 8.5 pt. Table continuation and Type0 missing-glyph paths fixed. Unsupported props stay no-ops without crash (fixture-58). |
| **API / CLI** | No new Document/CLI flags for the CSS program. Bindings constructors prefer `Content{HTML: …}` (existing `HTML()` helper still works). `VERSION` remains 0.2.5. |
| **Dependencies** | No new direct Go modules. Allowlist still `go-text/typesetting` and `tdewolff/canvas`. |
| **Binary size / build time** | Engine binary grows with new layout/css code paths. Frontend Vite build still ~1.6s; `docs/` must stay rebuilt. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Default text header/footer font size 12 pt -> 8.5 pt when unset | Pass an explicit HF font size if you relied on 12 pt. |
| Broader CSS application on existing HTML | Templates that previously ignored logicals, backgrounds, shadows, `:is`, or `@import` may layout differently. Re-check golden or sample PDFs. |
| `VERSION` still 0.2.5 | Binary stamp is not bumped in this PR even though CHANGELOG has a 0.2.6 section. Tag/stamp when cutting the release. |

Landing class rename is internal frontend only. `/` still serves `LandingPage`.

## Test plan

- [x] Engine golden corpus including fixtures 57-62 (`make golden` / `fixturePageBounds` envelopes)
- [x] Catalog honesty: 356 Implemented / 0 Partial / 462 Unsupported (`coverage-summary.json`, `css-catalog-map.py --check`)
- [x] Layout/css package tests for backgrounds, shadows, selectors, page rules, fixture-60 spill
- [x] Type0 cmap guard (`internal/pdf` / fixture-61 path)
- [x] `npm --prefix frontend run lint` and `npm --prefix frontend run build` with `docs/` copy
- [x] `make claim-scan`
- [x] `make lint` cleanup commit `21e9aee`
- [ ] Re-run full gate on `master` after merge if CI did not already: `make test`, `make golden`, `make lint`, `make claim-scan`, `CGO_ENABLED=0 go build`

### Commands

```sh
make test
make golden
make lint
make claim-scan
CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
npm --prefix frontend run lint
npm --prefix frontend run build
```

## Screenshots / sample output

```
Catalog (plans/0.2.6/catalog/coverage-summary.json):
  implemented: 356
  partial: 0
  unsupported: 462
  ignored: 0
  webref_properties: 818

Golden: fixtures 57-62 added (corpus size 62)

Landing (09d5399):
  Hero: "Your HTML, as a print-ready PDF."
  Static CLI: gowkhtmltopdf input.html output.pdf
  docs/ assets rotated (index-BUuSB4SA.js, index-25J-iRC3.css)
```

## Related issues

- Relates to `plans/0.2.6/` CSS coverage track on branch `feature/06-high-fun-extended-CSS-support-Fixing-issues`.
- Landing copy feedback had no GitHub issue; shipped as UX polish on the same integration branch.
- No `Closes` keyword: avoid auto-closing unrelated tickets.

## PR metadata checklist (author)

- [x] Self-assigned (`chinmay-sawant`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues filled
- [x] Body kept under `plans/PR/pr-0.2.6-css-coverage-and-landing.md` (replaces frontend-only draft)

## Follow-ups (out of scope)

- Bump `VERSION` and cut the 0.2.6 release tag so binary stamp matches CHANGELOG.
- Prefer stacked PRs next time when an engine tranche and a landing rewrite land on one branch.
- Optional landing smoke test for `PageTitle` and `/` route.

## Reviewer checklist

- [ ] Summary matches both engine and landing scope
- [ ] Catalog counts and golden fixtures 57-62 reviewed
- [ ] Behavior notes (HF 8.5 pt, broader CSS apply) accepted
- [ ] Public API / CLI changes documented
- [ ] PR has assignee and labels
- [ ] No secrets committed; `docs/` assets are required build output
