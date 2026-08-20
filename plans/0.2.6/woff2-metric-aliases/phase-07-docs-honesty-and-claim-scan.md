# Phase 07: Docs honesty and claim-scan

> **Status:** planned  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phases 03-06 (or docs-only stubs if code not yet merged)  
> **Unblocks:** Phase 08

## Overview

Align operator docs, deferred inventory, pending-09, and knowledge-base with
shipped behavior. Keep no-flag WeasyPrint divergence intentional; document
the opt-in path without over-claiming Fontconfig or PDF profile conformance.

## Checklist

- [ ] `documentation/fonts.md`: WOFF2 accepted after decode; alias flag;
  default Liberation table retained  
  Evidence: →
- [ ] `documentation/cli.md` / library docs: `--use-metric-font-aliases` /
  `UseMetricFontAliases`  
  Evidence: →
- [ ] `documentation/deferred.md`: rewrite WOFF2 and metric-alias rows to
  shipped or partial with this plan as gate  
  Evidence: →
- [ ] `documentation/compatibility-matrix.md` / `fidelity.md` as needed  
  Evidence: →
- [ ] Threat / integration-security: stop listing `.woff2` as unconditionally
  skipped once decode ships  
  Evidence: →
- [ ] Close remaining pending-09 WOFF2 rows with pointer to this track  
  Evidence: →
- [ ] KB: fonts synthesis, deferred summary, roadmap, log, index  
  Evidence: →
- [ ] `CHANGELOG.md` user-facing notes under the release that ships the work  
  Evidence: →
- [ ] `make claim-scan` if fidelity language touched  
  Evidence: →

## Honesty locks

- Do **not** claim PDF/A or PDF/UA from WOFF2 or aliases.
- Do **not** call the opt-in map “Fontconfig compatible”; say
  “metric-compatible accept list inspired by Fontconfig 30-metric-aliases”.
- Do **not** imply the alias flag discovers fonts by itself.
