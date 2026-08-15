## v0.2.2

Fourth public release of **gowkhtmltopdf**: a **pure-Go**, **no-cgo**, **no Qt/WebKit**, **no browser** HTML template engine that turns structured HTML and templates into multi-page PDFs and images.

**v0.2.0** shipped the print-CSS template pipeline. **v0.2.1** hardened embedder contracts and layout. **v0.2.2** extends the existing PDF writer with an explicit version policy and opt-in archival / accessibility profiles. It does **not** rebuild layout, fonts, or the convert pipeline.

Default output is still **unclaimed PDF 1.4**. `--pdf-version` / `WithPDFVersion` is a version header, **not** a conformance claim. The claim is `--pdf-profile` / `WithPDFProfile`. Encryption, AcroForm, signatures, and JavaScript remain out of scope.

- **License:** [MIT](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/LICENSE) — Copyright (c) 2026 Chinmay Sawant
- **Version source:** [`VERSION`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/VERSION) (`0.2.2`)
- **Site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
- **Compare:** https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.1...v0.2.2
- **Epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29) (newer PDF versions and compliance)
- **PRs since v0.2.1:** [#44](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/44) (v0.2.1 release title) · [#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45) (PDF 1.7 + PDF/A-3a + PDF/UA-1, #31) · [#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46) (PDF 2.0 + PDF/A-4 + PDF/UA-2, #32/#33) · [#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47) (criticality, sentinels, UA list nesting) · [#48](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/48) (docs, site, samples) · [#49](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/49) (lint / ObjRef)

---

### Highlights

| Area | What you get in v0.2.2 |
|------|------------------------|
| **PDF 1.7** | Opt-in `%PDF-1.7` via `--pdf-version 1.7` / `WithPDFVersion("1.7")`. Version only — no PDF/A or PDF/UA claim. |
| **PDF 2.0** | Opt-in `%PDF-2.0` via `--pdf-version 2.0` / `WithPDFVersion("2.0")` (trailer `/ID`, UTF-8 document strings, non-claiming XMP). Version only. |
| **PDF/A-3a + PDF/UA-1** | Opt-in `--pdf-profile a3a-ua1` (also `a3a`, `ua1`). Implies PDF 1.7. Claiming XMP (`pdfaid:part=3`, `pdfaid:conformance=A`, `pdfuaid:part=1`), sRGB OutputIntent, `/DefaultRGB`, MarkInfo, and a logical structure tree. Multi-page structure elements emit MCR dictionaries. |
| **PDF/A-4 + PDF/UA-2** | Opt-in `--pdf-profile a4-ua2` (also `a4`, `ua2`). Implies PDF 2.0. Claiming XMP (`pdfaid:part=4` / `rev=2020`, `pdfuaid:part=2` / `rev=2024`), sRGB+Gray OutputIntent, structure `/Namespace`, ListNumbering, and dual named destinations (`/D` page + `/SD` structure). PDF/A-4 omits trailer `/Info` (metadata lives in XMP). |
| **Tagged PDF wiring** | Cloned-page MCIDs, link/outline `/SD` identity, single `/Document`, header/footer isolation from the body tree. Lists nest `L` → `LI` → `LBody` → `Link` (inline `<a>` inside `<li>` is no longer a sibling of `LI`). CIDFontType2 `/FontName` matches parent `/BaseFont`. |
| **Profile Get and sentinels** | `Get("pdfprofile")` returns the canonical token (`a3a-ua1` → `PDF/A-3a+PDF/UA-1`). Wrong version + profile uses `ErrConformanceRequiresPDF17` / `ErrConformanceRequiresPDF20`. |
| **Samples** | `make samples` writes unclaimed `output/pdf-1.7/` and `output/pdf-2.0/` plus claimed `output/pdf-1.7-compliance/` (`a3a-ua1`) and `output/pdf-2.0-compliance/` (`a4-ua2`). |
| **Docs & site** | Guides and the documentation site state the 0.2.1 / 0.2.2 split honestly. README brand mark, site a11y, and issue-dossier verdicts updated. |

**Highest compliance by version**

| Base | Archival | Accessibility | Opt-in |
|------|----------|---------------|--------|
| PDF 1.7 | PDF/A-3a | PDF/UA-1 | `--pdf-profile a3a-ua1` |
| PDF 2.0 | PDF/A-4 | PDF/UA-2 | `--pdf-profile a4-ua2` |

---

### Showcase — version and compliance samples

Live gallery: https://chinmay-sawant.github.io/gowkhtmltopdf/#/showcase  
Committed PDFs: [`output/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.2/output) · source HTML: [`testdata/golden/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.2/testdata/golden)

The existing 64 template fixtures remain. New trees (same two fixtures: detailed report + architecture diagram):

| Dir | How produced | Claim? |
|-----|--------------|--------|
| `output/pdf-1.7/` | `--pdf-version 1.7` | No — version only |
| `output/pdf-1.7-compliance/` | `--pdf-profile a3a-ua1` | PDF/A-3a + PDF/UA-1 |
| `output/pdf-2.0/` | `--pdf-version 2.0` | No — version only |
| `output/pdf-2.0-compliance/` | `--pdf-profile a4-ua2` | PDF/A-4 + PDF/UA-2 |

These are **artifacts**, not golden byte baselines. A version flag is not a conformance claim.

---

### Install / build

Cross-platform binaries are attached to this release (`gowkhtmltopdf` and `gowkhtmltoimage` for linux / windows / darwin × amd64 / arm64) plus `SHA256SUMS`.

Or install the CLIs with Go 1.26+ (puts them on `GOBIN` / `$(go env GOPATH)/bin`):

```sh
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.2
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltoimage@v0.2.2
gowkhtmltopdf --version
```

From a source checkout:

```sh
git clone https://github.com/chinmay-sawant/gowkhtmltopdf.git
cd gowkhtmltopdf
git checkout v0.2.2
make build
```

Unclaimed PDF 1.7:

```sh
./bin/gowkhtmltopdf --pdf-version 1.7 --enable-local-file-access \
  testdata/golden/fixture-21-detailed-report.html /tmp/report-17.pdf
```

Dual PDF/A-3a + PDF/UA-1 (implies 1.7):

```sh
./bin/gowkhtmltopdf --pdf-profile a3a-ua1 --enable-local-file-access \
  testdata/golden/fixture-56-architecture-diagram.html /tmp/arch-a3a-ua1.pdf
```

Dual PDF/A-4 + PDF/UA-2 (implies 2.0):

```sh
./bin/gowkhtmltopdf --pdf-profile a4-ua2 --enable-local-file-access \
  testdata/golden/fixture-21-detailed-report.html /tmp/report-a4-ua2.pdf
```

Library (preferred typed API):

```go
package main

import (
	"bytes"
	"context"
	"os"

	"github.com/chinmay-sawant/gowkhtmltopdf"
)

func main() {
	ctx := context.Background()
	var out bytes.Buffer

	req := &gowkhtmltopdf.PDFRequest{
		Global: gowkhtmltopdf.NewPdfGlobalOptions().
			WithPageSize("A4").
			WithPDFVersion("1.7").     // optional: "1.4" (default), "1.7", "2.0"
			WithPDFProfile("a3a-ua1"). // optional; implies 1.7
			WithTitle("Quarterly report").
			Build(),
		Objects: []*gowkhtmltopdf.ObjectSettings{
			gowkhtmltopdf.NewObjectSettings().SetPage("report.html"),
		},
		Output: &out,
	}
	req.EnableLocalFileAccess()

	if err := gowkhtmltopdf.RunPDF(ctx, req); err != nil {
		panic(err)
	}

	_ = os.WriteFile("output.pdf", out.Bytes(), 0644)
}
```

CLI / settings keys: `--pdf-version` / `pdfversion` (`1.4`, `1.7`, `2.0`); `--pdf-profile` / `pdfprofile` (`a3a`, `ua1`, `a3a-ua1`, `a4`, `ua2`, `a4-ua2`, or the canonical `PDF/A-…` / `PDF/UA-…` names).

---

### What landed in v0.2.2

#### 1. PDF 1.7 version path and 1.7 profiles ([#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45), #31)

- Writer policy for `%PDF-1.7` on an explicit version flag.
- Opt-in PDF/A-3a, PDF/UA-1, and dual `a3a-ua1` via `--pdf-profile` / `WithPDFProfile` (implies PDF 1.7).
- Claiming XMP, sRGB OutputIntent, `/DefaultRGB`, MarkInfo, logical structure tree.
- Multi-page structure elements emit MCR dictionaries (ISO 32000-1 §14.7.4.2).
- CIDFontType2 `/FontName` equals parent `/BaseFont` (Arlington `FontDescriptorCIDType2.FontName`).
- Layout tagging no longer accumulates one document-wide mega-`/P` with duplicate bare MCIDs.

#### 2. PDF 2.0 version path and 2.0 profiles ([#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46), #32 / #33)

- Writer policy for `%PDF-2.0`: header + binary comment, trailer `/ID`, UTF-8 document strings, non-claiming XMP (`dc:format`, `pdf:Producer`, dates — no `pdfaid` / `pdfuaid`).
- Opt-in PDF/A-4, PDF/UA-2, and dual `a4-ua2` (implies PDF 2.0).
- PDF/A-4: `pdfaid:part=4` / `rev=2020`, sRGB + Gray OutputIntent, `/DefaultRGB` + `/DefaultGray`, trailer **omits `/Info`**.
- PDF/UA-2: `pdfuaid:part=2` / `rev=2024`, structure `/Namespace` (`http://iso.org/pdf2/ssn`), ListNumbering on `/L`, catalog `/Lang` / `/MarkInfo` / `/ViewerPreferences << /DisplayDocTitle true >>`.
- Dual named destinations: `/D` page XYZ (Arlington / PDF/A) plus `/SD` structure dest (UA-2 clause 8.8). Outline items also bind `/SE` to the heading struct elem.

#### 3. Tagged-PDF wiring, contracts, and cost ([#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47))

- Cloned pages keep MCIDs and content refs; header/footer links stay pagination artifacts (not under `/Document`).
- One `/Document`; outline `/SD` matches heading identity; internal and TOC links resolve dest struct elems.
- New leaf `internal/pdfprofile`: one alias table. `WithPDFVersion` / `WithPDFProfile` store canonical tokens. `Get("pdfprofile")` after `"a3a-ua1"` is `PDF/A-3a+PDF/UA-1`.
- Unified `ErrConformanceRequiresPDF17` / `ErrConformanceRequiresPDF20` (with `ErrProfileRequiresPDF17` / `ErrProfileRequiresPDF20` aliases). `ErrProfilePDF20Unsupported` remains defined for source compatibility but is never returned.
- Default unclaimed 1.4 stays isolated: no XMP, ICC, StructTree, ParentTree, named dests, `/Tabs /S`, or trailer `/ID`.
- ICC Flate bytes precomputed; structure serialize tightened; write benches cover all nine profile combinations.

#### 4. UA list nesting ([#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47))

Inline `<a>` inside `<li>` used to emit `L > Link` (illegal under ISO 32000-1 / ISO 32005). Links now live under `LI / LBody`. Regression: `TestStructureTreeListLinkHierarchy`.

#### 5. Docs, site, and samples ([#44](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/44), [#48](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/48))

- v0.2.1 GitHub release title simplified after that tag.
- README, overview, CLI, library API, architecture, fidelity, samples, deferred, and landscape-2026 describe 1.7 / 2.0 and profiles as 0.2.2 work (not as if they shipped in 0.2.1).
- Issue-dossier verdicts re-audited against 0.2.1 vs 0.2.2.
- Site: drop webfonts, a11y, Getting Started / benchmarks spacing, docs TOC hash-route, gopher brand, Pages assets rebuilt.
- `make samples` emits the 1.7 / 2.0 and compliance PDF trees; fixture PDFs refreshed.

#### 6. Lint hygiene ([#49](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/49))

- `Page.AddLinkURI` / `Page.AddLinkDest` return the exported `ObjRef` alias (no unused `//nolint:revive`).
- `make lint` also runs frontend `npm lint`.

---

### Documentation

| Doc | Link |
|------|------|
| **Site** | https://chinmay-sawant.github.io/gowkhtmltopdf/ |
| **Overview** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/overview.md |
| **Getting started** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/getting-started.md |
| **CLI** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/cli.md |
| **Library API** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/library-api.md |
| **Architecture (PDF writer)** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/architecture/09-pdf-writer.md |
| **Fidelity** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/fidelity.md |
| **Deferred** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/deferred.md |
| **Samples** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/output/README.md |
| **Contributing** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/CONTRIBUTING.md |
| **Changelog** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/CHANGELOG.md#022-2026-08-15 |
| **0.2.2 plans** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/plans/0.2.2/README.md |

---

### Breaking changes / migration

| Item | Migration |
|------|-----------|
| Default output | Still unclaimed PDF 1.4. Existing callers need no change. |
| `Get("pdfprofile")` | Returns the **canonical** token (`PDF/A-3a+PDF/UA-1`), not the short alias you set (`a3a-ua1`). Compare against canonical names or treat Get as display, not as the input string. |
| Profile + wrong explicit version | Fails with `ErrConformanceRequiresPDF17` / `ErrConformanceRequiresPDF20` (aliases `ErrProfileRequiresPDF17` / `ErrProfileRequiresPDF20`). |
| `ErrProfilePDF20Unsupported` | Still defined; **never returned**. `a4` / `ua2` / `a4-ua2` now succeed and imply PDF 2.0. |
| PDF/A-4 `/Info` | Trailer has no `/Info` under A-4. Read Catalog `/Metadata` XMP. |
| Dest arrays under UA-2 | Named dests via `/Names /Dests`. `/D` remains a page XYZ array; `/SD` is the structure dest. |
| PDF/UA title / alt | Empty document title → `ErrTitleRequired`. Figure/image without alt → `ErrPDFUAMissingAlt`. |

---

### Known limitations

- **Not a browser.** No JavaScript (`<script>` stripped; JS flags are unknown options). No Chrome / Wikipedia visual parity.
- Flex/grid/float/sticky remain a **print CSS subset**.
- CJK/Arabic **Partial** (operator-supplied faces + OT when present). `writing-mode: vertical-*` is parsed but lays out horizontal. No WOFF2.
- No AcroForm, no PDF encryption, no signatures. `--pdf-version` alone is **not** PDF/A or PDF/UA.
- Full list: [`documentation/deferred.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.2/documentation/deferred.md).

---

### Verification Gates

```sh
make test
make lint
make claim-scan
make build
```

Optional veraPDF (in-tree 1.30.2) on the committed compliance samples:

```sh
COMPLIANCE_FLAVOURS=3a,ua1 ./compliance/verify_pdfs.sh \
  --pdf output/pdf-1.7-compliance/fixture-21-detailed-report.pdf \
  --pdf output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf

./compliance/run_verapdf.sh --both \
  output/pdf-2.0-compliance/fixture-21-detailed-report.pdf \
  output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf
```

---

## What's Changed

* docs: simplify v0.2.1 release note title by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/44
* feat(pdf): PDF 1.7 + PDF/A-3a + PDF/UA-1 by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45
* feat(pdf): PDF 2.0 + PDF/A-4 + PDF/UA-2 by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46
* fix(pdf): 0.2.2 criticality, sentinels, and UA list nesting by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47
* docs: sync guides and site with PDF 1.7/2.0 profiles by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/48
* fix(lint): exported ObjRef and frontend lint in make lint by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/49

**Full Changelog**: https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.1...v0.2.2
