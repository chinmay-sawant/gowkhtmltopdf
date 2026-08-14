## Summary

Ships opt-in **PDF 2.0** version support (#32) and the dual **PDF/A-4 + PDF/UA-2** conformance profile on that base (#33). Default output remains unclaimed **PDF 1.4**. Version alone never claims archival or accessibility; claiming XMP, OutputIntent, structure namespaces, and tagging require an explicit `--pdf-profile` / `WithPDFProfile`.

Also hardens dual-profile destinations so machine checkers agree: classic page `/D` for Arlington/PDF/A page Dest validity, plus `/SD` structure destinations for PDF/UA-2 clause 8.8.

---

## Motivation / context

- **Epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29) — newer PDF versions and compliance
- **Version path:** [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32) — PDF 2.0 (ISO 32000-2) opt-in
- **Conformance path:** [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33) — PDF/A-4 + PDF/UA-2
- **Plans:**
  - `plans/0.2.2/pdf-2.0-plan/` (version ledger — completed)
  - `plans/0.2.2/pdf-2.0-plan/pdf-a4-ua2-compliance-plan.md` (A-4+UA-2 ledger — completed)
- **Reuses:** PDF 1.7 compliance machinery from #31 (XMP, ICC, OutputIntent, structure tree, layout tagging bridge)
- **Validators:** in-tree veraPDF (`-f 4`, `-f ua2`); dual named dests target Arlington DestXYZ + octopdf PDF/A page validity

Highest compliance by version:

| Base | Archival | Accessibility | Opt-in |
|------|----------|---------------|--------|
| PDF 1.7 (shipped) | PDF/A-3a | PDF/UA-1 | `--pdf-profile a3a-ua1` |
| **PDF 2.0 (this PR)** | **PDF/A-4** | **PDF/UA-2** | **`--pdf-profile a4-ua2`** |

---

## Changes

### PDF 2.0 version path (#32)

- `WriterPolicy` / `PDF20`: header `%PDF-2.0` + binary comment, trailer `/ID`, UTF-8 document strings, non-claiming XMP (`dc:format`, `pdf:Producer`, dates — **no** `pdfaid` / `pdfuaid`)
- Settings / CLI / library: `--pdf-version 2.0`, `WithPDFVersion("2.0")`, `ParsePDFVersion`
- Convert pipeline maps `PdfGlobal` → `pdf.WriterPolicy` in one place (`PolicyForGlobal`)
- Goldens / structural tests prove `%PDF-2.0` opt-in and default still `%PDF-1.4`
- Docs stress **version ≠ conformance**

### PDF/A-4 archival (#33)

- Profiles: `a4` / `PDF/A-4` (and aliases); dual `a4-ua2` / `PDF/A-4+PDF/UA-2`
- XMP: `pdfaid:part=4`, `pdfaid:rev=2020`
- OutputIntent + sRGB ICC (`/N 3`) and Gray ICC (`/N 1`)
- Page resources: `/DefaultRGB` and `/DefaultGray` → ICCBased
- Trailer **omits `/Info`** under A-4 only (descriptive metadata lives in XMP)
- Wrong base (1.4 / 1.7 + A-4 profile) → `ErrConformanceRequiresPDF20`

### PDF/UA-2 accessibility (#33)

- Profiles: `ua2` / `PDF/UA-2`; dual with A-4 as above
- XMP: `pdfuaid:part=2`, `pdfuaid:rev=2024` + `pdfaExtension` when dual-claiming
- Structure: `/Type /Namespace` (`http://iso.org/pdf2/ssn`), StructTreeRoot `/Namespaces`, Document `/NS`
- Catalog: `/Lang`, `/MarkInfo`, `/StructTreeRoot`, `/ViewerPreferences << /DisplayDocTitle true >>`
- **ListNumbering** on `/L` when LIs carry `/Lbl` (`Disc` for `ul`, `Decimal` for `ol`) — UA-2 8.2.5.25
- Reuses UA-1 structure types and layout tagging bridge on the 2.0 base

### Dual named destinations (A-4 + UA-2 + Arlington)

Earlier structure-only Dest arrays (`Dest[0] = StructElem`) passed veraPDF UA-2 but failed:

- PDF/A-4 / octopdf: “GoTo action specifies invalid page destination”
- Arlington: `DestXYZArray[0] is not PageObject` on outline Dest paths

**Fix:** PDF/UA-2 destinations use PDF 2.0 named dest dictionaries:

```text
Catalog /Names /Dests → (Dn) <<
  /D  [ <Page> /XYZ x y null ]      ← Arlington / PDF/A page Dest
  /SD [ <StructElem> /XYZ x y null ] ← UA-2 clause 8.8 structure dest
>>
Outline / Link: /Dest (Dn)
Outline items also: /SE <heading StructElem>
```

No Document-without-`/Pg` fallback for structure destinations.

### Settings / CLI / library

| Surface | Values |
|---------|--------|
| `--pdf-version` | `1.4` (default), `1.7`, `2.0` |
| `--pdf-profile` | `a3a-ua1` / `a3a` / `ua1` (imply 1.7); **`a4-ua2` / `a4` / `ua2` (imply 2.0)** |
| `WithPDFVersion` / `WithPDFProfile` | Same strings on `PDFRequest` |
| Converter global keys | `pdfversion`, `pdfprofile` |

### Samples and plans

- `output/pdf-2.0/` — unclaimed 2.0 (`fixture-21`, `fixture-56`)
- `output/pdf-2.0-compliance/` — `a4-ua2` dual profile (same fixtures)
- `output/README.md` documents regeneration commands
- Plan ledgers and `plans/0.2.2/README.md` marked completed with evidence
- User-facing matrix / deferred / CLI / README / landscape updated (no longer “deferred to #33”)

### Lint / hygiene

- Follow-up commit: golangci-lint clean (`lll`, `cyclop`/`nestif`, `wsl`, `varnamelen`, `gocyclo`, `goconst`, `dupl`)

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Negligible for default path. Dual named dests add a few objects per outline/link under UA-2 profiles only. |
| **Memory** | Small: named dest slice + structure dest refs only when PDF/UA-2 is active. |
| **Behavior / correctness** | Default unclaimed 1.4 unchanged. Opt-in 2.0 and `a4-ua2` emit claiming/tagged files; dual Dest form satisfies veraPDF 4+ua2 and page-based DestXYZ. |
| **API / CLI** | New accepted values for existing flags/setters (`2.0`, `a4`, `ua2`, `a4-ua2`, …). Invalid values remain hard errors. |
| **Dependencies** | None (still pure Go + allowlisted modules; veraPDF optional external binary). |
| **Binary size / build time** | Unchanged; sample PDFs committed under `output/pdf-2.0*`. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None for default users | Default remains PDF 1.4 unclaimed |
| Callers that expected `a4` / `ua2` to fail as “unsupported” | Those profiles now succeed and imply PDF 2.0 |
| PDF/A-4 consumers of dual files | Trailer has no `/Info`; use Catalog `/Metadata` XMP |
| Dest readers that only understand page Dest arrays | Named dests resolve via `/Names /Dests`; `/D` remains a page XYZ array |

---

## Test plan

- [x] `make test` / `go test ./... -count=1`
- [x] `make lint` (golangci-lint v1.64.8)
- [x] `make build` → `bin/gowkhtmltopdf`
- [x] Writer/unit: PDF20 header, trailer `/ID`, dual named dests, profile matrix, A-4 Info omission, UA-2 Namespace / ListNumbering
- [x] Convert goldens / compliance needles for 1.7 dual and 2.0 dual
- [x] Optional `TestVeraPDFOptionalValidation` — `-f 3a`, `-f ua1`, `-f 4`, `-f ua2` (skips if binary missing)
- [x] `./compliance/run_verapdf.sh --both` on `output/pdf-2.0-compliance/fixture-{21,56}-*.pdf` — **PASS PDF/A-4 + PDF/UA-2** (veraPDF 1.30.2)
- [x] Regenerated unclaimed `output/pdf-2.0/` samples (`%PDF-2.0`, no `pdfaid`/`pdfuaid`)
- [ ] Optional: re-upload `output/pdf-2.0-compliance/*.pdf` to octopdf for human Arlington / DestXYZ confirmation

### Commands

```sh
make lint
make test
make build

# Unclaimed PDF 2.0
./bin/gowkhtmltopdf --pdf-version 2.0 --enable-local-file-access \
  testdata/golden/fixture-21-detailed-report.html \
  output/pdf-2.0/fixture-21-detailed-report.pdf

# Dual PDF/A-4 + PDF/UA-2
./bin/gowkhtmltopdf --pdf-profile a4-ua2 --enable-local-file-access \
  testdata/golden/fixture-21-detailed-report.html \
  output/pdf-2.0-compliance/fixture-21-detailed-report.pdf
./bin/gowkhtmltopdf --pdf-profile a4-ua2 --enable-local-file-access \
  testdata/golden/fixture-56-architecture-diagram.html \
  output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf

./compliance/run_verapdf.sh --both \
  output/pdf-2.0-compliance/fixture-21-detailed-report.pdf \
  output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf
```

---

## Screenshots / sample output

```
%PDF-2.0  (unclaimed and dual)

veraPDF 1.30.2
==> output/pdf-2.0-compliance/fixture-21-detailed-report.pdf
PASS PDF/A-4: compliant (109 rules, …)
PASS PDF/UA-2: compliant (1727 rules, …)
PASSED: all checks

==> output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf
PASS PDF/A-4: compliant (109 rules, …)
PASS PDF/UA-2: compliant (1727 rules, …)
PASSED: all checks

Named dest form (dual):
  /Dest (D1)
  (D1) << /D [ <Page> /XYZ … ] /SD [ <StructElem> /XYZ … ] >>
  Outline also /SE <heading StructElem>

go test ./...  — ok
make lint      — clean
```

---

## Related issues

- Closes #32
- Closes #33
- Relates to #29

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues filled with real ticket IDs
- [x] Filled body under `plans/PR/pr-pdf-20-a4-ua2-compliance.md`

---

## Follow-ups (out of scope)

- Prefer per-target structure Dest (heading/id StructElem) over first-page-MCID fallback for richer AT navigation
- PDF/A-4e / A-4f, multiple OutputIntents
- Optional PAC / Matterhorn human checklist on very large multi-page docs
- Softer avalpdf heuristics if product wants strict mode on empty table cells
- Byte-stable CLI creation times (determinism product work; not part of #32/#33)

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented (cli.md, library-api.md, matrix, deferred)
- [ ] Default still unclaimed PDF 1.4 without profile/version flags
- [ ] Dual named dests: `/D` is Page, `/SD` is StructElem with page association
- [ ] Sample PDFs under `output/pdf-2.0*` are intentional artifacts (not golden byte baselines)
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed
