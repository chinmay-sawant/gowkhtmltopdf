# Phase 54: Paged media and fragmentation

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 54
> **Status:** not started
> **Estimated effort:** 1 week
> **Owner:** `internal/css` `@page`, `internal/convert` `applyCSSPageMargins`, `internal/layout` paint_flow
> **Depends on:** existing unnamed `@page` margin/size
> **Unblocks:** reports that change first-page margins

---

## Overview

`@page` today is unnamed only, descriptors `margin` and `size` (`internal/css/css.go:192-216`). Convert applies them before layout (`applyCSSPageMargins` in `internal/convert/convert_helpers.go`).

`page-break-*` and `break-*` alias into always/avoid (`style_properties.go:1222-1276`). `left` / `right` / `column` collapse to page `always`. `avoid-column` is ignored on purpose.

CLI `--header-*` / `--footer-*` remain the supported repeating chrome. CSS margin boxes (`@top-center`) are a maybe, not the default path.

## Goals

- Named `@page` and `:first` / `:left` / `:right`
- Honest matrix for break aliases
- Margin boxes only if a fixture beats CLI headers

## Checklist

### 54.1 named pages and page selectors

- [ ] 54.1.1 Parse `@page name` and `@page :first` / `:left` / `:right`. Proof: `go test ./internal/css -run TestParsePageSelectors`.
- [ ] 54.1.2 Apply first-page margin/size differently from later pages. Proof: `go test ./internal/convert -run TestPageFirstMargins`.
- [ ] 54.1.3 `page: ident` on an element starts that named page if cheap. Otherwise `[~]` with reason. Proof: test or `[~]`.

### 54.2 break values

- [ ] 54.2.1 Document the alias table in matrix §2.6: `left|right|page|column` -> `always`; `avoid-page` -> `avoid`; `avoid-column` ignored. Proof: matrix text.
- [ ] 54.2.2 Optional: `break-before: left` forces the next page to be a left page when copies/duplex exist. If the writer has no left/right page notion, keep the alias and do not fake it.

### 54.3 margin boxes

- [ ] 54.3.1 Default: `[~]` `@top-center` and friends. Reason: CLI headers already repeat on every page. Next gate: a named report fixture that cannot use CLI headers.
- [ ] 54.3.2 If a fixture is named in this row later, implement only `content` strings in margin boxes, not full nested HTML.

### 54.4 gates

- [ ] 54.4.1 Mapping at-rule `@page` notes updated. `make lint`, `make test`, `make golden` if page geometry of fixtures changes. Record tails.

## Dependencies

Existing `PageStyle` side-channel. Paginate in `paint_flow.go`.

## Evidence

Convert first-page margin test. Matrix alias table.

## Out of scope

GCPM `running()`, footnotes, named strings as browser headers. Matching Chrome's Wikipedia page count.

## Handoff

Next is Phase 55.
