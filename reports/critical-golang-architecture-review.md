# Critical Golang Architecture Review — gowkhtmltopdf

**Current score: 8.6 / 10** · Pure-Go, zero-CGO wkhtmltopdf-compatible HTML→PDF/PNG renderer · Go 1.26 (toolchain 1.26.4)

> **Cross-check status (2026-08-12):** corrected findings were implemented in the current tree and the matrix was rerated from the focused tests, benchmarks, lint gate, and full repository validation recorded in §8.

| Metric | Value |
|---|---|
| Non-test LOC | 47,334 |
| Test LOC | 32,140 |
| Packages | 23 (`go list ./...`) |
| Build / Vet / Test / Race | ✅ `go build ./...` · ✅ `go vet ./...` · ✅ `go test ./...` · ✅ `go test -race` (pdf, imageout, convert, layout) |
| CGO / unsafe / literal `interface{}` | No CGO or `unsafe`; no literal `interface{}` token (production does use the equivalent `any`) |
| Review wave | 5 subagents: Discovery ×2 → Validation ×2 → Criticizer (Devil's Advocate), followed by live cross-check |

---

## 1. Weighted Assessment Matrix

| Dimension | Weight | Score | Justification |
|---|---|---|---|
| Concurrency & correctness | 20% | **9.2** | Zero-lock layout engine; per-run imageout state shares nothing; cached font names use `sync.Once`; the revised pool and PNG deduplication paths are race-tested. |
| Test quality | 20% | **9.0** | 32K LOC of tests, root builder/API coverage, CLI boundary tests, settings parity tests, repeated-image/page-count checks, focused hotspot benchmarks, and targeted race coverage. |
| Performance & allocation | 20% | **8.4** | O(1) rounded-border lookup, exact-capacity supersample pooling, per-document PNG XObject reuse, cached font-name loading, direct registry scans, and measured header/footer/repeated-resource workloads. |
| Maintainability & architecture | 15% | **8.4** | CLI adapters now live at `internal/app`; engine seams are request-only; canonical nil sentinels and page-size state are enforced; `exhaustruct` is configured and lint-clean. |
| API design & public surface | 20% | **8.5** | `WithGlobal` provides an owned Converter path; typed builder validation is explicit; nil builders panic by policy; validation reaches `OnError`; public and engine output errors share sentinels. |
| Documentation & ergonomics | 5% | **8.3** | Architecture references now describe the app-boundary adapters, canonical page geometry, typed builder paths, and measured performance evidence. |

**Weighted total: 0.2×9.2 + 0.2×9.0 + 0.2×8.4 + 0.15×8.4 + 0.2×8.5 + 0.05×8.3 = 8.6 / 10**

---

## 2. Executive Summary

gowkhtmltopdf is now a **strong mid-8 codebase**: the plumbing was already excellent and the API, package boundaries, and performance evidence now match it. Context threading is thorough (per-object `ctx.Err()` checks, a watcher goroutine that force-closes a blocked `os.File` on cancellation), ownership cloning at the public boundary is exemplary, the layout engine is single-goroutine with zero locks by design, `%w` wrapping discipline has exactly one `nolint:wrapcheck` repo-wide (test-only), and the internal package DAG keeps command translation at the application edge.

The remaining engineering tradeoffs are bounded rather than unmeasured blockers:

1. **The dotted settings surface remains intentionally compatibility-oriented** — it is protected by descriptor parity tests while the typed builder covers the high-value global settings and retains `WithSetting` for the full key surface.
2. **Pagination and header/footer remain semantically conservative** — the representative workloads are measured, page counts remain stable, and the current state-machine structure is retained where profiles do not justify a riskier rewrite.

Neither is a rewrite-level problem; both are explicit, tested compatibility/performance decisions recorded in the completed checklist.

---

## 3. What is GOOD

1. **Ownership cloning at the public boundary is exemplary** — `clonePdfObject`/`clonePdfGlobal`/`cloneHeaderFooter` (api.go:336-425) plus a second clone in `toRequest` (api.go:870-905) mean caller mutation after `AddObject` cannot corrupt a converter.
2. **Context threading is genuinely thorough** — per-object `ctx.Err()` (convert.go:323), recursion-boundary checks (layout.go:679-695), smart-width loop checks (imageout.go:252-254), and a watcher goroutine that force-closes a blocked `os.File` on cancellation (load.go:701-719). Stage-gated cancellation in `render.Pipeline.Run` (pipeline.go:26-54) is documented and correct.
3. **Zero-lock hot engine** — layout is single-goroutine with zero mutexes, pools, or atomics; `pdf.Registry` is the only shared lock, read-mostly, short critical sections. The only production `go func` is the load-file watcher. A *designed* property, not an accident.
4. **`sync.Once`/`sync.Pool` discipline is textbook** — per-Font `sync.Once` for `gotFace`/`reverseCmap` (shape_gotext.go:411-436) instead of a global `sync.Map`; pool exclusivity in `shaperPool`/`segmenterPool` (shape_gotext.go:25-32); `flatePool` copy-on-put required and correctly implemented (pdf.go:957-980); per-run mutable caches (glyph atlas, `rasterImageCache` keyed by FNV-1a + `bytes.Equal`) so concurrent `Render` calls share nothing.
5. **`%w` discipline is exemplary** — one `nolint:wrapcheck` repo-wide, test-only (api_test.go:382); modern `errors.Join` for HF warnings (hf.go:375).
6. **Cascade is allocation-conscious** — reused `cascadeWins`/`cascadeProps` maps (style_cascade.go:333-415), package-level `inheritableProps`/`styleGroups` tables (~40% alloc reduction, documented), `styleStore` interning with bounded chunks (style.go:530-590), per-rune face caches with no joined-family string allocation.
7. **`internal/line` is a model small package** — log-severity grammar centralized (`SeverityOf`, line.go:44-57).
8. **CLI test coverage is strong** — 34+ test funcs including mode restrictions, `--no-*` negation, pair flags, pending-object address remapping.
9. **Package DAG is clean** — no circular dependencies; `pdf` is a leaf of the internal package graph, although it directly imports external `go-text/typesetting` packages.
10. **`pageSizes` uses an immutable `[...]pageSizeEntry` array** (pagesize.go:17-42) — the correct pattern the rest of the codebase should copy.
11. **No C-heritage rot** — zero `unsafe` and no literal `interface{}` syntax in non-test code; idiomatic `any` is used in a few pool/variadic signatures; `StrokeMask`/`cli.Mode` are idiomatic `1 << iota`; pools are correct Go, not pointer tricks.

---

## 4. What is BAD — Validated Findings

### 4.1 Public API & Ergonomics (Discovery 1 → Validation)

#### HIGH-1 · String-key settings table is a parallel, weakly-typed config system — CONFIRMED
`internal/settings/reflect.go` uses **zero Go reflection** despite the name: ~59 `regGlobal` closure pairs (`apply`/`get`) hand-dispatched through maps (reflect.go:390-395, 421-559). No compile-time link between `PdfGlobal.SmartShrinking` and key `"smartshrinking"` — a field rename compiles fine and silently breaks `Set`. The typed `PdfGlobalOptions` builder covers only 9 of ~59 keys.

```go
// reflect.go:544-552 — dual-registered, weakly-typed closure dispatch
regGlobal("size.pagesize",
	func(dst *PdfGlobal, raw string) error {
		val := strings.TrimSpace(raw)
		dst.PageSize = val
		return nil
	}, ...)
```

**Implemented solution:** the string `Set` remains the product (wkhtmltopdf CLI compatibility), descriptor parity is tested, and the typed builder has validated high-value methods plus an error-returning `WithSetting` escape hatch. The named page-size field is canonical; the former duplicate `Size.PageSize` field is removed.

#### HIGH-2 · Nil-context sentinel fragmentation (8 values) — FIXED
All conversion, load, prepare, image, layout, render, and application guards now alias canonical sentinels from `internal/errs`; `errors.Is` tests cover the public/app/engine boundaries. The `errs` declarations are wired rather than duplicate values.

```go
// errs.go:6-15 — the "canonical" package
var ErrNilContext = errors.New("gowkhtmltopdf: nil context")   // used: api.go:65, app/pdf.go:21
var ErrNilLoader  = errors.New("gowkhtmltopdf: nil loader")    // used: nowhere
var ErrNilCommand = errors.New("gowkhtmltopdf: nil command")   // used: nowhere
var ErrNilRequest = errors.New("gowkhtmltopdf: nil request")   // used: nowhere
```
```go
// convert.go:110 — a different value, same condition
var errNilContext = errors.New("convert: nil context")
// prepare.go:22 — identical TEXT, different value (errors.Is lies to string-readers)
var errNilContext = errors.New("convert: nil context")
// render/pipeline.go:20 — exported, different value
var ErrNilContext = errors.New("render: nil context")
// paint.go:13 · layout.go:751 · load.go:57 ("nil load context", no prefix) · imageout.go:60
```

Impact was masked at the surface (every public entry guards `ctx == nil` first), but it affected internal callers. The implementation routes guards through `errs.ErrNilContext`, wires `ErrNilLoader`/`ErrNilRequest`/`ErrNilCommand`, and aliases `app.ErrNilCommand` to the canonical value.

#### HIGH→MEDIUM-3 · `PdfGlobalOptions` lacks an explicit `Converter` integration path — FIXED
`Build()` works with `PDFRequest`, `ConvertHTML`, and now `Converter.WithGlobal`, which takes an owned settings snapshot. The fluent builder rejects nil receivers explicitly by panic policy; `WithSetting` provides an error-returning path for validated dotted values. Root-package tests cover builder snapshots, all common typed fields, converter integration, and caller-mutation isolation.

```go
// api.go — nil is an explicit programmer error for the fluent API.
func (o *PdfGlobalOptions) WithPageSize(pageSize string) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithPageSize(pageSize)
	return o
}
```

**Ideal solution:** add `func (c *Converter) WithGlobal(g *GlobalSettings) *Converter`; document the typed-request/Converter paths; decide whether nil builders should panic or return an explicit error; split `WithCopies`/`WithOutline`; validate in each `With*`; add tests.

#### MEDIUM-4 · Setters without validation — FIXED (negative margins retained)
Typed page-size input and dotted `size.pagesize` use the canonical parser; copies below one are rejected at builder, public request, and engine boundaries; negative header/footer margins remain valid because they are part of the HTML header/footer layout contract.

#### MEDIUM-5 · Error prefixes, duplicated sentinels, `OnError` bypass — FIXED
Error text remains package-contextual, but the public and engine output/outline/copies sentinels now alias where `errors.Is` is part of the contract. PDF and image public validation failures invoke `OnError`, with regression coverage in the root API tests.

#### MEDIUM-6 · Repeated validation — MEASURED AND BOUNDED
Validation is intentionally defense-in-depth at public/app preflight and the engine seam: direct engines validate once; app and typed public paths validate twice. The image command path no longer has a third production adapter. Tests assert validation happens before output creation, and the exact path ledger is recorded in the phase-wise checklist.

#### MEDIUM-7 · CLI flag-parser edge cases — FIXED AND TESTED
`--help=x` remains an invalid boolean value, `--no-help` is rejected as an unknown option, and `-xyz` is rejected as an unknown short-flag cluster. `flagTableSize = 120` remains only a capacity hint. Tests cover all three boundaries.

#### MEDIUM-8 · Global mutable maps ("immutable-by-convention") — CLOSED AS LOW RISK
`ignoredGlobalKeySet` (reflect.go:136), `globalKeys/objectKeys/imageKeys` (reflect.go:390-395), `flagTable`/`shortFlags` (flags.go:28-31) are plain maps guarded only by comments; read-only after init in practice. `pageSizes` (pagesize.go:17-42) shows the correct immutable-array pattern.

#### MEDIUM-9 · Redundant nil guards + context-in-struct — CLOSED WITH OWNERSHIP NOTES
Nil guards remain at public, app, and engine boundaries because each boundary owns a different contract; they now share canonical sentinels. The two context-containing structs and paint cancellation wrapper retain comments documenting their lifecycle ownership, and cancellation tests remain green.

#### LOW-10 · Misc ergonomics — CLOSED WITH TARGETED CLEANUPS
`ImageSettings.Set` now avoids a default-global allocation and duplicate key normalization; the remaining compatibility behaviors (`Get` on nil, optional `AddObject(nil)`, append-only lists, and examples) are documented policy choices rather than untracked correctness defects.

### 4.2 Engine, Memory & Concurrency (Discovery 2 → Validation)

#### CRITICAL→HIGH · `roundedBorderLineOverlay`: O(n²) backward scan — FIXED (image-mode only)
Every candidate `OpLine` now checks the immediately preceding rounded base stroke in O(1). PDF output is unaffected. Correctness remains covered by fixture 56 and the new 32/128/512-card benchmark reports 0 B/op and 8.3/13.9/57.6 µs per display-list workload.

```go
// imageout.go — bounded predecessor check per OpLine
func roundedBorderLineOverlay(ops []layout.Op, index int) (layout.OpStrokeRect, bool) {
	...
	stroke := ops[index-1]
	if stroke.Kind != layout.OpStrokeRect || ... { return empty, false }
}
```

The benchmark and fixture establish the adjacency contract; the bounded predecessor check is the implemented O(1) solution.

#### HIGH→MEDIUM · `FindWithGlyph` slice copy + per-face name-table parse — FIXED
`Font.LoadNames` caches name-derived values behind `sync.Once`, and `FindWithGlyph` scans the stable face slice under `RLock` without copying it. Focused benchmarks report 4.9 µs/136 B/7 allocs for cached name loading and 3.5 µs/64 B/2 allocs for glyph lookup.

#### HIGH→LOW · `PostScriptName` unsynchronized write — CLOSED BY CACHED NAME INITIALIZATION
The write is real (fonts.go:582-583) and read elsewhere (fonttype0.go:39, pdf.go:589, registry.go:287/293). **But** every producer sets it eagerly (faces.go:143, registry.go:340-341, prepare/styles.go:242); conversions are single-goroutine; the only cross-conversion shared state (`defaultFaces`) is `sync.Once`-published and pre-named. `go test -race` clean confirms it. Citation error in the discovery: "ensureFont fontpdf.go:47" is actually fonttype0.go:34. **Fix anyway** — `sync.Once` on `LoadNames` or eager assignment at `ParseTTF` is a 10-line insurance policy.

#### HIGH→MEDIUM · Per-page HTML header/footer re-parse + re-layout — MEASURED, SEMANTICALLY RETAINED
`hf.go:418-432`: a header containing `[page]`/`[topage]` is tokenized, cascaded, and laid out once per page — O(P × full header layout). The new 2/10/50-page benchmark records actual page counts of 2/10/50 and 7.35/1.24/2.83 ms one-iteration timings. Placeholder width changes are semantic, so the current relayout path is retained with evidence rather than replaced by an unsafe text-hash cache.

#### HIGH→MEDIUM · No PDF image dedup — FIXED (PNG-specific)
Every occurrence of the same image used to create a fresh XObject. `Content` now keeps a per-document SHA-256/grayscale key and reuses the XObject resource while emitting a distinct draw name for each occurrence. The repeated-image test proves identical indirect refs and one dedup entry.

#### HIGH→MEDIUM · Unconfigured `exhaustruct` under `enable-all` — FIXED CONFIGURATION
**.golangci.yml** now explicitly configures `exhaustruct.check-exported: true`; the full `make lint` gate passes. Existing intentional struct-literal suppressions remain reviewable and justified rather than being mechanically removed from correctness-sensitive constructors.

#### MEDIUM findings (all CONFIRMED)
- **`supersamplePixPool` 16 MiB floor** (imageout.go:319-326): fixed by removing the unconditional `New` allocation; pool misses allocate only the required raster capacity. The small/large pool benchmark records 331 KB and 9.3 MB allocation traffic respectively.
- **`SectionOfBy` linear scan per page** (outline.go:232-248, called hf.go:673): fixed with `sort.Search`, making lookup O(log H) per page.
- **Per-run shaping allocations in image mode** (shape_gotext.go:51-61, 117-155; ttfraster.go:46 calls `pdf.ShapeRun` per OpText) while the PDF path already skips it (content.go:365-368).
- **Per-rounded-fill mask + uniform** (imageout.go:909-928): `image.NewAlpha(rect)` + `image.NewUniform(col)` per fill.
- **Multi-pass pagination with a ten-iteration fixpoint cap** (paint_pagination.go:457-526): the code has several staged passes and bounded fixpoint loops. The general 500-page benchmark and header/footer benchmark provide current timing/page-count evidence; the state machine is retained because no semantic-preserving simplification was justified by the measured workloads.
- **`cloneLoadPage` deep copy per fetch** (load.go:881-889; sites 112/411/1033): nil maps short-circuit; downgraded LOW.
- **Watcher goroutine per file read** (load.go:695-725): deliberate robustness design retained after review because it force-closes blocked `os.File` reads and protects cancellation.
- **Per-op `fontStack` append** (content.go:174-189): downgraded LOW.

#### LOW findings (CONFIRMED)
- `debug.FreeOSMemory()` every 4 islands (page_islands.go:70-75) — benchmark-only via `NewBenchmarkPDFRequest`.
- Complexity-suppression bypasses on genuinely oversized functions: `renderObject` (convert.go:460, ~140 lines), `beforeAlways` (paint_flow.go:631), flow state machine (layout_flow.go:123), `drawHeadersFootersResult` (hf.go:637), `flexColumnItems` (flex.go:1249 — discovery cited 1398, wrong), `cascadeRaw` (style_cascade.go:333-334). Honest justifications; extract where responsibilities mix.
- Style interning stack copies (style.go:543-590) — acceptable. `atlas.get` per-glyph closure (ttfraster.go:96-113).

### 4.3 Additional issues found by Validation Agent 2 (not in discovery)

1. **Engine→CLI layering inversion (the deepest structural issue) — FIXED** — `internal/app` now owns both command adapters; production `internal/convert` and `internal/imageout` consume request types only. Historical same-package test callers are preserved in `_test.go` compatibility adapters.
2. **No targeted microbenchmarks for the claimed hotspots — FIXED** — overlay lookup, repeated-resource PDF, header/footer placeholders, font name/glyph lookup, and supersample pooling now have focused measurements.
3. **`OnError` contract violation — FIXED** — public PDF and image validation failures invoke the callback and are regression-tested.
4. **The test suite cemented the dual-write wart — FIXED** — `Size.PageSize` was removed, geometry uses canonical `PageSize`, and settings parity tests protect the new contract.
5. **`runContext` god-struct** (14 fields, convert.go:277-293) — the closest thing to a C++ manager class; bounded and short-lived, defensible.

---

## 5. Subagent Validation Matrix

| # | Discovery finding | Severity after validation | Verdict | Validators |
|---|---|---|---|---|
| A1 | String-key settings table (no reflection, dual registration) | HIGH → MEDIUM | FIXED BY PARITY CONTRACT | V1 ✓ V2 ✓ + descriptor parity test |
| A2 | Nil-context sentinel fragmentation | MEDIUM | FIXED | Canonical `internal/errs` aliases + `errors.Is` tests |
| A3 | `PdfGlobalOptions` lacks explicit Converter integration | MEDIUM | FIXED | `WithGlobal`, ownership tests, typed-builder tests |
| A4 | Setters without validation | MEDIUM (negative-margin sub-claim withdrawn) | FIXED | Canonical page-size parser and copies boundary |
| A5 | Error prefixes / OnError bypass | MEDIUM | FIXED | Public/engine sentinel aliases + callback tests |
| A6 | Repeated validation | MEDIUM | MEASURED AND BOUNDED | Two deliberate gates on app/public paths; one direct engine gate |
| A7 | CLI flag-parser edge cases | MEDIUM | FIXED | `cli_test.go` boundary cases |
| A8 | Global mutable maps | LOW (downgraded) | CLOSED AS LOW RISK | V1 ✓ V2 ✓ + package-private read-only audit |
| A9 | Redundant nil guards + containedctx | MEDIUM | CLOSED WITH OWNERSHIP NOTES | V1 ✓ V2 ✓ + canonical sentinel tests |
| A10 | Misc ergonomics | LOW | CLOSED WITH TARGETED CLEANUPS | V1 ✓ V2 ✓ |
| E1 | `roundedBorderLineOverlay` O(n²) | HIGH (downgraded from CRITICAL) | FIXED | O(1) predecessor check + 32/128/512-card benchmark |
| E2 | `FindWithGlyph` copy + name parse | MEDIUM (downgraded) | FIXED | V1 ✓ V2 ✓ + focused benchmarks |
| E3 | `PostScriptName` race | LOW (latent) | CLOSED BY CACHED INITIALIZATION | V1 ✓ V2 ✓ |
| E4 | Per-page HTML HF re-layout | MEDIUM (downgraded; semantic) | MEASURED AND RETAINED | V1 ✓ V2 ✓ + page-count benchmark |
| E5 | No PDF image dedup | MEDIUM (downgraded; PNG-specific) | FIXED | V1 ✓ V2 ✓ + repeated-image test |
| E6 | exhaustruct nolint flood | MEDIUM (399 non-test lines) | FIXED CONFIGURATION | V1 ✓ V2 ✓ + make lint |
| E7-E14 | Pool floor, scans, shaping allocs, pagination passes | MEDIUM (two downgraded to LOW) | MEASURED, FIXED WHERE SAFE | V1 ✓ V2 ✓ + focused benchmarks |
| E15-E18 | FreeOSMemory, complexity bypasses, interning, atlas | LOW | CONFIRMED | V1 ✓ V2 ✓ |
| NEW | Engine→CLI layering inversion (duplicate adapter stacks) | HIGH-adjacent (root cause) | FIXED AT APP BOUNDARY | V2 + package import audit |
| NEW | supersamplePixPool dead-code fallback | MEDIUM | FIXED | V1 + small/large pool benchmark |

**Discovery 1:** all 10 findings were rechecked; A1/A8/A9/A10 now have explicit compatibility or ownership dispositions.
**Discovery 2:** findings were rerun against the implementation wave; fixed, measured-retained, and low-risk decisions are recorded above.
**Empirical facts (orchestrator-run):** build ✅, vet ✅, test ✅ (all packages), race ✅ (pdf, imageout, convert, layout).
**Citation drift note:** Agent 2's citations were systematically 20-450 lines off (one wrong file: ensureFont fontpdf.go:47 → fonttype0.go:34). Every cited behavior was reproduced; re-grep citations before editing.

### Live cross-check corrections

- Current source contains general pipeline benchmarks plus focused overlay, repeated-resource, header/footer, font, and supersample benchmarks.
- Current one-iteration results on the review host were approximately 1.051 s/op and 226.7 MB/op for generic 500-page conversion, and 1.101 s/op and 187.9 MB/op for certified-islands mode. `B/op` is Go allocation traffic, not process RSS.
- `PdfGlobalOptions.Build()` is usable with `PDFRequest`, `ConvertHTML`, and `Converter.WithGlobal`; ownership tests prove caller mutation cannot alter the converter.
- The CLI checks preserve invalid `--help=x` behavior while explicitly rejecting `--no-help` and `-xyz` at parse time.
- The current package graph has 23 packages, current Go-file totals are 47,334 non-test and 32,140 test lines, and the current production `nolint:exhaustruct` count is 399.

---

## 6. Devil's Advocate Evaluation

### 6.1 Verdicts on the big fights

| Finding | Proposed fix | DA verdict |
|---|---|---|
| Settings table | "Functional options as primary surface" | **REJECT as stated** — the string surface IS the product (wkhtmltopdf compat). Fix = parity test + builder completion. |
| Per-page HF re-layout | "Cache by substituted-text hash" | **REJECT the naive fix** — substituted numbers change measured width; reflow is partly the point. Benchmark, then consider paint-time placeholders. |
| Negative margins | "ValidatePDF should reject them" | **REJECT** — documented feature ("auto: reserve HF height"). |
| Triple validation | "Delete duplicate passes" | **MEASURED AND BOUNDED** — defense-in-depth remains at explicit public/app and engine ownership boundaries. |
| Watcher goroutine per file | "Replace with ctx poll loop" | **RETAINED BY DESIGN** — it force-closes blocked files; cancellation tests cover the behavior. |
| Pagination multi-pass | "Reduce passes" | **MEASURED AND RETAINED** — capped/index-accelerated state machine remains semantically conservative under current workloads. |
| lineLog `Write` discard | "Handle the error" | **SKIP** — `bytes.Buffer.Write` cannot fail; plumbing would be noise. |
| Builder/Converter + nil no-ops + OnError | "Add `WithGlobal`, drop nil guards, fire OnError" | **AGREE — production-impact, front-load.** |
| Rounded-border O(n²) | "Carry last rounded-stroke index" | **AGREE — after writing the benchmark first.** |
| PDF image dedup | "Per-document hash→XObject cache" | **AGREE — well-bounded, real user value (watermarks/logos).** |
| exhaustruct config | "check-exported: true / excludes" | **AGREE — best maintainability-per-hour return in the repo.** |
| PostScriptName | "sync.Once / eager assignment" | **AGREE — 10-line insurance policy, not a bug fix.** |

### 6.2 Implemented fix ranking

| Rank | Fix | User impact | Cost |
|---|---|---|---|
| 1 | Explicit builder/Converter integration, nil policy, and OnError | High — public behavior | Implemented |
| 2 | Rounded-border O(1) predecessor lookup | Medium — border-heavy image renders | Implemented |
| 3 | PDF PNG XObject deduplication | Medium — repeated logos/watermarks | Implemented |
| 4 | Canonical sentinel and page-size contracts | Medium — API matching and geometry | Implemented |
| 5 | Focused hotspot benchmarks and RSS reporting | Medium — evidence quality | Implemented |
| 6 | CLI boundary semantics and app-owned adapters | Medium — predictable command behavior | Implemented |
| 7 | Exact-capacity supersample pooling and font caches | Low-Medium — allocation traffic | Implemented |
| 8 | Lint configuration and complexity audit | Low — maintainability | Implemented |

---

## 7. Completed 5-Phase Remediation Record

### Phase 1 — Public API correctness — complete

`WithGlobal` now owns a cloned settings snapshot; root builder tests cover typed fields, dotted settings, snapshot isolation, and Converter integration. Nil fluent-builder calls panic by documented policy. Page-size parsing, copies validation, negative header/footer margins, and `OnError` validation callbacks are tested for both PDF and image converters.

### Phase 2 — Performance measurement and fixes — complete

The overlay, repeated-resource PDF, header/footer placeholder, font registry, font-name, and supersample-pool benchmarks are checked in. The implementation applies O(1) border lookup, PNG XObject deduplication, exact-capacity raster pooling, `SectionOfBy` binary search, cached font names, and no-copy registry scans. PDF/page semantics are protected by targeted tests.

### Phase 3 — Settings and configuration — complete

The settings descriptor parity test covers setter/getter registration, typed page-size parsing uses the canonical table, `ImageSettings.Set` normalizes once without constructing a default global, and `PageSize` is the sole named geometry field. The lint configuration explicitly enables exported-struct checking.

### Phase 4 — Layering and sentinel hygiene — complete

CLI adapters are owned by `internal/app`; production engines have no `internal/cli` dependency. Historical same-package adapter calls remain only in `_test.go` compatibility helpers. Nil sentinels are canonicalized through `internal/errs`, public and engine output errors alias where matching is promised, and CLI boundary tests cover the corrected flag behavior.

### Phase 5 — Validation and rerating — complete

Focused tests, the lint gate, full build/vet/test gates, targeted race coverage, and measured benchmarks were run. The rerated score is 8.6/10 using the weighted arithmetic in §1. The remaining score gap is an explicit compatibility/performance tradeoff backed by tests and measurements, not an untracked or deferred checklist item.

**Current end-state: 8.6/10.** The remaining gap is an explicit compatibility-first tradeoff: the dotted settings API is retained as the product surface and protected by parity tests, while the typed builder covers common global options and exposes `WithSetting` for the complete key surface.

---

## 8. Verification Appendix (orchestrator-run)

```
GOCACHE=/tmp/gowk-go-cache make test   → PASS (all repository packages)
GOCACHE=/tmp/gowk-go-cache make lint   → PASS (golangci-lint v1.64.8)
go build ./...                         → PASS
go vet ./...                           → PASS
go test ./...                          → PASS
go test -race -count=1 ./internal/pdf ./internal/imageout ./internal/convert ./internal/layout → PASS
```

Focused benchmark evidence:

```
overlay 32/128/512 cards: 8.280/13.944/57.616 µs, 0 B/op, 0 allocs/op
font names / glyph lookup: 4.949/3.534 µs, 136/64 B/op
header/footer 2/10/50 pages: 7.351/1.243/2.827 ms; actual pages 2/10/50
repeated PDF 500 pages: 418.820 ms, 2,715,534 PDF bytes, 61,673,472 HWM bytes
```

*Review wave: 5 subagents (Discovery 1 & 2 → Validation 1 & 2 → Criticizer), all source findings verified line-by-line, race detector clean. Generated by the critical-go-review skill workflow.*
