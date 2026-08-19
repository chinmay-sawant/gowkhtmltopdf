## Summary

Ship **v0.2.4**: replace the wkhtml-shaped public library (`Converter`, dotted `Set`/`Get`, typed request wrappers) with a Go-native `Document` / `ImageDocument` model, redesign both CLIs to the same grammar, and freeze the three-engine external compare harness (wkhtmltopdf, WeasyPrint, Puppeteer) with refreshed benchmark artifacts and site docs.

This is an intentional pre-1.0 hard break. Engine layout and PDF writers stay; the outer contract and bench paths change. Migration guide: [`documentation/MIGRATION-0.2.4.md`](documentation/MIGRATION-0.2.4.md).

---

## Motivation / context

- Plans: [`plans/0.2.4/README.md`](plans/0.2.4/README.md), [`plans/0.2.4/31-canonical-0.2.4-roadmap.md`](plans/0.2.4/31-canonical-0.2.4-roadmap.md) (phases 31–39)
- Predecessor release: v0.2.3 module path / `go install` (#51, #52)
- Product docs: [`documentation/library-api.md`](documentation/library-api.md), [`documentation/cli.md`](documentation/cli.md), [`testdata/golden/benchmarks/README.md`](testdata/golden/benchmarks/README.md)
- Issues: see **Related issues**

---

## Changes

### Document API (phases 31–35)

- Add root-package `Document`, `ImageDocument`, `Content`, `Page`, options structs, validation, and writer-first `WritePDF` / `WriteImage` (plus byte-returning helpers)
- Delete the public wkhtml-shaped surface (`Converter`, dotted settings, `PDFRequest` / `ImageRequest` wrappers) with no `compat` refuge
- Keep engine adapters on existing `internal/convert` / `internal/imageout` paths

### CLI redesign (phase 36)

- Redesign `gowkhtmltopdf` / `gowkhtmltoimage` around `-o/--output`, explicit `--html` / `--url`, positional page files, `--cover` / `--toc`, and `--allow-local-files`
- Drop the old `page` / `cover` / `toc` object grammar
- Stamp `VERSION` / `cli.Version` as **0.2.4**

### External benchmarks (phase 39)

- Unify WeasyPrint / Puppeteer process compares under `scripts/bench-external.sh` + `make bench`
- Keep `make bench-cli-compare` for wkhtmltopdf; add `make bench-inprocess` and `make bench-lib` for allocation matrices
- Publish golden artifacts under `testdata/golden/benchmarks/` and document reproduce lines
- Add PDF generation pprof notes in [`plans/0.2.4/pdf-generation-pprof.md`](plans/0.2.4/pdf-generation-pprof.md)

### Layout correctness

- Bound inline highlight backgrounds so paint does not spill past glyph runs
- Trim sticky continuation chrome across page breaks
- Refresh related golden / fixture outputs

### Docs, examples, and site

- Migration guide, library API, CLI, getting started, architecture, examples
- Frontend / GitHub Pages refresh for Document API copy, benchmarks presentation, and showcase screenshots
- Dated `CHANGELOG.md` **0.2.4 (2026-08-18)** section

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Process benches remain faster than wkhtmltopdf / WeasyPrint / Puppeteer on the shared report fixture; public `Document.WritePDF` path profiled for large reports |
| **Memory** | RSS recorded in golden compare tables; no intentional RSS regression claimed beyond fixture refresh |
| **Behavior / correctness** | Sticky chrome and inline highlight paint fixes; Document validation rejects invalid content sources at the public boundary |
| **API / CLI** | Hard break: embedders and CLI users must migrate (see table below) |
| **Dependencies** | Optional host tools for benches only (wkhtmltopdf, WeasyPrint, Puppeteer); not Go module deps. Puppeteer harness under `scripts/puppeteer/` |
| **Binary size / build time** | Unchanged constraint: pure Go, no CGO |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| `Converter` / `ImageConverter` | `Document` / `ImageDocument` |
| Dotted `Set` / `Get` | Named struct fields on `Document` / `Page` / options |
| `PDFRequest` / `RunPDF` | `Document.WritePDF` / `Document.PDF` |
| `ImageRequest` / `RunImage` | `ImageDocument.WriteImage` / `ImageDocument.Image` |
| CLI `page` / `cover` / `toc` object grammar | `-o`, `--html`/`--url`, positional pages, `--cover`/`--toc` |
| `go install …@v0.2.3` | Use `@v0.2.4` after tag |

Full guide: [`documentation/MIGRATION-0.2.4.md`](documentation/MIGRATION-0.2.4.md).

---

## Test plan

- [ ] `make test`
- [ ] `make lint` / `go vet ./...`
- [ ] `make build` and smoke `./bin/gowkhtmltopdf --help` / `./bin/gowkhtmltoimage --help`
- [ ] Examples under `examples/pdf` and `examples/image` build against the new API
- [ ] Optional: `make bench-cli-compare` when wkhtmltopdf is installed
- [ ] Optional: `make bench` when WeasyPrint / Puppeteer are installed (missing engines skip with evidence)
- [ ] After merge (not this PR): tag `v0.2.4` and publish the GitHub Release

### Commands

```sh
make test
make lint
make build
./bin/gowkhtmltopdf --version
test "$(tr -d '[:space:]' < VERSION)" = "0.2.4"
```

---

## Screenshots / sample output

```
VERSION=0.2.4
CHANGELOG: ## 0.2.4 (2026-08-18)

# CLI (target grammar)
gowkhtmltopdf -o report.pdf --allow-local-files report.html
gowkhtmltopdf -o out.pdf --html '<h1>Hi</h1>'
gowkhtmltoimage -o page.png --allow-local-files page.html

# Library (sketch)
doc := gowkhtmltopdf.Document{
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{File: "report.html"},
    }},
    AllowLocalFiles: true,
}
_ = doc.WritePDF(ctx, out)
```

Benchmark tables live in `testdata/golden/benchmarks/{cli,weasyprint,puppeteer}-compare.md` and on the Benchmarks page of the docs site.

---

## Related issues

- Relates to #51 (GitHub module path / `go install`)
- Relates to #52 (v0.2.3 release prep)
- No dedicated tracking issue was filed for the 0.2.4 Document API ledger

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/0.2.4/PR/pr-0.2.4-document-api-refactor.md`

---

## Follow-ups (out of scope)

- Tag `v0.2.4` and publish the GitHub Release **after** this PR merges
- Further layout / CSS fidelity work
- Making WeasyPrint / Puppeteer hard CI deps on every PR

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in the diff
- [ ] Public API / CLI changes documented (`MIGRATION-0.2.4.md`, library-api, cli)
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
