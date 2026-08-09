# 10/10 Golang Architecture, Code Style, & Quality Roadmap

**Parent:** `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/`  
**Date:** 2026-08-09  
**Status:** Comprehensive Codebase Scan Complete; 10/10 Architecture & Implementation Blueprint Established  
**Scope:** Entire Go production codebase (`api.go`, `cmd/`, `internal/...`), test harnesses, benchmarks, and linter specifications  
**Methodology:** 4 specialized subagent audit tracks (API & Loader, Layout & Pipeline, Performance & Concurrency, Style & Ergonomics) reconciled into a single unified execution plan  

---

## 1. Executive Summary & Diagnosis of the 7.5/10 Score

### Why Was the Rating Previously Capped at 7.5/10?
The previous post-remediation review evaluated the codebase at **7.5/10 (Code Style v1.2.2)** and **7.6/10 (Design Patterns v1.1.5)** due to a combination of architectural boundaries, structural design debts, and methodology constraints:

1. **Source-Only Verification Boundary (No Runtime Certification):**  
   The review was conducted without executing Git or Go commands (`go test`, `go test -race`, `go bench`, `golangci-lint`). Without empirical execution proof, race detection, or verified benchmark allocations, the rating could not exceed the 7.5–7.6 band.
2. **Untyped Tagged Union Design (`convert.Request`):**  
   `convert.Request` mixed PDF (`settings.PdfGlobal`) and Image (`settings.ImageGlobal`) settings within a single struct using pointers (`Image *settings.ImageGlobal`). This permitted unrepresentable/invalid state combinations at compile-time and required manual validation guards.
3. **Anti-Pattern Deferred Error Storage (`initErr`) & Policy Duplication:**  
   `load.NewLoader` caught initialization errors (such as proxy URL parsing) and saved them inside `l.initErr`, returning a partially initialized loader that failed later during resource fetch rather than failing fast during construction. Policy fields (`Allow`, `EnableLocalFileAccess`) were also duplicated across `Loader` and `GlobalSettings`.
4. **Context Discarded on Disk I/O & Implicit `context.Background()` Fallbacks:**  
   Local file loading (`loadFile`) accepted `_ context.Context` and performed `os.Open` and `filepath.EvalSymlinks` without pre-checking `ctx.Err()`. Across `internal/layout`, `internal/convert`, and `internal/imageout`, missing contexts were silently replaced with `context.Background()`.
5. **Synthetic DOM Ownership & Page Island Mutations:**  
   Synthetic DOM construction for page islands (`benchmarkIslandRoot`) severed parent chains (`<head>`, container wrappers) and created orphaned heading pointers, breaking CSS ancestor matching and outline filtering.
6. **In-Place Mutation of Shared Cached Display Lists:**  
   When HTML header/footers were cached (`htmlHFLayout`), page rendering invoked `resolveRelativeLinkURIs`, which modified `res.Ops[i].URI` in place across page draws, creating concurrency risks and state corruption.
7. **Function Signature Parameter Bloat & Broad `nolint` Directives:**  
   Functions like `paintOpOnPage` (10 parameters), `PrepareDocument` (7 parameters), and `drawHeadersFootersResult` (8 parameters) passed raw primitive pointers down stack frames. Broad suppressions (`//nolint:gocognit,cyclop,funlen,lll`) in `hf.go` concealed structural complexity debt.
8. **Package-Level Globally Mutable State & Data Race Vectors:**  
   Package `var` maps (`namedColorTable`, `flagTable`, `autoClose`, `arabicForms`, `knownPlaceholders`) were globally exported or package-level mutable maps. `pdf.Document` and `pdf.Registry` lacked mutex protection (`sync.RWMutex`).

---

## 2. Four-Track Synthesis for 10/10 Rating

| Track | Current Score | 10/10 Target | Key Technical Requirements for 10/10 Rating | Weight |
|---|---:|---:|---|---:|
| **Track 1: API, Settings, Loader & CLI** | 7.8 / 10 | 10.0 / 10 | Sealed `Request` interface (`PDFRequest` vs `ImageRequest`); fail-fast `NewLoader(*Loader, error)`; `HardMaxBodyCap` (1 GiB) ceiling; thread-safe functional options on `Converter`. | 25% |
| **Track 2: Layout Engine & DOM Ownership** | 7.6 / 10 | 10.0 / 10 | `pagePainter` & `PrepareParams` parameter encapsulation; atomic DOM child creation; build-time HF link resolution; pristine vs painted TOC display list separation. | 30% |
| **Track 3: Performance & Concurrency** | 7.5 / 10 | 10.0 / 10 | `sync.RWMutex` on `pdf.Registry` & `pdf.Document`; zero-alloc `flateBytes` buffer reuse; `sync.Pool` supersample buffer recycling; NaN/Inf SVG bounds check; single-sweep page index validation. | 30% |
| **Track 4: Style, Idioms & Ergonomics** | 7.4 / 10 | 10.0 / 10 | Zero `nolint` complexity suppressions; canonical `errs.ErrNilContext`; replacement of all global `var` maps with `sync.OnceValue` or `switch`; enum `...Unknown = 0` zero values. | 15% |
| **Composite Synthesis** | **7.58 → 7.6 / 10** | **10.0 / 10** | **Fully certified via zero race conditions, 100% linter compliance, zero-alloc paths, and full execution proof.** | **100%** |

---

## 3. Comprehensive Line-by-Line Findings & 10/10 Solution Blueprint

---

### Track 1: API, Settings, Loader, CLI & App Boundaries

#### [CNV-01] Untyped Tagged Union Design of `convert.Request`
* **Severity:** High  
* **Files & Line Numbers:** `internal/convert/convert.go` (L53-L68, L125-L132), `api.go` (L337, L523)  
* **Root Cause:** `convert.Request` co-locates `Global settings.PdfGlobal` and `Image *settings.ImageGlobal`. This permits invalid states (e.g. a PDF request with non-nil Image settings) and forces manual runtime validation checks.  
* **10/10 Solution:** Replace `Request` with a sealed interface hierarchy:
  ```go
  type Request interface {
      isRequest()
      Validate() error
  }

  type PDFRequest struct {
      Global        settings.PdfGlobal
      Objects       []settings.PdfObject
      Now           func() time.Time
      Output        io.Writer
      OutlineOutput io.Writer
  }
  func (r *PDFRequest) isRequest() {}
  func (r *PDFRequest) Validate() error { ... }

  type ImageRequest struct {
      Global  settings.PdfGlobal
      Image   settings.ImageGlobal
      Objects []settings.PdfObject
      Output  io.Writer
  }
  func (r *ImageRequest) isRequest() {}
  func (r *ImageRequest) Validate() error { ... }
  ```
* **Acceptance Criteria:** Type safety enforced at compile-time; invalid request configurations are unrepresentable.

---

#### [LOAD-01] Loader Policy Duplication & Anti-Pattern Deferred Error Storage (`initErr`)
* **Severity:** High  
* **Files & Line Numbers:** `internal/load/load.go` (L262-L301, L395-L397), `internal/convert/prepare.go` (L26-L56)  
* **Root Cause:** `NewLoader` catches initialization errors (like malformed proxy URLs) and stores them in `l.initErr` to return a non-nil `*Loader`. The error is deferred until `Load()` is invoked. `l.Allow` and `l.EnableLocalFileAccess` also duplicate `l.global` fields.  
* **10/10 Solution:** Eliminate `initErr` and duplicate policy fields. Fail fast in the constructor:
  ```go
  type Loader struct {
      client       *http.Client
      global       settings.LoadGlobal
      log          io.Writer
      maxBodySize  int64
      maxRedirects int
  }

  func NewLoader(global settings.LoadGlobal) (*Loader, error) {
      g := global
      g.Allow = cloneStrings(global.Allow)

      l := &Loader{
          global:       g,
          log:          io.Discard,
          maxBodySize:  DefaultMaxBodySize,
          maxRedirects: DefaultMaxRedirects,
      }
      if err := l.initClient(); err != nil {
          return nil, fmt.Errorf("load: init loader client: %w", err)
      }
      return l, nil
  }
  ```
* **Acceptance Criteria:** `NewLoader` returns `(*Loader, error)`; `initErr` is deleted; policy fields are read exclusively through `l.global`.

---

#### [LOAD-02] Missing Context Cancellation Checks Prior to Local Disk I/O & Unbounded Body Read Risk
* **Severity:** Critical  
* **Files & Line Numbers:** `internal/load/load.go` (L678-L684, L725-L760), `internal/convert/prepare.go` (L88-L121)  
* **Root Cause:** `loadFile` executes `filepath.EvalSymlinks`, ACL evaluation, and `os.Open` without pre-checking `ctx.Err()`. `bodyReadLimit` accepts `1<<63 - 1` (`math.MaxInt64`), enabling potential OOM vulnerabilities on malformed body sizes.  
* **10/10 Solution:** Inject explicit pre-I/O `ctx.Err()` checks and enforce a non-negotiable hard ceiling:
  ```go
  const HardMaxBodyCap int64 = 1 << 30 // 1 GiB hard security cap

  func (l *Loader) loadFile(ctx context.Context, path string, pageLoad settings.LoadPage) (*Resource, error) {
      if ctx == nil {
          return nil, errs.ErrNilContext
      }
      if err := ctx.Err(); err != nil {
          return nil, err
      }

      filePath, err := filePathFromURL(path)
      if err != nil {
          return nil, err
      }

      if !l.fileAccessAllowed(filePath, pageLoad) {
          return nil, fmt.Errorf("%w: %s", ErrAccessDenied, filePath)
      }

      if err := ctx.Err(); err != nil {
          return nil, err
      }

      file, err := os.Open(filePath)
      if err != nil {
          return nil, fmt.Errorf("open %s: %w", filePath, err)
      }
      defer file.Close()

      return readFileResource(ctx, file, filePath, l.effectiveMaxBodySize())
  }
  ```
* **Acceptance Criteria:** Context cancellation short-circuits filesystem I/O immediately; body reads are strictly capped at `HardMaxBodyCap`.

---

#### [CTX-01] Inconsistent Nil Context Sentinels & Implicit `context.Background()` Fallbacks
* **Severity:** High  
* **Files & Line Numbers:** `api.go` (L40-L42), `internal/app/pdf.go` (L19, L56-L58), `internal/convert/convert.go` (L195-L197), `internal/layout/layout.go` (L661-L663)  
* **Root Cause:** Multiple distinct unexported sentinel errors (`app.errNilContext`, `convert.errNilContext`) exist separate from `gowkhtmltopdf.ErrNilContext`, breaking `errors.Is`. Internal functions also inject `context.Background()`.  
* **10/10 Solution:** Standardize on a single canonical package `internal/errs`:
  ```go
  // internal/errs/errs.go
  package errs
  import "errors"

  var ErrNilContext = errors.New("gowkhtmltopdf: nil context")

  // api.go
  var ErrNilContext = errs.ErrNilContext
  ```
* **Acceptance Criteria:** `errors.Is(err, gowkhtmltopdf.ErrNilContext)` returns `true` for all nil context errors across the entire library; zero implicit `context.Background()` fallbacks in `internal/`.

---

#### [API-02] Exported Mutable Struct Fields & Data Races on Callbacks
* **Severity:** High  
* **Files & Line Numbers:** `api.go` (L176-L186), `internal/settings/settings.go` (L206-L210, L286-L325)  
* **Root Cause:** `api.Converter` exports public callback function fields (`OnInfo`, `OnWarn`, `OnError`, `OnProgress`), permitting concurrent mutation during conversion. `settings` structs export raw maps and slices (`Allow`, `FontPaths`, `Cookies`).  
* **10/10 Solution:** Encapsulate callback fields with thread-safe functional options and protect settings slices:
  ```go
  type Option func(*Converter)

  func WithOnInfo(fn func(string)) Option {
      return func(c *Converter) {
          c.mu.Lock()
          defer c.mu.Unlock()
          c.onInfo = fn
      }
  }

  type Converter struct {
      mu      sync.RWMutex
      global  *GlobalSettings
      objects []*ObjectSettings
      onInfo  func(string)
      onWarn  func(string)
  }
  ```
* **Acceptance Criteria:** `Converter` callbacks are thread-safe; external caller cannot mutate slice or map headers during active conversion runs.

---

### Track 2: Layout Engine, DOM Ownership & Pipeline Architecture

#### [LAYOUT-01] Severe Parameter Bloat in `paintOpOnPage`
* **Severity:** High  
* **Files & Line Numbers:** `internal/layout/paint.go` (L342-L345)  
* **Root Cause:** `paintOpOnPage` accepts 10 positional arguments including raw mutable pointers (`nextImg *int`, `paintErr *error`), polluting stack frames and preventing clean unit testing of individual operation painters.  
* **10/10 Solution:** Encapsulate layout painting state inside a `pagePainter` receiver struct:
  ```go
  type pagePainter struct {
      child    *pdf.Content
      page     *pdf.Page
      pageN    int
      contentH float64
      pageH    float64
      opts     PaintOptions
      resName  func(*pdf.Font) string
      nextImg  int
      err      error
  }

  func (p *pagePainter) paintOp(paintOp *Op) {
      if paintOp.Kind == opKindNoop {
          return
      }
      if paintOp.Kind == OpLinkURI {
          drawLinkXform(p.page, paintOp, p.pageN, p.contentH, p.opts)
          return
      }
      ...
  }
  ```
* **Acceptance Criteria:** Zero raw primitive pointers passed down stack frames; `pagePainter` methods encapsulate painting state with zero allocations per op.

---

#### [CONV-01] Parameter Bloat in `PrepareDocument`, `drawHeadersFootersResult`, and `paintLayoutOps`
* **Severity:** Medium  
* **Files & Line Numbers:** `internal/convert/prepare.go` (L151-L159), `internal/convert/hf.go` (L508-L509, L668-L673)  
* **Root Cause:** `PrepareDocument` receives 7 parameters, `drawHeadersFootersResult` receives 8 parameters, and `paintLayoutOps` receives 7 parameters, creating unwieldy signatures.  
* **10/10 Solution:** Group execution options into domain parameters:
  ```go
  type PrepareParams struct {
      Loader   *load.Loader
      Page     string
      LoadPage settings.LoadPage
      Registry *pdf.Registry
      Log      io.Writer
      Options  PrepareOptions
  }
  func PrepareDocument(ctx context.Context, params PrepareParams) (*PreparedDocument, error)

  type HeaderFooterDrawContext struct {
      Loader   *load.Loader
      Font     *pdf.Font
      Doc      *pdf.Document
      Req      *Request
      Plan     *pagePlan
      Headings []*outline.Heading
      Log      io.Writer
  }
  func drawHeadersFootersResult(ctx context.Context, hfCtx HeaderFooterDrawContext) hfDrawResult
  ```
* **Acceptance Criteria:** Every function signature accepts $\le 3$ parameters (`ctx`, target struct, data slice).

---

#### [DOM-01] DOM Ancestor Severing and Stale Heading References in `benchmarkIslandRoot`
* **Severity:** Critical  
* **Files & Line Numbers:** `internal/convert/page_islands.go` (L204-L229)  
* **Root Cause:** Synthetic tree creation (`html -> body -> section`) strips `<head>`, styles, and wrapper containers, severing CSS selector ancestor chains. Outline headings collected from transient synthetic trees hold dangling node pointers after island garbage collection.  
* **10/10 Solution:** Preserve full DOM tree hierarchy and remap synthetic heading node pointers back to canonical DOM nodes:
  ```go
  func benchmarkIslandRoot(root, section *html.Node) (*html.Node, map[*html.Node]*html.Node) {
      if root == nil || section == nil {
          return nil, nil
      }
      nodeMap := make(map[*html.Node]*html.Node)
      copyRoot := cloneHTMLTreeStructure(root, nil, section, nodeMap)
      return copyRoot, nodeMap
  }
  ```
* **Acceptance Criteria:** Full CSS ancestor matching preserved; headings collected from page islands are cleanly remapped to canonical DOM nodes without dangling references.

---

#### [DOM-02] Non-Atomic Parent-Child Reparenting in `cloneHTMLNodeShell`
* **Severity:** High  
* **Files & Line Numbers:** `internal/convert/page_islands.go` (L231-L261)  
* **Root Cause:** `cloneHTMLNodeShell` sets `Parent: parent` but does not add the child to `parent.Children`. `parent.Children` is populated manually later, creating window states where tree relationships are internally contradictory.  
* **10/10 Solution:** Enforce atomic node creation:
  ```go
  func createChildNode(parent *html.Node, src *html.Node) *html.Node {
      if src == nil {
          return nil
      }
      child := &html.Node{
          Type:     src.Type,
          Name:     src.Name,
          Attrs:    cloneHTMLAttrs(src.Attrs),
          Text:     src.Text,
          Parent:   parent,
      }
      if parent != nil {
          parent.Children = append(parent.Children, child)
      }
      return child
  }
  ```
* **Acceptance Criteria:** Invariants `child.Parent == parent` and `parent.Children[i] == child` hold at all times.

---

#### [HF-01] In-Place Mutation of Shared Display-List Ops in Cached HTML HF Layouts
* **Severity:** Critical  
* **Files & Line Numbers:** `internal/convert/hf.go` (L469-L472, L516-L576)  
* **Root Cause:** Cached `htmlHFLayout.res` display list operations (`res.Ops`) undergo `resolveRelativeLinkURIs` on every page draw, mutating `res.Ops[i].URI` in place across page draws and corrupting shared cached display lists.  
* **10/10 Solution:** Perform relative link URI resolution **once** at layout build time:
  ```go
  func loadHTMLHF(...) (*htmlHFLayout, *pdf.Registry, error) {
      ...
      lst.res, err = layout.LayoutContext(ctx, root, layout.Options{...})
      if err != nil {
          return nil, nil, err
      }
      resolveRelativeLinkURIs(lst.res.Ops, res.Base)
      return lst, reg, nil
  }
  ```
* **Acceptance Criteria:** Cached display list operations are immutable during page rendering; zero in-place mutations during draw.

---

#### [TOC-01] Box Tree Aliasing and Post-Paint Result Mutation in `renderTOCObjects`
* **Severity:** Critical  
* **Files & Line Numbers:** `internal/convert/toc.go` (L271-L277), `internal/layout/layout.go` (L112-L136)  
* **Root Cause:** `state.tocRes` is overwritten with `painted` (`layout.PaintContext` output). `PaintContext` mutates ops in place (splitting rects, applying sticky shifts). Sub-slice aliasing in `cloneBoxGraph` also causes shared row box modifications.  
* **10/10 Solution:** Separate pristine pre-paint result `state.tocRes` from `state.tocPaintedRes`, and perform deep cloning of `box.rows`:
  ```go
  for _, state := range tocs {
      state.start = doc.PageCount()
      painted := cloneResult(state.tocRes)

      if err := layout.PaintContext(ctx, doc, painted, paintOptions(state.geom)); err != nil {
          return 0, fmt.Errorf("object %d: toc: paint: %w", state.idx+1, err)
      }
      state.tocPaintedRes = painted
      total += state.tocPages
  }
  ```
* **Acceptance Criteria:** `state.tocRes` remains pristine; `CloneResult` completely isolates all `box.rows` sub-slices.

---

### Track 3: Performance, Memory Allocations, Rasterization & Concurrency

#### [CONC-01] Lack of Mutex Synchronization on `Document` and `Registry`
* **Severity:** High  
* **Files & Line Numbers:** `internal/pdf/pdf.go` (L114-L131, L199-L209), `internal/pdf/registry.go` (L11-L13, L21-L39, L61-L80)  
* **Root Cause:** Neither `pdf.Document` nor `pdf.Registry` incorporates mutex synchronization. Concurrent layout workers calling `AddPage` or concurrent font lookups during `AddFont` cause data races on slice headers and map entries.  
* **10/10 Solution:** Protect internal data structures with `sync.RWMutex`:
  ```go
  type Registry struct {
      mu       sync.RWMutex
      byFamily map[string][]*Font
  }

  func (r *Registry) AddFont(fnt *Font) {
      if r == nil || fnt == nil {
          return
      }
      r.mu.Lock()
      defer r.mu.Unlock()
      ...
  }

  func (r *Registry) Lookup(families []string, weight int, italic bool) *Font {
      if r == nil {
          return nil
      }
      r.mu.RLock()
      defer r.mu.RUnlock()
      ...
  }
  ```
* **Acceptance Criteria:** `go test -race ./internal/pdf/...` passes with zero data races under heavy parallel loads.

---

#### [MEM-01] High Allocation Overhead in `flateBytes` and Pooled Buffer Discard Risk
* **Severity:** Medium  
* **Files & Line Numbers:** `internal/pdf/pdf.go` (L919-L947)  
* **Root Cause:** `flateBytes` executes `state.buf = bytes.Buffer{}` to return `state.buf.Bytes()` without allocation. However, this returns a 0-capacity buffer back to `flatePool`, forcing future pooled instances to reallocate their internal byte slice from scratch.  
* **10/10 Solution:** Copy the returned compressed bytes so `state.buf` retains its backing capacity in the pool:
  ```go
  func flateBytes(raw []byte) []byte {
      state, _ := flatePool.Get().(*flateState)
      if state == nil {
          state = &flateState{}
          state.zw, _ = zlib.NewWriterLevel(&state.buf, zlib.DefaultCompression)
      } else {
          state.buf.Reset()
          state.zw.Reset(&state.buf)
      }

      _, _ = state.zw.Write(raw)
      _ = state.zw.Close()

      res := append([]byte(nil), state.buf.Bytes()...)
      flatePool.Put(state)
      return res
  }
  ```
* **Acceptance Criteria:** Pooled buffer capacity preserved across requests; overall `B/op` allocation drops during PDF stream compression.

---

#### [PERF-01] Slice Allocation Waste in Supersampled Image Rasterization
* **Severity:** Medium  
* **Files & Line Numbers:** `internal/imageout/imageout.go` (L325, L349, L588-L686)  
* **Root Cause:** Supersampling allocates a 4x larger `image.NRGBA` canvas (e.g. 12.5 MB), downscales it, and discards the supersampled `img.Pix` buffer to garbage collection.  
* **10/10 Solution:** Implement a `sync.Pool` for supersample pixel buffers:
  ```go
  var supersamplePixPool = sync.Pool{
      New: func() any {
          return make([]byte, 0, 16<<20) // 16 MiB default max capacity
      },
  }

  func getSupersampleBuffer(size int) []byte {
      buf := supersamplePixPool.Get().([]byte)
      if cap(buf) < size {
          return make([]byte, size)
      }
      return buf[:size]
  }

  func putSupersampleBuffer(buf []byte) {
      if cap(buf) <= 16<<20 {
          supersamplePixPool.Put(buf[:0])
      }
  }
  ```
* **Acceptance Criteria:** Image rasterization allocations decrease by $>50\%$; zero intermediate supersample slice garbage.

---

#### [BND-01] Bounds Check & Negative/NaN Float Overflow Risks in SVG Rasterization
* **Severity:** High  
* **Files & Line Numbers:** `internal/svg/raster.go` (L129-L156, L192-L219, L249-L259)  
* **Root Cause:** SVG dimensions (`viewBox`, `width`, `height`) parsed as NaN, Inf, or negative floats convert to implementation-defined min-int values (`-9223372036854775808`), triggering buffer allocation panics in image canvas creation.  
* **10/10 Solution:** Validate all parsed dimensions against bounds and special float values:
  ```go
  func svgCSSPixelSize(data []byte, maxSide int) (int, int) {
      viewW, viewH := rootSVGSize(data)
      if math.IsNaN(viewW) || math.IsInf(viewW, 0) || viewW <= 0 {
          viewW = 100
      }
      if math.IsNaN(viewH) || math.IsInf(viewH, 0) || viewH <= 0 {
          viewH = 100
      }

      if maxSide <= 0 {
          maxSide = 512
      } else if maxSide > maxImageDimension {
          maxSide = maxImageDimension
      }

      scale := 1.0
      if viewW > float64(maxSide) || viewH > float64(maxSide) {
          scale = float64(maxSide) / math.Max(viewW, viewH)
      }

      pixW := int(math.Ceil(viewW * scale))
      pixH := int(math.Ceil(viewH * scale))

      if pixW < 1 { pixW = 1 } else if pixW > maxSide { pixW = maxSide }
      if pixH < 1 { pixH = 1 } else if pixH > maxSide { pixH = maxSide }

      return pixW, pixH
  }
  ```
* **Acceptance Criteria:** Fuzzing SVG rasterization with malformed inputs (`NaN`, `Inf`, negative numbers) returns clean errors without panicking.

---

#### [DET-01] Non-Deterministic PDF Byte Output via Un-pinned System `time.Now()`
* **Severity:** Medium  
* **Files & Line Numbers:** `internal/pdf/pdf.go` (L611-L616), `internal/convert/convert.go` (L70-L76)  
* **Root Cause:** When `Request.Now` is nil, `convert.Request.now()` falls back to system `time.Now()`, generating different PDF byte hashes (`CreationDate`) on consecutive runs.  
* **10/10 Solution:** Pin an explicit deterministic fallback epoch:
  ```go
  var DefaultDeterministicTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

  func (r *Request) now() time.Time {
      if r != nil && r.Now != nil {
          return r.Now()
      }
      return DefaultDeterministicTime
  }
  ```
* **Acceptance Criteria:** Re-running conversion produces byte-for-byte identical output files (`sha256sum` equality).

---

#### [PERF-02] O(N^2) Repetitive Page Index & Bounds Sweeps in Layout Painting
* **Severity:** Medium  
* **Files & Line Numbers:** `internal/layout/paint.go` (L89, L101, L120)  
* **Root Cause:** `validatePaintPageIndices` sweeps the full list of `res.Ops` 3 separate times during `PaintContext` execution.  
* **10/10 Solution:** Merge page index bounds checking directly into single-pass traversal inside `pageBuckets`:
  ```go
  func pageBuckets(ops []Op, contentH float64) ([]int, []int, error) {
      maxPage := 0
      pageOf := make([]int, len(ops))
      for idx := range ops {
          if ops[idx].Fixed {
              continue
          }
          p, ok := checkedFlowPageOfY(ops[idx].Y, contentH)
          if !ok {
              return nil, nil, fmt.Errorf("layout: op %d out of page range (Y=%v)", idx, ops[idx].Y)
          }
          pageOf[idx] = p
          if p > maxPage {
              maxPage = p
          }
      }
      return pageOf, counts, nil
  }
  ```
* **Acceptance Criteria:** Multi-pass bounds sweeps reduced from 3 to 1; CPU time decreases on large display list documents.

---

### Track 4: Go Code Style, Idioms, Lint Cleanup & Ergonomics

#### [LINT-01 & LINT-02] Broad `nolint` Complexity Directives in Header/Footer Rendering
* **Severity:** High / Medium  
* **Files & Line Numbers:** `internal/convert/hf.go` (L264-L280, L508-L525, L658-L680)  
* **Root Cause:** Monolithic functions suppress complexity linters (`//nolint:gocognit,cyclop,funlen,lll`) instead of splitting single-responsibility sub-functions.  
* **10/10 Solution:** Decompose `drawHeadersFooters` into small, focused sub-functions:
  ```go
  func drawHeadersFootersResult(ctx context.Context, hfCtx HeaderFooterDrawContext) hfDrawResult {
      cache := newHFCache()
      var res hfDrawResult
      for idx, page := range hfCtx.Doc.Pages() {
          if err := ctx.Err(); err != nil {
              return res
          }
          drawSinglePageHF(ctx, pageContext{
              page: page, pageNum: idx + 1, totalPages: hfCtx.Doc.PageCount(),
              cache: cache, hfCtx: hfCtx,
          }, &res)
      }
      return res
  }

  func drawSinglePageHF(ctx context.Context, pCtx pageContext, res *hfDrawResult) {
      drawHeaderBand(ctx, pCtx, res)
      drawFooterBand(ctx, pCtx, res)
  }
  ```
* **Acceptance Criteria:** All `//nolint:gocognit,cyclop,funlen,lll` directives in `hf.go` are deleted; cyclomatic complexity $\le 10$ for all functions.

---

#### [STATE-01 & STATE-02] Package-Level Globally Mutable `var` Maps
* **Severity:** High  
* **Files & Line Numbers:** `internal/cli/flags.go` (L27, L31), `internal/convert/hf.go` (L76), `internal/css/values.go` (L594), `internal/html/html.go` (L348), `internal/pdf/shape.go` (L160)  
* **Root Cause:** Global `var` maps (`flagTable`, `knownPlaceholders`, `namedColorTable`, `autoClose`, `arabicForms`) are globally mutable, risking runtime mutations and data races under concurrent access.  
* **10/10 Solution:** Replace package-level `var` maps with `sync.OnceValue` accessors or `switch` dispatch:
  ```go
  // internal/css/values.go
  var getNamedColorTable = sync.OnceValue(func() map[string][3]int {
      return map[string][3]int{
          "black": {0, 0, 0},
          "white": {255, 255, 255},
      }
  })

  func LookupNamedColor(name string) ([3]int, bool) {
      rgb, ok := getNamedColorTable()[name]
      return rgb, ok
  }

  // internal/convert/hf.go
  func isKnownPlaceholder(token string) bool {
      switch token {
      case "[page]", "[frompage]", "[topage]", "[isodate]", "[time]", "[date]", "[sitepage]", "[sitepages]":
          return true
      default:
          return false
      }
  }
  ```
* **Acceptance Criteria:** `var` maps removed; `nolint:gochecknoglobals` comments deleted; thread safety guaranteed under `go test -race`.

---

#### [ENUM-01 & ENUM-02] Unsafe Zero-Values for Custom Enum Types
* **Severity:** High / Medium  
* **Files & Line Numbers:** `internal/load/load.go` (L75-L79), `internal/cli/cli.go` (L85-L89), `internal/settings/settings.go` (L21-L24)  
* **Root Cause:** `KindHTTP = 0` and `flagBool = 0` make uninitialized zero-value structs (`Resource{}`) default to active operational kinds (HTTP), masking initialization errors.  
* **10/10 Solution:** Introduce explicit zero-value sentinels:
  ```go
  type Kind int
  const (
      KindUnknown Kind = iota // 0 is safe uninitialized sentinel
      KindHTTP
      KindFile
      KindInline
  )
  func (k Kind) IsValid() bool {
      return k >= KindHTTP && k <= KindInline
  }
  ```
* **Acceptance Criteria:** `Resource.Validate()` rejects `KindUnknown`; switch statements handle zero-values safely.

---

## 4. 10/10 Execution Roadmap & Action Plan

To transition `gowkhtmltopdf` from 7.5/10 to a certified **10/10 Architecture & Code Base**, execute the following 5 phase plan:

```
+-----------------------------------------------------------------------------------+
| Phase 1: Type Safety & Concurrency Core (CONC-01, STATE-01, STATE-02, CNV-01)    |
| - Add sync.RWMutex to pdf.Document & pdf.Registry                                 |
| - Replace all global var maps with sync.OnceValue or zero-alloc switch statements |
| - Implement sealed Request interface hierarchy (PDFRequest vs ImageRequest)       |
+-----------------------------------------------------------------------------------+
                                         |
                                         v
+-----------------------------------------------------------------------------------+
| Phase 2: Loader, Context & Security Ceiling (LOAD-01, LOAD-02, CTX-01, CTX-02)   |
| - Remove initErr & duplicate policy fields from load.Loader; fail-fast NewLoader  |
| - Inject pre-I/O ctx.Err() checks & enforce HardMaxBodyCap (1 GiB) limit          |
| - Unify nil context handling with canonical errs.ErrNilContext                    |
+-----------------------------------------------------------------------------------+
                                         |
                                         v
+-----------------------------------------------------------------------------------+
| Phase 3: Layout Engine & DOM Integrity (LAYOUT-01, DOM-01, DOM-02, HF-01, TOC-01) |
| - Refactor paintOpOnPage into pagePainter struct methods                          |
| - Fix benchmarkIslandRoot DOM ancestor context & atomic child creation            |
| - Move HF relative link resolution to layout build time; isolate TOC display lists|
+-----------------------------------------------------------------------------------+
                                         |
                                         v
+-----------------------------------------------------------------------------------+
| Phase 4: Performance, Bounds & Allocations (MEM-01, PERF-01, BND-01, DET-01)      |
| - Preserve pooled buffer capacity in flateBytes                                   |
| - Implement sync.Pool for supersampled pixel buffers in imageout                  |
| - Harden SVG float parsing (NaN/Inf bounds checks) & pin deterministic time       |
+-----------------------------------------------------------------------------------+
                                         |
                                         v
+-----------------------------------------------------------------------------------+
| Phase 5: Lint Elimination & Runtime Certification (LINT-01, LINT-02, ERGO-01)     |
| - Eliminate all broad //nolint complexity suppressions in hf.go                   |
| - Run complete test suite, benchmark suite, race detector, & golangci-lint        |
| - Verify 10/10 score target achieved across all tracks                            |
+-----------------------------------------------------------------------------------+
```

---

## 5. Certification Checklist for 10/10 Score

- [ ] **Type Safety:** `PDFRequest` and `ImageRequest` exist as distinct, sealed types; untyped request union eliminated.
- [ ] **Fail-Fast Loader:** `NewLoader` returns `(*Loader, error)`; `initErr` deleted; zero duplicate ACL fields.
- [ ] **Context Cancellation:** Every I/O operation checks `ctx.Err()` prior to execution; `errs.ErrNilContext` shared repository-wide.
- [ ] **DOM Ownership:** Island trees maintain complete `<head>` and wrapper hierarchy; headings remapped to canonical DOM nodes.
- [ ] **Thread Safety:** `pdf.Document` and `pdf.Registry` protected by `sync.RWMutex`; `go test -race ./...` clean.
- [ ] **Zero Global State:** No package-level `var` maps exist; all static tables use `sync.OnceValue` or `switch`.
- [ ] **Clean Lint Directives:** Zero broad `//nolint:gocognit,cyclop,funlen` directives in `hf.go` or convert pipeline.
- [ ] **Allocation Efficiency:** Supersample pixel buffers pooled via `sync.Pool`; `flateBytes` buffer capacity preserved.
- [ ] **Deterministic PDF:** Conversion returns byte-identical PDF outputs (`sha256sum`) across test runs.
