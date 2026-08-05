# Tier 2 Pending-3 — Research synthesis (2026-08-05)

> **Parent:** [`README.md`](README.md)  
> **Status:** reference (not an execution checklist)  
> **Sources:** web CSS research · web fonts/HF research · codebase explore

---

## How this was produced

Three parallel investigations fed the pending-3 ledgers:

1. **Web / CSS specs** — multicol, transforms, `:has`/`@container`, orphans/widows, flex/grid hard edges, sticky, float↔table
2. **Web / fonts + HF** — WOFF policy, `halt`/`palt`, Indic honesty, Noto path, wkhtmltopdf nested HF architecture
3. **Codebase seams** — `css.go`, `layout/*`, `paint.go`, `hf.go`, `MergeFontFaces`, `ShapeTextFont`

Executable checklists live in sibling `*.md` files; this note captures binding
decisions so implementers do not re-litigate scope.

---

## Binding product decisions

| Decision | Rationale |
|----------|-----------|
| Nested HTML HF **in scope now** (not v0.3.0) | Product request; seams exist in `hf.go` |
| HF model = **wkhtmltopdf child HTML document**, not GCPM running elements | CLI compatibility; code already URL-based HF |
| Multicol + orphans CSS + float/table = **Must** | Close phase 17/18 checklist holes |
| Static 2D transforms + `:has` then `@container` = **Should** | High report ROI; animations/3D out |
| True subgrid / L3 masonry / Chrome parity = **`[~]`** | Spec cost / unstable masonry; lite already shipped |
| Overflow sticky = **`[~]` honesty** | PDF has no scroll |
| WOFF2/remote/Noto bundle/`halt` = **`[~]` policy** | TTF/`--font-path` enough; Brotli needs amendment |
| Optional WOFF1 = stdlib zlib only if prioritized | No new module |

---

## Code seams (quick map)

| Area | Entry |
|------|-------|
| CSS parse | `internal/css/css.go` |
| Style | `internal/layout/style.go` `applyRestProps` |
| Layout dispatch | `internal/layout/layout.go` `build` |
| Flex / grid | `flex.go` / `grid.go` |
| Float | `float.go` |
| Sticky print | `sticky.go` / `applyStickyPrint` |
| Pagination | `paint.go` `paginateOps` / `orphansWidows` |
| HF | `internal/convert/hf.go` (+ proposed `hf_doc.go`) |
| Fonts | `convert.MergeFontFaces` · `pdf/shape_gotext.go` |

---

## Suggested fixture numbers (reserve; do not collide)

| ID | Topic |
|----|-------|
| 36 | Nested HF flex+image (optional golden) |
| 37 | CSS orphans/widows props |
| 38 | Float inside `td` |
| 39 | Multicol article |
| 40 | Static transform badge |
| 41 | `:has()` selector |
| 42 | `@container` inline-size |

---

## Out of scope reminders

- CGO HarfBuzz; extra modules beyond typesetting (unless WOFF2 amendment)
- Bundle full Noto CJK
- Browser HF; HF JS; independent multi-page HF
- CSS animations timelines; 3D transforms; SVG filter graphs
