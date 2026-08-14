# Phase 30 - Docs, Ledger Hygiene, and Release Closure

> **Parent:** `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 3–5 days after 24–29
> **Depends on:** Phases 24–29 closure gates
> **Unblocks:** v0.2.1 tag

---

## Overview

User docs, `claim-scan`, and `documentation/deferred.md` are the
source of truth for what the engine does. Some ledgers and package
comments still describe an earlier tree (stdlib-only, Phase 00
scaffold, `api.go` line counts). This phase makes those artifacts
match the code, stamps the version, and closes the 0.2.1 ledger.

## Executive Summary

| Artifact | Today | Target |
|----------|-------|--------|
| `documentation/deferred.md` | Live inventory | Still wins when other docs disagree |
| `plans/0.2.0/10-canonical-post-mvp-roadmap.md` | Header still mentions stdlib-only / zero module deps | `[~]` superseded rows point here; stale constraint line removed or dated |
| `internal/pdf/doc.go` | Phase 00 scaffold | Done in Phase 27; verify |
| `VERSION` | `0.2.0` | `0.2.1` only after 24–29 gates |
| Architecture line counts | Drift from `api.go` | Numbers dropped or regenerated |

---

## Phase 30 checklist

### 30.1 User docs

- [x] `documentation/library-api.md` matches Phase 24 (preferred path, nil/error table, local-file helper)
- [x] `documentation/architecture.md` import DAG matches Phase 27
- [x] `documentation/fidelity.md` and `documentation/compatibility-matrix.md` match Phases 25–26
- [x] `documentation/deferred.md` still lists JS, stdin `-`, forms, WOFF2, vertical writing
- [x] `README.md` version badge / status line says **v0.2.1**
- [x] No new claim that fails `make claim-scan`

### 30.2 Ledger hygiene

- [x] `plans/0.2.0/10-canonical-post-mvp-roadmap.md` constraint line updated to reflect allowlisted modules
- [x] 0.2.0 finished rows point to `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
- [x] This ledger’s status board matches the phase files
- [x] Review documents preserved cleanly
- [x] `plans/README.md` updated with 0.2.1 roadmap pointer

### 30.3 Package and CI comments

- [x] `internal/pdf/doc.go` updated to describe pure-Go PDF 1.4 writer
- [x] Root `doc.go` examples use the preferred API
- [x] `.github/workflows/ci.yml` branch list includes `master` and `main`
- [x] Architecture docs consistent with source line references

### 30.4 Release

- [x] `VERSION` → `0.2.1`
- [x] `CHANGELOG.md` section `0.2.1` lists user-visible contract, pagination/table, verification, and docs changes
- [x] `cli.Version` stamp matches `VERSION` (`make build` + `--version`)
- [x] Release workflow (`v*` tags) verified
- [x] Frontend / docs build verified clean

### 30.5 Closure gates

- [x] `make lint` → PASSED (golangci-lint run ./... clean)
- [x] `make test` → PASSED (go test ./... clean)
- [x] `make claim-scan` → PASSED (zero disallowed claims)
- [x] `make build` and `./bin/gowkhtmltopdf --version` equals `VERSION` (0.2.1)
- [x] Parent ledger Phase 30 row checked
- [x] Parent ledger status line: complete (2026-08-14)
- [x] Handoff: deferred items documented in `documentation/deferred.md`

---

## Leftover `[~]` after v0.2.1 (handoff)

| Item | Next gate |
|------|-----------|
| JavaScript | Phase 22 (not opened) |
| Chrome / open-web parity | Phase 23 (not opened) |
| Full CSS2.1 table auto layout + collapsing-border conflicts | New ledger after a fixture requires it |
| CSS Fragmentation fragmentainer | New ledger after display-list pagination fails a named invoice |
| Multi-float CSS2.1 placement | Phase 26 `[~]` |
| Typed builder for every dotted key | Phase 28 `[~]` |
| Pixel goldens | Not planned |

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 24–29 | v0.2.1 tag |

---

## Out of scope

- Rewriting the 0.2.0 review archive
- New product website features
- Adding dependencies
