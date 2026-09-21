# Phase 94.8: Version and compliance samples

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.3.1 and 94.6.7, because this phase reuses those anchor tables
> **Unblocks:** 94.9

---

## Overview

`make samples` writes the same two HTML files four ways. The loop is in the Makefile, just after the fixture loop. Two basenames, four directories, eight PDFs.

| Directory | Flag | PDF header the test also checks |
|-----------|------|----------------------------------|
| `output/pdf-1.7/` | `--pdf-version 1.7` | `%PDF-1.7` |
| `output/pdf-1.7-compliance/` | `--pdf-profile a3a-ua1` | `%PDF-1.7` |
| `output/pdf-2.0/` | `--pdf-version 2.0` | `%PDF-2.0` |
| `output/pdf-2.0-compliance/` | `--pdf-profile a4-ua2` | `%PDF-2.0` |

Basenames in every directory: `fixture-21-detailed-report.pdf`, `fixture-56-architecture-diagram.pdf`.

File `internal/convert/output_pos_compliance_test.go`. The fresh conversion starts from `requestForFixture` and sets `Global.PdfVersion` or `Global.PdfProfile` to the token in the table.

Try the fixture 21 and fixture 56 bands from 94.3 and 94.6 first. If a version or profile render stays inside those bands, the test calls the same table and this phase says so. If a string moves by more than 2 pt, record a separate band for that directory. Do not widen the original band.

`compliance/README.md` names make targets that are not in the Makefile. Do not run them. This phase does not call `compliance/verify_pdfs.sh` either. That script checks PDF/A flavour, not text position.

## Checklist

- [ ] 94.8.1 `output/pdf-1.7/fixture-21-detailed-report.pdf`. Header `%PDF-1.7`. Bands from 94.3.1, or a recorded separate table. `TestOutputPosPDF17Fixture21`.
- [ ] 94.8.2 `output/pdf-1.7/fixture-56-architecture-diagram.pdf`. Header `%PDF-1.7`. Bands from 94.6.7, or a recorded separate table. `TestOutputPosPDF17Fixture56`.
- [ ] 94.8.3 `output/pdf-1.7-compliance/fixture-21-detailed-report.pdf`. Header `%PDF-1.7`. Profile `a3a-ua1`. `TestOutputPosPDF17ComplianceFixture21`.
- [ ] 94.8.4 `output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf`. Header `%PDF-1.7`. Profile `a3a-ua1`. `TestOutputPosPDF17ComplianceFixture56`.
- [ ] 94.8.5 `output/pdf-2.0/fixture-21-detailed-report.pdf`. Header `%PDF-2.0`. `TestOutputPosPDF20Fixture21`.
- [ ] 94.8.6 `output/pdf-2.0/fixture-56-architecture-diagram.pdf`. Header `%PDF-2.0`. `TestOutputPosPDF20Fixture56`.
- [ ] 94.8.7 `output/pdf-2.0-compliance/fixture-21-detailed-report.pdf`. Header `%PDF-2.0`. Profile `a4-ua2`. `TestOutputPosPDF20ComplianceFixture21`.
- [ ] 94.8.8 `output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf`. Header `%PDF-2.0`. Profile `a4-ua2`. `TestOutputPosPDF20ComplianceFixture56`.
- [ ] 94.8.9 `wc -l internal/convert/output_pos_compliance_test.go` is at or under 2000. Record the count.

Proof for each row: `go test ./internal/convert -run '<TestName>$' -count=1` exit 0.
