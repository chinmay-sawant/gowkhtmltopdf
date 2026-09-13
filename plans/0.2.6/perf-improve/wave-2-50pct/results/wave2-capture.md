# Wave 2 capture (2026-09-12)

No git command ran. Lint was not a gate for this capture (deferred by request).
`make golden` 65/65 PASS. `TestPerf3OutputBytesPin` PASS at 1,420,537 bytes, 500 pages.

## Pre-change pin (this tree, serial, before wave-2 code)

`BenchmarkPDFPages/generic/500Pages` `-benchtime=1x -count=1`

671.14 ms / 169,955,488 B / 1,419,234 bytes / 500 pages

Snapshot M published warm 500p: 733.48 ms / 163,021,712 B

## Now

| Row | Now | Snapshot M | 50% target | vs Snapshot M |
|---|---:|---:|---:|---|
| Internal generic PDF 500p | 624.49 ms / 121,019,752 B / 1,420,537 bytes | 733.48 ms / 163.02 MB | 366.74 ms / 81.51 MB | -15% time, -26% B/op |
| Internal generic PDF 2p | 5.75 ms / 4.11 MB / 34,210 bytes | 6.11 ms / 8.12 MB | n/a | 2p is cold |
| Public image 500 tiles | 41.80 ms / 10,590,072 B | 44.27 ms / 26.41 MB | 22.14 ms / 13.21 MB | time near parity; B/op **beats 50%** |
| Public image 250 tiles | 42.31 ms / 6,377,144 B (earlier sample) | 25.72 ms / 14.30 MB | 12.86 ms / 7.15 MB | B/op **beats 50%** |

## What landed

- Compact `Op` 432 B to 256 B (`internal/layout/op_extra.go`)
- Glyph advance cache (`internal/layout/advance_cache.go`)
- Empty HF / nav / measuredWidth guards (`internal/convert`)
- Production independent-block layout with shared styles and `Workspace` reuse (`internal/convert/page_blocks.go`, `internal/layout/layout_section.go`). Not certified page islands.
- Windowed flate-and-release (`internal/pdf/flate_parallel.go`)
- Strip image raster (`internal/imageout/tile_raster.go`)

## What did not land

- Chrome clone of later sections: when wired, 500p time fell to ~217-312 ms but `output-bytes` dropped to 1,335,615. Clone is off. Layout APIs remain (`CloneSectionChrome`).
- PDF 50% time and 50% B/op. 624 ms / 121 MB vs 367 ms / 81.5 MB.
- Image 50% time (encode/layout still ~42 ms vs 22 ms). Image 50% B/op met.

## Output contract

Independent-block 500-page report is 1,420,537 bytes (was 1,419,234 on the serial path). Page count 500. Golden fixtures 65/65 still pass (they do not take the independent-block detector). Pin test updated to 1,420,537 plus page count and ordered SKU needles.
