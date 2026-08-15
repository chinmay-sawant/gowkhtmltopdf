# Settings system & errors

## 1. Responsibility & position in the pipeline

The `internal/settings` package is the **single source of truth for every
user-facing knob** in gowkhtmltopdf: page geometry, margins, headers/footers,
TOC behaviour, load policy (proxy, ACL), web behaviour, image output, and the
inert-but-accepted wkhtmltopdf keys. It is the *settings half* of the
`cli → settings → convert/load/layout/imageout` data path, and it is the only
package that understands the **wkhtmltopdf dotted-name vocabulary**
(`margin.top`, `load.jsdelay`, `web.background`, `toc.captiontext`, …).

Its position in the pipeline is upstream of everything:

```text
argv / library calls
      │  (string keys + values, or typed builder)
      ▼
internal/settings ── Set / Get / typed structs ──► internal/cli (Command)
      │                                                │
      │  cli.Command carries PdfGlobal/PdfObject/      │
      │  ImageGlobal snapshots                         │
      ▼                                                ▼
internal/app (command→request adapters)          api.go (Converter wrappers)
      │                                                │
      ▼                                                ▼
internal/convert (Request: Global, Objects, Image) ──► internal/load (NewLoader)
      │                                                │
      ├──► internal/layout / internal/pdf              ▼
      └──► internal/imageout (mediaFor, imageLoad)   resource fetches honor
                                                      LoadGlobal/LoadPage policy
```

Nothing in the engine reads argv or library setters directly; everything
reads the typed settings structs (`settings.PdfGlobal`, `settings.PdfObject`,
`settings.ImageGlobal`) that this package defines, default-initializes, and
mutates through its descriptor tables.

A second, smaller package is in scope: `internal/errs`, which owns the shared
domain **sentinel errors** (`ErrNilContext`, `ErrNilLoader`, `ErrNilCommand`,
`ErrNilRequest`) re-exported by `api.go` and `internal/app/pdf.go`. It is a
leaf package with no imports beyond `errors`; its only job is to keep the
nil-guard error identity stable across the library boundary.

## 2. Package / file map

| File | Responsibility | Approx. lines |
|------|----------------|---------------|
| `internal/settings/settings.go` | Typed settings model: `PdfGlobal`, `PdfObject`, `ImageGlobal`, sub-structs (`Web`, `LoadGlobal`, `LoadPage`, `HeaderFooter`, `TableOfContent`, `Margin`, `Size`, `CropSettings`, `PostItem`), enums (`Orientation`, `ColorMode`, `LoadErrorHandling`, `MediaType`), `ParsePDFVersion` / `ParsePDFProfile` (profile parse delegates to `internal/pdfprofile`), wkhtml-compatible defaults, `ResolveMedia` | 449 |
| `internal/pdfprofile/profile.go` | Leaf: canonical profile tokens (`PDF/A-3a`, `PDF/UA-1`, `PDF/A-3a+PDF/UA-1`, `PDF/A-4`, `PDF/UA-2`, `PDF/A-4+PDF/UA-2`), alias `Parse`, `IsPDFA*` / `IsPDFUA*` | 140 |
| `internal/settings/reflect.go` | The descriptor engine: `keyTable`/`field` tables, dotted-key `Set`/`Get`, type-coercing setters (`setBool`, `setFloat`, `setInt`, `setUnitMm`, …), ignored-key tables (Policy A), `ApplyImageKey` | 881 |
| `internal/settings/getters.go` | `Get` methods on the three settings types; canonical string formatting helpers (`fmtBool`, `fmtFloat`, `fmtInt`, `fmtStrings`) | 37 |
| `internal/settings/options.go` | `PdfGlobalOptions` typed builder (`WithPageSize`, `WithMargins`, …) for the library API; independent-snapshot `Build()` | 93 |
| `internal/settings/pagesize.go` | Static ISO/ANSI page-size table in points; `ParsePageSize(name) (w, h, err)` | 64 |
| `internal/settings/unitreal.go` | `UnitReal` scalar with unit suffix (`10mm`, `1.5in`, `12pt`, `96px`, `100%`); `Points()` / `Mm()` conversion; `ErrInvalidUnitReal` | 94 |
| `internal/settings/httperror.go` | `HttpStatusError` (load failure carrying HTTP status) and `HttpErrorCode` (status → wkhtmltopdf exit code: 404→2, 401→3, else 1) | 41 |
| `internal/settings/doc.go` | Package doc: **Policy A (settings honesty)** — only engine-consumed options get typed fields; inert wkhtml keys sink into `Ignored` | 17 |
| `internal/settings/settings_test.go` | Defaults snapshots, dotted Set/Get round-trips, ignored-key acceptance, unknown-key errors, `UnitReal`/`PageSize`/enum parsing, HTTP exit codes, header/footer inheritance, `ResolveMedia`, `ApplyImageKey` | 648 |
| `internal/settings/options_test.go` | Typed builder produces an independent, correctly-populated snapshot | 32 |
| `internal/errs/errs.go` | Canonical sentinel errors shared across packages | 16 |

Total package size: ~2,372 lines including tests.

## 3. Key types, functions & entry points

### 3.1 The three settings structs (the payload)

| Type | Purpose | File:line |
|------|---------|-----------|
| `PdfGlobal` | PDF-mode global settings: page geometry, orientation, `PdfVersion` / `PdfProfile` (`settings.go:418`–`422`), grayscale, copies/collate, outline, title, margins, header/footer, TOC, background, load policy, font paths, `Ignored` sink | `settings.go:412` |
| `PdfObject` | One page/cover/toc object: `Page`, link flags, outline inclusion, object-level header/footer overrides (`HeaderSet`/`FooterSet`), `LoadPage`, `Web`, `UseOutline`, `Ignored` | `settings.go:484` |
| `ImageGlobal` | Image-mode (wkhtmltoimage) settings: width/height/quality, smart width, crop, format, transparency, `Web`, `LoadGlobal`, `Ignored` | `settings.go:564` |

Supporting sub-structs (all in `settings.go`):

| Type | Purpose | File:line |
|------|---------|-----------|
| `Margin` | Four page margins in millimetres | `settings.go:160` (approx.) |
| `Size` | Custom page dimensions in mm; `PageSize` name is dual-written with `PdfGlobal.PageSize` | `settings.go:173` (approx.) |
| `Web` | Engine-consulted web behaviour: `Images`, `PrintMediaType`/`MediaType`, `SimplifyDOM(+Profile)`, `PrintLinkUnderline` | `settings.go:187` |
| `LoadGlobal` | Shared load policy: `Proxy`, `Allow` (ACL prefixes), `EnableLocalFileAccess` | `settings.go:206` |
| `LoadPage` | Per-page load settings: zoom, block-local-access, load-error handling, auth, custom headers, cookies, POST items, media, timeout, `InlineHTML`/`InlineBase` | `settings.go:214` |
| `HeaderFooter` | Text/HTML header & footer: font, left/right/center, line, spacing, `HTMLURL`, `Replace` map | `settings.go:240` |
| `TableOfContent` | TOC object settings: font scale, indentation, dotted lines, caption, forward/back links, XSL | `settings.go:262` |
| `PostItem` | One urlencoded form field for POST loads | `settings.go:239` (approx.) |
| `CropSettings` | wkhtmltoimage crop box (Left/Top/Width/Height, -1 = unset) | `settings.go:579` |

### 3.2 Enums and parsers

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `Orientation` + `ParseOrientation` | portrait/landscape, case-insensitive | `settings.go:61` (approx.) / `settings.go:78` (approx.) |
| `ColorMode` + `ParseColorMode` | `color`/`grayscale` parse helper; engine stores only `PdfGlobal.Grayscale` (ponytail note in source) | `settings.go:27` (approx.) / `settings.go:50` (approx.) |
| `LoadErrorHandling` + `ParseLoadErrorHandling` | abort/skip/ignore | `settings.go:93` (approx.) / `settings.go:112` (approx.) |
| `MediaType` + `ResolveMedia(base, global Web, obj *Web) string` | screen/print/ignore resolution; print-media-type override wins, then object media-type, then global, then base | `settings.go:126` (approx.) / `settings.go:137` |
| `ParsePDFVersion` / `ParsePDFProfile` | Version: `""`/`1.4`/`1.7`/`2.0`. Profile: aliases (`a3a-ua1`, `a4-ua2`, …) → canonical tokens. Profile parse is `pdfprofile.Parse` | `settings.go:56` / `settings.go:76` |

### 3.3 Defaults (wkhtmltopdf `pdfsettings.cc` / `imagesettings.cc` compatible)

| Function | Notable defaults | File:line |
|----------|------------------|-----------|
| `DefaultMargins()` | 10 mm all sides (`defaultMarginMM = 10`) | `settings.go:166` (approx.) |
| `DefaultHeaderFooter()` | Arial, font size 12, spacing 0 | `settings.go:257` (approx.) |
| `DefaultTableOfContent()` | font scale 0.8, indentation `1em`, dotted lines true, caption "Table of Contents" | `settings.go:276` (approx.) |
| `DefaultPdfGlobal()` | A4 portrait, 1 copy collated, outline depth 4, compression on, smart shrinking on, background on, images on, margins 10 mm, resolve-relative-links on | `settings.go:459` |
| `DefaultPdfObject()` | external/local links on, include in outline on, use outline on, block-local-file-access true, load-error abort | `settings.go:542` |
| `DefaultLoadPage()` | `BlockLocalFileAccess: true`, `LoadErrorHandling: LoadErrorAbort` (mirrors `loadsettings.cc`) | `settings.go:555` |
| `DefaultImageGlobal()` | width 1024, quality 94, smart width on, crop = (-1,-1,-1,-1) | `settings.go:587` |

### 3.4 The descriptor engine (`reflect.go`)

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `field[T]` | One dotted key's apply/get pair for a settings type | `reflect.go:78` |
| `keyTable[T]` | The one descriptor table per settings type (`globalKeys`, `objectKeys`, `imageKeys`) | `reflect.go:84`, built at `reflect.go:395` |
| `sub[T, S]` / `subTable` | Adapts a sub-struct descriptor (HeaderFooter, TOC, Web, LoadPage) to a containing type; object header/footer keys additionally set `HeaderSet`/`FooterSet` | `reflect.go:88` / `reflect.go:406` |
| `setForKey` / `getForKey` | Lookup path: descriptor table → known-ignored set → error / not-found | `reflect.go:97` / `reflect.go:116` |
| `ignoredGlobalKeySet` / `ignoredObjectKeySet` | Immutable-by-convention tables of inert wkhtml keys (dpi, javascript, js-delay, log-level, …) | `reflect.go:136` / `reflect.go:170` |
| `setBool` / `setFloat` / `setInt` / `setString(Default)` | Type-coercing setters; booleans accept `""/true/1/yes/on` and `false/0/no/off` | `reflect.go:209` (setBool), `reflect.go:226` (setFloat, approx.), `reflect.go:236` (setInt, approx.) |
| `setUnitMm` / `marginSetter` | Parse a unit real, convert to mm, store; context-named conversion errors | `reflect.go:359` / `reflect.go:370` (approx.) |
| `setGrayscaleFromColorMode` | Maps `colormode` strings onto the shared `Grayscale` bool | `reflect.go:289` (approx.) |
| `registerGlobalVersionProfileKeys` | Wires `pdfversion` / `pdfprofile`. `Set("pdfprofile", "a3a-ua1")` stores `PDF/A-3a+PDF/UA-1`; `Get("pdfprofile")` returns that canonical token | `reflect.go:501` |
| `registerGlobalKeys` … `registerImageKeys` | The eight table-wiring functions composing `buildKeyTables()` | `reflect.go:828` (`buildKeyTables`) |
| `(g *PdfGlobal) Set` / `(o *PdfObject) Set` / `(g *ImageGlobal) Set` | Public dotted-key entry points; unknown keys error via `errUnknownSetting` | `reflect.go:855` / `reflect.go:861` / `reflect.go:866` |
| `(g *PdfGlobal) Get` / `(o *PdfObject) Get` / `(g *ImageGlobal) Get` | Canonical string read-back; accepted ignored keys return last Set value | `getters.go:13` (and siblings) |
| `ApplyImageKey(global, img, name, value)` | Routes image-mode keys: `background`/`web.background` alias to `PdfGlobal.Background`; everything else to `ImageGlobal.Set` | `reflect.go:874` |

### 3.5 Units, page sizes, HTTP errors

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `UnitReal` | Scalar + unit suffix (`mm cm m in pt px em rem ex ch %`) | `unitreal.go:20` |
| `ParseUnitReal(raw, impliedUnit)` | Suffix detection + `strconv.ParseFloat`; `ErrInvalidUnitReal` on failure | `unitreal.go:30` |
| `(u UnitReal) Points()` | PDF points via 96 px/in CSS ratio; `%` and font-relative units return `ok=false` (resolved by layout) | `unitreal.go:57` |
| `(u UnitReal) Mm()` | Millimetres (72/25.4 pt per mm) | `unitreal.go:85` (approx.) |
| `pageSizes` / `ParsePageSize(name)` | Fixed 23-entry ISO/ANSI table in points (A0–A6, B0–B6, C5E, Comm10E, DLE, Executive, Folio, Ledger, Legal, Letter, Tabloid); case-insensitive, empty → A4 | `pagesize.go:19` / `pagesize.go:51` |
| `HttpStatusError` | Load failure carrying HTTP status + URL; used by `internal/load` | `httperror.go:16` |
| `HttpErrorCode(status)` | 404 → 2, 401 → 3, else 1 (wkhtmltopdf `utilities.cc` convention) | `httperror.go:27` |
| `(e *HttpStatusError) HttpErrorCode()` | Lets `cli.ExitCode` detect the exit code via an interface check | `httperror.go:39` |

### 3.6 Typed builder

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `PdfGlobalOptions` | Value-type builder for the library API; immutable (each `With*` returns a copy) | `options.go:6` |
| `NewPdfGlobalOptions()` | Starts from `DefaultPdfGlobal()` | `options.go:11` |
| `WithPageSize/WithMargins/WithTitle/WithCopies/WithOutline/WithSmartShrinking/WithBackground/WithCompression/WithResolveRelativeLinks` | Compile-time-discoverable setters | `options.go:15`–`options.go:68` (approx.) |
| `WithPDFVersion` / `WithPDFProfile` | Normalize valid input to canonical tokens; invalid strings are stored as-is and fail later at `PolicyForGlobal` / `Validate` | `options.go:71` / `options.go:81` |
| `(o PdfGlobalOptions) Build()` | Independent snapshot (clones slices and `Ignored` map) | `options.go:106` |

### 3.7 Sentinel errors (`internal/errs`)

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `ErrNilContext` | nil context guard; re-exported as `api.ErrNilContext` and `app.ErrNilContext` | `errs.go:10` |
| `ErrNilLoader` | nil loader guard | `errs.go:13` |
| `ErrNilCommand` | nil CLI command guard (app layer) | `errs.go:15` |
| `ErrNilRequest` | nil convert request guard | `errs.go:17` |

## 4. Data & control flow

### 4.1 CLI path (dotted strings)

1. `cmd/gowkhtmltopdf/main.go` parses argv via `internal/cli`.
2. `cli.Parse` builds a `cli.Command` whose struct **is** the settings payload:
   `Global settings.PdfGlobal`, `Image settings.ImageGlobal`,
   `Objects []settings.PdfObject` (`internal/cli/cli.go:45-47`), initialized
   with `settings.DefaultPdfGlobal()` / `settings.DefaultImageGlobal()` /
   `settings.DefaultPdfObject()` (`cli.go:169-170`, `cli.go:137`).
3. Each CLI flag has a handler that calls the dotted `Set` surface — e.g.
   `--margin-top 25mm` → `c.Global.Set("margin.top", "25mm")`
   (`internal/cli/flags.go:119`), `--enable-local-file-access` →
   `g.Set("enablelocalfileaccess", val)` + per-object
   `o.Set("load.blocklocalfileaccess", negBool(val))` (`flags.go:176-177`).
   The `--no-` prefix form is recognized for booleans (`cli.go:510-514`).
4. `internal/app.BuildPDFRequest(cmd, output, outline)` translates the
   command into `convert.NewPDFRequest(cmd.Global, cmd.Objects, …)` and calls
   `req.Validate()` before any I/O (`internal/app/pdf.go:31-45`).
5. `internal/convert` reads the structs directly: `Request.Global`,
   `Request.Objects`, and (image mode) `Request.Image`.

### 4.2 Library path (typed builder and/or dotted strings)

1. `api.go` wraps the same structs: `GlobalSettings` embeds
   `settings.PdfGlobal` (`api.go:62`), `ObjectSettings` embeds
   `settings.PdfObject` (`api.go:197`), `ImageConverter` embeds
   `settings.ImageGlobal` (`api.go:510`).
2. `GlobalSettings.Set(name, value)` delegates to `g.g.Set(...)`; the typed
   `PdfGlobalOptions` builder routes through `settings.PdfGlobalOptions` and
   `Build()`.
3. `ImageConverter.Set` routes through `settings.ApplyImageKey` so
   `background`/`web.background` lands on the shared `PdfGlobal.Background`
   (`api.go:546`).
4. `Converter.Convert(ctx)` deep-clones settings via `clonePdfGlobal` /
   `clonePdfObject` / `cloneImageGlobal` (`api.go:326-401`) — copies/collate
   re-use objects, so each `Convert` must not share mutable maps/slices with
   the caller's `GlobalSettings`.
5. Conversion executes through `convert.Run`/`RunTypedPDF` with the settings
   snapshots (`api.go:697`, `internal/convert/request.go:66-71`).

### 4.3 Engine consumption

| Consumer | What it reads | File:line |
|----------|---------------|-----------|
| `internal/load.NewLoader(global settings.LoadGlobal)` | Proxy, `Allow` ACL prefixes, `EnableLocalFileAccess` — applied once into the loader policy | `load.go:282` |
| `(*Loader).Load(ctx, input, pageLoad settings.LoadPage)` | Per-page zoom, auth, cookies, POST, media, timeout, `InlineHTML`/`InlineBase`, block-local-access | `load.go:394` |
| `(*Loader).fileAccessAllowed(path, pageLoad)` | `EnableLocalFileAccess && !BlockLocalFileAccess` + `AccessController` prefix match | `load.go:811-817` |
| `internal/load` HTTP failure path | Constructs `&settings.HttpStatusError{Status, URL}` → exit-code mapping at CLI | `load.go:1007-1010` |
| `internal/convert/convert_helpers.go` | `settings.ParsePageSize` for page geometry (`pageGeometry`) | `convert_helpers.go:117` |
| `internal/convert.PolicyForGlobal` | `PdfVersion` + `PdfProfile` → `pdf.WriterPolicy`. Empty profile + empty version → unclaimed PDF 1.4. Profile implies 1.7 or 2.0 unless the explicit version conflicts | `convert.go:254` |
| `internal/imageout` | `mediaFor(global, img, obj)` → `settings.ResolveMedia("screen", …)`; `imageLoadGlobal` → `settings.LoadGlobal`; `fontRegistry` → `PdfGlobal.FontPaths/UseSystemFonts` | `imageout.go:1396`, `imageout.go:1341`, `imageout.go:1321` |
| `internal/convert` (HF) | `obj.HeaderFor(g)` / `obj.FooterFor(g)` inheritance (object override wins) | `settings.go:374` / `settings.go:383` |
| `internal/cli.ExitCode` | Interface check `interface{ HttpErrorCode() int }`; `settings.HttpStatusError` satisfies it | `cli.go:517-524` |

### 4.4 Get flow

`Get(name)` normalizes dots (lowercase + trim), consults the descriptor
table (`getForKey`), and returns the canonical string form. Accepted ignored
keys return the **last Set value** stored in `Ignored`; truly unknown keys
return `ok=false` (`reflect.go:116-129`). After `Set("pdfprofile", "a3a-ua1")`,
`Get("pdfprofile")` is `PDF/A-3a+PDF/UA-1` (the stored canonical token, not
the alias).

## 5. Cross-package dependencies

### 5.1 What `internal/settings` imports

Stdlib plus one internal leaf: `settings.go` imports `internal/pdfprofile`
for profile `Parse` / canonical tokens (no new flavours). Remaining files
stay stdlib-only (`fmt`/`strconv`/`strings` in `reflect.go`,
`strconv`/`strings` in `getters.go`, `fmt` in `httperror.go`,
`errors`/`fmt`/`strconv`/`strings` in `unitreal.go`,
`errors`/`fmt`/`strings` in `pagesize.go`). `internal/pdfprofile` and
`internal/errs` are leaves (`pdfprofile` has no engine imports; `errs`
imports only `errors`).

### 5.2 Who depends on them

| Package | Direction of use |
|---------|------------------|
| `internal/cli` | Imports `settings`; `cli.Command` *is* settings structs; flags call `Set` |
| `api.go` (root) | Imports `settings` and `errs`; wraps structs; re-exports `errs` sentinels |
| `internal/app` | Imports `cli`, `convert`, `errs`; re-exports `ErrNilContext` |
| `internal/convert` | Imports `settings`; `Request` embeds `PdfGlobal`/`PdfObject`/`*ImageGlobal`; typed `PDFRequest`/`ImageRequest` wrap it (`request.go`) |
| `internal/load` | Imports `settings`; `NewLoader(settings.LoadGlobal)`, `Load(..., settings.LoadPage)`, emits `settings.HttpStatusError` |
| `internal/imageout` | Imports `settings`; consumes `PdfGlobal`/`ImageGlobal`/`PdfObject` and `ResolveMedia` |
| `internal/convert/prepare`, `internal/convert/render`, `internal/convert/islands` | Import `settings` transitively via `convert` types |

### 5.3 Import-direction rule

Settings types flow **down** into the engine and never back up: nothing in
`internal/settings` or `internal/errs` imports `cli`, `convert`, `load`, or
any other internal package. The conversion engine stays
CLI-independent — `internal/convert` is driven purely by the settings payload
so library callers never touch argv. `internal/cli` is the *only* place that
owns the flag→dotted-key vocabulary on the command side, while `api.go`
owns the typed-builder vocabulary on the library side; both converge on the
same `Set` surface.

### 5.4 Settings never cross into layout/pdf directly

`internal/layout` and `internal/pdf` do **not** import `internal/settings`.
`internal/convert` (and `internal/imageout`) translate settings into engine
values — page geometry (mm → points via `ParsePageSize`/`UnitReal.Points`),
media type, header/footer strings — before layout/paint. This keeps the
layout engine free of wkhtmltopdf vocabulary.

## 6. Design decisions & trade-offs

### 6.1 wkhtmltopdf work-alike surface (compat first)

The dotted keys, defaults, and exit codes are deliberately copied from
wkhtmltopdf (`pdfsettings.cc`, `imagesettings.cc`, `utilities.cc`,
`loadsettings.cc` — cited in doc comments). The `-no-` flag negation and the
`HttpErrorCode` mapping (404→2, 401→3, else 1) exist so that scripts and
wrappers written against wkhtmltopdf keep working with the same invocation
vocabulary. The trade-off: a *larger* key vocabulary than the engine actually
supports — which Policy A (below) contains.

### 6.2 Policy A — settings honesty (the core design rule)

Documented in `doc.go` and enforced by the code:

- **Only options with an engine consumer get typed fields.** The structs
  contain exactly the fields `convert`, `load`, `imageout`, and `layout`
  read. Everything else is either rejected (unknown key) or swallowed into
  `Ignored map[string]string`.
- **Inert wkhtml keys are accepted but inert.** `dpi`, `javascript`,
  `plugins`, `log-level`, `js-delay`, `user-style-sheet`, `produce-forms`,
  `default-encoding`, and ~20 more are accepted into `Ignored` so existing
  wkhtmltopdf invocations do not hard-fail, but they are never promoted to
  typed stubs without an engine consumer.
- **Dual storage is collapsed.** `Grayscale` is the sole color bit
  (`colormode` and `grayscale` keys both write it); page geometry is the
  `PageSize` name + `Size` width/height pair; `DumpOutline`/`DumpDefaultTOCXSL`
  live on `PdfGlobal`; `background`/`web.background` share one field; image
  mode has no `Web.Background` mirror.
- Origin: `plans/0.2.0/reviews/ponytail/ponytail-2026-08-06/ponytail-ultra-2026-08-06.md`
  Phase 1 (per `doc.go`). This is the antidote to the classic wkhtmltopdf
  clone failure mode where hundreds of settings are stored but never wired.

**Trade-off:** script compatibility (accept-and-ignore) vs. honesty
(reject what is not honored). The `Ignored` maps keep the "accepted"
promise visible and round-trippable through `Get`, while `unknownSettingError`
still catches genuinely misspelled keys — so typos fail loudly, but known
inert keys pass quietly.

### 6.3 Descriptor tables over reflection-based key mapping

The dotted `Set`/`Get` is driven by **explicit, typed descriptor tables**
(`field[T]{apply, get}` built once at package init by `buildKeyTables()`),
not by `reflect`-based struct traversal. Benefits:

- **Compile-time type safety** for every key (setters are closures over typed
  fields; the `sub` adapter bridges sub-structs without losing type info).
- **One registration site per key** — `apply` and `get` are always registered
  together, and `TestKeyTableSetGetParity` verifies every registered key has
  both halves.
- **Zero per-call reflection cost** — tables are built once
  (`globalKeys, objectKeys, imageKeys = buildKeyTables()` at `reflect.go:395`).

Trade-off: adding a new setting touches the struct, a setter, and a
`register*` wiring function — three mechanical edits — instead of one
reflect-annotated field.

### 6.4 Millimetres as the internal length unit

Margins and custom page sizes are stored as **float millimetres**
(`Margin`, `Size.Width/Height`), parsed from arbitrary units via
`ParseUnitReal` + `UnitReal.Mm()`. Page *names* resolve to points via the
static `pageSizes` table (1 pt = 1/72 in, matching Qt QPrinter). Conversion
to PDF points happens at the engine boundary (`convert_helpers.go:117`),
keeping the settings layer unit-agnostic on output.

### 6.5 Value semantics and defensive cloning

All settings structs are **plain value types** (no pointers, no methods that
mutate receivers) except the `Ignored` maps and slice fields. The library
API deep-clones (`clonePdfGlobal` etc. in `api.go:326-401`) so repeated
`Converter.Convert` calls — needed for copies/collate — cannot alias caller
state. `PdfGlobalOptions.Build()` clones slices and maps for the same reason
(`options.go:73-87`). CLI path gets fresh defaults per parse.

### 6.6 Errors as a shared leaf vocabulary

`internal/errs` exists so nil-guard sentinels (`ErrNilContext`,
`ErrNilRequest`, …) have a single identity re-exported by both the library
(`api.go:46`) and the app layer (`internal/app/pdf.go:21`). This lets
`errors.Is` checks work regardless of which adapter the caller used. The
HTTP-status error, by contrast, lives in `settings` (not `errs`) because it
carries *data* (status, URL) and its exit-code contract is exercised by
`cli.ExitCode` — `errs` stays purely sentinel-shaped.

## 7. Notable patterns & invariants

- **Dotted-key normalization:** `normalizeDots` lowercases and trims
  (`reflect.go:206`); keys are case-insensitive (`Orientation`,
  `ParsePageSize`, booleans, enums all `strings.ToLower`).
- **Boolean coercion:** `setBool` accepts `"" / true / 1 / yes / on` and
  `false / 0 / no / off` — wkhtmltopdf-style liberal parsing
  (`reflect.go:209-224`).
- **`Ignored` round-trip:** an accepted inert key is stored verbatim in
  `Ignored` and returned by `Get` with `ok=true` — settings are not silently
  lost (`reflect.go:116-129`, `TestGlobalGetSetRoundTripAndIgnored`).
- **Unknown keys fail loudly:** `errUnknownSetting` — a typo like
  `margin.topx` errors instead of being ignored (`reflect.go:20-26`,
  `TestGlobalSetUnknownKey`).
- **Header/footer inheritance:** object-level overrides only exist if the
  object explicitly set at least one header/footer key (`HeaderSet`/
  `FooterSet` bits set inside the descriptor apply — `reflect.go:614-634`),
  then `HeaderFor`/`FooterFor` fall back to global
  (`settings.go:374-387`).
- **`web.background` alias:** two keys, one field — `background` and
  `web.background` both write `PdfGlobal.Background`; image mode routes both
  through `ApplyImageKey` (`reflect.go:874-880`).
- **Media resolution precedence:** print-media-type override (global *or*
  object) > object media-type > global media-type > mode base (`print` for
  PDF, `screen` for image) — `ResolveMedia` (`settings.go:137-154`).
- **Defaults mirror pdfsettings.cc** with engine-consumed fields only —
  `DefaultPdfGlobal` never initializes `Ignored` (nil map, lazily created by
  `storeIgnored`).
- **Immutable tables:** `ignoredGlobalKeySet`/`ignoredObjectKeySet` are
  package-scope maps documented "immutable-by-convention"; the page-size
  table is a fixed array to avoid exposing a mutable map.
- **Typed builder immutability:** `PdfGlobalOptions` is a value type; each
  `With*` returns a copy, so chaining cannot corrupt a shared builder.
- **Exit-code contract via interface, not type switch:** `cli.ExitCode`
  checks `interface{ HttpErrorCode() int }` — any future error type can join
  without touching cli (`cli.go:517-524`).

## 8. Security considerations

The settings layer is where **security-relevant defaults and ACL policy
originate**:

- **Local file access is denied by default.** `DefaultLoadPage()` sets
  `BlockLocalFileAccess: true` (`settings.go:401-406`), and
  `DefaultPdfGlobal().Load.EnableLocalFileAccess` is false (zero value).
  The effective gate is `EnableLocalFileAccess && !BlockLocalFileAccess`
  enforced in `load.fileAccessAllowed` (`load.go:811-817`) — a global opt-in
  (`--enable-local-file-access`) *and* a per-object opt-out that stays on by
  default.
- **`Allow` prefixes** (`--allow` / `load.Allow`) are the only way to widen
  the local ACL; `NewLoader` clones them into an `AccessController`
  (`load.go:292-300`). Documented in `documentation/THREAT-MODEL.md` and
  `documentation/integration-security.md`.
- **Inert-key acceptance is a compat surface, not an attack surface.** Keys
  like `produceforms`, `javascript`, `plugins` are *accepted then ignored* —
  the engine never executes scripts or creates forms. `Set` validates the
  *key* vocabulary strictly (unknown keys error), so a malicious settings
  string cannot smuggle behavior in via misspelled keys.
- **Proxy must be an absolute URL** — `load.parseProxy` rejects non-absolute
  and non-http(s) proxies with `ErrInvalidProxy` (`load.go:369-383`), a
  guard against SSRF-adjacent proxy misconfiguration.
- **`HttpStatusError` drives exit codes, not data disclosure** — it carries
  only status + URL and is produced by `internal/load` on HTTP failures
  (`load.go:1007-1010`).
- **Timeouts/limits** (`load.timeout`, max redirects, max body size) are
  typed in `LoadPage.Timeout` and enforced by `internal/load`, not by the
  settings layer itself — settings merely carries the policy.

Related security docs: `documentation/THREAT-MODEL.md`,
`documentation/integration-security.md`.

## 9. Testing & verification

The package is validated by table-driven unit tests in
`internal/settings/settings_test.go` (648 lines) and
`internal/settings/options_test.go` (32 lines). The suite runs with
`t.Parallel()` throughout and exercises unexported tables via the
`package settings` (white-box) test package:

| Test | What it verifies |
|------|------------------|
| `TestDefaultPdfGlobalSnapshot` / `TestDefaultPdfObjectSnapshot` / `TestDefaultLoadPageSnapshot` / `TestDefaultImageGlobalNoQuietLogLevel` | wkhtmltopdf-compatible defaults (A4, 10 mm margins, Arial 12 header, depth-4 outline, block-local-file-access true, load-error abort, image 1024×94) |
| `TestGlobalSetDottedKeys` (+ `globalDottedGeometryChecks` / `globalDottedTextChecks`) | End-to-end dotted key application: `margin.top=25mm`, `margin.left=1in`→25.4 mm, canonical `size.pagesize`, `colormode`/`grayscale` aliasing, `web.background`→`Background`, `allow` append |
| `TestGlobalSetIgnoredKeys` / `TestObjectSetIgnoredKeys` | Policy A: inert keys accepted into `Ignored` and round-tripped |
| `TestGlobalSetUnknownKey` | Unknown keys error |
| `TestObjectSetDottedKeys` | Object keys incl. `load.*`, header/footer override bits (`HeaderSet`/`FooterSet`), `web.*` |
| `TestParseUnitReal` / `TestUnitRealPoints` | Unit parsing (incl. whitespace `" 7.5 mm "`), conversions (10 mm → 28.346 pt, 96 px → 72 pt), `em` non-convertible without font context, `ErrInvalidUnitReal` |
| `TestParsePageSize` | A4 595.28×841.89 pt, `letter` 612×792, unknown-size error |
| `TestParseEnums` | Case-insensitivity and invalid-value errors |
| `TestHttpErrorCode` | 404→2, 401→3, 500→1 |
| `TestHeaderForFooterForInherit` | Object override wins, global fallback otherwise |
| `TestImageSet` / `TestApplyImageKeyBackgroundAlias` | Image keys + `background`/`web.background` routing to `PdfGlobal.Background` |
| `TestGlobalGetSetRoundTripAndIgnored` | Get returns canonical strings; ignored keys readable; unknown keys `not-found` |
| `TestKeyTableSetGetParity` | Every registered key has both apply and get descriptors (table completeness invariant) |
| `TestResolveMedia` | Full precedence matrix (print-media-type > object > global > base) |
| `TestPdfGlobalOptionsBuildsIndependentTypedSnapshot` (`options_test.go`, external test package) | Typed builder produces independent, correctly populated snapshot |

Downstream integration coverage: golden CLI tests (`internal/cli/cli_test.go`),
convert seams tests (`internal/convert/seams_test.go`), load tests
(`internal/load/load_test.go`), and imageout fontface tests exercise the
settings values end-to-end. The `TestKeyTableSetGetParity` test is the
structural invariant that keeps `Set` and `Get` tables in sync as keys are
added.

## 10. Known limitations, deferred items & open questions

- **Ignored keys are a growing compat debt surface.** Every accepted-but-inert
  wkhtml key (see `ignoredGlobalKeySet`/`ignoredObjectKeySet`,
  `reflect.go:136`/`reflect.go:170`) is a "script-compatible but silently
  different" behavior. The ponytail review rule (Policy A) requires an engine
  consumer before promoting any of them; today `dpi`, `javascript`,
  `js-delay`, `produce-forms`, `log-level`, `default-encoding`, etc. have no
  effect.
- **`default-encoding` is accepted then ignored.** `doc.go`/`reflect.go`
  note the engine is UTF-8/ASCII only via `html.ParseDocument` + load charset
  seam; multi-charset decode is a stated upgrade path
  (`reflect.go:149-153`).
- **`web.background` on objects is inert** — paint background is global-only
  (`ignoredObjectKeySet` includes `web.background`); object-level background
  overrides are not honored.
- **`load.proxy` is object-ignored** — only `LoadGlobal.Proxy` is wired;
  per-object proxies silently do nothing (`reflect.go:160`).
- **`size.pagesize` uses one canonical field.** `PdfGlobal.PageSize` stores the
  named page geometry and `Size` stores only width/height measurements. The
  settings parity tests protect this single source of truth.
- **`PdfGlobalOptions` builder covers only global PDF settings.** Object
  options and image options have no typed builder; callers must use dotted
  `Set` (`options.go`).
- **`fontpath`/`allow` are append-only setters** — there is no way to clear
  previously appended values through the dotted surface (matches wkhtmltopdf
  flag accumulation semantics).
- **`Quiet` lives on `PdfGlobal`, not `ImageGlobal`** — image-mode callers
  must set it on the shared global (documented in `settings.go:408-412` and
  `TestDefaultImageGlobalNoQuietLogLevel`).
- **Exit-code mapping is HTTP-status-only** — 404→2 / 401→3 / else 1;
  wkhtmltopdf has additional fine-grained codes that are not modeled.
- Deferred feature context: `documentation/deferred.md` and
  `plans/0.2.0/10-canonical-post-mvp-roadmap.md`; fidelity claims about which
  settings are honored live in `documentation/fidelity.md` and
  `documentation/compatibility-matrix.md`.

## 11. Related documents

- **In this directory (sibling architecture deep-dives):**
  - [01-entrypoints-cli.md](01-entrypoints-cli.md) — where `Set` is called
    from flags; `cli.ExitCode` consumes `HttpErrorCode`
  - [02-library-api.md](02-library-api.md) — typed builder + dotted wrappers
    over the same structs; clone-on-convert
  - [04-load.md](04-load.md) — how `LoadGlobal`/`LoadPage` policy is enforced
  - [05-html-parser.md](05-html-parser.md) — `InlineHTML`/`InlineBase` input
    seam
  - [06-css.md](06-css.md) — media-type resolution feeding CSS media
    selection
  - [07-layout.md](07-layout.md) — unit conversion downstream of
    `UnitReal.Points` (font-relative units resolved in layout)
  - [08-convert-pipeline.md](08-convert-pipeline.md) — `Request` carries
    `PdfGlobal`/`PdfObject`/`ImageGlobal`; `ValidatePDF`/`ValidateImage`
  - [09-pdf-writer.md](09-pdf-writer.md) — receives already-converted page
    geometry (points), never settings keys
  - [10-imageout-svg.md](10-imageout-svg.md) — `mediaFor`/`imageLoadGlobal`/
    `fontRegistry` consumption of `ImageGlobal`/`Web`
- **Top-level documentation:**
  - [documentation/architecture.md](../architecture.md) — package map and
    pipeline overview this document expands
  - [documentation/library-api.md](../library-api.md) — public API usage
  - [documentation/cli.md](../cli.md) — flag ↔ dotted-key reference
  - [documentation/fidelity.md](../fidelity.md) — what is actually honored
  - [documentation/THREAT-MODEL.md](../THREAT-MODEL.md) — ACL/security
    defaults originating here
  - [documentation/compatibility-matrix.md](../compatibility-matrix.md) —
    per-feature support claims
  - [documentation/deferred.md](../deferred.md) — deferred features
- **Plans:** [plans/0.1.0/00-canonical-pure-go-rewrite.md](../../plans/0.1.0/00-canonical-pure-go-rewrite.md),
  [plans/0.2.0/reviews/ponytail/ponytail-2026-08-06/ponytail-ultra-2026-08-06.md](../../plans/0.2.0/reviews/ponytail/ponytail-2026-08-06/ponytail-ultra-2026-08-06.md)
  (Policy A origin, Phase 1)
