# Phase 82: Vendor aliases when base is done (tier 3, 48 properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 82
> **Status:** not started
> **Estimated effort:** M (slice 82.1 now; rest gated on Phase 80/83 bases)
> **Owner:** `internal/layout` cascade (`normalizeVendorPrefix`) + catalog
> **Depends on:** Phase 69 alias mechanism; Phase 80 for background bases; Phase 83 for mask bases
> **Unblocks:** prefixed template CSS matching Implemented unprefixed twins
> **Honesty:** `../HONESTY-GATES.md`
> **Inventory:** `../unsupported-triage.json` bucket `C_vendor_prefix_aliases` (**48** names)
> **Subagent scan (2026-08-29):** vendor alias paths

---

## Overview

Phase 82 owns the remaining **48** vendor-prefixed Unsupported names. Rule: **flip a prefixed name to Implemented only when the unprefixed base is already Implemented** and the alias has a prefixed-name test plus consumer evidence (Phase 69 pattern).

Mechanism today: `normalizeVendorPrefix` at `internal/layout/style_cascade.go:788-842`, called from `applyStyleProp` at `:849`. Only `-webkit-` today. **22** aliases already Implemented (Phase 69).

## Standing rules (every agent)

1. **No git commands** unless the user explicitly asks. Do not run `git add`, `git commit`, `git push`, `git restore`, `git clean`, `git reset`, or `git stash`.
2. **Code first, mapping last.** Follow `../HONESTY-GATES.md`. Implemented needs APPLY + FIELD + CONSUMER + TEST + MATRIX + MAPPING.
3. **Catalog sync is mandatory after the phase changes status counts or notes.** Update both:
   - `plans/0.2.6/catalog/mapping.json` (per-property `engine_status`, `code_path`, `notes`)
   - `plans/0.2.6/catalog/coverage-summary.json` (recount `properties_by_engine_status`)
   Also update `plans/0.2.6/property-counts.md` to match.
4. After mapping edits: `python3 scripts/css-catalog-map.py --check` must exit 0. Prefer hand recount over `--write` unless you understand `--write` can bump unrelated apply-arm rows to `partial`.
5. Close `[x]` only with proof (command + exit 0). Use `[~]` with reason, owner, and next gate when deferring inside the owned set.
6. Do not invent property lists. Ownership is locked to `../unsupported-triage.json` for this phase.


## Split of the 48 (from subagent scan)

| Slice | Count | When |
|------:|------:|------|
| A Bases already Implemented (old box flex + text-fill) | 6 | **82.1 now** |
| B Wait on Phase 80 background longhands | 3 | After `background-clip` / `origin` / `size` Implemented |
| C Wait on Phase 83 mask program | 14 | After mask bases Implemented |
| D Print-noop bases (animation/transition/3D/UI) | 20 | Do not flip (bases stay skip) |
| E WebKit-native / other (`line-clamp`, stroke*, size-adjust) | 5 | Only if base/consumer lands |
| **Total** | **48** | |

### A (6) ready now

`-webkit-box-align`, `-webkit-box-flex`, `-webkit-box-ordinal-group`, `-webkit-box-orient`, `-webkit-box-pack`, `-webkit-text-fill-color`.

Also map `display: -webkit-box` / `-webkit-inline-box` in `setDisplayKeyword` (`style_properties.go:67-75`) or box-* aliases write dead flex fields.

### B (3)

`-webkit-background-clip`, `-webkit-background-origin`, `-webkit-background-size`.

### C (14)

`-webkit-mask*` and `-webkit-mask-box-image*` family.

### D (20)

`-webkit-animation*`, `-webkit-transition*`, `-webkit-backface-visibility`, `-webkit-perspective*`, `-webkit-transform-style`, `-webkit-appearance`, `-webkit-user-select`.

### E (5)

`-webkit-line-clamp`, `-webkit-text-size-adjust`, `-webkit-text-stroke*`.

## Work order

1. Extend `normalizeVendorPrefix` (`style_cascade.go:788`) only for claimed aliases.
2. Add value remaps where needed (2009 box keywords).
3. Prefixed-name tests in `style_cascade_test.go` (extend `TestWebkitPrefixAliases` ~188).
4. Flip mapping **only** for packets that pass; leave the rest Unsupported with notes naming the blocking base.
5. Recount `coverage-summary.json`.

## Checklist

- [ ] 82.1.1 Lock all 48 names; record A/B/C/D/E split in this file (counts must sum to 48).
- [ ] 82.1.2 Implement slice A (6) with display `-webkit-box` remap + value remaps + prefixed tests.
- [ ] 82.1.3 Flip mapping for slice A only after flip packets; recount coverage-summary.
- [ ] 82.2.1 After Phase 80 ships background clip/origin/size: add slice B aliases + tests + flips.
- [ ] 82.3.1 After Phase 83 ships mask bases (if ever): add slice C aliases + tests + flips.
- [ ] 82.4.1 Slice D: keep Unsupported; notes say print-noop base.
- [ ] 82.5.1 Slice E: keep Unsupported unless `line-clamp` / stroke consumers land; no mass flip.
- [ ] 82.6.1 Matrix/docs list which prefixes are Implemented vs Unsupported.

### Catalog and gate close

- [ ] CATALOG.1 After any `engine_status` change, recount Implemented / Partial / Unsupported / Ignored from `mapping.json` with a `Counter` on `engine_status`.
- [ ] CATALOG.2 Write the same counts into `catalog/coverage-summary.json` `counts.properties_by_engine_status` and into `property-counts.md`.
- [ ] CATALOG.3 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] CATALOG.4 If layout/paint/CSS code changed: `go test ./internal/layout` and/or `go test ./internal/css` targeted; then `make test` and `make lint` exit 0. If paint/pagination changed: `make golden` exit 0.
- [ ] CATALOG.5 If matrix/docs claims changed: `make claim-scan` exit 0.
- [ ] CATALOG.6 No git commands were run unless the user explicitly asked.


## Forbidden proofs

- Mass-flipping all 48 because "webkit equals standard"
- Unprefixed-only tests as alias proof
- Flipping mask/animation prefixes while bases are Unsupported
- Git commands without explicit user request

## Ownership (48)

### `C_vendor_prefix_aliases` (48)

```
-webkit-animation, -webkit-animation-delay, -webkit-animation-direction
-webkit-animation-duration, -webkit-animation-fill-mode
-webkit-animation-iteration-count, -webkit-animation-name, -webkit-animation-play-state
-webkit-animation-timing-function, -webkit-appearance, -webkit-backface-visibility
-webkit-background-clip, -webkit-background-origin, -webkit-background-size
-webkit-box-align, -webkit-box-flex, -webkit-box-ordinal-group, -webkit-box-orient
-webkit-box-pack, -webkit-line-clamp, -webkit-mask, -webkit-mask-box-image
-webkit-mask-box-image-outset, -webkit-mask-box-image-repeat
-webkit-mask-box-image-slice, -webkit-mask-box-image-source
-webkit-mask-box-image-width, -webkit-mask-clip, -webkit-mask-composite
-webkit-mask-image, -webkit-mask-origin, -webkit-mask-position, -webkit-mask-repeat
-webkit-mask-size, -webkit-perspective, -webkit-perspective-origin
-webkit-text-fill-color, -webkit-text-size-adjust, -webkit-text-stroke
-webkit-text-stroke-color, -webkit-text-stroke-width, -webkit-transform-style
-webkit-transition, -webkit-transition-delay, -webkit-transition-duration
-webkit-transition-property, -webkit-transition-timing-function, -webkit-user-select
```


