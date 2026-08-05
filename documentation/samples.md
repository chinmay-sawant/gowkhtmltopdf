# Samples and golden fixtures

## Golden HTML (`testdata/golden/`)

Thirty report-style HTML fixtures (`fixture-01` … `fixture-30`) exercise
invoices, tables (incl. thead repeat), page breaks, CSS (float/flex/grid/
position lite), links, images, lists, typography, CJK/`--font-path`, and
orphan/widow heuristics. See
[`testdata/golden/README.md`](../testdata/golden/README.md).

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
| `output/fixture-21-detailed-report.png` | Detailed report via library image API |
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

## Expected quality bar (MVP+)

- Latin report HTML: readable text, sane letter-spacing, multi-page tables  
- TOC/outline showcase: navigable bookmarks, page headers/footers  
- Tier 2 report subset: float/flex/grid lite, thead repeat, Type0/CJK with a
  capable face via `--font-path`  
- Wikipedia-class pages: may open as multi-page PDFs but **open-web layout
  parity is not a pass criterion** — Phase 21 / fidelity smoke only
