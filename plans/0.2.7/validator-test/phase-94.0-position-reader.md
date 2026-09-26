# Phase 94.0: Position reader and shared helper

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** nothing in this ledger
> **Unblocks:** phases 94.1 through 94.9

---

## Overview

Build the thing that can answer "which page, and what x and y" for a string inside a compressed PDF. Prove it on synthetic files. Land the helper the fixture tests will call. Do not copy a band from `output/` in this phase.

The contract, the 2 pt band, and the file split live in the parent. Follow those. Do not restate a second contract here.

## Checklist

### 94.0.1 Reader

- [ ] 94.0.1.1 Add `internal/pdf/text_positions.go` with a function that returns one record per text show: page index, text, x, y, in PDF user space. Inflate Flate streams. Track `Td` and `Tm`. Pair them with the following `Tj` or `TJ`.
- [ ] 94.0.1.2 When `Do` paints a Form XObject, walk that form's stream and add the form's placement to the text point. The returned x and y are on the page.
- [ ] 94.0.1.3 When `Do` paints an image, record the lower-left from the `cm` in front of it, in page space.
- [ ] 94.0.1.4 Leave `internal/pdf/semantic.go` unchanged. Do not add fields to `SemanticPage`.

### 94.0.2 Synthetic proofs

File: `internal/pdf/text_positions_test.go`.

- [ ] 94.0.2.1 Uncompressed content stream, one `x y Td` and one `(Hello) Tj`. The reader returns that x, that y, and `Hello`.
- [ ] 94.0.2.2 The same stream wrapped in `/Filter /FlateDecode`. Same result. This is the case `output/` actually uses.
- [ ] 94.0.2.3 A `Tm` with a translation, then `Tj`. The reader returns the translation, not 0,0.
- [ ] 94.0.2.4 A `TJ` array that shows an ASCII word. The reader returns that word and the point in force before the array.
- [ ] 94.0.2.5 Text inside a Form XObject placed with a non-zero `cm`. The reported point includes the form placement.
- [ ] 94.0.2.6 An image `Do` with `w 0 0 h x y cm`. The recorded lower-left is that x, y.
- [ ] 94.0.2.7 Proof command, exit 0:

```bash
go test ./internal/pdf -run 'TestTextPositions' -count=1
```

### 94.0.3 Convert helper

File: `internal/convert/output_pos_test.go`. No fixture bands in this file.

- [ ] 94.0.3.1 Helper opens `../../output/<rel>` and also converts the named golden HTML through `requestForFixture`.
- [ ] 94.0.3.2 Helper checks both PDFs: same page count, count inside `fixturePageBounds`, and each anchor's first hit on the named 1-based page lies within 2 pt of the recorded x and y.
- [ ] 94.0.3.3 Helper accepts an optional `PdfVersion` or `PdfProfile` so phase 94.8 can reuse it. Tokens match the Makefile: `1.7`, `2.0`, `a3a-ua1`, `a4-ua2`.
- [ ] 94.0.3.4 `wc -l` on `text_positions.go`, `text_positions_test.go`, and `output_pos_test.go`. Each file is at or under 2000 lines. Record the three counts in this row when it closes.

## Proof template

```text
text_positions.go lines: N
text_positions_test.go lines: N
output_pos_test.go lines: N
go test ./internal/pdf -run 'TestTextPositions' -count=1: exit 0
```
