# Font Phase 2 — Central Resolver Seam

> **Status:** planned
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phase 1
> **Unblocks:** Phases 3–6

## Overview

Centralize family and glyph selection behind one internal `FontResolver`
module. The module should provide depth: layout knows how to ask for a face,
while registry precedence, aliases, style selection, glyph fallback, and face
identity stay local to the resolver.

## 2.1 Interface and ownership

- [ ] Place the resolver under `internal/pdf` beside `FaceSet` / `Registry`
  unless Phase 2 proves a layering cycle; do not invent `internal/font` by
  default.
- [ ] Define the smallest internal interface needed by layout **and HF** for:
  - family/style resolution;
  - missing-glyph resolution for a selected family stack;
  - last-resort registered glyph coverage.
- [ ] Keep `FaceSet`, `Registry`, and PDF `Font` implementation details behind
  the resolver seam where practical.
- [ ] Route `lookupFaceFor` / `faceForRune` / `facesWithGlyph` and
  `resolveHFFont` through the same seam so PDF, image, and HF stop diverging.
- [ ] Keep the public `Document`, CLI, and library font options unchanged.
- [ ] Make resolver construction explicit from bundled faces plus an optional
  registry; do not construct or scan host fonts inside layout.
- [ ] Decide whether diagnostics are returned as a separate result or emitted
  through the existing conversion log without making diagnostics part of the
  hot layout interface.
- [ ] Leave embed-preflight re-layout orchestration to `internal/convert`
  (Phase 5); the resolver only marks candidates unavailable.

## 2.2 Resolution implementation

- [ ] Implement exact registered-family lookup in CSS order.
- [ ] Implement generic-family mapping after named-family misses.
- [ ] Implement the Phase 1 weight/style selection rule.
- [ ] Implement per-rune fallback through the same family policy used for the
  primary face; remove duplicated policy from `facesWithGlyph` and registry
  lookup paths where safe.
- [ ] Give the resolver a way to mark a candidate unavailable with a
  diagnostic and continue to the next CSS/bundled face.
- [ ] Keep fallback selection before layout is committed; do not silently
  replace a face after its metrics have been used.
- [ ] Preserve deterministic tie-breaking for duplicate files and duplicate
  family names.
- [ ] Include the font fingerprint, selected face, family stack, weight/style,
  and rune where required in cache identity.
- [ ] Ensure a selected face with an unsupported glyph cannot cause the
  resolver to silently switch the whole Latin run to an unrelated face.

## 2.3 Tests

- [ ] Update `internal/pdf/registry_lookup_test.go` for the final named-family
  and generic-family contract.
- [ ] Replace or narrow `TestFaceResolveFamilyAliases` so it asserts the chosen
  policy rather than an accidental direct alias.
- [ ] Add resolver tests for exact custom family, generic fallback, author-stack
  continuation, style selection, duplicate-family tie-breaking, parse-failure
  fallback, and rune fallback.
- [ ] Add layout tests proving the same resolver result is used for metrics and
  painted glyphs.
- [ ] Add a cache-isolation test for two loaded files with the same display
  name but different bytes.

## 2.4 Closure gates

- [ ] `go test ./internal/pdf ./internal/layout` passes with resolver tests.
- [ ] No public API or compliance writer behavior changes unintentionally.
- [ ] A focused fixture render records the selected embedded family and page
  count before proceeding to input-format work.
