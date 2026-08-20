# Phase 05: Settings, CLI, and library surface

> **Status:** complete — validated 2026-08-20  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phase 04  
> **Unblocks:** Phase 06

## Overview

Expose the opt-in alias flag across CLI, dotted settings, and
`Document` / `ImageDocument`, defaulting false everywhere. Wire into
convert/layout `NewFontResolver`.

## Checklist

- [x] `PdfGlobal.UseMetricFontAliases` + reflect key `usemetricfontalias`  
  Evidence: →
- [x] CLI `--use-metric-font-aliases` on PDF and image (`ModeBoth`), beside
  `--use-system-fonts`  
  Evidence: →
- [x] `Document` / `ImageDocument` field `UseMetricFontAliases bool`  
  Evidence: →
- [x] Convert / imageout plumbing passes the bit into `FontResolver`  
  Evidence: →
- [x] Zero-value / default Document asserts false  
  Evidence: →
- [x] CLI parse test for the new flag  
  Evidence: →
- [x] Optional info diagnostic when flag on and registry empty (aliases
  cannot fire)  
  Evidence: →

## Naming lock

Prefer `--use-metric-font-aliases` over `--use-fontconfig-aliases`. The
engine does not run Fontconfig.

## Gates

- [x] `CGO_ENABLED=0 go test ./internal/cli ./internal/settings ./internal/convert` →
- [x] Public API godoc states default false and discovery independence
