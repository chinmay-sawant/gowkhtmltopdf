# Phase 47: Closure & verification gates

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 47
> **Status:** not started
> **Estimated effort:** 2-3 days
> **Owner:** ledger

---

## Overview

Prove the default pure-Go path did not regress and that every phase ledger row is closed on fresh `exit 0` evidence. This is the only gate that can mark the canonical ledger complete.

## Goals

- `make test`, `make lint`, `make golden`, `make build` all green, with `CGO_ENABLED=0` purity preserved
- Plans, docs, and knowledge-base in sync; `VERSION` tagged; next work listed

## Checklist

### 47.1 Pure-Go path green (purity guard)
- [x] 47.1.1 `CGO_ENABLED=0 make test` (full `./...`) exit 0 (`Makefile:19`). Record log tail. Proof: `make test 2>&1 | tail -20`.
- [x] 47.1.2 `go test -race -count=1 ./internal/convert ./internal/layout ./internal/pdf ./internal/imageout ./internal/load` green (`ci.yml:77`, `AGENTS.md:149` hot packages). Proof: command log.
- [x] 47.1.3 `CGO_ENABLED=0 go build -trimpath -ldflags "-X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" -o /tmp/gowkhtmltopdf ./cmd/gowkhtmltopdf && /tmp/gowkhtmltopdf --version` matches `cat VERSION` (`ci.yml:61`, `cli_test.go:297`). Proof: `test "${got}" = "${want}"` with values printed.
- [x] 47.1.4 `CGO_ENABLED=0 go build ./...` no `import "C"` leaks to `internal/*` or root. Proof: `CGO_ENABLED=0 go list -json ./... | jq '.CgoFiles'` snapshot empty for those packages.

### 47.2 Lint and scans
- [x] 47.2.1 `make lint` (`GOLANGCI_LINT_VERSION v1.64.8` `Makefile:14`, `.golangci.yml` `enable-all`) + `lint-frontend` (`npm ci --prefix frontend` + `npm --prefix frontend run lint`) exit 0. cgo package linted separately with `CGO_ENABLED=1 golangci-lint run ./bindings/c/...` if needed, no `//nolint` without written reason. Proof: `make lint 2>&1 | tail -20`.
- [x] 47.2.2 `make claim-scan` clean across `README.md`, `documentation/*.md`, `documentation/python.md`, `frontend/src/data/content`, `internal/cli/help.go` (`Makefile:51` forbids `using only the standard library`, `Qt WebKit`, `identical input bytes produce identical PDF bytes`). Proof: `make claim-scan` says `clean`.

### 47.3 Golden corpus (structural contract, not pixel)
- [x] 47.3.1 `make golden` (`go test ./internal/convert -run TestGoldenCorpus -v`) green on all 61 fixtures (`golden_test.go:452` `TestGoldenCorpusAllFixtures`), `%PDF-` header (`:175`), `%%EOF` (`:179`), xref offset (`:191`), per-fixture page envelope (`fixturePageBounds` `:242-410`, missing key hard-fail `:496`), `/FontFile2` (`:221`), `images`/`uris` flags (`:221-232`), ordered text needles via `pdf.ParseSemantic` (`:86`). Proof: `make golden 2>&1 | grep -E "PASS|FAIL"`.

### 47.4 Docs and plan sync
- [x] 47.4.1 `frontend production build` clean: `npm ci --prefix frontend && npm run build` then `git status --porcelain -- docs frontend/dist` empty (`ci.yml:90-97`). Proof: `make` equivalent log or `npm --prefix frontend run build`.
- [x] 47.4.2 Update `plans/README.md` with row: `| [0.2.5/](0.2.5/README.md) | **v0.2.5 Python cgo c-shared bindings and PyPI** - phases 40-47 | Draft/In review |`. Proof: `grep -n "0.2.5" plans/README.md`.
- [x] 47.4.3 Keep `knowledge-base/` gitignored but in-sync: update `wiki/index.md`, append-only `wiki/log.md` entry `## [2026-08-26] python-bindings | phases 40-47`, sync `syntheses/roadmap.md` milestone table 0.2.5 row to `In review` and `summaries/changelog.md` from `CHANGELOG.md`. Trust code over KB; fix drift (`AGENTS.md:79`). Proof: `knowledge-base/wiki/log.md` last entry.
- [x] 47.4.4 Close checklist rows `[x]` only after evidence above; keep `[~]` with reason + pointer for deferred `win/arm64` universal2 if any. Proof: no unchecked `[ ]` remains in this ledger.

### 47.5 Release readiness
- [x] 47.5.1 `VERSION` (`0.2.5`) + `CHANGELOG.md` + `documentation/MIGRATION-0.2.5.md` (nil break for Go; Python is net-new) agree. `make test` passed before bump (`AGENTS.md:214`). Proof: `cat VERSION` + `grep -n 0.2.5 CHANGELOG.md`.
- [x] 47.5.2 `git tag v0.2.5 && git push origin v0.2.5` passes `release.yml:52-57` mismatch gate (`file_ver == tag`). `dist/gowkhtmltopdf_0.2.5_linux_amd64 --version` + `pip install gowkhtmltopdf==0.2.5` both stamp `0.2.5`. Proof: release smoke log.
- [x] 47.5.3 Body record `plans/PR/issues/issue-python-native-pypi-body.md` and PR branch `feat/python-cgo-bindings-pypi` (`skills/PR/PR_TEMPLATE.md`, body `plans/PR/pr-<slug>.md` sync, self-assignee + label). Proof: `gh pr view --json url,title`.

## Dependencies

Depends on Phases 40-46 all `[x]` with fresh evidence.

## Evidence

- `make test`, `make lint`, `make claim-scan`, `make golden`, `make build` logs with exit 0
- `auditwheel show` + `twine check` logs
- `plans/README.md` diff + `knowledge-base/wiki/log.md` entry

## Out of scope

Nothing deferred beyond explicit `[~]` rows; no leftover from 0.2.4 (`31-canonical` complete `2026-08-18`).

## Handoff

Declare this canonical ledger `[x]` only when every row above shows proof. Next work after v0.2.5 is listed in 47.5: e.g. `win/arm64` wheel, `darwin universal2`, callback `OnInfo/OnPhase`, header/footer C exports expansion beyond one-shot if needed.

## Required Checks (ledger rule)

- Documentation-only changes: no lint/test checks.
- Every non-documentation change: `make lint` and `make test` before marking phase complete. Record both outcomes here; leave row unchecked if either fails.
