## Context

MVP v0.1.0 ships a pure-Go, stdlib-only HTML→PDF/image pipeline (phases 0–9). Sample artifacts under `output/` (via `make samples`) and a Wikipedia URL smoke test show the engine is **usable for Latin report HTML** but still has clear **rendering quality and web-CSS gaps**.

This epic tracks post-MVP work to raise visual quality (image mode + fonts) and broaden CSS/font support toward common public pages, without abandoning the stdlib-only / no-browser constraint unless the plan is explicitly amended.

Parent plans: `plans/00-canonical-pure-go-rewrite.md`, deferred list in `README.md` / `documentation/overview.md`.  
Related PR: https://github.com/chinmay-sawant/gowkhtmltopdf/pull/1

## Child issues

- [ ] #3 - Image-mode PNG raster quality (fixture-01 analysis)
- [ ] #4 - Residual font / word spacing (PDF + layout)
- [ ] #5 - Wikipedia-class / common-site CSS (Ana de Armas)
- [ ] #6 - Multi-font support (beyond Liberation Sans Regular)

## Scope (in)

1. Image-mode raster quality (bitmap font, spacing, anti-aliasing path).
2. PDF text spacing / typography polish beyond the 1000-unit `/Widths` fix.
3. Broader CSS support for Wikipedia-class and common-site layouts.
4. Multi-font support (bold/italic/additional faces; Unicode longer-term).

## Out of scope

- Full WebKit/Chrome parity or embedding a browser
- JavaScript execution
- Shipping third-party commercial HTML→PDF APIs
- PDF encryption / PDF/A (unless separately prioritized)

## Success criteria

- [ ] All child issues closed or explicitly deferred with rationale
- [ ] `make samples` artifacts show measurable visual improvement on agreed fixtures
- [ ] Compatibility matrix and docs updated for any new CSS/font capabilities
- [ ] Stdlib-only / no-cgo constraint still holds (or plan amendment recorded)

## Plan

- Ledgers: `plans/phases/phase-04-html-css-layout.md`, `phase-07-image-converter.md`, intermediate roadmap in `phase-09-hardening-closure.md`
- Issue bodies archived under `plans/PR/issues/`

## References

- Samples: `output/fixture-01-simple-invoice.png`, `output/fixture-01-simple-invoice.pdf`, `output/wiki-ana-de-armas.pdf`
- Docs: `documentation/samples.md`, `documentation/compatibility-matrix.md`
- URL smoke: https://en.wikipedia.org/wiki/Ana_de_Armas
