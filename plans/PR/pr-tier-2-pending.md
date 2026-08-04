## Summary

Closes Tier 2 follow-ups after #16: HTML entity decode, fuller flex (shrink/basis/order), float `%` + right packing, lite `z-index`, TrueType-flavored `.otf` discovery, vertical-rl stacked glyphs, and docs honesty (no HarfBuzz under stdlib).

---

## Motivation / context

- Plans: Tier 2 pending after merge of #16
- Branch: `feature/tier-2-pending` from `master` @ #16 merge
- Constraint: **stdlib-only** — HarfBuzz / full OpenType shaping remains out of scope

---

## Changes

### Misc

- Decode HTML entities in text + attributes (`&amp;` → `&` on fixture-08)
- README / fidelity / fonts.md nits for flex, CJK, entities, z-index

### Phase 17 leftovers

- Float `width:%` covered + right-float packing improvement; `clear` unchanged semantics with tests
- Lite `z-index` on positioned boxes (ops + chrome paint sort)
- Flex: `flex-shrink`, `flex-basis` (%/length), `order`

### Phase 19 leftovers (stdlib-safe)

- Scan `.otf` when TrueType-outlined; clear error for CFF/`OTTO`
- `writing-mode: vertical-rl|lr` lite (stacked glyphs, not rotated CJK)
- Docs: Hangul needs Hangul-capable face; **no HarfBuzz**

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Entities, flex shrink/order, z-index paint, vertical lite |
| **Dependencies** | None |
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
- [ ] Optional: Hangul with `fonts-noto-cjk` on `--font-path`

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

- Real HarfBuzz / Arabic joining / Indic (needs plan amendment + deps)
- Full flex algorithm (flex-grow fractional iterations, min/max content)
- Full CSS vertical typesetting / glyph rotation
- Bundled Hangul face
