# Phase 87.8: Closure and full integration gates

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** planned
> **Estimated effort:** M
> **Owner:** release / layout lead for the next-72 wave
> **Depends on:** batches 87.1–87.7 as far as product chose to ship
> **Unblocks:** v0.2.7 CSS coverage claim for the promoted subset
> **This is the only batch that runs full-suite integration commands.**

---

## Overview

Close the next-72 program: recount catalog, sync matrix and docs, verify
fixture-64, then run **full** integration gates.

Mid-batch package tests already ran inside 87.1–87.7. Do **not** re-open those
batches to run `make test`. Run them here once.

## Explicit gate policy

| Command | In 87.1–87.7 | In 87.8 |
|---------|--------------|---------|
| `go test ./internal/<pkg> -run …` | yes | optional recheck |
| `python3 scripts/css-catalog-map.py --check` | yes when arms change | **required** |
| `make test` | **forbidden** | **required** |
| `make golden` | **forbidden** | **required** (layout/paint/pagination changed) |
| `make claim-scan` | only if docs edited mid-batch | **required** if matrix/docs touched |
| `make lint` | **forbidden** (owner runs manually after this ledger) | **forbidden** |

Never run bare `go test ./...` (uncapped concurrency). Use Makefile targets.

## Checklist

### 87.8.1 inventory and honesty audit

- [ ] 87.8.1.1 Diff `plans/0.2.6/catalog/mapping.json` against the flip packets from 87.1–87.7. Every new `implemented` row has APPLY + FIELD + CONSUMER + TEST + MATRIX lines (`HONESTY-GATES.md`).
- [ ] 87.8.1.2 Names left Unsupported/Partial (border drafts, VF no-op, shape-inside, etc.) have matrix/deferred notes that match code.
- [ ] 87.8.1.3 Recount: update `coverage-summary.json` and `plans/0.2.6/catalog/property-counts.md` (and `plans/0.2.6/property-counts.md` if still mirrored). Record new Implemented / Partial / Unsupported totals in the parent canonical Overview.

### 87.8.2 docs and fixture

- [ ] 87.8.2.1 `documentation/compatibility-matrix.md` updated for every promoted name (honest subset text).
- [ ] 87.8.2.2 Fixture-64 still converts; Effect cells for Implemented names show a visible change in the **gowk PDF** (Chrome optional and only for `Chrome yes` rows).
- [ ] 87.8.2.3 Regenerate sample PDF for inspection:
  ```bash
  make build
  ./bin/gowkhtmltopdf --allow-local-files \
    --font-path testdata/fonts/implemented-audit \
    -o output/fixture-64-next-72-props.pdf \
    testdata/golden/fixture-64-next-72-props.html
  ```
- [ ] 87.8.2.4 If page count moved, update `fixturePageBounds` for `fixture-64-next-72-props.html` in `internal/convert/golden_test.go`.

### 87.8.3 file-size / architecture check

- [ ] 87.8.3.1 Confirm `style_properties.go` and `layout.go` did not grow past allowlist without a reviewed allowlist edit (`scripts/file-size-allowlist.txt`). Prefer proving with `make size-check` only if the owner wants it; otherwise `wc -l` those files and compare to allowlist.
- [ ] 87.8.3.2 New apply arms live in focused `style_*_props.go` files on `styleGroups`, not dumped into allowlisted giants.

### 87.8.4 catalog check

- [ ] 87.8.4.1 `python3 scripts/css-catalog-map.py --check` exit 0. Record arm count in the proof note.

### 87.8.5 FULL INTEGRATION (required; run only here)

- [ ] 87.8.5.1 **Integration test suite:**
  ```bash
  make test
  ```
  Exit 0 required. Record the command and exit code in this checklist when closing.
- [ ] 87.8.5.2 **Golden corpus:**
  ```bash
  make golden
  ```
  Exit 0 required after layout/paint/pagination changes. Record exit code.
- [ ] 87.8.5.3 Targeted fixture recheck (optional but recommended):
  ```bash
  go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-64' -count=1
  ```
- [ ] 87.8.5.4 If documentation claims changed: `make claim-scan` exit 0.

### 87.8.6 lint (owner-owned; not part of this ledger)

- [ ] 87.8.6.1 **Do not** run `make lint` as part of closing 87.8. Owner runs lint manually afterwards and fixes findings in a follow-up change.

### 87.8.R program close

- [ ] 87.8.R.1 Mark parent `87-canonical-0.2.7-next-72.md` status complete (or partial-complete with remaining Unsupported listed).
- [ ] 87.8.R.2 Update `plans/0.2.7/README.md` status line.
- [ ] 87.8.R.3 Knowledge-base wiki note if the session updates local KB (gitignored); committed docs already updated above.

## Proof template (paste when closing)

```text
CATALOG: Implemented=N Partial=P Unsupported=U / 818
css-catalog-map: check ok (K apply arms)
make test: exit 0
make golden: exit 0
fixture-64 pages: A (envelope min-max)
make lint: skipped (owner manual)
```

## Out of scope

Starting new property batches. Mass-flipping mapping without consumers. Running
`make lint` inside this phase.
