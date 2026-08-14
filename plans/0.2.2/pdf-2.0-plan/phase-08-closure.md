# Phase 8 — Closure

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15)
> **Estimated effort:** 1 day after 1–7
> **Depends on:** Phases 1–7
> **Unblocks:** issue #32 can be closed when evidence is attached; #33 may start later

---

## Overview

Close the #32 ledger only with current evidence. Do not mark #29 or
#33 done. Do not treat a green `make test` as a PDF/A claim.

---

## Phase 8 checklist

### 8.1 Required checks

- [x] `make lint` → record outcome
  (First run: ~50 enable-all findings, all leftovers of phases 1–6 — unused
  `//nolint` directives on new tests (`cli_test.go`, `convert_test.go`,
  `golden_test.go`, `convert.go`), `cyclop`/`funlen` on new 2.0 tests
  (`policy_test.go`, `pdf20_test.go`), `wsl`/`err113`/`nlreturn`/`gci` in
  new test bodies, `dupl` between the intentional 1.7/2.0 test twins
  (`golden_test.go`, `fonttype0_test.go`↔`pdf20_test.go`,
  `semantic_converted_test.go`), `varnamelen`, `goconst`, one unused test
  helper. Fixed as mechanical zero-behavior-change edits: merged/removed
  directives, added `//nolint` lines consistent with repo style, blank
  lines for wsl, static sentinel errors for err113, local renames, line
  wrap, and the `pdfVersion20` constant for the 5 `"2.0"` literals.
  Final run 2026-08-15: `golangci-lint run ./...` **clean, exit 0**.)
- [x] `make test` → record outcome
  (Ran 2026-08-15: `go test ./...` exit 0, all packages `ok` —
  gowkhtmltopdf, cmd/gowkhtmltopdf, internal/app, internal/cli,
  internal/convert, internal/pdf, internal/layout, internal/settings, …)
- [x] 1.4 default proof: a convert/golden test without `--pdf-version` still starts with `%PDF-1.4`
  (`internal/convert/golden_test.go` `TestConvertPDF20GoldenNeedles` step 2
  converts `fixture-01-simple-invoice.html` with no version setting and
  asserts the `%PDF-1.4\n` prefix and absence of `%PDF-2.0`;
  `internal/pdf/policy_test.go` `TestDefaultNewDocumentAsserts14` asserts
  header `%PDF-1.4`, no `/Metadata`, no trailer `/ID`; both green in
  `make test`)
- [x] 2.0 opt-in proof: the phase-6 needle test is green
  (`internal/convert/golden_test.go` `TestConvertPDF20GoldenNeedles`:
  `%PDF-2.0\n%\xe2\xe3\xcf\xd3\n` header, trailer `/ID [ <hex> <hex> ]`,
  `/Type /Metadata /Subtype /XML`, `/Producer (gowkhtmltopdf 2.0)`, and
  no `pdfaid`/`pdfuaid` — green in `make test`; plus
  `TestConvertPDF20MultiPageTOCHF` structural TOC+HF parse)

### 8.2 Ledger hygiene

- [x] Every in-scope row in phases 1–7 is marked done only with evidence, or partial with a pointer
  (verified by grepping the plan folder for unchecked/partial checkbox
  patterns — zero matches across the 8 phase files and the canonical
  ledger; phases 1–6 were already complete with citations, 7 and 8 now
  are too)
- [x] Parent `00-canonical-pdf-20-plan.md` success-criteria boxes match reality
  (all six `[x]` with evidence; feature matrix reconciled to shipped
  behavior — see ledger note on catalog `/Version`)
- [x] `plans/0.2.2/README.md` status line updated
  (pdf-2.0-plan row now reads "**completed**")
- [x] No leftover gocorepdfengine / Zerodha / `engine/` paths in this folder
  (grep over `plans/0.2.2/pdf-2.0-plan/` returns only the canonical
  ledger's historical note that those phases "are gone"; no stale paths)

### 8.3 Issue boundary

- [x] #32 description / this plan still lists #33 work as out of scope
  (canonical ledger "Out of scope" explicitly lists PDF/A-4, PDF/UA-2,
  tagging, ICC, OutputIntent (#33); `pdf-a4-ua2-compliance-plan.md` stays
  `draft — not started`)
- [x] No catalog `/MarkInfo`, `/StructTreeRoot`, `/OutputIntents`, or `pdfaid` in default or 2.0 fixtures
  (catalog shape is pinned by regex in `internal/pdf/pdf20_test.go`
  `TestPDF20CatalogAndMetadataStream` — `<< /Type /Catalog /Metadata N 0 R
  /Pages N 0 R /Outlines N 0 R /PageMode /UseOutlines >>` — any of those
  keys would break the match; explicit negatives: no
  `pdfaid`/`pdfuaid`/`pdfaExtension` (pdf20_test.go:199-204,
  golden_test.go:859-862), no `/OutputIntents` or `/ICCBased`
  (pdf20_test.go:545-546), no `/ProcSet` on 2.0 pages
  (pdf20_test.go:552-553); 1.4 default asserts no
  `/MarkInfo`/`/StructTreeRoot`/`/StructParents`/`pdfuaid`
  (structure_test.go:432-435) and `CreateStructTreeRoot()` returns nil on
  unclaimed documents. All green in `make test`.)
- [x] #31 shared policy: `PDF17` is either implemented by the sibling plan or still reserved/rejected — not a silent 1.7 header
  (`PDF17` is implemented by the 1.7 plan and reuses the same
  `WriterPolicy` seam this plan extends with `PDF20`; `HeaderVersion()` /
  `ProducerVersion()` spell the version from the policy; `--pdf-version`
  with garbage errors `ErrInvalidPDFVersion` before any document exists
  (`TestPDFVersionNegativeValidation`); `policy.Validate()` rejects
  out-of-range versions with `ErrUnsupportedPDFVersion`)

### 8.4 What this does **not** close

- [x] Epic #29 remains open until #31 and #33 are decided or done
  (nothing in this plan marks #29 done; only #32 evidence is recorded)
- [x] PDF 1.7 feature completeness remains #31
  (this plan extends the #31 `WriterPolicy` with `PDF20`; 1.7 feature
  completeness stays owned by #31's ledgers)
- [x] PDF/A-4 and PDF/UA-2 remain #33
  (`WriterPolicy.Validate()` still returns
  `pdf.ErrConformanceProfilesUnsupported` for A-4/UA-2 profiles on 2.0
  (policy.go:175-184); the #33 ledger file stays `draft — not started`)

---

## Done when

Lint and test are recorded, 1.4 is still the default, 2.0 is opt-in
and proven, and the docs/plan no longer describe another product.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 1–7 | Issue #32 evidence; later #33 can consume the 2.0 path |
