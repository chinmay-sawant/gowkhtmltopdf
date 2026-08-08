## Summary

Adopts golangci-lint (enable-all) as the project lint gate, clears the bulk of auto-fixable and priority lint debt across the tree, and fixes several correctness regressions that the renames and keyword rewrites introduced in layout: missing `display:flex`/`grid`, pagination `shiftFlowY` infinite loops / double-shifts, and incomplete `page-break-before:always` application on long multi-section reports. Regenerates committed sample and benchmark PDF artifacts so they match the fixed engine.

---

## Motivation / context

- Lint: `make lint` previously ran only `go vet` + gofmt; the branch moves CI and local workflows onto a pinned `golangci-lint` v1.64.8 suite so findings stay consistent.
- Correctness: after mechanical lint cleanups (varnamelen/mnd/goconst keyword tables), flex/grid containers fell back to `display:inline`, flow-index shifts could re-process the same op forever, and forced page breaks only applied ~10 times — breaking flex/grid tests, golden fixtures, 10‑minute timeouts, and 50–500 page benchmark PDF generation.
- Artifacts: `make samples` and `TestGenerateBenchmarkOutputs` refreshed `output/` and `testdata/golden/benchmarks/output/` for reviewer smoke.

---

## Changes

### Lint infrastructure

- Add `.golangci.yml` with enable-all (disable deprecated `tenv`; disable `gofumpt` on Go 1.26 false positives).
- Point `Makefile` `lint` at golangci-lint v1.64.8 (install with `GOTOOLCHAIN=local` when missing).
- Wire the same pin into GitHub Actions before `make lint`.
- Document the target in `CONTRIBUTIONS.md` / samples docs.

### Mechanical and priority lint cleanup

- `golangci-lint run --fix` across ~160 Go files (wsl, nlreturn, gofumpt/gofmt where applicable, godot, whitespace).
- Follow-up passes: `paralleltest`, `varnamelen`, `mnd` package constants (`internal/layout/mnd_const.go`, `internal/pdf/numbers.go`), `exhaustruct` nolints, cyclop splits (`LengthToPt`, `skipBoxShift`), wrapcheck on HF band paint.
- Relocate ponytail debt docs under dated paths.

### Layout correctness

- **`setDisplayKeyword`**: accept `flex`, `grid`, and `inline-grid` so author rules no longer leave containers at initial `inline` after winning the cascade.
- **`shiftFlowY` flow index**: re-read live page buckets each step after swap-remove; grow outer page slices via `*[][]int` so negative shifts are not applied twice and positive shifts no longer hang.
- **`beforeAlways`**: apply every `page-break-before:always` in one tree walk, rebuilding prefix max-Y after each shift (was one break per call, capped by a 10-iteration fixpoint — 50-section benchmarks produced 43 pages).

### Sample / benchmark artifacts

- Regenerate golden fixture samples under `output/` (`make samples`).
- Regenerate benchmark matrix PDFs `pdf-pages-*` and `template-pages-*` (2…500) under `testdata/golden/benchmarks/output/`.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Removes pathological infinite `shiftFlowY` loops on large tables / orphans-widows; multi-break pagination no longer needs N outer fixpoint passes for N forced sections. |
| **Memory** | Flow-index buckets behave correctly when pages grow; no new long-lived allocations beyond existing pagination indexes. |
| **Behavior / correctness** | Flex/grid layouts work again; forced multi-section page counts match section count; golden fixtures and HF link/copy tests pass. |
| **API / CLI** | No public API or CLI flag changes. |
| **Dependencies** | Dev/CI only: golangci-lint pin. Runtime module allowlist unchanged. |
| **Binary size / build time** | Negligible; lint install is CI/local tooling. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None for library/CLI consumers | - |
| Contributors: `make lint` now runs golangci-lint | Install happens automatically via Makefile; first run may download v1.64.8 |

---

## Test plan

- [x] `make test` — full suite green after layout fixes
- [x] `make lint` — clean after cyclop/wsl/wrapcheck follow-ups
- [x] Focused regression runs: flex/grid suite, `TestShiftFlowYNegative*`, `TestPageBreakBeforeStacked`, golden fixtures (08/24), convert HF link tests, `TestTenPageTableReportPerformance`
- [x] `GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 go test ./internal/convert -run '^TestGenerateBenchmarkOutputs$'` — page counts 2…500 match matrix
- [x] `make samples` — fixture/sample PDFs regenerated

### Commands

```sh
make lint
make test
make samples
GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 \
  go test ./internal/convert -run '^TestGenerateBenchmarkOutputs$' -count=1
```

---

## Screenshots / sample output

Benchmark matrix page counts after `beforeAlways` fix:

```
n=25  pdfPages=25
n=50  pdfPages=50
n=100 pdfPages=100
n=500 pdfPages=500
```

`make test` / `make lint` both exit 0 on this branch.

---

## Related issues

- Relates to prior performance and layout work merged via #24
- No single issue fully closed by this PR; branch is a lint + correctness maintenance stream on `chore/misc`

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`bug`, `enhancement`)
- [x] Related issues filled
- [x] Filled body under `plans/PR/pr-chore-misc.md`

---

## Follow-ups (out of scope)

- Remaining golangci style noise (`lll`, broader `cyclop`, `gosec`, `err113`, `testpackage`) intentionally not zeroed on this branch
- Live movie benchmark regeneration (`GOWKHTMLTOPDF_LIVE_BENCHMARK=1`) not re-run here
- Further display keywords / cascade hardening if new CSS values appear

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] Lint-only commits do not hide intentional logic changes (see layout commits `d18cc01`, `3e79487`)
- [ ] No public API / CLI changes required beyond docs for `make lint`
- [ ] Sample/benchmark PDF binary updates are intentional smoke artifacts
- [ ] PR has assignee and labels
- [ ] Related issues keywords are appropriate (Relates / no false Closes)
- [ ] No secrets committed
