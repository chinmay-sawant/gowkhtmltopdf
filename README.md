<h1 align="center">
  <img src="docs/logo.png" width="220" alt="gowkhtmltopdf"><br>
  gowkhtmltopdf
</h1>

gowkhtmltopdf is a no-cgo **PDF engine based on HTML templates** (and HTML→image)
for **structured templates and documents**: invoices, receipts, certificates,
storybooks, posters, statements, tables, and multi-page documents with headers,
footers, tables of contents, and PDF outlines — without any wrappers.

It is a clean-room work-alike of the [wkhtmltopdf](https://wkhtmltopdf.org/)
CLI surface. Native builds use **no browser process**, **no cgo**, and no
native converter process. Two static binaries (`gowkhtmltopdf`,
`gowkhtmltoimage`) and a Go library run an in-repo pipeline (load → parse →
style → layout → paginate → paint → write). An opt-in browser build runs the
same pipeline through WebAssembly for inline HTML previews. Direct modules are allowlisted:
[`go-text/typesetting`](https://github.com/go-text/typesetting) (OpenType
shaping) and [`tdewolff/canvas`](https://github.com/tdewolff/canvas) (SVG
rasterization). The product is HTML templates and documents, not Chrome visual parity.

**Status:** **v0.2.5** (current release). The native Document API and explicit
CLI grammar are now the supported surface. Opt-in PDF 1.7 / 2.0 and PDF/A +
PDF/UA profiles. **License:** [MIT](LICENSE).

> **Note on `master`:** `master` tracks active development and may be broken
> at times. For a stable build, use the [latest tagged release](https://github.com/chinmay-sawant/gowkhtmltopdf/releases).

## What it is for

| You need… | This project |
|-----------|----------------|
| Invoices, tables, page breaks from Go | Yes |
| Headers, footers, TOC, PDF bookmarks | Yes |
| PDF 1.4 / PDF 1.7 / PDF 2.0 output | Default PDF 1.4. Opt-in 1.7 / 2.0 via `--pdf-version`. Version alone is **not** a PDF/A or PDF/UA claim |
| PDF/A-3a & PDF/UA-1 compliance | Opt-in via `--pdf-profile a3a-ua1` / `WithPDFProfile` (implies PDF 1.7) |
| PDF/A-4 & PDF/UA-2 compliance | Opt-in via `--pdf-profile a4-ua2` / `WithPDFProfile` (implies PDF 2.0) |
| Offline native static binaries; no browser process / no cgo | Yes |
| Browser preview for inline HTML | Yes, through the opt-in [WASM adapter](documentation/wasm.md) |
| Full CSS, JavaScript, or Chrome parity | No — print CSS subset; no JS |
| CJK / complex Unicode | Partial — Type0/CID + `--font-path`; see [fonts.md](documentation/fonts.md) |

## Quick start

Requires Go 1.26+.

```sh
make build
./bin/gowkhtmltopdf --allow-local-files -o /tmp/invoice.pdf \
  testdata/golden/fixture-01-simple-invoice.html
```

Committed samples live in [output/](output/) (`make samples`).
Python API samples land under [output/python/](output/python/)
(`make samples-python`; needs `CGO_ENABLED=1`).
Install, flags, and HTTP URLs: [getting-started.md](documentation/getting-started.md).
Browser build and preview: [wasm.md](documentation/wasm.md).

## Documentation

| Document | What it covers |
|----------|----------------|
| [documentation/README.md](documentation/README.md) | Documentation index |
| [documentation/overview.md](documentation/overview.md) | Product overview and design principles |
| [documentation/getting-started.md](documentation/getting-started.md) | Install and first conversion |
| [documentation/cli.md](documentation/cli.md) | CLI grammar and flags |
| [documentation/library-api.md](documentation/library-api.md) | Go library API |
| [documentation/wasm.md](documentation/wasm.md) | Browser WASM conversion and previews |
| [documentation/python.md](documentation/python.md) | Python bindings: in-process `pip install gowkhtmltopdf` |
| [documentation/MIGRATION-0.2.4.md](documentation/MIGRATION-0.2.4.md) | 0.2.3 library/CLI to the 0.2.4 Document API |
| [documentation/architecture.md](documentation/architecture.md) | Package map and pipeline |
| [documentation/architecture/README.md](documentation/architecture/README.md) | Deep-dive architecture notes |
| [documentation/fidelity.md](documentation/fidelity.md) | Fidelity tiers and claims language |
| [documentation/compatibility-matrix.md](documentation/compatibility-matrix.md) | Per-element / per-property / per-flag contract |
| [documentation/fonts.md](documentation/fonts.md) | Bundled faces, `--font-path`, `@font-face` |
| [documentation/samples.md](documentation/samples.md) | Golden fixtures and `output/` |
| [documentation/performance.md](documentation/performance.md) | Benchmarks and how to measure |
| [documentation/benchmarks.md](documentation/benchmarks.md) | Consolidated current benchmark capture (2026-09-12) |
| [testdata/golden/benchmarks/README.md](testdata/golden/benchmarks/README.md) | Current CLI vs wkhtmltopdf snapshot |
| [documentation/deferred.md](documentation/deferred.md) | Deferred features and next gates |
| [documentation/THREAT-MODEL.md](documentation/THREAT-MODEL.md) | Security / ACL / network policy |
| [documentation/integration-security.md](documentation/integration-security.md) | Embedding in HTTP apps (SSRF) |
| [comparison: go-wkhtmltopdf](documentation/comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md) | Binary wrapper vs this in-process engine |
| [comparison: 2026 landscape](documentation/comparison-with-others/landscape-2026.md) | Chromium, wkhtmltopdf, WeasyPrint, Prince |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Setup, tests, PR workflow |
| [CHANGELOG.md](CHANGELOG.md) | Release history |
| [examples/](examples/) | Library example programs |
| [output/](output/) | Regenerable sample PDFs/PNG |
| [plans/README.md](plans/README.md) | Implementation ledger index |
| [LICENSE](LICENSE) | MIT license |

## Library

The supported Go API is `Document` / `ImageDocument` with explicit `Content`
sources. The pre-0.2.4 wkhtml-shaped root exports are removed.

```go
doc := gowkhtmltopdf.Document{
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{
            HTML: []byte(`<html><body><h1>Invoice</h1></body></html>`),
        },
    }},
    PageSize: "A4",
}
pdfBytes, err := doc.PDF(ctx)
```

Local files, TOC/cover fields, network policy, and the migration table:
[documentation/library-api.md](documentation/library-api.md),
[documentation/MIGRATION-0.2.4.md](documentation/MIGRATION-0.2.4.md).

Python callers get the same engine in-process through an opt-in shared
library: [documentation/python.md](documentation/python.md).

```python
from gowkhtmltopdf import PDFOptions, convert_html_to_pdf

pdf_bytes = convert_html_to_pdf(
    b"<html><body><h1>Invoice</h1></body></html>",
    options=PDFOptions(page_size="A4"),
)
```

## Performance

**Current snapshot (2026-09-12):** generic `bin/gowkhtmltopdf` (`VERSION`
0.2.5 on the 0.2.6 working tree) versus installed **wkhtmltopdf 0.12.6.1
(patched Qt)** on Linux amd64 (WSL2), 13th Gen Intel Core i7-13700HX. Same
report fixture (20 invoice rows per requested page), median of three timed
process runs after one warmup.

| Pages | gowkhtmltopdf | wkhtmltopdf | Faster by |
|------:|--------------:|------------:|----------:|
| 2 | 14 ms | 260 ms | **18.50x** |
| 10 | 26 ms | 286 ms | **11.18x** |
| 100 | 126 ms | 546 ms | **4.35x** |
| 500 | 562 ms | 1.760 s | **3.13x** |

Faster at every tested size. Gowk also used less peak RSS at every tested
size in this capture, including 500 pages (79,296 KiB versus 123,076 KiB).

Same host, same fixture family against other engines (default external
matrix: 2 / 10 / 50 / 100 pages):

| Pages | vs WeasyPrint | vs Puppeteer / Chrome |
|------:|--------------:|----------------------:|
| 2 | **40.86x** | **92.85x** |
| 10 | **53.62x** | **57.16x** |
| 50 | **76.88x** | **25.10x** |
| 100 | **89.43x** | **17.16x** |

Full matrices, RSS, PDF sizes, internal-engine and public-library
`go test -bench` rows, and historical snapshots:

- [documentation/benchmarks.md](documentation/benchmarks.md)
- [documentation/performance.md](documentation/performance.md)
- [testdata/golden/benchmarks/README.md](testdata/golden/benchmarks/README.md)
- [cli-compare.md](testdata/golden/benchmarks/cli-compare.md)
- [weasyprint-compare.md](testdata/golden/benchmarks/weasyprint-compare.md)
- [puppeteer-compare.md](testdata/golden/benchmarks/puppeteer-compare.md)

Reproduce:

```sh
make build
make bench-cli-compare
./scripts/bench-external.sh
./scripts/bench-performance-recovery.sh --mode=<mode>
make bench
make bench-engine
make bench-lib
make python-benchmarks
```

`make python-benchmarks` rebuilds `dist/libgowkhtmltopdf.so` (`CGO_ENABLED=1`)
and times the in-process Python `Document.pdf()` / `ImageDocument.image()`
path on the same `report.html.tmpl` fixture (20 invoice rows per page) that
`make bench-lib` uses. Override sizes with
`GOWKHTMLTOPDF_BENCH_SIZES=2,10,50`. The Python architecture sample is
`make python-api` (`testdata/golden/python_api` ->
`output/python/architecture-diagram.pdf`).

## Development

gowkhtmltopdf is a pure-Go rendering engine written from scratch. Humans own
the architecture and domain design. AI tools including Grok, OpenAI Codex,
OpenCode, and Cursor help with implementation drafts, visual checks against
golden fixtures, and test suites.

Much of the code starts as an AI draft. It does not ship unchecked. Human
maintainers own the pipeline of load, parse, style, layout, paginate, paint,
and write. Humans review changes and require proof before merge: `make test`,
`make lint`, `make golden`, and `make claim-scan`, and they also validate
all 50+ sample templates manually.

Performance is part of that proof. The full record lives in
[documentation/performance.md](documentation/performance.md). Raw numbers and
reproduce steps live in
[testdata/golden/benchmarks/README.md](testdata/golden/benchmarks/README.md).
Profile with `go test -cpuprofile` plus `go tool pprof -top`; see
[How to measure](documentation/performance.md#how-to-measure). The CI perf
budget is `TestTenPageTableReportPerformance` in
[internal/convert/perf_test.go](internal/convert/perf_test.go).

## License

[MIT License](LICENSE) — Copyright (c) 2026 **Chinmay Sawant**.

Bundled Liberation and DejaVu fonts are SIL OFL / Bitstream Vera; see
[internal/pdf/assets/NOTICE](internal/pdf/assets/NOTICE).
The Noto KR test subset ships [testdata/fonts/OFL.txt](testdata/fonts/OFL.txt).
