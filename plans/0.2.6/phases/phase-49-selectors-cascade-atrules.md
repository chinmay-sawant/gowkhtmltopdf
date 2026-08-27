# Phase 49: Selectors, cascade, at-rules

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 49
> **Status:** not started
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

- [ ] 49.1.1 Parse `:is()` arguments as a selector list in `appendFunctionalPseudo` (`internal/css/css.go` around 1048). Nested `:is` allowed; `::` in arguments rejected like `:has()`. Proof: `go test ./internal/css -run 'TestParseIs' -v`.
- [ ] 49.1.2 Match if any argument matches. Specificity is the most specific argument (Selectors 4). Proof: `go test ./internal/css -run 'TestIsPseudo|TestIsSpecificity'`.
- [ ] 49.1.3 Layout consume: a box styled only through `:is(div, p)` gets the declarations. Proof: `go test ./internal/layout -run TestIsPseudoStyle`.

### 49.2 `:where()`

- [ ] 49.2.1 Same matching as `:is()`. Specificity contribution 0. Proof: `go test ./internal/css -run TestWherePseudo`.
- [ ] 49.2.2 A later type selector still wins over `:where(#id)` when expected. Proof: specificity unit test.

### 49.3 `@import`

- [ ] 49.3.1 `parseAtRule` keeps `@import` url + optional media prelude instead of `skipAtRule`. Proof: `go test ./internal/css -run TestParseImport`.
- [ ] 49.3.2 `prepare.CollectSheets` (`internal/convert/prepare/styles.go`) fetches via `FetchSub` with the same ACL and `NetworkPolicy` as `<link rel=stylesheet>`. Depth cap. Cycle skip. Failed fetch skips the sheet, does not fail the convert. Proof: `go test ./internal/convert -run TestImportStylesheet`.
- [ ] 49.3.3 Media on `@import` uses existing `MediaMatches`. Proof: print-only import applies on PDF path, not when media is `screen` in image mode if that is the convert media.

### 49.4 Optional, fixture-gated

- [ ] 49.4.1 `:first-of-type` / `:nth-of-type` only if a named fixture fails without them. Otherwise `[~]` with the fixture name that would justify them.
- [ ] 49.4.2 Attribute `i` flag only if a named fixture needs case-insensitive attr match.

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
