## Summary

Lands the full **phases 0–9 pure-Go, stdlib-only** HTML→PDF / HTML→image rewrite of a wkhtmltopdf work-alike (MVP v0.1.0): load → parse → CSS → layout → paginate → paint → PDF write, plus CLI, library API, golden corpus, CI, MIT license, and last-mile fixes so generated PDFs open correctly in real viewers.

Built **from scratch** with **zero third-party Go modules or HTML→PDF APIs** (no Chrome/WebKit/Qt, no cgo). Plans and last-mile correctness via Grok 4.5; bulk phase implementation via DeepSeek; remaining Unicode/complex-page work is documented follow-up.

---

## Motivation / context

- Plans: [`plans/00-canonical-pure-go-rewrite.md`](../00-canonical-pure-go-rewrite.md), [`plans/phases/`](../phases/)
- `origin/master` only held the initial plan commit; this branch ships the engine
- Issues: see **Related issues**

---

## Changes

### Phases 0–9 (engine)

| Phase | Delivered |
|------:|-----------|
| **0** | Scope freeze, compatibility matrix, golden fixture layout, module scaffold |
| **1** | Settings model, UnitReal, dotted `Set`, wkhtmltopdf-style CLI multi-object grammar |
| **2** | Loader: HTTP(S)/file/`data:`, auth/cookies/POST, local-file ACL |
| **3** | PDF 1.4 writer: objects/xref, Flate, TTF subset, images, links, outlines |
| **4** | HTML + CSS subset, layout engine, display-list paint |
| **5** | Pagination, multi-object assembly, copies/collate |
| **6** | Headers/footers, TOC, PDF outline, internal/external links |
| **7** | `gowkhtmltoimage` (PNG/JPEG) |
| **8** | Public `NewConverter` / `NewImageConverter` API + examples |
| **9** | ≥20 golden fixtures, threat model, perf gate, CI, VERSION/CHANGELOG |

### Last-mile viewer-valid PDFs

- **zlib `/FlateDecode`** — content/font streams use `compress/zlib` (RFC 1950), not raw DEFLATE
- **Catalog `/Outlines`** — outline refs finalized *before* Catalog is written (`/Outlines N 0 R`)
- **CLI page-scoped pending** — no ghost empty page when `--enable-local-file-access … toc …` runs (`make samples` showcase)
- **Glyph `/Widths`** — scale TTF advances by `1000/unitsPerEm` (fixes letter-spacing like `A c m e`)
- **Latin-1 `pdfString`** — encode per rune; fold common Unicode punctuation; subset matches string bytes

### Docs / license / tooling

- **MIT License** — Copyright (c) 2026 Chinmay Sawant (`LICENSE`)
- **README** — clean-room / no third-party APIs; AI build story (Grok plans + last-mile, DeepSeek ~90% phases); deferred list
- **`make samples`** — regenerate fixture PDFs, TOC/HF showcase, sample PNG
- **`output/` gitignored**

### Package map (short)

| Area | Role |
|------|------|
| `cmd/gowkhtmltopdf`, `cmd/gowkhtmltoimage` | CLIs |
| root `api.go` | Library API |
| `internal/{settings,cli,load,html,css,layout,pdf,outline,convert,imageout}` | Pipeline |
| `testdata/golden/` | 20 report-style fixtures |
| `docs/`, `plans/` | Matrix, threat model, phase ledgers |

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | 10-page table report ~140 ms cold; CI budget &lt; 5 s |
| **Memory** | Report-sized documents; full corpus loads in tests |
| **Behavior / correctness** | Viewer-valid Latin PDFs; deterministic bytes for fixed creation time |
| **API / CLI** | New binaries + library; wkhtmltopdf-style flags (JS-era flags accepted as no-ops) |
| **Dependencies** | **None** — Go stdlib only; `CGO_ENABLED=0` |
| **Binary size / build time** | Static binaries; offline build |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None (greenfield vs `master` plans-only) | Consumers adopt CLI or library API as new |

---

## Test plan

- [x] `make test` / `go test ./...`
- [x] `go test ./internal/pdf/ ./internal/cli/ ./internal/convert/` after last-mile fixes
- [x] `make samples` — fixtures + showcase TOC/HF/outline open in Ghostscript
- [x] URL smoke: `gowkhtmltopdf https://en.wikipedia.org/wiki/Ana_de_Armas` produces multi-page PDF (Latin OK; non-Latin still limited)
- [ ] CI green on PR (`make test`, `make lint`, `CGO_ENABLED=0` builds)

### Commands

```sh
make test
make lint
make golden
make samples
CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
CGO_ENABLED=0 go build -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage
go run ./cmd/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-01-simple-invoice.html /tmp/out.pdf
```

---

## Screenshots / sample output

```
make samples → output/fixture-01…20.pdf, showcase-toc-hf-outline.pdf, fixture-01-simple-invoice.png
After Widths fix: "Acme Widgets GmbH" renders with normal letter spacing (not "A c m e").
Wikipedia URL conversion succeeds (~24 pages); non-Latin language names still show as "?".
```

---

## Related issues

- Relates to pure-Go rewrite plans on `master` (`plans/00-canonical-pure-go-rewrite.md`)
- No numbered GitHub issues closed by this PR (greenfield MVP integration)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues filled (plans-based; no ticket IDs)
- [x] Filled body under `plans/PR/pr-pure-go-mvp.md`

---

## Follow-ups (out of scope)

- Type0/CID fonts for CJK and non-Latin scripts (Wikipedia-class Unicode)
- Real bold/italic faces (today: fake bold via text render mode 2)
- Richer CSS: floats/position, flex/grid, richer selectors
- Table header repeat, smart-shrinking re-layout, inline `#` link rects
- Coalesce word-by-word `BT`/`ET` text ops
- Full WebKit parity — **not planned**

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented (README)
- [ ] Golden fixtures cover report-style HTML
- [ ] PR has assignee and labels
- [ ] No secrets or generated `output/` artifacts committed
- [ ] Confirmed `go.mod` has zero third-party requires
