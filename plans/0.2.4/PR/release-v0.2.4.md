## v0.2.4

Fifth public release of **gowkhtmltopdf**: a **pure-Go**, **no-cgo**, **no Qt/WebKit**, **no browser** HTML template engine that turns structured HTML and templates into multi-page PDFs and images.

**v0.2.3** was the `go install` / GitHub module-path tag on the same engine as v0.2.2. **v0.2.4** is an intentional pre-1.0 **hard break** of the public library and both CLIs: the wkhtml-shaped `Converter` / dotted `Set`/`Get` / typed request wrappers are gone; the product surface is a Go-native `Document` / `ImageDocument` model with a matching CLI grammar. Layout and PDF writers stay; the outer contract and external bench paths change.

Default output is still **unclaimed PDF 1.4**. `--pdf-version` / `Document.PDFVersion` is a version header, **not** a conformance claim. The claim is `--pdf-profile` / `Document.PDFProfile`. Encryption, AcroForm, signatures, and JavaScript remain out of scope.

- **License:** [MIT](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/LICENSE) — Copyright (c) 2026 Chinmay Sawant
- **Version source:** [`VERSION`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/VERSION) (`0.2.4`)
- **Site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
- **Migration:** [`documentation/MIGRATION-0.2.4.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/MIGRATION-0.2.4.md)
- **Compare:** https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.3...v0.2.4
- **PR:** [#53](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/53) (Document API, CLI redesign, benches, docs)

---

### Highlights

| Area | What you get in v0.2.4 |
|------|------------------------|
| **Document API** | Root-package `Document` / `ImageDocument`, explicit `Content` sources (`HTML` / `File` / `URL`), `Page` / `Cover` / `TOC`, named option fields, validation before engine work, writer-first `WritePDF` / `WriteImage` plus byte helpers `PDF` / `Image`. |
| **Hard break** | Public `Converter`, `ImageConverter`, dotted settings, and `PDFRequest` / `ImageRequest` / `RunPDF` / `RunImage` are **removed** (no `compat` package). |
| **CLI redesign** | Both binaries use required `-o`/`--output`, positional page files, `--html` / `--url`, `--cover` / `--toc`, and `--allow-local-files`. The old `page` / `cover` / `toc` object grammar is rejected. |
| **Migration guide** | Step-by-step library + CLI mapping in [`MIGRATION-0.2.4.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/MIGRATION-0.2.4.md). |
| **External benches** | Frozen three-engine process compares: wkhtmltopdf (`make bench-cli-compare`), WeasyPrint + Puppeteer (`make bench` / `scripts/bench-external.sh`). Optional host tools skip with evidence. Artifacts under `testdata/golden/benchmarks/`. |
| **Layout fixes** | Inline highlight backgrounds bounded to glyph runs; sticky continuation chrome trimmed across page breaks; related goldens refreshed. |
| **Docs & site** | Library API, CLI, getting started, architecture, examples, and the documentation site updated for the Document model and new bench presentation. |
| **Version stamp** | `VERSION` / `internal/cli.Version` / dated `CHANGELOG` **0.2.4 (2026-08-18)**. |

PDF 1.7 / 2.0 and PDF/A + PDF/UA profiles from v0.2.2 remain available on the new surface (`Document.PDFVersion` / `Document.PDFProfile` and the matching CLI flags).

**Highest compliance by version** (unchanged from v0.2.2 / v0.2.3)

| Base | Archival | Accessibility | Opt-in |
|------|----------|---------------|--------|
| PDF 1.7 | PDF/A-3a | PDF/UA-1 | `--pdf-profile a3a-ua1` |
| PDF 2.0 | PDF/A-4 | PDF/UA-2 | `--pdf-profile a4-ua2` |

---

### Showcase — Document grammar and benches

Live gallery: https://chinmay-sawant.github.io/gowkhtmltopdf/#/showcase  
Benchmarks page: https://chinmay-sawant.github.io/gowkhtmltopdf/#/benchmarks  
Committed PDFs: [`output/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.4/output) · source HTML: [`testdata/golden/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.4/testdata/golden)

Process-compare artifacts (shared report fixture; medians after warmup — not an SLA):

| Compare | Reproduce | Snapshot speedup (indicative) |
|---------|-----------|-------------------------------|
| vs wkhtmltopdf | `make bench-cli-compare` | ~15× at 2 pages → ~1.6× at 500 pages |
| vs WeasyPrint | `make bench` | tens of × across 2–100 pages |
| vs Puppeteer / Chrome | `make bench` | faster process wall time on the fixture (RSS methods differ — read artifact footnotes) |

Tables and footnotes: [`testdata/golden/benchmarks/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.4/testdata/golden/benchmarks).

---

### Install / build

Cross-platform binaries are attached to this release (`gowkhtmltopdf` and `gowkhtmltoimage` for linux / windows / darwin × amd64 / arm64) plus `SHA256SUMS`.

Or install the CLIs with Go 1.26+ (puts them on `GOBIN` / `$(go env GOPATH)/bin`):

```sh
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.4
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltoimage@v0.2.4
gowkhtmltopdf --version
```

Library pin:

```sh
go get github.com/chinmay-sawant/gowkhtmltopdf@v0.2.4
```

From a source checkout:

```sh
git clone https://github.com/chinmay-sawant/gowkhtmltopdf.git
cd gowkhtmltopdf
git checkout v0.2.4
make build
```

Local PDF (0.2.4 CLI grammar):

```sh
./bin/gowkhtmltopdf --allow-local-files -o /tmp/invoice.pdf \
  testdata/golden/fixture-01-simple-invoice.html
```

Inline HTML / cover + TOC:

```sh
gowkhtmltopdf -o out.pdf --html '<h1>Hi</h1>'
gowkhtmltopdf --allow-local-files --cover cover.html --toc -o book.pdf chapter.html
gowkhtmltoimage --allow-local-files -o page.png page.html
```

Unclaimed PDF 1.7:

```sh
./bin/gowkhtmltopdf --pdf-version 1.7 --allow-local-files \
  -o /tmp/report-17.pdf testdata/golden/fixture-21-detailed-report.html
```

Dual PDF/A-3a + PDF/UA-1 (implies 1.7):

```sh
./bin/gowkhtmltopdf --pdf-profile a3a-ua1 --allow-local-files \
  -o /tmp/arch-a3a-ua1.pdf testdata/golden/fixture-56-architecture-diagram.html
```

Library (Document API):

```go
package main

import (
	"context"
	"os"

	"github.com/chinmay-sawant/gowkhtmltopdf"
)

func main() {
	ctx := context.Background()
	doc := gowkhtmltopdf.Document{
		Pages: []gowkhtmltopdf.Page{{
			Source: gowkhtmltopdf.Content{File: "report.html"},
		}},
		PageSize:        "A4",
		PDFVersion:      "1.7",     // optional: "1.4" (default), "1.7", "2.0"
		PDFProfile:      "a3a-ua1", // optional; implies 1.7
		Title:           "Quarterly report",
		AllowLocalFiles: true,
	}

	out, err := os.Create("output.pdf")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	if err := doc.WritePDF(ctx, out); err != nil {
		panic(err)
	}
}
```

CLI / document fields: `--pdf-version` / `Document.PDFVersion` (`1.4`, `1.7`, `2.0`); `--pdf-profile` / `Document.PDFProfile` (`a3a`, `ua1`, `a3a-ua1`, `a4`, `ua2`, `a4-ua2`, or the canonical `PDF/A-…` / `PDF/UA-…` names).

---

### What landed in v0.2.4

#### 1. Document / ImageDocument public API ([#53](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/53))

- Add `Document`, `ImageDocument`, `Content`, `Page`, `TOC`, options structs, and validation at the public boundary.
- Writer-first `WritePDF` / `WriteImage`; byte-returning `PDF` / `Image`; explicit `WritePDFOutline` when an outline sink is needed.
- Delete the wkhtml-shaped public surface (`Converter`, dotted `Set`/`Get`, `PDFRequest` / `ImageRequest`, `RunPDF` / `RunImage`) with **no** compatibility shim.
- Engine adapters remain on `internal/convert` / `internal/imageout`. Document order is Cover → TOC → Pages.

#### 2. CLI redesign ([#53](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/53))

- Required `-o` / `--output`; positional page files; `--html` / `--url`; `--cover` / `--toc`.
- `--allow-local-files` replaces `--enable-local-file-access`.
- Old object grammar (`page` / `cover` / `toc` keywords, final positional output) is no longer accepted.
- Sample and benchmark commands use the native 0.2.4 grammar.

#### 3. External benchmarks and harness freeze ([#53](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/53))

- `make bench-cli-compare` → wkhtmltopdf process compare.
- `make bench` → WeasyPrint + Puppeteer via `scripts/bench-external.sh`, then wkhtmltopdf.
- `make bench-lib` / related targets exercise the public `Document.WritePDF` path.
- Checked-in snapshots under `testdata/golden/benchmarks/{cli,weasyprint,puppeteer}-compare.md` (+ CSV).

#### 4. Layout correctness ([#53](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/53))

- Bound inline highlight backgrounds so paint does not spill past glyph runs.
- Trim sticky continuation chrome across page breaks.
- Refresh related golden / fixture outputs and showcase screenshots.

#### 5. Docs, examples, VERSION, CHANGELOG, site ([#53](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/53))

- Ship [`documentation/MIGRATION-0.2.4.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/MIGRATION-0.2.4.md).
- Refresh library API, CLI, getting started, architecture, and `examples/pdf` + `examples/image`.
- Frontend / GitHub Pages copy for Document API and benchmarks.
- `VERSION=0.2.4`; dated `CHANGELOG.md` **0.2.4 (2026-08-18)**.

---

### Documentation

| Doc | Link |
|------|------|
| **Site** | https://chinmay-sawant.github.io/gowkhtmltopdf/ |
| **Migration 0.2.4** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/MIGRATION-0.2.4.md |
| **Overview** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/overview.md |
| **Getting started** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/getting-started.md |
| **CLI** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/cli.md |
| **Library API** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/library-api.md |
| **Performance / benches** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/testdata/golden/benchmarks/README.md |
| **Fidelity** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/fidelity.md |
| **Deferred** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/deferred.md |
| **Samples** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/output/README.md |
| **Contributing** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/CONTRIBUTING.md |
| **Changelog** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/CHANGELOG.md#024-2026-08-18 |
| **0.2.4 plans** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/plans/0.2.4/README.md |

---

### Breaking changes / migration

| Item | Migration |
|------|-----------|
| `Converter` / `ImageConverter` | `Document` / `ImageDocument` |
| Dotted `Set` / `Get` | Named struct fields on `Document` / `Page` / options |
| `PDFRequest` / `RunPDF` | `Document.WritePDF` / `Document.PDF` |
| `ImageRequest` / `RunImage` | `ImageDocument.WriteImage` / `ImageDocument.Image` |
| `SetPage` / `SetBody` | `Content{File:…}` / `Content{HTML:…, Base:…}` (or helpers) |
| Global + object local-file ACL pair | `AllowLocalFiles: true` / `--allow-local-files` |
| CLI `page` / `cover` / `toc` + final positional output | `-o`, positional pages, `--cover` / `--toc`, `--html` / `--url` |
| `--enable-local-file-access` | `--allow-local-files` |
| `go install …@v0.2.3` | Use `@v0.2.4` |

There is **no** source-compatible adapter package. Full guide: [`MIGRATION-0.2.4.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/MIGRATION-0.2.4.md).

---

### Known limitations

- **Not a browser.** No JavaScript (`<script>` stripped; JS flags are unknown options). No Chrome / Wikipedia visual parity.
- Flex/grid/float/sticky remain a **print CSS subset**.
- CJK/Arabic **Partial** (operator-supplied faces + OT when present). `writing-mode: vertical-*` is parsed but lays out horizontal. No WOFF2.
- No AcroForm, no PDF encryption, no signatures. `--pdf-version` alone is **not** PDF/A or PDF/UA.
- Full list: [`documentation/deferred.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.4/documentation/deferred.md).

---

### Verification Gates

```sh
make test
make lint
make build
./bin/gowkhtmltopdf --version
test "$(tr -d '[:space:]' < VERSION)" = "0.2.4"
```

Optional:

```sh
make bench-cli-compare
make bench   # WeasyPrint / Puppeteer when installed; missing engines skip with evidence
```

---

## What's Changed

* feat(api): ship 0.2.4 Document model, CLI redesign, and benches by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/53

**Full Changelog**: https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.3...v0.2.4
