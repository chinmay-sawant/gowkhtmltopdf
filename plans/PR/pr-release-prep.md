## Summary

Prepares the project for release by closing the 2026-08-12/13 architecture and application-review ledgers: conversion/API seams, network and image policy, layout/print-fidelity fixes across multi-page chrome, pagination, flex/grid, and wiki-like content, a full user-docs rewrite grounded in live code, lint cleanup across the layout engine, regenerated sample PDFs, and white-background showcase page screenshots with a reusable raster script.

---

## Motivation / context

- Plans / ledgers: `plans/reviews/improve-codebase/` (2026-08-12 application review; 2026-08-13 architecture/extension/practices checklists)
- Skills pack: `skills/improve-codebase/`, `skills/debug-html-template/`
- Branch `chore/release-prep` stacks release-prep artifacts, critical contract fixes, architecture hardening, layout fidelity, docs rewrite, samples, and showcase thumbs so `master` can absorb one coherent release-prep merge
- Issues: see **Related issues** (no single ticket owns the whole branch; PDF version/compliance work remains open)

---

## Changes

### Critical contracts and conversion architecture

- Close critical renderer contract gaps (outline/PDF stdout multiplexing rejected before sinks open; TOC-XSL terminal parsing; typed request validation; ImageSettings background aliases and Get round-trips)
- Writer-first `ConvertTo` / cloned settings; page islands restricted to explicit benchmark requests with parent-consistent clones
- Harden conversion architecture and API boundaries: global PDF option validation, canonical page-size semantics, CLI adapters separated from convert/imageout, centralized error sentinels in `internal/errs`
- PDF image resource dedup; border/supersample/font/outline/registry optimizations; focused benchmarks (repeated resources, overlays, pools, headers/footers, registries)
- Image policy and request architecture hardening; Restricted dial pinning plus CLI flags
- Health-review seams: golden semantic needles, fixture-54 local assets, CI/version/docs hygiene
- Decouple imageout from convert (`imageout.Request`); dynamic prepare viewport for `@media`; media resolution and font registry consolidation; mode-neutral `render.Plan`; CLI validation / dump-outline centralization; preflight errors via OnError

### Layout and print fidelity

- Pagination: land `page-break-before:always` at next page top; experimental keep-together for overflowing aside callouts; multi-page section chrome closed without page-edge hairlines
- Borders: stop stretching dashed segments into solid stubs; stop stretching `border-left` past blockquote content; expand border shorthand so accent tops win
- Tables / wiki: keep repeated thead clones off forced-break suffix shifts; tighten wiki thumbs, link rules, and table continuations
- Flex / grid: honor grid gap paint, subgrid columns, and masonry packing; keep flex row-gap after paint chrome stretch; measure inline-block width; honor length `vertical-align`
- CSS / imageout parity wave: paint policy prologue (transforms, opacity, text-transform, RotateDeg); writing-mode and text-indent; CSS `@page { size }`; HF FontName via pdf.Registry; UA defaults for meter/progress
- Fixture authoring: template CSS for sticky gutters (fixture-31); architecture-diagram sample path wired through `make samples`

### Lint and maintainability

- Resolve golangci-lint violations across layout engine and tests (cyclop/gocognit/funlen, mnd/goconst, varnamelen, wsl, lll, testpackage, etc.)
- Prior lint waves across pdf, layout, load, convert, tests, and benchmarks

### Documentation and skills

- Rewrite user docs from multi-agent source scan; slim root README to landing + quick start + doc index
- Refresh overview, getting-started, CLI, library-api, architecture, fonts, samples, deferred, fidelity, compatibility-matrix, THREAT-MODEL, performance
- Add improve-codebase skill pack and 2026-08-13 phase-wise checklist ledger
- Remove dedicated third-party license reference surface where obsolete; prefer template CSS guidance in debug-html-template skill

### Samples and showcase site assets

- Regenerate architecture diagram and sample outputs (`make samples` wiring for golden API generator)
- Commit refreshed `output/` sample PDFs
- Add `scripts/screenshot_showcase.py`: PyMuPDF raster of every `output/*.pdf` page to PNG with **opaque white RGB background** at 96 dpi (`{name}.png`, `{name}-N.png`)
- Replace all `frontend/src/assets/showcase/*.png` thumbs; set `wiki-ana-de-armas` showcase page count to 12

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | PDF image dedup and registry/border/pool optimizations; benchmark artifacts regenerated |
| **Memory** | Lower repeated-image PDF size paths; no intentional peak-memory regressions |
| **Behavior / correctness** | Layout pagination, chrome stretch, thead, flex/grid, wiki thumbs, and API validation contracts fixed |
| **API / CLI** | Stronger preflight/validation; Restricted network policy flags; image mode CLI allow/exclude-from-outline wiring; ConvertTo/cloned settings semantics |
| **Dependencies** | Host tool only for screenshots: optional `pymupdf` for `scripts/screenshot_showcase.py` (not a Go module dep) |
| **Binary size / build time** | No material converter binary growth; repo grows by regenerated samples/showcase PNGs |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Multiplexed outline XML + PDF on the same stdout sink | Rejected early; write outline and PDF to distinct sinks |
| Invalid global PDF options / empty convert configs | Surface as validation/preflight errors (OnError when set) |
| Restricted network dial pinning | Dial behavior follows policy/flags; operators using unrestricted fetches must set the matching CLI/API policy |
| None for ordinary local golden → PDF converts | Re-run `make samples` / golden suite after pull |

---

## Test plan

- [ ] `make lint` / golangci-lint suite used by CI
- [ ] `make test` (or `go test ./... -count=1`)
- [ ] `CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf`
- [ ] `CGO_ENABLED=0 go build -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage`
- [ ] `go test ./internal/convert/ -run 'TestGoldenCorpus' -count=1`
- [ ] Spot-check layout regressions: fixtures 18, 21, 31, 40, 48, 56; wiki smoke PDF if network available
- [ ] `python3 scripts/screenshot_showcase.py` regenerates white-background showcase PNGs
- [ ] Optional: `cd frontend && npm run build` to refresh `docs/` hashed assets for GitHub Pages

### Commands

```sh
make lint
make test
CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
python3 scripts/screenshot_showcase.py
```

---

## Screenshots / sample output

```
python3 scripts/screenshot_showcase.py
# architecture-diagram.pdf … wiki-ana-de-armas.pdf
# wrote 167 PNG(s) to frontend/src/assets/showcase
# all RGB, 794×1123 (A4 @ 96 dpi), white page canvas
```

Showcase thumbs under `frontend/src/assets/showcase/` now match current `output/*.pdf` pages (including `wiki-ana-de-armas-12.png`).

---

## Related issues

- Refs #29 (epic: newer PDF versions and compliance — still open; this PR does not implement 1.7/2.0/UA)
- Refs #31, #32, #33 (PDF version / UA tickets remain follow-up)
- Refs #35 (Python bindings — orthogonal, not delivered here)

No `Closes` keywords: this is an integration release-prep branch rather than a single-issue fix.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-release-prep.md`

---

## Follow-ups (out of scope)

- Rebuild and commit `docs/` Vite bundle after showcase PNG refresh (`cd frontend && npm run build`)
- PDF 1.7 / 2.0 / UA-2 / A-4 compliance work (#29–#33)
- Python c-shared / PyPI bindings (#35)
- Further tightening of experimental aside keep-together heuristics

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated secrets or credentials in the tree
- [ ] Public API / CLI changes documented in `documentation/`
- [ ] Golden / sample artifacts match the fixed engine
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates/Refs keywords
- [ ] Showcase PNG naming still matches `ShowcasePage` / `Cardbox` (`name.png`, `name-N.png`)
