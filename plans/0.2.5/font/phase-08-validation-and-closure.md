# Font Phase 8 — Performance, Full Gates, and Closure

> **Status:** planned
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phases 1–7
> **Unblocks:** explicit font-track closure and any later release amendment

## Overview

Close the track only after the resolver, input paths, writer, compliance
profiles, fixtures, documentation, and performance evidence agree.

## 8.1 Performance and determinism

- [ ] Benchmark registry name loading and family/style lookup with and without
  caching.
- [ ] Benchmark the resolver on the external benchmark template and a
  multi-page document; distinguish font work from layout and PDF writing.
- [ ] Confirm font parsing happens once per loaded face and does not repeat per
  page or per glyph.
- [ ] Confirm subset caching remains document-wide and fingerprint-safe.
- [ ] Confirm default output is deterministic across repeated runs.
- [ ] Confirm opt-in font discovery does not occur when no font option is set.

## 8.2 Required validation commands

- [ ] `GOCACHE=/tmp/gowkhtmltopdf-go-cache go test ./internal/pdf ./internal/layout ./internal/convert ./internal/cli .`
- [ ] `GOCACHE=/tmp/gowkhtmltopdf-go-cache go test ./...`
- [ ] `GOCACHE=/tmp/gowkhtmltopdf-go-cache go test -race ./...`
- [ ] `make lint` with any findings repaired rather than suppressed.
- [ ] `make build` and the focused CLI/library font smokes.
- [ ] Compliance wrapper for PDF/A-3a + PDF/UA-1 and PDF/A-4 + PDF/UA-2
  outputs when veraPDF is available.
- [ ] Corpus render and raster inspection commands from Phase 6.
- [ ] `make bench-engine` and `make bench-lib` if resolver/layout code changes
  are on the hot path; record font-related interpretation separately from
  historical benchmark data.

## 8.3 Ledger closure

- [ ] Record changed files, exact test commands, tool versions, and output
  artifact paths in each phase file.
- [ ] Mark only rows with matching implementation evidence as complete.
- [ ] Update this folder's status and `plans/0.2.5/README.md` only after all
  required gates pass. Update `VERSION` / `CHANGELOG.md` only as part of an
  actual 0.2.5 release prep, not merely because the plan exists.
- [ ] Keep the predecessor 0.2.4 phases 31–39 status honest; do not mark their
  old closure evidence as proof for this track.
- [ ] Confirm knowledge-base font/roadmap pages cite `plans/0.2.5/font/` and
  match `documentation/fonts.md`.
- [ ] Record deferred work explicitly, such as proprietary-font licensing,
  browser pixel parity, WOFF2/Brotli (pending `09-remote-webfonts.md`),
  variable-font support, or synthetic bold/italic if Phase 1 rejected it.

## Definition of done

- [ ] All supported input paths select the intended family deterministically.
- [ ] All selected faces pass writer and conformance validation.
- [ ] Fixture-55 and the broader font corpus have accepted visual evidence.
- [ ] Documentation and knowledge-base accurately state exact-font versus
  fallback behavior.
- [ ] No unchecked required work remains in this track at closure.
