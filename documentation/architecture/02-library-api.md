# Public library API

## 1. Responsibility & position in the pipeline

The root package `gowkhtmltopdf` is the **idiomatic pure-Go library surface**
for the conversion engine. It is the "wave 2" library API described by
[phase 08 of the canonical plan](../../plans/0.1.0/00-canonical-pure-go-rewrite.md) and
wraps the wkhtmltopdf lifecycle (`pdf.h` / `image.h`, without any C). It has
two jobs:

1. **Configure** a conversion using either wkhtmltopdf-compatible dotted
   setting strings (`GlobalSettings.Set("size.pagesize", "A4")`) or
   compile-time-discoverable typed builders (`PdfGlobalOptions`,
   `PDFRequest`, `ImageRequest`).
2. **Drive** the engine: `Converter.Convert(ctx)` reduces to building an
   internal `convert.Request` and handing it to `convert.Run` (PDF) or
   `imageout.RunRequest` (image). All output is captured in memory.

In the pipeline `load → parse → style → layout → paginate → paint → write`
(see [../architecture.md](../architecture.md)), the library API sits **before**
the pipeline: it produces the request contract (`Request`) and receives the
finished bytes. It never touches parsing, layout, or PDF writing itself.

The module boundary is the enforcement point of the project's core product
rule — *pure Go, no cgo, no third-party PDF/HTML/CSS API*: `api.go` imports
only the Go standard library plus internal packages, while the module's direct
dependency allowlist is documented in `go.mod`; redistribution archives should
retain the upstream notices for those dependencies and bundled fonts.

## 2. Package / file map

The root package holds exactly three Go files.

| File | Responsibility | Approx. lines |
|------|----------------|---------------|
| `api.go` | All exported types and functions: settings wrappers, Converter / ImageConverter, typed request API, logging/progress hooks, deep-copy helpers, sentinel errors | 960 |
| `api_test.go` | End-to-end and unit tests for every public surface (round trips, deep copy, cancellation, callbacks, PNG/JPEG decode, line-log severity) | 601 |
| `doc.go` | Package documentation: quick start, dotted settings, local-file ACL pair, in-memory HTML source kind, thread-safety contract | 65 |

There are no other files in the package; example programs live under
`examples/pdf` and `examples/image` and use the public API end-to-end.

### Direct internal dependencies used by the root package

| Import | What the library uses it for |
|--------|------------------------------|
| `internal/convert` | `Request`, `NewPDFRequest`, `NewImageRequest`, `PDFRequest`/`ImageRequest` typed adapters, `Run` (via `RunTypedPDF`) |
| `internal/imageout` | `RunRequest` (image pipeline sink) |
| `internal/settings` | `PdfGlobal`, `PdfObject`, `ImageGlobal`, `HeaderFooter`, `PostItem`, `DefaultPdfGlobal/DefaultPdfObject/DefaultImageGlobal`, `PdfGlobalOptions`, `ApplyImageKey` |
| `internal/line` | `SeverityOf` — log-line severity grammar for the `lineLog` classifier |
| `internal/errs` | `ErrNilContext` canonical sentinel (re-exported as `ErrNilContext`) |

`context`, `bytes`, `errors`, `fmt`, `io`, `strings`, `time` (stdlib) round
out the import list; `time` exists solely for the `Now func() time.Time`
determinism hook on the typed request API.

## 3. Key types, functions & entry points

Exact locations in `api.go` unless another file is named.

### 3.1 Version banner

| Symbol | Purpose | Location |
|--------|---------|----------|
| `const LibraryVersion = "0.12.7-dev"` | wkhtmltopdf release this library tracks (upstream 0.12.x compatibility surface) | api.go:24 |
| `func Version() string` | Returns `"0.12.7-dev (gowkhtmltopdf pure-go)"` | api.go:50 |

### 3.2 Sentinel errors (matchable with `errors.Is`)

| Sentinel | Meaning |
|----------|---------|
| `ErrNoPageObjectsAdded` | PDF `Convert` without any `AddObject` |
| `ErrEmptyHTML` | `ConvertHTML` given an empty document |
| `ErrNoInputPageAdded` | Image `Convert` with neither `Page` nor `InlineHTML` |
| `ErrNilConverter` / `ErrNilImageConverter` | Method on a nil converter receiver |
| `ErrNilGlobalSettings` / `ErrNilObjectSettings` / `ErrNilImageSettings` | `Set`/`Get` on nil settings wrappers |
| `ErrNilContext` | Re-exported alias of `errs.ErrNilContext` (api.go:46) |

All are declared in one `var` block at api.go:27-48.

### 3.3 Settings wrappers

| Symbol | Purpose | Location |
|--------|---------|----------|
| `type GlobalSettings struct { g settings.PdfGlobal }` | Public wrapper keeping the internal settings type out of the public surface | api.go:61 |
| `NewGlobalSettings() *GlobalSettings` | wkhtmltopdf-compatible defaults from `settings.DefaultPdfGlobal()` (A4 portrait, 10 mm margins, background on) | api.go:161 |
| `GlobalSettings.Set(name, value string) error` | Dotted-key write; wraps errors as `global set %q: %w`; unknown names error | api.go:168 |
| `GlobalSettings.Get(name string) (string, bool)` | Canonical string read via the same key table; `ok=false` for unknown or non-scalar keys | api.go:185 |
| `type ObjectSettings struct { o settings.PdfObject }` | Per-page wrapper (input page + load/web/header/footer overrides) | api.go:196 |
| `NewObjectSettings() *ObjectSettings` | `settings.DefaultPdfObject()` — local-file block on by default | api.go:204 |
| `ObjectSettings.Set` / `.Get` | Dotted object keys (`page`, `load.*`, `web.*`, `header.*`, `footer.*`); errors wrapped as `object set %q: %w` | api.go:211 / 224 |
| `ObjectSettings.SetPage(page string) *ObjectSettings` | Sets input page (path, URL, `inline:…`, `data:…`); chainable | api.go:234 |
| `ObjectSettings.SetBody(html []byte, base string) *ObjectSettings` | In-memory document source kind: clears `Page`, copies bytes into `Load.InlineHTML`, sets `Load.InlineBase`; no URL guessing | api.go:248 |

`SetBody` copies the caller's byte slice at the boundary (see
`cloneBytes`), so later caller mutation cannot corrupt the object.

### 3.4 Typed PDF builder

| Symbol | Purpose | Location |
|--------|---------|----------|
| `type PdfGlobalOptions` | Typed builder alternative to string `Set` | api.go:68 |
| `NewPdfGlobalOptions() *PdfGlobalOptions` | Starts from engine defaults | api.go:73 |
| `WithPageSize / WithMargins / WithTitle / WithCopies / WithOutline / WithSmartShrinking / WithBackground / WithCompression / WithResolveRelativeLinks` | One fluent setter per common option; nil-receiver safe | api.go:77-149 |
| `Build() *GlobalSettings` | Independent settings snapshot (slices/maps cloned) | api.go:151 |

The underlying implementation is `settings.PdfGlobalOptions`
(`internal/settings/options.go:6`), where `Build` (options.go:73) deep-copies
`ExcludeFromOutline`, `FontPaths`, and `Ignored` so the builder can be reused
safely.

### 3.5 PDF converter

```go
type Converter struct {
    global  *GlobalSettings
    objects []*ObjectSettings
    output  []byte
    OnInfo, OnWarn, OnError func(string)   // log lines by severity
    OnPhase    func(string)                // "Loading pages (1/1)" …
    OnProgress func(int)                   // 0-100
}
```

| Symbol | Purpose | Location |
|--------|---------|----------|
| `NewConverter() *Converter` | Default global settings, no objects | api.go:290 |
| `Converter.Global() *GlobalSettings` | Lazy-initializes `global` if nil | api.go:295 |
| `Converter.AddObject(s *ObjectSettings) *Converter` | Deep-copies `s.o` via `clonePdfObject`; nil pointers ignored; chainable | api.go:311 |
| `Converter.AddHTML(page []byte, baseURL string) *Converter` | `SetBody` sugar for in-memory documents | api.go:416 |
| `Converter.Convert(ctx context.Context) error` | Builds `convert.NewPDFRequest(...)` and runs `executePDF`; stores output; errors also delivered to `OnError` | api.go:426 |
| `Converter.Output() []byte` | Copy of last successful result (nil before first Convert) | api.go:463 |
| `ConvertHTML(ctx, html []byte, global *GlobalSettings) ([]byte, error)` | One-shot helper: optional cloned global, `SetBody(html, "")`, `Convert`, `Output` | api.go:477 |

### 3.6 Image converter

```go
type ImageConverter struct {
    global      *GlobalSettings
    image       settings.ImageGlobal
    object      *ObjectSettings   // single, most-recent AddObject page
    output      []byte
    initialized bool              // lazy-init guard for zero-value receivers
    OnInfo, OnWarn, OnError func(string)
}
```

| Symbol | Purpose | Location |
|--------|---------|----------|
| `NewImageConverter() *ImageConverter` | 1024 px smart-width viewport, PNG output; `Object()` valid immediately | api.go:525 |
| `ImageConverter.Set(name, value string) error` | Image-key writes via `settings.ApplyImageKey`; `background`/`web.background` alias into shared `Global.Background` | api.go:539 |
| `ImageConverter.Global() *GlobalSettings` | Only `enablelocalfileaccess` and `allow` influence image conversion (loader ACL) | api.go:555 |
| `ImageConverter.AddObject(page string) *ImageConverter` | Registers the single page to render (first page only) | api.go:568 |
| `ImageConverter.Object() *ObjectSettings` | Per-page load options (e.g. unblocking local files); always valid | api.go:586 |
| `ImageConverter.Convert(ctx) error` | Rejects empty page; clones object; builds `convert.NewImageRequest`; runs `executeImage` | api.go:617 |
| `ImageConverter.Output() []byte` | Encoded PNG/JPEG bytes, copied | api.go:648 |
| `ImageConverter.ensureDefaults()` | Lazy-init so a `&ImageConverter{}` zero value still works | api.go:596 |

Note the asymmetry: `ImageConverter` has no `OnPhase`/`OnProgress` — image
rendering is single-shot rasterization; only the PDF path reports phase/
progress (`convertHooks.progress()` is only attached in `executePDF`).

### 3.7 Shared executor hooks

| Symbol | Purpose | Location |
|--------|---------|----------|
| `type convertHooks` | Holds the optional callbacks shared by both converter kinds | api.go:663 |
| `convertHooks.lineLog() *lineLog` | Adapter from `io.Writer` (engine log sink) to info/warn/error callbacks | api.go:669 |
| `convertHooks.progress() func(string, int)` | Returns nil when no phase/progress callback set; forwards otherwise | api.go:677 |
| `convertHooks.executePDF(ctx, req) ([]byte, error)` | Sets `req.Output = &bytes.Buffer`, calls `convert.Run`, reports failures to `OnError` | api.go:697 |
| `convertHooks.executeImage(ctx, req) ([]byte, error)` | Sets `req.Output = &bytes.Buffer`, calls `imageout.RunRequest` | api.go:720 |
| `type lineLog` + `func (w *lineLog) Write(payload []byte)` | Buffers lines, splits on `\n`, classifies with `line.SeverityOf`, dispatches to callbacks | api.go:918 / 925 |

### 3.8 Typed request API (Phase 2 / P1-8 era)

| Symbol | Purpose | Location |
|--------|---------|----------|
| `type PDFRequest` | Public typed request: `Global`, `Objects`, `Now func() time.Time`, `Output io.Writer`, `OutlineOutput io.Writer` | api.go:746 |
| `type ImageSettings` + `NewImageSettings` | Type-safe image settings wrapper (`format`, `width`, `quality`, …); `background` set tracking | api.go:755 / 762 |
| `type ImageRequest` | `Global`, `Image`, `Object`, `Now`, `Output` — single object only | api.go:801 |
| `PDFRequest.toRequest() *convert.PDFRequest` | Clones global plus every object; returns the internal typed adapter | api.go:809 |
| `ImageRequest.toRequest() *convert.ImageRequest` | Clones settings; propagates `background` into `PdfGlobal.Background` when set | api.go:836 |
| `RunPDF(ctx, req *PDFRequest) error` | One-shot typed PDF: `convert.RunTypedPDF(ctx, req.toRequest(), nil, nil)` | api.go:874 |
| `RunImage(ctx, req *ImageRequest) error` | One-shot typed image: `imageout.RunRequest(ctx, req.toRequest().ToRequest(), nil)` | api.go:894 |

`ImageSettings` deliberately lacks a `Web` struct and PDF-only keys, so an
image request cannot accidentally carry PDF settings — this is the
compile-time-safety guarantee the shared internal `Request` union cannot give
(see [../documentation/architecture.md](../architecture.md) and
`internal/convert/request.go`).

## 4. Data & control flow

### 4.1 PDF conversion via `Converter` (most common path)

```text
caller
  ├─ NewConverter()                       -> defaults (api.go:290)
  ├─ Global().Set("size.pagesize", "A4")  -> settings.PdfGlobal.Set (reflect key table)
  ├─ NewObjectSettings().SetPage("r.html")
  ├─ AddObject(obj)                       -> clonePdfObject deep copy (api.go:311/326)
  └─ Convert(ctx)                         (api.go:426)
        ├─ guard: nil receiver, nil global, >=1 object (else ErrNoPageObjectsAdded)
        ├─ req := convert.NewPDFRequest(c.global.g, objects, nil, nil)   (convert.go:114)
        ├─ executePDF(ctx, req)                                           (api.go:697)
        │     ├─ req.Output = &bytes.Buffer            (in-memory sink; no temp file)
        │     └─ convert.Run(ctx, req, h.lineLog(), h.progress())    (convert.go:310)
        │           ├─ req.ValidatePDF()               (sentinel errors, page-size mirror)
        │           ├─ loader := load.NewLoaderWithError(req.Global.Load)   (ACL/security)
        │           ├─ pdf.DefaultFont() + loadFontRegistry(req.Global)
        │           └─ render.Run(ctx, &pdfPipeline{...})   (HF/TOC/links/outline/paint)
        └─ c.output = buf.Bytes()
Output() -> copy of c.output (api.go:463)
```

Engine log lines flow back through the `io.Writer` (a `lineLog`): each
`\n`-terminated line is classified by `line.SeverityOf` (internal/line:45)
using the `info: `/`warning: `/`error: ` marker grammar, and dispatched to
`OnInfo`/`OnWarn`/`OnError` (api.go:925-960). Phase/progress callbacks are
fed by `convert.Run`'s `progress func(phase string, percent int)`.

### 4.2 Image conversion

```text
NewImageConverter() -> Set("width","200") / Object().SetBody(html, base)
AddObject(path) or Object().SetBody(html, base)
Convert(ctx) (api.go:617)
  ├─ guard: Page=="" && no InlineHTML -> ErrNoInputPageAdded
  ├─ obj := clonePdfObject(c.object.o)
  ├─ req := convert.NewImageRequest(c.global.g, c.image, []settings.PdfObject{obj}, nil)  (convert.go:125)
  └─ executeImage(ctx, req) (api.go:720)
        ├─ req.Output = &bytes.Buffer
        └─ imageout.RunRequest(ctx, req, h.lineLog())  (imageout.go:1094)
              ├─ req.ValidateImage()  (Image must be non-nil)
              └─ render display list -> PNG/JPEG encode into req.Output
Output() -> encoded bytes
```

### 4.3 Typed one-shot path

```text
PDFRequest{Global, Objects, Now, Output, OutlineOutput}
RunPDF(ctx, req) (api.go:874)
  └─ convert.RunTypedPDF(ctx, req.toRequest(), nil, nil)   (request.go:70)
        └─ convert.Run(ctx, req.ToRequest(), log, progress)  (shared engine seam)

ImageRequest{Global, Image, Object, Now, Output}
RunImage(ctx, req) (api.go:894)
  └─ imageout.RunRequest(ctx, req.toRequest().ToRequest(), nil)
```

`PDFRequest.toRequest()` clones the public settings into
`convert.PDFRequest`, which in turn has a `ToRequest()` producing the shared
`convert.Request` union — the public surface is thus two thin adapters deep,
and the engine stays CLI-independent (the CLI builds the same `Request`
through `internal/app.RunPDF`).

### 4.4 Cross-domain interfaces

| Boundary | Contract |
|----------|----------|
| `api.go` → `internal/convert` | `convert.NewPDFRequest` / `NewImageRequest` / `Request` / `RunTypedPDF`. The library never touches `cli.Command`. |
| `api.go` → `internal/imageout` | `imageout.RunRequest(ctx, *convert.Request, io.Writer)` — the image engine consumes the same shared request. |
| `api.go` → `internal/settings` | Typed settings structs, defaults, `ApplyImageKey`, dotted `Set/Get` via reflection (`reflect.go`). |
| `api.go` → `internal/line` | `line.SeverityOf` — the only severity grammar; `api.go` owns the callback mapping (test: `TestLineLogSeverity`). |
| `convert.Request` → engine | `Validate/ValidatePDF/ValidateImage` then `render.Run`; `Request.now()` injects `Now` for deterministic metadata (convert.go:68). |

## 5. Cross-package dependencies

### What the root package imports (and why)

| Import | Reason | Direction rule |
|--------|--------|----------------|
| `internal/convert` | Request construction and PDF pipeline entry | **root → convert**; convert never imports root |
| `internal/imageout` | Image pipeline entry | **root → imageout** |
| `internal/settings` | Typed settings model, defaults, key tables | **root → settings** |
| `internal/line` | Severity grammar for log callbacks | **root → line** |
| `internal/errs` | Canonical `ErrNilContext` sentinel | **root → errs** |
| stdlib only (`bytes`, `context`, `errors`, `fmt`, `io`, `strings`, `time`) | Buffering, cancellation, errors.Is matching, writer sinks, determinism hook | — |

There are **no third-party imports** — this is the module rule "pure Go, no
cgo, stdlib + internal only" made visible in one import block. Because
`internal/` is Go's enforced import restriction, external consumers can only
touch the root package; everything else is unreachable by design.

### What depends on the root package

- `cmd/gowkhtmltopdf` and `cmd/gowkhtmltoimage` (CLI binaries) — the CLI path
  is independent: it goes through `internal/cli` → `internal/app.RunPDF`,
  so **nothing in `cmd/` imports the root package**. The root package is a
  parallel consumer of the same engine seam.
- `examples/pdf` and `examples/image` — demonstration programs.
- External embedders (the intended user of the library).

### Import-direction invariant

`api.go` never imports `internal/cli`; `internal/cli` never imports the root
package. Both funnel into `internal/convert.Request`, which is documented as
"the PDF pipeline input, independent of the CLI parser" (convert.go:44). This
single-seam design is what allows a future C ABI (phase 08, optional) and any
other frontend to reuse the engine without touching CLI code.

## 6. Design decisions & trade-offs

1. **Two configuration dialects.** The string-based dotted `Set`/`Get`
   mirrors the CLI surface 1:1, so library users and CLI users share the same
   mental model and the same key tables (`reflect.go`). The typed
   `PdfGlobalOptions` and `PDFRequest`/`ImageRequest` add compile-time
   discoverability for the common cases. The cost is two parallel paths that
   must be kept in sync — `ApplyImageKey` (reflect.go:874) exists precisely to
   reconcile image keys across both faces.

2. **Ownership boundary via deep copy.** `AddObject` clones the whole
   `PdfObject` (maps: `CustomHeaders`, `Cookies`, `Replace`, `Ignored`; slices:
   `Post`; bytes: `InlineHTML`) so callers can mutate their settings object
   after `AddObject` (api.go:326-414). This is validated by a dedicated
   goroutine-mutation test (`TestConverterAddObjectSnapshotIsIndependentUnderMutation`),
   designed for `go test -race`. The same cloning pattern protects
   `ConvertHTML`'s global clone and `Output()`'s returned slice.

3. **In-memory output, no temp files.** `executePDF`/`executeImage` set
   `req.Output` to a `bytes.Buffer`. The phase-08 design note originally
   called for converting to a temp file and reading it back; the memory-buffer
   path replaced that, simplifying lifecycle and eliminating temp-file failure
   modes (api.go:697-744).

4. **Nil-receiver defensive programming.** Nearly every method nil-guards and
   either returns a sentinel error (`ErrNilConverter`, `ErrNilGlobalSettings`,
   …) or a safe zero value, and setters ignore nil receivers so chains cannot
   panic. This is unusual for Go APIs but matches the library's "embedding in
   servers must not panic" posture.

5. **Determinism hook.** `Request.Now func() time.Time` (convert.go:55) lets
   typed callers inject a stable clock shared by PDF metadata and
   header/footer substitutions, making byte-stable output possible for fixed
   input. The `Converter` surface deliberately omits it — the string-based
   path stays wkhtmltopdf-simple; determinism is a typed-API concern.

6. **Log severity grammar centralized.** `lineLog` does not substring-scan
   messages; it classifies by `line.SeverityOf` markers, so the grammar lives
   in one package (`internal/line`) and the mapping lives in `api.go`
   (api.go:918-969, pinned by `TestLineLogSeverity`).

7. **Security posture is non-negotiable in the API.** Local-file access is
   blocked by default (documented in doc.go and enforced by the loader ACL);
   the library requires the *same* two-step opt-in as the CLI:
   `Global().Set("enablelocalfileaccess","true")` **and**
   `obj.Set("load.blocklocalfileaccess","false")`. `SetBody`/`ConvertHTML`
   bypass URL guessing entirely (an explicit source kind), so in-memory HTML
   cannot be coerced into a file read or SSRF through path tricks.

8. **Dropped phase-08 surface.** The plan listed `HttpErrorCode() int`
   (placeholder 0) on `Converter`. It was not carried into the final surface:
   HTTP-status → exit-code mapping lives in `internal/settings/httperror.go`
   and is used by `internal/cli` only (cli.go:518-523). Library failures are
   reported through wrapped errors and `OnError` instead. This is a deliberate
   API-consistency decision worth re-evaluating if embedders need
   wkhtmltopdf-compatible HTTP error codes.

## 7. Notable patterns & invariants

- **Dotted settings with fail-fast validation.** Unknown keys error at
  `Set` time, not at `Convert` time; `Get` returns `(value, false)` for
  unknown/non-scalar keys. Wrapping preserves `errors.Is` on sentinels
  (`global set %q: %w`, `object set %q: %w`, `image set %q: %w`).
- **One converter per conversion.** Thread-safety contract (doc.go): a
  `Converter`/`ImageConverter` is *not* safe for concurrent `Convert`; create
  one per conversion or guard with a mutex. Settings may be read while idle.
- **`ensureDefaults` lazy-init.** A zero-value `ImageConverter` (or one built
  without `NewImageConverter`) self-initializes on first use, so `&ImageConverter{}`
  never panics (api.go:596).
- **Copy-out on every boundary.** `SetBody` copies input bytes; `AddObject`
  copies the object; `Output` copies result bytes; `ConvertHTML` clones the
  supplied global. The rule: *no slice or map the caller owns ever escapes to
  the engine, and no slice the engine produced ever escapes to the caller
  directly.*
- **Progress terminates at 100.** `OnProgress` receives a final 100
  (asserted by `TestConverterCallbacks`), mirroring the wkhtmltopdf CLI
  progress contract.
- **Shared `Request` union.** All frontends (CLI, library, typed API) reduce
  to one `convert.Request`; validation enforces mode invariants at the
  boundary (`ValidatePDF` rejects `Image != nil`; `ValidateImage` requires it).
- **Extension points are intentionally small.** New settings keys require
  entries in `internal/settings` reflect tables, not this package; the root
  package is a thin, stable façade. Nothing here is a plugin framework.

## 8. Security considerations

The library surface is the primary trust boundary for embedders that accept
untrusted HTML (typical for web-service report generation):

- **ACL default-deny on local files.** Gated in the loader
  (`internal/load`), not the API — but the API documents the required opt-in
  pair (doc.go; NewObjectSettings doc at api.go:204). `SetBody` with an empty
  base makes linked local assets unresolvable rather than silently readable.
- **`ConvertHTML` does not relax ACL.** Linked local CSS/images still need the
  enable + unblock pair; the helper clones global settings but never widens
  access (api.go:477-505).
- **Context cancellation threads into every load.** `ctx` is passed through
  the `Request` into `convert.Run` / `imageout.RunRequest`; cancelling aborts
  an in-flight conversion (tested by `TestConvertContextCancel`). This is the
  library's primary DoS bound for server use.
- **Engine limits remain in force.** Per-request caps (`maxConversionObjects`
  = 10 000, `maxConversionCopies` = 1 000, `maxConversionPages` = 100 000,
  `maxStylesheetRules` = 1 000 000 — convert.go:42-47) and loader timeouts
  apply identically to library and CLI calls; see
  [THREAT-MODEL.md](../THREAT-MODEL.md) section on "Many concurrent converts /
  huge pages | DoS (CPU/RAM) | Rate-limit + timeouts".
- **Error strings are safe to propagate.** Sentinels carry no file contents or
  headers; `OnError` receives pre-wrapped messages suitable for logs.

## 9. Testing & verification

`api_test.go` (root package, 601 lines) is the acceptance suite for the
public surface; it runs in the same package (`package gowkhtmltopdf` +
`//nolint:testpackage`) so it can assert on unexported state like
`conv.objects[0].o` and `obj.o.Load.InlineHTML`.

| Test | What it pins |
|------|--------------|
| `TestConvertPDFToBytes` | End-to-end PDF: `%PDF-` magic, `/Type /Page` present, fresh `Output()` is nil |
| `TestConvertHTMLHelper` | One-shot `ConvertHTML` with nil and non-nil globals |
| `TestRunPDFTypedRequest` / `TestRunImageTypedRequest` | Typed `RunPDF`/`RunImage` produce correct magic bytes (PDF / PNG signature) |
| `TestGlobalSettingsGetSetRoundTrip` + `assertGlobalDefaults` | Set/Get round trip for representative keys; **defaults pinned** (A4, Portrait, margin 10, background true); unknown keys fail; invalid lengths fail |
| `TestObjectSettingsGetSet` | Object keys round trip; unknown object key errors |
| `TestObjectSettingsSetBodyCopiesInput` | `SetBody` clones the caller buffer |
| `TestConverterAddObjectDeepCopiesNestedData` + `assertSnapshotUntouched` | Every map/slice field of `PdfObject` is snapshotted |
| `TestConverterAddObjectSnapshotIsIndependentUnderMutation` | Goroutine-mutation isolation (race-detector material; uses `sync.WaitGroup`, 1000 iterations each side) |
| `TestConvertContextCancel` | Pre-cancelled context fails `Convert` |
| `TestConverterCallbacks` | `OnPhase`, `OnProgress` (final 100), `OnInfo` all fire during a real conversion |
| `TestImageConverterPNG` / `TestImageConverterJPEG` | Decodes output with `image/png`/`image/jpeg`; verifies viewport width **and** actual pixel colour (`pixelAt` = `#336699`) |
| `TestImageConverterSetBody` | P2-04 InlineHTML source kind in image mode |
| `TestImageConverterNeedsPage` / `TestConverterNeedsObject` | Precondition errors |
| `TestLineLogSeverity` | Severity protocol: marker decides bucket, message content does not |
| `TestVersion` | Version banner contains `LibraryVersion` |

Broader validation: `make test` runs `go test ./...` across all internal
packages; golden fixtures under `internal/convert` and `internal/imageout`
exercise the produced bytes; `make lint` runs `go vet` / `gofmt`. Example
programs build and run as smoke tests (`go run ./examples/pdf`,
`go run ./examples/image`).

## 10. Known limitations, deferred items & open questions

- **No `HttpErrorCode()` on `Converter`.** Phase-08 checklist item dropped
  from the final surface; HTTP error codes remain CLI-only
  (`internal/settings/httperror.go`, internal/cli:518-523). Open question:
  should the library expose `HttpErrorCode` for embedders that must match
  wkhtmltopdf exit semantics?
- **Converter has no `Now` override.** Byte-stable output requires the typed
  `PDFRequest`/`ImageRequest` path (or `internal/convert` directly); the
  string-based `Converter` uses the wall clock. Documented asymmetry, not a
  bug — but callers seeking reproducibility must choose the typed API.
- **TOC as a first-class library object is not supported.** Per
  [documentation/library-api.md](../library-api.md): "TOC as a first-class
  CLI object is primarily a CLI feature; from the library, body pages with
  headings + global outline/TOC settings cover most report cases." `AddObject`
  objects carrying `IsTableOfContent=true` are not constructed by any public
  helper.
- **JavaScript / full-CSS input.** Out of scope by design (fidelity.md);
  `<script>` is stripped. Library callers passing JS-heavy SPA HTML get the
  server-rendered snapshot only — align expectations with
  [documentation/deferred.md](../deferred.md) (SPA URLs are
  explicitly the *last* priority workload).
- **One image page only.** `ImageConverter` renders the single most-recent
  `AddObject` page; multi-page image output is not part of the surface.
- **No C ABI yet.** Optional `wkhtmltopdf_*` cgo exports deferred pending
  consumer demand (phase-08 §8.4); the pure-Go surface is what ships today.
- **Compatibility is a living contract.** Anything the API accepts but the
  engine cannot honour degrades gracefully (ignored keys land in
  `PdfGlobal.Ignored`/`PdfObject.Ignored` via the Policy A key set; see
  [compatibility-matrix.md](../compatibility-matrix.md)).

## 11. Related documents

- [../architecture.md](../architecture.md) — package map + pipeline overview that this deep-dive expands
- [../library-api.md](../library-api.md) — user-facing quick start for the library (install, examples, ACL pair)
- [../overview.md](../overview.md) and [../README.md](../../README.md) — product positioning
- [../fidelity.md](../fidelity.md) — controlled-report fidelity tiers and explicit non-claims
- [../THREAT-MODEL.md](../THREAT-MODEL.md) — ACL, limits, DoS mitigations
- [../compatibility-matrix.md](../compatibility-matrix.md) — normative per-feature contract
- [../deferred.md](../deferred.md) — workload prioritisation (template-first, SPA-last)
- [../fonts.md](../fonts.md) — font discovery relevant when adding font-path settings
- [../../plans/0.1.0/00-canonical-pure-go-rewrite.md](../../plans/0.1.0/00-canonical-pure-go-rewrite.md) phase 8 (line 60) and [phase-08-library-api.md](../../plans/0.1.0/phases/phase-08-library-api.md) — the plan this surface implements
- [CHANGELOG.md](../../CHANGELOG.md) — release notes (library API entry, line 63)
- Sibling architecture deep-dives in this directory:
  - [01-entrypoints-cli.md](01-entrypoints-cli.md) — CLI path and the `internal/app` command adapters alongside this library
  - [03-settings.md](03-settings.md) — the settings model and dotted-key tables both CLI and library wrap
  - [08-convert-pipeline.md](08-convert-pipeline.md) — the `convert.Request`/`Run` engine both frontends share
  - [10-imageout-svg.md](10-imageout-svg.md) — the `imageout.RunRequest` sink used by `RunImage`/`ImageConverter`
