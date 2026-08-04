# Tier 2 Subplan - Image-mode `@font-face` parity

> **Parent:** [`plans/phases/phase-19-fonts-i18n.md`](../phase-19-fonts-i18n.md) (19.3 / image mode)  
> **Status:** not started  
> **Estimated effort:** 0.5–2 days  
> **Constraint:** stdlib-only; reuse PDF `mergeFontFaces` ACL path  
> **Related:** [`phase-19-pending.md`](phase-19-pending.md) Partial boundaries

---

## Overview

PDF convert already loads local `@font-face` TTF/OTF via `mergeFontFaces` +
`FetchSub` ACL. Image mode (`internal/imageout`) only applies `--font-path` /
`--use-system-fonts`. This subplan wires the **same** `@font-face` merge into
the image pipeline so CSS-declared local faces work for PNG/JPEG output.

## Executive Summary

| Path | `@font-face` local TTF today | Target |
|------|------------------------------|--------|
| PDF (`convert.mergeFontFaces`) | Yes | Keep |
| Image (`imageout`) | No | Call shared merge |

---

## Phase 1: Shared helper

### 1.1 Extract

- [ ] Refactor `mergeFontFaces` (or thin wrapper) into a package both convert and
      imageout can call without import cycles — prefer `internal/convert` export
      used by imageout **or** move to `internal/fonts` / keep in convert and call
      from imageout if already depending
- [ ] Preserve: skip WOFF/WOFF2/https/`data:`; ACL via loader `FetchSub`
- [ ] Preserve: weight/style descriptors ignored (document) unless wired later
- [ ] Path: `internal/convert/convert.go`; `internal/imageout/imageout.go`

### 1.2 Wire imageout

- [ ] After stylesheet parse / before layout in image pipeline, merge font faces
      into the registry used by layout
- [ ] Proof: image convert with `@font-face` + Custom family renders glyphs
      (not tofu) when ACL allows

---

## Phase 2: Tests

### 2.1 Cases

- [ ] Image mode: local `@font-face` embeds/uses face (ACL on)
- [ ] Image mode: ACL deny → fallback, no panic
- [ ] PDF FontFace tests still green (`go test ./internal/convert -run FontFace`)
- [ ] Path: `internal/imageout/*_test.go` (new or extend)

### 2.2 Gates

- [ ] `go test ./internal/imageout -count=1`
- [ ] `go test ./internal/convert -run FontFace -count=1`
- [ ] `make lint` → ; `make test` → ; record outcomes

---

## Phase 3: Docs

### 3.1 Honesty

- [ ] `documentation/fonts.md`: remove “image mode N/A” for local `@font-face`
- [ ] Matrix `@font-face` Partial note: PDF + image local TTF
- [ ] Phase 19 Pending / 19.7 rows updated

---

## Phase 4: Closure

- [ ] Image + PDF parity for local `@font-face` proven
- [ ] Parent phase-19 `[~]` image-mode row → `[x]`
- [ ] Next: shaping-gotext or sticky as prioritized

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| PDF `mergeFontFaces` | Shared behavior |
| Image layout registry | Face resolution |

---

## Out of scope

- Remote webfont download
- WOFF2
- Applying `@font-face` font-weight/style descriptors (optional follow-up)
