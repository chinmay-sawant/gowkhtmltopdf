# Phase 39 - External Compare Benchmarks

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
> **Estimated effort:** 3–5 days (harness already largely present in the worktree; this phase owns path freeze, docs, CLI-flag follow-up, and checked-in artifacts)
> **Depends on:** none to start the harness; **Phase 36** before final committed compare tables (new CLI flags)
> **Unblocks:** Phase 38 closure (benchmark path honesty in README / performance docs)

---

## Overview

v0.2.4 owns the **process-level external compare** surface: gowkhtmltopdf
CLI vs three engines on the same generated report fixture.

| Engine | How it is invoked | Artifact paths |
|--------|-------------------|----------------|
| **wkhtmltopdf** | `make bench` → `make bench-cli-compare` → `TestCompareWithWkhtmltopdfBinary` | `testdata/golden/benchmarks/cli-compare.md`, `cli-compare-results.csv` |
| **WeasyPrint** | `make bench` → `scripts/bench-external.sh` → `scripts/weasyprint/print.sh` | `testdata/golden/benchmarks/weasyprint-compare.md`, `weasyprint-compare-results.csv` |
| **Puppeteer / Chrome** | `make bench` → `scripts/bench-external.sh` → `scripts/puppeteer/print.sh` → `scripts/puppeteer_print.js` | `testdata/golden/benchmarks/puppeteer-compare.md`, `puppeteer-compare-results.csv` |

Internal engine allocation benches use `make bench-engine`; the compatibility
alias `make bench-inprocess` remains available. Public library allocation
benches use `make bench-lib` and call `Document.WritePDF` /
`ImageDocument.WriteImage` directly. All three matrices remain documented
under the same `testdata/golden/benchmarks/README.md`.

## Executive Summary

| Path | Role |
|------|------|
| `scripts/bench-external.sh` | Orchestrator: generate report fixture, warmup + N timed runs, write WeasyPrint/Puppeteer MD+CSV |
| `scripts/weasyprint/print.sh` | Measured WeasyPrint command (`weasyprint -q in.html out.pdf`) |
| `scripts/puppeteer/print.sh` | Measured Puppeteer entry (`node …/puppeteer_print.js`) |
| `scripts/puppeteer_print.js` | headless Chrome PDF via `puppeteer-core` |
| `scripts/puppeteer/package.json` (+ lock) | `puppeteer-core` pin; install with `npm ci --prefix scripts/puppeteer` |
| `Makefile` `bench` | `build`, external WeasyPrint/Puppeteer compare, then `bench-cli-compare` |
| `Makefile` `bench-cli-compare` | Standalone wkhtmltopdf process compare (also invoked by `bench`) |
| `Makefile` `bench-engine` | Internal `convert` `-bench` allocation matrix |
| `Makefile` `bench-lib` | Public `Document` / `ImageDocument` `-bench` allocation matrix |
| `testdata/golden/benchmarks/README.md` | Operator-facing methodology + tables for all three engines |

Default external matrix in `bench-external.sh`: pages **2 / 10 / 50 / 100**,
**3** timed runs after one warmup. Override with `--sizes` / `--runs` /
`--engines=weasyprint,puppeteer`.

---

## Phase 39 checklist

### 39.1 Path freeze (canonical layout)

- [x] Document the three-engine path table (above) in `testdata/golden/benchmarks/README.md`
- [x] Confirm `make bench` runs `scripts/bench-external.sh` (WeasyPrint + Puppeteer), then invokes `bench-cli-compare` for wkhtmltopdf
- [x] Confirm `make bench-cli-compare` remains the dedicated standalone target for the wkhtmltopdf process compare
- [x] Confirm artifact names stay `{cli,weasyprint,puppeteer}-compare.md` and `-compare-results.csv` under `testdata/golden/benchmarks/`
- [x] `.gitignore`: ignore `scripts/puppeteer/node_modules/` (and any temp HTML/PDF the harness writes); **do not** ignore the committed MD/CSV result snapshots unless product decides otherwise
- [x] Do not commit `scripts/puppeteer/node_modules/` contents

### 39.2 Harness correctness

- [x] `scripts/bench-external.sh`: engines that are not installed are skipped with a clear message; zero engines → non-zero exit
- [x] Measured command for WeasyPrint is exactly `scripts/weasyprint/print.sh` (no second weasyprint invocation path in the orchestrator)
- [x] Measured command for Puppeteer is exactly `scripts/puppeteer/print.sh`
- [x] Fixture source is `testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per page) — same family as cli-compare
- [x] Ghostscript page-count verification when `gs` is present; honest `0` / warning when absent
- [x] Puppeteer RSS methodology (process-tree sampling) documented next to WeasyPrint/`%M` so tables are not misread as identical RSS definitions
- [x] `PUPPETEER_EXECUTABLE_PATH` override documented (default `/usr/bin/google-chrome`)

### 39.3 Makefile and operator commands

- [x] `make bench` / `make bench-cli-compare` / `make bench-engine` / `make bench-lib` comments match the path freeze
- [x] README reproduce one-liners:
  - `make bench-cli-compare`
  - `make bench` or `./scripts/bench-external.sh --engines=weasyprint`
  - `./scripts/bench-external.sh --engines=puppeteer`
  - `make bench-engine`
  - `make bench-lib`
  - `npm ci --prefix scripts/puppeteer` prerequisite for Puppeteer
- [x] Optional extended sizes (`--sizes=2,5,10,20,50,100,200,250,500`) documented as opt-in, not default for WeasyPrint/Puppeteer (default stays 2/10/50/100)

### 39.4 CLI redesign follow-up (after Phase 36)

- [x] Update gowkhtmltopdf argv in `bench-external.sh` and `internal/convert` cli-compare test to the **new** CLI flags (`-o`, `--allow-local-files`, …) — no leftover `--enable-local-file-access` / wkhtml object grammar in the measured gowk command
- [x] Re-run and refresh committed `cli-compare*`, `weasyprint-compare*`, `puppeteer-compare*` on the reference host after the flag change
- [x] Record host / toolchain / engine versions in the MD headers (already patterned in current artifacts)
- [x] Public `bench-lib` PDF uses the exact `report.html.tmpl` fixture, 20-row page data, and physical page-count contract used by the external CLI comparisons

### 39.5 Docs and product honesty

- [x] `testdata/golden/benchmarks/README.md`: sections for wkhtmltopdf, WeasyPrint, Puppeteer, the internal engine matrix, and the public library matrix with methodology + “what this is not”
- [x] `documentation/performance.md` (if present) points at the three artifact pairs and the benchmark target families
- [x] Frontend benchmarks page / site data updated only if it still claims a single external compare engine
- [x] Explicit: external benches are **operator / release evidence**, not a default `make test` CI gate (unless a later amendment adds an opt-in job)

### 39.6 Proof

- [x] `make build && ./scripts/bench-external.sh --engines=weasyprint --sizes=2 --runs=1` succeeds when WeasyPrint is installed (or skips cleanly when not)
- [x] `./scripts/bench-external.sh --engines=puppeteer --sizes=2 --runs=1` succeeds when node + `npm ci` + Chrome are available (or skips cleanly)
- [x] `make bench-cli-compare` still produces `cli-compare*` when wkhtmltopdf is on PATH (or documented skip)
- [x] `make bench-lib` passes the public PDF/image benchmark matrix without starting a CLI
- [x] No generated `node_modules` remains under the workspace; `.gitignore` covers `scripts/puppeteer/node_modules/` (Git status was intentionally not run)

### 39.7 Closure gates

- [x] `make lint` → changed-surface lint passes; repository-wide result recorded in Phase 38
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Parent Phase 39 row checked
- [x] Next: Phase 38 (if 31–37 also done)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Existing report template + cli-compare methodology | Comparable fixtures |
| Phase 36 new CLI | Final measured gowk argv + refreshed tables |
| Optional host tools: `weasyprint`, `node`, Chrome, `wkhtmltopdf`, `gs`, GNU `time` | Runnable engines |

---

## Out of scope

- Changing layout/CSS to chase WeasyPrint or Chrome pixel parity
- Making Puppeteer/WeasyPrint a hard CI dependency on every PR
- Redesigning the internal `bench-engine` allocation matrix beyond its naming/ownership split
- 500-page one-second target work (stays under `plans/0.2.0/deferred/`)

## Validation record (2026-08-18)

- `make bench-engine` passed the generic, certified-islands, template, and image matrices. `make bench-lib` passed the public PDF and image matrices through `Document.WritePDF` and `ImageDocument.WriteImage` without starting a CLI; its PDF workload now uses the exact external `report.html.tmpl` fixture and page-count contract. `make bench-cli-compare` passed across 2/5/10/20/50/100/200/250/500 pages with the new gowk CLI flags.
- `make build && ./scripts/bench-external.sh --engines=weasyprint --sizes=2 --runs=1` passed; Puppeteer passed with Chrome 143 and `puppeteer-core 24.43.1` using the same 2-page smoke. Refreshed `cli-compare*`, `weasyprint-compare*`, and `puppeteer-compare*` artifacts include host/tool notes and methodology.
- `scripts/puppeteer/node_modules/` was used for the smoke and is ignored/generated; it is removed before handoff. The external script does not invoke wkhtmltopdf; `make bench` invokes the dedicated comparison target after the script completes.
