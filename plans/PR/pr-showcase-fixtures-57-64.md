## Summary

The showcase page reads its sample list from `frontend/src/data/showcase.js`. That list stopped at fixture 56. This adds fixtures 59-64 and the nested-float sample `fixture-29-wpt`. Fixtures 57 and 58 stay as PDFs under `output/` and are not shown on the site. Fixture 64's heading names the property range, `font-feature-settings` through `counter-set`.

---

## Motivation / context

- Plans: none
- Issues: see **Related issues**

The golden corpus now runs through `fixture-64-next-72-props.html`. The gallery catalog still ended at `fixture-56-architecture-diagram`. Fixtures 01-04 were already in the catalog. They sit at the bottom of the list.

---

## Changes

### Showcase catalog

- Added fixtures 59, 60, 61, 62, 63, and 64 to `frontend/src/data/showcase.js`, under CSS & layout fixtures. Fixtures 57 and 58 are omitted. Their PDFs remain in `output/`.
- Fixture 64's page title and catalog card say `font-feature-settings to counter-set`. The HTML, `output/fixture-64-next-72-props.pdf`, and the page screenshots were regenerated with `bin/gowkhtmltopdf --font-path testdata/fonts/implemented-audit`. The golden needle stays `NEXT-72-PROPS`.
- Added `fixture-29-wpt-break-nested-float-print`. Its three page images were already generated and it was the only other page-1 screenshot missing from the catalog.
- Page counts match the PNG files in `frontend/src/assets/showcase/`: 9, 8, 8, 8, 7, and 8 for fixtures 59-64, and 3 for fixture 29-wpt.
- `frontend/README.md` now says 68 samples (65 golden outputs plus 3 specials). It previously said 61.

### Published site

- Rebuilt `docs/` with `npm run build`. The page images were already in `docs/assets`. The entry script and the chunks that import it picked up new filenames because the catalog strings live in the main bundle.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No engine change. The showcase list gains fixtures 59-64 and fixture 29-wpt, and leaves out 57 and 58. |
| **Memory** | None. |
| **Behavior / correctness** | The showcase, its category filters, and the command palette include fixtures 59-64 and fixture 29-wpt. Fixture 64 is titled by its property range. |
| **API / CLI** | None. |
| **Dependencies** | None. |
| **Binary size / build time** | None for the Go binaries. The site rebuild rewrote six hashed JS filenames. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] Catalog check: every showcase row has a PNG and a WebP for each listed page, and every page-1 PNG is in the catalog (68 items, 0 gaps)
- [x] `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-64' -count=1`
- [x] `node frontend/scripts/lint-data.mjs`
- [x] `npx eslint src/data/showcase.js` from `frontend/`
- [x] `npm run build && npm test` in `frontend/`
- [ ] `make test` (no Go code in this change)
- [ ] `make lint` (frontend eslint on the edited file passed; full golangci-lint not run)

### Commands

```sh
node frontend/scripts/lint-data.mjs
cd frontend && npx eslint src/data/showcase.js && npm run build && npm test
```

Chrome headless against `vite preview` at `#/showcase`:

- 68 cards
- No card for fixture 57 or fixture 58
- Fixture 64 card title is `font-feature-settings to counter-set` and its thumbnail loaded (400 px wide)

---

## Screenshots / sample output

Fixture 64 page images were regenerated from the new PDF. Fixtures 57 and 58 no longer have site images. Their PDFs in `output/` are unchanged.

---

## Related issues

- No tracking issue

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`bug`)
- [ ] Related issues filled with real ticket IDs (no tracking issue)
- [x] Filled body committed under `plans/PR/pr-showcase-fixtures-57-64.md`

---

## Follow-ups (out of scope)

- `documentation/samples.md` still has no separate inventory row for fixtures 59 and 63.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.html` | 2 | 8 | 8 |
| `.js` | 8 | 72 | 16 |
| `.md` | 2 | 129 | 1 |
| `.pdf` | 1 | Binary | Binary |
| `.png` | 41 | Binary | Binary |
| `.webp` | 42 | Binary | Binary |
| **Total** | **96** | **209** | **25** |
