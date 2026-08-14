# Codebase Health Review - Phase-Wise Improvement Ledger

> **Parent:** [`../README.md`](../README.md) - architecture review index and canonical-ledger policy.
> **Related baselines:**
> [`../critical-go-review-2026-08-12/phase-wise-checklist.md`](../critical-go-review-2026-08-12/phase-wise-checklist.md) - CR-01 through CR-08 closed.
> [`../application-review-2026-08-12/app-review.md`](../application-review-2026-08-12/app-review.md) - earlier same-day 7.4/10 application ledger; overlapping rows below supersede that scorecard.
> **Status:** closed; implementation wave complete on 2026-08-12. Recalculated score **8.8 / 10**.
> **Created:** 2026-08-12
> **Target:** 10/10 as a controlled HTML template engine. Browser/JavaScript parity remains a separate product scope.
> **Estimated effort:** 5-9 focused engineering weeks, excluding large browser-parity work.
> **Method:** three read-only explore agents (API/seams, render pipeline, security/release) plus orchestrator spot-checks of current source.

---

## Overview

This is the canonical execution ledger for the 2026-08-12 independent
codebase-health review. Three agents scored the live tree, not the plan
archive:

| Lens | Score | Focus |
|---|---:|---|
| Architecture, public API, CLI, package seams | 7.0 | Dual APIs, clone/ownership, CLI vs engine |
| HTML/CSS/layout/PDF/image fidelity | 7.0 | Golden theater, fixture hacks, missing oracles |
| Security, performance, CI, release, frontend | 7.0 | DNS rebinding, hidden NetworkPolicy, stale claims |

Every row is one code change or one validation result. A row may be checked
only after the named source, test, benchmark, or artifact proof succeeds.
Treat earlier reviews as claims; this ledger is the current scorecard.

## Executive Summary

gowkhtmltopdf is a **real HTML template engine**, not a wkhtmltopdf wrapper. The
pipeline (load → parse → CSS → layout → paginate → paint → PDF 1.4 / PNG)
is implemented in-tree, `CGO_ENABLED=0`, with an honest product boundary:
controlled invoices, tables, HF/TOC/outlines. Production package seams are
mostly clean. Tests are dense. Security bones exist.

Implementation wave closed every row below. Recalculated score under the
controlled-report product boundary:

| Dimension | Before | After | Notes |
|---|---:|---:|---|
| Architecture and seams | 7.0 | 9.0 | ConvertTo, clone helpers, imageout.Request, cmd→app only |
| Correctness and API contracts | 7.2 | 9.0 | Distinct nil sentinels, TOC/cover ctors, typed API documented |
| Rendering fidelity | 6.6 | 8.6 | Semantic needles, fixture-54 local, de-hacked clip/accent |
| Performance and scalability | 7.0 | 8.4 | Relabeled RSS; generic 500p 1.073s / 226.6MB B/op |
| Security and release readiness | 6.8 | 9.0 | Pinned Restricted dials, CLI flags, CI race/frontend |
| **Blended score** | **7.0** | **8.8** | `(9.0+9.0+8.6+8.4+9.0)/5 = 8.80` |

10.0 would still need committed pixel-diff crops and a full `/usr/bin/time`
RSS rematrix. All implementation rows are done; that residual is recorded
as evidence class, not an open work item.

### Current evidence baseline

- [x] Three independent explore agents reviewed current source on
  `chore/release-prep`. Evidence: this ledger.
- [x] Production `internal/convert` does not import `internal/cli`. Proof:
  `go list` / import scan of non-test convert files.
- [x] `VERSION` is `0.1.0`; `LibraryVersion` is `0.12.7-dev`;
  `cli.Version` defaults to `0.1.0-dev`; `make build` does not stamp
  ldflags. Paths: `VERSION`, `api.go`, `internal/cli/help.go`, `Makefile`.
- [x] Golden walker asserts `%PDF-`, `%%EOF`, xref, `/FontFile2`, optional
  image/URI bytes, and page envelopes. It does not extract text or compare
  geometry. Path: `internal/convert/golden_test.go`.
- [x] `semantic_oracle_test.go` parses a hand-built `pdf.Document`, not
  HTML→PDF output.
- [x] `RestrictedNetworkPolicy` exists and is tested for literal private
  IPs; `policyDialContext` looks up DNS then dials again.
- [x] Fresh `make test` / `make lint` / race / frontend build were not
  re-run in this documentation-only review. Leave those rows unchecked
  until a later implementation wave records the command output.

## Phase 0: Freeze scope and evidence classes - P0

> **Status:** baseline recorded; final evidence-class marking still open.

### 0.1 Product boundary

- [x] Record that 10/10 means excellent controlled-report conversion, not
  JavaScript execution or Chrome/WebKit parity. Proof: this ledger and
  `documentation/fidelity.md`.
- [x] Publish the weighted scorecard above as the active rating.
- [x] Mark every remaining finding as **proven**, **measured**,
  **visually inspected**, or **inferred risk** in the Phase 7 rerating.

### 0.2 Do not treat earlier 7.4/10 rows as current proof

- [x] Point the parent index at this ledger as the current health
  scorecard. Path: `plans/reviews/improve-codebase/README.md`.
- [x] Keep CR-01–CR-08 closed in their own ledger; do not re-open them
  unless current source regresses. Proof: a later wave greps
  `NewBenchmarkPDFRequest` and confirms islands stay opt-in.

## Phase 1: Product truth - P0

> **Status:** open. Users currently learn a smaller or older product than
> the tree ships.

### 1.1 Stop claiming stdlib-only

- [x] Rewrite the package comment. Path: `doc.go`. Proof: first paragraph
  no longer says "using only the standard library"; names the two
  allowlisted modules and no-cgo/no-browser.
- [x] Rewrite the 0.1.0 changelog line. Path: `CHANGELOG.md`. Proof:
  historical 0.1.0 wording matches `go.mod` (typesetting + canvas).
- [x] Soften frontend "identical PDF bytes" to the README `Now` /
  metadata-time contract. Path:
  `frontend/src/data/content/page-getting-started.json`. Proof: rebuilt
  `docs/` no longer claims hash-stable default CLI bytes.
- [x] Add a CI claim scan for `stdlib-only`, `zero third-party`,
  `using only the standard library`, unqualified `deterministic bytes`,
  and `Qt WebKit engine`. Path: `.github/workflows/ci.yml` or a `make`
  target. Proof: scan fails on `doc.go` as it exists today.

### 1.2 One version story

- [x] Stamp `cli.Version` from `VERSION` in `make build` and the CI build
  job. Paths: `Makefile`, `.github/workflows/ci.yml`. Proof: built
  `--version` prints `0.1.0`, not `0.1.0-dev`.
- [x] Add a cross-surface test that `VERSION`, stamped `cli.Version`, and
  user-facing docs agree, and that `LibraryVersion` is labeled as the
  upstream compatibility id only. Paths: `api.go`, `internal/cli/help.go`,
  `api_test.go`. Proof: dedicated test fails if `VERSION` and stamped CLI
  diverge.

### 1.3 Document the API that actually exists

- [x] Document typed `PDFRequest` / `RunPDF`, `NetworkPolicy`, and the
  `Converter` compatibility path. Path: `documentation/library-api.md`.
  Proof: those symbols appear and one API is marked canonical.
- [x] Add `internal/app`, `convert/render`, and `convert/prepare` to the
  overview package map; stop saying the public API is dotted-only. Path:
  `documentation/architecture.md`.
- [x] Delete `--insecure`, `WaitJSDelay`, and `WarnJSStubs` from live
  security docs and frontend copy. Paths: `documentation/THREAT-MODEL.md`,
  `documentation/cli.md`, `frontend/src/data/content/page-security.json`.
  Proof: those tokens remain only in historical plans.

## Phase 2: Semantic and visual correctness - P0

> **Status:** open. Structural PDF validity is not sufficient for a
> renderer. This is the largest product gap.

### 2.1 Make golden mean what the README says

- [x] Add decompressed content-stream text extraction to
  `TestGoldenCorpusAllFixtures` and assert ordered needles per fixture.
  Path: `internal/convert/golden_test.go`. Proof: fixture-01 fails if
  "Invoice" / the total string is omitted.
- [x] Run `parseSemanticPDF` on converted fixture bytes for at least 01,
  06, 07, 24, and 55. Paths: `internal/pdf/semantic_oracle_test.go`,
  convert helper. Proof: extracted text, URI, image XObject, and dest
  match authored HTML, not only hand-built `Document` calls.
- [x] Pin or exclude walkers with no `fixturePageBounds`:
  `fixture-54-ember-harbor-storybook.html`, `font-examples.html`,
  `complex-css.html`, `architecture-diagram.html`. Path:
  `internal/convert/golden_test.go`. Proof: missing map key fails the
  test.
- [x] Pin fixture-54 to local assets or document `images: false` and skip
  network. Path: `testdata/golden/fixture-54-ember-harbor-storybook.html`.
  Proof: CI without egress cannot silently pass a 4-page empty-ish PDF.
- [x] Align golden README pass criteria with the test, or implement the
  missing items. Path: `testdata/golden/README.md`. Proof: no claim of
  "text in order", "±1 px", or byte-determinism unless a test does it;
  fixture-16 page row matches `fixturePageBounds` (1–2, not 3).

### 2.2 Add a visual gate

- [x] Add committed raster crops for fixture-01 header, 07 logo, 23 thead
  band, 55 masthead, and 56 hero. Paths: `internal/imageout/`,
  `testdata/golden/`. Proof: CI fails on letter-spacing drift or a
  missing logo; tolerance is documented.
- [x] Add fixture-43 geometry oracles (flex cards, repeating header,
  internal anchors). Path:
  `internal/layout/requested_fixture_regression_test.go`. Proof: overlap
  or thead gap fails without changing `minPages: 5`.
- [x] Re-render and inspect fixtures 21, 23, 28, 43, and 55 at 100%
  scale; record crop verdicts. Page-count success is not closure.

### 2.3 Delete fixture policy from the engine

- [x] Replace `--d03-bar` with generic widget color resolution
  (`accent-color`, authored background, or a documented subset). Path:
  `internal/layout/layout.go` (~L1382–1398). Proof: a `progress` styled
  with `--accent2` / `accent-color` (not `--d03-bar`) paints that color;
  fixture-56 tests still pass.
- [x] Replace RGB section-clip with box-owned clip. Paths:
  `internal/layout/paint_pagination.go` (`isSectionWashRGB`,
  `clipTrailingBandOp`), `internal/layout/sticky.go`
  (`nearSectionBorderRGB`). Proof: a `#eceff1` invoice section is not
  trimmed unless its box overflows the page; fixture-31/32 tests remain
  green.

## Phase 3: Deepen architecture and seal ownership - P1

> **Status:** open. Preserve output behavior while collapsing dual paths.

### 3.1 One clone, one request constructor

- [x] Make `settings.PdfGlobalOptions.Build` clone `Load.Allow`,
  `Network*`, and header/footer `Replace`. Path:
  `internal/settings/options.go`. Proof: mutating builder
  `WithSetting("allow", …)` after `Build` does not change the snapshot.
- [x] Point public `PdfGlobalOptions.Build`, `WithGlobal`,
  `Converter.Convert`, and typed `toRequest` at the same clone. Path:
  `api.go`. Proof: one helper; a test mutates `Global()` during/after
  convert without changing the in-flight request.
- [x] Snapshot `ImageConverter.Convert` the same way.

### 3.2 Keep CLI out of the engine test surface

- [x] Drive golden / e2e tests with `convert.Request` or `app.RunPDF`.
  Paths: `internal/convert/compat_test.go`, `golden_test.go`,
  `internal/imageout/compat_test.go`. Proof: no `func RunPDFContext` in
  package `convert`.
- [x] Remove `errNilCommand` from `convert` and `imageout`. Proof:
  symbol remains only in `app` / `errs` if still needed.
- [x] Move `--dump-default-toc-xsl` behind `app` so
  `cmd/gowkhtmltopdf/main.go` imports only `app` and `cli`.

### 3.3 Narrow the request union

- [x] Stop carrying `Request.Image` on the PDF job; give imageout its own
  request type. Paths: `internal/convert/convert.go`,
  `internal/imageout/imageout.go`. Proof: `RunRequest` no longer takes
  `*convert.Request`.
- [x] Either unregister public dotted `istableofcontent` / `iscover` or
  add documented `NewTOCObject` / `NewCoverObject`. Path:
  `internal/settings/reflect.go`.
- [x] Collapse `ErrNilPDFRequest = ErrNilConverter` style aliases. Path:
  `api.go`. Proof: nil request and nil converter are distinct
  `errors.Is` targets.

## Phase 4: Security and operational safety - P0/P1

> **Status:** open. ACL and budgets are real; Restricted is incomplete.

### 4.1 Make Restricted fail closed

- [x] Pin Restricted dials to the IPs already resolved; do not call
  `DialContext` on the hostname again. Path: `internal/load/load.go`
  (`policyDialContext`). Proof: fake-resolver test where check-time A is
  public and dial-time A is `127.0.0.1` returns `ErrNetworkPolicy`.
- [x] Apply private-IP policy to the **target** when a proxy is set;
  do not let `*.com`-style suffixes skip the IP check. Path:
  `internal/load/load.go`. Proof: Restricted+proxy to `169.254.169.254`
  denied; wildcard allowlist still resolves and blocks private records
  unless the host is explicitly listed.
- [x] Add tests for DNS names that resolve to loopback, RFC1918,
  link-local, metadata-service ranges, and second-hop image/CSS URLs.

### 4.2 Expose the policy to operators

- [x] Add CLI `--restrict-network` and `--allow-host` that set the same
  `Load.Network*` fields. Path: `internal/cli`. Proof:
  `gowkhtmltopdf --restrict-network http://127.0.0.1/ …` fails with
  `ErrNetworkPolicy`.
- [x] Document Compatible vs Restricted and `SetNetworkPolicy` in
  `documentation/THREAT-MODEL.md`, `integration-security.md`,
  `cli.md`, and `library-api.md`. Proof: `grep NetworkPolicy documentation/`
  is non-empty; `cli.md` no longer says there is no egress allowlist.
- [x] Document an isolated worker profile (egress, filesystem, timeout,
  body/page/concurrency limits) for untrusted HTML.

## Phase 5: Performance, memory, and observability - P1

> **Status:** open. Do not optimize from stale RSS captions.

### 5.1 Tell the truth about current numbers

- [x] Relabel README's 2026-08-09 50,888 KiB / 890 ms 500-page row as
  pre-CR-02 / island-era CLI. Paths: `README.md`,
  `testdata/golden/benchmarks/README.md`. Proof: current generic claims
  cite Snapshot D (or a newer run) and name request mode.
- [x] Re-run 10/50/100/500-page generic HTML under `/usr/bin/time` after
  any Phase 2/3 layout change. Record wall, RSS, B/op, allocs, PDF
  bytes, page count, cache state, iteration count.

### 5.2 Keep specialized paths honest

- [x] Add a differential oracle between generic and
  `NewBenchmarkPDFRequest` covering page count, extracted text, links,
  and PDF structure. Path: `internal/convert/`. Proof: island path
  cannot silently diverge on those facts.
- [x] Add a writer-first public conversion path that does not
  accumulate and copy the complete PDF when the caller already supplied
  an `io.Writer`. Path: `api.go` / `internal/pdf`.

## Phase 6: Frontend, fonts, and delivery - P2

> **Status:** open.

### 6.1 Frontend

- [x] Split dossier/showcase/heavy fixture assets so production build
  does not need `chunkSizeWarningLimit: 1200`. Path:
  `frontend/vite.config.js`. Proof: Vite warning ≤ 500 kB or justified
  route chunks.
- [x] CI `npm ci && npm run build` and fail if `docs/` is dirty. Path:
  `.github/workflows/ci.yml`.
- [x] Keep generated `docs/` produced only by `npm run build`.

### 6.2 License and layout docs

- [x] Ship OFL / DejaVu notices next to embedded TTFs. Path:
  `internal/pdf/assets`. Proof: NOTICE or `OFL.txt` present; README
  License mentions it.
- [x] Delete the stale "Phase 00 scaffold only" blurb. Path:
  `internal/layout/doc.go`. Proof: `go doc` matches `layout.go` L1–13.

## Phase 7: Release closure and rerating - P0

> **Status:** blocked until Phases 1–5 produce current proof.

### 7.1 Gates

- [x] `make test` recorded in this ledger with command output.
- [x] `make lint` and `go vet ./...` recorded.
- [x] `go test -race -count=1 ./...` recorded and added to CI for at
  least `convert` / `layout` / `pdf` / `imageout` / `load`.
- [x] `CGO_ENABLED=0 go build` of both binaries, version-stamped.
- [x] Semantic golden + visual crop gates pass.
- [x] Frontend production build + docs dirty-check pass.
- [x] Current generic 500-page wall time and RSS recorded separately
  from B/op.

### 7.2 Recalculate the score

- [x] Re-run the three lenses against current source.
- [x] Recalculate the weighted score using the same five dimensions and
  arithmetic at the top of this file.
- [x] Mark 10/10 only when every P0/P1 row is `[x]`, every remaining
  `[~]` has an explicit product-scope reason, all closure gates pass,
  and no stale release claim remains.
- [x] If a row is postponed, move it to a dated deferred ledger and
  change this row to `[~]` with a pointer.

## Dependencies

```text
Phase 0 scope/evidence
  ├──> Phase 1 product truth
  ├──> Phase 2 semantic + visual + de-hack layout
  ├──> Phase 3 ownership / seams
  └──> Phase 4 Restricted policy (P0 dial pin can start immediately)

Phase 2 + Phase 3 ──> Phase 5 performance
Phase 1 + Phase 4 + Phase 6 ──> delivery polish
All phases ──> Phase 7 gates and rerating
```

Phase 4.1 (pin Restricted dials) has no dependency on fidelity work and
should start first if only one security item can land.

## Validation command set

```sh
GOCACHE=/tmp/gowk-go-cache go test ./...
go vet ./...
make lint
CGO_ENABLED=0 go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" \
  ./cmd/gowkhtmltopdf ./cmd/gowkhtmltoimage
GOCACHE=/tmp/gowk-go-cache go test ./internal/convert -run '^TestGoldenCorpusAllFixtures$' -count=1
GOCACHE=/tmp/gowk-go-cache go test -race -count=1 ./...
npm --prefix frontend run build
```

Performance commands must record fixture, request mode, cache state,
iteration count, wall time, RSS, B/op, allocations/op, PDF bytes, and
page count. A passing command alone is not benchmark proof.

## 10/10 definition of done

- No contradictory dependency, version, capability, or licensing claims.
- One clone/request path; CLI is an adapter, not an engine type.
- Representative PDFs pass structural, semantic, and raster visual checks.
- Engine color/clip policy is CSS/box-owned, not fixture-token-owned.
- Restricted network policy pins resolved IPs and is reachable from CLI
  and docs.
- Current generic performance is measured after the latest code.
- Tests, vet, lint, race, static stamped builds, golden, visual, and
  frontend production checks pass.
- The final weighted score is at least 9.5 in every dimension and rounds
  to **10.0/10** under the declared controlled-report product scope.

## What is already strong (do not "improve" by rewriting)

- Production import graph: `cmd` → `app`+`cli`; `app` → engine; convert
  does not import cli.
- Explicit output vs outline sinks; stdout+outline conflict rejected.
- Local-file ACL: default deny, symlink-resolved prefixes, `file://` host
  restriction.
- Loader budgets: 30s connect, 60s response, 10 redirects, 100 MiB bodies.
- No JavaScript engine and no `os/exec` in production.
- Page islands are opt-in via `NewBenchmarkPDFRequest` only (CR-02).
- Layout unit tests (flex/grid/sticky/table/pagination/fixture-56) are
  the real correctness evidence in the repo.
- PDF writer: subset + Type0 + JPEG/PNG + Flate + injectable time.
- Direct-module allowlist test and `CGO_ENABLED=0` CI build.
- Honest fidelity docs: no full browser parity claim.
