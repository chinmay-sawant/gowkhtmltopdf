## Summary

Improves the project site (showcase cards, documentation IA, landing/layout polish) and expands the golden corpus with print-story fixtures 49–53—posters, letter stationery, Asteria storybook, airline boarding pass, and observatory poster—using CSS generic fonts only (no third-party font files under `testdata/golden/assets`).

---

## Motivation / context

- Showcase UX was hard to browse: cards needed clearer affordances, template source links, and a coherent docs area.
- Documentation was split across flat routes; consolidating under `/documentation/:docId` with a sidebar matches how users discover CLI/library/security material.
- Print-oriented samples (poster, letter, storybook, boarding pass) exercise nested assets, multi-page story layout, and transactional document chrome without redistributing Lato/Ubuntu/extra Liberation files under the golden assets tree (license clarity for MIT project packaging of fixtures).
- Issue drafts under `plans/PR/issues/` capture PDF version / UA compliance and original-template rendering follow-ups.

---

## Changes

### Site / frontend

- Showcase cards open the viewer from the full card; template source links and Open PDF / View template styling.
- New landing page polish; sticky documentation page with sidebar nav (`DocumentationPage.jsx`).
- Routes: `/documentation/:docId` plus legacy redirects for former top-level doc paths.
- Scroll-to-top on client route change; footer casing; typography and callout card polish.
- Rebuilt static docs bundle (`docs/assets`, `docs/index.html`) for GitHub Pages.

### Golden fixtures 49–53

| Fixture | Role |
|---------|------|
| `fixture-49-night-train-poster.html` | One-page poster, linked theme + image |
| `fixture-50-letter-template.html` | Stationery letter + mark + quote rule |
| `fixture-51-asteria-storybook.html` | Four-page original story with illustrations |
| `fixture-52-airline-boarding-pass.html` | E-ticket itinerary + four boarding stubs (document-style, no bundled art fonts) |
| `fixture-53-asteria-observatory-poster.html` | Second poster variant |

- Shared `theme-print-stories.css` uses only `sans-serif` / `serif` / `monospace` (engine free defaults).
- Artwork under `testdata/golden/assets/` (Asteria images, letter SVG); no `assets/fonts/` tree.
- `copyGoldenTree` in `internal/convert/golden_test.go` preserves nested assets when isolating fixtures; page envelopes for 49–53.
- Committed sample PDFs: `output/fixture-49-*.pdf` … `output/fixture-53-*.pdf` (regenerated from current HTML).

### Plans / issue drafts

- `plans/PR/issues/issue-original-template-rendering-body.md`
- `plans/PR/issues/issue-epic-newer-pdf-versions-compliance-body.md`
- `plans/PR/issues/issue-pdf-17-20-compliance-body.md`
- `plans/PR/issues/issue-pdf-20-support-body.md`
- `plans/PR/issues/issue-pdf-ua2-pdfa4-compliance-body.md`

### Dossier / matrix touch-ups

- Selected issue reclassifications (e.g. rowspan / CJK / nested-table notes) reflected in frontend issue data where this branch already updated them.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None material for convert path; site assets only |
| **Memory** | None material |
| **Behavior / correctness** | New golden fixtures + nested asset copy in tests; page envelopes pinned |
| **API / CLI** | None |
| **Dependencies** | None (no new font packages) |
| **Binary size / build time** | None for converter binary; repo grows by fixture art + sample PDFs |
| **Licensing** | Golden assets avoid redistributed third-party font files; engine still embeds built-in defaults as before |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Docs URLs on the site | Old paths redirect to `/documentation/<id>` |
| None for CLI/library users | - |

---

## Test plan

- [x] `go test ./internal/convert/ -run 'TestGoldenCorpusAllFixtures/fixture-5' -count=1` (fixtures 49–53 pass)
- [x] Regenerated `output/fixture-49` … `fixture-53` PDFs with `--enable-local-file-access`
- [ ] Full `make test` / `make golden` on CI or local
- [ ] `make lint` / `go vet` as available
- [ ] Spot-check site: showcase cards, documentation sidebar, route redirects, scroll-to-top
- [ ] Confirm no `testdata/golden/assets/fonts/` in tree

### Commands

```sh
go test ./internal/convert/ -run 'TestGoldenCorpusAllFixtures' -count=1
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-52-airline-boarding-pass.html \
  /tmp/boarding.pdf
# frontend (optional)
cd frontend && npm run build
```

---

## Screenshots / sample output

Sample PDFs (committed):

- `output/fixture-49-night-train-poster.pdf`
- `output/fixture-50-letter-template.pdf`
- `output/fixture-51-asteria-storybook.pdf`
- `output/fixture-52-airline-boarding-pass.pdf`
- `output/fixture-53-asteria-observatory-poster.pdf`

HTML sources: `testdata/golden/fixture-49-*.html` … `fixture-53-*.html`.

---

## Related issues

- Relates to rendering / showcase quality work on this branch (no single Closes ticket yet for fixtures 49–53)
- Refs draft bodies under `plans/PR/issues/` for PDF 1.7/2.0, PDF/UA-2 & PDF/A-4, and original-template rendering follow-ups (file as GitHub issues when ready)

---

## PR metadata checklist (author)

- [ ] Self-assigned (`--assignee @me`)
- [ ] Labels applied (`enhancement`, `documentation` as appropriate)
- [x] Related issues section filled (drafts referenced; link numbers when filed)
- [x] Filled body under `plans/PR/pr-showcase-improvements.md`

---

## Follow-ups (out of scope)

- File GitHub issues from `plans/PR/issues/issue-pdf-*.md` and `issue-original-template-rendering-body.md`
- Add fixtures 49–53 to the frontend showcase gallery cards if desired
- Align `internal/pdf/assets` Liberation license comment with actual font license text (separate hygiene)
- Dedicated night-train artwork if poster-49 should stop sharing observatory art

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes beyond site + golden print corpus + issue drafts
- [ ] Public API / CLI unchanged
- [ ] New fixtures have headers naming the fixture and page envelopes in `golden_test.go`
- [ ] PR has assignee and labels
- [ ] No secrets; sample PDFs/art are intentional committed assets
- [ ] No third-party font binaries under `testdata/golden/assets`
