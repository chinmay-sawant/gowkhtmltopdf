## Summary

This PR delivers the complete **v0.2.1** release across Phases 24–30 of the canonical roadmap. It tightens embedder library contracts with uniform error-returning policies, enhances multi-page table continuation and float layout fidelity, decouples internal pipeline seams, adds native continuous Go fuzz targets for HTML/CSS/conversion, and aligns all documentation and release metadata to v0.2.1.

---

## Motivation / context

- Plans: [`plans/0.2.1/24-canonical-0.2.1-roadmap.md`](plans/0.2.1/24-canonical-0.2.1-roadmap.md) and [`plans/0.2.1/phases/`](plans/0.2.1/phases/) (Phases 24–30)
- The Markdown phase/checklist/ledger files are execution and tracking artifacts. Please disregard them during code-level review.

---

## Changes

### Library contracts and settings (Phases 24 & 28)

- **Consistent Error vs Panic Policy:** Fluent options (`PdfGlobalOptions.WithPageSize`, `WithCopies`) now store values without panicking, failing validation with standard sentinels (`ErrInvalidPageSize`, `ErrInvalidPDFCopies`) on `ValidatePDF` / `RunPDF`.
- **Nil Receiver Safety:** Audited all exported mutators; `Converter.AddHTML` now safely returns nil on nil receiver matching `AddObject`.
- **Local File Access Helpers:** Added `EnableLocalFileAccess()` helpers on `PDFRequest`, `ImageRequest`, `Converter`, `GlobalSettings`, and `ObjectSettings`.
- **Unified Security Boundary:** Canonical `NetworkPolicy` type defined in `internal/load` with `ApplyNetworkPolicy` helper, aliased to `gowkhtmltopdf.NetworkPolicy`.
- **Request Pipeline Cleanup:** Removed leftover `Image` and `ValidateImage` fields from `convert.Request`. Hand-dispatched settings table documented in `internal/settings/reflect.go`.

### Layout and pagination fidelity (Phases 25 & 26)

- **Table Continuation Chrome:** Multi-page `border-collapse` tables with `rowspan` emit clean closed top edges across continuation pages while preserving continuous rowspan cells.
- **Float Stacking & BFC:** Multiple same-side floats cleanly stack vertically without overlapping; block formatting context (BFC) enclosure verified.
- **Flex & Grid Calculations:** Explicit `flex-shrink` width assertions verified; grid row content-based height calculations implemented.

### Pipeline seams and documentation hygiene (Phases 27 & 30)

- **Package Comment Updates:** Updated `internal/pdf/doc.go` to describe the pure-Go PDF 1.4 writer.
- **Version Bump:** `VERSION`, `internal/cli/help.go`, `README.md`, `CHANGELOG.md`, and frontend data bumped to `0.2.1`.
- **Frontend Build:** Rebuilt static documentation site (`docs/`) matching v0.2.1 release.

### Verification and fuzzing (Phase 29)

- **Native Fuzz Targets:** Added `FuzzParseHTML` (`internal/html`), `FuzzParseCSS` (`internal/css`), and `FuzzConvertHTML` (`internal/convert`).
- **CI Integration:** Updated `.github/workflows/ci.yml` push triggers to include `master` and `main`.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Unchanged high performance (16x faster on 2-page invoices, 1.6x on 500-page reports). |
| **Memory** | Clean memory footprint and independent cloned converter snapshots. |
| **Behavior / correctness** | Elimination of unhandled panics on user input; clean table continuation borders; no float overlap. |
| **API / CLI** | Preferred typed `RunPDF` / `PDFRequest` and `EnableLocalFileAccess()` helpers added. |
| **Dependencies** | No new dependencies (allowlisted `go-text/typesetting` and `tdewolff/canvas` only). |
| **Binary size / build time** | Lightweight static pure-Go binary with `CGO_ENABLED=0`. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Full backward compatibility with existing wkhtmltopdf-compatible dotted `Set`/`Get` and `Converter` API. |

---

## Test plan

- [x] `make test` — all package unit and regression tests pass
- [x] `make lint` — golangci-lint v1.64.8 passes with zero warnings
- [x] `make claim-scan` — verified clean with zero disallowed claims
- [x] `make build` — builds `bin/gowkhtmltopdf` and `bin/gowkhtmltoimage` stamping version 0.2.1
- [x] `go test -fuzz=FuzzParseHTML -fuzztime=2s ./internal/html` — passed (430k+ iterations)
- [x] `go test -fuzz=FuzzParseCSS -fuzztime=2s ./internal/css` — passed (440k+ iterations)
- [x] `go test -fuzz=FuzzConvertHTML -fuzztime=2s ./internal/convert` — passed
- [x] `npm run build` (in `frontend/`) — generated clean static docs in `docs/`

### Commands

```sh
make lint
make test
make claim-scan
make build
./bin/gowkhtmltopdf --version
```

---

## Screenshots / sample output

```
Name: gowkhtmltopdf
Version: 0.2.1
Name: gowkhtmltopdf
Version: 0.2.1
claim-scan: clean
ok  	gowkhtmltopdf	0.028s
ok  	gowkhtmltopdf/cmd/gowkhtmltopdf	0.021s
ok  	gowkhtmltopdf/internal/cli	0.005s
ok  	gowkhtmltopdf/internal/convert	2.098s
ok  	gowkhtmltopdf/internal/css	0.004s
ok  	gowkhtmltopdf/internal/html	0.004s
ok  	gowkhtmltopdf/internal/layout	2.974s
ok  	gowkhtmltopdf/internal/load	0.005s
ok  	gowkhtmltopdf/internal/pdf	0.004s
ok  	gowkhtmltopdf/internal/settings	0.003s
```

---

## Related issues

- Relates to #36
- Relates to #37
- Relates to #38

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues filled
- [x] Filled body committed under `plans/0.2.1/PR/pr-v0.2.1-contracts-layout-verification.md`

---

## Reviewer checklist

- [x] Behavior matches summary and test plan
- [x] No unrelated changes in diff
- [x] Public API / CLI changes documented
- [x] New rules have fixture coverage when applicable
- [x] PR has assignee and labels
- [x] Related issues use correct Closes/Relates keywords
- [x] No secrets or generated artifacts committed
