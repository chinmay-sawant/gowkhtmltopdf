# Phase 81: Niche or draft (tier 4, 94 properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 81
> **Status:** not started (honest Unsupported notes; no engine promotions expected)
> **Estimated effort:** M
> **Owner:** catalog policy (+ matrix/deferred docs)
> **Depends on:** Phase 80 plan published (may run in parallel for docs-only rows)
> **Unblocks:** cleaner Unsupported notes for recount/closure
> **Honesty:** `../HONESTY-GATES.md`
> **Inventory:** `../unsupported-triage.json` tier `4_niche_or_draft` (**94** names)
> **Subagent scan (2026-08-29):** skip/niche pattern from phases 72-77

---

## Overview

Tier 4 is **out of scope** for the near-term print push: draft corner-shape, MathML/ruby/rhythmic niche, and draft gap/row-rule decorations. All **94** names stay `engine_status: unsupported` with honest notes. No apply arms. No consumers. No Implemented flips in this phase.

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


## Ownership buckets (94)

| Bucket | Count | Policy |
|--------|------:|--------|
| `D_draft_corner_shape` | 34 | Draft corner-shape CSS; not print PDF |
| `D_math_ruby_niche` | 33 | Ruby / MathML / rhythmic / misc niche |
| `D_draft_gap_decorations` | 27 | Draft `row-rule*` / `rule*` gap decorations |
| **Total** | **94** | |

## Work order

1. Lock names from triage (do not invent).
2. Replace honesty-revert stub notes in `catalog/mapping.json` with bucket-specific prose.
3. Keep `engine_status: unsupported` and `code_path: ""`.
4. Align `documentation/compatibility-matrix.md` / `documentation/deferred.md` language (no Implemented claims).
5. Run `--check`; recount only if counts changed (note-only should stay 202/0/616/0 unless other phases moved).

## Checklist

- [ ] 81.1.1 Ownership locked to 94 names in the three `D_*` buckets below.
- [ ] 81.2.1 Policy: niche/draft deferred for print engine (`[~]` ok with next gate = later version or permanent non-goal).
- [ ] 81.2.2 Every owned `mapping.json` row: `unsupported`, empty `code_path`, bucket-specific notes (not the mass-revert stub alone).
- [ ] 81.2.3 Scripted verify: all 94 still unsupported.
- [ ] 81.2.4 Matrix/deferred prose names the three families as unsupported.
- [ ] 81.2.5 Do **not** add fake apply arms in `internal/layout` for these names.

### Catalog and gate close

- [ ] CATALOG.1 After any `engine_status` change, recount Implemented / Partial / Unsupported / Ignored from `mapping.json` with a `Counter` on `engine_status`.
- [ ] CATALOG.2 Write the same counts into `catalog/coverage-summary.json` `counts.properties_by_engine_status` and into `property-counts.md`.
- [ ] CATALOG.3 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] CATALOG.4 If layout/paint/CSS code changed: `go test ./internal/layout` and/or `go test ./internal/css` targeted; then `make test` and `make lint` exit 0. If paint/pagination changed: `make golden` exit 0.
- [ ] CATALOG.5 If matrix/docs claims changed: `make claim-scan` exit 0.
- [ ] CATALOG.6 No git commands were run unless the user explicitly asked.


## Forbidden proofs

- Any flip of these 94 to `implemented` or `partial`
- Treating `corner-*` as `border-radius`, or `row-rule*` as `column-rule`
- Closing because `--check` is green alone
- Git commands without explicit user request

## Note templates

- corner-shape: `Draft corner-shape CSS; not in print PDF engine. Left unsupported.`
- math/ruby/niche: `Ruby/MathML/rhythmic niche; not implemented for print PDF. Left unsupported.`
- gap decorations: `Draft gap/row-rule decorations; no print consumer. Left unsupported.`

## Ownership (94)

### `D_draft_corner_shape` (34)

```
corner, corner-block-end, corner-block-end-shape, corner-block-start
corner-block-start-shape, corner-bottom, corner-bottom-left, corner-bottom-left-shape
corner-bottom-right, corner-bottom-right-shape, corner-bottom-shape, corner-end-end
corner-end-end-shape, corner-end-start, corner-end-start-shape, corner-inline-end
corner-inline-end-shape, corner-inline-start, corner-inline-start-shape, corner-left
corner-left-shape, corner-right, corner-right-shape, corner-shape, corner-start-end
corner-start-end-shape, corner-start-start, corner-start-start-shape, corner-top
corner-top-left, corner-top-left-shape, corner-top-right, corner-top-right-shape
corner-top-shape
```

### `D_math_ruby_niche` (33)

```
block-ellipsis, block-step, block-step-align, block-step-insert, block-step-round
block-step-size, box-snap, column-height, column-wrap, continue, copy-into
flex-line-count, frame-sizing, interpolate-size, line-fit-edge, line-grid
line-height-step, line-padding, line-snap, link-parameters, math-depth, math-shift
math-style, min-intrinsic-sizing, overlay, reading-flow, reading-order, ruby-align
ruby-merge, ruby-overhang, ruby-position, text-size-adjust, zoom
```

### `D_draft_gap_decorations` (27)

```
row-rule, row-rule-break, row-rule-color, row-rule-inset, row-rule-inset-cap
row-rule-inset-cap-end, row-rule-inset-cap-start, row-rule-inset-end
row-rule-inset-junction, row-rule-inset-junction-end, row-rule-inset-junction-start
row-rule-inset-start, row-rule-style, row-rule-visibility-items, row-rule-width, rule
rule-break, rule-color, rule-inset, rule-inset-cap, rule-inset-end, rule-inset-junction
rule-inset-start, rule-overlap, rule-style, rule-visibility-items, rule-width
```


