## Summary

The showcase page reads its sample list from `frontend/src/data/showcase.js`. That list stopped at fixture 56, so fixtures 57-64 never appeared even though `make screenshots` had already written their page images. This adds those fixtures, plus the nested-float sample `fixture-29-wpt`, and rebuilds the published site bundle.

---

## Motivation / context

- Plans: none
- Issues: see **Related issues**

The golden corpus now runs through `fixture-64-next-72-props.html`. The gallery catalog still ended at `fixture-56-architecture-diagram`. Fixtures 01-04 were already in the catalog. They sit at the bottom of the list.

---

## Changes

### Showcase catalog

- Added fixtures 57, 58, 59, 60, 61, 62, 63, and 64 to `frontend/src/data/showcase.js`, in that descending order, under CSS & layout fixtures.
- Added `fixture-29-wpt-break-nested-float-print`. Its three page images were already generated and it was the only other page-1 screenshot missing from the catalog.
- Page counts match the PNG files in `frontend/src/assets/showcase/`: 9, 9, 9, 8, 8, 8, 7, and 8 for fixtures 57-64, and 3 for fixture 29-wpt.
- `frontend/README.md` now says 70 samples (67 golden outputs plus 3 specials). It previously said 61.

### Published site

- Rebuilt `docs/` with `npm run build`. The page images were already in `docs/assets`. The entry script and the chunks that import it picked up new filenames because the catalog strings live in the main bundle.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No engine change. The showcase JS bundle grows by the nine new catalog rows. |
| **Memory** | None. |
| **Behavior / correctness** | The showcase, its category filters, and the command palette now include fixtures 57-64 and fixture 29-wpt. |
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

- [x] Catalog check: every showcase row has a PNG and a WebP for each listed page, and every page-1 PNG is in the catalog (70 items, 0 gaps)
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

- 70 cards, meta text `70 of 70 samples`
- Fixture 64 thumbnail loaded (400x566) and the modal listed pages 1-8
- CSS & layout filter showed 39 cards, including Next 72 CSS audit
- Invoices & receipts filter showed 10 cards, including Simple invoice, and did not include Next 72
- Ctrl+K search for `Next 72` hit the new card
- A 390px-wide viewport still listed 70 cards

---

## Screenshots / sample output

No new images. The cards use the PNGs and WebP thumbs already in `frontend/src/assets/showcase/`.

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
| `.html` | 1 | 1 | 1 |
| `.js` | 7 | 94 | 22 |
| `.md` | 2 | 127 | 1 |
| **Total** | **10** | **222** | **24** |
