# Phase 09 — Hardening & Closure Gates

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 2–4 months overlapping late Phase 5–8  
> **Depends on:** MVP features (Phases 0–6)  
> **Unblocks:** release

---

## Overview

Evidence-backed quality bar for shipping MVP. Do not mark parent plan complete without these gates.

## Checklist

### 9.1 Corpus & correctness
- [x] ≥20 golden HTML→PDF fixtures covering: invoices, tables, images, multi-page, HF, TOC, links — 20 fixtures in `testdata/golden/` (01–20: invoices, tables+colspan, images, page-breaks, lists, pre, typography, cover, margins/sizing), each with a comment header; assets `logo.png`, `style-05.css`
- [x] Regression script: `go test ./testdata/golden/...` or `make golden` — `make golden` = `go test ./internal/convert/ -run 'TestGoldenCorpus' -v`; `TestGoldenCorpusAllFixtures` walks all fixtures, asserts PDF structure + per-fixture page envelopes + feature xobjects
- [x] Flag support matrix complete (supported / partial / ignored / error) — documentation/compatibility-matrix.md §7: 109 flags classified (~71 Supported, 5 Partial, 33 Ignored, 0 Error); every dotted key traced to its consumer
- [x] Compatibility matrix matches actual code — verified against `applyRestProps`/`uaRules`/consumers; stale zoom note corrected

### 9.2 Security
- [x] Local file ACL tests (deny by default) — internal/load tests incl. symlink escape (fixed), path traversal (raw + %2e%2e), file:// hosts, subresource ACL
- [x] Remote URL: timeouts, max redirects, max bytes — connect 30s + request 60s defaults; redirect cap (off-by-one fixed); MaxBodySize enforced on HTTP (Content-Length + read-side) and files (was unbounded); tests: oversized bodies, redirect limit, slow server, ctx cancel mid-read
- [x] No JS execution paths remain — audit: zero os/exec outside test helpers; --enable-javascript accepted but inert (warned)
- [x] Document threat model in README — documentation/THREAT-MODEL.md (trust boundary, ACL matrix, network behavior, exfiltration, out-of-scope, container guidance)

### 9.3 Performance (record measurements — execution ≠ proof)
- [x] Cold run: 10-page table report — command, machine, time, PDF bytes — internal/convert/perf_test.go TestTenPageTableReportPerformance: full RunPDF pipeline, 10-page report; cold ≈ 111 ms, warm ≈ 100 ms, 96,341 bytes; machine go1.26.4 linux/amd64 i7-13700HX; numbers in test comment + README
- [x] Warm run same — measured in the same test (run twice)
- [x] Memory note (optional `pprof` snapshot path) — README documents `go test -cpuprofile` + `go tool pprof -top`

### 9.4 Tooling
- [x] CI: test + vet on push (if GitHub Actions desired) — .github/workflows/ci.yml: test+lint job and CGO_ENABLED=0 static build job on push+PR
- [x] `make lint` and `make test` both green — record output in parent ledger — `go test ./...` all ok, `go vet ./...` clean, `gofmt -l .` empty (2026-08-03)
- [x] Reproducible builds (`CGO_ENABLED=0`) — verified: both binaries statically linked; command in README

### 9.5 Release
- [x] VERSION file / ldflags version — VERSION = 0.1.0; cli.Version ldflags-stampable (`-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)`, fallback 0.1.0-dev); command in README
- [x] CHANGELOG for gowkhtmltopdf — CHANGELOG.md, section "0.1.0 (2026-08-03)" with phases 0–9 + limitations
- [x] README install + usage + limits + time estimates reconciled — README rewritten (12 sections): install/build, both CLIs with live-verified examples, library snippet, perf table, versioning, deferred list, docs index
- [x] Tag v0.1.0 MVP only after 9.1–9.4 — pending tag at final verification (post-commit)

### 9.6 Explicit remaining `[~]` items
- [x] List every deferred feature with next gate (XSLT TOC, forms, SVG image, SOCKS5, complex scripts, …) — README "Deferred / not planned": XSLT TOC, AcroForm, SVG/BMP, SOCKS5, JS, floats/position, flex/grid, richer selectors, CJK fonts, bitmap-font AA, inline-anchor rects, cross-object URL map, resolveRelativeLinks, HTML-HF links on body pages, [topage] with copies, [subject], dump-outline TOC offset, table-header-repeat, PDF/A, cgo ABI, stdin batch loop; Intermediate roadmap quoted

### 9.7 Closure
- [x] Parent plan Phase 9 rows checked — ledger row `[x] 2026-08-03`, status line updated
- [x] Handoff note: next unchecked work after MVP (Intermediate roadmap) — README deferred list + phase-09 Intermediate roadmap table (floats, partial flex, CJK, forms, selectors)

---

## Design notes (filled 2026-08-03)

- **Fixture 08 regression found and fixed during closure**: `page-break-before` sections stacked directly on the previous block (collapsed margins → previous boundary op at exactly the section top) dragged that op along each fixpoint iteration, drifting one page per iteration (26 pages instead of 5). Fixed in `internal/layout/paint.go` `shiftFlowY` (range-scoped + strictly-below shifts); regression covered by `TestPageBreakBeforeStacked` and fixture-08 (workaround removed).
- **Relative subresource resolution fixed**: `load.Base` for file resources lacked the trailing slash, so `url.ResolveReference` resolved relative css/img against the parent directory; fixture-05/06/07 assets now load end-to-end.
- **Security fixes landed in 9.2**: symlink escape in the ACL, unbounded file reads, HTTP body silently truncated (LimitReader without check), redirect-cap off-by-one, `file://remote-host` accepted, dead `NewAccessController` removed.
- **Known remaining `[~]`**: table-header repeat across pages not implemented; dump-outline XML pages body-relative; TOC fixed-point may drift if re-layout changes count.

## Intermediate roadmap (post-MVP, not MVP gates)

| Item | Extra effort (order) |
|------|----------------------|
| Floats / position | +2–4 mo |
| Partial flex | +2–3 mo |
| Better CJK fonts | +1–2 mo |
| Forms AcroForm | +1–2 mo |
| Richer selectors | +1–2 mo |

Full WebKit parity remains **not planned**.
