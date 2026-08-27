# Phase 54: Paged media and fragmentation

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 54
> **Status:** complete (lite leftovers documented in 54.1.3 / 54.3)
> **Estimated effort:** 1 week
> **Owner:** `internal/css` `@page`, `internal/convert` `applyCSSPageMargins`, `internal/layout` `page_named.go`
> **Depends on:** existing unnamed `@page` margin/size
> **Unblocks:** reports that change first-page, side, or named-page margins

---

## Overview

`parsePageRule` (`internal/css/css.go`) stores every `@page` on `Stylesheet.Pages`. Selectors are `""`, `:first`, `:left`, `:right`, or a page name ident. Nested `@top-*` / `@bottom-*` quoted `content` strings land on `PageRule.Boxes` (`internal/css/page_margin.go`).

Convert applies unnamed margin/size, then stores `:first` / `:left` / `:right` / named margin overrides (`applyCSSPageMargins` in `internal/convert/convert_helpers.go`). Paint cascade is unnamed, then named ident, then `:left`/`:right` by page side, then `:first` on page 1 (`layout.PaintOptions.forPage` in `page_named.go`). Size stays unnamed-only.

LTR print, not duplex: page 1 is `:right` (recto), even pages `:left`, odd pages `:right`. `break-before: left` still aliases to `always` (54.2).

`page: ident` is a used value on `ResolvedStyle.PageName`. Unspecified/`auto` keeps the parent name. A sibling whose used name changes gets `break-before: always`. Parent/child (for example `body { page: chapter }`) does not insert a blank first page.

CLI `--header-*` / `--footer-*` remain the repeating chrome path. Unnamed `@page` margin boxes fill empty Left/Center/Right slots on that path.

## Goals

- Named `@page` and `:first` / `:left` / `:right`
- Honest matrix for break aliases
- Margin boxes lite on the CLI header/footer path

## Checklist

### 54.1 named pages and page selectors

- [x] 54.1.1 Parse `@page` selectors. Proof: `TestParsePageSelectors`.
- [x] 54.1.2 `:first` / `:left` / `:right` margins. Size unnamed-only. Page 1 is `:right`; `:first` wins on page 1. Proof: `TestPageFirstMargins`, `TestPageLeftRightMargins`, `TestPageFirstWinsOverLeftRight`.
- [x] 54.1.3 `page: ident` used-value inherit; sibling name change forces `break-before: always`; named `@page` margin on pages that overlap a box with that name. Size unnamed-only. Continuation pages with no overlapping named box keep unnamed/`:left`/`:right`/`:first`. Link/outline `hfGeom.pageMargins` uses the side/`:first` cascade only (no per-page name list). Proof: `TestPageNameInherits`, `TestPageNameBreak`, `TestPageNamedMargins`. `go test ./internal/css ./internal/convert ./internal/layout -count=1` exit 0.

### 54.2 break values

- [x] 54.2.1 Document the alias table in matrix §2.6: `left|right|page|column` -> `always`; `avoid-page` -> `avoid`; `avoid-column` ignored. Proof: `documentation/compatibility-matrix.md` §2.6 (`applyBreakBeforeProps` `style_properties.go`).
- [x] 54.2.2 Writer has no duplex even/odd sheet. Keep the alias: `break-before: left`/`right` store `always`. Do not fake a blank page to reach a left sheet. `@page :left`/`:right` **margins** are a separate paint-origin shift (54.1.2). Proof: matrix §2.6 plus §5 "PDF encryption, duplex, AcroForm" out of scope.

### 54.3 margin boxes

- [x] 54.3.1 `@top-left/center/right` and `@bottom-left/center/right` quoted `content` parse. Proof: `TestParsePageMarginBoxes`.
- [x] 54.3.2 Unnamed `@page` boxes map onto CLI header/footer empty slots (repeat every page). Occupied CLI slots and HTMLURL win. `counter()` / `running()` drop. `:first`/`:left`/`:right`/named boxes parse onto `PageRule.Boxes` but do not change per-page chrome. Proof: `TestPageMarginBoxes`, `TestPageMarginBoxesCLIWins`. Leftover: no CSS page-margin formatting context, no corners, no GCPM.

### 54.4 gates

- [x] 54.4.1 Matrix @page rows. Targeted proof `go test ./internal/css ./internal/convert ./internal/layout -count=1` exit 0 (2026-08-27). Prior `make lint`/`test`/`golden` remain the 2026-08-27 session-end gate (not re-run in this leftover pass).

## Dependencies

Existing `PageStyle` side-channel. Named-page breaks clone interned style then reuse `beforeAlways` in `paint_flow.go`. New logic lives in `internal/layout/page_named.go` and `internal/css/page_margin.go`.

## Evidence

`TestPageFirstMargins`, `TestPageLeftRightMargins`, `TestPageNamedMargins`, `TestPageNameBreak`, `TestPageMarginBoxes`. Matrix §2.6.

## Out of scope

GCPM `running()`, footnotes, named strings as browser headers. Combined selectors (`@page chapter:first`). Per-page page size. Matching Chrome's Wikipedia page count.

## Handoff

Next is Phase 55.
