## Summary

Implements opt-in PDF 1.7 compliance profiles (PDF/A-3a, PDF/UA-1, and dual `a3a-ua1`) and hardens the tagged-PDF path so stricter machine checkers accept the output. Fixes Arlington CIDFontType2 FontName mismatches and PDF/UA-1 structure-tree invalidity that caused Matterhorn 01-005 unmarked-content failures even when veraPDF passed.

## Motivation / context

- Plans: `plans/0.2.2/pdf-1.7-compliance-plan/`
- Epic: PDF 1.7 / 2.0 support (#29)
- PDF 1.7 support: #31
- Validators: veraPDF, in-repo `structure_tree_check.py`, avalpdf; external octopdf (Arlington + PDF/UA-1)

## Changes

### Compliance profiles and writer

- Opt-in PDF/A-3a + PDF/UA-1 claiming metadata, OutputIntent / sRGB, MarkInfo, and structure tree under `--pdf-profile` / `WithPDFProfile`
- PDF 1.7 writer policy, fonts, subsetting, and convert/layout bridges for tagged content

### PDF/UA-1 structure tree hardening

- Store marked content as `(page, MCID)` pairs; emit **MCR** dictionaries when a structure element spans multiple pages (ISO 32000-1 §14.7.4.2)
- Stop the layout tagging fallback from accumulating one document-wide mega-`/P` with duplicate bare MCIDs (structurally invalid; caused octopdf UA-1 fail + Matterhorn 01-005)
- Contiguous unmapped text runs share a `/P`; semantic or non-text ops end the run
- Stronger marking of real content vs `/Artifact` (pagination / background / layout) in paint and HF paths
- Link structure / OBJR wiring refinements

### Arlington / Type0 fonts

- CIDFontType2 FontDescriptor `/FontName` now equals parent `/BaseFont` (`NameIdentity`), fixing Arlington `FontDescriptorCIDType2.FontName`
- Regression: `TestType0FontDescriptorNameMatchesBaseFont`

### Fixtures and tests

- Regenerated `output/pdf-1.7-compliance/fixture-21-detailed-report.pdf` and `fixture-56-architecture-diagram.pdf` with `a3a-ua1`
- Multi-page MCR, content-marked completeness, structure, and phase6/link tests

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Minor: slightly more structure elements (many small `/P` instead of one mega-P); MCR only when multi-page |
| **Memory** | Negligible (content refs store page pointer per MCID) |
| **Behavior / correctness** | Tagged PDFs under UA-1 profiles become structure-valid for stricter checkers; default unprofiled output unchanged |
| **API / CLI** | No new flags in this fix commit; profiles already exposed via `--pdf-profile` / `WithPDFProfile` |
| **Dependencies** | None |
| **Binary size / build time** | Unchanged |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Profile remains opt-in; no default behavior change |

## Test plan

- [x] `go test ./internal/pdf/ ./internal/layout/ ./internal/convert/ -count=1`
- [x] `TestMultiPageStructElemUsesMCR`, `TestType0FontDescriptorNameMatchesBaseFont`, `TestPDFUA1ContentMarkedCompleteness`
- [x] `COMPLIANCE_FLAVOURS=3a,ua1 ./compliance/verify_pdfs.sh` on both compliance fixtures (veraPDF PASS 3a+ua1; structure-tree PASS)
- [x] Regenerated compliance PDFs; no duplicate bare multi-page MCIDs; Arlington FontName rule satisfied on Type0 faces
- [ ] Optional: re-upload `output/pdf-1.7-compliance/*.pdf` to octopdf for human confirmation

### Commands

```sh
go test ./internal/pdf/ ./internal/layout/ ./internal/convert/ -count=1
go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
./bin/gowkhtmltopdf --pdf-profile a3a-ua1 --enable-local-file-access \
  testdata/golden/fixture-56-architecture-diagram.html \
  output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf
COMPLIANCE_FLAVOURS=3a,ua1 ./compliance/verify_pdfs.sh \
  --pdf output/pdf-1.7-compliance/fixture-21-detailed-report.pdf \
  --pdf output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf
```

## Screenshots / sample output

```
PASS PDF/A-3a / PDF/UA-1 (veraPDF) on fixture-21 and fixture-56
PASS structure-tree: ParentTree TD ownership, no TR mismatches
Arlington: FontDescriptorCIDType2.FontName == parent BaseFont (0 mismatches)
Structure: no document-wide mega-P with duplicate bare MCIDs
```

## Related issues

- Closes #31
- Relates to #29

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body under `plans/PR/pr-pdf-17-compliance-ua1-arlington.md`

## Follow-ups (out of scope)

- PDF/A-4 / PDF/UA-2 on PDF 2.0 (#33)
- Softer avalpdf heuristics (empty table cells) if product wants strict mode
- Optional Sect grouping to quiet “incorrectly nested groups” warnings on large Document trees

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed
