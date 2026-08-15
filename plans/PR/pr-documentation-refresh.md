## Summary

Refresh documentation, the GitHub Pages site, and sample PDFs so they match the shipped 0.2.1 / unreleased 0.2.2 split. Guides no longer claim PDF 1.7 / 2.0 or PDF/A+UA on the tagged 0.2.1 release. The README gets a centered transparent gopher mark, the site nav uses the same brand, and `output/` is regenerated including the 1.7 / 2.0 and compliance sample trees.

---

## Motivation / context

- Plans: `plans/0.2.2/` (docs follow-through after [#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45), [#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46), [#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47))
- Issues: see **Related issues**

After the 0.2.2 engine work landed, the README, architecture guides, landscape comparison, issue dossier, and sample PDFs were still written as if those profiles were already in 0.2.1. This branch is the documentation and site correction.

---

## Changes

### Docs and changelog honesty

- Label PDF 1.7 / 2.0 and PDF/A + PDF/UA as **unreleased 0.2.2** on `master`; tagged **0.2.1** stays PDF 1.4 only
- Sync `documentation/` (overview, CLI, library API, architecture, fidelity, samples, landscape-2026)
- Re-audit the issue-dossier verdicts against 0.2.1 vs 0.2.2
- Changelog and getting-started notes match the same split

### Site

- Cut webfonts and tighten accessibility
- Fix Getting Started hero spacing, benchmark heading spacing, and docs TOC hash-route hijack
- Brand the nav with the Go gopher
- Rebuild `docs/` so generated Pages assets match current frontend source
- Keep `logo.png` / `gopher.gif` / `gopher.png` in `frontend/public/` so `npm run build` does not wipe the README mark

### README brand

- Centered transparent gopher above `gowkhtmltopdf`
- Cropped pad, larger display size, no cyan halo on GitHub light or dark

### Samples

- `make samples` now emits 1.7 / 2.0 and compliance PDF trees
- Regenerated `output/` fixture PDFs and README

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | None in the converter. Docs and sample artifacts only |
| **API / CLI** | Docs only; no flag or API change |
| **Dependencies** | None |
| **Binary size / build time** | None |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [ ] `make test`
- [ ] `make lint` / `go vet`
- [ ] `CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf`
- [ ] `CGO_ENABLED=0 go build -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage`
- [ ] Frontend: `cd frontend && npm ci && npm run build` leaves `docs/` and `frontend/dist` clean
- [ ] README logo renders centered, transparent, on GitHub light and dark

### Commands

```sh
make test
make lint
cd frontend && npm ci && npm run build
git status --porcelain -- docs frontend/dist
```

CI jobs: `test + lint`, `static build (CGO_ENABLED=0)`, `race (hot packages)`, `frontend production build`.

---

## Screenshots / sample output

README header: centered `docs/logo.png` above `gowkhtmltopdf`.

---

## Related issues

- Relates to #29 (docs follow-through after the PDF version / compliance epic)
- Relates to #31, #32, #33 (guides now state what is tagged 0.2.1 vs unreleased 0.2.2)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-documentation-refresh.md`

---

## Follow-ups (out of scope)

- Site nav still uses the animated gopher; README uses the static transparent mark
- No engine or profile behavior changes

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated converter / API changes in the diff
- [ ] Public API / CLI docs match tagged 0.2.1 vs unreleased 0.2.2
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed
