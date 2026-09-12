# Ponytail packet 4 - PDF and image output

> **Scope:** `internal/pdf/`, `internal/imageout/`, `internal/pdfprofile/`, and `internal/line/`
> **Canonical rows:** `PT26-OUT-01` through `PT26-OUT-04`
> **Method:** current caller search and focused source reading. No files changed and no tests ran.

## Active evidence

`internal/pdf/semantic.go:L15-105`: **delete** the process-global mutex-protected 64-entry semantic regex LRU. All `compileSemanticRE` callers are in `semantic.go:L816,846,872,905,927`; retain `regexp.Compile` and its current error path. Estimated cut: 75 production lines plus 52 cache-only test lines. Prove with `go test ./internal/pdf -run TestSemantic`.

`internal/imageout/ycbcr.go:L1-85`, `internal/imageout/encode.go:L44,L88-101`: **delete** the custom YCbCr 4:2:0 preconversion. `image/jpeg` accepts the existing NRGBA directly, and `ycbcr_test.go:L53-137` already asserts byte parity. Estimated cut: 98 production lines plus 135 path-specific test lines. Prove targeted JPEG tests.

`internal/imageout/compat_test.go:L1-53`: **delete** the test-only CLI facade. It exists for five same-package callers and can be replaced with `Request` / `RunRequest`, `app.RunImage`, and `imageLoadGlobal` in those tests. Estimated cut: 53 test lines. Prove with `go test ./internal/imageout`.

`internal/imageout/debug_test.go:L1-41`: **delete** the assertion-free ASCII pixel-map debug test. Data-URI behavior is already asserted in `imageout_test.go:L299-317` and `images_gate_test.go:L19-127`. Estimated cut: 41 test lines. Prove with `go test ./internal/imageout`.

## Qualified leads, not active rows

These are valid simplification leads, but they can change a throughput result, retained heap, or encoded bytes. They need an explicit benchmark or format decision before becoming a checklist row.

- `internal/imageout/pngfast.go:L37-99`: remove only the full-canvas fast-PNG branch and keep the strip row encoder. This changes large PNG bytes while preserving decoded pixels.
- `internal/imageout/pixbuffer.go:L1-124`, `imageout.go:L484-515`: remove the global supersample buffer cache and allocate the already-bounded canvas locally. This intentionally trades retained memory for allocation work.
- `internal/pdf/flate_parallel.go:L1-170`: remove Flate worker-pool parallelism and always use the serial finalizer. This intentionally trades throughput for one path.
- `internal/pdf/font_file_cache.go:L1-226`, `registry.go:L346-377`: remove the global external-font LRU and parse each permitted font file per scan. This changes warm scan time.
- `internal/imageout/imageout.go:L612-621`: remove `rasterPaintOrder`, a local test seam that forwards to `layout.PaintOrder`.
- `internal/pdf/fontpdf.go:L143-184`: reduce `runesKey` to an ordered rune sequence after the existing sorted union. Add a Unicode and `RuneError` probe first.

## Rejected lookalikes

- `internal/imageout/imageout.go:L2028-2032`: keep `ResolveFormat`; production `internal/app/image.go:L32` uses it.
- `internal/pdfprofile/profile.go:L18-58`: profile aliases and sentinels are a prior compatibility-sensitive Ponytail row, not a current duplicate.
- `internal/imageout/imageout.go:L659-671`: `rasterImageHash` already uses `hash/fnv.New64a`; the old hand-written hash finding is stale.
- `internal/pdf/fonttype0.go:L12-200`, `shape.go:L138+`: dual embedding and Arabic fallback are documented product capabilities, not unused flexibility.

Packet estimate for active rows: about -267 source and test lines.
