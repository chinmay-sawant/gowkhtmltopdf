# Entrypoints & CLI (governs the multi-object document grammar)

## 1. Responsibility & position in the pipeline

This domain is the **outermost shell** of `gowkhtmltopdf`: it converts raw
process arguments into a fully validated, settings-shaped `cli.Command` and
hands it to the application adapters that build engine requests. It is the
only place that understands the wkhtmltopdf-compatible argument grammar
(`page` / `cover` / `toc` objects, free positionals, output file, doc flags),
and it owns the **exit-code contract** (0 ok / 1 error / 2 HTTP 404 / 3
HTTP 401) that wkhtmltopdf users expect from a drop-in CLI.

Position in the overall pipeline:

```text
shell argv
   │
   ▼
cmd/gowkhtmltopdf|image main ──parse──► cli.Parse(argv, Mode) ──► Command (settings-shaped)
   │                                                                    │
   │                                              internal/app adapters │ BuildPDFRequest / RunImage
   │                                                                    ▼
   │                                                 convert.Request (engine input)
   │                                                                    │
   ▼                                                                    ▼
exit code ◄────────────────── cli.ExitCode(err) ◄── errors   load → parse → style → layout → paginate → paint → write
```

The conversion pipeline itself (see 08-convert-pipeline.md and
09-pdf-writer.md, 10-imageout-svg.md) never parses argv; it consumes
`convert.Request`, which is precisely why the CLI domain can change flag
spelling without touching the engine and why the library API (api.go) can
drive the same engine without any shell involvement.

Two binaries share one parser package:

- `gowkhtmltopdf` — PDF mode (`cli.ModePDF`)
- `gowkhtmltoimage` — image mode (`cli.ModeImage`)

A `Mode` bitmask restricts which flags each binary accepts, so the parser
remains a single implementation while each binary presents a faithful
wkhtmltopdf-style surface.

## 2. Package / file map

| File | Lines (approx) | Responsibility |
|------|---------------:|----------------|
| `cmd/gowkhtmltopdf/main.go` | 83 | PDF binary entry: parse, doc-flag dispatch, `--dump-default-toc-xsl` shortcut, quiet/outline wiring, signals, exit code |
| `cmd/gowkhtmltoimage/main.go` | 53 | Image binary entry: parse (ModeImage), doc-flag dispatch, signals, exit code |
| `internal/cli/cli.go` | ~548 | Parser core: `Command` struct, `Parse` loop, object grammar (`page`/`cover`/`toc`), positionals/free resolution, value extraction, `ExitCode` mapping |
| `internal/cli/flags.go` | ~495 | Flag table construction (`flagTable`), `Mode` constants, 71 registered flags grouped by policy (`add*Flags` helpers) and routing appliers (`pageOnlyFlag`, `hfFlag`, `tocFlag`, `printMediaFlag`, `applyPage`) |
| `internal/cli/help.go` | ~71 | `PrintHelp` / `PrintVersion` / `PrintLicense`, the ldflags-stampable `Version` var, mode-filtered `flagList` |
| `internal/cli/doc.go` | ~11 | Package doc: multi-object grammar; Phase 1 Policy A (only engine-consumed flags registered) |
| `internal/cli/cli_test.go` | 838 | Table-driven parser tests: settings targets, grammar, modes, doc flags, exit codes, stub-flag rejection, output-writer precedence |

The two `cmd` roots total ~136 lines and are intentionally thin: everything
decision-shaped lives in `internal/cli` plus the `internal/app` adapters
(see section 5).

## 3. Key types, functions & entry points

### 3.1 `cmd/gowkhtmltopdf/main.go` (PDF)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `main()` | main.go:18 | `os.Exit(run(os.Args[1:]))` — keeps `run` testable as a `func([]string) int` |
| `run(argv []string) int` | main.go:23 | Full PDF command lifecycle (below) |
| `cli.Parse(argv, cli.ModePDF)` | main.go:24 | Entry to the shared parser, PDF mode |
| `convert.DefaultTOCXSL()` | main.go:49 | Sole dump home for `--dump-default-toc-xsl`: prints built-in XSL to stdout and exits 0 |
| `app.RunPDF(ctx, cmd, logw, nil, outline)` | main.go:75 | Application adapter; converts `Command` → `convert.Request`, opens output, runs engine |

`run` order of operations (the PDF contract):

1. `cli.Parse` — errors are dispatched: `ErrHelp`/`ErrExtHelp` → help to
   stdout, exit 0; `ErrVersion` → version, exit 0; `ErrLicense` → license,
   exit 0; anything else → `gowkhtmltopdf: <err>` to stderr, exit 1.
2. `--dump-default-toc-xsl` short-circuit (Global bit) before any I/O.
3. Empty/missing output → stderr, exit 1 (input is never rendered speculatively).
4. `--quiet` redirects the log writer to `io.Discard` (errors still print to
   stderr unconditionally).
5. `signal.NotifyContext(SIGINT, SIGTERM)` so Ctrl-C cancels conversion
   cleanly (context propagated through `app.RunPDF` → `convert.Run`).
6. `--dump-outline` (either `cmd.DumpOutline` or `cmd.Global.DumpOutline`)
   routes the outline XML writer to stdout.
7. Exit code: `cli.ExitCode(runErr)` (0 on success).

### 3.2 `cmd/gowkhtmltoimage/main.go` (image)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `main()` | main.go:16 | `os.Exit(run(os.Args[1:]))` |
| `run(argv []string) int` | main.go:20 | Parse with `cli.ModeImage`, doc-flag dispatch, then `app.RunImage`; exit code via `cli.ExitCode` |

The image binary is ~36 lines shorter than PDF: no `--dump-default-toc-xsl`
handling, no outline writer, log writer is `os.Stderr` directly. All image
specific-decision logic (format sniffing, crop, quality) lives in
`internal/imageout` (see 10-imageout-svg.md).

### 3.3 `internal/cli/cli.go` — parser core

| Symbol | Location | Purpose |
|--------|----------|---------|
| `ErrHelp`, `ErrVersion`, `ErrLicense`, `ErrExtHelp` | cli.go:16 | Sentinel **doc-flag** errors; caller prints and exits 0. Keeps `os.Exit` out of the library-parseable package. |
| `errUnknownOption … errInvalidBoolValue` | cli.go:25 | Sentinel parse-stage errors, each wrapped with the offending token so callers can match with `errors.Is` |
| `ExitOK = 0`, `ExitError = 1` | cli.go:38 | Exit-code constants (existing HTTP exit codes 2/3 are produced via `ExitCode`) |
| `Command` | cli.go:44 | Parse result: `Global settings.PdfGlobal`, `Image settings.ImageGlobal`, `Objects []settings.PdfObject`, `Output string`, `OutputWriter io.Writer`, `OutlineWriter io.Writer`, `DumpDefaultTOCXSL bool`, `DumpOutline bool` |
| `(*Command).OpenOutput()` | cli.go:65 | Writer resolution: `OutputWriter` (library/embed path) → `Output` file → stdout; returns writer + closer (no-op for writer/stdout) |
| `flagKind` / `flagSpec` | cli.go:83 / 93 | `flagBool` (one canonical `"true"`/`"false"`), `flagValue` (one token), `flagPair` (two tokens `name value`); `flagSpec{kind, mod, app}` |
| `objectCtx` | cli.go:99 | Current object + `pending` buffer for page-scoped settings seen before any object keyword (upstream “address remapping”) |
| `parseState` | cli.go:111 | Mutable loop state: cmd, current object context, argv, index, mode, free positionals |
| `(*objectCtx).object/newObject/newFreshObject` | cli.go:122 / 132 / 148 | Object lifecycle; `newFreshObject` (used for `toc`) appends **without** consuming `pending`, so pre-object page flags apply to the first real page, never to the TOC entry |
| `Parse(argv, modes …Mode)` | cli.go:162 | Main parser: optional mode, seeds defaults, token loop, `resolveFree` |
| `(*parseState).step` | cli.go:197 | Per-token dispatch: doc flags, `--` end-of-options, `--long`, `-x` short, positional |
| `docFlagErr` | cli.go:221 | `-h/--help`, `-V/--version`, `-L/--license`, `-E/--extended-help` → sentinel |
| `parseLongFlag` / `parseShortFlag` | cli.go:243 / 261 | Lookup, mode check (`checkMode`), `--no-` negation resolution, value application |
| `ParseMode` / `parseMode` | cli.go:278 / 282 | Explicit mode entry; at most one mode, omitted → `ModeBoth` (historical library behavior accepting the union) |
| `checkMode` | cli.go:299 | Rejects flags not allowed by the binary's mode (`option --X is not supported in pdf/image mode`) |
| `(*Command).positional` | cli.go:315 | Object keywords `page` / `cover` / `toc` plus bare-positional queueing |
| `(*Command).resolveFree` | cli.go:351 | Last free positional → `Output`; the rest → implicit page objects (first one filling an open empty object, or promoting `pending`) |
| `(*Command).validate` | cli.go:382 | Invariant: at least one non-TOC object with a non-empty `Page` URL |
| `(*objectCtx).applyPage` | cli.go:403 | Dual-write helper: applies a page-scoped flag to the Global half first, then the current object, or accumulates `pending` for the first page |
| `(*parseState).apply` | cli.go:427 | Flag-kind dispatch + value extraction (`=value`, next-arg, two-arg pairs) + validation (`errOptionRequiresValue/Pair`) |
| `parseBool` | cli.go:479 | Bool values `true/1/yes/on` and `false/0/no/off`; inline `=value` wins, otherwise `--no-` negates |
| `splitFlag` / `lookupFlag` | cli.go:494 / 503 | `--name=value` split; `--no-<boolflag>` lookup for bool flags only |
| `ExitCode(err error) int` | cli.go:520 | `errors.As` for an `interface{ HttpErrorCode() int }` (implemented by `settings.HttpStatusError`); else 1 |

### 3.4 `internal/cli/flags.go` — the flag table

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Mode` / `ModePDF`/`ModeImage`/`ModeBoth` | flags.go:17 / 19 | Bitmask: PDF = 1, Image = 2, Both = 3 |
| `flagTable` / `shortFlags` | flags.go:27 / 31 | Static tables built once at package init, never mutated; `shortFlags` (`q g O s T B L R c t`) alias long-form specs |
| `buildFlagTable` | flags.go:64 | Assembles the ~120-slot table via one `add*Flags` group per policy |
| `addDocFlags` | flags.go:87 | `help/version/license/extended-help` as `nopFlag` (handled by `docFlagErr` before table lookup; present so `flagList` shows them) |
| `addGlobalFlags` | flags.go:95 | `quiet`, `collate`, `copies`, `orientation`, `page-size`, `grayscale`, `title`, `margin-top/bottom/left/right`, `page-width/height`, `no-pdf-compression`, `page-offset`, `--pdf-version` (`pdfversion`, flags.go:142), `--pdf-profile` (`pdfprofile`, flags.go:145), `enable/disable-smart-shrinking`. Version is not a PDF/A/UA claim; `--pdf-profile` is (`a3a-ua1` / `a4-ua2` / …, ModePDF only) |
| `addOutlineFlags` | flags.go:152 | `outline`, `outline-depth`, `dump-outline`, `dump-default-toc-xsl` — “one home: Global settings” |
| `addLocalAccessFlags` | flags.go:173 | `enable/disable-local-file-access` (dual-write via `applyPage` with `load.blocklocalfileaccess` negation), `allow` (ACL prefix), `background`/`no-background` |
| `addWebPageFlags` | flags.go:202 | `simplify-dom`, `simplify-dom-profile`, `print-link-underline`, `print-media-type`/`no-print-media-type`, `media-type` (all routed global+object) |
| `addLoadPageFlags` | flags.go:243 | `zoom`, `load-error-handling`, `proxy`, `username`, `password`, `timeout`, `external-links`, `internal-links`, `resolve-relative-links`, `keep-relative-links` |
| `addFontFlags` | flags.go:285 | `font-path`, `use-system-fonts` |
| `addPairFlags` | flags.go:295 | `cookie`, `custom-header` (map entries), `post` (repeated `PostItem` list), `replace` (header/footer text substitution) |
| `addHeaderFooterFlags` | flags.go:323 | `header|footer` × `left/right/center/font-name/font-size/spacing/line/html` — generated by nested loops |
| `addTOCFlags` | flags.go:338 | `xsl-style-sheet`, `toc-header-text`, `toc-text-size-shrink`, `disable/enable-toc-links`, `disable-dotted-lines`, `toc-level-indentation`, `toc-back-links` |
| `addImageFlags` | flags.go:350 | `width`, `height`, `crop-x/y/w/h`, `format`, `quality`, `transparent`, `smart-width`/`no-smart-width` (ModeImage only) |
| `pageOnlyFlag` | flags.go:414 | Routing helper for flags with **no Global consumer** (zoom, username, password, timeout, links): current object or pending first-page settings |
| `hfFlag` | flags.go:433 | Header/footer flags: object-level when an object exists, otherwise **global-only** storage (every object inherits via `HeaderFor`/`FooterFor`) — pending is never created |
| `tocFlag` / `tocFlagBool` | flags.go:446 / 456 | `toc.*` keys on the current object when it is a TOC object, else global (inherited via effective TOC) |
| `(*Command).replaceHF` | flags.go:471 | `--replace` routing: object `Header.Replace` map when `HeaderSet`, else `Global.Header.Replace` |
| `negBool` | flags.go:492 | Flips canonical `true`/`false` strings |
| `nopFlag` | flags.go:386 | No-op applier for doc flags |

### 3.5 `internal/cli/help.go`

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Version` | help.go:15 | Default `"0.1.0"` (matches `VERSION`). `make build` stamps `-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)` |
| `PrintHelp` | help.go:14 | Name/synopsis/object-grammar/description + mode-filtered `--flag` list + cross-link to the compatibility matrix |
| `PrintVersion` / `PrintLicense` | help.go:41 / 46 | Version banner; MIT + wkhtmltopdf attribution banner |
| `flagList(mode)` | help.go:60 | Sorts `flagTable` names, filters by `Mode`, renders `<value>` suffix for non-bools |

## 4. Data & control flow

### 4.1 Typical PDF invocation

```text
$ gowkhtmltopdf --page-size A4 cover cover.html toc page ch1.html ch2.html out.pdf
```

1. `main` → `run(os.Args[1:])`.
2. `cli.Parse(argv, cli.ModePDF)` seeds `Command{Global: DefaultPdfGlobal(),
   Image: DefaultImageGlobal()}` and walks tokens:
   - `--page-size A4` → long-flag: `splitFlag`, `lookupFlag`, `checkMode`
     (ModePDF accepted), `apply` → `flagValue` consumes `A4` →
     `c.Global.Set("size.pagesize", "A4")`.
   - `cover` → `positional` → `newObject` appends a `DefaultPdfObject`,
     stamps `IsCover=true`, `IncludeInOutline=false`, cleared
     `Header/Footer` with `HeaderSet/FooterSet=true`.
   - `cover.html` → fills the open cover object's `Page`.
   - `toc` → `newFreshObject` (pending untouched): `IsTableOfContent=true`,
     `UseOutline=false`.
   - `page` → `newObject` (a fresh body object; pending consumed if any).
   - `ch1.html` / `ch2.html` fill the body page objects.
   - `out.pdf` → free positional queued.
3. `resolveFree`: last free (`out.pdf`) → `Command.Output`; here all
   positionals were consumed by explicit object keywords, so the loop adds
   nothing; `validate()` passes (2 non-TOC objects with URLs).
4. `run` checks output, wires `--quiet`/dump-outline, builds a
   signal-cancellable context, calls `app.RunPDF(ctx, cmd, logw, nil,
   outline)`.
5. `app.RunPDF` (internal/app/pdf.go:34):
   - `BuildPDFRequest(cmd, io.Discard, outline)` → `convert.NewPDFRequest` +
     `req.Validate()` — **validation happens before any output file is
     created or truncated**;
   - empty `cmd.Objects` → `ErrNoPageObjects`;
   - `cmd.OpenOutput()` → real file/stdout writer → `req.Output = out`;
   - `convert.Run(ctx, req, log, progress)` (08-convert-pipeline.md).
6. Exit: 0 on success, `cli.ExitCode(err)` otherwise (2 for HTTP 404, 3 for
   HTTP 401, 1 for everything else).

### 4.2 Typical image invocation

```text
$ gowkhtmltoimage --width 800 --crop-x 0 page.html out.png
```

`cli.Parse(argv, cli.ModeImage)` rejects PDF-only flags at parse time
(`--page-size` → `not supported in image mode`) but accepts image flags
(`--width`, `crop-x` …) into `Command.Image`. `app.RunImage`
(internal/app/image.go:24) validates a discard-sink
`convert.NewImageRequest(cmd.Global, cmd.Image, cmd.Objects, io.Discard)`,
then owns output opening and delegates to `imageout.RunRequest`
(10-imageout-svg.md), which sniffs the format from `--format`/output
extension (`imageout.ResolveFormat`), re-validates, opens the sink, and
executes `imageout.RunRequest`. The command adapter owns this translation;
the image engine does not import `internal/cli`.

### 4.3 Page-scoped flag routing (address remapping)

A flag may be:

- **Global-only**: `--quiet`, `--copies`, `--title`, `--outline-depth`,
  `--allow`, `--font-path` → writes one Global field. Negated/aligned forms
  (`--no-background`, `--disable-smart-shrinking`) write the same home.
- **Page-scoped with global twin** (`applyPage` router): `--zoom`,
  `--enable-local-file-access`, `--media-type`, `--proxy`,
  `--load-error-handling`, `--print-media-type`, `--simplify-dom`,
  `--print-link-underline`. The Global half is written **and** the object
  half; before any object keyword the object half accumulates in
  `objectCtx.pending` and is **promoted** into the first real page object
  (`resolveFree`), matching wkhtmltopdf's “address remapping” so
  `--zoom 0.67 url out.pdf` stamps the zoom on the first page.
- **Page-only** (`pageOnlyFlag`): `--zoom`-class flags with no Global
  consumer write only the current object or pending, never Global.
- **TOC-scoped** (`tocFlag`): on a current TOC object → `toc.*` object key;
  otherwise global (inherited by all TOCs via effective TOC resolution).

Invariant guarded by tests: TOC objects do **not** consume `pending`, so
`--enable-local-file-access toc page in.html out.pdf` leaves no ghost page
prior to the TOC (TestPageScopedBeforeTOCNoGhost).

### 4.4 Value extraction

`apply` (cli.go:427) implements the three flag kinds:

- `flagBool`: inline `--flag=true` wins over `--no-flag` negation;
  `parseBool` accepts `true/1/yes/on` and `false/0/no/off` (errors on other
  values via `errInvalidBoolValue`).
- `flagValue`: inline `--name=value` or next-arg consumption
  (`errOptionRequiresValue` at end of argv).
- `flagPair`: two tokens `name value` (`cookie`, `custom-header`, `post`,
  `replace`); requires both or errors `errOptionRequiresPair`.

### 4.5 Error → exit-code mapping

`cli.ExitCode` (cli.go:520) uses `errors.As` on
`interface{ HttpErrorCode() int }`. `settings.HttpStatusError`
(internal/settings/httperror.go) implements it: HTTP 404 → 2, HTTP 401 → 3,
else 1 (wkhtmltopdf `utilities.cc` convention). Errors are wrapped with
`fmt.Errorf("... %w", ...)` throughout so wrapped HTTP errors still resolve.

## 5. Cross-package dependencies

### Import direction (verified against sources)

```text
cmd/gowkhtmltopdf ──► internal/app ──► internal/convert ──► internal/cli ──► internal/settings
cmd/gowkhtmltoimage ─► internal/app ──► internal/imageout ─► internal/cli
        ▲                  │                 │
        │                  └──► internal/convert ──► render, css, layout, line, load, outline, pdf
        └──► internal/cli ──► internal/settings      (engine packages, 08/09/07-…)
root api.go ──► convert, imageout, line, settings, errs   (library path, 02-library-api.md)
```

| Edge | Evidence | Comment |
|------|----------|---------|
| `cmd/*` → `internal/app` | both mains import app | Or if `app.RunPDF`/`RunImage` |
| `cmd/gowkhtmltopdf` → `internal/convert` | main.go:10 | Only for `convert.DefaultTOCXSL()` |
| `internal/app` → `internal/cli` + `internal/convert` | app/pdf.go:12-13, app/image.go:9-10 | Command → Request translation lives here, keeping `internal/convert` CLI-adjacent but engine-first |
| `internal/app` → `internal/imageout` | app/image.go:11 | Image adapter delegates to imageout (which takes `*cli.Command` for the compatibility seam) |
| `internal/cli` → `internal/settings` only | cli.go + flags.go imports | **Leaf discipline**: the parser is engine-independent and re-usable by any settings consumer |
| `internal/app` → `internal/cli` | app/pdf.go, app/image.go | Application adapters translate `cli.Command` into request structs; the core engines take no CLI types |
| `internal/imageout` → `internal/cli` | imageout.go import | `Run(ctx, cmd *cli.Command, …)` CLI seam; `RunRequest` is the CLI-free core |
| `root api.go` → convert/imageout | api.go imports | CLI never imports the library root (keeps `internal` self-contained) |

**Rule stated in code**: `internal/app` exists so “command mains stay
orchestration-only while internal/convert remains a CLI-independent engine
for library callers” (app/pdf.go package doc). No package below
`internal/cli` may ever import a `cmd/*` package, and the engine never
needs process-global stdout (all sinks are explicit `io.Writer`s).

## 6. Design decisions & trade-offs

- **wkhtmltopdf work-alike surface**: flag spellings, the multi-object
  grammar, and exit codes 0/1/2/3 mirror wkhtmltopdf so CI scripts written
  for wkhtmltopdf keep working. This is a deliberate compatibility bet: it
  costs parser complexity (address remapping, cover/toc special cases) that
  a fresh CLI would not need.
- **One parser, two binaries** via `Mode` bitmask: avoids flag-table
  duplication and gives shared behavior (doc flags, exit codes) for free.
  The trade-off: every flag must declare its mode, and mode violations are
  parse-time errors rather than runtime warnings.
- **Policy A — no inert flags** (doc.go, TestStubFlagsRemoved): only flags
  with an actual engine consumer (`convert`/`load`/`imageout` or a
  `Command` field read by a main) are registered. wkhtmltopdf stubs such as
  `--dpi`, `--javascript-delay`, `--enable-javascript`, `--cookie-jar`,
  `--use-xserver`, `--log-level` are **rejected as unknown**, not accepted
  no-ops. Rationale: silent acceptance would make users believe a feature
  exists; the trade-off is CLI incompatibility for unsupported wkhtmltopdf
  surface, which is documented in the compatibility matrix.
- **Sentinel doc errors instead of `os.Exit`**: `--help` etc. return
  `ErrHelp`/`ErrVersion`/`ErrLicense`/`ErrExtHelp`; the caller prints and
  exits 0. Keeps `internal/cli` library-safe and unit-testable.
- **Validate before touching output**: both adapters validate with an
  `io.Discard` sink *before* `OpenOutput()` creates/truncates the file, so
  a bad request never destroys a previous output file.
- **Explicit writer discipline**: `Command.OutputWriter` takes precedence
  over `Output` (embedders need no temp files); `convert` never reaches
  into process-global stdout; the outline (dump) stream is separate from
  the document stream (see convert's `ErrMissingOutlineOutput`).
- **Pending settings (address remapping)** faithfully reproduces
  wkhtmltopdf's page-scoped-before-keyword behavior, with a deliberate
  improvement: TOC objects never consume pending, avoiding ghost pages
  (a wkhtmltopdf wart).
- **Header/footer and TOC inheritance**: object-level `HeaderSet/FooterSet`
  decide object override vs global inheritance (`HeaderFor`/`FooterFor`);
  `hfFlag` stores global-only before any object keyword. This is the
  minimal model that satisfies the invoice/report use cases without a
  full property-inheritance system.
- **Deep-copy-free value settings**: `Command` holds value structs
  (`settings.PdfGlobal` etc.); no pointer sharing, no ownership transfers,
  so the parser cannot alias state into the engine.
- **Pure-Go, no cgo**: the entire CLI layer is stdlib-only (`errors`, `fmt`,
  `io`, `os`, `os/signal`, `strconv`, `strings`, `syscall`), keeping
  `CGO_ENABLED=0` builds trivially static.

## 7. Notable patterns & invariants

- **Dotted settings names**: appliers write engine settings through
  `settings.Set(name, value)` (e.g. `size.pagesize`, `margin.top`,
  `load.blocklocalfileaccess`, `crop.left`, `web.simplifydom`), so the flag
  table is a thin translation layer over the settings domain
  (03-settings.md). Direct field writes are reserved for structured values
  (cookie maps, post lists).
- **“One home” rule**: each semantic setting has exactly one field home;
  e.g. `--dump-default-toc-xsl` writes `Global.DumpDefaultTOCXSL` only; the
  engine reads Global only, and legacy `Command.DumpOutline` is OR-ed into
  `Global.DumpOutline` at the adapter boundary (app and convert do the same
  OR). Comments in flags.go state this explicitly to prevent dual-home drift.
- **`--no-<flag>` negation** resolved by `lookupFlag` (bool flags only),
  and **enable/disable pairs** for settings with no sensible bare form
  (e.g. smart-shrinking).
- **Canonical bools**: appliers always receive `"true"`/`"false"`; `negBool`
  flips between them — no caller-side boolean formatting.
- **Invariants**:
  - `Command.Global`/`Command.Image` are always populated from
    `DefaultPdfGlobal()`/`DefaultImageGlobal()` — never zero-valued.
  - After `Parse` success, at least one non-TOC object holds a non-empty
    `Page` (validate, cli.go:382).
  - Parse-stage errors are sentinel-wrapped with the offending token
    (`--name`, `-x`), enabling `errors.Is` matching in mains and tests.
  - Exit codes are computed by one function only (`ExitCode`).
- **Extension point**: adding a flag = one `add(...)` line + an applier
  (usually a `settings.Set` call) + a test. There is no plugin system and
  none is planned — the CLI surface is intentionally small and
  engine-consumer-driven.

## 8. Security considerations

- **Local-file ACL is opt-in**: `--enable-local-file-access` flips
  `Global.Load.EnableLocalFileAccess` *and* negates object-level
  `load.blocklocalfileaccess` (dual-write via `applyPage`,
  flags.go:173-200); `--allow <prefix>` adds ACL prefixes to
  `Global.Load.Allow`. The default object policy
  (`settings.DefaultLoadPage`) is `BlockLocalFileAccess: true`.
  Enforcement lives in `internal/load` (04-load.md) against
  THREAT-MODEL.md.
- **HTTP(S) inputs make the process a network client** for the primary URL
  *and* every linked stylesheet/image/font; `--proxy`, `--username/--password`,
  `--cookie`, `--custom-header`, and `--post` channel credentials/data into
  the loader. cli.md's “Remote URL security” section documents this threat
  class for CLI users.
- **Validation precedes file creation** so malformed flags cannot truncate
  an existing output file.
- **Diagnostic streams stay separate**: outline/dump output never shares
  the document sink (`OutlineWriter` vs `OutputWriter`), preventing mixed
  formats and accidental exfiltration of dump data into a deliverable PDF.
- **No external process execution**: the CLI spawns no browser, no native
  converter, no network daemon (pure-Go engine), so argv cannot influence
  a subprocess command line.
- **Graceful interrupt handling**: SIGINT/SIGTERM cancel the conversion
  context rather than killing mid-write.

## 9. Testing & verification

`internal/cli/cli_test.go` (838 lines, same-package tests reaching
`flagTable`/`shortFlags` directly) is the primary safety net; all tests are
table-driven and `t.Parallel()`:

| Test | Verifies |
|------|----------|
| `TestGlobalFlagsToSettings` | Exact settings fields written per flag (page-size → canonical `PageSize`, orientation enum, mm margins → mm numbers, outline-depth, grayscale, quiet) |
| `TestShortFlags` | `-s -O -q -T -c -t` map to long-form specs. `-L` is **license**, not `--margin-left` |
| `TestFlagEqualsSyntax` | `--flag=value` inline values |
| `TestBoolFlagValues` | `true/1/yes/on` + `false/0/no/off` + `--no-` negation + invalid values |
| `TestMultiObjectGrammar` | `cover … toc page … page …` object composition and order |
| `TestImplicitFirstPageObject` | First bare positional becomes the first page |
| `TestStdinOutputAndInput` | `-` as **output** is stdout; `-` as input is **not** stdin HTML (`GuessURL("-")` → `http://-` unless a file named `-` exists) |
| `TestPairFlags` / `TestHeaderFooterFlags` / `TestTOCFlags` / `TestLoadFlags` | cookie/header/post/replace maps; header/footer keys; toc.* keys; zoom/auth/timeout/proxy/links |
| `TestPageOnlyFlagPreObjectPending` | Address-remapping accumulation and promotion |
| `TestGrayscaleSetsConvertField` / `TestSmartShrinkingEnableDisable` / `TestBackgroundPDFAndImage` / `TestDumpOutlineGlobalHome` | “One home” routing invariants |
| `TestStubFlagsRemoved` | Policy A: ~20 inert wkhtmltopdf flags must be rejected (dpi, image-dpi, lowquality, use-xserver, cookie-jar, read-args-from-stdin, log-level, javascript-delay, window-status, run-script, debug-javascript, user-style-sheet, minimum-font-size, enable-plugins, produce-forms, enable-javascript, stop-slow-scripts, default-encoding, custom-header-propagation) |
| `TestSimplifyDOM*`, `TestPrintLinkUnderlineFlag` | Web-behaviour flags route global + object |
| `TestUnknownFlagErrors` / `TestDocFlags` / `TestEndOfOptions` | Unknown-flag errors, `--help/-h/-V/--version/-L/--license/-E`, `--` terminator |
| `TestImageFlags` | Image-only flags and defaults |
| `TestParseModeRejects/Accepts/Validation` | Mode bitmask enforcement |
| `TestValidateNoInput` | Missing input file error |
| `TestPageScopedBeforeTOCNoGhost` / `TestPageScopedBeforePageKeyword` | Pending promotion correctness |
| `TestExitCode` | 0/1/2/3 mapping incl. wrapped `HttpStatusError` |
| `TestOpenOutputWriterPrecedence` | `OutputWriter` beats `Output` path |

Beyond unit tests: CI (`make lint`, `make test`) runs golangci-lint
(including `exhaustruct` where intentional partial structs are annotated
with `//nolint:exhaustruct`) and the full suite. End-to-end golden fixtures
(`internal/convert/golden_test.go`, `make samples`, `make golden`) exercise
the full path from settings → PDF and act as an integration gate over CLI
settings that survive Request assembly.

## 10. Known limitations, deferred items & open questions

- **`--extended-help` (-E) currently prints the same text as `--help`** —
  there is no richer extended-flag catalog yet (both sentinels route to
  `PrintHelp` in main; the parser registers the flag via `addDocFlags`).
- **Rejected wkhtmltopdf surface is intentional but real**: any script
  using `--dpi`, `--javascript-*`, `--login-*`, `--run-script`, or
  `--read-args-from-stdin` will fail with `unknown option` (Policy A).
  Projects migrating from wkhtmltopdf should consult
  documentation/compatibility-matrix.md before adopting.
- **No flag aliases / no env-var config / no config file**: the CLI is
  strictly argv-driven; CI users must pass flags explicitly
  (documentation/deferred.md).
- **`--smart-shrinking` has enable/disable pair only** (no bare toggle);
  same for `--background`/`--no-background`, `--print-media-type`/`--no-…`.
- **Multi-object/copies limits are enforced in the engine, not the
  parser**: `convert` caps objects at 10,000, copies at 1,000, pages at
  100,000 (`internal/convert/convert.go` consts) — the CLI can parse
  absurdly large jobs that only fail later.
- **`Version` defaults to `0.1.0`** (same as the `VERSION` file).
  `make build` stamps `gowkhtmltopdf/internal/cli.Version` from that file.
- **No shell completion, no `--read-args-from-stdin`**, no interactive
  prompts — all aligned with the controlled-report scope.
- **Open question**: whether a dedicated `-E` extended help listing (and
  deprecation warnings for wkhtmltopdf-only flags) will be added post-MVP
  (see plans/0.2.0/10-canonical-post-mvp-roadmap.md).

## 11. Related documents

- Sibling deep-dive docs (this directory):
  - 01-entrypoints-cli.md (this document)
  - 02-library-api.md — the root `api.go` path that bypasses argv entirely
  - 03-settings.md — dotted `Set`/`Get`, defaults, UnitReal (what the appliers write)
  - 04-load.md — where ACL, proxy, credentials, cookies, POST are enforced
  - 05-html-parser.md — where loaded bytes become a tree
  - 06-css.md — style sheet handling referenced by `--print-media-type`
  - 07-layout.md — engine that consumes page geometry set by CLI flags
  - 08-convert-pipeline.md — `convert.Request`, `Run`, copies/collate constraints
  - 09-pdf-writer.md — outline/dump artifacts produced at `--dump-outline`
  - 10-imageout-svg.md — the ModeImage sink consumed by `gowkhtmltoimage`
- Repository documentation:
  - documentation/architecture.md — top-level package map and pipeline
  - documentation/cli.md — user-facing CLI reference (grammar, examples, exit codes, remote-URL security)
  - documentation/library-api.md — library equivalents of these flags
  - documentation/fidelity.md — claims language for feature coverage
  - documentation/compatibility-matrix.md — per-flag/per-feature contract
  - documentation/THREAT-MODEL.md — local-file ACL and remote-input threats
  - documentation/deferred.md — explicitly unsupported wkhtmltopdf surface
  - documentation/README.md — full doc index
  - plans/0.1.0/00-canonical-pure-go-rewrite.md — Phase 1 (CLI/settings) ledger
