# Phase 2 — Document prep & pipeline

> **Parent:** [`architecture-review-2026-08-07.md`](../architecture-review-2026-08-07.md) — canonical architecture-review ledger
> **Status:** pending (findings gathered 2026-08-07 by 7 explore agents; remediation not started)
> **Depends on:** see phase map in ledger
> **Evidence archive:** raw agent findings were archived off-repo on 2026-08-07; every row below carries its Before/After snippets inline

---

## Overview

…

## Checklist

- [x] **P2-01** — One stylesheet-gatherer module instead of two near-copies — convert.CollectSheets exported; imageout consumes + deleted twin
- [x] **P2-02** — Settle Heading.Page's contract: flatten once, never mutate, never un-flatten — DocPage set once in flatHeadings; consumers use DocPage / DocPage view
- [x] **P2-03** — Consolidate the page-index/copies model into one type — pagePlan (OwnerOf/Remap/Ranges) in convert
- [~] **P2-04** — Make in-memory HTML an explicit source kind with a base URL — api/settings/load done; full wiring (wave 2) pending
- [x] **P2-05** — Move SortHeadings + section lookup into the outline package — outline exports + convert SectionOf/BuildTree via DocPage view
- [x] **P2-06** — Delete `locationOf`; one y-down→PDF helper next to geometry fields — hfGeom.pdfY/pdfXY/pdfRect
- [x] **P2-07** — Loader policy lives in one construction; stop caller-poked ACL fields — convert + imageout pokes removed; NewLoader applies LoadGlobal
- [x] **P2-08** — Give html.Node the traversal primitives callers keep re-implementing — Walk/TextContentOf; convert + imageout call sites done
- [x] **P2-09** — Own the bytes→runes seam that Parse assumes but nobody implements — ParseDocument at convert/imageout load sites; `--default-encoding` still FIX-REVIEW
- [ ] **P2-10** — Slim `objectState` god-struct; make the pass-order handshakes explicit — deferred (risk; logged in fix-convert)
- [x] **P2-11** — One page-geometry/`layout.Options` constructor instead of arithmetic at 6 sites — newHFGeom + bodyLayoutOpts
- [~] **P2-12** — Drop `applyInternalLinks`' ignored `cmd`; stop `paintCount` swallowing layout errors — cmd dropped; paintCount still swallows (FIX-REVIEW)
- [x] **P2-13** — Remove the magic OpKind-sentinel link-neutralization leak into convert — layout.DeactivateOp + convert consumer
- [x] **P2-14** — Unify the document-prep prologue (sheet harvesting + font-face merge) with imageout — CollectSheets + MergeFontFaces shared

---

<a id="p2-01"></a>
## P2-01 — One stylesheet-gatherer module instead of two near-copies

- [ ] **P2-01** — pending (fix-convert side never landed; imageout consume = wave 2)

- **Locations:** `internal/convert/convert.go:528-615` and `internal/imageout/imageout.go:628-697` (collectSheets + styleText + linkStylesheet), `internal/convert/convert.go:529-611` (collectSheets, styleText, linkStylesheet) and the verbatim siblings in `internal/imageout/imageout.go:631-705`
- **Evidence sources:** area-3 F5; area-4 F1

---

### Evidence A — F5

- **Severity:** medium
- **Location:** `internal/convert/convert.go:528-615` and `internal/imageout/imageout.go:628-697` (collectSheets + styleText + linkStylesheet)
- **Current (verbatim, imageout copy):**
```go
func collectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, log io.Writer) []*css.Stylesheet {
	var sheets []*css.Stylesheet
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		switch n.Name {
		case "style":
			sheet, err := css.Parse(styleText(n))
			if err != nil {
				fmt.Fprintf(log, "warning: skipping <style>: %v\n", err)
			} else if sheet != nil {
				sheets = append(sheets, sheet)
			}
			return // raw-text element; no element children
		case "link":
			if linkStylesheet(n) {
				href := n.Attribute("href")
				r, err := loader.FetchSub(ctx, base, href, lp)
				if err != nil {
					fmt.Fprintf(log, "warning: skipping <link href=%q>: %v\n", href, err)
					return
				}
				sheet, err := css.Parse(string(r.Body))
				if err != nil {
					fmt.Fprintf(log, "warning: skipping <link href=%q>: %v\n", href, err)
					return
				}
				sheets = append(sheets, sheet)
			}
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return sheets
}

The `internal/convert` copy (convert.go:531-615) is the same walk with three
extra parameters (`viewportW/H, mediaType`, a >25k-rules soft warning and
`index` in the log lines); `styleText` and `linkStylesheet` are duplicated
identically in both files, with the media gate already drifted (imageout
hardcodes `768, 576, "screen"` and lost the soft-warning). `internal/convert`
is already the shared module for image mode (`MergeFontFaces`,
`SimplifyDOMProfile` are exported and called by `internal/imageout`).
- **Future:**
```go
// internal/convert — one implementation, exported like MergeFontFaces
func CollectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, idx int, log io.Writer, viewportW, viewportH float64, mediaType string) []*css.Stylesheet
```
`convert` call sites pass their args as today; `imageout.go:454` becomes
`sheets := convert.CollectSheets(ctx, loader, root, res.Base, obj.Load, 1, log, 768, 576, "screen")`
and the three duplicated funcs in imageout are deleted. (If the team prefers,
host it in `internal/html`/a `load` document helper instead — the point is one
owner, not the package name.) Caller that must move: `internal/imageout` (one
call site + three deletions). API is otherwise unchanged.
- **Why:** this is the hunt-priority-#3 locality violation with proof of decay:
two callers, one stale copy. The DOM-walk+fetch+media-filter knowledge is
stylesheets-on-document knowledge sitting in two output pipelines; every fix
(single rule cap, `media` handling, http caching) costs two edits and a bug
like the missing `softRuleWarn` is exactly the divergence this creates. Depth:
the gatherer hides ~25 lines of fetch/walk/parse behind one call.

### Evidence B — F1

- **Severity:** high
- **Location:** `internal/convert/convert.go:529-611` (collectSheets, styleText, linkStylesheet) and the verbatim siblings in `internal/imageout/imageout.go:631-705`
- **Current (verbatim):**
```go
func collectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, log io.Writer) []*css.Stylesheet {
	var sheets []*css.Stylesheet
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		switch n.Name {
		case "style":
			sheet, err := css.Parse(styleText(n))
			if err != nil {
				fmt.Fprintf(log, "warning: skipping <style>: %v\n", err)
			} else if sheet != nil {
				sheets = append(sheets, sheet)
			}
			return // raw-text element; no element children
		case "link":
			if linkStylesheet(n) {
				href := n.Attribute("href")
				r, err := loader.FetchSub(ctx, base, href, lp)
				if err != nil {
					fmt.Fprintf(log, "warning: skipping <link href=%q>: %v\n", href, err)
					return
				}
				sheet, err := css.Parse(string(r.Body))
				if err != nil {
					fmt.Fprintf(log, "warning: skipping <link href=%q>: %v\n", href, err)
					return
				}
				sheets = append(sheets, sheet)
			}
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return sheets
}
```
`convert/convert.go:529-611` carries the same walk and the same `styleText`, plus `linkStylesheet(n, vw, vh, mediaType)` evaluated against the real viewport and a >25k-rule warning. `imageout/imageout.go:631-699` is a copy whose `linkStylesheet` hardcodes `const vw, vh = 768.0, 576.0` and media `"screen"`. The two copies differ in exactly the places a bug lands: warning prefixes, the rule-volume warn (convert only), and the link-media policy.
- **Future:** one deep module at the sheet-collection seam; both pipelines become thin fetch adapters:
```go
// internal/css/collect.go (package css; css already imports internal/html)
//
// SheetSource fetches one <link rel=stylesheet> body. The loader is an
// adapter: pdf vs image pass their own fetch policy unchanged.
type SheetSource func(ctx context.Context, ref, base string) ([]byte, error)

// CollectSheets gathers <style> blocks and <link rel="stylesheet"> resources
// from the DOM in document order. Broken stylesheets are skipped and reported
// via logf; vw/vh (pt) and mediaType gate link media attributes.
func CollectSheets(ctx context.Context, root *html.Node, load SheetSource, mediaType string, vw, vh float64, logf func(format string, args ...any)) []*Stylesheet {
	var sheets []*Stylesheet
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		switch n.Name {
		case "style":
			sheet, err := Parse(collectText(n))
			if err != nil {
				logf("skipping <style>: %v", err)
			} else if sheet != nil {
				sheets = append(sheets, sheet)
			}
			return
		case "link":
			if !linkOK(n, mediaType, vw, vh) {
				return
			}
			body, err := load(ctx, n.Attribute("href"), "")
			if err != nil {
				logf("skipping <link href=%q>: %v", n.Attribute("href"), err)
				return
			}
			if sheet, err := Parse(string(body)); err != nil {
				logf("skipping <link href=%q>: %v", n.Attribute("href"), err)
			} else if sheet != nil {
				sheets = append(sheets, sheet)
			}
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return sheets
}
```
`convert` passes a `SheetSource` closing over `loader.FetchSub(ctx, base, href, lp)` and keeps its `object %d` prefix and rule-volume warn as the small post-check it is; `imageout` passes its loader with `"screen", 768, 576`. Both `styleText` and both `linkStylesheet` variants disappear.
- **Why:** the DOM→stylesheet knowledge (`style` raw-text, `rel=stylesheet` + media gate, fetch-and-parse, warn-don't-fail) is invariant across both pipelines, yet it is two copies with two intertwined policy knobs — a locality violation where any fix (media attribute handling, `type="text/css"` filtering, fetch error policy) must be made twice, and the copies have already drifted (fixed 768×576 viewport vs the real one; different warning text). The only thing that genuinely varies across callers is fetching — one adapter, not yet a real seam. Moving the walk into the css package makes the seam explicit and shrinks each caller by ~60 lines. No behavior change by construction. hypothesis: the two copies continue to drift as image mode grows — validate by re-grepping `FetchSub` + `MediaMatches` in both packages after the next image-mode change.

> **Fan-in note:** Merged from area-3 F5 + area-4 F1. Related: P2-14 (document-prep prologue duplication) and P5-02 (page-assembly fork) — schedule those rows after this one.

---

<a id="p2-02"></a>
## P2-02 — Settle Heading.Page's contract: flatten once, never mutate, never un-flatten

- [~] **P2-02** — outline side done (Heading.DocPage, fix-html-load-outline); convert consumers pending (fix-convert)

- **Locations:** `internal/outline/outline.go:23-32` (Heading), `internal/convert/outline.go:84-96` (rebase), `internal/convert/outline.go:150-162` (`emitOutline` un-rebase), consumers `internal/convert/toc.go:172-176`, `internal/convert/links.go:102-113`, `internal/convert/hf.go:525-544`)
- **Evidence sources:** area-3 F1

---

### Evidence — F1

- **Severity:** high
- **Location:** `internal/outline/outline.go:23-32` (Heading), `internal/convert/outline.go:84-96` (rebase), `internal/convert/outline.go:150-162` (`emitOutline` un-rebase), consumers `internal/convert/toc.go:172-176`, `internal/convert/links.go:102-113`, `internal/convert/hf.go:525-544`)
- **Current (verbatim):**
```go
func collectObjectHeadings(root *html.Node, res *layout.Result, offset int, g settings.PdfGlobal, obj settings.PdfObject, log io.Writer) []*outline.Heading {
	if !obj.UseOutline || !obj.IncludeInOutline {
		return nil
	}
	hs := outline.CollectHeadings(root)
	hs = outline.Lookup(hs, res.Locations)
	if len(hs) == 0 {
		return nil
	}
	for _, h := range hs {
		h.Page += offset
	}
	return hs
}
```

`Heading.Page` is documented as "object-local; callers rebase it to document
pages", `emitOutline` then *un-rebases* (`locPage := h.Page - st.offset`) to
recover that same object-local number for the y flip, and the TOC/link/header
passes add `tocGuess`/`tocTotal` on top. One field, three contradictory meanings
across the module boundary, one in-place mutation of a shared value.
- **Future:**
```go
// outline.Heading — Page stays object-local forever; DocPage is set exactly
// once during assembly and never mutated afterwards.
type Heading struct {
	Node        *html.Node
	Title       string
	Level       int            // 1..6
	Page        int            // 0-based page within the calling object's layout
	DocPage     int            // document-global page, filled by AssembleHeadings
	X, Y, W, H  float64        // canvas coords (from layout.ElementLocation)
	Anchor      string
}

// convert
func flatHeadings(bodies []*objectState) []*outline.Heading {
	var all []*outline.Heading
	for _, st := range bodies {
		for _, h := range st.headings {
			h.DocPage = st.offset + h.Page // the single explicit rebase
			all = append(all, h)
		}
	}
	outline.AssignAnchors(all)
	outline.SortHeadings(all)
	return all
}
```
`collectObjectHeadings` keeps only `CollectHeadings`+`Lookup` (no mutation);
`emitOutline` uses `h.DocPage` for `PageRef`/`bodyStateFor` and derives the
object-local page as `h.DocPage - st.offset`; `pageOf` and `applyTOCLinks`
switch to `h.DocPage`. Additive field on `Heading`; callers move only their
internal reads — no exported behavior changes.
- **Why:** the invariant "what spell of `Page` am I holding?" is spread across
convert/outline, toc, links and hf. `emitOutline` has to *reverse* the earlier
mutation, which is exactly the double-bookkeeping a future reader will get
wrong. hypothesis: the next consumer of `Heading` will read the wrong page
spell; validate by asserting in a render golden test that `h.DocPage -
st.offset == element location page` for every outline entry after the change.

---

<a id="p2-03"></a>
## P2-03 — Consolidate the page-index/copies model into one type

- [ ] **P2-03** — pending (fix-convert)

- **Locations:** `internal/convert/hf.go:556-596`; siblings `internal/convert/convert.go:204-264` (`pageRange`, `tocFirstOrder`, `materializeCopies`, `nonCollateOrder`) and `:146-152` (ranges rebuild); `internal/convert/links.go:175-185` (`remapPageForCopies`); `internal/convert/outline.go:56-57` (`objectState.start`)
- **Evidence sources:** area-6 F1

---

### Evidence — F1

- **Severity:** high
- **Location:** `internal/convert/hf.go:556-596`; siblings `internal/convert/convert.go:204-264` (`pageRange`, `tocFirstOrder`, `materializeCopies`, `nonCollateOrder`) and `:146-152` (ranges rebuild); `internal/convert/links.go:175-185` (`remapPageForCopies`); `internal/convert/outline.go:56-57` (`objectState.start`)
- **Current (verbatim):**
```go
	type owner struct {
		st    *objectState
		local int // page index within the object
	}
	owners := make([]owner, 0, total)
	for _, st := range tocs {
		for i := 0; i < st.tocPages; i++ {
			owners = append(owners, owner{st, i})
		}
	}
	for _, st := range bodies {
		for i := 0; i < st.pages; i++ {
			owners = append(owners, owner{st, i})
		}
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	clock := now.Format("15:04:05")
	logicalN := len(owners)
	copies := cmd.Global.Copies
	if copies < 1 {
		copies = 1
	}
	idIndex := buildBodyIDIndex(bodies)
	for p := 0; p < total; p++ {
		var own owner
		switch {
		case logicalN == 0:
			continue
		case copies <= 1 || total == logicalN:
			if p >= logicalN {
				continue
			}
			own = owners[p]
		case cmd.Global.Collate:
			own = owners[p%logicalN]
		default:
			// non-collate: copies of page i are contiguous
			own = owners[p/copies]
		}
	}
```
- **Future:** one `pagePlan` type owns the whole index model: logical owner list (TOC pages then body pages), the TOC-first permutation, the copy materialization order, and both directions of the copy remap. `drawHeadersFooters`, the link passes, and `RunPDFContext` consume it instead of re-deriving it:
```go
// pagePlan is the single owner of the document's page-index model: the
// logical (pre-copy) order, the TOC front-matter offset, and the
// copy/collate permutation onto final document pages.
type pagePlan struct {
	owners   []pageOwner // logical page -> owning object + local index
	tocTotal int
	copies   int
	collate  bool
}

type pageOwner struct {
	st    *objectState
	local int // page index within the object
}

func newPagePlan(tocs, bodies []*objectState, copies int, collate bool) *pagePlan {
	pp := &pagePlan{copies: copies, collate: collate}
	for _, st := range tocs {
		pp.tocTotal += st.tocPages
		for i := 0; i < st.tocPages; i++ {
			pp.owners = append(pp.owners, pageOwner{st, i})
		}
	}
	for _, st := range bodies {
		for i := 0; i < st.pages; i++ {
			pp.owners = append(pp.owners, pageOwner{st, i})
		}
	}
	return pp
}

// OwnerOf resolves the object that owns final page p (header/footer and
// link passes). ok is false for pages outside the logical set.
func (pp *pagePlan) OwnerOf(p int) (pageOwner, bool) {
	n := len(pp.owners)
	if n == 0 {
		return pageOwner{}, false
	}
	var i int
	switch {
	case pp.copies <= 1:
		i = p
	case pp.collate:
		i = p % n
	default: // non-collate: copies of page i are contiguous
		i = p / pp.copies
	}
	if i >= n {
		return pageOwner{}, false
	}
	return pp.owners[i], true
}

// Remap converts a logical (pre-copy) dest page to the final page in the
// same copy group as srcPage (replaces links.go remapPageForCopies).
func (pp *pagePlan) Remap(logicalDest, srcPage int) int {
	n := len(pp.owners)
	if pp.copies <= 1 || n <= 0 {
		return logicalDest
	}
	if pp.collate {
		return (srcPage/n)*n + logicalDest
	}
	return logicalDest*pp.copies + srcPage%pp.copies
}
```
Callers are all internal: `RunPDFContext` (convert.go:146-152, 176-264 — `pageRange`/`tocFirstOrder`/`materializeCopies`/`nonCollateOrder` shrink to `pp.Ranges()`/`pp.FirstOrder()` or disappear), `drawHeadersFooters` (hf.go:554-596 → `pp.OwnerOf(p)`), the link passes and `hfLinkContext` (links.go:175-185, hf.go:277-297, 618-628 → hold `*pagePlan` instead of `logicalN/copies/collate/srcPage`). Existing unit tests for `remapPageForCopies`/`materializeCopies` (convert_test.go) move to `pagePlan` tests unchanged.
- **Why:** the same copy/collate model is computed three times with different arithmetic — the `owners` switch above, `remapPageForCopies` (links.go:175-185), and `materializeCopies`+`nonCollateOrder` (convert.go:230-264) — plus the span bookkeeping (`pageRange`, `tocFirstOrder`) rebuilt from the same `tocs`/`bodies` data at convert.go:146-152. A change to the copies model (e.g. an interleave option, or duplex-aware ordering) touches four files, and the three arithmetic variants can disagree with each other; there is no single place that states what the final page order *is*. `pagePlan` is a deep module: ~50 lines of subtle index math behind a 3-method interface, paying off across the TOC, link, HF and copies passes. hypothesis: the derivations already disagree for copies>1 + TOC; validate by unit tests asserting `OwnerOf(p)` equals today's owners-switch output for collate/non-collate with copies 2-3 and a TOC present.

---

---

<a id="p2-04"></a>
## P2-04 — Make in-memory HTML an explicit source kind with a base URL

- [~] **P2-04** — api (SetBody/AddHTML/ConvertHTML) + settings (InlineHTML/InlineBase) + load (InlineHTML-first) done; full wiring (wave 2) pending

- **Locations:** `api.go:196-208` (`ConvertHTML`); classification rule `internal/load/load.go:88-105` (`IsHTML`/`GuessURL`)
- **Evidence sources:** area-1 F4

---

### Evidence — F4

- **Severity:** medium
- **Location:** `api.go:196-208` (`ConvertHTML`); classification rule `internal/load/load.go:88-105` (`IsHTML`/`GuessURL`)
- **Current:**
```go
func ConvertHTML(ctx context.Context, html []byte, global *GlobalSettings) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(html) == 0 {
		return nil, errors.New("gowkhtmltopdf: empty HTML")
	}
	c := NewConverter()
	if global != nil {
		c.global = global
	}
	obj := NewObjectSettings().SetPage(string(html))
	c.AddObject(obj)
```
- **Future:** an explicit document source on the object, with an optional base:
```go
// ObjectSettings.SetBody sets an in-memory HTML document as the input
// page. base resolves relative subresources (<link>, <img>, <a>); empty
// leaves them unresolvable. No URL guessing is involved.
func (s *ObjectSettings) SetBody(html []byte, base string) *ObjectSettings {
	s.o.Page = ""
	s.o.InlineHTML = html
	s.o.InlineBase = base
	return s
}

func ConvertHTML(ctx context.Context, html []byte, global *GlobalSettings) ([]byte, error) {
	if len(html) == 0 {
		return nil, errors.New("gowkhtmltopdf: empty HTML")
	}
	c := NewConverter()
	if global != nil {
		c.global = global
	}
	c.AddObject(NewObjectSettings().SetBody(html, ""))
	if err := c.Convert(ctx); err != nil {
		return nil, err
	}
	return c.Output(), nil
}
// internal/load.Load checks lp.InlineHTML first and skips GuessURL
// entirely; subresources resolve against InlineBase when set.
```
Exported behaviour change: `ObjectSettings` gains `SetBody`; `ConvertHTML` and `SetPage` keep working (SetBody is the only new path). The two fields on `settings.LoadPage`/`PdfObject` have a real engine consumer (`load.Load`), so settings Policy A permits them.
- **Why:** `ConvertHTML` funnels document bytes through the page *string*, so whether they are treated as a document depends on `load.IsHTML`'s first-character heuristic (`TrimSpace` then must start with `<`, or a BOM+`<`). A caller passing HTML that starts with, say, an XML prolog after whitespace, or any leading text, silently gets an `http://` fetch of their own markup. The contract is invisible at the call site and duplicated across `doc.go`, `api.go` docs, and `load.go`. Worse, inline documents have no base: `load.Load` returns `Base:""` for `KindInline`, so relative `<link href>`/`<img src>` resolve against the *process working directory* (scheme-less paths go to `loadFile`), which is neither the document's location nor anything the caller controls — an embedder converting a template with `<link href="/css/site.css">` can end up reading `/css/site.css` from the host filesystem once local access is enabled. A `SetBody(html, base)` seam makes "inline document" a real variant with a base, removes the guessing rule from the API's contract, and lets `ConvertHTML` documents actually link assets. hypothesis: relative subresources in ConvertHTML docs resolve against CWD today (and absolute paths hit the host filesystem when local access is enabled); validate with an api_test that converts inline HTML with `<link href="style.css">` next to a temp CWD and asserts where the stylesheet came from.

---

<a id="p2-05"></a>
## P2-05 — Move SortHeadings + section lookup into the outline package

- [~] **P2-05** — outline side done (SortHeadings/SectionOf, fix-html-load-outline); convert consumption pending (fix-convert)

- **Locations:** duplicate comparators `internal/convert/outline.go:117-134` and `internal/outline/outline.go:137-154`; order-dependent section lookup `internal/convert/hf.go:525-544`
- **Evidence sources:** area-3 F2

---

### Evidence — F2

- **Severity:** medium
- **Location:** duplicate comparators `internal/convert/outline.go:117-134` and `internal/outline/outline.go:137-154`; order-dependent section lookup `internal/convert/hf.go:525-544`
- **Current (verbatim):**
```go
func flatHeadings(bodies []*objectState) []*outline.Heading {
	var all []*outline.Heading
	for _, st := range bodies {
		all = append(all, st.headings...)
	}
	outline.AssignAnchors(all)
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	return all
}
```

`BuildTree` sorts with an identical comparator immediately afterwards (so the
pre-sort only serves `sectionOf`, which lives two files away in hf.go and
re-implements the "current heading for a page" query outside the outline
package).
- **Future:**
```go
// internal/outline — canonical order, exported once
// SortHeadings brings headings into the order used by the tree, the TOC and
// the section lookup: page, y-down within a page, then x.
func SortHeadings(hs []*Heading) {
	sort.SliceStable(hs, func(i, j int) bool {
		a, b := hs[i], hs[j]
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
}

// SectionOf mirrors the wkhtmltopdf outline cache: section = first heading at
// or before page, subsection = last. Headings must be in SortHeadings order.
func SectionOf(hs []*Heading, page int) (section, subsection string) {
	var first, last *Heading
	for _, h := range hs {
		if h.Page > page {
			break
		}
		if first == nil {
			first = h
		}
		last = h
	}
	if first != nil {
		section = first.Title
	}
	if last != nil {
		subsection = last.Title
	}
	return section, subsection
}
```
`BuildTree` now calls `SortHeadings(sel)`; `flatHeadings` is concat +
`AssignAnchors` + `SortHeadings`; `sectionOf` is deleted and `drawHeadersFooters`
calls `outline.SectionOf(headings, p-tocTotal)`. No exported change for
outline; internal moves only.
- **Why:** ordering is outline-domain knowledge — today it is copy-pasted in
two packages and the section query (which reads outline data) sits in the
header/footer file. Any ordering or clamp change must be made twice and the
"must be sorted by (Page, Y, X)" contract traveled in a comment in hf.go
instead of the object that owns the list.

---

<a id="p2-06"></a>
## P2-06 — Delete `locationOf`; one y-down→PDF helper next to geometry fields

- [ ] **P2-06** — pending (fix-convert)

- **Locations:** `internal/convert/links.go:32-47` (pageRect + destPoint), `internal/convert/links.go:130-138` (`locationOf`), `internal/convert/outline.go:156-162` (`emitOutline` y-flip)
- **Evidence sources:** area-3 F3

---

### Evidence — F3

- **Severity:** medium
- **Location:** `internal/convert/links.go:32-47` (pageRect + destPoint), `internal/convert/links.go:130-138` (`locationOf`), `internal/convert/outline.go:156-162` (`emitOutline` y-flip)
- **Current (verbatim):**
```go
func pageRect(loc layout.ElementLocation, g hfGeom) [4]float64 {
	x1 := g.marginLeft + loc.X
	yTop := g.pageH - g.marginTop - (loc.Y - float64(loc.Page)*g.contentH)
	yBot := g.pageH - g.marginTop - (loc.Y + loc.H - float64(loc.Page)*g.contentH)
	return [4]float64{x1, yBot, x1 + loc.W, yTop}
}

// destPoint converts a location's top-left corner into a PDF /XYZ
// destination (x, y-up).
func destPoint(loc layout.ElementLocation, g hfGeom) (float64, float64) {
	x := g.marginLeft + loc.X
	y := g.pageH - g.marginTop - (loc.Y - float64(loc.Page)*g.contentH)
	return x, y
}
```

The same `pageH - marginTop - (Y - page*contentH)` expression appears a third
time in `emitOutline` (outline.go:161), and `headingDest` (links.go:122-128)
re-derives the location by linearly scanning `st.res.Locations` for the node —
even though `outline.Lookup` already copied that location into `h.X/Y/W/H`.
- **Future:**
```go
// on hfGeom, next to the margin/content numbers it uses:
func (g hfGeom) pdfY(loc layout.ElementLocation) float64 {
	return g.pageH - g.marginTop - (loc.Y - float64(loc.Page)*g.contentH)
}

func (g hfGeom) pdfXY(loc layout.ElementLocation) (float64, float64) {
	return g.marginLeft + loc.X, g.pdfY(loc)
}

func (g hfGeom) pdfRect(loc layout.ElementLocation) [4]float64 {
	x1 := g.marginLeft + loc.X
	return [4]float64{x1, g.pdfY(loc) - loc.H, x1 + loc.W, g.pdfY(loc)}
}
```
Callers collapse: `pageRect(loc,g)` → `g.pdfRect(loc)`; `destPoint(loc,g)` →
`g.pdfXY(loc)`; `emitOutline` uses `g.pdfY` with the object-local page
`h.DocPage - st.offset`; `headingDest` stops scanning and uses the heading's
own fields: `g.pdfXY(layout.ElementLocation{Page: h.DocPage - st.offset, X: h.X, Y: h.Y})`.
Delete `locationOf`. All private — no callers move.
- **Why:** the canvas→PDF transform is implemented three times and the
location data has two homes (the `ElementLocation` in `res.Locations` and the
copy in `Heading`); the pointer scan is O(locations) per heading for values
already in hand. One helper next to the geometry fields makes margin/contentH
changes (and the y-up flip) land in exactly one place, and the heading keeps
a single trusted location source.

---

<a id="p2-07"></a>
## P2-07 — Loader policy lives in one construction; stop caller-poked ACL fields

- [~] **P2-07** — settings (LoadGlobal.Allow/EnableLocalFileAccess) + load (NewLoader applies) done; convert.go:47-48 + imageout.go:422-423 pokes pending (currently breaks `go build ./...`)

- **Locations:** `internal/load/load.go:161-184` (struct + `NewLoader`), `internal/convert/convert.go:45-48`, `internal/imageout/imageout.go:420-423` (identical pokes), `internal/settings/settings.go` LoadGlobal (Proxy only)
- **Evidence sources:** area-3 F4

---

### Evidence — F4

- **Severity:** medium
- **Location:** `internal/load/load.go:161-184` (struct + `NewLoader`), `internal/convert/convert.go:45-48`, `internal/imageout/imageout.go:420-423` (identical pokes), `internal/settings/settings.go` LoadGlobal (Proxy only)
- **Current (verbatim):**
```go
type Loader struct {
	Client       *http.Client
	Global       settings.LoadGlobal
	Log          io.Writer
	MaxBodySize  int64
	MaxRedirects int

	// Allow prefixes and the effective local-access flag, injected by the
	// caller from settings (convert wires PdfGlobal.Allow + EnableLocalFileAccess).
	Allow                 []string
	EnableLocalFileAccess bool
}

// NewLoader builds a Loader from global load settings.
func NewLoader(g settings.LoadGlobal) *Loader {
	l := &Loader{
		Global:       g,
		Log:          io.Discard,
		MaxBodySize:  DefaultMaxBodySize,
		MaxRedirects: DefaultMaxRedirects,
	}
	l.initClient()
	return l
}

Every caller then splices the same three fields by hand:
```go
loader := load.NewLoader(cmd.Global.Load)          // convert.go:45
loader := load.NewLoader(cmd.Image.Load)           // imageout.go:420
loader.Allow = cmd.Settings.Allow
loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess
```
while `settings.LoadGlobal` — the natural config seam — carries only `Proxy`.
- **Future:**
```go
// settings — policy goes where the rest of the settings live
type LoadGlobal struct {
	Proxy                 string
	Allow                 []string // local ACL prefixes (--allow)
	EnableLocalFileAccess bool
}

// load — constructor applies everything; no after-the-fact pokes
func NewLoader(g settings.LoadGlobal) *Loader {
	l := &Loader{
		Global:                g,
		Log:                   io.Discard,
		MaxBodySize:           DefaultMaxBodySize,
		MaxRedirects:          DefaultMaxRedirects,
		Allow:                 g.Allow,
		EnableLocalFileAccess: g.EnableLocalFileAccess,
	}
	l.initClient()
	return l
}
```
Both call sites become `load.NewLoader(cmd.Global.Load)` (one place — the
settings layer — bridges the CLI flags). `Client *http.Client` is an exported
leak no caller replaces; keep it only if tests need a fake dial, otherwise
unexport it with a test accessor. Callers that must move: the two 3-line
pokes above disappear; everything else unchanged.
- **Why:** the ACL is a security policy whose *full contract* is currently
reconstructed by hand at each call site; the two copies have already drifted
in formatting and the image copy is the third duplicated spell of the same
cookbook incantation. Constructing the Loader with its complete policy makes
"did I forget `Allow`?" a type question, not a code-review question — the
missed seam of "one adapter" becoming real as new loaders (tests, future
fetchers) are added.

---

<a id="p2-08"></a>
## P2-08 — Give html.Node the traversal primitives callers keep re-implementing

- [~] **P2-08** — html.Walk/TextContentOf + outline refactor done (fix-html-load-outline); convert/imageout call sites pending

- **Locations:** `internal/html/html.go:25-64` (interface); re-implementations: `internal/outline/outline.go:57-77` (CollectHeadings walk), `internal/convert/outline.go:59-77` (docTitle walk), `internal/convert/convert.go:531-612` and `internal/imageout/imageout.go:631-673` (collectSheets walks ×2), `internal/convert/convert.go:581-589` and `internal/imageout/imageout.go:673-681` (styleText ×2), `internal/outline/outline_test.go:38-53` (headNodes)
- **Evidence sources:** area-3 F6

---

### Evidence — F6

- **Severity:** medium
- **Location:** `internal/html/html.go:25-64` (interface); re-implementations: `internal/outline/outline.go:57-77` (CollectHeadings walk), `internal/convert/outline.go:59-77` (docTitle walk), `internal/convert/convert.go:531-612` and `internal/imageout/imageout.go:631-673` (collectSheets walks ×2), `internal/convert/convert.go:581-589` and `internal/imageout/imageout.go:673-681` (styleText ×2), `internal/outline/outline_test.go:38-53` (headNodes)
- **Current (verbatim):**
```go
// Attribute returns an attribute value, or "".
func (n *Node) Attribute(name string) string { return n.Attrs[strings.ToLower(name)] }

// FirstChild returns the first element child with name, or nil.
func (n *Node) FirstChild(name string) *Node {
	for _, c := range n.Children {
		if c.Type == ElementNode && c.Name == name {
			return c
		}
	}
	return nil
}

// TextContent concatenates all descendant text.
func (n *Node) TextContent() string {
	var b strings.Builder
	n.appendText(&b)
	return b.String()
}
```

The package's whole DOM surface is `Attribute` + `FirstChild` + `TextContent`,
so every consumer that needs "walk descendants, filter by name/attribute/type"
writes its own recursive closure (seen 7+ times in production code above, one
more in tests).
- **Future:**
```go
// Walk visits n and every descendant in pre-order (document order).
func (n *Node) Walk(f func(*Node)) {
	f(n)
	for _, c := range n.Children {
		c.Walk(f)
	}
}

// TextContentOf returns the text content of the first element descendant
// named name, or "" when there is none.
func (n *Node) TextContentOf(name string) string {
	var out string
	n.Walk(func(c *Node) {
		if out == "" && c.Type == ElementNode && c.Name == name {
			out = c.TextContent()
		}
	})
	return out
}
```
Callers collapse to the primitive:
```go
func CollectHeadings(root *html.Node) []*Heading {
	var out []*Heading
	root.Walk(func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if lvl := headingLevel(n.Name); lvl > 0 {
			out = append(out, &Heading{Node: n, Title: CollapseWS(n.TextContent()), Level: lvl})
		}
	})
	return out
}

func docTitle(root *html.Node) string {
	if t := root.TextContentOf("title"); t != "" {
		return outline.CollapseWS(t)
	}
	return ""
}
```
and the two `collectSheets` walks shorten to a `root.Walk` with their switch.
Additive interface — no caller must move; each caller only shrinks.
- **Why:** an interface smaller than the need forces N callers to re-invent
the traversal, and each re-invention re-decides subtle facts (raw-text
elements, only-element children, order). The html package is the most-invoked
leaf module in the repo; one `Walk` moves the recursive-closure knowledge into
though where it can be tested once and used by layout, css, outline, and
both converters — leverage against every current and future consumer.

---

<a id="p2-09"></a>
## P2-09 — Own the bytes→runes seam that Parse assumes but nobody implements

- [~] **P2-09** — html.ParseDocument + load charset seam done (fix-html-load-outline); convert.go:169 call site pending; `--default-encoding` FIX-REVIEW marker in settings

- **Locations:** `internal/html/html.go:96-98` (contract), `internal/load/load.go:48-56` (ContentType written at 5 sites, read nowhere: `:225, :232, :269, :379, :409`), `internal/settings/reflect.go:110` + `internal/settings/settings.go:163-165` (`--default-encoding` accepted then Ignored)
- **Evidence sources:** area-3 F7

---

### Evidence — F7

- **Severity:** low
- **Location:** `internal/html/html.go:96-98` (contract), `internal/load/load.go:48-56` (ContentType written at 5 sites, read nowhere: `:225, :232, :269, :379, :409`), `internal/settings/reflect.go:110` + `internal/settings/settings.go:163-165` (`--default-encoding` accepted then Ignored)
- **Current (verbatim):**
```go
// Parse turns HTML source into a tree with a synthetic root. The source is
// decoded UTF-8; callers handle charset detection beforehand.
func Parse(source string) (*Node, error) {
	tok, err := tokenize(source)
```

No caller performs that promised charset detection: `convert.go:169` and
`imageout.go:203` both do `html.Parse(string(res.Body))`; `Resource.ContentType`
is set everywhere and consumed nowhere; the CLI accepts `--default-encoding`
and settings routes it to Ignored. Three locations each describe half of a
decode seam and none owns it.
- **Future:**
```go
// html — the bytes→tree seam, decided in exactly one place
func ParseDocument(body []byte) (*Node, error) {
	s := string(body)
	if strings.HasPrefix(s, "\ufeff") { // BOM, mirroring load.IsHTML
		s = s[1:]
	}
	return Parse(s)
}
```
`convert` and `imageout` move to `html.ParseDocument(res.Body)` (one line
each); the stale comment is deleted. `load` keeps `ContentType` as metadata
but gains the *declaration* of the rule — or better: `Load` checks the
Content-Type charset (and `<meta charset>`) and returns a clear
"unsupported charset: %s (only UTF-8/ASCII)" error at the load seam instead
of letting Latin-1 pages silently garble. Either way one module owns "what
encoding is this document"; the current state is a contract with no
implementer.
- **Why:** a seam nobody implements is a lying comment — the cost is paid in
mojibake debugging by end users. hypothesis: a Latin-1 or Windows-1252 page
currently renders garbled text silently; validate by adding one fixture with a
`<meta charset="windows-1252">` document asserting `ParseDocument` refuses (or
decodes) at the seam. Deepening keeps the HTML module's input contract
unconditional and moves all decode knowledge next to the bytes that need
decoding.

---

<a id="p2-10"></a>
## P2-10 — Slim `objectState` god-struct; make the pass-order handshakes explicit

- [ ] **P2-10** — pending (fix-convert)

- **Locations:** `internal/convert/outline.go:20-57` (struct); hidden-mutation handshake `internal/convert/convert.go:363-364` and `internal/convert/hf.go:247-260`; placement fields written late in `RunPDFContext`/`renderTOCObjects`
- **Evidence sources:** area-6 F4

---

### Evidence — F4

- **Severity:** medium
- **Location:** `internal/convert/outline.go:20-57` (struct); hidden-mutation handshake `internal/convert/convert.go:363-364` and `internal/convert/hf.go:247-260`; placement fields written late in `RunPDFContext`/`renderTOCObjects`
- **Current (verbatim):**
```go
type objectState struct {
	obj   *settings.PdfObject
	idx   int // 0-based object index (messages use idx+1)
	isTOC bool

	geom   hfGeom
	header settings.HeaderFooter
	footer settings.HeaderFooter
	repl   map[string]string // merged --replace map (header+footer)
	toc    settings.TableOfContent

	headerHTML *htmlHFLayout
	footerHTML *htmlHFLayout
…
	// final document placement (filled after the page reorder):
	start int // document page index of the first page
}
```
(Full struct at `outline.go:20-57`; the trimmed middle holds `registry`, `base`, `lp`, `media`, `imagesFn`, `doctitle`, `res`, `pages`, `offset`, `headings`, `tocPages`, `tocRoot`, `tocRes`.)
- **Future:** split what the load pass produces (immutable after construction) from what later passes compute, and make the registry handshake a return value:
```go
// objectState: immutable after initTOCState/renderObject construct it.
type objectState struct {
	obj, idx, isTOC, geom, header, footer, repl, toc, base, lp, media,
	imagesFn, doctitle, registry  // registry = fonts known at body-layout time
	// body: res, headings   TOC: tocRoot, tocRes
}

// objectPlacement: computed once by the page plan, then read-only.
type objectPlacement struct {
	pages  int // pre-copy page count
	offset int // body-global page index of the first page
	start  int // final document page index (after TOC reorder/copies)
}
```
```go
// hf.go — no st.registry mutation behind the caller's back.
func loadHTMLHF(ctx context.Context, loader *load.Loader, font *pdf.Font, st *objectState, rawOrURL string, log io.Writer) (l *htmlHFLayout, reg *pdf.Registry, err error) {
	...
	reg = MergeFontFaces(ctx, loader, st.registry, sheets, res.Base, st.lp, st.idx+1, log)
	if reg == nil {
		reg = st.registry
	}
	l = &htmlHFLayout{registry: reg, ...}
	return l, reg, nil
}
```
`effectiveMargins` returns the extended registry instead of writing `st.headerHTML/footerHTML`; `renderObject` threads it into `layout.Layout` directly. `drawHeadersFooters` still caches per-page layouts, but as its own pass-local state.
- **Why:** `objectState` carries 20+ fields across four files and three lifecycle stages (built in `renderObject`/`initTOCState`, mutated by `effectiveMargins`, extended by `loadHTMLHF`, finalized by `RunPDFContext`/`renderTOCObjects`). The cost is the invisible ordering invariant at convert.go:363-364 — `effectiveMargins` loads HTML HFs and can extend `st.registry`, so body layout must re-read it through a local: `// HF MergeFontFaces may have extended the registry; keep body layout in sync. registry = st.registry`. Reordering the pipeline (e.g. HF loading after body layout) silently breaks font matching with no compiler help, and `start` is invalid before the reorder pass. the split plus explicit registry returns turn two hidden dependencies into data flow. hypothesis: a body whose only face sources are `--header-html` @font-face rules currently renders with fallback metrics when HF loads late — validate by deleting `registry = st.registry` and observing a golden change.

---

---

<a id="p2-11"></a>
## P2-11 — One page-geometry/`layout.Options` constructor instead of arithmetic at 6 sites

- [ ] **P2-11** — pending (fix-convert)

- **Locations:** `internal/convert/convert.go:271-296` (initTOCState) and `:338-364` (renderObject, identical geom literal); margin arithmetic re-derived at `convert.go:320-321, 367, 383, 399`, `hf.go:260, 518-520`, `toc.go:159`; `layout.Options` literals at `convert.go:366-377, 397-412`, `hf.go:267-283, 313-326`, `toc.go:172-182`
- **Evidence sources:** area-6 F5

---

### Evidence — F5

- **Severity:** medium
- **Location:** `internal/convert/convert.go:271-296` (initTOCState) and `:338-364` (renderObject, identical geom literal); margin arithmetic re-derived at `convert.go:320-321, 367, 383, 399`, `hf.go:260, 518-520`, `toc.go:159`; `layout.Options` literals at `convert.go:366-377, 397-412`, `hf.go:267-283, 313-326`, `toc.go:172-182`
- **Current (verbatim):**
```go
	st := &objectState{
		obj:      obj,
		idx:      idx,
		isTOC:    true,
		header:   obj.HeaderFor(cmd.Global),
		footer:   obj.FooterFor(cmd.Global),
		repl:     mergedReplaces(obj, cmd.Global),
		toc:      effectiveTOC(*obj, cmd.Global),
		registry: registry,
		media:    mediaFor(cmd.Global, obj),
		geom: hfGeom{
			pageW:        pageW,
			pageH:        pageH,
			marginTop:    cmd.Global.Margin.Top * mmToPt,
			marginBottom: cmd.Global.Margin.Bottom * mmToPt,
			marginLeft:   cmd.Global.Margin.Left * mmToPt,
			marginRight:  cmd.Global.Margin.Right * mmToPt,
		},
		lp: obj.Load,
	}
	st.geom.contentH = st.geom.pageH - st.geom.marginTop - st.geom.marginBottom
	if err := effectiveMargins(ctx, loader, font, cmd, st, log); err != nil {
		return nil, fmt.Errorf("object %d: %w", idx+1, err)
	}
	return st, nil
}
```
- **Future:** one constructor derives the geometry — adding `contentW`, today recomputed six times — and one builder produces the layout options; smart-shrink, HF and TOC layouts reuse it:
```go
// newHFGeom is the single place page geometry is derived from settings.
func newHFGeom(g settings.PdfGlobal, pageW, pageH float64) hfGeom {
	return hfGeom{
		pageW: pageW, pageH: pageH,
		marginTop:    g.Margin.Top * mmToPt,
		marginBottom: g.Margin.Bottom * mmToPt,
		marginLeft:   g.Margin.Left * mmToPt,
		marginRight:  g.Margin.Right * mmToPt,
		contentW:     pageW - g.Margin.Left*mmToPt - g.Margin.Right*mmToPt,
		contentH:     pageH - g.Margin.Top*mmToPt - g.Margin.Bottom*mmToPt,
	}
}

// bodyLayoutOpts builds the layout options for one body object's content
// area; the smart-shrink re-layout calls it again with a different zoom.
func (st *objectState) bodyLayoutOpts(zoom float64) layout.Options {
	return layout.Options{
		Width: st.geom.contentW, Height: st.geom.contentH,
		Font: st.font, Registry: st.registry, Sheets: st.sheets,
		Media: st.media, Images: st.imagesFn,
		Background: st.background, Zoom: zoom,
		PrintLinkUnderline: st.printLinkUnderline,
	}
}
```
`initTOCState` and `renderObject` both become `newObjectState(...)`; the six `layout.Options` literals shrink to one builder (HF/TOC variants where fields differ keep a small `hfLayoutOpts`). No exported behaviour change (`hfGeom` is unexported).
- **Why:** the geometry literal is duplicated byte-for-byte in two constructors and the margin arithmetic is re-derived at six call sites (`st.geom.pageW - st.geom.marginLeft - st.geom.marginRight` appears at convert.go:367, 383, hf.go:260, toc.go:159, plus the `cmd.Global.Margin.**mmToPt` form at convert.go:320-321). A margin-model change must mirror six places, and one already computes `contentH` with a different fallback (hf.go:518-520). The constructor/builder pair is a small deep module: one place states "page geometry is this", and body, smart-shrink, HF and TOC layouts all read the same answer.

---

---

<a id="p2-12"></a>
## P2-12 — Drop `applyInternalLinks`' ignored `cmd`; stop `paintCount` swallowing layout errors

- [ ] **P2-12** — pending (fix-convert)

- **Locations:** `internal/convert/links.go:190-211` (`cmd` dead parameter); `internal/convert/convert.go:144` (caller); `internal/convert/toc.go:143-149` (`paintCount`)
- **Evidence sources:** area-6 F6

---

### Evidence — F6

- **Severity:** medium
- **Location:** `internal/convert/links.go:190-211` (`cmd` dead parameter); `internal/convert/convert.go:144` (caller); `internal/convert/toc.go:143-149` (`paintCount`)
- **Future:**
```go
// links.go — drop the dead parameter and the nil-doc defensiveness.
func applyInternalLinks(doc *pdf.Document, bodies []*objectState, tocTotal int) {
	if doc == nil {
		return
	}
	idLoc := buildBodyIDIndex(bodies)
	for _, st := range bodies {
		// unchanged loop body
	}
}
```
Caller moves: `convert.go:144` `applyInternalLinks(doc, bodies, tocTotal, cmd)` → `applyInternalLinks(doc, bodies, tocTotal)`.
```go
// toc.go — propagate the failure instead of guessing "1 page".
func paintCount(res *layout.Result, g hfGeom) (int, error) {
	scratch := pdf.NewDocument()
	if err := layout.Paint(scratch, cloneResult(res), paintOptions(g)); err != nil {
		return 0, fmt.Errorf("toc: paint count: %w", err)
	}
	return scratch.PageCount(), nil
}
```
`renderTOCObjects` (toc.go:155-200) propagates the error through the `fixed-point loop.
- **Current (verbatim):**
```go
func applyInternalLinks(doc *pdf.Document, bodies []*objectState, tocTotal int, cmd *cli.Command) {
	if doc == nil {
		return
	}
	_ = cmd
```
- **Why:** `_ = cmd` is a dangling surface — the parameter exists at the seam (convert.go shares it with every other pass that *does* use `cmd`) and no behaviour consumes it, misleading the next reader into thinking a `cmd`-driven policy (e.g. per-object link limits) is implemented; delete it or implement it. `paintCount`'s `return 1` is a swallowed error: a layout failure during TOC fixed-point measuring silently yields a "1 page" guess that propagates into every TOC entry page number and the reorder — the document is wrong without any error surfacing. Both changes are API-compatible and make failures visible exactly at the seam where they occur.

---

---

<a id="p2-13"></a>
## P2-13 — Remove the magic OpKind-sentinel link-neutralization leak into convert

- [ ] **P2-13** — pending (fix-convert + layout DeactivateOp)

- **Locations:** `internal/convert/links.go:13-30` (`linkOpSkip`, `stripLinkURIs`); uses at `links.go:207-208`; the contract lives in `internal/layout/paint.go:120-136` (switch with no case for 255)
- **Evidence sources:** area-6 F7

---

### Evidence — F7

- **Severity:** low
- **Location:** `internal/convert/links.go:13-30` (`linkOpSkip`, `stripLinkURIs`); uses at `links.go:207-208`; the contract lives in `internal/layout/paint.go:120-136` (switch with no case for 255)
- **Current (verbatim):**
```go
// linkOpSkip is a sentinel OpKind for neutralized link ops. Paint's switch
// has no case for it, so a skipped op paints nothing; paginateOps and
// isSplittable likewise ignore it. The op keeps its slot in the Ops slice
// because the layout engine's box tree stores op indices (opStart/opEnd)
// that Paint relies on — removing entries would corrupt pagination.
const linkOpSkip = layout.OpKind(255)

// stripLinkURIs neutralizes external (http/https/mailto) link ops in place.
// Same-document fragment links (#id) are left for applyInternalLinks.
func stripLinkURIs(ops []layout.Op) []layout.Op {
	for i := range ops {
		if ops[i].Kind == layout.OpLinkURI && !strings.HasPrefix(ops[i].URI, "#") {
			ops[i].Kind = linkOpSkip
			ops[i].URI = ""
		}
	}
	return ops
}
```
- **Future:** the neutralization contract moves into the owner of the op type, and convert calls the seam:
```go
// layout/ops.go
const opKindNoop OpKind = 255 // no case in Paint's dispatch: renders nothing

// DeactivateOp marks an op so every painter (Paint, PaintBand) and every
// pagination helper ignores it while keeping its slot in Ops: the box tree
// stores op indices (opStart/opEnd) that Paint relies on, so entries must
// not be removed.
func DeactivateOp(op *Op) {
	op.Kind = opKindNoop
	op.URI = ""
}
```
```go
// convert/links.go — the sentinel and the "Paint's switch has no case"
// knowledge leave convert's behalf callers use the seam:
layout.DeactivateOp(&ops[i])
```
`stripLinkURIs` (links.go:22-30) and `applyInternalLinks` (links.go:207-208) call `layout.DeactivateOp`; `const linkOpSkip` is deleted. No behaviour change.
- **Why:** convert encodes an invariant that belongs to layout — "OpKind 255 paints nothing because Paint's switch has no case for it" — i.e. the seam leaks the implementation type (`layout.OpKind` numbering, `Paint`'s dispatch cases, `paginateOps`/`isSplittable` behaviour). The knowledge lives in convert's comment (links.go:13-17) but is enforced in layout's switch: if layout ever grows a `default` case (e.g. erroring on unknown kinds), every neutralized link becomes a hard failure, and nothing in layout documents that 255 is reserved. The reservation and the deactivation contract belong beside the op definition; convert's call sites then read as intent ("deactivate this op"), not magic-number mutation.

---

## Notes on what was NOT reported

- `simplify.go` (72 lines, 5 exported symbols) looks shallow but is a real seam: `imageout` consumes `SimplifyDOMEnabled/Profile/AppendSimplifySheet` through convert, and the profile/default logic is non-trivial. Deleting would re-duplicate — it earns its keep.
- `RunPDF` → `RunPDFContext` is a deliberate 6-line façade; `percent`, `pageGeometry`, `mergedReplaces`, `mediaFor` are small but single-purpose.
- `DefaultTOCXSL` is a static compat string — thin by design (wkhtmltopdf interface parity), not architecture-blocking.
- The removed `_ = cmd` sits in only one function; the pipeline otherwise wraps errors with `%w` consistently and threads `ctx` everywhere — no systemic wrapping gaps found.

---

<a id="p2-14"></a>
## P2-14 — Unify the document-prep prologue (sheet harvesting + font-face merge) with imageout

- [ ] **P2-14** — pending (fix-convert side never landed; imageout = wave 2)

- **Locations:** `internal/convert/convert.go:531-602` (`collectSheets`/`styleText`/`linkStylesheet`); verbatim twin `internal/imageout/imageout.go:631-701`; prologue duplication `internal/convert/convert.go:316-336` vs `internal/imageout/imageout.go:420-477`
- **Evidence sources:** area-6 F3

---

### Evidence — F3

- **Severity:** high
- **Location:** `internal/convert/convert.go:531-602` (`collectSheets`/`styleText`/`linkStylesheet`); verbatim twin `internal/imageout/imageout.go:631-701`; prologue duplication `internal/convert/convert.go:316-336` vs `internal/imageout/imageout.go:420-477`
- **Current (verbatim):**
```go
func collectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, idx int, log io.Writer, viewportW, viewportH float64, mediaType string) []*css.Stylesheet {
	var sheets []*css.Stylesheet
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		switch n.Name {
		case "style":
			sheet, err := css.Parse(styleText(n))
			if err != nil {
				fmt.Fprintf(log, "warning: object %d: skipping <style>: %v\n", idx, err)
			} else if sheet != nil {
				sheets = append(sheets, sheet)
			}
			return // raw-text element; no element children
		case "link":
			if linkStylesheet(n, viewportW, viewportH, mediaType) {
				href := n.Attribute("href")
				r, err := loader.FetchSub(ctx, base, href, lp)
				if err != nil {
					fmt.Fprintf(log, "warning: object %d: skipping <link href=%q>: %v\n", idx, href, err)
					return
				}
				sheet, err := css.Parse(string(r.Body))
				if err != nil {
					fmt.Fprintf(log, "warning: object %d: skipping <link href=%q>: %v\n", idx, href, err)
					return
				}
				sheets = append(sheets, sheet)
			}
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
```
- **Future:** convert owns and exports the seam; imageout already imports convert (`MergeFontFaces`, `AppendSimplifySheet`), so it becomes a caller:
```go
// convert/convert.go — the one sheet-harvesting entry point.
type SheetOptions struct {
	ViewportW, ViewportH float64 // pt, for <link media> feature queries
	MediaType            string  // "print" / "screen" / "" (all)
}

func CollectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, opts SheetOptions, log io.Writer) []*css.Stylesheet {
	// body of today's collectSheets (idx folded into an objectId param);
	// keeps the 25k-rule warning.
}
```
```go
// imageout/imageout.go — the local twins are deleted; call the seam.
sheets := convert.CollectSheets(ctx, loader, root, res.Base, obj.Load, convert.SheetOptions{
	ViewportW: 768, ViewportH: 576, // today's image-mode defaults
	MediaType: mediaFor(cmd, obj),  // fixes the hardcoded "screen" bug
}, log)
```
`styleText` and `linkStylesheet` become unexported convert helpers; the imageout copies at 631-701 are deleted. Callers that must move: `internal/imageout/imageout.go` (`Run` at :454 and the three helpers at :631-701).
- **Why:** ~70 lines of identical domain logic — walk DOM, parse `<style>`, fetch `<link rel=stylesheet>` under ACL, append — live in two packages, and the copies have already diverged in both directions: convert gained viewport/media-aware link filtering and the 25k-rule warning; imageout kept a hardcoded `"screen"` + fixed 768x576 viewport (`imageout.go:698-700`), so image mode with `--print-media-type` still filters `<link media="print">` stylesheets out while the layout runs in print media — a live bug. The same holds for the wider prologue: loader setup, `DefaultFont`, font-path scan, `html.Parse`, simplify, `MergeFontFaces` and the `imagesFn` closure are re-implemented in `imageout.Run` (:420-477) against `RunPDFContext`/`renderObject`, and the two `mediaFor` helpers (convert.go:502-523, imageout.go:597-608) encode opposite defaults under the same name. The fetch→parse→sheets→fonts→image-fetcher sequence is the convert pipeline's front door; deepening it into one small interface concentrates the subresource/ACL/font knowledge and pays off across both modes.

---

> **Fan-in note:** Partner row to P2-01: P2-01 is the stylesheet gatherer module (`CollectSheets`); this row removes the duplicated prologue in `convert` vs `imageout` (loader ACL, font scanning, sheet gathering, simplify, @font-face merge) — the shell in which P5-02 (page-assembly pipeline) will land.

---
