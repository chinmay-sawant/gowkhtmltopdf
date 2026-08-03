## Summary

Ships post-MVP rendering quality work: multi-font Liberation faces, TTF image-mode raster, richer CSS selectors, text-run coalescing, and layout correctness fixes for table cell backgrounds, `<pre>` whitespace, and `margin: auto` centering. Closes epic #2 and child issues #3–#6.

---

## Motivation / context

- Plans: `plans/10-canonical-post-mvp-roadmap.md`, phases 10–23 checklists
- Issues: see **Related issues**
- Product constraint: pure Go, **stdlib-only** (no third-party modules, no browser embed)

---

## Changes

### Typography and fonts (#6, #4)

- Embed Liberation Sans **Bold / Italic / BoldItalic** beside Regular
- `pdf.FaceSet` resolves CSS `font-weight` / `font-style` to real faces
- PDF multi-face subset embed; fake bold only when a bold face is missing
- Coalesce same-style text runs to stabilize word spacing

### Image mode (#3)

- Pure-Go TrueType outline raster with coverage AA (`internal/imageout/ttfraster.go`)
- Same face metrics as PDF/layout (no 5×7 advance mismatch)

### CSS selectors and wiki-class progress (#5)

- Attribute selectors `[attr]` / `[attr=value]`
- `:first-child`, `:last-child`, `:nth-child(odd|even|an+b)`
- Sibling combinators `+` / `~` (correct combinator walk)
- Vendored `testdata/web/wiki-like-article.html` + quality tests

### Layout correctness (fixture reports)

- **Table cell backgrounds:** set `cell.h = rowH` so fills/borders paint (fixes fixture-14/16 header and zebra backgrounds; white text on dark headers visible again)
- **`<pre>`:** UA `white-space: pre` (was missing; only `textarea` had it) so code blocks keep newlines/indent (fixture-13)
- **`margin: auto`:** horizontal centering for definite-width blocks (fixture-17 cover rule)

### Docs / samples / plans

- Compatibility matrix + README deferred table updated
- Sample PDFs/PNG regenerated; wiki smoke PDF preserved
- Post-MVP phase-wise roadmap under `plans/`

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Image-mode glyph raster is heavier than 5×7; glyph cache mitigates repeats |
| **Memory** | Four embedded TTFs (~0.5 MB assets); glyph cache bounded per face/size |
| **Behavior / correctness** | Bold/italic, spacing, table BGs, pre, centered rules improved |
| **API / CLI** | No public API break; library consumers get multi-font automatically |
| **Dependencies** | Still zero module deps |
| **Binary size / build time** | Binary larger by three TTF faces |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Drop-in for report HTML; visual output may change (real bold, BGs, pre lines) |

---

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet`
- [x] `CGO_ENABLED=0 go build ./cmd/gowkhtmltopdf ./cmd/gowkhtmltoimage`
- [x] `make samples` regenerates `output/` fixtures
- [x] Targeted layout tests: table cell BG height, pre newlines, margin auto center
- [x] Convert quality tests: wiki-like fixture, bold face embed

### Commands

```sh
make test
make lint
make samples
CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
CGO_ENABLED=0 go build -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage
```

---

## Screenshots / sample output

```
make samples
# fixture-01 PNG uses TTF AA text (larger file vs old 5x7)
# fixture-13 pre lines preserved with indentation
# fixture-14/16 table header backgrounds have non-zero height
# fixture-17 cover rule horizontally centered
```

---

## Related issues

- Closes #2 (epic: post-MVP rendering quality)
- Closes #3 (image-mode PNG raster quality)
- Closes #4 (residual font/word spacing)
- Closes #5 (Wikipedia-class / common-site CSS)
- Closes #6 (multi-font support)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `bug`, `documentation`)
- [x] Related issues filled with real ticket IDs
- [x] Filled body under `plans/PR/pr-mvp-2-rendering-quality.md`

---

## Follow-ups (out of scope)

- Floats / position / partial flex (phases 16–17)
- System/folder font discovery and CJK Type0 (phase 19)
- Full JavaScript (phase 22 staged / Tier 3 deferred)
- Table header repeat across pages (phase 18)

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets committed; OFL font assets are intentional
