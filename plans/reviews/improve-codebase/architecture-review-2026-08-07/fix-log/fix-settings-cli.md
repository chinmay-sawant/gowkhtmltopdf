# fix-settings-cli — wave-1 remediation log (2026-08-07)

Agent: fix-settings-cli · owns `internal/settings/*` + `internal/cli/*`

## Per-CID status

| CID | Status | Summary |
|---|---|---|
| P1-2 | done | Stringly-typed flag contract replaced: `flagKind` (flagBool/flagValue/flagPair), `flagSpec.app func(c, ctx, vals []string)`. `apply` now dispatches on `spec.kind`; `parseBool` handles negated/inline once; pair flags deliver `vals[0],vals[1]` via `setMapEntry`. Deleted `pairSep`, `isPairFlag`, `splitPair`, `boolVal`. `settings.Set` keeps its string vocabulary. |
| P1-3 | done | One router `(ctx *objectCtx) applyPage(c, glob, obj, val)` in cli.go (global → current object → pending promotion). `pageOnlyFlag` added for zoom/username/password/timeout/external-links/internal-links — rejects the pre-object position loudly (`option must follow a page/cover/toc object`). `hfFlag`/`tocFlag` keep explicit global-only pre-object routing (pending never created); `tocFlagBool` delegates through them. Hand-rolled pending inside `printMediaFlag` deleted. |
| P1-4 (cli) | done | `printMediaFlag` writes one field home — `Global.Web.PrintMediaType` via the router — plus the `load.printmediatype` object override; `Image.Web.PrintMediaType` leg deleted. `ResolveMedia(base string, global Web, obj *Web) string` exported in internal/settings (semantics: obj||global PrintMediaType → "print"; obj.MediaType → print/screen; global.MediaType → print/screen; fallback base). fix-convert (`"print"`) and fix-imageout (`"screen"`) consume it in later waves. |
| P1-5 | done (cli + settings) | dump-outline/dump-default-toc-xsl appliers now write `c.Global.Set("dumpoutline"/"dumpoutlinewithdefaulttocxsl", vals[0])` — negation rides the value (`--no-dump-outline`, `--dump-default-toc-xsl=false` work). `TestDumpOutlineCommandFieldOnly` replaced by `TestDumpOutlineGlobalHome`. `// FIX-REVIEW: P1-5` marker left at the registration for cmd/gowkhtmltopdf/main.go (`Command.DumpDefaultTOCXSL` read) → fix-root-api; convert.go:130 OR read → fix-convert. `Command.Dump*` fields kept (API stable). |
| P1-6 | done | reflect.go rewritten as one generic field-descriptor table: `field[T]`, `keyTable[T]`, generic `setForKey`/`getForKey` (repo Go is 1.26 — generics used), `sub()` adapter, `subTable()` builder for HeaderFooter/TableOfContent/Web/LoadPage. The 8 apply/get switches collapsed; lazy `sync.Once`/`ensureKeyTables` deleted (plain `var` tables built in `init`). Key names, ignored-keys storage, and Set/Get error messages preserved. `TestKeyTableSetGetParity` rewritten against the single tables. |
| P1-7 (settings) | done | `ApplyImageKey(global *PdfGlobal, img *ImageGlobal, name, value string) error` exported (normalizeDots; "background"/"web.background" → `global.Set("background", …)`; else `img.Set`). fix-root-api's `ImageConverter.Set` becomes a one-line delegate (their file). |
| P6-3 (httperror) | done | `HttpErrorCode(int)` moved from reflect.go into httperror.go next to `HttpStatusError`; `func (e *HttpStatusError) HttpErrorCode() int` added. `cli.ExitCode(err error) int` added (errors.As → `interface{ HttpErrorCode() int }`, else `cli.ExitError`). Mains migration is wave 2 (fix-engine-migration-wave2). |
| P2-07 (settings) | done | `LoadGlobal` gains `Allow []string` + `EnableLocalFileAccess bool`; fields MOVED off `PdfGlobal`; reflect keys `allow`/`enablelocalfileaccess` re-routed to `Load.Allow`/`Load.EnableLocalFileAccess` (CLI behavior byte-identical). `load.NewLoader` already consumes these (fix-html-load-outline's side, now compiles). Caller pokes `loader.Allow = cmd.Global.Allow` / `loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess` in convert.go:47-48 / imageout.go:422-423 now fail to compile → **contract #6 fallout**, see below. |
| P2-04 (settings) | done | `LoadPage` gains `InlineHTML []byte` + `InlineBase string` (fed through `PdfObject.Load`). `internal/load` already consumes them (fix-html-load-outline's side, now compiles). |
| P2-09 | done (marker) | `// FIX-REVIEW: P2-09 default-encoding ignored; engine decodes UTF-8/ASCII only (html/load decode seam is fix-html-load-outline's)` added at the `defaultencoding` registration in `ignoredGlobalKeys`. No code change (the html/load seam is another agent's row). |
| P1-1 (cli note) | n/a | No `cli.Command.ToRequest()` added (would create an import cycle in wave 1). The Request adapter lives in internal/convert (fix-convert, contract #1). `cli.Command` left as-is. |

## Files changed

- `internal/settings/reflect.go` — descriptor-table registry rewrite (P1-2 helpers live in cli; here: P1-6, P1-7, P2-07 keys, P2-09 marker, P6-3 move out).
- `internal/settings/getters.go` — three `Get` bodies collapse to `getForKey`.
- `internal/settings/settings.go` — `ResolveMedia` export (P1-4); `LoadGlobal.Allow`/`EnableLocalFileAccess` (P2-07); `LoadPage.InlineHTML`/`InlineBase` (P2-04); `PdfGlobal` field move; Dump* comment update (P1-5).
- `internal/settings/httperror.go` — `HttpErrorCode` + `(*HttpStatusError).HttpErrorCode` (P6-3).
- `internal/settings/settings_test.go` — `allow` assertion → `g.Load.Allow`; `TestKeyTableSetGetParity` rewritten; added `TestResolveMedia`, `TestApplyImageKeyBackgroundAlias`.
- `internal/cli/cli.go` — `flagKind`/`flagSpec`/`apply`/`parseBool`/`applyPage`/`lookupFlag` (P1-2, P1-3); `ExitCode` (P6-3).
- `internal/cli/flags.go` — full flag-registration rewrite (kinds, vals, router, pageOnlyFlag, hfFlag/tocFlag/printMediaFlag thin wrappers, dump flags → Global, pair flags via setMapEntry).
- `internal/cli/help.go` — `spec.kind == flagBool`.
- `internal/cli/cli_test.go` — `EnableLocalFileAccess` assertions → `Load.EnableLocalFileAccess`; `TestLoadFlags` reordered (page keyword; Image.Web.PrintMediaType now must be false); added `TestPageOnlyFlagPreObjectRejected`, `TestDumpOutlineGlobalHome`, `TestExitCode`.

## Validation (verbatim)

```
$ go build ./internal/settings/... ./internal/cli/...
BUILD-OK
$ go test ./internal/settings/... ./internal/cli/...
ok  	gowkhtmltopdf/internal/settings
ok  	gowkhtmltopdf/internal/cli
$ go vet ./internal/settings/... ./internal/cli/...
VET-OK
$ go build ./internal/load/...   → ok   (was failing before this wave; my P2-04/P2-07 fields fixed it)
$ go test ./internal/load/...    → ok
```

Repo-wide `go build ./...` currently stops at `internal/convert/convert.go:47-48`
(`cmd.Global.Allow` / `cmd.Global.EnableLocalFileAccess` undefined) — **expected
contract #6 fallout**: fix-convert must drop the loader pokes
(`load.NewLoader(cmd.Global.Load)` now applies the full policy); the identical
pokes at `internal/imageout/imageout.go:422-423` (fix-imageout-wave2, P2-07)
will surface once convert compiles. Not fixable from my packages (their files).

## Remaining markers

- `internal/settings/reflect.go` — `// FIX-REVIEW: P2-09 default-encoding ignored; engine decodes UTF-8/ASCII only (html/load decode seam is fix-html-load-outline's)`.
- `internal/cli/flags.go` — `// FIX-REVIEW: P1-5 cmd/gowkhtmltopdf/main.go still reads Command.DumpDefaultTOCXSL — fix-root-api must switch to cmd.Global.DumpDefaultTOCXSL; convert.go:130 OR read is fix-convert's.`

## Cross-package notes

- fix-convert: convert.go:47-48 loader pokes break (contract #6); convert.go:130 `cmd.DumpOutline || cmd.Global.DumpOutline` OR can collapse to `cmd.Global.DumpOutline`; convert.go `mediaFor` should delegate to `settings.ResolveMedia("print", …)`.
- fix-imageout-wave2: imageout.go:422-423 pokes break (contract #6); imageout.go `mediaFor` → `settings.ResolveMedia("screen", …)`.
- fix-root-api: main.go dump-default-toc-xsl read; `ImageConverter.Set` → one-line `settings.ApplyImageKey` delegate; mains error handling → `cli.ExitCode`.
- fix-html-load-outline: P2-04/P2-07 settings fields are now in place (load compiles).
- Not mine: internal/pdf `strconv` unused in fontpdf.go (fix-pdf-fonts, transient).
