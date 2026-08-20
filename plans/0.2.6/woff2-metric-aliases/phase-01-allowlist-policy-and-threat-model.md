# Phase 01: Allowlist policy and threat model

> **Status:** complete — validated 2026-08-20  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** closed `plans/0.2.5/font/`  
> **Unblocks:** Phase 02  
> **Amendment:** [amendments/2026-08-20-woff2-brotli-allowlist.md](amendments/2026-08-20-woff2-brotli-allowlist.md)

## Overview

Lock the Brotli allowlist exception and update threat-model wording so WOFF2
is treated as capped untrusted parse input, not “permanently skipped for
policy”.

## Checklist

- [x] Finalize amendment text (module pin, forbidden list, acceptance)  
  Evidence: `amendments/2026-08-20-woff2-brotli-allowlist.md` →
- [x] Promote `github.com/andybalholm/brotli` to a direct `go.mod` require
  (match canvas graph version when implementing)  
  Evidence: `go list -m -f '{{if and (not .Main) (not .Indirect)}}{{.Path}}{{end}}' all` →
- [x] Extend `TestDirectModuleAllowlist` allowed set to typesetting + canvas +
  brotli; keep HarfBuzz banned  
  Evidence: `go test ./internal/pdf -run TestDirectModuleAllowlist` →
- [x] Update `AGENTS.md` (and CONTRIBUTING / architecture notes that cite the
  two-module allowlist)  
  Evidence: paths →
- [x] Threat model / security posture: WOFF2 decoded with table/SFNT/Brotli
  output caps; still untrusted `@font-face` input via `FetchSub`  
  Evidence: `documentation/THREAT-MODEL.md` / related →
- [x] Confirm `CGO_ENABLED=0 go build ./..` green after allowlist change
  (decode may land in Phase 02)  
  Evidence: →

## Gates

- [x] Amendment acceptance rows addressed or explicitly deferred to Phase 02
  with pointer
- [x] No fourth direct module sneaks in
