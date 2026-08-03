# Phase 09 — Hardening & Closure Gates

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
> **Estimated effort:** 2–4 months overlapping late Phase 5–8  
> **Depends on:** MVP features (Phases 0–6)  
> **Unblocks:** release

---

## Overview

Evidence-backed quality bar for shipping MVP. Do not mark parent plan complete without these gates.

## Checklist

### 9.1 Corpus & correctness
- [ ] ≥20 golden HTML→PDF fixtures covering: invoices, tables, images, multi-page, HF, TOC, links
- [ ] Regression script: `go test ./testdata/golden/...` or `make golden`
- [ ] Flag support matrix complete (supported / partial / ignored / error)
- [ ] Compatibility matrix matches actual code

### 9.2 Security
- [ ] Local file ACL tests (deny by default)
- [ ] Remote URL: timeouts, max redirects, max bytes
- [ ] No JS execution paths remain
- [ ] Document threat model in README

### 9.3 Performance (record measurements — execution ≠ proof)
- [ ] Cold run: 10-page table report — command, machine, time, PDF bytes
- [ ] Warm run same
- [ ] Memory note (optional `pprof` snapshot path)

### 9.4 Tooling
- [ ] CI: test + vet on push (if GitHub Actions desired)
- [ ] `make lint` and `make test` both green — record output in parent ledger
- [ ] Reproducible builds (`CGO_ENABLED=0`)

### 9.5 Release
- [ ] VERSION file / ldflags version
- [ ] CHANGELOG for gowkhtmltopdf
- [ ] README install + usage + limits + time estimates reconciled
- [ ] Tag v0.1.0 MVP only after 9.1–9.4

### 9.6 Explicit remaining `[~]` items
- [ ] List every deferred feature with next gate (XSLT TOC, forms, SVG image, SOCKS5, complex scripts, …)

### 9.7 Closure
- [ ] Parent plan Phase 9 rows checked
- [ ] Handoff note: next unchecked work after MVP (Intermediate roadmap)

---

## Intermediate roadmap (post-MVP, not MVP gates)

| Item | Extra effort (order) |
|------|----------------------|
| Floats / position | +2–4 mo |
| Partial flex | +2–3 mo |
| Better CJK fonts | +1–2 mo |
| Forms AcroForm | +1–2 mo |
| Richer selectors | +1–2 mo |

Full WebKit parity remains **not planned**.
