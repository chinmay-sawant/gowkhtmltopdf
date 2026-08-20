# Phase 08: Validation and closure

> **Status:** planned  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phases 01-07

## Overview

Record full gates, flip track status to complete, and leave
`plans/0.2.5/font/` closed.

## Checklist

- [ ] `CGO_ENABLED=0 make test` →  
  Evidence: →
- [ ] `CGO_ENABLED=0 make lint` →  
  Evidence: →
- [ ] `CGO_ENABLED=0 make build` →  
  Evidence: →
- [ ] `TestDirectModuleAllowlist` green with three directs  
  Evidence: →
- [ ] Optional `make samples` if a WOFF2 or alias sample was added  
  Evidence: →
- [ ] Canonical DoD rows in `00-canonical-…` checked with evidence  
  Evidence: →
- [ ] Track README status → complete; `plans/0.2.6/README.md` updated  
  Evidence: →
- [ ] pending-09 disposition `[~]` superseded / closed for WOFF2 decode  
  Evidence: →
- [ ] Confirm no checklist reopen under `plans/0.2.5/font/` phases 01-08  
  Evidence: →

## Closure rule

Mark this phase and the track complete only when every required row above
has a recorded command outcome or artifact path. Creating plan files is
not evidence.
