# Phase 1 — Engine seam & surface

> **Parent:** [`architecture-review-2026-08-07.md`](../architecture-review-2026-08-07.md) — canonical architecture-review ledger
> **Status:** pending (findings gathered 2026-08-07 by 7 explore agents; remediation not started)
> **Depends on:** see phase map in ledger
> **Evidence archive:** raw agent findings were archived off-repo on 2026-08-07; every row below carries its Before/After snippets inline

---

## Overview

…

## Checklist

- [ ] **P1-1** — Give the engines a neutral, CLI-independent `Request` type
- [ ] **P1-2** — Replace the stringly-typed flag value contract (NUL-joined pairs, 3 bool vocabularies)
- [ ] **P1-3** — One flag-routing helper for the pre-object "address remapping" rule (4 implementations)
- [ ] **P1-4** — `--print-media-type`: one resolution rule instead of two pasted copies
- [ ] **P1-5** — Collapse dump-flag homes into one typed field; stop swallowing the bool value
- [ ] **P1-6** — Turn the 843-line reflect.go registry into one field-descriptor table
- [ ] **P1-7** — Move the image↔global `background` aliasing rule into the settings model
- [ ] **P1-8** — Collapse the two near-identical Convert driver bodies into one shared executor
- [ ] **P1-9** — Fix `ImageConverter.Object()` nil-receiver panic before `AddObject`

---

<a id="p1-1"></a>
## P1-1 — Give the engines a neutral, CLI-independent `Request` type

- [ ] **P1-1** — Give the engines a neutral, CLI-independent `Request` type

- **Locations:** `api.go:146-187` (`Converter.Convert`); `api.go:285-315` (`ImageConverter.Convert`); seam at `internal/convert/convert.go:44` and `internal/imageout/imageout.go:407`, `internal/cli/cli.go:28-63`; `internal/convert/convert.go:44-45`; `internal/imageout/imageout.go:407-408`; `api.go:120-126`; `internal/convert/convert_test.go:44-58`
- **Evidence sources:** area-1 F1; area-2 F1

---

### Evidence A — F1

- **Severity:** high
- **Location:** `api.go:146-187` (`Converter.Convert`); `api.go:285-315` (`ImageConverter.Convert`); seam at `internal/convert/convert.go:44` and `internal/imageout/imageout.go:407`
- **Current:**
```go
func (c *Converter) Convert(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(c.objects) == 0 {
		return errors.New("gowkhtmltopdf: no page objects added")
	}
	var buf bytes.Buffer
	cmd := &cli.Command{Global: c.global.g, OutputWriter: &buf}
	for _, o := range c.objects {
		cmd.Objects = append(cmd.Objects, o.o)
	}

	flush := &lineLog{
		onInfo:  c.OnInfo,
		onWarn:  c.OnWarn,
		onError: c.OnError,
	}
	var progress func(phase string, percent int)
	if c.OnPhase != nil || c.OnProgress != nil {
		progress = func(phase string, percent int) {
			if c.OnPhase != nil {
				c.OnPhase(phase)
			}
			if c.OnProgress != nil {
				c.OnProgress(percent)
			}
		}
	}
	if err := convert.RunPDFContext(ctx, cmd, flush, progress); err != nil {
		if c.OnError != nil {
			c.OnError(err.Error())
		}
		return err
	}
	c.output = buf.Bytes()
	return nil
}

// Output returns the PDF bytes produced by the last successful Convert, or
// nil if none ran yet. The returned slice is a copy; it stays valid across
// later conversions.
```
- **Future:** introduce one small engine-side input type that carries exactly what the pipeline consumes, and make the CLI parser an adapter that produces it:
```go
// internal/convert (and imageout consumes the same type or its own slice)
// Request is the pipeline input, independent of the CLI parser. Both
// cmd/* mains (via cli.Command.ToRequest) and the library API build it.
type Request struct {
	Global       settings.PdfGlobal
	Objects      []settings.PdfObject
	Output       io.Writer     // PDF/image bytes sink; nil = stdout
	DumpOutline  bool          // engine-owned dump flags
}

func Run(ctx context.Context, req *Request, log io.Writer, progress func(phase string, percent int)) error

// in internal/cli: parser result converts to the pipeline input.
func (c *Command) ToRequest() *convert.Request {
	return &convert.Request{Global: c.Global, Objects: c.Objects, Output: c.OutputWriterOrFile(), DumpOutline: c.DumpOutline || c.Global.DumpOutline}
}

// api.go then stops importing internal/cli entirely:
func (c *Converter) Convert(ctx context.Context) error {
	// … nil-ctx, no-objects guards unchanged …
	var buf bytes.Buffer
	req := &convert.Request{Global: c.global.g, Objects: c.objectCopies(), Output: &buf}
	if err := convert.Run(ctx, req, c.lineLog(), c.progress()); err != nil {
		c.report(err)
		return err
	}
	c.output = buf.Bytes()
	return nil
}
```
Exported behaviour changes: `convert.RunPDFContext`/`convert.RunPDF` and `imageout.Run` signatures change from `*cli.Command` to `*Request`; callers that must move: `cmd/gowkhtmltopdf/main.go:55`, `cmd/gowkhtmltoimage/main.go:36`, `api.go`, and every test helper that hand-builds `cli.Command` (`internal/convert/convert_test.go:42`, `golden_test.go:96`, `phase6_test.go:124`, `internal/imageout/imageout_test.go:257`, `fontface_test.go` in both packages). The root package's public API is unchanged.
- **Why:** the deepest modules in the repo (convert, imageout) publish an interface shaped as *everything the CLI parser touched*: `Command` carries `Output string`, `OutputWriter`, `DumpDefaultTOCXSL`, `Image settings.ImageGlobal`, `DumpOutline` — the library path only ever uses `Global`, `Objects`, `OutputWriter`, and imageout adds `Image`. The public API is therefore built on top of a CLI parse-result struct (an internal type), so the surface package must know `internal/cli`'s data model, and `cli.Command`'s fields are a union of two CLIs plus the library — every new CLI flag widens the pipeline's interface even when the engine never reads it. This is a seam that leaks implementation types: one seam (the pipeline entry point) with two adapters (CLI parse, library API) whose shared contract is a superset of both. A `Request` type gives the engine one small, stable interface; the CLI parser becomes a pure adapter; api.go imports two internal packages instead of four. hypothesis: most of `Command`'s fields are read by at most one of the three consumers; validate by grepping each `cmd.X` reference in convert/imageout and counting fields used by both library and CLI paths.

### Evidence B — F1

- **Severity:** high
- **Location:** `internal/cli/cli.go:28-63`; `internal/convert/convert.go:44-45`; `internal/imageout/imageout.go:407-408`; `api.go:120-126`; `internal/convert/convert_test.go:44-58`
- **Current (verbatim):**
```go
// Command is the result of parsing argv.
type Command struct {
	Global  settings.PdfGlobal
	Image   settings.ImageGlobal
	Objects []settings.PdfObject
	// Output is a path or "-" (stdout). Ignored when OutputWriter is set.
	Output string
	// OutputWriter, when non-nil, receives PDF/image bytes directly (library
	// path). Takes precedence over Output so embedders need no temp files.
	OutputWriter io.Writer

	DumpDefaultTOCXSL bool
	DumpOutline       bool
}
```
```go
func RunPDFContext(ctx context.Context, cmd *cli.Command, log io.Writer, progress func(phase string, percent int)) error {
```
- **Future:**
```go
// internal/convert — the engine's own config, defined where it is consumed.
type Request struct {
	Global     settings.PdfGlobal
	Objects    []settings.PdfObject
	Output     io.Writer            // engine writes bytes here; no path, no "cmd-"
	DumpOutline bool
	Progress   func(phase string, percent int)
}

func RunPDFContext(ctx context.Context, req Request, log io.Writer) error

// internal/cli — the parser adapts its parse result to the engine seam.
func (cmd *Command) PDFRequest() convert.Request {
	return convert.Request{Global: cmd.Global, Objects: cmd.Objects,
		Output: writer(cmd), DumpOutlineDefault: cmd.DumpOutlineDefault}
}
```
`imageout.Run` gets the mirror `imageout.Request`. The output path/stdout decision moves into one
small writer adapter owned by `main`/`api`; `Command` keeps `OpenOutput` only if `main` needs it.
Callers that must move: `cmd/gowkhtmltopdf/main.go:45`, `cmd/gowkhtmltoimage/main.go:30`,
`api.go:122-143` and `api.go:196-207`, plus the ~6 test files that hand-assemble
`&cli.Command{Global: ..., Objects: ..., Output: ...}` (`convert_test.go:49`, `golden_test.go:118`,
`phase6_test.go:252`, `fontface_test.go:87`, `imageout/fontface_test.go:135`, …).
- **Why:** the seam between "parse argv" and "run the engine" does not exist: the parser struct is
  the engine's configuration interface. `import "gowkhtmltopdf/internal/cli"` appears inside
  `internal/convert` and `internal/imageout`, so the engine depends on the parser instead of the
  reverse — `cli` is the hub of the internal dependency graph. Concretely: the library (`api.go`)
  must build parser objects to call the engine (`cmd := &cli.Command{Global: c.global.g, OutputWriter: &buf}`),
  engine code reads grammar artifacts (`cmd.Output` for format sniffing in `imageout.resolveFormat`),
  file creation (`os.Create`) lives in the parser package, and every engine test has to know the
  perl `Command` shape plus the frozen defaults (`defaultObject` comment: "Callers / CLI must supply
  these"). One adapter is not a seam; today there are two adapters (CLI and the library) both
  forced through the parser's struct. Paying the cost: api.go, both mains, and the test suite every
  time the parser's storage moves; hypothesis: a settings field rename currently forces edits in
  cli + convert + imageout + api — validate by counting files touched by `Global.DumpOutline` moves.

> **Fan-in note:** MERGED from area-1 F1 + area-2 F1: the seam between argv parsing and the engine does not exist — engines, api.go and tests all depend on the CLI parser struct.

---

<a id="p1-2"></a>
## P1-2 — Replace the stringly-typed flag value contract (NUL-joined pairs, 3 bool vocabularies)

- [ ] **P1-2** — Replace the stringly-typed flag value contract (NUL-joined pairs, 3 bool vocabularies)

- **Locations:** `internal/cli/cli.go:236-273`; `internal/cli/flags.go:20,28-35,214-242,402-410`; `internal/settings/reflect.go:187-205` (setBool)
- **Evidence sources:** area-2 F2

---

### Evidence — F2

- **Severity:** high
- **Location:** `internal/cli/cli.go:236-273`; `internal/cli/flags.go:20,28-35,214-242,402-410`; `internal/settings/reflect.go:187-205` (setBool)
- **Current (verbatim):**
```go
const pairSep = "\x00"
...
func isPairFlag(name string) bool {
	switch name {
	case "cookie", "custom-header", "post", "replace":
		return true
	}
	return false
}
```
```go
// apply runs a flag with value extraction (next-arg or =value). Pair flags
// consume two values joined by pairSep.
func apply(c *Command, cur *objectCtx, name string, spec flagSpec, negated bool, inlineVal string, hasInline bool, argv []string, i *int) error {
	if spec.kind == "bool" {
		...
	}
	// value flag
	if hasInline {
		return spec.app(c, cur, inlineVal)
	}
	if *i >= len(argv) {
		return fmt.Errorf("option requires a value")
	}
	val := argv[*i]
	*i++
	if isPairFlag(name) {
		if *i >= len(argv) {
			return fmt.Errorf("option requires two values (name value)")
		}
		val += pairSep + argv[*i]
		*i++
	}
	return spec.app(c, cur, val)
}
```
- **Future:**
```go
type flagKind uint8
const (
	flagBool  flagKind = iota // app receives (c, ctx, bool)
	flagValue                 // app receives (c, ctx, []string{...})
	flagPair                  // app receives exactly two tokens
)

type flagSpec struct {
	kind flagKind
	mod  Mode
	app  func(c *Command, ctx *objectCtx, vals []string) error
}
```
```go
func apply(c *Command, ctx *objectCtx, name string, spec flagSpec, negated bool, inlineVal string, hasInline bool, argv []string, i *int) error {
	switch spec.kind {
	case flagBool:
		b, err := parseBool(inlineVal, negated, hasInline)
		return spec.app(c, ctx, []string{strconv.FormatBool(b)})
	case flagValue:
		vals := []string{inlineVal}
		if !hasInline { ... consume next argv ... }
		return spec.app(c, ctx, vals)
	case flagPair:
		...consume two argv / inline+next...
		return spec.app(c, ctx, []string{k, v})
	}
}
```
`cookie`/`custom-header`/`post`/`replace` then read `vals[0], vals[1]` — no `splitPair`, no
`pairSep`, no `isPairFlag`, no per-flag `if o.Load.Cookies == nil` map-init (a small `setMapEntry`
helper suffices). Bool flags hand the applier a real `bool`; `settings.Set` keeps its string
vocabulary for the compat surface.
- **Why:** the value contract between `apply` and ~90 appliers is *"strings; pairs encoded with
`\x00`"* — a hand-rolled stdlib replacement for a two-element slice, split across two files. Bool
parsing is implemented three times with different semantics: `apply`'s inline switch (errors on
unknown), `boolVal` (silently returns `false` on unknown), and `settings.setBool` (accepts `""`
and "yes"/"on"). A bool flag today round-trips CLI-text → "true"/"false" string → re-parse in
`setBool`, so the two vocabularies can silently diverge (they already differ on `""`). Payload:
anyone adding a flag must learn the NUL convention and remember `isPairFlag` exists as a global
lookup parallel to `spec.kind`. This is the un-idiomatic-Go item in the hunt ("hand-rolled
stdlib").

---

<a id="p1-3"></a>
## P1-3 — One flag-routing helper for the pre-object "address remapping" rule (4 implementations)

- [ ] **P1-3** — One flag-routing helper for the pre-object "address remapping" rule (4 implementations)

- **Locations:** `internal/cli/flags.go:344-400` (pageScoped / hfFlag / tocFlag / printMediaFlag); `internal/cli/cli.go:64-100` (objectCtx.pending promotion); `internal/cli/flags.go:168-199` (no-op global closures)
- **Evidence sources:** area-2 F3

---

### Evidence — F3

## F3: One flag-routing helper for the pre-object rule — the "address remapping" invariant is reimplemented four ways today
- **Severity:** high
- **Location:** `internal/cli/flags.go:344-400` (pageScoped / hfFlag / tocFlag / printMediaFlag); `internal/cli/cli.go:64-100` (objectCtx.pending promotion); `internal/cli/flags.go:168-199` (no-op global closures)
- **Current (verbatim):**
```go
// pageScoped routes a flag to the current object when one exists, else
// accumulates it as pending first-page settings (upstream address remapping:
// page settings before any object keyword apply to the first page).
func pageScoped(glob func(g *settings.PdfGlobal, val string) error, obj func(o *settings.PdfObject, val string) error) flagApplier {
	return func(c *Command, cur *objectCtx, val string) error {
		if err := glob(&c.Global, val); err != nil {
			return err
		}
		if cur.obj != nil {
			return obj(cur.obj, val)
		}
		if cur.pending == nil {
			o := settings.DefaultPdfObject()
			cur.pending = &o
		}
		return obj(cur.pending, val)
	}
}

// two of the six page-only flags that smuggle a silent no-op as the "global" half:
add("username", ModeBoth, "value", pageScoped(
	func(g *settings.PdfGlobal, val string) error { return nil },
	func(o *settings.PdfObject, val string) error { return o.Set("load.username", val) },
))
add("timeout", ModeBoth, "value", pageScoped(
	func(g *settings.PdfGlobal, val string) error { return nil },
	func(o *settings.PdfObject, val string) error { return o.Set("load.timeout", val) },
))
```
```go
// hfFlag targets header.* or footer.* on the current object or global.
func hfFlag(prefix, field, kind string) flagApplier {
	return func(c *Command, cur *objectCtx, val string) error {
		key := prefix + "." + field
		if kind == "bool" {
			val = boolVal(val, "true", "false")
		}
		if cur.obj != nil {
			return cur.obj.Set(key, val)
		}
		return c.Global.Set(key, val)
	}
}
```
- **Future:**
```go
// one router owns the pre-object rule; every "page-ish" flag goes through it.
func (ctx *objectCtx) applyPage(c *Command, glob func(g *settings.PdfGlobal, val string) error,
	obj func(o *settings.PdfObject, val string) error, val string) error {
	if err := glob(&c.Global, val); err != nil {
		return err
	}
	if ctx.obj != nil {
		return obj(ctx.obj, val)
	}
	if ctx.pending == nil {
		o := settings.DefaultPdfObject()
		ctx.pending = &o
	}
	return obj(ctx.pending, val)
}

// page-only flags (zoom, username, password, timeout, external/internal-links)
// use pageOnlyFlag, which rejects the pre-object position loudly when there
// is no global consumer, instead of silently dropping the value:
func pageOnlyFlag(obj func(o *settings.PdfObject, val string) error) flagApplier { ... }
```
` hfFlag`/`tocFlag`/`printMediaFlag` become thin wrappers over the same router (e.g. hf/toc
explicitly set *global-only* routing instead of silently diverging); the hand-rolled `pending`
incl. creation inside `printMediaFlag` is deleted.
- **Why:** the upstream "address remapping" rule ("page settings before any object keyword apply to
  the first page") is documented once in `pageScoped`'s comment but implemented four different
  ways in the same file: `pageScoped` promotes a pending object and also writes global;
  `printMediaFlag` hand-builds its own `pending` (and writes three fields); `hfFlag`/`tocFlag`
  skip pending entirely and fall back to *global* storage. So the very same syntactic position
  (`--flag value in.html out.pdf`) routes differently per flag family — for a two-page invocation
  `--header-left X in1.html in2.html out.pdf` stamps the header on both pages, while
  `--username U in1.html in2.html out.pdf` stamps only page 1 — and six flags (`zoom`, `username`,
  `password`, `timeout`, `external-links`, `internal-links`) degrade the *global* half of
  `pageScoped` to `return nil`, silently absorbing anything before the first object.
  hypothesis: `--zoom 2 toc page a.html out.pdf` silently drops the zoom if a `toc` intervenes —
  validate by parsing that exact argv and diffing `Objects[0].Load.ZoomFactor`. A single router +
  an explicit `pageOnlyFlag` makes the rule live in one place and turns silent no-ops into parse
  errors.

---

<a id="p1-4"></a>
## P1-4 — `--print-media-type`: one resolution rule instead of two pasted copies

- [ ] **P1-4** — `--print-media-type`: one resolution rule instead of two pasted copies

- **Locations:** `internal/cli/flags.go:305-327` (printMediaFlag); `internal/convert/convert.go:498-526` (mediaFor); `internal/imageout/imageout.go:547-562` (mediaFor)
- **Evidence sources:** area-2 F4

---

### Evidence — F4

- **Severity:** medium
- **Location:** `internal/cli/flags.go:305-327` (printMediaFlag); `internal/convert/convert.go:498-526` (mediaFor); `internal/imageout/imageout.go:547-562` (mediaFor)
- **Current (verbatim):**
```go
func printMediaFlag(enable bool) flagApplier {
	return func(c *Command, cur *objectCtx, val string) error {
		on := enable
		if enable {
			on = boolVal(val, "true", "false") == "true"
		}
		c.Image.Web.PrintMediaType = on
		c.Global.Web.PrintMediaType = on
		if cur.obj != nil {
			cur.obj.Load.PrintMediaType = on
			return nil
		}
		if cur.pending == nil {
			o := settings.DefaultPdfObject()
			cur.pending = &o
		}
		cur.pending.Load.PrintMediaType = on
		return nil
	}
}
```
```go
func mediaFor(g settings.PdfGlobal, obj *settings.PdfObject) string {
	media := "print"
	if g.Web.PrintMediaType {
		media = "print"
	}
	switch g.Web.MediaType {
	case settings.MediaPrint:
		media = "print"
	case settings.MediaScreen:
		media = "screen"
	}
	if obj == nil {
		return media
	}
	if obj.Load.PrintMediaType {
		media = "print"
	}
	switch obj.Load.MediaType {
	case settings.MediaPrint:
		media = "print"
	case settings.MediaScreen:
		media = "screen"
	}
	return media
}
```
- **Future:** one typed home for the flag plus one shared resolver in the place the data lives:
```go
// internal/settings — near Web/MediaType, where the invariant belongs.
// base is the mode default ("print" for PDF, "screen" for image).
func ResolveMedia(base string, global Web, obj *Web) string {
	switch {
	case obj.PrintMediaType || global.PrintMediaType:
		return "print"
	}
	switch obj.MediaType {
	case MediaPrint:
		return "print"
	case MediaScreen:
		return "screen"
	case MediaIgnore:
		if obj.MediaType == 0 { return base } // per-mode fallthrough below
	}
	switch global.MediaType {
	case MediaPrint:
		return "print"
	case MediaScreen:
		return "screen"
	}
	return base
}
```
```go
// convert.go
media := settings.ResolveMedia("print", cmd.Global.Web, obj.Load)
// imageout.go
media := settings.ResolveMedia("screen", cmd.Global.Web, obj.Load)
```
and `printMediaFlag` writes one field (`c.Global.Web.PrintMediaType`) + the object loader override
through one router (delete the `Image.Web.PrintMediaType` leg; `ImageConverter.Set` / image mode
routes any `Set("web.printmediatype", …)` to the same single home, mirroring the existing
`web.background` routing in `api.go`).
- **Why:** a single command-line boolean is stored in three places (`Image.Web.PrintMediaType`,
  `Global.Web.PrintMediaType`, `obj.Load.PrintMediaType`) and each consumer reads a *different*
  subset (`convert.mediaFor`: global + obj; `imageout.mediaFor`: image + obj — with the comment in
  flags.go documenting the split). The resolution rule then lives twice in two packages, and they
  have already drifted: image mode maps `MediaIgnore` to `""`, PDF defaults "print", image
  defaults "screen". Anyone adding a media flag must update the flag, both `mediaFor` bodies, and
  both storage decisions. hypothesis: a future third consumer (e.g. CSS `@media` collection in
  convert vs imageout) cannot tell which home to read — validate by counting `PrintMediaType`
  references across packages today (≥6).

---

<a id="p1-5"></a>
## P1-5 — Collapse dump-flag homes into one typed field; stop swallowing the bool value

- [ ] **P1-5** — Collapse dump-flag homes into one typed field; stop swallowing the bool value

- **Locations:** `internal/cli/flags.go:112-121`; `internal/convert/convert.go:130`; `cmd/gowkhtmltopdf/main.go:37-44`; `internal/settings/reflect.go:866-873` & `874-881` (keys `dumpoutline`, `dumpoutlinewithdefaulttocxsl`)
- **Evidence sources:** area-2 F5

---

### Evidence — F5

- **Severity:** medium
- **Location:** `internal/cli/flags.go:112-121`; `internal/convert/convert.go:130`; `cmd/gowkhtmltopdf/main.go:37-44`; `internal/settings/reflect.go:866-873` & `874-881` (keys `dumpoutline`, `dumpoutlinewithdefaulttocxsl`)
- **Current (verbatim):**
```go
	// Dump home: Command fields only. convert ORs cmd.DumpOutline with
	// Global.DumpOutline (library/reflect path); main uses DumpDefaultTOCXSL.
	add("dump-outline", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		c.DumpOutline = true
		return nil
	})
	add("dump-default-toc-xsl", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		c.DumpDefaultTOCXSL = true
		return nil
	})
```
```go
// internal/convert/convert.go:130
		if cmd.DumpOutline || cmd.Global.DumpOutline {
```
- **Future:**
```go
	// one home: Global settings (CLI and library both write it); engine reads it only.
	add("dump-outline", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("dumpoutline", val)
	})
	add("dump-default-toc-xsl", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("dumpoutlinewithdefaulttocxsl", val)
	})
```
```go
// convert.go:130 becomes
	if cmd.Global.DumpOutline {
```
and `cmd/gowkhtmltopdf/main.go:39` reads `cmd.Global.DumpDefaultTOCXSL`. Callers that must move:
`cmd/gowkhtmltopdf/main.go`, the two assertions in `internal/cli/cli_test.go:66-78` which pin
"must not dual-write Global", and every `cmd.DumpOutline = true` in real code/tests.
- **Why:** "is dump enabled" is one logical setting stored in two places (`Command.Dump…` and
  `Global.Dump…`) whose union only the engine knows (an OR in convert). The CLI appliers also
  delete the bool value entirely (`c.DumpOutline = true` regardless of `val`), so `--no-dump-outline`
  and `--dump-outline=false` still dump — negation lookups exist in `lookupFlag` but the applier
  ignores the negation. Meanwhile `settings` exposes a typed `dumpoutlinewithdefaulttocx` key that
  nobody ever reads (main deliberately says "Do not also read Global.DumpDefaultTOCXSL"), which
  breaks Policy A (typed keys must have engine consumers). Collapsing to one storage makes the
  setting honest, restores negation, and deletes the special case inside the engine.

---

<a id="p1-6"></a>
## P1-6 — Turn the 843-line reflect.go registry into one field-descriptor table

- [ ] **P1-6** — Turn the 843-line reflect.go registry into one field-descriptor table

- **Locations:** `internal/settings/reflect.go:124-177,327-523,669-831`; `internal/settings/getters.go:13-55`
- **Evidence sources:** area-2 F6

---

### Evidence — F6

- **Severity:** medium
- **Location:** `internal/settings/reflect.go:124-177,327-523,669-831`; `internal/settings/getters.go:13-55`
- **Current (verbatim):**
```go
// Get reads a dotted global key as its canonical string form. ok is false
// for unknown keys. Accepted ignored keys return the last Set value.
func (g *PdfGlobal) Get(name string) (string, bool) {
	ensureKeyTables()
	key := normalizeDots(name)
	if fn, ok := globalGetTable[key]; ok {
		return fn(g)
	}
	if g.Ignored != nil {
		if v, ok := g.Ignored[key]; ok {
			return v, true
		}
	}
	return "", false
}
```
```go
	for _, k := range []string{"fontsize", "fontname", "left", "right", "center", "line", "spacing", "htmlurl"} {
		key := k
		gReg(set, get, "header."+key,
			func(g *PdfGlobal, raw string) error { return hfApply(&g.Header, key, raw) },
			func(g *PdfGlobal) (string, bool) { return hfGet(&g.Header, key) },
		)
		gReg(set, get, "footer."+key,
			func(g *PdfGlobal, raw string) error { return hfApply(&g.Footer, key, raw) },
			func(g *PdfGlobal) (string, bool) { return hfGet(&g.Footer, key) },
		)
	}
```
```go
func hfApply(hf *HeaderFooter, key, raw string) error {
	switch key {
	case "fontsize":
		return setFloat(&hf.FontSize)(raw)
	case "fontname":
		return setString(&hf.FontName)(raw)
	...
```
- **Future:**
```go
// one descriptor per key; the name appears once, the apply+get fn per type once.
type field struct {
	apply func(*Tlf, string) error   // generic over                                     C
	get   func(*Tlf) (string, bool)
}
var globalKeys = keyTable[PdfGlobal]{
	"header.fontsize": {apply: hf("fontsize", setFloat), get: hfFloat("fontsize")},
	...
}

// Go 1.26 generics collapse the 6 Set/Get copies:
func (g *PdfGlobal) Set(name, value string) error { return setForKey(globalKeys, &g.Ignored, name, value) }
func (g *PdfGlobal) Get(name string) (string, bool) { return getForKey(globalKeys, &g.Ignored, name) }
```
```go
func hf(name string, setter setter) applyFn { return func(h *HeaderFooter, raw string) error { ... } }
```
(no new dependencies; tables remain `var` + `init`); getters.go's three `Get` bodies and
reflect.go's three `Set` bodies become one generic pair each; the eight parallel
apply/get switches (hf/toc/web/load × apply/get) become four small descriptor tables.
- **Why:** today the *same key name* is spelled three times per setting — once in the
  `for _, k := range []string{…}` registration loop (2× for header/footer), once as a `case` in the
  apply helper, once as a `case` in the get helper — and two of the 8 helpers are 20+ lines of
  verbatim switch. Every settings author must touch 3-4 locations to add one key, and the
  apply/get must not drift. The three `Get` bodies and three `Set` bodies are byte-identical
  modulo table names; a generic lookup kills them. This is the biggest repetitive-plumbing and
  export-one-true-tables hotspot in Settings; it also moves the registration-accidental
  `ensureKeyTables()`/`sync.Once` lazy init into plain package-level tables.

---

<a id="p1-7"></a>
## P1-7 — Move the image↔global `background` aliasing rule into the settings model

- [ ] **P1-7** — Move the image↔global `background` aliasing rule into the settings model

- **Locations:** `api.go:249-261` (`ImageConverter.Set`); rule duplicated in `internal/settings/reflect.go:655`, `internal/settings/doc.go`, `internal/imageout/imageout.go`
- **Evidence sources:** area-1 F5

---

### Evidence — F5

- **Severity:** medium
- **Location:** `api.go:249-261` (`ImageConverter.Set`); rule duplicated in `internal/settings/reflect.go:655`, `internal/settings/doc.go`, `internal/imageout/imageout.go`
- **Current:**
```go
// switch so image and PDF share one background field.
func (c *ImageConverter) Set(name, value string) error {
	key := strings.ToLower(strings.TrimSpace(name))
	// Sole paint field is Global.Background (image has no Web.Background).
	if key == "web.background" || key == "background" {
		return c.global.g.Set("background", value)
	}
	return c.image.Set(name, value)
}

// Global returns the shared global settings (only "enablelocalfileaccess"
// and "allow" influence image conversion, via the loader ACL).
func (c *ImageConverter) Global() *GlobalSettings {
```
- **Future:** keep the decision in the package that owns the model; api.go becomes a one-line delegate:
```go
// internal/settings: the image key table's routing helper owns the
// background alias (no new typed fields — the paint switch stays
// PdfGlobal.Background, per Policy A).
func ApplyImageKey(global *PdfGlobal, img *ImageGlobal, name, value string) error {
	switch normalizeDots(strings.TrimSpace(name)) {
	case "background", "web.background":
		return global.Set("background", value)
	default:
		return img.Set(name, value)
	}
}

// api.go
func (c *ImageConverter) Set(name, value string) error {
	return settings.ApplyImageKey(&c.global.g, &c.image, name, value)
}
```
API-compatible; no caller moves.
- **Why:** the rule "image mode has no Web.Background; the paint switch is PdfGlobal.Background" is stated in at least four places — `api.go:250-253` (the only *executable* copy), `internal/settings/reflect.go:655` ("image web.background is not typed here — ImageConverter.Set routes it"), `internal/settings/doc.go` ("Body paint background is PdfGlobal.Background only"), and `internal/imageout/imageout.go` ("body paint background is Global.Background only"). The executable copy lives in the *public root package*, whose job is presentation, not settings-model policy; when the settings model changes (say the alias set grows), only api.go must be updated — or worse, forgotten. Settings already owns this class of alias (`colormode`/`grayscale` → `Grayscale`); routing through `settings.ApplyImageKey` puts rule and action in one place and makes the root a pure adapter. hypothesis: a future image-mode key that must also alias a global (e.g. `quiet`) will be implemented in api.go again, widening the root's settings knowledge; validate by checking how many settings cross-references the root already contains.

---

<a id="p1-8"></a>
## P1-8 — Collapse the two near-identical Convert driver bodies into one shared executor

- [ ] **P1-8** — Collapse the two near-identical Convert driver bodies into one shared executor

- **Locations:** `api.go:146-187` (`Converter.Convert`) vs `api.go:285-315` (`ImageConverter.Convert`)
- **Evidence sources:** area-1 F2

---

### Evidence — F2

- **Severity:** medium
- **Location:** `api.go:146-187` (`Converter.Convert`) vs `api.go:285-315` (`ImageConverter.Convert`)
- **Current:**
```go
func (c *Converter) Convert(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(c.objects) == 0 {
		return errors.New("gowkhtmltopdf: no page objects added")
	}
	var buf bytes.Buffer
	cmd := &cli.Command{Global: c.global.g, OutputWriter: &buf}
	for _, o := range c.objects {
		cmd.Objects = append(cmd.Objects, o.o)
	}

	flush := &lineLog{
		onInfo:  c.OnInfo,
		onWarn:  c.OnWarn,
		onError: c.OnError,
	}
…
	if err := convert.RunPDFContext(ctx, cmd, flush, progress); err != nil {
		if c.OnError != nil {
			c.OnError(err.Error())
		}
		return err
	}
	c.output = buf.Bytes()
```
```go
func (c *ImageConverter) Convert(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.object == nil || strings.TrimSpace(c.object.o.Page) == "" {
		return errors.New("gowkhtmltopdf: no input page added")
	}
	var buf bytes.Buffer
	cmd := &cli.Command{
		Global:       c.global.g,
		Image:        c.image,
		Objects:      []settings.PdfObject{c.object.o},
		OutputWriter: &buf,
	}
	flush := &lineLog{
		onInfo:  c.OnInfo,
		onWarn:  c.OnWarn,
		onError: c.OnError,
	}
	if err := imageout.Run(ctx, cmd, flush); err != nil {
		if c.OnError != nil {
			c.OnError(err.Error())
```
- **Future:** one private driver, both converters become thin configuration holders:
```go
// execute runs one conversion: builds the pipeline request from the
// settings wrappers, wires log/progress hooks, forwards failures to
// OnError, and captures the output bytes.
func (h *hooks) execute(ctx context.Context, req *convert.Request) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var buf bytes.Buffer
	req.Output = &buf
	if err := convert.Run(ctx, req, h.lineLog(), h.progress()); err != nil {
		if h.OnError != nil {
			h.OnError(err.Error())
		}
		return nil, err
	}
	return buf.Bytes(), nil
}
// Converter.Convert and ImageConverter.Convert each reduce to building
// their Request and calling h.execute(ctx, req) — ~6 lines each.
```
API-compatible; no caller moves.
- **Why:** the two bodies duplicate the same plumbing: nil-ctx guard, no-input check, `bytes.Buffer` + `cli.Command{…OutputWriter}` assembly, `lineLog` construction from the three callbacks, `err → OnError` forwarding, `buf.Bytes()` capture. The only differences are which Run to call and which settings fields go in. Today a fix to one driver (e.g. the progress-closure wiring that only PDF has, or the `OnError` forwarding) must be repeated in the other; the duplicated `OnError` forwarding already diverges in phrasing (PDF wraps the error through `OnError` only for run failures, image likewise, but the callback plumbing is copy-pasted). One executor concentrates callback wiring and output capture in one place — locality for the surface's only real logic.

---

<a id="p1-9"></a>
## P1-9 — Fix `ImageConverter.Object()` nil-receiver panic before `AddObject`

- [ ] **P1-9** — Fix `ImageConverter.Object()` nil-receiver panic before `AddObject`

- **Locations:** `api.go:268-283`
- **Evidence sources:** area-1 F7

---

### Evidence — F7

- **Severity:** low
- **Location:** `api.go:268-283`
- **Current:**
```go
func (c *ImageConverter) Object() *ObjectSettings {
	return c.object
}

// Convert runs the conversion, replacing the previous Output. ctx is threaded
// into the load; cancel it to abort. Errors are also reported to OnError.
```
- **Future:** seed the object in the constructor so `Object()` is always valid; the no-page check stays on the page *content*:
```go
func NewImageConverter() *ImageConverter {
	return &ImageConverter{
		global: NewGlobalSettings(),
		image:  settings.DefaultImageGlobal(),
		object: NewObjectSettings(), // always present; Page empty until AddObject
	}
}
// Convert's guard becomes: if strings.TrimSpace(c.object.o.Page) == "" { … }
```
API-compatible (a nil `Object()` was never documented as a feature); no caller moves.
- **Why:** `Object()` is a public accessor whose doc promises "the settings of the page to convert", but before `AddObject` it returns nil and `c.Object().Set("load.blocklocalfileaccess", "false")` panics with a nil-pointer dereference — a testability trap at the seam (the state "no object yet" is representable even though `Convert` treats it as an error). Seeding the object in the constructor makes the invariant "Object is always a valid settings handle" true everywhere, so the only invalid state is an empty page, which `Convert` already rejects with a real error.

---
