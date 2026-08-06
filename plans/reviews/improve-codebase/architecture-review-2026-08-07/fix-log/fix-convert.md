# fix-convert — architecture-review-2026-08-07

Agent: **fix-convert** (owns `internal/convert/*` only)

## Per-CID status

| CID | Status | Notes |
|---|---|---|
| **P2-07** | **done** | Deleted loader pokes (`loader.Allow` / `loader.EnableLocalFileAccess`). `load.NewLoader(req.Global.Load)` applies policy. Tests use `cmd.Global.Load.EnableLocalFileAccess` / `cmd.Global.Load.Allow`. |
| **P1-4** | **done** | `mediaFor` → thin wrapper: project `obj.Load` media fields onto `settings.Web`, call `settings.ResolveMedia("print", g.Web, objWeb)`. |
| **P1-5** | **done** | Engine reads `req.Global.DumpOutline` only. Adapter ORs legacy `cmd.DumpOutline` into `req.Global.DumpOutline` before `Run`. |
| **P2-09** | **done** | Production loaded-body path: `html.ParseDocument(res.Body)` in `renderObject`. TOC/HF string templates still use `html.Parse`. Golden perf helper updated. |
| **P2-02** | **done** | `collectObjectHeadings` no longer mutates `h.Page`. `flatHeadings` sets `h.DocPage = st.offset + h.Page` once. Consumers (`emitOutline`, TOC links, `headingDest`, HF section) use DocPage / DocPage view. |
| **P2-05** | **done** | Local sort comparator replaced (DocPage order in `flatHeadings`). Local `sectionOf` removed; `drawHeadersFooters` calls `outline.SectionOf(headingsDocPageView(headings), p-tocTotal)`. `BuildTree` receives `headingsDocPageView` so outline `SortHeadings` (keys on Page) sees document pages. |
| **P2-06** | **done** | `hfGeom.pdfY` / `pdfXY` / `pdfRect` own canvas→PDF. `pageRect`/`destPoint`/`locationOf` deleted. `headingDest` uses `h.X/Y/W/H` + local page. |
| **P2-08** | **done** | `docTitle` → `TextContentOf("title")`. `CollectSheets` uses `root.Walk`. |
| **P3-02** | **done** | `parseExcludeSelectors` → `css.ParseSelectors`; appends full comma list; `line.Emit` on invalid. |
| **P6-01** | **done** | Explicit `warning:`/`info:` sites → `line.Emit(Warn/Info, …)`. Bare phase progress stays `fmt.Fprintf` (no `info:` prefix). |
| **P1-1** | **done** | `type Request` + `func Run(ctx, *Request, log, progress)`. `RunPDF`/`RunPDFContext` thin adapters (`OpenOutput` + DumpOutline OR). |
| **P2-01 / P2-14** | **done** | Exported `CollectSheets(..., SheetOptions, log)` + `SheetOptions`. Call sites in convert/HF use it. |
| **P2-03** | **done** | `pagePlan` with `OwnerOf`/`Remap`/`Ranges`/`LogicalN`; HF + link remap consume it. |
| **P2-11** | **done** | `newHFGeom` + `objectState.bodyLayoutOpts` share geometry/layout option construction. |
| **P2-12** | **partial** | Unused `cmd` dropped from `applyInternalLinks`. `paintCount` still swallows paint errors (returns 1). Marker left. |
| **P2-13** | **done** | Uses `layout.DeactivateOp` for link neutralization (sentinel deleted). |
| **P2-10** | **deferred** | `objectState` not slimmed (risk); no marker at a single call site — logged here. |
| **P5-01** | **done** | HF `paintLayoutOps` → `layout.PaintBand` for visual ops; link annotations remain convert-side. |
| **P5-02** | **deferred** | Full page-assembly pipe extract not landed (wave-2/imageout shared); logged. |
| **P3-03** | **done** | TOC `lengthToPt` delegates to `css.LengthToPt` after `css.ParseLength`. |

## Files changed

- `internal/convert/convert.go` — Request/Run, loader policy, mediaFor, CollectSheets, line.Emit, pagePlan, ParseDocument, BuildTree DocPage view
- `internal/convert/outline.go` — DocPage assembly, parseExcludeSelectors, docTitle, emitOutline pdfXY
- `internal/convert/links.go` — pdfGeom helpers consumers, DeactivateOp, DocPage destinations
- `internal/convert/hf.go` — hfGeom.pdf*, pagePlan HF, SectionOf, CollectSheets, PaintBand, line.Emit
- `internal/convert/toc.go` — Request/Global, LengthToPt, DocPage pageOf, line.Emit
- `internal/convert/convert_test.go`, `golden_test.go`, `phase6_test.go`, `fontface_test.go` — Load.* ACL homes, ParseDocument

## Validation

```
gofmt -w internal/convert/*.go
go build ./internal/convert/...   # GREEN
go vet ./internal/convert/...     # GREEN
go test ./internal/convert/... -count=1   # GREEN (ok, ~1.5–2s)
```

## Remaining markers

- `internal/convert/toc.go` — `// FIX-REVIEW: P2-12 paintCount swallows paint errors (returns 1); …`
- P2-10 objectState slim / P5-02 shared pipe: deferred in this log (no silent drop of required MUST rows).

## Cross-package notes for orchestrator

- `outline.SortHeadings` / `SectionOf` still key on `Heading.Page`. Convert uses `headingsDocPageView` so document-global consumers see DocPage projected into Page without mutating shared object-local Page.
- `settings.ResolveMedia` takes `*Web`; object PDF media lives on `LoadPage` — convert projects Load→Web at the mediaFor boundary.
- `layout.PaintBand` + `layout.DeactivateOp` consumed (were available by wave-1 layout).
