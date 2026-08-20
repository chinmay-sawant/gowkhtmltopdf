# Phase 04: Metric alias contract (resolver)

> **Status:** planned  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** closed `plans/0.2.5/font/` resolver  
> **Unblocks:** Phase 05  
> **Note:** No new `go.mod` dependency for this phase.

## Overview

Add an opt-in alias step to `FontResolver` after exact registry miss and
before bundled generics. Alias targets consult **Registry only**.

## v1 accept map (freeze)

| CSS token | Registry tries |
|-----------|----------------|
| `georgia` | Gelasio |
| `courier new` | Cousine |
| `times new roman` | Tinos |
| `arial` | Arimo |
| `cambria` | Caladea |
| `calibri` | Carlito |

## Checklist

- [ ] Extend `FontResolver` with opt-in enablement (default off)  
  Evidence: →
- [ ] Implement alias step in `resolveToken` after exact `Registry.Lookup`  
  Evidence: →
- [ ] Alias targets never bind bundled `FaceSet` Liberation names in v1  
  Evidence: →
- [ ] Keep default-path tests: `TestGelasioDoesNotRenameToGeorgia`,
  `TestFontResolverNoLegacyAliases`  
  Evidence: →
- [ ] Unit positives: Gelasio + aliases on → Georgia; off → Liberation via
  `serif`; absent Gelasio → stack continues; exact Georgia wins  
  Evidence: `go test ./internal/pdf -run 'Resolver|Alias|Gelasio'` →
- [ ] Cousine / Courier New case  
  Evidence: →
- [ ] `ResolveRune` uses the same token path (no alias bypass)  
  Evidence: →

## Gates

- [ ] Flag-off behavior matches v0.2.5 contract for bare `Georgia, serif`
- [ ] `CGO_ENABLED=0 go test ./internal/pdf ./internal/layout` →
