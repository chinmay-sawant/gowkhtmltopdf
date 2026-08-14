## Summary

Adds the Go benchmark matrix for local templates, PDF conversion, deterministic HTTP image fetching, and opt-in live TVmaze poster fetching. It also fixes table-row pagination so forced benchmark sections do not separate row text from collapsed-table borders.

## Motivation / context

The benchmark report exposed a pagination defect in the fifth forced section: row 12 text was snapped during provisional pagination before `page-break-before` positioning completed, leaving the row chrome behind. The benchmark fixtures now provide reproducible 2, 5, 10, 20, 50, 100, 200, 250, and 500-page workloads.

## Changes

### Benchmark coverage

- Add Go benchmarks for pre-rendered HTML-to-PDF conversion and template execution plus PDF conversion.
- Add deterministic local HTTP image-fetch benchmarks with inline-image comparison.
- Add opt-in live TVmaze API and poster-CDN benchmarks.
- Add benchmark templates, recorded results, README instructions, and generated PDF benchmark artifacts.

### Pagination correctness

- Resolve forced section starts before provisional text snapping.
- Add row and section page-break avoidance rules to the benchmark report template.
- Add a regression test covering five forced sections and the previously broken row 12.
- Refresh the tracked golden PDF outputs and include the current benchmark PDF outputs requested for local review.

## Impact

| Area | Impact |
|---|---|
| **Performance** | Adds benchmark coverage across the requested workload sizes; no production conversion path is intentionally bypassed. |
| **Memory** | Benchmark-only fixture and output additions; no new runtime dependency. |
| **Behavior / correctness** | Prevents text/table-chrome separation across forced page breaks. |
| **API / CLI** | No API or CLI changes. |
| **Dependencies** | None. |
| **Binary size / build time** | No production binary changes; generated PDF fixtures increase repository size. |

## Breaking changes / migration

| Item | Migration |
|---|---|
| None | - |

## Test plan

- [x] `GOCACHE=/tmp/gowkhtmltopdf-go-cache go test ./...`
- [x] `GOCACHE=/tmp/gowkhtmltopdf-go-cache make lint`
- [x] `gofmt -l internal/layout/paint.go internal/layout/benchmark_report_pagination_test.go`
- [x] `git diff --check`
- [x] Regenerated benchmark PDF artifacts with `TestGenerateBenchmarkOutputs`.
- [x] Visually checked page 5 of `pdf-pages-005.pdf` after regeneration.

### Commands

```sh
GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 \
  go test ./internal/convert -run '^TestGenerateBenchmarkOutputs$' -count=1
```

## Screenshots / sample output

The regenerated `testdata/golden/benchmarks/output/pdf-pages-005.pdf` contains five complete report sections; row 12 on page 5 remains aligned with its table borders.

## Related issues

- No related issue number was provided for this user-requested benchmark work.

## Follow-ups (out of scope)

- Record live Internet benchmark numbers separately from deterministic CI baselines.
- Decide whether generated benchmark PDFs should remain committed long-term or move to release artifacts.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] Generated PDFs were intentionally requested for this PR
