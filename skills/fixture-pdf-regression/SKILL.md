---
name: fixture-pdf-regression
description: >
  Diagnose gowkhtmltopdf golden-fixture PDF layout regressions by comparing an
  original/good PDF to a current output PDF, mapping the broken region to HTML/CSS
  and Go layout/paint code, then tracing git history without worktrees. Use when
  a fixture PDF looks wrong, a table/chip/page breaks after a Go change, the user
  says "compare fixture PDFs", "diff checker", or runs /fixture-pdf-regression.
---

# Fixture PDF regression

Find why a golden fixture PDF diverged from a known-good sample, then fix the
engine (or flag a fixture/CSS issue). **Do not use git worktrees.**

## Inputs

Collect before digging:

1. **Current PDF** (usually `output/fixture-NN-....pdf`)
2. **Good PDF** (user path, prior commit sample, or `og_*.pdf`)
3. **Symptom** (page number, region name, what looks wrong)

If the current PDF may be stale: rebuild `bin/gowkhtmltopdf` and regenerate that
one fixture with `--allow-local-files -o output/<name>.pdf testdata/golden/<name>.html`.

## Phase 1 - Diff the PDFs

Prefer PyMuPDF (`fitz`) + Pillow. Poppler is optional.

1. Render the suspect page(s) from both PDFs at ~1.5-2.5x.
2. Crop the broken region; write a labeled side-by-side.
3. Measure structure, not vibes:
   - page count
   - word `(x,y)` for labels in the broken region
   - fill rects (chip/header backgrounds): `x, y, w, h`, fill RGB
   - rounded paths: curve segment count and approximate corner radius from Bezier spans
4. Classify the delta:
   - **geometry** (column crush, row overlap, wrong radius)
   - **pagination** (orphan line, +1 blank page, content shift)
   - **paint** (missing fill, extra strokes)
   - **reflow** (wrap/nowrap, chip height)

Keep a short metric table (good vs current). That table is the progress bar.

Helper (optional): `skills/fixture-pdf-regression/scripts/compare_pages.py`.

## Phase 2 - Map to HTML/CSS

1. Grep `testdata/golden/` for unique text from the broken region.
2. Open the fixture `.html` and linked `.css`.
3. Note the exact selectors, table/flex/grid structure, and "fallback-first"
   modern CSS (`clamp()`, `oklch()`, logical props) that the lite engine may
   drop or mishandle.
4. Quote the rules that control the broken geometry (`width`, `white-space`,
   `border-radius`, `padding`, column layout).

## Phase 3 - Map to Go

1. Grep `internal/layout` (and paint/pdf if needed) for the property or feature
   named by the CSS.
2. Read the resolve → used-value → paint path with `file:line` citations.
3. Prefer the smallest owning function (cascade allowlist, radius scale, table
   cell measure, etc.) over rewriting a large file.

## Phase 4 - History without worktrees

Use read-only git on the current tree only:

```bash
git log --oneline -S'<symbol>' -- internal/layout/<file>.go
git log -p -S'<symbol>' -- internal/layout/<file>.go | head
git blame -L <start>,<end> internal/layout/<file>.go
git show <commit>:<path> | rg -n '<symbol>'
```

**Do not** `git worktree add`, checkout other commits in-place, or mutate the
index for bisect. Infer the breaking change from blame/`-S` diffs and the
metric table. If you must rebuild an old binary, use `git show commit:file`
piped into a temp copy outside the repo, or ask the user; do not attach a
worktree.

Name the first commit that introduced the mismatch class (cascade, clamp gate,
radius scale, measure, pagination).

## Phase 5 - Fix and prove

1. Write or extend a focused unit/regression test that fails on the bug class.
2. Patch the owning Go code (keep fixture HTML/CSS unless the fixture itself is wrong).
3. `go test` the touched package with the new test.
4. Regenerate the fixture PDF into `output/`.
5. Re-run Phase 1 metrics; good and current must match on the classified delta.
6. Note any remaining unrelated deltas (extra pages elsewhere, etc.) without
   burying the fixed claim.

## Output shape

Report:

- metric table (before → after)
- HTML/CSS selectors involved
- Go `file:line` and the blamed commit
- what changed in the fix
- path of the regenerated PDF
