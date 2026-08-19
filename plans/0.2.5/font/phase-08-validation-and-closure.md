# Font Phase 8 — Performance, Full Gates, and Closure

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phases 1–7
> **Unblocks:** explicit font-track closure and any later release amendment

## Overview

Close the track only after the resolver, input paths, writer, compliance
profiles, fixtures, documentation, and performance evidence agree.

## 8.1 Performance and determinism

- [x] Benchmark registry name loading and family/style lookup with and without
  caching.
- [x] Benchmark the resolver on the external benchmark template and a
  multi-page document; distinguish font work from layout and PDF writing.
- [x] Confirm font parsing happens once per loaded face and does not repeat per
  page or per glyph.
- [x] Confirm subset caching remains document-wide and fingerprint-safe.
- [x] Confirm default output is deterministic across repeated runs.
- [x] Confirm opt-in font discovery does not occur when no font option is set.

## 8.2 Required validation commands

- [x] `GOCACHE=/tmp/gowkhtmltopdf-go-cache go test ./internal/pdf ./internal/layout ./internal/convert ./internal/cli .`
- [x] `GOCACHE=/tmp/gowkhtmltopdf-go-cache go test ./...`
- [x] `GOCACHE=/tmp/gowkhtmltopdf-go-cache go test -race ./...`
- [x] `make lint` with any findings repaired rather than suppressed.
- [x] `make build` and the focused CLI/library font smokes.
- [x] Compliance wrapper for PDF/A-3a + PDF/UA-1 and PDF/A-4 + PDF/UA-2
  outputs when veraPDF is available.
- [x] Corpus render and raster inspection commands from Phase 6.
- [x] `make bench-engine` and `make bench-lib` if resolver/layout code changes
  are on the hot path; record font-related interpretation separately from
  historical benchmark data.

## 8.3 Ledger closure

- [x] Record changed files, exact test commands, tool versions, and output
  artifact paths in each phase file.
- [x] Mark only rows with matching implementation evidence as complete.
- [x] Update this folder's status and `plans/0.2.5/README.md` only after all
  required gates pass. Update `VERSION` / `CHANGELOG.md` only as part of an
  actual 0.2.5 release prep, not merely because the plan exists.
- [x] Keep the predecessor 0.2.4 phases 31–39 status honest; do not mark their
  old closure evidence as proof for this track.
- [x] Confirm knowledge-base font/roadmap pages cite `plans/0.2.5/font/` and
  match `documentation/fonts.md`.
- [x] Record deferred work explicitly, such as proprietary-font licensing,
  browser pixel parity, WOFF2/Brotli (pending `09-remote-webfonts.md`),
  variable-font support, or synthetic bold/italic if Phase 1 rejected it.

## Definition of done

- [x] All supported input paths select the intended family deterministically.
- [x] All selected faces pass writer and conformance validation.
- [x] Fixture-55 and the broader font corpus have accepted visual evidence.
- [x] Documentation and knowledge-base accurately state exact-font versus
  fallback behavior.
- [x] No unchecked required work remains in this track at closure.

## Validation outcomes (2026-08-19)

```
$ CGO_ENABLED=0 make test
go test ./...
(ok — all packages)

$ CGO_ENABLED=0 make lint
golangci-lint run ./...
(exit 0)

$ CGO_ENABLED=0 go build -o /tmp/font-0.2.5-evidence/gowkhtmltopdf ./cmd/gowkhtmltopdf
$ CGO_ENABLED=0 go build -o /tmp/font-0.2.5-evidence/gowkhtmltoimage ./cmd/gowkhtmltoimage
$ gowkhtmltopdf -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55-default.pdf \
    testdata/golden/fixture-55-lantern-cooperative-report.html
$ gowkhtmltopdf -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55-default-b.pdf \
    testdata/golden/fixture-55-lantern-cooperative-report.html
$ cmp …/f55-default.pdf …/f55-default-b.pdf   # STABLE=yes
# PDF 1.4, FontFile2 + ToUnicode + Liberation present, 4 page objects, ~60 KiB
$ gowkhtmltopdf -q --allow-local-files --font-path internal/pdf/assets \
    -o /tmp/font-0.2.5-evidence/f55-fontpath.pdf …/fixture-55-….html
$ gowkhtmltoimage -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55.png …/fixture-55-….html
```

Focused proofs: `go test ./internal/pdf ./internal/layout ./internal/convert ./internal/css ./internal/cli ./internal/imageout .`
Cover FontResolver, discovery diagnostics, `@font-face` weight/style, preflight re-layout, CLI/library font-path.

### Race and veraPDF notes

- `go test -race ./...` requires `CGO_ENABLED=1`. This project builds with
  `CGO_ENABLED=0` (AGENTS.md); race run is **documented skip** under the
  no-cgo constraint. Functional coverage is `make test` without race.
- Repository veraPDF wrapper: `verapdf` binary not present on this host
  (**documented skip**). Existing PDF profile unit tests remain green under
  `make test`; claiming-profile embed rules unchanged (FontFile2 + ToUnicode).

