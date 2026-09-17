# Phase 88.9: Closure and full integration gates

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** M
> **Owner:** release / layout lead for the next-100 wave
> **Depends on:** batches 88.1-88.8 as far as product chose to ship
> **Unblocks:** v0.2.7 CSS coverage claim for the promoted subset
> **This is the only batch that runs full-suite integration commands.**

---

## Overview

Close the next-100 program: recount catalog, sync matrix and docs, verify
fixture-65, then run **full** integration gates.

Mid-batch package tests already ran inside 88.1-88.8. Do **not** re-open those
batches to run `make test`. Run them here once.

## Explicit gate policy

| Command | In 88.1-88.8 | In 88.9 |
|---------|--------------|---------|
| `go test ./internal/<pkg> -run …` | yes (except 88.8 defer) | optional recheck |
| `python3 scripts/css-catalog-map.py --check` | yes when arms change | **required** |
| `make test` | **forbidden** | **required** |
| `make golden` | **forbidden** | **required** if layout/paint/pagination changed |
| `make claim-scan` | only if docs edited mid-batch | **required** if matrix/docs touched |
| `make lint` | **forbidden** (owner runs manually after this ledger) | **forbidden** |

Never run bare `go test ./...` (uncapped concurrency). Use Makefile targets.

## Checklist

### 88.9.1 inventory and honesty audit

- [ ] 88.9.1.1 Diff `plans/0.2.6/catalog/mapping.json` against the flip packets from 88.1-88.7. Every new `implemented` row has APPLY + FIELD + CONSUMER + TEST + MATRIX lines (`HONESTY-GATES.md`).
- [ ] 88.9.1.2 Names left Unsupported/Partial (88.8 defer set, Partial hints) have matrix/deferred notes that match code.
- [ ] 88.9.1.3 Recount: update `coverage-summary.json` and `plans/0.2.6/catalog/property-counts.md`. Record new Implemented / Partial / Unsupported totals in the parent canonical Overview.
- [ ] 88.9.1.4 Confirm zero overlap with `next-72-properties.json`.

### 88.9.2 docs and fixture

- [ ] 88.9.2.1 `documentation/compatibility-matrix.md` updated for every promoted name (honest subset text).
- [ ] 88.9.2.2 Fixture-65 still converts; Effect cells for Implemented names show a visible change in the **gowk PDF** (Chrome optional and only for `Chrome yes` rows).
- [ ] 88.9.2.3 If fixture-65 exists, regenerate:
  ```bash
  make build
  ./bin/gowkhtmltopdf --allow-local-files \
    --font-path testdata/fonts/implemented-audit \
    -o output/fixture-65-next-100-props.pdf \
    testdata/golden/fixture-65-next-100-props.html
  ```
- [ ] 88.9.2.4 If page count moved, update `fixturePageBounds` for `fixture-65-next-100-props.html` in `internal/convert/golden_test.go`.

### 88.9.3 full gates (this batch only)

- [ ] 88.9.3.1 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 88.9.3.2 `make test` exit 0.
- [ ] 88.9.3.3 `make golden` exit 0 if layout/paint/pagination changed.
- [ ] 88.9.3.4 `make claim-scan` exit 0 if matrix/docs/frontend content changed.
- [ ] 88.9.3.5 Do **not** run `make lint` here. Owner runs it after the ledger.

### 88.9.R close

- [ ] 88.9.R.1 Mark parent `88-canonical-0.2.7-next-100.md` status complete (or partial-complete with remaining Unsupported listed).
- [ ] 88.9.R.2 Sync `plans/0.2.7/README.md` and `plans/0.2.7/next-100-properties.json` rollups (`implemented` / `partial` / `unsupported` / `catalog_*`).
- [ ] 88.9.R.3 Proof note `_proof-88.9.md` (gitignored) with command exit codes.

## Out of scope

Re-opening 88.1-88.8 to run the full suite. Claiming Implemented from fixture authorship alone.
