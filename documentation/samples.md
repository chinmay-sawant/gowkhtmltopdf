# Samples and golden fixtures

## Golden HTML (`testdata/golden/`)

Twenty report-style HTML fixtures (`fixture-01` … `fixture-20`) exercise
invoices, tables, page breaks, CSS, links, images, lists, typography, and
more. See [`testdata/golden/README.md`](../testdata/golden/README.md).

CI / local structure tests:

```sh
make golden
# equivalent: go test ./internal/convert/ -run 'TestGoldenCorpus' -v
```

## Committed PDF/PNG samples (`output/`)

| Artifact | Source |
|----------|--------|
| `output/fixture-*.pdf` | Each golden HTML via `gowkhtmltopdf` |
| `output/fixture-01-simple-invoice.png` | Image converter smoke |
| `output/showcase-toc-hf-outline.pdf` | TOC + HF + outline on fixture-16 |
| `output/wiki-ana-de-armas.pdf` | Optional complex URL smoke (Wikipedia) |

Regenerate:

```sh
make samples
```

These files are **illustrative**. They are not byte-for-byte golden masters;
font subsets and timestamps may differ across rebuilds depending on settings.
Always re-open a regenerated PDF in a real viewer when changing the PDF writer.

## Makefile targets

| Target | Action |
|--------|--------|
| `make test` | `go test ./...` |
| `make lint` | `go vet` + gofmt check |
| `make build` | `bin/gowkhtmltopdf`, `bin/gowkhtmltoimage` |
| `make golden` | Golden corpus tests |
| `make samples` | Refresh `output/` |
| `make clean` | Remove `testdata/golden/out` |

## Expected quality bar (MVP)

- Latin report HTML: readable text, sane letter-spacing, multi-page tables  
- TOC/outline showcase: navigable bookmarks, page headers/footers  
- Wikipedia-class pages: may open as multi-page PDFs but **layout and
  non-Latin fonts are incomplete** - tracked as follow-up (CID fonts, CSS)
