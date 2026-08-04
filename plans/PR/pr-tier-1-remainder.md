## Summary

Closes Tier 1 remainder: float lite / inline-block / box-sizing for invoice CSS, PDF image docs + `web.images=false`, and library install/`ConvertHTML` polish. Regenerates committed `output/` samples including new fixture-22.

---

## Motivation / context

- Plans: `plans/10-canonical-post-mvp-roadmap.md`, phases 11 / 14 / 16
- Issues: see **Related issues**
- Product constraint: pure Go, **stdlib-only** (no third-party modules, no browser embed)

---

## Changes

### Phase 16 — Invoice CSS (float lite)

- `float: left|right` + `clear` with simple side exclusion (`internal/layout/float.go`)
- Real `display: inline-block` (atomic inline box)
- `box-sizing: content-box` (default) and `border-box`
- Simple `text-align: justify`; table-cell `vertical-align` top/middle/bottom
- New golden `fixture-22-float-invoice-chrome.html` + layout unit tests

### Phase 14 — PDF images polish

- Document JPEG DCT pass-through, PNG alpha soft-mask, ignored DPI/quality knobs
- Honor `Global.Web.Images=false` (no image XObjects); image mode gated similarly
- Matrix / fidelity updates for the solid logo/grid path

### Phase 11 — Library embedder DX

- Install / `replace` story in `documentation/library-api.md`
- `ConvertHTML(ctx, html, global)` one-shot helper; `GuessURL` accepts `inline:` prefix
- Close phase 11/14/16 checklists; mark Tier 1 closed in plan index

### Samples

- `make samples` regenerated fixture PDFs/PNGs + showcase; adds `output/fixture-22-float-invoice-chrome.pdf`

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Negligible; float lite is O(children) per block |
| **Memory** | Unchanged |
| **Behavior / correctness** | Invoice chrome can use floats; `box-sizing` default is CSS `content-box` (was effectively border-box for explicit widths); `web.images=false` skips embeds |
| **API / CLI** | New exported `ConvertHTML`; global `web.images` consumed |
| **Dependencies** | None (stdlib-only) |
| **Binary size / build time** | Unchanged |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Explicit `width` + padding without `box-sizing` | Boxes grow (content-box). Add `box-sizing: border-box` for prior visual size |
| None otherwise | - |

---

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet`
- [x] `make samples` (regenerated `output/`)
- [x] Golden corpus includes fixture-22 page envelope
- [ ] Visual smoke: open `output/fixture-22-float-invoice-chrome.pdf` (logo left / meta right / clear)

### Commands

```sh
make lint
make test
make samples
```

---

## Screenshots / sample output

```
output/fixture-22-float-invoice-chrome.pdf   # new float chrome invoice
output/fixture-*.pdf                         # regenerated corpus
output/showcase-toc-hf-outline.pdf
```

---

## Related issues

- Relates to post-MVP Tier 1 closure (`plans/10-canonical-post-mvp-roadmap.md` phases 11, 14, 16)
- Relates to #2 (rendering quality epic) where still open / tracking

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled
- [x] Filled body under `plans/PR/pr-tier-1-remainder.md`

---

## Follow-ups (out of scope)

- Phase 17 partial flex / position
- Phase 18 pagination polish (thead repeat)
- Full CSS2 float edge cases

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed
