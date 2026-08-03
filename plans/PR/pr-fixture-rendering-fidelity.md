## Summary

Fixes display-list paint order and multi-image XObject naming so backgrounds no longer cover text and multiple images render correctly. Also tightens colspan column sizing and regenerates golden samples.

---

## Motivation / context

- User report after MVP-2: fixtures 05/08/13/14/16/19 looked like “font color matches background” and multi-image pages only showed the last image.
- Root cause 1: block backgrounds/borders were appended *after* text ops, so later painting covered content.
- Root cause 2: every image used resource name `I0`, so later images replaced earlier ones in the PDF.

---

## Changes

### Paint order (`internal/layout/layout.go`)

- After laying out a block, **prepend** background + border ops ahead of child/content ops (`prependChrome`).
- Fixes yellow notice (fixture-05), light-blue keep boxes (fixture-08), pre grey band (fixture-13), colorful fills (fixture-14/19).

### Unique image resources (`internal/layout/paint.go`, `internal/convert/hf.go`)

- Per-page names `I0`, `I1`, … (HF: `HFI0`, …).
- Fixes logo + data URI (fixture-07) and four-swatch grid (fixture-20).

### Tables

- Colspan cells contribute width across spanned columns.
- fixture-10 header `colspan` corrected 4 → 5 to match five columns.

### Tests / samples

- `TestBackgroundPaintsUnderText`, `TestMultiImageUniqueOps`
- `make samples` regenerated under `output/`

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Text readable on colored backgrounds; multi-image pages correct |
| **Performance** | Negligible (slice insert for chrome ops) |
| **API / CLI** | None |
| **Dependencies** | None |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Visual output improves for affected fixtures |

---

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make samples`
- [x] Content-stream checks: fill index before “Important”; `/I0`–`/I3` Do ops on fixture-20; pre lines preserved on fixture-13

### Commands

```sh
make test
make lint
make samples
```

---

## Related issues

- Relates to #2 (post-MVP rendering quality epic)
- Relates to #3 / #4 / #5 / #6 (follow-on fidelity after those ships)

---

## PR metadata checklist (author)

- [x] Self-assigned
- [x] Labels applied
- [x] Related issues filled
- [x] Body under `plans/PR/pr-fixture-rendering-fidelity.md`

---

## Follow-ups (out of scope)

- Full border-collapse / complex rowspan
- Fixture-16 multi-page HF polish beyond current pagination
