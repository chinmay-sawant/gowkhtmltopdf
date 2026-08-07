# Cross-mode contract matrix

This matrix is the Phase 5 closure artifact. It records the contract owner,
the PDF/image behavior, and current executable proof from the same working-tree
diff. No row is deferred.

| Contract | PDF path | Image path | Proof |
|---|---|---|---|
| Settings and mode | `cli.Parse(argv, ModePDF)`; `convert.NewPDFRequest` rejects image settings | `cli.Parse(argv, ModeImage)`; `convert.NewImageRequest` requires image settings | `go test ./internal/cli ./internal/convert`; both CLI smoke paths |
| Flag applicability | PDF-only registry entries accepted; image-only entries rejected | Image-only registry entries accepted; PDF-only entries rejected | `TestParseModeRejectsInapplicableFlags`, `TestParseModeAcceptsApplicableFlags`; `go run` rejection smoke |
| Output sinks | `internal/app.RunPDF` owns file/stdout selection; `convert.Run` receives `Output` and optional `OutlineOutput` | `internal/app.RunImage` delegates explicit image request/output ownership | `TestRunRequiresExplicitOutputSink`, `TestRunRequiresDedicatedOutlineSink`, `TestRunPropagatesDocumentWriterError`, `go test ./...` |
| Inline HTML | `SetBody` snapshot loads through the shared preparation path | `ImageConverter.SetBody` uses the same preparation path | `TestObjectSettingsSetBodyCopiesInput`, `TestImageConverterSetBody`, `TestPrepareDocumentBindsSharedResourceContext` |
| ACL and resource base | `convert.ResourceContext` binds loader/base/load policy for CSS, fonts, images, and HTML headers/footers | `PreparedDocument.Resources` is used for raster image fetches | `TestResourceContextBindsBaseAndPolicy`, `TestPrepareDocumentBindsSharedResourceContext`, `go test ./internal/load ./internal/imageout` |
| Media and simplify | `PrepareDocument` gathers sheets with PDF media and simplify profile | `PrepareDocument` gathers sheets with image media and simplify profile | `TestMediaForPDF`, simplify tests, image conversion tests, `go test ./...` |
| Error routing | engine returns wrapped errors to caller; CLI prints them on stderr | same request/output error route through `RunImage` | `TestRunPropagatesDocumentWriterError`, CLI smoke commands, `make test` |
| Cancellation | load, object loop, layout and PDF checkpoints observe `context.Context` | load, layout, raster traversal and downscale observe `context.Context` | `TestRunPDFContextCancel`, `TestLayoutContextHonorsCancellation`, `go test -race ./...` |

## Reproduction gates

```text
make lint
make test
go test -race ./...
go test ./internal/css ./internal/layout -count=1
```

All four commands passed on 2026-08-07 from the repository root. The focused
layout benchmark was run separately from correctness tests:

```text
go test ./internal/layout -run '^$' -bench 'BenchmarkUsedImageSize' -benchmem -count=1
```

Linux/amd64, Go 1.26.4, 13th Gen Intel(R) Core(TM) i7-13700HX:
`80.93 ns/op`, `48 B/op`, `1 allocs/op` for used-image sizing and
`8,554,807 ns/op` for the 10k-operation/100-page display-list benchmark.
