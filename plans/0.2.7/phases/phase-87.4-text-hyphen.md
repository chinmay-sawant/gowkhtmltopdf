# Phase 87.4: Text box/spacing + hyphenation limits

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** done
> **Estimated effort:** M–L (13 properties)
> **Owner:** `internal/layout` (inline / measure)
> **Depends on:** none hard; benefits from 87.2 if OT spacing features overlap
> **Unblocks:** 87.5 drop-cap wrap quality
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Add text-box trim/edge, autospace/spacing, and hyphenation limit / hanging
punctuation consumers. `hyphens` / `hyphenate-character` are now read for
authored soft-hyphen (U+00AD) breaks; dictionary `auto` stays out.

Do not grow `style_properties.go` or `layout.go`. Prefer new `style_text_*` and
`inline_*` files; register apply on `styleGroups`.

## Properties (13)

**Text (8):** `text-autospace`, `text-box`, `text-box-edge`, `text-box-trim`,
`text-fit`, `text-group-align`, `text-spacing`, `text-spacing-trim`

**Hyphenation (5):** `hanging-punctuation`, `hyphenate-limit-chars`,
`hyphenate-limit-last`, `hyphenate-limit-lines`, `hyphenate-limit-zone`

## Seams

| Piece | Location |
|-------|----------|
| Wave3 text apply | `style_text_props.go:9-75` |
| Line metrics half-leading | `inline.go:1065-1114` |
| Pack / soft split | `inline.go:284-349`, `725-838` |
| Measure runs | `inline_paint.go:1555+` |
| Soft wrap policy | `layout_measure.go:512-551` |

## New files

- `style_text_box_props.go` + `inline_text_box.go` (optional)
- `style_text_spacing_props.go` + `inline_text_spacing.go` (optional)
- `style_hyphenation_props.go` + `inline_hyphenation.go`

## Checklist

### 87.4.1 text-box*

- [x] 87.4.1.1 Apply `text-box` / `text-box-edge` / `text-box-trim`.
- [x] 87.4.1.2 Consumer trims half-leading in `lineMetrics` using ascent/descent/cap-height refs.
- [x] 87.4.1.3 Tests: `TestTextBoxTrimBothShrinksHalfLeading`. Flip when proven.

### 87.4.2 spacing / autospace / group-align / fit

- [x] 87.4.2.1 Apply autospace + spacing + spacing-trim; advance adjustments in measure/pack.
- [x] 87.4.2.2 `text-group-align` at emit/line origin (Chrome no: still valid print target).
- [x] 87.4.2.3 `text-fit`: scale search is **L**; ship lite or leave Unsupported with note.
- [x] 87.4.2.4 Tests: `TestTextAutospaceIdeographAlpha`, `TestTextGroupAlignCenter`.

### 87.4.3 hyphenation limits + hanging punctuation

- [x] 87.4.3.1 Wire readers for existing `Hyphens` / `HyphenateCharacter` (manual SHY first).
- [x] 87.4.3.2 Apply + consume `hyphenate-limit-chars` / `zone` / `lines` / `last`.
- [x] 87.4.3.3 `hanging-punctuation:first` overhang at line start.
- [x] 87.4.3.4 True `hyphens:auto` dictionary is **L** / out of batch unless a small built-in list is accepted.
- [x] 87.4.3.5 Tests: `TestSoftHyphenUsesHyphenateCharacter`, `TestHyphenateLimitChars`, `TestHangingPunctuationFirst`.

### 87.4.R batch gate (package only)

- [x] 87.4.R.1 `go test ./internal/layout -run 'TestTextBox|TestTextAuto|TestTextGroup|TestTextSpacing|TestSoftHyphen|TestHyphenate|TestHangingPunct' -count=1` exit 0.
- [x] 87.4.R.2 Flip packets only for names with consumers. **No `make test` / `make lint`.**

## Out of scope

`:first-letter` selector (still rejected). Initial-letter (87.5). Full ICU hyphen dictionaries.
