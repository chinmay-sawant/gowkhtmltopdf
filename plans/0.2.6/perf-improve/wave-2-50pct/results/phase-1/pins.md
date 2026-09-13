# Phase 1 pins (PERF3-02, PERF3-03)

Ceilings for later phases. Method is in `method.md`. Cold and warm are
labeled. `B/op` is never averaged. Generic path only.

## 500-page output-bytes and pages (PERF3-03)

| item | value |
|---|---:|
| output-bytes | 1,419,234 |
| pages | 500 |
| PDF header | `%PDF-` |
| path | generic `NewPDFRequest` via public `Document.WritePDF` |

Test: `document_perf3_pin_test.go` (`TestPerf3OutputBytesPin`). Fails if
bytes drift, the header is missing, page count is not 500, or ordered
text needles (`SKU-001-001` then `SKU-500-020`) move.

Evidence already on this tree: `pre-change-500p.txt` reports 1,419,234
bytes and 500.0 pages from
`BenchmarkPDFPages/generic/500Pages`. Snapshot M reports the same
output-bytes on every 500-page generic row.

Golden 65/65 stays `make golden`. This pin file does not replace that
gate.

## 500-page B/op (PERF3-02)

| item | value | label |
|---|---:|---|
| Snapshot M warm-matrix ceiling | 163,021,712 B | warm 500 pages (median-time sample) |
| Snapshot M standalone median | 167,865,712 B | 2p cold then 500p warm |
| this tree pre-change pin | 169,955,488 B | 500p-only cold generic |

Later warm-matrix 500-page `B/op` must stay at or below the captured
warm value 163,021,712 B (plan: "warm 500p B/op <= the captured value").
The 169,955,488 B figure is the cold 500p-only row in
`pre-change-500p.txt`. Do not use it as the warm ceiling.

## style storage (PERF3-02)

| item | value |
|---|---|
| pin | 221,208 B per conversion |
| test | skipped |

`styleStore` and `styleStore.append` live only inside `internal/layout`
(`style.go:718-750`). Nothing outside that package can read the 221,208 B
figure. This drop does not change production layout to expose it. The
number stays the standing pin from Snapshot M / the time-wave phase-1
measurement: one 64-record chunk per conversion, same size at 2 pages and
at 500 pages.

## public image 250 / 500 tiles (PERF3-02)

Snapshot M median-time sample, not re-measured in this pin drop:

| tiles | B/op ceiling |
|---:|---:|
| 250 | 14,296,096 B |
| 500 | 26,414,016 B |

Later image rows must stay at or below these values.

## Op size (PERF3-02)

| item | value |
|---|---:|
| ceiling | 432 B |
| test | `internal/layout/perf3_op_size_pin_test.go` (`TestPerf3OpSizePin`) |

`unsafe.Sizeof(Op{})` must not grow above 432. Shrink (phase 3 target
<=256) still passes. 432 is a ceiling, not a floor.

Source of 432: PERFT-11 packing, `internal/layout/layout.go:408-505`.
The plan records 66,500 live ops at 500 pages.

## standing pin summary

| metric | ceiling | status |
|---|---:|---|
| 500p output-bytes | 1,419,234 exact | pinned by `TestPerf3OutputBytesPin` |
| 500p pages | 500 exact | pinned by `TestPerf3OutputBytesPin` |
| warm 500p B/op | 163,021,712 B | captured Snapshot M; this drop's pre-change row is the cold 169,955,488 B sample |
| style storage | 221,208 B | recorded; no exported observer to assert in a new test |
| image 250 tiles B/op | 14,296,096 B | Snapshot M ceiling |
| image 500 tiles B/op | 26,414,016 B | Snapshot M ceiling |
| Op size | 432 B | pinned by `TestPerf3OpSizePin` |
