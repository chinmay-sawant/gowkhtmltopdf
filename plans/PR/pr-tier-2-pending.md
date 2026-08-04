## Summary

Closes Tier 2 follow-ups after #16, plus the three pending polish items:
stdlib-compatible Arabic joining / Indic honesty (plan amendment, no HarfBuzz),
Hangul-capable face path for fixture-27, and fuller flex / grid spans /
rotated CJK vertical.

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

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Entities, flex shrink/order/min-max, z-index, grid spans, Arabic joining, vertical rotate, Hangul glyphs |
| **Dependencies** | None (stdlib); test font is OFL subset only |
| **API / CLI** | Unchanged flags; font-path accepts TT-flavored otf |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `go test ./...`
- [x] Fixture-08 heading shows `Documentation & forms`
- [x] Fixture-27 with `testdata/fonts` embeds `NotoSansKR` Type0 Hangul glyphs
- [x] Arabic joining unit tests; vertical CJK `RotateDeg=90`; grid `span 2` + nested

### Commands

```sh
make test
make lint
```

---

## Related issues

- Relates to #16 (Tier 2 landed)
- Relates to `plans/10-canonical-post-mvp-roadmap.md` phases 17/19 polish

---

## Follow-ups (still out of scope)

- Real HarfBuzz / OpenType GSUB/GPOS (would need deps + cgo — rejected by amendment)
- Full Indic matra reordering / mark positioning
- Full CSS Grid (areas, dense packing, row spans) / iterative flex content sizing
- Full CSS vertical typesetting (tate-chu-yoko, upright CJK, etc.)
