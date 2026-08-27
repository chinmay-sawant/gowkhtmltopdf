# Phase 54: Paged media and fragmentation

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 54
> **Status:** complete except `[~]` `page: ident` and margin boxes
> **Estimated effort:** 1 week
> **Owner:** `internal/css` `@page`, `internal/convert` `applyCSSPageMargins`, `internal/layout` paint_flow
> **Depends on:** existing unnamed `@page` margin/size
> **Unblocks:** reports that change first-page margins

---

## Overview

`@page` today is unnamed only, descriptors `margin` and `size` (`parsePageRule` `internal/css/css.go:222`). The selector after `@page` is ignored, so `:first` is not a page selector. Convert applies `Stylesheet.Page` before layout (`applyCSSPageMargins` in `internal/convert/convert_helpers.go:22`).

`page-break-*` and `break-*` alias into always/avoid (`applyPageBreakProps` `style_properties.go:1467`). `left` / `right` / `page` / `column` collapse to page `always`. `avoid-page` maps to `avoid`. `avoid-column` is ignored on purpose. No even/odd pages.

CLI `--header-*` / `--footer-*` remain the supported repeating chrome. CSS margin boxes (`@top-center`) stay `[~]`: not parsed, not the default path.

## Goals

- Named `@page` and `:first` / `:left` / `:right`
- Honest matrix for break aliases
- Margin boxes only if a fixture beats CLI headers

## Checklist

### 54.1 named pages and page selectors

- [x] 54.1.1 Parse `@page` selectors. Proof: `TestParsePageSelectors`.
- [x] 54.1.2 `:first` margins on page 1. Size unnamed-only. Proof: `TestPageFirstMargins`.
- [~] 54.1.3 `page: ident` not implemented. Writer has no named-page model.

### 54.2 break values

- [x] 54.2.1 Document the alias table in matrix §2.6: `left|right|page|column` -> `always`; `avoid-page` -> `avoid`; `avoid-column` ignored. Proof: `documentation/compatibility-matrix.md` §2.6 (`applyBreakBeforeProps` `style_properties.go:1480`).
- [x] 54.2.2 Writer has no left/right or even/odd page side (duplex out of scope). Keep the alias: `left`/`right` store `always`. Do not fake left/right pages. Proof: matrix §2.6 plus §5 "PDF encryption, duplex, AcroForm" out of scope.

### 54.3 margin boxes

- [~] 54.3.1 `@top-center` and friends. Reason: CLI `--header-*` / `--footer-*` already repeat on every page (`documentation/compatibility-matrix.md` §2.6 and §7.7). `parsePageRule` (`css.go:222`) keeps `margin`/`size` only. Next gate: a named report fixture that cannot use CLI headers.
- [~] 54.3.2 No fixture named. Not implementing margin-box `content` strings. Owner: 54.3. Pointer: matrix §2.6 "Not implemented".

### 54.4 gates

- [x] 54.4.1 Matrix @page rows. `make lint`/`test`/`golden` green 2026-08-27.

## Dependencies

Existing `PageStyle` side-channel. Paginate in `paint_flow.go`.

## Evidence

Convert first-page margin test. Matrix alias table.

## Out of scope

GCPM `running()`, footnotes, named strings as browser headers. Matching Chrome's Wikipedia page count.

## Handoff

Next is Phase 55.
