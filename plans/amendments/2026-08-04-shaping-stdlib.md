# Plan amendment: complex-script shaping under stdlib / no-cgo

> Date: 2026-08-04  
> Branch: `feature/tier-2-pending`  
> Amends: `plans/phases/phase-19-fonts-i18n.md`, product constraint in
> `plans/10-canonical-post-mvp-roadmap.md`

## Decision

**Do not** introduce the C HarfBuzz library or any third-party Go module for
shaping. The product rules remain:

1. Go **stdlib only** (no `require` deps beyond the module itself)
2. **`CGO_ENABLED=0`** static builds (CI gate)

Linking HarfBuzz would violate both (cgo + system shared library).

## What we ship instead (same *goals*, different mechanism)

| Goal | Mechanism |
|------|-----------|
| Arabic joining (initial/medial/final/isolated) | In-tree Unicode Arabic Presentation Forms mapping + join-type state machine |
| RTL visual order | Existing run reverse in `ShapeText` (kept) |
| Indic (Devanagari et al.) | Best-effort: NFC normalize + document **no** full reordering/matra positioning; optional virama-aware skip |
| Hangul | Discover/use a Hangul-capable face via `--font-path` / system fonts; CI uses `testdata/fonts/NotoSansKR-HangulSubset.ttf` (OFL) |
| Full OpenType GSUB/GPOS | **Still deferred** |

## Honesty language

Docs must say: “Arabic joining is **best-effort presentation-form mapping**,
not HarfBuzz / OpenType shaping. Indic production output is **not claimed**.”

## Acceptance

- Arabic sample with joining forms renders differently from raw isolate codes
  when a face has presentation glyphs
- Hangul fixture proves glyphs with a Noto CJK (or similar) path
- CI stays `CGO_ENABLED=0` and `go.mod` has no third-party requires
