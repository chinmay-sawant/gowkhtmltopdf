# Critical Golang Architecture Review — 2026-08-12

> **Parent:** [`plans/reviews/improve-codebase/README.md`](../README.md) — new dated architecture ledger; distinct from the Ponytail leanness audit.
> **Status:** review complete; CR-01 through CR-08 remediated and validated on 2026-08-12.
> **Scope:** current Go application source, CLI/library contracts, layout/PDF/image pipeline, tests, benchmarks, linter, and race validation.

---

## Executive verdict

**8.8 / 10 — remediation restored the strict output/API contracts; remaining scale work is evidence-led.**

The repository has unusually good foundations for a renderer: a real pipeline seam, explicit context entry points, a narrow dependency policy, source ownership snapshots, a broad test suite, and passing race/lint gates. The initial review found two output-integrity failures and several contract gaps; this remediation wave now rejects the conflicting stdout mode, makes the island path explicit and parent-consistent, restores terminal/API validation, bounds style cancellation, stabilizes fallback selection, and documents PDF writer ownership.

The rating remains intentionally release-oriented. It rewards the new public-seam regressions and measured evidence while keeping global caching, aggregate resource limits, and broader worst-case layout work as roadmap items until their corpus measurements justify them.

| Dimension | Weight | Score | Weighted |
|---|---:|---:|---:|
| Output correctness and CLI contracts | 35% | 8.8 | 3.08 |
| Public API and validation boundaries | 20% | 8.8 | 1.76 |
| Architecture and maintainability | 20% | 8.6 | 1.72 |
| Cancellation and concurrency semantics | 10% | 8.4 | 0.84 |
| Verification and operational discipline | 15% | 9.4 | 1.41 |
| **Total** | **100%** |  | **8.81 → 8.8** |

## Review method and validation boundary

Five independent roles were used: API/error/config discovery; engine/memory/concurrency discovery; empirical validation; Go-idiom/package-boundary validation; and a devil's-advocate synthesis. Findings below are either **confirmed** with current source/runtime proof or explicitly labelled **risk to measure**.

| Check | Result |
|---|---|
| `GOCACHE=/tmp/gowk-go-cache go test ./...` | passed |
| `GOCACHE=/tmp/gowk-go-cache go test -race -count=1 ./...` | passed; no race report |
| `GOCACHE=/tmp/gowk-go-cache GOLANGCI_LINT_CACHE=/tmp/gowk-golangci-cache make lint` | passed |
| Stored Snapshot D code benchmark | 500-page PDF: 1.359 s/op; 171.5 MB/op; 966,532 allocs/op (one iteration; retained baseline) |
| Stored Snapshot D direct CLI process run | 500 pages: 960 ms; 54,632 KiB peak RSS; 1,385,450-byte PDF; Ghostscript page count verified (historical report fixture baseline) |
| Fresh Snapshot E remediation benchmarks | Repeated resources 1/10/500 pages: 13.26/25.36/802.70 ms; deep chrome 10/100/1,000 items: 9.19/1.93/24.06 ms; style cancellation 145.63 ms; cold glyph rows recorded in the benchmark ledger |

Green unit, race, and lint runs prove a useful baseline. They do **not** prove a CLI byte stream is valid, the specialized renderer is semantically equivalent, or cancellation is prompt inside CPU-heavy cascade work.

## What is good

- **The dependency direction is sound.** The root library avoids `internal/cli`; command adapters stay outside the engine; `convert.Request` and `render.Pipeline` give both output modes a recognizable lifecycle.
- **Ownership is taken seriously at the public boundary.** `Converter.AddObject`, typed requests, and `Output` clone mutable maps, slices, and output bytes. This is backed by mutation/race-oriented API tests.
- **Security defaults are concrete.** Local files default deny; the load layer applies ACLs, redirects, timeouts, and body caps through a shared resource boundary.
- **There is real verification discipline.** The repository uses an enable-all linter setup, normal and race test suites, golden corpus coverage, resource caps in the loader and image paths, and explicit conversion benchmarks.
- **Several performance directions are good.** Style interning, display-list workspace reuse, per-layout image cache, document-wide font subset union, and the normal deferred-chrome merge reduce obvious repeated work.

## Confirmed findings

The evidence snippets in this section record the pre-remediation state that
motivated each finding. The current source disposition and proof are listed in
the remediation table below; the original paths remain useful for audit
history, not as a claim that the defects are still open.

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

## Remediation disposition

The findings above are the baseline review record. The current source closes each
confirmed item as follows:

| Finding | Current disposition | Proof |
|---|---|---|
| CR-01 | CLI rejects `--dump-outline` with PDF stdout before opening either sink; explicit file/`io.Writer` library paths remain available. | CLI, app, and command regression tests |
| CR-02 | Ordinary requests use the generic renderer; the benchmark island path is explicit and clones a parent-consistent tree. | island unit tests, parity fixtures, race-focused conversion tests |
| CR-03 | `--dump-default-toc-xsl` is modeled as a terminal operation and works without an input document. | parser and command tests |
| CR-04 | Image background aliases normalize once and round-trip through `Get`, including effective pixel behavior. | API alias matrix and image tests |
| CR-05 | Typed PDF/image requests share root-owned structural validation and reject invalid input before writes or loader setup. | API/request validation matrix |
| CR-06 | Stylesheet collection, cascade, and container measurement poll cancellation at bounded work intervals. | cancellation-latency test and style benchmark |
| CR-07 | Fallback candidates are snapshotted and selected with deterministic face identity tie-breaking outside the registry lock. | equal-score permutation test |
| CR-08 | `pdf.Document` is explicitly single-owner; the partial mutex was removed and the ownership contract is documented. | full race suite and writer documentation |

The closure gates are recorded in the phase-wise checklist. The measured
resource, chrome, and glyph rows are baselines rather than blanket guarantees;
their remaining architectural decisions stay in the risk table below.

## Remaining scale risks and follow-ups

| ID | Risk, not confirmed defect | Required evidence before implementation |
|---|---|---|
| R-01 | Cross-object resource reuse and aggregate resource budgets | The focused 1/10/500 corpus is measured; a larger asset corpus with process RSS, freshness, eviction, and visual output is required before a global cache or aggregate cap. |
| R-02 | Chrome/pagination worst-case complexity | The 10/100/1,000 corpus and pprof baseline are measured; nested production-shaped flow/deferred-chrome workloads remain before an indexing redesign. |
| R-03 | Generic geometry regression watch | Fixture-ID/class/text predicates were removed and approved fixture checks pass; retain differential goldens as new fixtures are added. |
| R-04 | Glyph raster allocation follow-up | Cold 12/24/72 px Latin/CJK rows and exact raster tests are recorded; broader font/size profiles remain before another raster redesign. |

## Devil's-advocate assessment

- Do **not** add global caches, aggregate caps, or a full document concurrency redesign based only on static suspicion. Establish corpus hit rates, RSS, output-size, and latency first.
- Do **not** broaden the specialized island matcher. Its performance idea is reasonable only for an explicitly owned immutable workload with a parity oracle; generic HTML must remain on the full renderer.
- Do **not** confuse a zero-CGO design or clean `-race` run with output correctness. Keep subprocess/byte-stream and rendered-output regression tests at public seams.
- Do **not** rewrite every dotted setting at once. Preserve it as the wkhtmltopdf compatibility escape hatch while fixing public normalization and introducing typed options incrementally.

## 5-phase roadmap disposition

1. **P0 — Output integrity:** completed; conflicting outline/PDF stdout is rejected and ordinary documents remain on the generic renderer.
2. **P1 — Contract restoration:** completed; TOC-XSL is terminal, image background aliases round-trip, and typed request validation has public errors.
3. **P2 — Lifecycle semantics:** completed; style cancellation is bounded, `pdf.Document` is explicitly single-owner, and fallback selection is stable.
4. **P3 — Measured scale work:** baselined; repeat-resource, deep-chrome/forced-break, and cold-glyph corpora are recorded without promoting unmeasured global caches or caps.
5. **P4 — Closure:** completed; targeted regressions, `make lint`, `make test`, race validation, and narrow rendered checks are recorded in the phase-wise checklist. Future work is limited to larger production-shaped corpora and any decisions they justify.

## Subagent validation matrix

| Audit role | Confirmed contribution | Validation disposition |
|---|---|---|
| API & ergonomics discovery | stdout mixing, terminal CLI gap, typed image setting contract, typed PDF preflight | retained; image/PDF scope narrowed to proven cases |
| Engine & memory discovery | page-island trust/topology, cascade cancellation, fallback determinism, partial document mutex | retained; broad resource/pagination claims moved to risks pending benchmarks |
| Empirical validation | runtime proof of stdout mixing and unreachable TOC-XSL; full race validation | retained; no new race/panic/cap bypass asserted |
| Go idioms validation | parent-link dependency proof, single-owner concern, deterministic registry proof | retained; package DAG judged healthy |
| Devil's advocate | release-oriented weighted score and P0/P1 separation | adopted; avoids speculative rewrites |
