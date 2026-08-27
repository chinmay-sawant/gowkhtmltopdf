# Phase 48: Catalog freeze and honesty

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 48
> **Status:** partial (48.1 done 2026-08-27; 48.2 open)
> **Estimated effort:** 2-3 days remaining
> **Owner:** `plans/0.2.6/catalog`, `documentation/compatibility-matrix.md`, `scripts/`
> **Depends on:** none
> **Unblocks:** Phase 49

---

## Overview

Pin the CSS name universe, map it onto current code, and stop the matrix from lying. Implementation of new properties starts in Phase 49.

## Goals

- Catalogs on disk with checksums
- Mapping file agents can grep
- Print-noop UI not sitting in the must-implement bucket
- Stale matrix rows fixed
- A script that rebuilds the mapping

## Deliverables

- `plans/0.2.6/catalog/*` (present)
- `scripts/css-catalog-map.py` plus a short note in `scripts/` docs or Makefile target
- Matrix line-number and `table-layout` / `ex`/`ch` fixes
- Pointer from `plans/0.2.0/phases/pending-phase-items/README.md`

## Checklist

### 48.1 Vendor catalogs (done 2026-08-27)

- [x] 48.1.1 Store webref CSS JSON. Proof: `plans/0.2.6/catalog/webref-css.json`, sha256 `b26a0501c6ee972ca343d2f91be620aaef0c719ec5602a2a70f317fd22135d75`.
- [x] 48.1.2 Store W3C all-properties JSON. Proof: `catalog/w3c-all-properties.json`, sha256 in `catalog/SOURCE.md`.
- [x] 48.1.3 Store mdn units and properties overlays. Proof: `catalog/mdn-units.json`, `catalog/mdn-properties.json`.
- [x] 48.1.4 Generate mapping from webref plus apply-handler inventory. Proof: `catalog/mapping.json` and `catalog/coverage-summary.json` with 818 properties, statuses 75 implemented / 45 partial / 488 unsupported / 210 ignored.
- [x] 48.1.5 Write `catalog/SOURCE.md` and `catalog/README.md`. Proof: files exist.

### 48.2 Reclassify, script, matrix

- [ ] 48.2.1 Set `goal: ignore` on print-noop UI in `mapping.json`: `cursor`, `caret-color`, `resize`, `user-select`, `pointer-events`, `touch-action`, `appearance`. SVG `fill`/`stroke` presentation as CSS stays later/ignore unless a PDF text path needs them. Proof: python one-liner or `scripts/css-catalog-map.py --check` after the script exists.
- [ ] 48.2.2 Add `scripts/css-catalog-map.py` that reads frozen catalogs plus greps `internal/layout/style_properties.go` / `style_cascade.go` apply arms and writes `mapping.json`. `--check` mode diffs against the committed mapping and exits 1 on drift. Proof: `python3 scripts/css-catalog-map.py --check` exit 0. Note in `scripts/` README or Makefile. Anything used twice belongs in `scripts/` (`AGENTS.md`).
- [ ] 48.2.3 Matrix honesty: `table-layout` is consumed lite at `internal/layout/layout_tables.go:45`, not "auto only". `ex`/`ch` resolve as 0.5em at `internal/css/container.go:133-134`, not dropped. `applyRestProps` citation is `internal/layout/style_cascade.go:666` plus `style_properties.go`, not `style.go:340`. Proof: those sentences in `documentation/compatibility-matrix.md`; `make claim-scan` still clean.
- [ ] 48.2.4 Add missing matrix rows that code already honors: `overflow-wrap` / `word-wrap` / `word-break`, `accent-color`, `border-radius`, `z-index`, `container-*`, `var()` / `--*`, `calc()` three-token subset, `:not()`, `:root`. Status Partial or Implemented with `file:line`. Proof: grep those names in `documentation/compatibility-matrix.md`.
- [x] 48.2.5 Banner on `plans/0.2.0/phases/pending-phase-items/README.md`: remaining CSS coverage lives at `plans/0.2.6/48-canonical-0.2.6-css-coverage.md`. Proof: first paragraph pointer 2026-08-27.
- [ ] 48.2.6 `make lint` / `make test` if any Go or scripts tests were added; skip if docs-only besides the script. Proof: command tails on this file.

## Dependencies

None. Catalog files are already in tree.

## Evidence

- `catalog/SOURCE.md` checksums
- `coverage-summary.json`
- matrix diff
- script `--check` exit code

## Out of scope

New CSS behavior. WOFF2. Chrome PDFs.

## Handoff

Next is Phase 49 (`:is()` / `:where()` / `@import`).
