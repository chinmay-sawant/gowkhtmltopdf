# Phase 01: Allowlist policy and threat model

> **Status:** planned  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** closed `plans/0.2.5/font/`  
> **Unblocks:** Phase 02  
> **Amendment:** [amendments/2026-08-20-woff2-brotli-allowlist.md](amendments/2026-08-20-woff2-brotli-allowlist.md)

## Overview

Lock the Brotli allowlist exception and update threat-model wording so WOFF2
is treated as capped untrusted parse input, not “permanently skipped for
policy”.

## Checklist

- [ ] Finalize amendment text (module pin, forbidden list, acceptance)  
  Evidence: `amendments/2026-08-20-woff2-brotli-allowlist.md` →
- [ ] Promote `github.com/andybalholm/brotli` to a direct `go.mod` require
  (match canvas graph version when implementing)  
  Evidence: `go list -m -f '{{if and (not .Main) (not .Indirect)}}{{.Path}}{{end}}' all` →
- [ ] Extend `TestDirectModuleAllowlist` allowed set to typesetting + canvas +
  brotli; keep HarfBuzz banned  
  Evidence: `go test ./internal/pdf -run TestDirectModuleAllowlist` →
- [ ] Update `AGENTS.md` (and CONTRIBUTING / architecture notes that cite the
  two-module allowlist)  
  Evidence: paths →
- [ ] Threat model / security posture: WOFF2 decoded with table/SFNT/Brotli
  output caps; still untrusted `@font-face` input via `FetchSub`  
  Evidence: `documentation/THREAT-MODEL.md` / related →
- [ ] Confirm `CGO_ENABLED=0 go build ./..` green after allowlist change
  (decode may land in Phase 02)  
  Evidence: →

## Gates

- [ ] Amendment acceptance rows addressed or explicitly deferred to Phase 02
  with pointer
- [ ] No fourth direct module sneaks in
