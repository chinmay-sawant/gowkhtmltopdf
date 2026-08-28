# Phase 78: Ignored program closure

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 78
> **Status:** not started
> **Estimated effort:** S
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 77
> **Unblocks:** Ignored program complete

---

## Overview

All 247 former Ignored names are Implemented or explicitly [~] with reason; recount coverage-summary and property-counts; matrix + gates.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

No new property ownership; closes 69-77 leftovers.

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 78.1 closure

- [ ] 78.1.1 Every name from the original 247 Ignored set is `implemented`, or `[~]` with reason and owner.
- [ ] 78.1.2 Recount `coverage-summary.json` and `property-counts.md`.
- [ ] 78.1.3 Matrix + fidelity honesty for newly Implemented rows.
- [ ] 78.1.4 `python3 scripts/css-catalog-map.py --check`, `make test`, `make lint`, `make golden`, `make claim-scan` exit 0.

### 78.2 browser-print golden harness (571 goal:implement)

- [x] 78.2.1 Expand `testdata/golden/fixture-57-vanguard-telemetry-audit.html` so every `goal: implement` property (571) appears once with a representative print value, keep the Vanguard narrative pages, and mark coverage with `VANGUARD-CSS-571-COVERAGE`. Proof: fixture header comment; CSS rules count 571; golden needle present.
- [x] 78.2.2 Update `fixturePageBounds` for fixture-57 (13 pages) and `testdata/golden/README.md` row. Proof: `go test ./internal/convert -run TestGoldenCorpusAllFixtures -count=1` includes fixture-57 PASS.
- [ ] 78.2.3 After Ignored names move onto the work list / Implemented set, extend or refresh fixture-57 (or a sibling) so new Implemented claims stay covered. Proof: linked from this row when done.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Ignored program complete.
