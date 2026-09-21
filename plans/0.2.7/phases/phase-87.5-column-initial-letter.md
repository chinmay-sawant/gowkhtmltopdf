# Phase 87.5: Column height/wrap + initial-letter

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** done
> **Estimated effort:** M–L (5 properties)
> **Owner:** `internal/layout` (multicol + inline)
> **Depends on:** 87.4 helpful for wrap metrics; float exclusion ideas from `float.go`
> **Unblocks:** 87.6 shape exclusion reuse
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

CSS Multicol 2 `column-height` / `column-wrap`, plus CSS Inline `initial-letter*`.
`:first-letter` is rejected in the CSS selector parser today, so initial-letter must
work on a real element (e.g. leading `<span>`) or the engine synthesizes a first
letter box. Record the chosen authoring rule in the matrix.

Do not grow `layout.go`. Put drop-cap logic in `inline_initial_letter.go`.

## Properties (5)

`column-height`, `column-wrap`, `initial-letter`, `initial-letter-align`,
`initial-letter-wrap`

## Seams

| Piece | Location |
|-------|----------|
| Multicol apply | `style_multicol_props.go` (`applyMulticolGroup`) |
| Multicol layout | `multicol.go` (height clamp + wrap rows) |
| Float exclusion (reuse pattern) | `float.go`; line shorten `inline.go` |
| First-letter selector | rejected `internal/css/selector_parser.go:431-432` |
| Initial-letter apply/consumer | `style_initial_letter_props.go` + `inline_initial_letter.go` |

## New files

- `style_multicol_props.go` (extracted multicol apply + height/wrap)
- `style_initial_letter_props.go` + `inline_initial_letter.go`

## Checklist

### 87.5.1 column-height / column-wrap

- [x] 87.5.1.1 Apply arms after extracting multicol apply out of `style_properties.go` if needed.
- [x] 87.5.1.2 `column-height`: clamp column boxes before spilling to more columns (`multicol.go`).
- [x] 87.5.1.3 `column-wrap`: multi-row column packing; if full model is too large, ship height-only and leave wrap Partial/Unsupported with note.
- [x] 87.5.1.4 Tests: `TestColumnHeightCapsColumn`, `TestColumnWrapCreatesRow` (if shipped).

### 87.5.2 initial-letter*

- [x] 87.5.2.1 Apply three props; document authoring (real element vs synthesized first letter).
- [x] 87.5.2.2 `initial-letter: N` sizes/sinks first letter across N lines; exclusion shortens following lines (float-like, not necessarily `float:`).
- [x] 87.5.2.3 `initial-letter-align` / `initial-letter-wrap`: implement lite or Partial (Chrome no on both).
- [x] 87.5.2.4 Tests: `TestInitialLetterSpansThreeLines`, geometry asserts next-line x > letter box.

### 87.5.R batch gate (package only)

- [x] 87.5.R.1 `go test ./internal/layout -run 'TestColumnHeight|TestColumnWrap|TestInitialLetter' -count=1` exit 0.
- [x] 87.5.R.2 Flip packets only for shipped consumers. **No `make test` / `make lint`.**

## Out of scope

CSS Shapes (87.6). Page floats. Enabling `:first-letter` selector (separate CSS work if desired).
