# Phase 28 - Settings and Request Types

> **Parent:** `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 1 week
> **Depends on:** Phase 24 policy (preferred public request)
> **Unblocks:** Phase 30 docs; fewer hand-copied policy structs

---

## Overview

Network policy, PDF requests, and dotted-key dispatch exist in more than
one shape. Image conversion already has `imageout.Request`. PDF still
has a union-ish `convert.Request` plus `convert.PDFRequest` plus public
`PDFRequest`. This phase collapses the duplicates that callers and
adapters must keep in sync.

## Executive Summary

| Type | Today | Target |
|------|-------|--------|
| `NetworkPolicy` | Defined in `api.go` and `internal/load/load.go` | One definition; the other is an alias or a conversion at the boundary |
| `convert.Request` | Still has `Image` / `ValidateImage` leftovers | PDF-only; image jobs do not construct it |
| `convert.PDFRequest` vs public `PDFRequest` | Parallel structs | Public type wraps or maps to the engine type in one function |
| `settings/reflect.go` | Hand-written maps, no `reflect` | Rename **or** a file comment that says it is not reflection |
| Ignored keys (Policy A) | `Ignored` map | Keep; optional `OnWarn` when a library `Set` stores an ignored key |

---

## Phase 28 checklist

### 28.1 Network policy

- [ ] Single `NetworkPolicy` struct owned by `internal/load` (trust boundary) — `internal/load/load.go`
- [ ] Public `gowkhtmltopdf.NetworkPolicy` is that type (alias) or a documented conversion in `api.go`
- [ ] `CompatibleNetworkPolicy` / `RestrictedNetworkPolicy` stay on the public package
- [ ] Settings flattened fields (`NetworkAllowedSchemes`, …) are filled from one conversion helper
- [ ] Test: restricted policy still blocks RFC1918, link-local, and `169.254.169.254` — `internal/load/load_test.go`

### 28.2 PDF request leftovers

- [ ] `convert.Request.Image` removed or ignored with a compile-time proof that no production caller sets it — `internal/convert/convert.go`
- [ ] `ValidateImage` is not on the PDF request type
- [ ] CLI / `internal/app` / `api.go` build `convert.PDFRequest` (or `NewPDFRequest`) only
- [ ] Test: `go test ./internal/convert ./internal/app` — no image settings accepted on a PDF request (`Err` already exists; keep it)

### 28.3 Public mapping

- [ ] One function maps public `PDFRequest` → engine request (clone ownership stays)
- [ ] One function maps public `ImageRequest` → `imageout.Request`
- [ ] `Now`, outline sink, and progress hooks: each exists on the surface that docs advertise (Phase 24); this phase does not add a fourth API

### 28.4 Dotted-key table

- [ ] `internal/settings/reflect.go`: rename to something that is not `reflect` **or** add a package comment that the file is hand-dispatched maps
- [ ] Descriptor parity tests still fail if `"smartshrinking"` and `PdfGlobal.SmartShrinking` drift
- [ ] `[~]` Typed builder covering every dotted key — not required; `WithSetting` remains the escape hatch

### 28.5 Policy A visibility

- [ ] Library `Set` of an ignored key (`load.jsdelay`, `web.javascript`, …) still succeeds
- [ ] Optional: `OnWarn` once per ignored key on `Converter` / typed run — if added, test it; if not, docs already say Policy A
- [ ] No new silent “success” flags for JS / plugins

### 28.6 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 28 row checked
- [ ] Next: Phase 29

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 24 preferred API | Phase 30 library docs |
| `internal/load` ACL tests | Unchanged Restricted policy |

---

## Out of scope

- Replacing dotted names with a full typed `PdfGlobal` struct for every key
- Functional-options rewrite
- Changing Policy A to fail-closed (would break wkhtml script compatibility)
