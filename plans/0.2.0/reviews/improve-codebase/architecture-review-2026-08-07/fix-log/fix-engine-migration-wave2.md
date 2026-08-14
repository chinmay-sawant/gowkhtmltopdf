# fix-engine-migration-wave2 — fix log (architecture-review 2026-08-07)

Agent: fix-engine-migration-wave2  
Rows: **P1-1** (api/mains), **P1-8** (shared executor), **P6-03** (mains ExitCode), **P2-04** (full wiring)

Status: **done**

## Per-CID status

### P1-1 — api + mains off `cli.Command` for engine entry — DONE

- **api.go PDF:** `Converter.Convert` builds `convert.Request{Global, Objects}` and calls `convertHooks.executePDF` → `convert.Run` (no `cli.Command`, no `RunPDFContext`).
- **api.go image:** `ImageConverter.Convert` builds `convert.Request{Global, Image, Objects}` and calls `executeImage` → `imageout.RunRequest`.
- **api.go imports:** dropped `gowkhtmltopdf/internal/cli` entirely; surface uses `convert` + `imageout` + `line` + `settings` only.
- **cmd/gowkhtmltopdf:** opens output via `cmd.OpenOutput()`, builds `convert.Request` (ORs legacy `cmd.DumpOutline` into `Global.DumpOutline`), calls `convert.Run` directly.
- **cmd/gowkhtmltoimage:** keeps thin `imageout.Run` adapter (OpenOutput + format-from-path + RunRequest); engine path is already Request-based inside.

### P1-8 — shared convert executor — DONE

Private `convertHooks` in `api.go`:

- fields: `OnInfo`, `OnWarn`, `OnError`, `OnPhase`, `OnProgress`
- `lineLog()` / `progress()` build the shared plumbing
- `executePDF(ctx, *convert.Request) ([]byte, error)` — buffer Output, `convert.Run`, OnError forward
- `executeImage(ctx, *convert.Request) ([]byte, error)` — buffer Output, `imageout.RunRequest`, OnError forward

Both public `Convert` methods are thin: validate input → build Request → execute → stash `c.output`.

### P6-03 — mains exit-code dispatch — DONE

Both mains replaced manual `err.(interface{ HttpErrorCode() int })` with `return cli.ExitCode(err)`.

### P2-04 — full InlineHTML wiring — DONE

Library path was already complete for PDF (`SetBody` / `AddHTML` / `ConvertHTML`). Wave 2 closed image mode:

- `ImageConverter.Convert` accepts `Page != ""` **or** `len(Load.InlineHTML) > 0`
- `Object().SetBody` works (existing `ObjectSettings.SetBody`)
- `imageout.firstObject` no longer skips empty `Page` when `InlineHTML` is set (tiny engine adapter)
- `doc.go` documents image-mode SetBody
- `api_test.go:TestImageConverterSetBody` pins the path

## Files changed

| File | Change |
|------|--------|
| `api.go` | Request-based Convert; convertHooks executor; drop cli import |
| `api_test.go` | `TestImageConverterSetBody` |
| `doc.go` | ImageConverter SetBody note |
| `cmd/gowkhtmltopdf/main.go` | OpenOutput + convert.Run + cli.ExitCode |
| `cmd/gowkhtmltoimage/main.go` | cli.ExitCode |
| `internal/imageout/imageout.go` | firstObject accepts InlineHTML |
| `phases/phase-01-…md` | P1-1, P1-8 → [x] |
| `phases/phase-02-…md` | P2-04 → [x] |
| `phases/phase-06-…md` | P6-03 → [x] |

## Validation

```text
$ go build ./...
# exit 0

$ go test ./... -count=1
ok  gowkhtmltopdf  0.167s
ok  gowkhtmltopdf/internal/cli  0.013s
ok  gowkhtmltopdf/internal/convert  2.787s
ok  gowkhtmltopdf/internal/css  0.020s
ok  gowkhtmltopdf/internal/html  0.014s
ok  gowkhtmltopdf/internal/imageout  0.761s
ok  gowkhtmltopdf/internal/layout  0.535s
ok  gowkhtmltopdf/internal/line  0.012s
ok  gowkhtmltopdf/internal/load  3.214s
ok  gowkhtmltopdf/internal/outline  0.019s
ok  gowkhtmltopdf/internal/pdf  0.952s
ok  gowkhtmltopdf/internal/settings  0.013s
ok  gowkhtmltopdf/internal/svg  0.073s
```

All packages green; root package + cmds compile.
