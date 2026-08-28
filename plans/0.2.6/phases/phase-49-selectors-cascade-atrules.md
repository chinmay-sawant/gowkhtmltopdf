# Phase 49: Selectors, cascade, at-rules

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 49
> **Status:** complete (selectors, imports, of-type matching, and attribute flags verified 2026-08-28)
> **Estimated effort:** 4-7 days
> **Owner:** `internal/css`, `internal/convert/prepare`
> **Depends on:** Phase 48.2 mapping reclassify
> **Unblocks:** Phase 50. Utility sheets start matching.

---

## Overview

`:is()` and `:where()` are parsed as strict selector lists by `appendIsWherePseudo` (`internal/css/css.go:1223`) and matched by the pseudo matcher. `@import` is recorded by `parseImportRule` (`internal/css/import.go:10`) and fetched by the prepare pipeline under the shared resource policy.

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

- [x] 49.4.1 `:first-of-type` / `:last-of-type` / `:nth-of-type()` / `:nth-last-of-type()`: same-tag sibling index; an+b via `parseNthArg`/`matchNth`. Proof: `go test ./internal/css -run 'TestNthOfType|TestFirstOfType|TestLastOfType|TestNthLastOfType'`.
- [x] 49.4.2 Attribute ASCII `i` flag on valued selectors (`=` `~=` `*=` `^=` `$=` `|=`). Proof: `go test ./internal/css -run TestAttrIFlag`.

### 49.5 Honesty and gates

- [x] 49.5.1 never-match pseudos. Proof: `go test ./internal/css`.
- [x] 49.5.2 Matrix `:is()` / `:where()` Implemented. `make claim-scan` clean.
- [x] 49.5.3 `make lint`/`test`/`golden` green 2026-08-28.

## Dependencies

Phase 48 catalog. Existing `:has()` / `:not()` parse paths to copy (`css.go:1048-1077`).

## Evidence

- `go test ./internal/css -run 'TestIs|TestWhere|TestParseImport'`
- convert import test
- golden if any fixture added


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Forgiving selector lists. Shadow DOM. `@layer` ordering. Full `@supports`. `@charset` / `@namespace`.

## Handoff

Next is Phase 50.
