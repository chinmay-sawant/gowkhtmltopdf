# Phase 60: Text, lists, generated content

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 60
> **Status:** complete (honest)
> **Estimated effort:** M
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 59
> **Unblocks:** Phase 61

---

## Overview

white-space leftovers, font/font-family, list-style*, content/quotes/counters.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 60 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 60.1 scope lock

- [x] 60.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `white-space`, `font`, `font-family`, `list-style`, `list-style-image`, `list-style-position`, `content`, `quotes`, `counter-increment`, `counter-reset` (10 properties).

### 60.2 implementation

- [x] 60.2.1 Implement exit criteria for each owned name. Proof: `TestFontShorthand`, `TestWhiteSpacePre`, `TestWhiteSpacePreWrap`, `TestListStyleImage`, `TestListStylePositionInside`, `TestQuotes`, `TestCounterInBefore`, `TestCounterResetIncrementLayout`.

### 60.3 catalog and docs

- [x] 60.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 137 implemented, 37 partial; `property-counts.md` updated.
- [x] 60.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Section 2.3 updated; `make claim-scan` clean.

### 60.4 gates

- [x] 60.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestFont.*|TestListStyle.*|TestQuotes|TestCounter.*"` exit 0.
- [x] 60.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 60.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.


## Honesty audit (2026-08-28)

Audit 2026-08-28: KEEP_IMPLEMENTED for owned text/list/generated-content names.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 61.
