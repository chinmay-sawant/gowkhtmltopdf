## Context

**Parent epic:** #2 - [epic: post-MVP rendering quality - image mode, fonts, CSS for real sites](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/2)

**Siblings under #2:** #3 (image-mode PNG) · #4 (font spacing) · #5 (Wikipedia CSS) · #6 (multi-font)

After last-mile PDF fixes (zlib Flate, Catalog outlines, `/Widths` scaled to **1000 units/em**, Latin-1 `pdfString`), Latin text is readable and no longer “A c m e” letter-spaced by ~2×. Residual **spacing issues** remain in samples:

- Inter-word gaps can look uneven or too large  
- Word-by-word `BT`…`Tj`…`ET` (layout splits on `Fields`)  
- Fake bold (`TextRenderMode(2)` stroke) changes perceived width vs layout  
- Image mode still uses 5×7 fixed advances (see sibling image-mode issue)  
- Missing spaces / tight joins in some runs when layout trims trailing space inconsistently  

Evidence: `output/fixture-01-simple-invoice.pdf` / `.png`, showcase and multi-page fixtures under `output/`.

## Scope (in)

1. Audit layout text ops (`internal/layout/inline.go`) - word splitting, space width, trailing space on runs.
2. Align **paint** advances with **layout** advances for PDF (Liberation Sans) and document any intentional differences.
3. Revisit fake bold: compensate advance or use a real bold face (coordinate with multi-font issue).
4. Consider coalescing adjacent text ops on the same baseline into fewer `Tj` strings (smaller streams, fewer gaps).
5. Add regression tests (golden layout widths / PDF content-stream heuristics) for a fixture known to show spacing bugs.
6. Update `documentation/compatibility-matrix.md` typography notes.

## Out of scope

- Full kerning/OpenType shaping (HarfBuzz-class)  
- CJK / complex scripts (Unicode fonts issue)  
- Full CSS `letter-spacing` / `word-spacing` / `text-align: justify` parity unless already half-implemented  

## Success criteria

- [ ] Visually even word spacing on fixture-01 and fixture-16 (PDF) in a real viewer  
- [ ] No regression of the 1000-unit `/Widths` fix (no return of double letter-spacing)  
- [ ] Tests cover at least one spacing regression  
- [ ] Docs mention remaining limits (kerning, justify, etc.)

## Plan

- Parent epic: #2  
- Code: `internal/layout/inline.go`, `internal/layout/paint.go`, `internal/pdf/content.go` / `fontpdf.go`  
- Sibling: multi-font (real bold/italic faces)

## References

- Relates to #2 (parent epic)
- Prior fix: commit history “correct glyph advances and Latin-1 text encoding”  
- Samples: `output/fixture-01-simple-invoice.pdf`  
- Paint: `drawText` fake bold in `internal/layout/paint.go`

