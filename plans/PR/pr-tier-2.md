## Summary

Delivers Tier 2 phases **18 → 20 → 17 → 19** plus deferred layout/font follow-ups, then hardens visual fidelity for fixtures 25–28 and the TOC/HF showcase (CJK Type0, flex/grid backgrounds, absolute paint order, text HF baselines, grid gap sizing).

---

## Motivation / context

- Plans: `plans/10-canonical-post-mvp-roadmap.md`, `plans/phases/phase-17-broader-css.md`, `phase-18-pagination-polish.md`, `phase-19-fonts-i18n.md`, `phase-20-hf-links-edges.md`
- Issues: see **Related issues**
- Product constraint: pure Go, **stdlib-only** (no third-party modules, no browser embed)

---

## Changes

### Phase 18 — Pagination polish

- `<thead>` repeat on continuation pages; orphan/widow + heading heuristics
- Zoom / smart-shrink wiring
- New golden `fixture-23-thead-repeat`

### Phase 20 — Links / HF edges

- Inline `#anchor` → GoTo; `--resolve-relative-links` / `--keep-relative-links`
- `[topage]` / outline TOC offset polish; HTML HF link annotations on body pages
- New golden `fixture-24-internal-anchors`

### Phase 17 — Broader CSS (flex / position / grid)

- `position: relative|absolute|fixed|sticky` lite; float packing
- Partial flex (row/column, justify, gap, flex-grow, flex-wrap)
- CSS grid lite (`grid-template-columns` / `repeat` / gap)
- New goldens `fixture-25` … `fixture-26`, `fixture-28`

### Phase 19 — Fonts / CJK

- `--font-path` / `--use-system-fonts`, CSS `font-family` registry
- Type0 / CID Identity-H Unicode path; local `@font-face` TTF
- Mixed Latin + CJK: Liberation fallback without breaking Type0 sibling
- Composite glyph subset `MORE_COMPONENTS` fix; docs in `documentation/fonts.md`
- New golden `fixture-27-cjk-fontpath` (Han/kana; Hangul needs a Hangul-capable face)

### Visual QA / correctness follow-ups

- Parse CSS `background` shorthand for fills
- Flex measure → grow so auto-width children and `flex-grow` stay on-page
- Defer absolute/fixed paint after in-flow; keep containing-block origin at content top
- Text HF ascent/descent baseline signs (showcase footer no longer clipped)
- Grid `1fr` tracks subtract `(n−1)×gap` so cells fit inside the border
- `make samples` passes `--font-path` when Droid is installed

### Samples

- Regenerated `output/fixture-23` … `28` + `output/showcase-toc-hf-outline.pdf`

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Modest: Type0 subset + font-path scan when opted in |
| **Memory** | Higher for CJK subset embeds (expected) |
| **Behavior / correctness** | Richer CSS layout; Unicode CJK when a face is available; HF text on-page |
| **API / CLI** | `--font-path`, `--use-system-fonts`, link-resolve flags; HF/outline unchanged CLI shape |
| **Dependencies** | None (stdlib-only) |
| **Binary size / build time** | Unchanged core; sample PDFs grow when CJK embeds |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None for Latin-only default path | - |
| CJK without `--font-path` / system fonts | Still tofu/`?`; pass a Unicode TTF directory |

---

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet`
- [x] `make samples` (fixtures 23–28 + showcase under `output/`)
- [x] Spot-check flex fills / absolute overlay: `output/fixture-25`, `26`
- [x] Spot-check CJK with Droid: `output/fixture-27-cjk-fontpath.pdf`
- [x] Spot-check grid gap + FIXED badge: `output/fixture-28-flex-wrap-grid-fixed.pdf`
- [x] Spot-check HF not clipped: `output/showcase-toc-hf-outline.pdf`

### Commands

```sh
make lint
make test
make samples
```

---

## Screenshots / sample output

```
output/fixture-23-thead-repeat.pdf
output/fixture-24-internal-anchors.pdf
output/fixture-25-flex-row.pdf
output/fixture-26-position-lite.pdf
output/fixture-27-cjk-fontpath.pdf   # with --font-path when available
output/fixture-28-flex-wrap-grid-fixed.pdf
output/showcase-toc-hf-outline.pdf   # toc + header/footer + outline on fixture-16
```

---

## Related issues

- Relates to Tier 2 phases 17–20 in `plans/10-canonical-post-mvp-roadmap.md`
- Relates to #15 (Tier 1 landed on `master`)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled
- [x] Filled body under `plans/PR/pr-tier-2.md`

---

## Follow-ups (out of scope)

- Full CSS Grid (span, areas, dense pack)
- HarfBuzz / true complex-script shaping (Arabic joining, Indic)
- Hangul-capable default face bundling
- Vertical writing modes / ruby

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed
