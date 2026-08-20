# Phase 02: WOFF2 decode and caps

> **Status:** complete — validated 2026-08-20  
> **Parent:** [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)  
> **Depends on:** Phase 01  
> **Unblocks:** Phase 03

## Overview

Implement `DecodeWOFF2` in `internal/pdf`, wire `ParseFontBytes` `wOF2`
branch, and enforce WOFF1-class size/overlap caps plus Brotli output limits.

## Checklist

- [x] Add `DecodeWOFF2` (prefer `internal/pdf/woff2.go`)  
  Evidence: path →
- [x] `ParseFontBytes` calls decode instead of `errWOFF2Unsupported` for
  valid `wOF2`  
  Evidence: →
- [x] Caps: table count, per-table length, reconstructed SFNT size, Brotli
  output LimitReader (parity with `woffMax*` constants)  
  Evidence: constants + tests →
- [x] Reject `OTTO` / CFF flavor; reject collections / unknown critical
  transforms with clear errors  
  Evidence: unit tests →
- [x] Reconstruct transformed `glyf`/`loca` (WOFF2 transform v0); null-
  transform tables pass through  
  Evidence: round-trip / fixture →
- [x] Replace gap/skip unit expectations with positive decode + malformed
  cases  
  Evidence: `go test ./internal/pdf -run 'WOFF2|ParseFontBytes'` →
- [x] Post-decode `ParseTTF` still rejects `fvar`  
  Evidence: →

## Gates

- [x] `CGO_ENABLED=0 go test ./internal/pdf -run 'WOFF2|ParseFontBytes|DirectModule'` →
- [x] No panic on truncated / oversize / overlapping adversarial inputs
