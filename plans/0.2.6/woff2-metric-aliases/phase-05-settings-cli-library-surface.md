# Phase 05: Settings, CLI, and library surface

> **Status:** planned  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phase 04  
> **Unblocks:** Phase 06

## Overview

Expose the opt-in alias flag across CLI, dotted settings, and
`Document` / `ImageDocument`, defaulting false everywhere. Wire into
convert/layout `NewFontResolver`.

## Checklist

- [ ] `PdfGlobal.UseMetricFontAliases` + reflect key `usemetricfontalias`  
  Evidence: →
- [ ] CLI `--use-metric-font-aliases` on PDF and image (`ModeBoth`), beside
  `--use-system-fonts`  
  Evidence: →
- [ ] `Document` / `ImageDocument` field `UseMetricFontAliases bool`  
  Evidence: →
- [ ] Convert / imageout plumbing passes the bit into `FontResolver`  
  Evidence: →
- [ ] Zero-value / default Document asserts false  
  Evidence: →
- [ ] CLI parse test for the new flag  
  Evidence: →
- [ ] Optional info diagnostic when flag on and registry empty (aliases
  cannot fire)  
  Evidence: →

## Naming lock

Prefer `--use-metric-font-aliases` over `--use-fontconfig-aliases`. The
engine does not run Fontconfig.

## Gates

- [ ] `CGO_ENABLED=0 go test ./internal/cli ./internal/settings ./internal/convert` →
- [ ] Public API godoc states default false and discovery independence
