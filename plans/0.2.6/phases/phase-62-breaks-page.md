# Phase 62: Breaks and page

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 62
> **Status:** complete (honest with documented break aliases)
> **Estimated effort:** L
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 61
> **Unblocks:** Phase 63

---

## Overview

break-*/page-break-* print values, orphans/widows, page named behavior.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 62 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 62.1 scope lock

- [x] 62.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `break-after`, `break-before`, `break-inside`, `page-break-after`, `page-break-before`, `page-break-inside`, `orphans`, `widows`, `page` (9 properties).

### 62.2 implementation

- [x] 62.2.1 Implement exit criteria for each owned name. Proof: `TestPageBreakParsing`, `TestPageBreakBeforeAlways`, `TestPageBreakInsideAvoid`, `TestOrphansWidows`, `TestPageNameInherits`, `TestPageNameBreak`, `TestPageNamedMargins`, `TestPageMarginsSharePaintCascade`.

### 62.3 catalog and docs

- [x] 62.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 154 implemented, 20 partial; `property-counts.md` updated.
- [x] 62.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Section 2.6 updated; `make claim-scan` clean.

### 62.4 gates

- [x] 62.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestPage.*|TestOrphans.*"` exit 0.
- [x] 62.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 62.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.


## Honesty audit (2026-08-28)

Audit 2026-08-28: KEEP_IMPLEMENTED for break/page/orphans/widows with matrix alias table.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 63.
