# Orchestrator integration — follow-up closure

No Git commands or commits were performed. The five agent slices were
integrated in the shared working tree after their package-local validation.

## Integration changes

- Command mains now pass `ModePDF`/`ModeImage` and call `internal/app` adapters.
- `convert.NewPDFRequest`/`NewImageRequest` and mode validators make the
  fix-contract compatibility union explicit without changing its required
  fields.
- Image mode consumes `convert.PrepareDocument` and its bound resource context;
  HTML header/footer preparation now binds child base URLs and fetches through
  the same loader/load policy.
- Outline consumers use `outline.DocumentPage` through `BuildTreeBy`,
  `SectionOfBy`, and `DumpOutlineXMLBy`; no copied page view is required.
- `LayoutContext`, `PaintContext`, `PaintBandContext`, and `RenderContext`
  carry cancellation through layout, PDF/HF paint, and raster traversal.
- Added failed-writer, mode-constructor, paint-cancellation,
  render-cancellation, and 10k-op/100-page benchmark coverage.

## Validation

```text
make lint
make test
go test -race ./...
go test ./internal/css ./internal/layout -count=1
go test ./internal/layout -run '^$' -bench 'Benchmark(DisplayListIdentity10kOps100Pages|UsedImageSize)' -benchmem -count=1
go test ./internal/pdf -run '^$' -bench 'Benchmark(ShapeRun|Write50Pages)' -benchmem -count=1
```

All commands passed on Linux/amd64, Go 1.26.4, 13th Gen Intel(R) Core(TM)
i7-13700HX. The final focused layout run reported `BenchmarkUsedImageSize`
at `80.93 ns/op` and `BenchmarkDisplayListIdentity10kOps100Pages` at
`8,554,807 ns/op`.

CLI mode smoke evidence:

```text
go run ./cmd/gowkhtmltopdf --transparent testdata/golden/fixture-01-simple-invoice.html /tmp/gowkhtmltopdf-mode-reject.pdf
# rejected: option --transparent is not supported in pdf mode
go run ./cmd/gowkhtmltoimage --page-size A4 testdata/golden/fixture-01-simple-invoice.html /tmp/gowkhtmltopdf-mode-reject.png
# rejected: option --page-size is not supported in image mode
```
