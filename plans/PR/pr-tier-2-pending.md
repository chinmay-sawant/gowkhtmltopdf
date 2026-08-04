## Summary

Closes Tier 2 follow-ups after #16: HTML entities, fuller flex/grid/z-index,
stdlib-safe Arabic joining + Hangul face path, vertical CJK rotation, and
TrueType subset fixes so composite CJK (e.g. `東京都、`) renders cleanly in
PDF viewers to match the HTML.

---

## Motivation / context

- Plans: Tier 2 pending after merge of #16
- Branch: `feature/tier-2-pending` from `master` @ #16 merge
- Constraint: **stdlib-only** — real HarfBuzz / CGO remains out of scope
- Amendment: [`plans/amendments/2026-08-04-shaping-stdlib.md`](../amendments/2026-08-04-shaping-stdlib.md)

---

## Changes

### Misc

- Decode HTML entities in text + attributes (`&amp;` → `&` on fixture-08)
- README / fidelity / fonts.md updated for flex, CJK, Arabic joining, Hangul

### Phase 17 leftovers

- Float `width:%` covered + right-float packing improvement; `clear` unchanged semantics with tests
- Lite `z-index` on positioned boxes (ops + chrome paint sort)
- Flex: `flex-shrink`, `flex-basis` (%/length), `order`, post-grow/shrink **min/max-width clamp**
- Grid: occupancy placement, `grid-column: span N` / `start / end`, nested grids

### Phase 19 leftovers (stdlib-safe)

- Scan `.otf` when TrueType-outlined; clear error for CFF/`OTTO`
- `writing-mode: vertical-rl|lr`: stacked Latin upright + **90° rotated** ideographic/Hangul/kana
- **Arabic presentation-form joining** + Lam-Alef in `ShapeText` (not OpenType)
- Hangul: vendored OFL subset under `testdata/fonts/` + fixture-27; `--font-path` in samples/golden
- Per-rune CSS `font-family` fallback (Droid for Han, Noto KR for Hangul)

### PDF font subsetting (CJK fidelity)

- Pad `glyf`/`loca` to **4-byte** boundaries (unaligned offsets corrupted composites in viewers)
- Preserve **LSB** in subset `hmtx`
- **Strip TrueType hint bytecode** from simple/composite outlines (subsets omit `fpgm`/`prep`/`cvt`)
- Keep **full-em** CJK punctuation metrics (half-em compression cramped `東京都、` vs HTML+Droid)
- Regression: `TestSubsetGlyfFourByteAligned`; regenerate sample PDFs

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Entities, flex/grid/z-index, Arabic joining, vertical rotate, Hangul+CJK glyphs, clean PDF CJK outlines |
| **Performance** | Unchanged layout path; slightly smaller embedded fonts after hint strip |
| **Memory** | Negligible |
| **API / CLI** | Unchanged flags; font-path accepts TT-flavored otf |
| **Dependencies** | None (stdlib); test font is OFL subset only |
| **Binary size / build time** | Sample PDFs smaller from hint-stripped subsets |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `go test ./...`
- [x] `make lint` / CI `test + lint` + `static build (CGO_ENABLED=0)`
- [x] Fixture-08 heading shows `Documentation & forms`
- [x] Fixture-27 with `testdata/fonts` + Droid: Han/kana + Hangul Type0 glyphs
- [x] Arabic joining unit tests; vertical CJK `RotateDeg=90`; grid `span 2` + nested
- [x] Subset glyf 4-byte aligned + hints stripped; `東京都、` spacing matches HTML+Droid

### Commands

```sh
make test
make lint
make samples
```

---

## Screenshots / sample output

```
output/fixture-27-cjk-fontpath.pdf — 国际化报告 / こんにちは / 汉字与假名：東京都、上海、深圳。 / 안녕하세요
```

---

## Related issues

- Relates to #16 (Tier 2 landed)
- Relates to `plans/10-canonical-post-mvp-roadmap.md` phases 17/19 polish

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues filled
- [x] Filled body under `plans/PR/pr-tier-2-pending.md`

---

## Follow-ups (out of scope)

- Real HarfBuzz / OpenType GSUB/GPOS (would need deps + cgo — rejected by amendment)
- Full Indic matra reordering / mark positioning
- Full CSS Grid (areas, dense packing, row spans) / iterative flex content sizing
- Full CSS vertical typesetting (tate-chu-yoko, upright CJK, etc.)
- OpenType `halt`/`palt` for fonts that expose them (Droid has no halt; HTML+Droid is full-em)

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed
