# Phase 00 — Scope Freeze & Project Foundations

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 2–4 weeks (solo) · 1–2 weeks (pair)  
> **Depends on:** none  
> **Unblocks:** all later phases

---

## Overview

Lock the product contract before writing engine code. A pure-stdlib Go rewrite cannot be “full wkhtmltopdf”; this phase documents what **will** and **will not** ship for MVP.

## Executive Summary

Upstream maintainers already steer report users to WeasyPrint/Prince and dynamic sites to Puppeteer (`wkhtmltopdf/docs/status.md`). Our MVP is a **controlled HTML → PDF** tool with CLI familiarity, not a browser.

---

## Checklist

### 0.1 HTML/CSS allowlist document
- [x] Create `docs/compatibility-matrix.md` (or `plans/compatibility-matrix.md`)
- [x] List supported tags: `html,head,body,title,meta,style,link,div,span,p,br,hr,h1-h6,ul,ol,li,table,thead,tbody,tfoot,tr,th,td,img,a,strong,em,b,i,u,small,pre,code,blockquote`
- [x] List supported CSS properties (box model, font, color, border, text-align, display block/inline/table, page-break-*)
- [x] List supported units: `px, pt, mm, cm, in, em, %`
- [x] List unsupported: flex/grid (MVP), float stacks, position fixed, transforms, filters, `@font-face` remote, JS

### 0.2 Non-goals & security
- [x] JS: permanently out of MVP (strip scripts; `--enable-javascript` ignored with warning)
- [x] Untrusted HTML: document “not for untrusted input” same as upstream
- [x] Default `blockLocalFileAccess=true`
- [x] SSRF policy note for remote URLs (timeouts, redirect limits)

### 0.3 Fixture corpus
- [x] Create `testdata/golden/README.md` describing HTML in / PDF out / tolerance
- [x] Seed ≥3 invoice-like fixtures (simple, table-heavy, multi-page)
- [x] Define pass criteria (structure + optional image diff later)

### 0.4 Go project scaffold
- [x] `go mod init` (module path agreed: `gowkhtmltopdf`)
- [x] Package tree:
  ```
  cmd/gowkhtmltopdf/
  cmd/gowkhtmltoimage/
  internal/settings/
  internal/cli/
  internal/load/
  internal/html/
  internal/css/
  internal/layout/
  internal/pdf/
  internal/outline/
  internal/convert/
  internal/imageout/
  ```
- [x] `Makefile`: `test`, `lint` (`go test ./...`, `go vet ./...` + gofmt check)
- [x] `.gitignore` for binaries, coverage, temp

### 0.5 Closure gates
- [ ] Allowlist reviewed (human sign-off recorded in this file) — **pending human sign-off on `docs/compatibility-matrix.md`**
- [x] Scaffold builds: `go build ./...` (empty mains ok)
- [x] Proof: `go test ./...` exits 0 on empty packages
      Evidence 2026-08-03: `go build ./... && go vet ./... && go test ./...` exit 0; `make test` and `make lint` both pass.

---

## Dependencies

None. Must complete before Phase 4 design freezes.

## Deliverables

| Artifact | Path |
|----------|------|
| Canonical parent update | Phase 0 rows in `plans/00-canonical-pure-go-rewrite.md` |
| Compatibility matrix | `docs/compatibility-matrix.md` |
| Module scaffold | repo root |

## Risks

- Starting layout before allowlist → unbounded scope (R1).
