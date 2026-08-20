# Phase 03: `@font-face` and prepare integration

> **Status:** complete — validated 2026-08-20  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phase 02  
> **Unblocks:** Phase 06 (WOFF2 corpus)

## Overview

Stop suffix-skipping `.woff2` in `fetchFontFace`. Local and HTTPS WOFF2
sources register through existing `MergeFontFaces` / ACL path on PDF and
image modes.

## Checklist

- [x] Remove `.woff2` from unsupported suffix list in
  `internal/convert/prepare/styles.go`; keep `.eot` and `data:` skips  
  Evidence: →
- [x] Local `@font-face` WOFF2 embeds family `/BaseFont` (mirror
  `TestFontFaceWOFFEmbed`)  
  Evidence: replace/extend `TestFontFaceWOFF2Skipped` →
- [x] Malformed `wOF2` → warn + CSS/bundled fallback; no false Custom face  
  Evidence: →
- [x] HTTPS `@font-face` WOFF2 uses `FetchSub` (same ACL/timeout/body); no
  new network policy  
  Evidence: →
- [x] Image-mode shares `MergeFontFaces` (no duplicate decoder)  
  Evidence: →
- [x] Prefer magic-based handling so mislabeled bodies still decode when
  `ParseFontBytes` sees `wOF2`  
  Evidence: note / test →

## Gates

- [x] `CGO_ENABLED=0 go test ./internal/convert ./internal/convert/prepare ./internal/imageout` →
