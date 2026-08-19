# Font Phase 1 — Resolution Contract and Baseline Matrix

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** current `FaceSet`, `Registry`, CSS cascade, and fixture corpus
> **Unblocks:** Phase 2

## Overview

Turn the Chrome/WeasyPrint/Gowkhtmltopdf observations into a precise renderer
contract before changing selection code. This phase must distinguish:

- an exact author-selected font;
- a deterministic bundled substitute;
- a missing-glyph fallback;
- a host-specific alias that must not leak into default output.


## Decision record (locked 2026-08-19)

1. Exact registered/`@font-face`/`--font-path` family match first (name-table), per CSS token order.
2. Author comma-stack continues when a named family is absent.
3. Generics only: `serif`/`sans-serif`/`monospace` → Liberation; `system-ui` → DejaVu. Real Liberation names also match.
4. Legacy display names (Georgia/Arial/Times/Courier New/…) are **not** aliased. They win only when supplied; otherwise the stack continues (e.g. `Georgia, serif` → Liberation Serif via generic).
5. Weight/style: nearest face within family (`resolveFamilyFaces`); **no synthetic bold/italic**.
6. Per-rune: keep primary when it covers the glyph; else family stack → Liberation → DejaVu → `FindWithGlyph`.
7. No Fontconfig / host alias import; discovery remains opt-in.
8. Unavailable optional faces warn and continue; embed preflight failure → MarkUnavailable + re-layout (`layout.WithFontPreflight`).
9. Terminal error only when no embeddable face remains for required text.
10. Variable fonts (`fvar`): **rejected** with diagnostic; WOFF2 remains a separate allowlist epic.

## 1.1 Current behavior inventory

Use the package map in
[00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
as the starting inventory; verify against code, do not invent new ownership.

- [x] Record the current resolution path from CSS parsing through
  `layout.lookupFaceFor` and `faceForRune` (and HF `resolveHFFont`).
- [x] Record every bundled face, its family name, weight, italic flag, and
  supported glyph coverage (`LoadDefaultFaces` / `internal/pdf/assets`).
- [x] Record registry sources separately: `--font-path`, `--use-system-fonts`,
  and document `@font-face` (`MergeFontFaces` via `FetchSub`).
- [x] Record the current silent-skip cases for invalid or unsupported font
  files (`ScanFontDirs` / `ParseTTF`) and the current operator log output
  (today: discovery skips are mostly silent; `@font-face` unsupported formats
  warn).
- [x] Record the current `@font-face` limitation: `css.FontFace` stores only
  `Family` + `Src`; weight/style descriptors are **not retained** in the model
  (docs that say “parsed but ignored” understate this).
- [x] Note docs drift to fix later: compatibility-matrix alias wording vs
  `FaceSet.ResolveFamily`; deferred.md HTTPS WOFF1 understatement.

## 1.2 Contract decisions

- [x] Lock exact family matching as the first choice for explicitly registered
  faces.
- [x] Lock author comma-stack order when a named family is not available.
- [x] Lock generic mappings: `serif`, `sans-serif`, and `monospace` resolve to
  the bundled canonical families.
- [x] Decide whether legacy aliases such as `Georgia`, `Arial`, and `Courier
  New` remain compatibility aliases or become final fallback aliases only.
  The recommended choice is exact family first, then author stack, then
  generic fallback.
- [x] Lock the requested-weight/requested-style nearest-face rule, including
  whether synthetic bold/italic is permitted when no face exists.
- [x] Lock the per-rune fallback order after the primary family is selected.
- [x] Lock that Fontconfig aliases and host font ordering are never consulted
  by the default renderer.
- [x] Lock that an unavailable optional face falls through to the CSS stack or
  bundled Liberation fallback instead of immediately returning an error.
- [x] Lock that a face which fails actual subset/embed preflight is rejected
  before final paint and triggers fallback plus re-layout when possible.
- [x] Lock the terminal error condition: return an error only when no valid
  face can cover the required text and satisfy the writer contract.

## 1.3 Acceptance matrix

Create a small HTML/CSS matrix covering these cases:

| Case | Expected result |
|------|-----------------|
| `font-family: serif` | bundled Liberation Serif |
| `font-family: Georgia, serif` without Georgia supplied | deterministic serif fallback, not host Gelasio |
| `font-family: Georgia, serif` with a family-named Georgia TTF supplied | supplied Georgia face |
| `font-family: Missing, sans-serif` | bundled sans face after the missing family |
| `font-family: Custom` with document `@font-face` | document face |
| supplied face fails parsing | warning plus CSS/bundled fallback |
| supplied face fails subset/embed preflight | fallback before final paint and re-layout |
| same family at 400/700/italic/700 italic | correct registered face or documented nearest-face behavior |
| Latin plus a missing Unicode glyph | primary face for Latin, documented glyph fallback for the missing rune |
| same family from two files with the same display name | stable fingerprint-based selection/cache isolation |

- [x] Add expected font provenance and output metrics to the matrix.
- [x] Include page-3 fixture-55 heading text and a wrapping-sensitive sample.
- [x] Include a negative case proving a Fontconfig alias does not affect the
  default renderer.

## 1.4 Closure gates

- [x] Decision record is written into this phase with no unresolved default
  resolution rule.
- [x] Matrix has expected selected-family names and expected page/layout
  observations.
- [x] Focused tests pass without changing generated corpus outputs.
- [x] Parent plan is updated with the chosen contract and the next phase.

## Out of scope

- Implementing the resolver.
- Selecting or bundling a new Georgia-compatible font.
- Changing fixture CSS or committed PDFs.

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
