# Phase 06: Corpus, regression, and engine compare notes

> **Status:** planned  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phases 03 and 05  
> **Unblocks:** Phase 07

## Overview

Prove WOFF2 embed and opt-in alias behavior end-to-end without weakening
no-flag Liberation defaults. Engine compares are differential notes, not
pixel-parity gates.

## Checklist

### WOFF2

- [ ] Convert test: local WOFF2 `@font-face` produces `/FontFile2` for the
  custom family  
  Evidence: →
- [ ] Convert test: bad WOFF2 falls back; no Custom `/BaseFont`  
  Evidence: →
- [ ] Optional HTTPS fixture under ACL (skip cleanly if offline policy
  denies)  
  Evidence: →

### Metric aliases

- [ ] Convert: patched Gelasio on `--font-path` + `--use-metric-font-aliases`
  → Georgia resolves to Gelasio  
  Evidence: →
- [ ] Same HTML, aliases off → Liberation Serif via `serif`  
  Evidence: →
- [ ] Cousine / Courier New pair  
  Evidence: →
- [ ] Image mode shares alias behavior  
  Evidence: →
- [ ] Embed-preflight still falls through if aliased face fails embed  
  Evidence: →

### Fixture-55 / WeasyPrint notes

- [ ] Record no-flag fixture-55 still Liberation (regression guard)  
  Evidence: PDF/PNG path or test →
- [ ] Optional host evidence: `--use-system-fonts --use-metric-font-aliases`
  when Gelasio present: differential note only (not CI-hard on host fonts)  
  Evidence: →

## Gates

- [ ] `CGO_ENABLED=0 go test ./internal/convert ./internal/imageout` →
- [ ] No-flag Georgia→Liberation contract unchanged
