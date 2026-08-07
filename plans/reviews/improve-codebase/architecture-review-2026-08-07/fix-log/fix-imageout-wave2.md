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
| **P5-05** | **done** | `glyphAtlas` per `rasterize` run (mutex + map, max 4096 + half-eviction). Concurrent Renders no longer share package-level mutable cache. Nil atlas in `ttfDrawString` allocates a private one (tests). |
| **P1-1** | **done** | `RunRequest(ctx, *convert.Request, log)` is the engine entry. Old `Run(ctx, *cli.Command, log)` is thin adapter: `OpenOutput` + format sniff into `Image.Format` + build Request. |
| **P5-01** | **done** | `paint` uses `layout.StyleOf` for stroke width + `layout.FakeBoldFor` for text double-draw (CJK/Latin gate). Fill keeps raw `op.R/G/B/Alpha` + `draw.Over` (transparent canvas vs PDF white paper pre-composite) — documented. |
| **P5-02** | **done** | Sheet harvest + font faces shared (`CollectSheets` + `MergeFontFaces`); PDF multi-page assembly remains convert-specific. No further extract without convert importing imageout. |

## Files changed

- `internal/imageout/imageout.go` — P2-07, P1-4, P2-09, P2-01/14, P6-01, P1-1, P5-01 (StyleOf/FakeBoldFor), P5-02 note, P5-05 atlas wire-through
- `internal/imageout/imageout_test.go` — `Global.Load.EnableLocalFileAccess`
- `internal/imageout/fontface_test.go` — ACL on `Global.Load`, NewLoader via `imageLoadGlobalCmd`, ParseDocument, CollectSheets
- `internal/imageout/ttfraster.go` — P5-05 `glyphAtlas` type, per-run ownership, get+evict
- `internal/imageout/baseline_test.go` — `ttfDrawString` atlas arg (nil)

## Validation

```
go build ./...                     → (run after edits)
go test ./internal/imageout/... -count=1  → (run after edits)
```

## Remaining markers

- none in imageout for P5-01 / P5-02 / P5-05

## Notes for orchestrator / other agents

- Image mode ACL home is **`cmd.Global.Load`** (CLI `enablelocalfileaccess` / `allow`); Image.Load only has Proxy from image keys. `imageLoadGlobal` ORs them.
- `RunRequest` requires non-nil `req.Image` and `req.Output` (adapter always supplies both).
- Link media for sheets now tracks layout media (print-media-type no longer silently drops print stylesheets while layout runs in print).
- **P5-01 fill-alpha:** StyleOf pre-composites translucent fills for PDF paper; raster keeps raw alpha on NRGBA by design. FakeBold + stroke come from the layout semantics table.
- **P5-02:** sheet harvest + font faces shared; full page-assembly pipe not extracted (PDF multi-page stays convert-only).
