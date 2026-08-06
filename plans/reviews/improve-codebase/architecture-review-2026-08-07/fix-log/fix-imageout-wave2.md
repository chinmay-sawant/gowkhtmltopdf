# Fix log — fix-imageout-wave2

Rows: P2-07, P1-4, P2-09, P2-08, P2-01/P2-14 (consume), P6-01, P5-05 (atlas), P1-1 (imageout), P5-01/P5-02 markers.

## Per-CID status

| CID | Status | Notes |
|---|---|---|
| **P2-07** | **done** | Removed `loader.Allow` / `loader.EnableLocalFileAccess` pokes. `imageLoadGlobal(global, image)` merges Image.Load Proxy with Global.Load ACL (`Allow` + `EnableLocalFileAccess`); `load.NewLoader` applies full policy. Tests use `cmd.Global.Load.*` and `imageLoadGlobalCmd`. |
| **P1-4** | **done** | `mediaFor` → `settings.ResolveMedia("screen", web, objWeb)`. Merges Image.Web + Global.Web.PrintMediaType/MediaType; maps object `Load`/`Web` media fields into a temporary `*settings.Web`. Same media string feeds layout and `CollectSheets` link gating. |
| **P2-09** | **done** | Production `html.Parse(string(res.Body))` → `html.ParseDocument(res.Body)`. Font-face tests use ParseDocument too. |
| **P2-08** | **done** (local) | Local `collectSheets`/`styleText` custom walks deleted (replaced by convert gatherer). No remaining imageout DOM walk twins. |
| **P2-01 / P2-14** | **done** | `convert.CollectSheets` + `SheetOptions{ViewportW:768, ViewportH:576, MediaType: media}` (media from ResolveMedia — fixes prior hardcoded `"screen"` link filter vs print layout bug). Local `collectSheets`/`styleText`/`linkStylesheet` deleted. No import cycle (convert does not import imageout). |
| **P6-01** | **done** | Warning/info `fmt.Fprintf`/`Fprintln` sites → `line.Emit` (`line.Warn` / `line.Info`). |
| **P5-05** | **partial** | Glyph atlas still package-level (shared across Renders) but **bounded** at 4096 entries with half-eviction on insert (`maxGlyphCache`). Full per-Render ownership deferred (phase Future higher risk). |
| **P1-1** | **done** | `RunRequest(ctx, *convert.Request, log)` is the engine entry. Old `Run(ctx, *cli.Command, log)` is thin adapter: `OpenOutput` + format sniff into `Image.Format` + build Request. |
| **P5-01** | **FIX-REVIEW** | Layout PaintBand/PaintStyle never landed; marker on `paint`. |
| **P5-02** | **FIX-REVIEW** | Full shared page-assembly pipeline deferred; marker on `paint`. CollectSheets/MergeFontFaces already shared. |

## Files changed

- `internal/imageout/imageout.go` — P2-07, P1-4, P2-09, P2-01/14, P6-01, P1-1, P5-01/02 markers
- `internal/imageout/imageout_test.go` — `Global.Load.EnableLocalFileAccess`
- `internal/imageout/fontface_test.go` — ACL on `Global.Load`, NewLoader via `imageLoadGlobalCmd`, ParseDocument, CollectSheets
- `internal/imageout/ttfraster.go` — P5-05 maxGlyphCache bound + eviction

## Validation

```
go build ./internal/imageout/...   → ok
go vet ./internal/imageout/...     → ok
go test ./internal/imageout/... -count=1  → ok (all pass)
```

(Briefly blocked while `internal/convert` mid-edit; re-ran after convert compiled.)

## Remaining markers

- `// FIX-REVIEW: P5-01` on `paint` (layout paint-semantics table missing)
- `// FIX-REVIEW: P5-02` on `paint` (full prologue/pipeline extract)

## Notes for orchestrator / other agents

- Image mode ACL home is **`cmd.Global.Load`** (CLI `enablelocalfileaccess` / `allow`); Image.Load only has Proxy from image keys. `imageLoadGlobal` ORs them.
- `RunRequest` requires non-nil `req.Image` and `req.Output` (adapter always supplies both).
- Link media for sheets now tracks layout media (print-media-type no longer silently drops print stylesheets while layout runs in print).
