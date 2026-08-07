# fix-root-api — fix log (architecture-review 2026-08-07)

Agent: fix-root-api (rows: P1-9, P1-7 api, P6-1 root, P6-2, P2-04 api+doc, cmd/main P1-5 read-side)

Status: all rows verified **done** (code landed in the cancelled prior session, re-verified here);
validation of the root package is **blocked transitively** by fix-convert's in-flight
`internal/convert` state (see Validation). No code edits were needed during this verification
pass — every row already matched its phase snippet exactly.

## Per-CID status

### P1-9 — `ImageConverter.Object()` nil-receiver panic — DONE
- `NewImageConverter` seeds `object: NewObjectSettings()` (api.go:259-265); `Convert`'s guard is
  page-content only: `if strings.TrimSpace(c.object.o.Page) == ""` (api.go:307). Matches the
  phase-01 `<a id="p1-9">` Future snippet. `api_test.go:TestImageConverterNeedsPage` still passes
  conceptually (Convert without a page returns the real error, no panic).

### P1-7 (api side) — `ImageConverter.Set` delegates background aliasing — DONE
- `Set` is a one-line delegate: `return settings.ApplyImageKey(&c.global.g, &c.image, name, value)`
  (api.go:272-274). `settings.ApplyImageKey` exists at internal/settings/reflect.go:642 with the
  contract shape (`background`/`web.background` → `global.Set("background", …)`, else
  `img.Set`) — fix-settings-cli landed it; `settings_test.go:TestApplyImageKeyBackgroundAlias` passes.
- No `strings` import leftover concerns: `strings.TrimSpace` still used elsewhere in api.go.

### P6-1 (root side) — log severity protocol via internal/line — DONE
- `internal/line` exists per fix-contract contract #7: `type Severity`, `Emit(w io.Writer, sev Severity, format string, args ...any)`, `SeverityOf(string)` (internal/line/line.go) + tests (line_test.go).
- `api.go` lineLog.Write classifies via `line.SeverityOf(l)` (api.go:367); the callback mapping
  stays in api.go; **no substring `Contains` guessing remains anywhere in api.go** (grep
  verified — only `strings.TrimSpace` remains).
- `api_test.go:TestLineLogSeverity` pins the protocol including the "info: … mentions error is not
  an error line" regression.

### P6-2 — examples rewritten to stdlib flag + mustSet — DONE
- examples/pdf/main.go and examples/image/main.go: `flag.NewFlagSet(..., flag.ContinueOnError)`,
  `ErrHelp → nil`, `NArg() != 2` guard, `mustSet(name, value)` wraps errors with
  `fmt.Errorf("%s: %w", name, err)`. Every `c.Global().Set` / `obj.Set` / `c.Set` / `c.Object().Set`
  return value is checked (pdf: 4 checked sites, image: 4). CLI surface unchanged: pdf keeps
  `--page-size/--orientation/--margin-top/--enable-local-file-access`, image keeps
  `--width/--format/--enable-local-file-access` (+ image's format whitelist pre-check).

### P2-04 (api + doc side) — in-memory HTML as explicit source kind — DONE
- `ObjectSettings.SetBody(html []byte, base string)` sets `o.Page = ""`, `o.Load.InlineHTML = html`,
  `o.Load.InlineBase = base` (api.go:99-104) — byte-identical to the phase-02
  `<a id="p2-04">` Future snippet field set.
- `Converter.AddHTML(page []byte, baseURL string)` exists (api.go:158-161);
  `ConvertHTML` uses `SetBody(html, "")` (api.go:228) — no `SetPage(string(html))` guessing left.
- `doc.go` gained the "In-memory HTML" section ("no URL guessing is applied … optional base URL
  resolves relative subresources").
- `internal/settings.LoadPage` carries `InlineHTML []byte` + `InlineBase string`
  (internal/settings/settings.go:204-208) — fix-settings-cli landed; real consumer is
  `internal/load` (fix-html-load-outline). Policy A satisfied.

### P1-1 (prep only) / P1-8 — NOT OURS IN WAVE 1
- Per contract ownership table: `fix-engine-migration-wave2` owns the api.go/mains migration to
  the `Request` seam and the P1-8 shared-executor collapse. api.go still builds `cli.Command`
  (api.go:175, 311) — that is the wave-2 baseline state, left untouched. Logged, not actioned.

### cmd/gowkhtmltopdf/main.go (P1-5 read-side) — DONE, kept as-is
- main.go:37-42 reads `cmd.Global.DumpDefaultTOCXSL` — consistent with the P1-5 Future snippet
  ("main.go:39 reads cmd.Global.DumpDefaultTOCXSL"). CLI appliers write Global (cli_test.go:
  `TestDumpOutlineGlobalHome` pins it). Verified `settings.PdfGlobal.DumpDefaultTOCXSL` +
  reflect keys `dumpoutline`/`dumpoutlinewithdefaulttocxsl` exist (internal/settings/reflect.go:342-347).
- **Stale marker elsewhere:** internal/cli/flags.go:113 carries `// FIX-REVIEW: P1-5 … main.go still
  reads Command.DumpDefaultTOCXSL — fix-root-api must switch…`. The switch IS done; that marker is
  now stale. It lives in fix-settings-cli's file, so per contract rule 5 I did not edit it —
  flagged here for fix-settings-cli / wave-3 integration to remove.

## Files changed (by this fixer, incl. prior cancelled session)

- `api.go` — P1-9 seed, P1-7 delegate, P6-1 lineLog via internal/line, P2-04 SetBody/AddHTML/ConvertHTML
- `doc.go` — P2-04 "In-memory HTML" section
- `cmd/gowkhtmltopdf/main.go` — P1-5 read side (`cmd.Global.DumpDefaultTOCXSL`)
- `examples/pdf/main.go`, `examples/image/main.go` — P6-2 flag + mustSet rewrite
- `api_test.go` — `TestLineLogSeverity` (P6-1 pin)
- `internal/line/line.go`, `internal/line/line_test.go` — P6-1 package (contract #7)
- This session's verification pass made **no further edits** — all rows already matched.

## Validation (verbatim results)

- `gofmt -l` on all owned files → **clean** (no output).
- `go build . ./examples/...` → **FAIL**:
  ```
  # gowkhtmltopdf/internal/convert
  internal/convert/convert.go:47:28: cmd.Global.Allow undefined (type settings.PdfGlobal has no field or method Allow)
  internal/convert/convert.go:48:44: cmd.Global.EnableLocalFileAccess undefined (type settings.PdfGlobal has no field or method EnableLocalFileAccess)
  ```
- `go test .` → **FAIL [build failed]** — same two errors; root tests cannot run until fix-convert lands.
- `go vet . ./examples/...` → **FAIL** — same two errors.
- `go build ./...` → **FAIL confined to internal/convert**: per-package sweep shows OK for
  internal/line, internal/settings, internal/load, internal/cli, internal/layout, internal/html,
  internal/css, internal/pdf, internal/outline, internal/svg; only `internal/imageout` also fails,
  purely transitively (it imports internal/convert). internal/layout builds fine.
- `go test ./internal/line` → **ok** (cached). `go test ./internal/settings` → **ok** (cached).

### Failure attribution (not ours)
`internal/convert/convert.go:45` already calls `load.NewLoader(cmd.Global.Load)` (new shape,
contract #6 landed by fix-settings-cli), but convert.go:47-48 still pokes the pre-migration
top-level fields `cmd.Global.Allow` / `cmd.Global.EnableLocalFileAccess` (now
`cmd.Global.Load.Allow` / `cmd.Global.Load.EnableLocalFileAccess`, per
internal/settings/settings.go:184-188 `LoadGlobal`). These two lines belong to fix-convert's
mid-flight P2-07 landing; per contract rule 5 I did not edit them. Note in my log: the fix for
fix-convert is to **delete lines 47-48** ("callers stop poking fields"; NewLoader applies the
policy). Root-package validation (`go build .`, `go test .`, `go vet .`) must be re-run by
fix-integration (wave 3) after fix-convert lands.

## Remaining markers

- None in files owned by fix-root-api.
- One stale cross-file marker: internal/cli/flags.go:113 `FIX-REVIEW: P1-5 …` (premise resolved;
  see above).
