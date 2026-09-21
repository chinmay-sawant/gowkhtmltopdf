# 94 - Output PDF position ranges (v0.2.7)

> **Parent:** `plans/0.2.7/README.md`
> **Status:** not started
> **Estimated effort:** L
> **Owner:** `internal/pdf` for the reader, then `internal/convert` for the fixture files
> **Depends on:** committed sample PDFs from `make samples`, and `pdf.ParseSemantic` stream inflation
> **Unblocks:** a fast check that a golden sample still puts the same words on the same page, in the same place
> **Number:** 94. Phases 86-93 belong to `plans/0.2.6/86-canonical-0.2.6-wasm.md`. Phase 89 there is the WASM preview. Do not reuse it.

---

## Overview

`TestGoldenCorpusAllFixtures` in `internal/convert/golden_test.go` converts every golden HTML file and checks four things. The PDF starts with `%PDF-` and ends with a reachable `xref`. A subset font is embedded. The page count sits inside `fixturePageBounds`. When a fixture lists needles, `pdf.ParseSemantic` finds those words in order.

That walk never asks where a word sits. `SemanticPage` in `internal/pdf/semantic.go` keeps the page text as one string, the page rectangle, font and image names, and link targets. It drops the x and y of each string.

The next-72 work added place checks, and they are real, but they answer a different question. `TestFixture64ContinuationRowsStartAtHeaderBottom` in `internal/layout/requested_fixture_regression_test.go` paints `fixture-64-next-72-props.html` through `paintGoldenFixture` and looks at layout ops. Y grows downward from the top of page 0. The test allows 0.75 pt. It never opens `output/fixture-64-next-72-props.pdf`. `paintGoldenFixture` also hard-codes a 12 mm margin for that one file and 28.35 pt for every other file. It does not read `@page`, and it does not attach `fixture-36-header.html`.

This suite checks the PDFs already in `output/`. A test opens the committed file, finds a few known strings, and checks the page plus a band around the measured baseline. The same band must hold for a PDF the test builds from the matching golden HTML in the same run. One band, two PDFs.

The file in `output/` is what a person opens. The fresh conversion is what the current code produces. They can drift. Checking only the file stays green after a layout bug until someone runs `make samples`. Checking only the fresh conversion ignores the committed file.

`output/README.md` is right that these files are not byte baselines. This suite does not compare bytes. It compares a page number and a 2 pt band. After `make samples`, update the band in the same change when a string actually moved. `make golden` staying green does not prove these bands.

## Coordinate contract

PDF user space. One unit is one point, 1/72 inch. The origin is the bottom-left corner of that page. Y grows up. A4 is 595.28 by 841.89, from `internal/settings/pagesize.go`. The writer emits `/MediaBox [0 0 width height]`.

Ordinary text is drawn by `drawText` in `internal/layout/paint.go`. It converts the layout point with `canvasToPDF`, then writes `x y Td` and `(text) Tj`. `Td` moves the pen. `Tj` draws the letters. Rotated text and fake-oblique text write `Tm` instead of `Td`. `Tm` is the same move, with room for a turn or a slant. Text-autospace writes `TJ`, which draws the letters and inserts extra gaps. An image is `width 0 0 height x y cm` followed by `Do`. `cm` sets the image's box. `Do` paints it. Numbers are written to 3 decimal places.

The band is plus or minus 2 pt on x and on y, the same slack `TestPageFirstMargins` uses in `internal/convert/page_first_test.go`. Do not widen it to hide a miss. A miss larger than 2 pt is a flag mismatch or a real move.

Page numbers in the anchor tables are 1-based, so they match a PDF viewer. The reader may store a 0-based index. The assertion converts.

Anchors are ASCII strings that appear as a literal `(...) Tj`. Hex Type0 text is not an anchor. `decodePDFHex` does not return the original characters, so a CJK search against page text is the wrong tool. Fixture 27 and fixture 64 still have ASCII labels. Avoid small-caps words. A 2026-09-21 note records that the same glyph can extract as `-`, U+00AD, or U+00A0 across runs.

Each fixture gets three text anchors.

- On a multi-page fixture, put them on different pages. Include the first page and the last page.
- On a one-page fixture, pick three different baselines. Title, a middle line, and a line near the bottom.
- The string must be unique on that page. The check uses the first hit on the named page.

Fixtures whose `fixturePageBounds` entry sets `images: true` also get one image anchor. That point is the lower-left of the `cm` that places the image, in page space, same 2 pt band.

`images: true` today: fixtures 07, 20, 36, 43, 47, 49, 50, 51, 53, 54, 59, 60, 61, 62, 64.

## How a number gets into the test

Measure, then copy. Do not invent a coordinate.

1. The phase 94.0 reader is already green on synthetic PDFs.
2. Point it at the committed file and print page, x, y, and the text snippet for the chosen string.
3. Copy those three numbers into the anchor table.
4. Run the fixture test. The committed file and the fresh conversion must both land inside the band.
5. If the fresh conversion misses, stop. Record the CLI flags that built the committed file. Do not edit the band until both PDFs agree.
6. Delete any print-only test before closing the phase. The permanent test is the band check.

The fresh conversion uses `requestForFixture` in `internal/convert/golden_test.go`. That is A4, 10 mm margins, backgrounds on, local files on, and `testdata/fonts` when that directory exists. Fixture 36 also attaches the header and footer HTML, which `requestForFixture` already does through `attachHFCompanions`. Do not call `paintGoldenFixture` from this suite.

## Reader rules

New file `internal/pdf/text_positions.go`. Leave `internal/pdf/semantic.go` alone. It is 1023 lines and its job is text, names, and the media box.

The reader must:

- Inflate `/Filter /FlateDecode` streams. `decodeSemanticStream` already does this for `ParseSemantic`. Default `UseCompression` is true in `internal/settings/settings.go`, and `make samples` does not turn it off. `firstPageTextPos` reads the raw stream and only the first `Td`. It cannot see these files.
- Track `Td` and `Tm`, then pair each with the following `Tj` or `TJ`.
- Walk into a Form XObject when a `Do` paints one, and add that form's placement to the text point. Transparency groups are Form XObjects. A reader that only scans the page stream will miss text drawn inside a group.
- Record an image `Do` lower-left from the `cm` in front of it.
- Return page index, text, x, and y. It does not need width or height. `Td` does not carry a box.

Synthetic proof lives in `internal/pdf/text_positions_test.go` before any fixture band is copied. Cases: plain `Td` plus `Tj`, the same bytes after Flate, a `Tm`, a `TJ`, and text inside a form whose reported point is on the page, not inside the form.

Shared test helper lives in `internal/convert/output_pos_test.go`. It opens `../../output/<path>` because `go test` runs with the package directory as the working directory. It converts the golden HTML with `requestForFixture` and checks both results. It also checks that both PDFs have the same page count, and that the count sits inside `fixturePageBounds`.

## File split

65 numbered sample PDFs sit at `output/fixture-*.pdf`. Fixture 01 through fixture 64, with no gaps, and fixture 29 twice: `fixture-29-float-beside-table.pdf` and `fixture-29-wpt-break-nested-float-print.pdf`. Six files of 10 would cover 60 and drop the rest. The cap is 10 fixtures in one test file, so the split is seven fixture files. The last holds five. An eighth file covers the version and compliance copies.

| Phase | Go test file | PDFs |
|-------|----------------|------|
| 94.1 | `internal/convert/output_pos_01_10_test.go` | fixture 01-10 |
| 94.2 | `internal/convert/output_pos_11_20_test.go` | fixture 11-20 |
| 94.3 | `internal/convert/output_pos_21_29_test.go` | fixture 21-28 plus both fixture 29 files |
| 94.4 | `internal/convert/output_pos_30_39_test.go` | fixture 30-39 |
| 94.5 | `internal/convert/output_pos_40_49_test.go` | fixture 40-49 |
| 94.6 | `internal/convert/output_pos_50_59_test.go` | fixture 50-59 |
| 94.7 | `internal/convert/output_pos_60_64_test.go` | fixture 60-64 |
| 94.8 | `internal/convert/output_pos_compliance_test.go` | 8 version and compliance PDFs |

Each new `.go` file stays at or under 2000 lines. `scripts/check-file-size.sh` scans test files. None of these files go on `scripts/file-size-allowlist.txt`. If a file passes 1600 lines while the anchors are being written, split it before adding another fixture.

One test function per PDF. `go test -run` takes a regex, so proof commands end the name with `$`, as in `-run 'TestOutputPosFixture01$'`. `TestOutputPosFixture01` does not match `TestOutputPosFixture10`. The `$` still keeps the command obvious, and it keeps `TestOutputPosFixture29` from also running `TestOutputPosFixture29Float`.

## Executive summary

| Phase | File | What it closes |
|-------|------|----------------|
| 94.0 | [phase-94.0-position-reader.md](phase-94.0-position-reader.md) | Reader, synthetic proofs, shared helper. No fixture bands yet. |
| 94.1 | [phase-94.1-fixtures-01-10.md](phase-94.1-fixtures-01-10.md) | 10 PDFs, fixture 01-10 |
| 94.2 | [phase-94.2-fixtures-11-20.md](phase-94.2-fixtures-11-20.md) | 10 PDFs, fixture 11-20 |
| 94.3 | [phase-94.3-fixtures-21-29.md](phase-94.3-fixtures-21-29.md) | 10 PDFs, fixture 21 through both 29s |
| 94.4 | [phase-94.4-fixtures-30-39.md](phase-94.4-fixtures-30-39.md) | 10 PDFs, fixture 30-39 |
| 94.5 | [phase-94.5-fixtures-40-49.md](phase-94.5-fixtures-40-49.md) | 10 PDFs, fixture 40-49 |
| 94.6 | [phase-94.6-fixtures-50-59.md](phase-94.6-fixtures-50-59.md) | 10 PDFs, fixture 50-59 |
| 94.7 | [phase-94.7-fixtures-60-64.md](phase-94.7-fixtures-60-64.md) | 5 PDFs, fixture 60-64, after the font-path check |
| 94.8 | [phase-94.8-version-compliance-samples.md](phase-94.8-version-compliance-samples.md) | 8 PDFs under `output/pdf-1.7`, `pdf-1.7-compliance`, `pdf-2.0`, `pdf-2.0-compliance` |
| 94.9 | [phase-94.9-closure.md](phase-94.9-closure.md) | `make size-check`, `make test`, `make golden` |

## Gate policy

Same rule as the rest of v0.2.7. Mid-phase commands are single-package:

```bash
go test ./internal/pdf -run 'TestTextPositions' -count=1
go test ./internal/convert -run 'TestOutputPos' -count=1
```

`make test` and `make golden` run only in phase 94.9. `make lint` is not part of this ledger. The owner runs it after. `make size-check` is allowed in 94.9 because it is the 2000-line gate and it is not `make lint`.

Never run bare `go test ./...`.

Do not grow `internal/layout/layout.go`, `internal/layout/inline_paint.go`, or `internal/imageout/imageout.go`. They are already on the allowlist.

## Out of scope

- [~] `output/python/`. Most fixture PDFs there are gitignored. The committed set is fixture 55, fixture 56, `architecture-diagram.pdf`, `invoice-inline.pdf`, and architecture diagrams in the four Python version dirs. Needs `CGO_ENABLED=1`. A later ledger can point here.
- [~] `output/wkhtmltopdf/`. Reference renders, not this engine.
- [~] `output/profiles/`. Bench logs, no PDFs.
- [~] `output/font-examples.pdf` and `output/chrome_ana.pdf`. `make samples` does not rewrite them.
- [~] `output/architecture-diagram.pdf`, `output/showcase-toc-hf-outline.pdf`, `output/wiki-ana-de-armas.pdf`. Real samples, different flags, not part of the 65 fixture bodies.
- [~] `testdata/golden/complex-css.html` and `testdata/golden/architecture-diagram.html`. The golden walker converts them. There is no matching `output/complex-css.pdf`, and the root `architecture-diagram.pdf` comes from `testdata/golden/api`, not from that HTML.
- [~] PNG samples. This suite reads PDF text positions.
- [~] Link rectangles. The writer stores `/Rect`, and `SemanticAnnot` drops it. Text position is the check this ledger promised.
- [~] Replacing `TestFixture64ContinuationRowsStartAtHeaderBottom` and the other layout geometry tests. Those stay. They use the layout ruler. This suite uses the PDF ruler.

## Dependencies

94.0 blocks every later phase. The fixture phases can run in order, and they can run one at a time. Do not start 94.2 until 94.1's helper has survived a real file, so a reader bug is fixed once.

94.7 waits on a written answer to the font-path question. `make samples` passes `--font-path` for `/usr/share/fonts/truetype/droid` when that directory exists, and for `testdata/fonts`. It does not pass `testdata/fonts/implemented-audit`. Fixture 60 and fixture 64 demos were built with that extra directory in earlier sessions. The committed PDF and a samples-default conversion can disagree. Resolve it in the 94.7.0 row before copying bands.

94.8 reuses the fixture 21 and fixture 56 anchor tables from 94.3 and 94.6. It starts after those two phases.

94.9 starts after 94.1 through 94.8 are checked.
