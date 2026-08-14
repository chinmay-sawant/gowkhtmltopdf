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

- [ ] `documentation/library-api.md` matches Phase 24 (preferred path, nil/error table, local-file helper)
- [ ] `documentation/architecture.md` import DAG matches Phase 27 (`layout` → `pdf` only if 27 left it)
- [ ] `documentation/fidelity.md` and `documentation/compatibility-matrix.md` match Phases 25–26
- [ ] `documentation/deferred.md` still lists JS, stdin `-`, forms, WOFF2, vertical writing
- [ ] `README.md` version badge / status line says **v0.2.1** when `VERSION` moves
- [ ] No new claim that fails `make claim-scan`

### 30.2 Ledger hygiene

- [ ] `plans/0.2.0/10-canonical-post-mvp-roadmap.md` constraint line no longer says “Zero module deps” / “Go standard library only” without the go-text / canvas amendment
- [ ] Any 0.2.0 row this ledger finished is `[~]` there with a pointer to `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
- [ ] This ledger’s status board matches the phase files
- [ ] Do not add a scored review document as a release artifact
- [ ] `plans/README.md` 0.2.1 row stays the active pointer until the tag

### 30.3 Package and CI comments

- [ ] `internal/pdf/doc.go` is current (Phase 27)
- [ ] Root `doc.go` examples use the preferred API
- [ ] `.github/workflows/ci.yml` branch list matches `CONTRIBUTING.md` (Phase 29)
- [ ] Architecture docs that cite `api.go` line counts either drop the counts or use `wc` at close-out

### 30.4 Release

- [ ] `VERSION` → `0.2.1`
- [ ] `CHANGELOG.md` section `0.2.1` lists user-visible contract, pagination/table, verification, and docs changes — not internal ledger filenames
- [ ] `cli.Version` stamp still matches `VERSION` (`make build` + `--version`)
- [ ] Release workflow (`v*` tags) unchanged unless Phase 29 required a branch name fix
- [ ] Frontend / `docs/` copy that states the version is regenerated if it is a release gate (`npm run build` dirty check)

### 30.5 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] `make claim-scan` →
- [ ] `make build` and `./bin/gowkhtmltopdf --version` equals `VERSION`
- [ ] Parent ledger Phase 30 row checked
- [ ] Parent ledger status line: complete (date)
- [ ] Handoff: leftover `[~]` rows (JS, fragmentainer, full table auto layout, `go test -race ./...` on every PR) listed below — do not open a second status document

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
