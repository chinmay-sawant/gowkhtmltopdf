# Font Phase 2 — Central Resolver Seam

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phase 1
> **Unblocks:** Phases 3–6

## Overview

Centralize family and glyph selection behind one internal `FontResolver`
module. The module should provide depth: layout knows how to ask for a face,
while registry precedence, aliases, style selection, glyph fallback, and face
identity stay local to the resolver.

## 2.1 Interface and ownership

- [x] Place the resolver under `internal/pdf` beside `FaceSet` / `Registry`
  unless Phase 2 proves a layering cycle; do not invent `internal/font` by
  default.
- [x] Define the smallest internal interface needed by layout **and HF** for:
  - family/style resolution;
  - missing-glyph resolution for a selected family stack;
  - last-resort registered glyph coverage.
- [x] Keep `FaceSet`, `Registry`, and PDF `Font` implementation details behind
  the resolver seam where practical.
- [x] Route `lookupFaceFor` / `faceForRune` / `facesWithGlyph` and
  `resolveHFFont` through the same seam so PDF, image, and HF stop diverging.
- [x] Keep the public `Document`, CLI, and library font options unchanged.
- [x] Make resolver construction explicit from bundled faces plus an optional
  registry; do not construct or scan host fonts inside layout.
- [x] Decide whether diagnostics are returned as a separate result or emitted
  through the existing conversion log without making diagnostics part of the
  hot layout interface.
- [x] Leave embed-preflight re-layout orchestration to `internal/convert`
  (Phase 5); the resolver only marks candidates unavailable.

## 2.2 Resolution implementation

- [x] Implement exact registered-family lookup in CSS order.
- [x] Implement generic-family mapping after named-family misses.
- [x] Implement the Phase 1 weight/style selection rule.
- [x] Implement per-rune fallback through the same family policy used for the
  primary face; remove duplicated policy from `facesWithGlyph` and registry
  lookup paths where safe.
- [x] Give the resolver a way to mark a candidate unavailable with a
  diagnostic and continue to the next CSS/bundled face.
- [x] Keep fallback selection before layout is committed; do not silently
  replace a face after its metrics have been used.
- [x] Preserve deterministic tie-breaking for duplicate files and duplicate
  family names.
- [x] Include the font fingerprint, selected face, family stack, weight/style,
  and rune where required in cache identity.
- [x] Ensure a selected face with an unsupported glyph cannot cause the
  resolver to silently switch the whole Latin run to an unrelated face.

## 2.3 Tests

- [x] Update `internal/pdf/registry_lookup_test.go` for the final named-family
  and generic-family contract.
- [x] Replace or narrow `TestFaceResolveFamilyAliases` so it asserts the chosen
  policy rather than an accidental direct alias.
- [x] Add resolver tests for exact custom family, generic fallback, author-stack
  continuation, style selection, duplicate-family tie-breaking, parse-failure
  fallback, and rune fallback.
- [x] Add layout tests proving the same resolver result is used for metrics and
  painted glyphs.
- [x] Add a cache-isolation test for two loaded files with the same display
  name but different bytes.

## 2.4 Closure gates

- [x] `go test ./internal/pdf ./internal/layout` passes with resolver tests.
- [x] No public API or compliance writer behavior changes unintentionally.
- [x] A focused fixture render records the selected embedded family and page
  count before proceeding to input-format work.

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
