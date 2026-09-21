# Phase 94.7: Fixtures 60-64

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.0, and a written font-path answer in 94.7.0
> **Unblocks:** 94.9

---

## Overview

File `internal/convert/output_pos_60_64_test.go`. Five PDFs. This is the short file. The earlier guess of six files of ten would have stopped at fixture 60 and left these out.

Fixtures 60, 61, 62, and 64 set `images: true`. Each of those gets one image anchor on top of the three text anchors.

The existing fixture 64 layout tests stay where they are. `TestFixture64RepeatedHeaderKeepsCollapsedGridOnContinuationPage`, `TestFixture64TextAutospaceAddsIdeographAlphaGap`, and `TestFixture64ContinuationRowsStartAtHeaderBottom` use layout ops and a y-down ruler. This phase checks the committed PDF with the PDF ruler.

## Checklist

- [ ] 94.7.0 Font path, before any band is copied. `make samples` in the Makefile passes `--font-path` for droid when that directory exists, and for `testdata/fonts`. It does not pass `testdata/fonts/implemented-audit`. Write down the flags that match the committed `output/fixture-60-implemented-props-a.pdf` and `output/fixture-64-next-72-props.pdf`. The fresh conversion in this file uses those flags. If the two PDFs disagree by more than 2 pt under samples-default flags, this row stays open until the test uses the flags that built the file. Record the command here.
- [ ] 94.7.1 `output/fixture-60-implemented-props-a.pdf`. Three ASCII text anchors plus one image. `TestOutputPosFixture60`. Uses the font path from 94.7.0.
- [ ] 94.7.2 `output/fixture-61-implemented-props-b.pdf`. Three text anchors plus one image. `TestOutputPosFixture61`.
- [ ] 94.7.3 `output/fixture-62-implemented-props-c.pdf`. Three text anchors plus one image. `TestOutputPosFixture62`.
- [ ] 94.7.4 `output/fixture-63-page-level-demos.pdf`. Envelope is 6 to 8 pages. Anchor on the first page and the last page. `TestOutputPosFixture63`.
- [ ] 94.7.5 `output/fixture-64-next-72-props.pdf`. Three ASCII text anchors, including `NEXT-72-PROPS` if the reader finds it as a literal `Tj`, plus one image. One anchor on the first page and one on the last page. Do not use the string `汉A汉A汉` as an anchor. `TestOutputPosFixture64`. Uses the font path from 94.7.0.
- [ ] 94.7.6 `wc -l internal/convert/output_pos_60_64_test.go` is at or under 2000. Record the count.

Proof for each fixture row: `go test ./internal/convert -run 'TestOutputPosFixtureNN$' -count=1` exit 0.
