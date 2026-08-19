# Font Phase 1 — Resolution Contract and Baseline Matrix

> **Status:** planned
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

## 1.1 Current behavior inventory

Use the package map in
[00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
as the starting inventory; verify against code, do not invent new ownership.

- [ ] Record the current resolution path from CSS parsing through
  `layout.lookupFaceFor` and `faceForRune` (and HF `resolveHFFont`).
- [ ] Record every bundled face, its family name, weight, italic flag, and
  supported glyph coverage (`LoadDefaultFaces` / `internal/pdf/assets`).
- [ ] Record registry sources separately: `--font-path`, `--use-system-fonts`,
  and document `@font-face` (`MergeFontFaces` via `FetchSub`).
- [ ] Record the current silent-skip cases for invalid or unsupported font
  files (`ScanFontDirs` / `ParseTTF`) and the current operator log output
  (today: discovery skips are mostly silent; `@font-face` unsupported formats
  warn).
- [ ] Record the current `@font-face` limitation: `css.FontFace` stores only
  `Family` + `Src`; weight/style descriptors are **not retained** in the model
  (docs that say “parsed but ignored” understate this).
- [ ] Note docs drift to fix later: compatibility-matrix alias wording vs
  `FaceSet.ResolveFamily`; deferred.md HTTPS WOFF1 understatement.

## 1.2 Contract decisions

- [ ] Lock exact family matching as the first choice for explicitly registered
  faces.
- [ ] Lock author comma-stack order when a named family is not available.
- [ ] Lock generic mappings: `serif`, `sans-serif`, and `monospace` resolve to
  the bundled canonical families.
- [ ] Decide whether legacy aliases such as `Georgia`, `Arial`, and `Courier
  New` remain compatibility aliases or become final fallback aliases only.
  The recommended choice is exact family first, then author stack, then
  generic fallback.
- [ ] Lock the requested-weight/requested-style nearest-face rule, including
  whether synthetic bold/italic is permitted when no face exists.
- [ ] Lock the per-rune fallback order after the primary family is selected.
- [ ] Lock that Fontconfig aliases and host font ordering are never consulted
  by the default renderer.
- [ ] Lock that an unavailable optional face falls through to the CSS stack or
  bundled Liberation fallback instead of immediately returning an error.
- [ ] Lock that a face which fails actual subset/embed preflight is rejected
  before final paint and triggers fallback plus re-layout when possible.
- [ ] Lock the terminal error condition: return an error only when no valid
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

- [ ] Add expected font provenance and output metrics to the matrix.
- [ ] Include page-3 fixture-55 heading text and a wrapping-sensitive sample.
- [ ] Include a negative case proving a Fontconfig alias does not affect the
  default renderer.

## 1.4 Closure gates

- [ ] Decision record is written into this phase with no unresolved default
  resolution rule.
- [ ] Matrix has expected selected-family names and expected page/layout
  observations.
- [ ] Focused tests pass without changing generated corpus outputs.
- [ ] Parent plan is updated with the chosen contract and the next phase.

## Out of scope

- Implementing the resolver.
- Selecting or bundling a new Georgia-compatible font.
- Changing fixture CSS or committed PDFs.
