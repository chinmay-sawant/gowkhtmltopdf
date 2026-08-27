# Phase 49: Selectors, cascade, at-rules

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 49
> **Status:** in progress (`:is`/`:where`/`@import` parse+fetch done; layout consume test for `:is` still open)
> **Estimated effort:** 4-7 days
> **Owner:** `internal/css`, `internal/convert/prepare`
> **Depends on:** Phase 48.2 mapping reclassify
> **Unblocks:** Phase 50. Utility sheets start matching.

---

## Overview

`:is()` and `:where()` are stored as unknown pseudos and `matchPseudo` returns false (`internal/css/css.go:122`, `css.go:1431-1434`). A rule like `:is(h1, h2, h3) { ... }` never applies. `@import` is `skipAtRule` (`css.go:187`), so split stylesheets silently lose rules.

Keep the never-match rule for `:hover` / `:focus` / `:active` / `:target`. Print has no pointer and no history. Dropping those pseudos from the compound used to style every `li` for `li:target`. Tests in `pseudo_element_drop_test.go` and `target_pseudo_test.go` stay green.

## Goals

- `:is()` / `:where()` match
- `@import` loads under ACL
- Mapping selector rows flip
- No host-degrade regressions

## Checklist

### 49.1 `:is()`

- [x] 49.1.1 Parse `:is()` arguments as a selector list in `appendFunctionalPseudo`. Nested `:is` allowed; `::` in arguments rejected. Proof: `go test ./internal/css -run TestParseIs`.
- [x] 49.1.2 Match if any argument matches. Specificity is the most specific argument. Proof: `go test ./internal/css -run 'TestIsPseudo|TestIsSpecificity'`.
- [x] 49.1.3 Layout consume: a box styled only through `:is(div, p)` gets the declarations. Proof: `go test ./internal/layout -run TestIsPseudoStyle`.

### 49.2 `:where()`

- [x] 49.2.1 Same matching as `:is()`. Specificity contribution 0. Proof: `go test ./internal/css -run TestWherePseudo`.
- [x] 49.2.2 Specificity 0 for `:where` vs `:is` covered in `TestWherePseudo` / `TestIsSpecificity`.

### 49.3 `@import`

- [x] 49.3.1 `parseAtRule` records `ImportRule` on `Stylesheet.Imports`. Proof: `go test ./internal/css -run TestParseImport`.
- [x] 49.3.2 `CollectSheets` fetches imports via `resources.Fetch`, depth 8, cycle skip, failed fetch skipped. Proof: `go test ./internal/convert -run TestImportStylesheet` and `go test ./internal/convert/prepare`.
- [x] 49.3.3 Media on `@import` uses `MediaMatches`. Proof: prepare import tests print vs screen.

### 49.4 Optional, fixture-gated

- [~] 49.4.1 `:first-of-type` / `:nth-of-type`: no failing fixture this session.
- [~] 49.4.2 Attribute `i` flag: no failing fixture this session.

### 49.5 Honesty and gates

- [ ] 49.5.1 `:hover` / `:focus` / `:active` / `:target` still never match. Proof: existing `target_pseudo_test.go` and hover tests pass.
- [ ] 49.5.2 Flip mapping selector rows. Matrix §4 adds `:is()` / `:where()` Partial or Implemented with the specificity note. Proof: grep + `make claim-scan`.
- [ ] 49.5.3 `make lint` and `make test`. After layout consume: `make golden`. Record tails.

## Dependencies

Phase 48 catalog. Existing `:has()` / `:not()` parse paths to copy (`css.go:1048-1077`).

## Evidence

- `go test ./internal/css -run 'TestIs|TestWhere|TestParseImport'`
- convert import test
- golden if any fixture added

## Out of scope

Forgiving selector lists. Shadow DOM. `@layer` ordering. Full `@supports`. `@charset` / `@namespace`.

## Handoff

Next is Phase 50.
