# Critical Golang Architecture Review — 2026-08-12

> **Parent:** [`plans/reviews/improve-codebase/README.md`](../README.md) — new dated architecture ledger; distinct from the Ponytail leanness audit.
> **Status:** review complete; remediation not started.
> **Scope:** current Go application source, CLI/library contracts, layout/PDF/image pipeline, tests, benchmarks, linter, and race validation.

---

## Executive verdict

**5.7 / 10 — solid internal direction, but not release-ready for strict output correctness.**

The repository has unusually good foundations for a renderer: a real pipeline seam, explicit context entry points, a narrow dependency policy, source ownership snapshots, a broad test suite, and passing race/lint gates. Those strengths do not compensate for two output-integrity failures: `--dump-outline` can prepend XML to a PDF written to stdout, and a user-controlled HTML shape can select a specialized page-island renderer whose virtual DOM has invalid parent links.

The rating is intentionally a release-readiness score for a document converter. Producing a corrupt stdout stream or silently taking a semantically different renderer path has more weight than clean style checks.

| Dimension | Weight | Score | Weighted |
|---|---:|---:|---:|
| Output correctness and CLI contracts | 35% | 4.0 | 1.40 |
| Public API and validation boundaries | 20% | 5.5 | 1.10 |
| Architecture and maintainability | 20% | 6.5 | 1.30 |
| Cancellation and concurrency semantics | 10% | 5.0 | 0.50 |
| Verification and operational discipline | 15% | 9.0 | 1.35 |
| **Total** | **100%** |  | **5.65 → 5.7** |

## Review method and validation boundary

Five independent roles were used: API/error/config discovery; engine/memory/concurrency discovery; empirical validation; Go-idiom/package-boundary validation; and a devil's-advocate synthesis. Findings below are either **confirmed** with current source/runtime proof or explicitly labelled **risk to measure**.

| Check | Result |
|---|---|
| `GOCACHE=/tmp/gowk-go-cache go test ./...` | passed |
| `GOCACHE=/tmp/gowk-go-cache go test -race -count=1 ./...` | passed; no race report |
| `GOCACHE=/tmp/gowk-go-cache GOLANGCI_LINT_CACHE=/tmp/gowk-golangci-cache make lint` | passed |
| Fresh Snapshot D code benchmark | 500-page PDF: 1.359 s/op; 171.5 MB/op; 966,532 allocs/op (one iteration) |
| Fresh direct CLI process run | 500 pages: 960 ms; 54,632 KiB peak RSS; 1,385,450-byte PDF; Ghostscript page count verified |

Green unit, race, and lint runs prove a useful baseline. They do **not** prove a CLI byte stream is valid, the specialized renderer is semantically equivalent, or cancellation is prompt inside CPU-heavy cascade work.

## What is good

- **The dependency direction is sound.** The root library avoids `internal/cli`; command adapters stay outside the engine; `convert.Request` and `render.Pipeline` give both output modes a recognizable lifecycle.
- **Ownership is taken seriously at the public boundary.** `Converter.AddObject`, typed requests, and `Output` clone mutable maps, slices, and output bytes. This is backed by mutation/race-oriented API tests.
- **Security defaults are concrete.** Local files default deny; the load layer applies ACLs, redirects, timeouts, and body caps through a shared resource boundary.
- **There is real verification discipline.** The repository uses an enable-all linter setup, normal and race test suites, golden corpus coverage, resource caps in the loader and image paths, and explicit conversion benchmarks.
- **Several performance directions are good.** Style interning, display-list workspace reuse, per-layout image cache, document-wide font subset union, and the normal deferred-chrome merge reduce obvious repeated work.

## Confirmed findings

### CR-01 — HIGH: `--dump-outline` can corrupt a PDF sent to stdout

**Evidence:** [`cmd/gowkhtmltopdf/main.go:70`](../../../../cmd/gowkhtmltopdf/main.go:70) assigns `os.Stdout` as the outline sink whenever dumping is requested; [`internal/cli/cli.go:65`](../../../../internal/cli/cli.go:65) also returns stdout for output `-`. [`internal/convert/pdf_pipeline.go:100`](../../../../internal/convert/pdf_pipeline.go:100) writes outline XML during assembly, before [`internal/convert/pdf_pipeline.go:177`](../../../../internal/convert/pdf_pipeline.go:177) writes PDF bytes in finalization.

Runtime proof: `go run ./cmd/gowkhtmltopdf --outline --dump-outline '<inline html>' -` emitted `<?xml` at byte zero rather than `%PDF-`; the command exited successfully.

**Impact:** a caller receiving stdout does not receive a valid PDF. This breaks pipelines, HTTP handlers, and shell redirection.

**Fix:** reject `--dump-outline` together with PDF output `-` before any sink opens, until a distinct outline path/writer is modeled. Do not attempt generic `io.Writer` identity checks; enforce this at the CLI/application boundary.

**Regression proof:** black-box command test asserts nonzero exit and no mixed XML/PDF output; positive tests cover PDF file output plus outline stdout and independently supplied library writers.

### CR-02 — HIGH: the “certified” page-island path is input-spoofable and uses an invalid virtual DOM

**Evidence:** [`internal/convert/islands/plan.go:27`](../../../../internal/convert/islands/plan.go:27) qualifies any input having a public comment marker, title `Benchmark report`, and `section.benchmark-page` children. [`internal/convert/convert.go:496`](../../../../internal/convert/convert.go:496) automatically chooses the specialized path for every such document.

The virtual tree built at [`internal/convert/islands/plan.go:114`](../../../../internal/convert/islands/plan.go:114) sets `copyBody.Children` to a cloned section but creates that section with `Parent: nil` and retains descendants whose parent pointers still refer to the original section ([`plan.go:117`](../../../../internal/convert/islands/plan.go:117)-[`120`](../../../../internal/convert/islands/plan.go:120)). CSS selector and layout code use `Parent`, so ancestor/child/sibling selectors, `:has`, positional selectors, and container lookup are not sound in this path.

The fast path also discards body children and forces `debug.FreeOSMemory` during user conversion ([`internal/convert/page_islands.go:55`](../../../../internal/convert/page_islands.go:55)-[`80`](../../../../internal/convert/page_islands.go:80)); a section that expands to two pages returns an error rather than using the generic renderer ([`page_islands.go:127`](../../../../internal/convert/page_islands.go:127)-[`129`](../../../../internal/convert/page_islands.go:129)).

**Impact:** markup content, rather than an explicit trusted request mode, chooses a different renderer with potentially different output or an avoidable hard failure. This is output-integrity debt, not a claimed security exploit.

**Fix:** disable the optimization for ordinary conversions now. Reintroduce it only as a private, explicit benchmark fixture/request mode with a structural eligibility oracle and a fully parent-consistent isolated tree. Generic documents must fail closed to the full renderer.

**Regression proof:** spoof-marker input uses generic rendering; a two-page matching section uses generic rendering rather than failing; full-vs-specialized parity covers parent/child, sibling, `:nth-*`, `:has`, links, headings, and container queries.

### CR-03 — MEDIUM: standalone `--dump-default-toc-xsl` is unreachable

**Evidence:** [`cmd/gowkhtmltopdf/main.go:46`](../../../../cmd/gowkhtmltopdf/main.go:46) implements the output only after parsing succeeds. [`internal/cli/cli.go:188`](../../../../internal/cli/cli.go:188) always resolves positionals, and [`internal/cli/cli.go:382`](../../../../internal/cli/cli.go:382)-[`395`](../../../../internal/cli/cli.go:395) rejects an invocation with no input. `go run ./cmd/gowkhtmltopdf --dump-default-toc-xsl` exits 1 with “you need to specify at least one input file”.

**Fix and proof:** model terminal non-conversion actions in the parser (like help/version), bypassing only positional validation for a true dump. Test the exact standalone invocation returns XSL/XML and exit 0, while malformed/conflicting flags remain errors.

### CR-04 — MEDIUM: typed image background settings accept aliases but do not reliably retain or read them

**Evidence:** [`internal/settings/reflect.go:874`](../../../../internal/settings/reflect.go:874)-[`880`](../../../../internal/settings/reflect.go:880) normalizes `background` aliases. [`api.go:772`](../../../../api.go:772)-[`787`](../../../../api.go:787) calls that normalizer but checks the *raw* supplied string before setting `backgroundSet`; e.g. `Set("WEB.BACKGROUND", "false")` succeeds but loses the override. [`api.go:790`](../../../../api.go:790)-[`797`](../../../../api.go:797) delegates `Get` to `ImageGlobal`, which does not own that PDF-global paint setting, so accepted `background` keys do not round trip.

**Fix and proof:** normalize the semantic key once at the public boundary; route both `Set` and `Get` aliases to explicit shared state. Test lowercase, uppercase, dotted, whitespace, true/false variants and effective output pixels.

### CR-05 — MEDIUM: typed PDF requests do not preflight “has renderable input”

**Evidence:** [`api.go:874`](../../../../api.go:874)-[`887`](../../../../api.go:887) sends `PDFRequest` directly to the engine. [`internal/convert/convert.go:137`](../../../../internal/convert/convert.go:137)-[`167`](../../../../internal/convert/convert.go:167) validates sinks and limits but not a page or inline-body object. It can create loader/font/document state before final serialization rejects an empty document ([`internal/pdf/pdf.go:502`](../../../../internal/pdf/pdf.go:502)-[`509`](../../../../internal/pdf/pdf.go:509)). The legacy converter and application adapter already preflight this invariant.

**Fix and proof:** centralize renderable-object validation in `ValidatePDF`, with root-package matchable errors. Test no object, TOC-only, empty object, inline body, output, and outline sink cases; invalid requests must perform zero output writes. Image input is already validated before loader/font setup and is not claimed broken here.

### CR-06 — MEDIUM: cancellation is delayed through stylesheet collection and cascade hot paths

**Evidence:** [`doc.go:61`](../../../../doc.go:61)-[`64`](../../../../doc.go:64) promises fully context-aware conversion. `CollectSheets` walks the DOM without checks between nodes ([`internal/convert/prepare/styles.go:57`](../../../../internal/convert/prepare/styles.go:57)-[`61`](../../../../internal/convert/prepare/styles.go:61)); [`internal/layout/layout.go:775`](../../../../internal/layout/layout.go:775) calls style resolution without context. The cascade then walks each element and all rules/selectors without bounded cancellation checks ([`internal/layout/style.go:358`](../../../../internal/layout/style.go:358)-[`398`](../../../../internal/layout/style.go:398), [`internal/layout/style_cascade.go:215`](../../../../internal/layout/style_cascade.go:215)-[`268`](../../../../internal/layout/style_cascade.go:268)).

**Fix and proof:** thread `context.Context` or a lightweight periodic checker through collection, cascade, and container passes; add a generated DOM/CSS cancellation-latency test and a non-cancelled benchmark.

### CR-07 — MEDIUM: ultimate glyph fallback is nondeterministic for equal-score fonts

**Evidence:** [`internal/pdf/registry.go:139`](../../../../internal/pdf/registry.go:139)-[`172`](../../../../internal/pdf/registry.go:172) iterates a Go map and keeps the first equal-score result. Equal compatible fallback fonts can therefore vary by map iteration, even though normal CSS family lookup is ordered.

**Fix and proof:** maintain deterministic registration order (or a deterministic snapshot) and tie-break explicit face identity. Register equal-score candidates in different orders and assert the same face and rendered metrics.

### CR-08 — MEDIUM: `pdf.Document` has a partial mutex, not a coherent concurrency contract

**Evidence:** [`internal/pdf/pdf.go:113`](../../../../internal/pdf/pdf.go:113) declares `sync.RWMutex`, but only object allocation/setting and page append lock ([`161`](../../../../internal/pdf/pdf.go:161)-[`195`](../../../../internal/pdf/pdf.go:195), [`221`](../../../../internal/pdf/pdf.go:221)-[`231`](../../../../internal/pdf/pdf.go:231)). Public mutators/readers, page navigation, page annotations, font-rune maps, ordering, and finalization do not consistently lock; [`internal/pdf/content.go:613`](../../../../internal/pdf/content.go:613)-[`628`](../../../../internal/pdf/content.go:628) mutates document maps without synchronization.

**Fix and proof:** document `Document` as single-owner/single-goroutine and remove/replace partial synchronization, or design a full lifecycle state machine. Do not promise parallel document assembly until page/content/font resource ownership is redesigned. This is not an observed production race: the current conversion path is sequential and the race suite passed.

## Risks to measure before changing architecture

| ID | Risk, not confirmed defect | Required evidence before implementation |
|---|---|---|
| R-01 | Cross-object resource reuse and aggregate resource budgets | 1/10/500-page repeated-resource corpus: wall time, RSS, fetched bytes, cache hit rate, PDF bytes, and visual output. |
| R-02 | Chrome/pagination worst-case complexity | 10/100/1,000 nested transform/sticky/forced-break benchmark plus pprof and golden parity. |
| R-03 | Fixture-named/text-named rendering fixes | A generic geometry rule must reproduce all approved visual fixtures before extraction. |
| R-04 | Glyph raster scanline allocations | Cold 12/24/72 px Latin/CJK benchmark, CPU/allocation profile, and exact raster golden. |

## Devil's-advocate assessment

- Do **not** add global caches, aggregate caps, or a full document concurrency redesign based only on static suspicion. Establish corpus hit rates, RSS, output-size, and latency first.
- Do **not** broaden the specialized island matcher. Its performance idea is reasonable only for an explicitly owned immutable workload with a parity oracle; generic HTML must remain on the full renderer.
- Do **not** confuse a zero-CGO design or clean `-race` run with output correctness. Keep subprocess/byte-stream and rendered-output regression tests at public seams.
- Do **not** rewrite every dotted setting at once. Preserve it as the wkhtmltopdf compatibility escape hatch while fixing public normalization and introducing typed options incrementally.

## 5-phase roadmap to 10/10

1. **P0 — Output integrity:** reject dump-outline/PDF-stdout multiplexing; remove or disable automatic islands for ordinary documents; add subprocess and parity tests.
2. **P1 — Contract restoration:** make TOC-XSL dump terminal; normalize and round-trip image background aliases; centralize typed PDF input/sink validation with public errors.
3. **P2 — Lifecycle semantics:** add bounded cascade cancellation; make `pdf.Document` explicitly single-owner (or deliberately concurrency-safe); make fallback selection stable.
4. **P3 — Measured scale work:** capture repeat-resource, deep-chrome, forced-break, and cold-glyph baselines before choosing cache/index/pool work.
5. **P4 — Closure:** run targeted regressions, `make lint`, `make test`, `go test -race ./...`, golden/rendered PDF comparison, and release-matrix checks; then re-score from new evidence.

## Subagent validation matrix

| Audit role | Confirmed contribution | Validation disposition |
|---|---|---|
| API & ergonomics discovery | stdout mixing, terminal CLI gap, typed image setting contract, typed PDF preflight | retained; image/PDF scope narrowed to proven cases |
| Engine & memory discovery | page-island trust/topology, cascade cancellation, fallback determinism, partial document mutex | retained; broad resource/pagination claims moved to risks pending benchmarks |
| Empirical validation | runtime proof of stdout mixing and unreachable TOC-XSL; full race validation | retained; no new race/panic/cap bypass asserted |
| Go idioms validation | parent-link dependency proof, single-owner concern, deterministic registry proof | retained; package DAG judged healthy |
| Devil's advocate | release-oriented weighted score and P0/P1 separation | adopted; avoids speculative rewrites |
