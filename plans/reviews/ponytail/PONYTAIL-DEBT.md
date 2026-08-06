# Ponytail debt ledger

> Harvested from `// ponytail:` markers after Phases 0–4 remediation on
> `chore/ponytail-review-1` (2026-08-07).
>
> Convention: `ponytail: <ceiling>, <upgrade trigger>`.

## Markers

| File | Ceiling | Upgrade trigger |
|------|---------|-----------------|
| `api.go` (Convert / ImageConvert) | Path-based `cli.Command` DTO + temp file for library bytes | Embedders need zero temp files → add writer/bytes sink in convert/imageout |
| `internal/cli/doc.go` | Multi-object wkhtml CLI grammar; Policy A flag surface only | New flag only when convert/load/imageout reads it |
| `internal/settings/settings.go` | ColorMode not stored (Grayscale only); PageSize dual name mirror; `Ignored` sink for inert keys | Second real consumer of a key → promote to typed field |
| `internal/html/html.go` | Custom `Node` tree (Parent/Attrs/void) | Layout/CSS rewritten to `x/net/html` — not a free delete |
| `internal/html/entities.go` | Thin `Contains("&")` gate over `html.UnescapeString` | Measurable win from always-unescape → drop gate |
| `internal/css/container.go` | Full `@container` Cond tree (`and`/`or`/`not`) | No nested and/or in real sheets → shrink to single comparisons |
| `internal/layout/grid.go` | `subgrid` → ordinary grid; `masonry` keyword → dense auto-flow only | Product needs Grid L3 masonry/subgrid → reintroduce pack/inherit |
| `internal/svg/raster.go` | canvas sole SVG raster path | Thinner SVG dep or pre-rasterized logos only |
| `internal/pdf/fonttype0.go` | Type0 + simple dual embed (Latin-1 vs CJK) | Single embed model if product drops one script class |
| `internal/pdf/faces.go` | Liberation faces bundled in-tree | Always require system fonts → drop assets |
| `internal/pdf/woff.go` | WOFF1 in-tree; no WOFF2/Brotli direct dep | Product needs WOFF2 → accept Brotli or host-decode |
| `internal/pdf/shape.go` / `shape_gotext.go` | Manual Arabic joining when no GSUB; OT via go-text when available | All production faces have GSUB → drop manual tables |
| `internal/imageout/imageout.go` | Fake-bold double-draw for missing bold face | Synthetic bold outlines in pdf faces |

## Summary

**18 markers** across production Go. **0 `no-trigger`** tags — every row names a ceiling and upgrade path.

```
rg -n '// ponytail:' --glob '*.go' | wc -l   # expect ≥ 15
```

## Intentional non-markers (documented elsewhere)

- TOC fixed-point / `cloneResult` — convert package comments + phase plans
- wkhtml multi-object grammar — `internal/cli` package doc
- Policy A ignored settings keys — `internal/settings` package doc + `Ignored` maps
