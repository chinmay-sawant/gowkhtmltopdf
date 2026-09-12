# Ponytail packet 3 - layout

> **Scope:** `internal/layout/` and its focused tests
> **Canonical rows:** `PT26-LAY-01` through `PT26-LAY-12`
> **Method:** current caller search and focused source reading. No files changed and no tests ran.

## Dormant experiments and unused state

`internal/layout/parallel.go:L17-454`: **delete** the opt-in `ParallelLayout` experiment. Its only caller is `parallel_test.go:L221`; production conversion uses serial layout. Estimated cut: about 454 production lines plus 308 test lines. Prove its existing parallel tests before deletion, then run layout tests and `make golden`.

`internal/layout/section_clone.go:L14-138`: **delete** `ErrSectionClone`, `SectionChromeHash`, and `CloneSectionChrome`. Non-definition uses are only `section_clone_test.go` and `independent_blocks_test.go`. Estimated cut: about 138 production lines plus tests. Prove targeted tests before deletion, then run layout tests and `make golden`.

`internal/layout/independent_blocks.go:L11-86`: **delete** test-only `PaintMetadata`, `CopyPaintMetadata`, and `locationsFromBoxes`. Their uses are in `independent_blocks_test.go:L86,117`. Estimated cut: 86 production lines plus tests. Prove with `go test ./internal/layout -run IndependentBlocks`.

`internal/layout/style.go:L348-362`, `style_advanced_props.go:L62-120`, `style_cascade.go:L284-294`, `style_intern_gen.go`: **yagni** delete unread image, color-scheme, containment, and content-visibility state. It is written, inherited, and interned but has no layout, paint, or image reader. Keep `MarginTrim`, which `layout_flow.go:L227` reads. Estimated cut: 140-180 lines. Update mapping and tests, then run layout tests and `make golden`.

`internal/layout/style.go:L363-375`, `style_advanced_props.go:L148-191`, `style_cascade.go:L297-314`, `style_intern_gen.go`: **yagni** delete unread variable-font, bidi, and decoration state. Keep `TextDecorationSkipInk`, read by `inline_paint.go:L563`. Estimated cut: 130-170 lines. Update mapping and tests, then run layout tests and `make golden`.

`internal/layout/inline_paint.go:L1236-1294`, `internal/layout/pseudo_content.go:L500-553`: **delete** unreachable inline-bullet fallback and its private marker chain. `emitInlineBullet` has no caller. Estimated cut: 113 lines. Prove list tests and `make golden`.

`internal/layout/style_logical_border.go:L9-102`: **delete** the unused logical-border utility subgraph. Retain logical-radius support from line 120 onward. Estimated cut: 94 lines. Prove logical-border tests and `make golden`.

## Caches and style views

`internal/layout/advance_cache.go:L13-44`, `layout.go:L624-628`, `layout_flow.go:L773`, `inline_paint.go:L1496,L1539`: **yagni** remove the per-engine glyph cache and use `face.AdvanceInPoints` directly. Estimated cut: 120-130 lines. Run layout and golden tests, then record a benchmark before closure.

`internal/layout/style_memo.go:L10-153`, `style.go:L508,L625`: **yagni** remove the style-resolution memo. A miss already calls the ordinary resolver and `styleStore.append` remains the sharing boundary. Estimated cut: 600-630 lines. Run layout and golden tests, then record a benchmark before closure.

`internal/layout/style_views.go:L3-69`: **shrink** remove `boxModelStyle` and `boxModelStyleOf`. Same-package sizing helpers can accept `*ResolvedStyle`. Estimated cut: 90-120 lines. Prove layout tests and `make golden`.

`internal/layout/paint_style_view.go:L3-33`: **shrink** remove `paintChromeStyle` and `paintChromeStyleOf`. The local chrome checks can read `boxNode.style` after their existing nil checks. Estimated cut: 50-65 lines. Prove chrome, outline, and overflow tests, then `make golden`.

`internal/layout/layout.go:L1261-1268`: **delete** unread `ResolvedStyle.VertChrome`. Retain `HorizChrome`, read in `container.go:L122,133`. Estimated cut: 8 lines. Prove with `go test ./internal/layout`.

## Rejected lookalikes

- Keep `IndependentBlocksForOptions`: `internal/convert/convert.go:L641` calls it in production.
- Keep `ResolveStyles` and `NodeWithWorkspace`: `internal/convert/page_blocks.go:L23,37` uses them for certified blocks.
- Keep `ctxPoll`: active layout loops construct it in flex, grid, and table code.
- Keep `opOwnedBy`: pagination chrome, overflow clipping, and sticky sealing call it.

Packet estimate: about -2,500 source and test lines before shared imports and benchmark decisions.
