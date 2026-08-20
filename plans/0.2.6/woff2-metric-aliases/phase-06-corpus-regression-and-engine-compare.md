# Phase 06: Corpus, regression, and engine compare notes

> **Status:** complete — validated 2026-08-20  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phases 03 and 05  
> **Unblocks:** Phase 07

## Overview

Prove WOFF2 embed and opt-in alias behavior end-to-end without weakening
no-flag Liberation defaults. Engine compares are differential notes, not
pixel-parity gates.

## Checklist

### WOFF2

- [x] Convert test: local WOFF2 `@font-face` produces `/FontFile2` for the
  custom family  
  Evidence: →
- [x] Convert test: bad WOFF2 falls back; no Custom `/BaseFont`  
  Evidence: →
- [x] Optional HTTPS fixture under ACL (skip cleanly if offline policy
  denies)  
  Evidence: →

### Metric aliases

- [x] Convert: patched Gelasio on `--font-path` + `--use-metric-font-aliases`
  → Georgia resolves to Gelasio  
  Evidence: →
- [x] Same HTML, aliases off → Liberation Serif via `serif`  
  Evidence: →
- [x] Cousine / Courier New pair  
  Evidence: →
- [x] Image mode shares alias behavior  
  Evidence: →
- [x] Embed-preflight still falls through if aliased face fails embed  
  Evidence: →

### Fixture-55 / WeasyPrint notes

- [x] Record no-flag fixture-55 still Liberation (regression guard)  
  Evidence: PDF/PNG path or test →
- [x] Optional host evidence: `--use-system-fonts --use-metric-font-aliases`
  when Gelasio present: differential note only (not CI-hard on host fonts)  
  Evidence: →

## Gates

- [x] `CGO_ENABLED=0 go test ./internal/convert ./internal/imageout` →
- [x] No-flag Georgia→Liberation contract unchanged
