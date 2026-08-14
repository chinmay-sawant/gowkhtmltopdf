# Phase 2 — Layout Primitives (Untagged)

**Status:** ✅ COMPLETED — packages created (`layout`, `image`); not yet wired into `engine.Generate()`  
**Depends on:** Phase 1 complete  
**Base plan refs:** §4 (`layout`, `image`, `content`), §5 Phase 2–3 (untagged), §11 Phase B

---

## Goal

Add real drawing: text runs, tables, borders, multi-page flow, and raster images — still **untagged**, still **no PDF/A**.

---

## Package layout

- [x] `engine/layout/` — margins, flow, tables, page break
- [x] `engine/image/` — JPEG/PNG decode → XObject (DeviceRGB OK)
- [x] Extend `engine/content/` — path operators, text state, `Do` for images
- [x] Extend `engine/page/` — multi-page kids, per-page content streams

---

## Checklist — text & geometry

- [x] Positioned text with font size / color
- [x] Text escape for PDF string literals `(…)`
- [x] Path borders: `m` / `l` / `re` / `S` / `f`
- [x] Page margins and content box
- [x] Multi-page: content overflow creates new page objects + updates `/Count`

## Checklist — tables

- [x] Fixed-column table grid
- [x] Cell text with wrapping (basic)
- [x] Cell borders (vector)
- [x] Header row styling (visual only; no `/TH` structure yet)
- [x] Table can span pages (row split or page-break between rows)

## Checklist — images

- [x] JPEG → `/Subtype /Image` with `/Filter /DCTDecode`
- [x] PNG → decode RGB, `/Filter /FlateDecode`
- [x] XObject keys: `/Type /XObject`, `/Subtype /Image`, `/Width`, `/Height`, `/ColorSpace /DeviceRGB` (or gray), `/BitsPerComponent`, `/Length`
- [x] Image drawn via content `cm` + `/ImN Do`
- [x] Optional: dedupe identical image bytes → one XObject

## Checklist — resources

- [x] Per-page `/Resources` with `/Font` and `/XObject` maps
- [x] Content stream compression with `/Filter /FlateDecode` (recommended)

## Checklist — fixtures

- [ ] Fixture: single page title + paragraph
- [ ] Fixture: 3×3 table
- [ ] Fixture: multi-page long table
- [ ] Fixture: page with embedded JPEG and/or PNG

---

## Acceptance criteria

- [ ] All fixtures open cleanly in a viewer
- [ ] Multi-page `/Count` matches actual pages
- [ ] Images visible and correctly sized
- [ ] Manual visual parity vs simple gopdfsuit templates (spot-check)

---

## Explicitly out of scope

- [ ] BDC / EMC / structure tree
- [ ] ICCBased color / OutputIntent
- [ ] Liberation embedding (use standard fonts or phase-3 prep only)
- [ ] veraPDF gates

---

## Done when

Engine can emit multi-page documents with text, tables, borders, and images without compliance claims.
