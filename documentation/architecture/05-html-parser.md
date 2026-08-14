# HTML parser: allowlisted tokenizer & tree builder

## 1. Responsibility & position in the pipeline

`internal/html` is the **parse** stage of the `load → parse → style → layout →
paginate → paint → write` pipeline. It turns the decoded bytes of one document
(`<body>` string, or raw UTF-8 document bytes) into an in-memory DOM tree that
every downstream stage walks:

- `internal/css` **matches selectors against the tree** (`css.Match`, `:has`,
  `:nth-child`, attribute selectors, …);
- `internal/convert/prepare` **walks the tree** to collect `<link
  rel=stylesheet>` / `<style>` sheets and `@font-face` fonts;
- `internal/layout` **resolves styles per node and walks the tree to build the
  box tree / display list**;
- `internal/outline` **walks the tree** to collect `h1..h6` headings for PDF
  bookmarks;
- `internal/convert` re-uses the tree for **TOC generation**, **header/footer
  HTML**, **relative-link resolution**, and the **benchmark "page islands"**
  fast path.

The package is deliberately *not* a full HTML5 parser and *not* a browser DOM.
It implements the **HTML subset gowkhtmltopdf accepts** — tags, attributes,
text, comments, doctype, self-closing and void elements — with a hand-rolled
stack-based tree builder, and makes **no attempt at browser-grade error
recovery** (package doc, `internal/html/html.go:1-10`):

> *"No browser-grade error recovery: common malformed nesting degrades to a
> usable tree, not a crash."*

Two entry points cover the whole repo:

- `Parse(source string)` — parse a Go string; used by header/footer HTML and
  the generated TOC HTML (`internal/convert/hf.go:267,421`,
  `internal/convert/toc.go:192`).
- `ParseDocument(body []byte)` — parse raw document bytes, stripping a leading
  UTF-8 BOM; used by the main document path
  (`internal/convert/prepare/prepare.go:154`).

The package holds the **only DOM representation in the codebase** — there is no
x/net/html and no other tree type. Every other package imports it (see §5),
which makes the `Node` shape a de-facto cross-cutting contract.

## 2. Package / file map

| File | Responsibility | Approx. lines |
|------|----------------|---------------|
| `internal/html/html.go` | Tokenizer (`scanTokens` + per-construct scanners), tree builder (`Parse`, `ParseDocument`, `appendToken` + helpers), node model (`Node`, `NodeType`), tree walkers (`Walk`, `TextContent`, `TextContentOf`), element vocabularies (`isVoidElement`, `isRawTextElement`, `autoClose` map) | 832 |
| `internal/html/entities.go` | `UnescapeEntities` — thin `Contains("&")` gate over stdlib `html.UnescapeString` for named + numeric character references | 18 |
| `internal/html/doc.go` | Package doc indirection (doc lives in html.go) | 2 |
| `internal/html/html_test.go` | Same-package tests: tokenizer tests, tree-builder tests, malformed-input tests, walker tests; helpers `mustParse`, `treeString`, `assertChildren` | 901 |

Total: ~1,753 lines (roughly half test code — the tokenizer/tree surface is
densely covered).

There is **no separate selector engine** in this package: selector *parsing*
and *matching* live in `internal/css` (§5), which consumes the `Node` tree
produced here.

## 3. Key types, functions & entry points

### 3.1 Node model (`html.go:26-115`)

```go
type NodeType int        // html.go:26
const (
    ElementNode NodeType = iota  // 0
    TextNode                    // 1
    CommentNode                 // 2
    DoctypeNode                 // 3
)

type Node struct {        // html.go:36
    Type     NodeType
    Name     string            // element name, lowercased by tokenizer
    Attrs    map[string]string // attribute name → value, keys lowercased
    Text     string            // text/comment/doctype content
    Children []*Node
    Parent   *Node
}
```

Key methods:

| Member | Line | Purpose |
|--------|------|---------|
| `(*Node) Attribute(name string) string` | 48 | Case-insensitive attribute lookup; fast path when name is already lowercase, `ToLower` fallback otherwise (e.g. CSS `attr(NAME)`). Missing attr → `""`. |
| `(*Node) FirstChild(name string) *Node` | 59 | First **element** child with the given name, or nil. |
| `(*Node) TextContent() string` | 70 | Concatenated descendant text (comments/doctype contribute nothing). |
| `(*Node) Walk(f func(*Node))` | 79 | Pre-order (document-order) recursive walk of node + all descendants. **The main iteration primitive used by every consumer.** |
| `(*Node) TextContentOf(name string) string` | 89 | Text content of the *first* element descendant with that name (used e.g. for `<title>`). |

### 3.2 Public entry points

| Function | Line | Purpose |
|----------|------|---------|
| `Parse(source string) (*Node, error)` | 118 | Scan a string into a tree with a synthetic root element named `#document`. Streaming: `scanTokens` emits tokens to a callback that builds the tree incrementally (no whole-token slice is retained). |
| `ParseDocument(body []byte) (*Node, error)` | 293 | Bytes → tree; strips a leading UTF-8 BOM (mirrors `load.IsHTML`, `internal/load/load.go:182`). |

### 3.3 Tokenizer internals

| Symbol | Line | Purpose |
|--------|------|---------|
| `type tokenKind` / `tokDoctype, tokStart, tokEnd, tokText, tokComment` | 378-385 | Token classification. |
| `type token struct { kind; data string; attrs []string; selfClosing bool }` | 397 | Token payload. `attrs` is an **interleaved name,value slice** (not a map) to avoid per-token map allocation. |
| `type tokenSink func(token)` | 405 | Push-style token consumer (streaming). |
| `tokenize(src) ([]token, error)` | 409 | Test-only collector: buffers all tokens via the sink (used by tokenizer-level tests). |
| `scanTokens(src, emit)` | 425 | Main scanner loop: walks the source, splitting on `<`, dispatching to per-construct scanners. Bare `<` not followed by a valid construct becomes text. |
| `scanBang(src, pos, emit)` | 475 | `<!-- comment -->`, case-insensitive `<!doctype ...>`, or **bogus declarations skipped to `>`**. |
| `scanEndTag(src, pos, emit)` | 511 | `</name>`; name trimmed + lowercased via `endTagName` (529). |
| `scanPI(src, pos)` | 586 | Skips `<?...?>` processing instructions (no token emitted). |
| `scanStartTag(src, pos, emit)` | 597 | Start tags + attribute parsing + **raw-text element content capture** (see §3.4). |
| `tagEnd(src, start)` | 655 | Finds the closing `>` respecting quoted attribute values; `-1` if never closed. |
| `rawTextEnd(src, from, name)` | 679 | Byte-wise search for the real closing tag of a raw-text element (no lowered copy, no needle string). |
| `parseTag(body)` | 725 | Splits `<...>` body into name + interleaved attrs + self-close flag. |
| `nextAttr(rest)` / `attrTail(rest)` | 771 / 805 | One attribute per call: name up to `=`/whitespace; quoted or unquoted value. |

### 3.4 Tree builder internals

| Function | Line | Purpose |
|----------|------|---------|
| `appendToken(stack, tok)` | 133 | Dispatches one token onto the open-element stack. |
| `appendTextToken` | 162 | Entity-decodes text (`UnescapeEntities`) and **merges adjacent text nodes** into one node. |
| `openElement` | 186 | Creates the element node, applies attrs, pushes onto stack unless void/self-closing. |
| `mergeRootElement` | 219 | `html`/`head`/`body` duplicates **merge** into the existing same-name element instead of nesting (browser-style behavior). |
| `autoCloseOpen` | 239 | Pops open elements a new start tag implicitly closes (table rows/cells, `p`, `li`, `dd/dt`, section rows; see `autoClose` map at 348). |
| `applyAttributes` | 265 | Stores attrs on the node; **first value of a duplicated attribute wins**; values entity-decoded. |
| `closeElement` | 278 | Pops the stack back to (and including) the first open element with that name; **stray end tags are a no-op**. |
| `isVoidElement(name)` | 303 | `area, base, br, col, embed, hr, img, input, link, meta, param, source, track, wbr`. |
| `isRawTextElement(name)` | 315 | `script, style, textarea, title` — content captured verbatim up to the closing tag. |

### 3.5 Error model

Six sentinel errors (`html.go:16-24`) that callers can match/wrap:

`errUnterminatedComment`, `errUnterminatedDoctype`, `errUnterminatedDecl`,
`errUnterminatedEndTag`, `errUnterminatedPI`, `errUnterminatedAttrVal`.

Note the asymmetry: an **unterminated start tag** (`<p ...` with no `>`) does
*not* error — `scanStartTag` treats the rest of the source as text
(`html.go:600-604`). Unterminated raw-text elements similarly degrade: the
remaining content becomes a text token and the tree is returned
(`html.go:664-668`). The philosophy is "degrade to a usable tree, not a crash."

## 4. Data & control flow

### 4.1 Main document path

```text
internal/convert/prepare/prepare.go:154
    prepare.Document(ctx, loader, page, ...)
        → loader.Load(ctx, page, loadPage)      // bytes + charset gate (UTF-8/ASCII only)
        → html.ParseDocument(res.Body)          // BOM strip + Parse
        → root *html.Node                       // synthetic "#document" root
        → Resources.CollectSheets(ctx, root, …) // root.Walk: <style>, <link>, inline style attrs
        → Sheets []*css.Stylesheet
        → AppendSimplifySheet(sheets, …)        // optional "chrome" display:none sheet
        → mergeFontFaces(ctx, …)                // @font-face discovery + registry merge
        → Prepared{Root, Sheets, Registry, …}
```

- **Charset seam:** `internal/load` enforces UTF-8/ASCII-only *before* the
  parser ever sees bytes (`load.checkDocumentCharset`,
  `internal/load/load.go:492-520`; charset from Content-Type, falling back to a
  `<meta charset>` scan via `metaCharset` at `load.go:540`). The parser
  therefore always receives valid UTF-8/ASCII — decoding is never its job.
- **BOM mirror:** `ParseDocument` strips `\ufeff` the same way
  `load.IsHTML` recognizes inline HTML (`load.go:182-186`), so both seams agree
  on what "starts with HTML" means.

### 4.2 Tokenizer → tree flow (inside `Parse`)

```text
Parse(source)
  └─ scanTokens(source, emit)          // streaming, one pass, no token slice
       ├─ '<' text runs        → emit(tokText)
       ├─ '<!--'               → scanBang  → emit(tokComment)
       ├─ '<!doctype' (case-insensitive) → emit(tokDoctype)
       ├─ '<!bogus'            → scanBang  → skipped to '>'
       ├─ '<?...?>'            → scanPI    → skipped (no token)
       ├─ '</name>'            → scanEndTag→ emit(tokEnd)
       ├─ '<name ...>'         → scanStartTag → emit(tokStart)
       │     └─ raw-text element → capture content → emit(tokText) → emit(tokEnd)
       └─ bare '<'             → text
  └─ appendToken(&stack, tok)          // builds Node tree on the fly
       ├─ openElement / mergeRootElement / autoCloseOpen / closeElement
       ├─ appendTextToken (entity-decode + adjacent-merge)
       └─ doctype/comment attach
  └─ root Node "#document"
```

Key design point: `Parse` uses `scanTokens` with a **sink callback**, so a
document never materializes a whole token slice in memory (the `tokenize`
collector at `html.go:409` exists only for tests). Tree building and scanning
are interleaved in a single pass.

### 4.3 Consumer flows (how the tree is walked downstream)

- **CSS matching** — `internal/css` walks/tests nodes directly:
  `css.Match(sel, node)` (`internal/css/css.go:1216`), `:has` via
  `matchPseudoWalk`, structural pseudo-classes via parent/sibling pointer
  walks (`css.go:1237-1540`). The synthetic `#document` root is special-cased:
  `isRootElement` (`css.go:1461-1478`) ensures it never matches and never
  blocks `<html>` from matching.
- **Style resolution** — `internal/layout`: `resolveStylesCtx` walks the tree
  computing `map[*html.Node]*ResolvedStyle` (`internal/layout/style.go:350`);
  `matchedRules` gathers cascaded rule hits per node
  (`internal/layout/style_cascade.go:205`); inline `style` attributes are
  parsed per node (`style_cascade.go:337`, via `css.ParseInline`).
- **Layout** — the engine walks the tree to build boxes and the display list:
  `Layout(root, opts)` (`internal/layout/layout.go:696`), `flowChildren`
  (`layout_flow.go:124`), `collectInline` (`inline_collect.go:11`), `buildTable`
  (`layout_tables.go:9`). ElementLocation keeps the originating `*html.Node`
  (`layout.go:238-246`) so links/outline can map paint ops back to DOM nodes.
- **Outline / bookmarks** — `internal/outline`: `CollectHeadings(root)`
  walks and collects `h1..h6` (`internal/outline/outline.go:121-124`); nodes
  are retained in `outline.Location{Node: *html.Node, …}` (`outline.go:22-36`).
- **Title** — `docTitle(root)` reads `root.TextContentOf("title")`
  (`internal/convert/outline.go:85-100`).
- **Headers/footers & TOC** — HF HTML strings are parsed separately with
  `html.Parse` (`internal/convert/hf.go:267,421`); TOC HTML is a generated
  template string parsed at `internal/convert/toc.go:192` (the converter
  *generates* HTML and re-enters the pipeline at the parse seam — a neat
  demonstration of the parser as the shared input contract).
- **Benchmark islands** — the "page islands" fast path clones the tree shell
  (`cloneShell`/`Root`, `internal/convert/islands/plan.go:101-131`) to render
  benchmark-report sections in parallel; it relies on `FirstChild("html")`,
  `FirstChild("body")`, `Walk`, `TextContentOf("title")`
  (`islands/plan.go:62-77`).

## 5. Cross-package dependencies

### 5.1 What `internal/html` imports

- **Standard library only**: `errors`, `strings` (`html.go:11-13`) and
  `html` (aliased `stdhtml`) + `strings` in `entities.go:4-6`.
- **Zero internal imports, zero third-party imports.** This makes the package a
  dependency leaf — nothing in the repo can depend on the HTML parser
  depending on anything else, and the import graph cannot cycle through it.

This is intentional and load-bearing: the `Node` type is the single shared
DOM, so the package must stay dependency-free to keep it usable by every layer.

### 5.2 Who imports `internal/html` (consumers)

| Package | Why it needs the tree |
|---------|----------------------|
| `internal/css` | Selector matching against nodes (`css.go`, `has.go`). |
| `internal/layout` | Style resolution, box building, inline collection, tables, floats, multicol, images, paint (`style.go`, `style_cascade.go`, `layout_flow.go`, `inline_collect.go`, `layout_tables.go`, `flex.go`, `grid.go`, `float.go`, `multicol.go`, `layout_images.go`, `paint_flow.go`, `container.go`, `pseudo_content.go`, `layout_measure.go`). |
| `internal/convert` (+ `prepare/`, `islands/`) | Main parse entry (`prepare/prepare.go:154`), stylesheet collection (`prepare/styles.go:32-99`), HF/TOC/outline/links/page-islands (`hf.go`, `toc.go`, `outline.go`, `links.go`, `page_islands.go`, `islands/plan.go`). |
| `internal/outline` | Heading collection for PDF bookmarks (`outline.go:121`). |
| `internal/imageout` | Image-mode rendering shares the same parse→layout path (`imageout.go:121,128,170`). |

### 5.3 Import-direction rule

`internal/html` sits at the **bottom of the dependency graph**:

```text
cmd/*  →  internal/cli, internal/app, root api.go
                      ↓
            internal/convert  ──→  internal/layout, internal/css,
            internal/imageout      internal/pdf, internal/outline,
                                   internal/load, internal/settings
                           ↓  ↓  ↓
                  internal/css ─┐
                  internal/layout ─┼─→  internal/html   (leaf, stdlib-only)
                  internal/outline ─┘
```

Nothing imports *into* `internal/html` from the repo except through its public
API (`Parse`, `ParseDocument`, `Node` and friends). The `ponytail` comment at
`html.go:6` records the constraint explicitly:

> *"ponytail: custom Node tree (Parent/Attrs/void); migrate to x/net/html only
> if layout/css rewritten, not free delete."*

i.e. replacing this tree with `x/net/html` would be a cross-cutting rewrite of
`layout` and `css`, not a local swap.

## 6. Design decisions & trade-offs

### 6.1 Hand-rolled tokenizer instead of `x/net/html` or a browser engine

- **Why:** the project rule is *pure Go, no cgo, no third-party HTML/PDF/CSS
  APIs* (README, `documentation/fidelity.md`). A hand-rolled allowlisted
  tokenizer keeps the dependency surface at zero and matches the
  "controlled reports" product scope: invoices, statements, tables, TOC —
  not arbitrary websites.
- **Cost:** no full HTML5 error recovery, no foster parenting, no implied
  `tbody`/`colgroup` insertion, no template element, no character-reference
  edge cases beyond the stdlib decoder. Malformed-but-common markup works;
  pathological markup yields a usable-but-imperfect tree.
- **Migration note:** the ponytail audit flags that swapping in `x/net/html`
  would force a rewrite of `layout` and `css` — the tree shape is load-bearing
  everywhere.

### 6.2 Streaming single pass (sink callback)

`scanTokens` emits tokens through `tokenSink` as they are recognized, and tree
building happens in the same loop. This avoids materializing a token slice for
production documents (the `tokenize` collector is test-only) and keeps memory
bounded by tree size, not token count.

### 6.3 Allowlisted-by-construction, not allowlisted-by-validation

The tokenizer does **not** validate tag names against a whitelist. Instead the
*scope* of accepted constructs is narrow by design (tags/attrs/text/comments/
doctype/void/raw-text), and **rendering** is gated downstream:

- `<script>`/`<style>`/`<title>`/`<head>`/`<meta>`/`<link>` get UA style
  `display: none` (`internal/layout/style_values.go:980-1005` via the
  `uaDecls` map), which is what "strips" them at the layout stage.
- No JavaScript engine exists, so event-handler attributes (`onclick`,
  `onload`, …) are inert data — parsed as ordinary attributes, never executed.

The parser is therefore *permissive in structure, conservative in rendering* —
the opposite of a browser's security posture, and appropriate for server-side
report generation.

### 6.4 wkhtmltopdf work-alike behavior

`ParseDocument`'s BOM handling mirrors `load.IsHTML`; the `#document`
synthetic root and `html/head/body` merging match browser expectations so CSS
like `html { … }` / `body { … }` resolves naturally. The auto-close vocabulary
(`html.go:348-364`) covers the table/list/paragraph constructs report HTML
actually uses (`li`, `p`, `tr`, `td`, `th`, `option`, `dd/dt`, thead/tbody/
tfoot, head/body/html).

### 6.5 Performance micro-decisions

- `Node.Attribute` fast path avoids `ToLower` copies for already-lowercase
  lookups (`html.go:48-57`).
- `endTagName` returns a **zero-copy substring** for the common `</name>`
  form, only copying when trimming/folding actually changes the bytes
  (`html.go:529-574`).
- `rawTextEnd` is byte-wise over the remaining source — no lowered copy of the
  document, no needle allocation (`html.go:679-708`).
- Token attrs use an interleaved `[]string` (2 slots per pair) instead of a
  map to reduce allocation pressure in the hot scanner; the map is built only
  on `openElement` when attributes exist.

### 6.6 Entities via stdlib

`UnescapeEntities` (`entities.go:10-17`) is a thin gate over
`html.UnescapeString`: skip entirely when the input contains no `&`. This gives
named references (`&amp;`, `&nbsp;`, …) and numeric references (`&#NN;`,
`&#xHH;`) for free, without maintaining a private entity table. Decoding
happens at **text append** (`html.go:162-184`) and **attribute application**
(`html.go:265-276`), i.e. once, at parse time — downstream code never re-decodes.

## 7. Notable patterns & invariants

1. **Single shared DOM.** `*html.Node` is the one tree type in the repo;
   `ElementLocation.Node`, `outline.Location.Node`, style maps
   (`map[*html.Node]*ResolvedStyle`) all key off node identity, and the
   `#document` synthetic root is a stable invariant (`css.isRootElement`
   special-cases names starting with `#`, `css.go:1461-1478`).

2. **Streaming scanner + callback tree build.** No intermediate token list in
   production paths; `tokenize` exists purely for tests.

3. **Adjacent text merging.** Consecutive text tokens merge into one `TextNode`
   (`html.go:171-182`), keeping the tree small and making `TextContent`
   deterministic.

4. **First-wins duplicate attributes.** `applyAttributes` keeps the first
   value of a duplicated attribute (`html.go:268-273`) — a deterministic rule
   where browsers vary.

5. **Lowercasing at the boundary.** Element names and attribute names are
   lowercased once by the tokenizer; downstream lookups assume lowercase.
   `Node.Attribute` still handles uppercase callers gracefully.

6. **Raw-text elements are content, not markup.** `script`/`style`/
   `textarea`/`title` contents are captured verbatim as a text node, so `<`
   inside them never opens elements (see `TestParseRawTextTree`,
   `html_test.go:703`, and the `<script>if (a < b) …` case). Stripping them
   from *rendering* is delegated to UA `display:none` styles in layout.

7. **Sentinel errors, graceful text fallbacks.** Unterminated constructs
   return sentinel errors; unterminated *start tags* and raw-text runs degrade
   to text instead of erroring. `TestParseUsableTreeNoPanic`
   (`html_test.go:825-844`) locks in the "never panic on hostile input"
   invariant.

8. **Extension points.** The `autoClose` map, `isVoidElement`, and
   `isRawTextElement` are the three vocabularies a new HTML construct would
   touch; everything else is generic (any element name is accepted).

## 8. Security considerations

The parser is the first trust boundary for *markup*, and the design leans on
**structural inertness + downstream rendering gating**:

- **No script execution by construction.** `<script>` becomes a text node
  (`html.go:315-323` raw-text handling), and the layout UA sheet sets
  `display: none` (`style_values.go:986-988`). Event-handler attributes are
  inert data. `--enable-javascript` is accepted for CLI compatibility but
  ignored with a warning (`documentation/compatibility-matrix.md:245,339`;
  `documentation/fidelity.md:22,24,176`).
- **No forms / no automatic submission.** There is no form submission path;
  the SSRF posture is documented in `documentation/THREAT-MODEL.md` and the
  compatibility matrix ("POST only via explicit `--post` flags; no cookies
  auto-forwarded", `compatibility-matrix.md:264`). Forms and inputs degrade to
  ordinary boxes at best.
- **No iframe/frameset/object embedding.** Not in the supported vocabulary;
  unsupported tags become inert element nodes that render as empty/inline
  boxes per their UA styles, or are hidden.
- **Deterministic, non-crashing parsing.** Malformed or hostile input cannot
  panic (`TestParseUsableTreeNoPanic`); unterminated constructs surface as
  sentinel errors that `prepare.Document` wraps and the converter reports
  cleanly (`internal/convert/prepare/prepare.go:155-158`).
- **Charset is enforced before parse.** Only UTF-8/ASCII reaches the parser
  (`load.checkDocumentCharset`, `load.go:492-520`), closing the classic
  charset-confusion/encoding-mismatch vector.
- **Attribute values are decoded, not executed.** Entity decoding happens at
  parse time; there is no mechanism to turn attribute content into behavior
  (`html.go:265-276`).

The model: the HTML parser produces a *safe, inert data structure*; all
"dangerous" HTML features are structurally impossible to execute, not merely
filtered at the last moment.

## 9. Testing & verification

All tests are **same-package** (`//nolint:testpackage` at
`html_test.go:1`) so tokenizer internals (`tokenize`, `tokenKind`) are tested
directly.

| Test helper | Line | Purpose |
|-------------|------|---------|
| `mustParse(t, src)` | 10 | Parse or fatal. |
| `treeString(node)` | 22 | Renders the tree for failure messages. |
| `assertChildren(t, node, names...)` | 51 | Asserts element children in order. |

Tokenizer-level tests (`html_test.go:77-380`):

- `TestTokenizeAttributes` (77), `TestTokenizeWhitespaceAroundEquals` (106),
  `TestTokenizeGreaterThanInQuotedValue` (132) — attribute tokenization,
  whitespace tolerance, `>` inside quoted values.
- `TestTokenizeComments` (158), `TestTokenizeDoctype` (187),
  `TestTokenizeDeclarationsAndPI` (219) — comment/doctype/PI/bogus-declaration
  handling (bogus decls skipped, PI skipped).
- `TestTokenizeRawText` (238), `TestTokenizeRawTextClosesOnlyRealEndTag` (273)
  — raw-text capture and that only a *real* closing tag terminates it.
- `TestTokenizeBareLessThanIsText` (294), `TestTokenizeUnterminated` (328) —
  degrade-to-text behavior and sentinel errors.
- `TestParseMatchesCollectedTokenBuilder` (352) — **parse/tokenize agreement**
  (the two paths must produce identical structures).
- `TestUnescapeEntitiesInText` (383) — entity decoding.

Tree-builder tests (`html_test.go:403-901`):

- Nesting (403), parent pointers (422), void elements (439).
- Auto-close: table (468), `p` (493), lists (504), table sections (519).
- `html`/`head`/`body` merging (534), head→body transition (584).
- Text merging (596), duplicate attributes (632), self-closing (658),
  comments & doctype (677), raw-text tree shape (703).
- Malformed input table (724) — unclosed tags, stray end tags, `<>`, sloppy
  attributes, implicit `li` closing; each asserts a *specific usable tree*.
- No-panic invariant (825), pre-order walk (846), `TextContentOf` (870),
  `ParseDocument` BOM handling (887).

**Cross-package validation** (the parser is also exercised indirectly by
hundreds of layout/css/convert tests that parse HTML fixtures — e.g.
`internal/layout/*_test.go`, `internal/css/*_test.go`,
`internal/convert/golden_test.go` — and by the committed golden PDF/PNG
samples regenerated via `make samples`; see
`documentation/samples.md` and `documentation/fidelity.md`).

## 10. Known limitations, deferred items & open questions

- **Not an HTML5 parser.** No full error recovery, no implied
  `tbody`/`colgroup`/`head`/`body` insertion beyond the merge logic, no
  foster-parenting, no template handling, no `<iframe>` semantics. This is
  *by design* (report subset), not a regression.
- **CDATA claim vs behavior.** The package doc (`html.go:2`) lists CDATA as
  accepted, but `scanBang` treats `<![CDATA[…]]>` as a *bogus declaration
  skipped to `>`* (`html.go:503-509`) — content is dropped, not kept. The doc
  comment slightly overstates the implementation; worth a doc fix or a real
  CDATA path if any fixture needs it.
- **Charset strictness.** UTF-8/ASCII only; other charsets error at the load
  seam (`load.go:492-520`, `errUnsupportedCharset`). Documented in
  `documentation/deferred.md` / compatibility matrix.
- **JavaScript-dependent pages** are out of scope — `<script>` stripped, JS
  flags warn only (`documentation/fidelity.md:22,24,176`;
  `compatibility-matrix.md:245`; `documentation/deferred.md:27,36` discusses
  SPA/JS-constructed DOM as a non-goal for MVP).
- **No DOM APIs.** No `getElementById`-style lookup, no tree mutation API
  beyond parse-time construction; `Walk`/`FirstChild`/`TextContentOf` are the
  only traversal surface (sufficient for all current consumers).
- **Entity scope** is whatever stdlib `html.UnescapeString` covers — unknown
  named references pass through as literal text rather than being flagged.
- **Migration question (tracked):** the ponytail note at `html.go:6` — adopt
  `x/net/html` only if `layout`/`css` are rewritten; not a free delete. Open
  question: whether a future "decent print for arbitrary URLs" phase
  (post-MVP Phase 21, `documentation/fidelity.md`) will force that rewrite.
- **Unknown tags** are accepted structurally and render via UA defaults
  (usually empty/inline); the compatibility matrix is the normative contract
  for what is *styled* (`documentation/compatibility-matrix.md`).

## 11. Related documents

- Pipeline overview: [`../architecture.md`](../architecture.md)
- Fidelity tiers & degrade rules: [`../fidelity.md`](../fidelity.md)
- Security model & local-file ACL: [`../THREAT-MODEL.md`](../THREAT-MODEL.md)
- Support matrix (per-element / per-property): [`../compatibility-matrix.md`](../compatibility-matrix.md)
- Deferred / not-planned items: [`../deferred.md`](../deferred.md)
- Fonts & shaping (feeds text layout that consumes this tree): [`../fonts.md`](../fonts.md)
- Library API (how callers supply HTML bytes): [`../library-api.md`](../library-api.md)
- Canonical plan (why the parser was built this way): [`../../plans/0.1.0/00-canonical-pure-go-rewrite.md`](../../plans/0.1.0/00-canonical-pure-go-rewrite.md)

Sibling architecture deep-dives (same directory):

- [01-entrypoints-cli.md](01-entrypoints-cli.md) — `cmd/*` + `internal/cli`
- [02-library-api.md](02-library-api.md) — root `api.go` public API
- [03-settings.md](03-settings.md) — `internal/settings` dotted config
- [04-load.md](04-load.md) — `internal/load` (the seam that feeds this parser)
- [06-css.md](06-css.md) — selector matching *against this tree*
- [07-layout.md](07-layout.md) — style resolution + box building over this tree
- [08-convert-pipeline.md](08-convert-pipeline.md) — orchestration (HF/TOC/outline reuse this parser)
- [09-pdf-writer.md](09-pdf-writer.md) — PDF sink for the display list
- [10-imageout-svg.md](10-imageout-svg.md) — image-mode sink (shares parse→layout)
