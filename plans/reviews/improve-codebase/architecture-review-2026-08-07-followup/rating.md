# Current architecture rating — 9.0 / 10

This is a weighted assessment of the current working tree after the five-agent
follow-up implementation. The score is based on executable tests, focused
benchmarks, CLI smoke paths, and the phase closure artifacts; it is not copied
from the earlier 7.9 score.

| Area | Weight | Score | Evidence and remaining deduction |
|---|---:|---:|---|
| Public API / CLI seam | 20% | 9.2 | Mode-aware parsing, mode-specific request constructors, explicit output sinks, outward app adapters, and deep snapshots are covered. Compatibility adapters remain for existing callers. |
| Dependency direction / module depth | 20% | 9.0 | Shared document preparation and resource context now own cross-mode policy; compatibility entrypoints still retain narrow CLI imports. |
| State ownership / context / concurrency | 20% | 8.8 | Public snapshots, `LayoutContext`, `RenderContext`, raster checkpoints, and race coverage are present. Some compatibility APIs still intentionally use background contexts. |
| Logging / errors / output contracts | 15% | 9.1 | PDF and outline sinks are explicit, failed writers are wrapped, and CLI stderr routing is smoke-tested. |
| Fixtures / golden tests / benchmarks | 15% | 8.9 | High-severity regressions, cross-mode matrix, golden corpus, race suite, layout/PDF benchmarks, and release/debug CLI timings are recorded. |
| Examples / documentation locality | 10% | 8.8 | Phase ledgers, five fix logs, the contract matrix, and this arithmetic are current; some compatibility documentation remains intentionally broad. |
| **Weighted total** | **100%** | **8.98 → 9.0** | `9.2×.20 + 9.0×.20 + 8.8×.20 + 9.1×.15 + 8.9×.15 + 8.8×.10 = 8.98` |

## Validation snapshot

- `make lint` passed: `go vet ./...` and no `gofmt` output.
- `make test` passed: `go test ./...`.
- `go test -race ./...` passed.
- `go test ./internal/css ./internal/layout -count=1` passed.
- `BenchmarkUsedImageSize`: `80.93 ns/op`, `48 B/op`, `1 allocs/op` on Linux/amd64 with Go 1.26.4.
- `BenchmarkShapeRun`: `1313 ns/op`, `528 B/op`, `2 allocs/op` in the captured run.
- `BenchmarkWrite50Pages`: `24,287,371 ns/op`, `1.43 MB/s`, `46,766,645 B/op`, `23,914 allocs/op` in the captured run.
- Release/debug CLI timing used `fixture-16-invoice-with-css.html`, `GOMAXPROCS=1`, 20 copies, Linux WSL2, Go 1.26.4, and separate cold/warm process runs. Debug: `0.03s` cold / `0.04s` warm; stripped release: `0.03s` cold / `0.04s` warm. These are directional smoke timings, not a performance claim.
