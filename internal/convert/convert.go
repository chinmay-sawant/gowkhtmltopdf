package convert

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// mmToPt converts millimetres to PostScript points.
const mmToPt = 72.0 / 25.4

// RunPDF executes the full pdf conversion with a background context and no
// progress callback: load every object, lay it out, paint all objects into
// one document, and write the result.
func RunPDF(cmd *cli.Command, log io.Writer) error {
	return RunPDFContext(context.Background(), cmd, log, nil)
}

// RunPDFContext is RunPDF with a cancellable context and a progress hook.
// ctx is threaded into every load; progress receives human-readable phase
// names and a 0-100 percentage as the conversion advances (nil disables it).
// Progress lines are also written to log unless cmd.Global.Quiet is set.
//
// Pipeline: every body object is loaded, laid out and painted (headings and
// locations are recorded); table-of-contents objects are generated from the
// collected outline and painted with a two-iteration fixed point on their
// page count; pages are reordered so all TOC pages come first; then the PDF
// outline, TOC link annotations and the per-page headers/footers are wired
// using the final page indices.
func RunPDFContext(ctx context.Context, cmd *cli.Command, log io.Writer, progress func(phase string, percent int)) error {
	loader := load.NewLoader(cmd.Global.Load)
	loader.Log = log
	loader.Allow = cmd.Global.Allow
	loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}
	registry := loadFontRegistry(cmd, log)

	report := func(phase string, percent int) {
		if progress != nil {
			progress(phase, percent)
		}
		if log != nil && log != io.Discard && !cmd.Global.Quiet {
			fmt.Fprintf(log, "%s\n", phase)
		}
	}

	doc := pdf.NewDocument()
	n := len(cmd.Objects)
	var bodies []*objectState
	var tocs []*objectState
	for i := range cmd.Objects {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("object %d: %w", i+1, err)
		}
		report(fmt.Sprintf("Loading pages (%d/%d)", i+1, n), percent(i+1, n))
		obj := &cmd.Objects[i]
		applyObjectDefaults(obj)
		if obj.IsTableOfContent {
			st, err := initTOCState(ctx, loader, font, registry, cmd, obj, i, log)
			if err != nil {
				return err
			}
			tocs = append(tocs, st)
			continue
		}
		st, err := renderObject(ctx, loader, font, registry, doc, cmd, obj, i, log)
		if err != nil {
			return err
		}
		if st != nil {
			bodies = append(bodies, st)
		}
	}

	headings := flatHeadings(bodies)

	tocTotal := 0
	if len(tocs) > 0 {
		report("Building table of contents (1/1)", 100)
		// The TOC lists the full outline (all levels); the PDF outline
		// applies outline-depth separately below.
		tocTree := outline.BuildTree(headings, outline.Options{})
		tocTotal, err = renderTOCObjects(font, doc, cmd, tocs, tocTree.Flatten(), log)
		if err != nil {
			return err
		}
		order := tocFirstOrder(tocs, bodies)
		if err := doc.ReorderPages(order); err != nil {
			return fmt.Errorf("toc assembly: %w", err)
		}
		pos := 0
		for _, tr := range tocs {
			tr.start = pos
			pos += tr.tocPages
		}
		for _, bg := range bodies {
			bg.start = tocTotal + bg.offset
		}
	}

	report("Counting pages (1/1)", 100)
	report("Resolving links (1/1)", 100)
	report("Printing pages (1/1)", 100)
	report("Done", 100)

	if cmd.Global.Outline {
		outTree := outline.BuildTree(headings, outline.Options{MaxDepth: cmd.Global.OutlineDepth})
		// --dump-outline writes the wkhtmltopdf XML to stdout; the CLI sets
		// the Command field, the reflect surface sets Global.DumpOutline.
		if cmd.DumpOutline || cmd.Global.DumpOutline {
			if _, err := os.Stdout.Write(outline.DumpOutlineXMLOffset(outTree, tocTotal)); err != nil {
				return fmt.Errorf("dump outline: %w", err)
			}
		}
		root := emitOutline(doc, outTree, bodies, tocTotal)
		if len(root.Children) > 0 {
			doc.SetOutline(root)
		}
	}

	if len(tocs) > 0 {
		applyTOCLinks(doc, tocs, bodies, tocTotal, headings)
	}
	applyInternalLinks(doc, bodies, tocTotal, cmd)

	var ranges []pageRange
	for _, tr := range tocs {
		ranges = append(ranges, pageRange{start: tr.start, count: tr.tocPages})
	}
	for _, bg := range bodies {
		ranges = append(ranges, pageRange{start: tocTotal + bg.offset, count: bg.pages})
	}

	if cmd.Global.Title != "" {
		doc.SetInfo("Title", cmd.Global.Title)
	}
	doc.SetInfo("Producer", "gowkhtmltopdf")
	doc.SetCompression(cmd.Global.UseCompression)
	doc.SetGrayscale(cmd.Global.Grayscale)
	doc.SetCreationTime(time.Now())

	if cmd.Global.Copies > 1 {
		if err := materializeCopies(doc, ranges, cmd.Global.Copies); err != nil {
			return err
		}
		if !cmd.Global.Collate {
			order := nonCollateOrder(ranges, cmd.Global.Copies)
			if err := doc.ReorderPages(order); err != nil {
				return fmt.Errorf("assemble copies: %w", err)
			}
		}
	}

	// Headers/footers after copies so [page]/[topage] reflect the final page set.
	drawHeadersFooters(ctx, loader, font, doc, cmd, tocs, bodies, tocTotal, headings, log)

	var out io.Writer = os.Stdout
	closeOut := func() error { return nil }
	if cmd.Output != "" && cmd.Output != "-" {
		f, err := os.Create(cmd.Output)
		if err != nil {
			return fmt.Errorf("output %q: %w", cmd.Output, err)
		}
		out = f
		closeOut = f.Close
	}
	if err := doc.Write(out); err != nil {
		closeOut()
		return fmt.Errorf("write %q: %w", cmd.Output, err)
	}
	return closeOut()
}

// percent rounds i/n to a 0-100 percentage.
func percent(i, n int) int {
	if n <= 0 {
		return 100
	}
	return int(math.Round(float64(i) * 100 / float64(n)))
}

// pageRange is a half-open span [start, start+count) of document page
// indices produced by one object.
type pageRange struct {
	start int
	count int
}

// tocFirstOrder builds the page permutation that puts every TOC object's
// pages (in object order) before every body object's pages.
func tocFirstOrder(tocs, bodies []*objectState) []int {
	order := make([]int, 0, len(tocs)+len(bodies))
	for _, tr := range tocs {
		for i := 0; i < tr.tocPages; i++ {
			order = append(order, tr.start+i)
		}
	}
	for _, bg := range bodies {
		for i := 0; i < bg.pages; i++ {
			order = append(order, bg.offset+i)
		}
	}
	return order
}

// materializeCopies appends fresh page objects so the document holds
// `copies` identical runs of the original page sequence, in object order.
// After this, the collated page order is exactly the document page order;
// non-collated output is obtained by a permutation (nonCollateOrder).
func materializeCopies(doc *pdf.Document, ranges []pageRange, copies int) error {
	for c := 1; c < copies; c++ {
		for _, r := range ranges {
			for i := r.start; i < r.start+r.count; i++ {
				if _, err := doc.DuplicatePage(i); err != nil {
					return fmt.Errorf("assemble copies: %w", err)
				}
			}
		}
	}
	return nil
}

// nonCollateOrder builds the /Kids permutation for non-collated output:
// each object's pages, repeated for every copy, before the next object.
// materializeCopies appended the runs in object order per copy, so copy c of
// object page i sits at i + c*origTotal, where origTotal is the page count
// before duplication.
func nonCollateOrder(ranges []pageRange, copies int) []int {
	origTotal := 0
	for _, r := range ranges {
		origTotal += r.count
	}
	var order []int
	for _, r := range ranges {
		for c := 0; c < copies; c++ {
			for i := r.start; i < r.start+r.count; i++ {
				order = append(order, i+c*origTotal)
			}
		}
	}
	return order
}

// initTOCState builds the per-object state of a table-of-contents object:
// geometry (with auto margins resolved) and the effective TOC settings.
func initTOCState(ctx context.Context, loader *load.Loader, font *pdf.Font, registry *pdf.Registry, cmd *cli.Command, obj *settings.PdfObject, idx int, log io.Writer) (*objectState, error) {
	pageW, pageH, err := pageGeometry(cmd.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d: %w", idx+1, err)
	}
	st := &objectState{
		obj:      obj,
		idx:      idx,
		isTOC:    true,
		header:   obj.HeaderFor(cmd.Global),
		footer:   obj.FooterFor(cmd.Global),
		repl:     mergedReplaces(obj, cmd.Global),
		toc:      effectiveTOC(*obj, cmd.Global),
		registry: registry,
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

// renderObject loads, lays out and paints one body object into doc and
// returns the per-object state the later passes need (nil when the load
// policy skipped the object).
func renderObject(ctx context.Context, loader *load.Loader, font *pdf.Font, registry *pdf.Registry, doc *pdf.Document, cmd *cli.Command, obj *settings.PdfObject, idx int, log io.Writer) (*objectState, error) {
	res, err := loader.Load(ctx, obj.Page, obj.Load)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): load: %w", idx+1, obj.Page, err)
	}
	if res.Skip {
		fmt.Fprintf(log, "warning: object %d (%s): load error policy is skip, omitting\n", idx+1, obj.Page)
		return nil, nil
	}

	root, err := html.Parse(string(res.Body))
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): parse html: %w", idx+1, obj.Page, err)
	}

	sheets := collectSheets(ctx, loader, root, res.Base, obj.Load, idx+1, log)
	registry = MergeFontFaces(ctx, loader, registry, sheets, res.Base, obj.Load, idx+1, log)

	pageW, pageH, err := pageGeometry(cmd.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	imagesFn := func(src string) ([]byte, error) {
		if !cmd.Global.Web.Images {
			return nil, fmt.Errorf("images disabled")
		}
		r, err := loader.FetchSub(ctx, res.Base, src, obj.Load)
		if err != nil {
			return nil, err
		}
		return r.Body, nil
	}

	st := &objectState{
		obj:      obj,
		idx:      idx,
		header:   obj.HeaderFor(cmd.Global),
		footer:   obj.FooterFor(cmd.Global),
		repl:     mergedReplaces(obj, cmd.Global),
		base:     res.Base,
		lp:       obj.Load,
		registry: registry,
		geom: hfGeom{
			pageW:        pageW,
			pageH:        pageH,
			marginTop:    cmd.Global.Margin.Top * mmToPt,
			marginBottom: cmd.Global.Margin.Bottom * mmToPt,
			marginLeft:   cmd.Global.Margin.Left * mmToPt,
			marginRight:  cmd.Global.Margin.Right * mmToPt,
		},
		imagesFn: imagesFn,
		doctitle: docTitle(root),
	}
	st.geom.contentH = st.geom.pageH - st.geom.marginTop - st.geom.marginBottom
	if err := effectiveMargins(ctx, loader, font, cmd, st, log); err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}
	// HF MergeFontFaces may have extended the registry; keep body layout in sync.
	registry = st.registry

	lres, err := layout.Layout(root, layout.Options{
		Width:      st.geom.pageW - st.geom.marginLeft - st.geom.marginRight,
		Height:     st.geom.contentH,
		Font:       font,
		Registry:   registry,
		Sheets:     sheets,
		Media:      "print",
		Zoom:       obj.Load.ZoomFactor,
		Images:     imagesFn,
		Background: cmd.Global.Background,
	})
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): layout: %w", idx+1, obj.Page, err)
	}

	if cmd.Global.SmartShrinking {
		contentW := st.geom.pageW - st.geom.marginLeft - st.geom.marginRight
		if cw := measuredWidth(lres); cw > contentW {
			// Smart shrinking: scale-to-width re-layout. The layout engine
			// scales everything by Options.Zoom; the page geometry is
			// unchanged, so the content fits the content area. A user
			// zoom factor composes multiplicatively.
			zoom := contentW / cw
			if zoom > 0 && zoom < 1 {
				fmt.Fprintf(log, "info: object %d (%s): content width %.1fpt exceeds the %.1fpt content area; smart shrinking with zoom %.3f\n",
					idx+1, obj.Page, cw, contentW, zoom)
				effZoom := zoom
				if zf := obj.Load.ZoomFactor; zf > 0 {
					effZoom = zoom * zf
				}
				lres, err = layout.Layout(root, layout.Options{
					Width:      contentW,
					Height:     st.geom.pageH - st.geom.marginTop - st.geom.marginBottom,
					Font:       font,
					Registry:   registry,
					Sheets:     sheets,
					Media:      "print",
					Zoom:       effZoom,
					Images:     imagesFn,
					Background: cmd.Global.Background,
				})
				if err != nil {
					return nil, fmt.Errorf("object %d (%s): smart-shrink layout: %w", idx+1, obj.Page, err)
				}
			}
		}
	}

	if cmd.Global.ResolveRelativeLinks {
		resolveRelativeLinkURIs(lres.Ops, st.base)
	}

	// --no-external-links strips URI link ops before painting (the object
	// flag is the CLI's --external-links target; it defaults on).
	if !obj.ExternalLinks {
		lres.Ops = stripLinkURIs(lres.Ops)
	}

	before := doc.PageCount()
	if err := layout.Paint(doc, lres, paintOptions(st.geom)); err != nil {
		return nil, fmt.Errorf("object %d (%s): paint: %w", idx+1, obj.Page, err)
	}
	st.pages = doc.PageCount() - before
	st.offset = before
	st.res = lres
	st.headings = collectObjectHeadings(root, lres, before, cmd.Global, *obj, log)
	return st, nil
}

// applyObjectDefaults resolves the per-object defaults for the flags Phase 6
// consults (external-links, local-links, use-outline, include-in-outline).
// settings.DefaultPdfObject defines all four as ON, but internal/cli builds
// objects as zero values and never applies those defaults, so a false
// boolean is ambiguous between "unset" and an explicit --no-* flag. This
// build resolves the ambiguity in favour of the documented default: the
// gates always read as on and an explicit --no-external-links on an object
// is accepted but not honored (pre-existing CLI default quirk; fixing it
// belongs in internal/cli, which is out of scope here). Library callers
// wanting the gates off must be handled there as well.
func applyObjectDefaults(o *settings.PdfObject) {
	def := settings.DefaultPdfObject()
	o.ExternalLinks = o.ExternalLinks || def.ExternalLinks
	o.LocalLinks = o.LocalLinks || def.LocalLinks
	o.UseOutline = o.UseOutline || def.UseOutline
	o.IncludeInOutline = o.IncludeInOutline || def.IncludeInOutline
}

// mergedReplaces merges the --replace maps of the global and object header
// and footer settings. The CLI stores --replace on the header only; merging
// all four surfaces keeps footer --replace working for library users.
func mergedReplaces(obj *settings.PdfObject, g settings.PdfGlobal) map[string]string {
	out := map[string]string{}
	for k, v := range g.Header.Replace {
		out[k] = v
	}
	for k, v := range obj.Header.Replace {
		out[k] = v
	}
	for k, v := range g.Footer.Replace {
		out[k] = v
	}
	for k, v := range obj.Footer.Replace {
		out[k] = v
	}
	return out
}

// measuredWidth returns the effective content width of a layout result: the
// reported Result.Width, raised to the widest visual op extent when the
// report only mirrors the viewport (layout currently sets Result.Width to
// Options.Width - see internal/layout/layout.go - so over-wide fixed-width
// boxes show up only as op extents). Text and link ops never force a page
// wider, so they are ignored; rects and images are what push content out.
func measuredWidth(res *layout.Result) float64 {
	w := res.Width
	for _, op := range res.Ops {
		switch op.Kind {
		case layout.OpFillRect, layout.OpStrokeRect, layout.OpImage:
			if ext := op.X + op.W; ext > w {
				w = ext
			}
		}
	}
	return w
}

// pageGeometry resolves the page size in points. Explicit size.width/height
// (mm) win over a named size; landscape swaps the pair.
func pageGeometry(g settings.PdfGlobal) (w, h float64, err error) {
	name := g.PageSize
	if name == "" {
		name = g.Size.PageSize
	}
	if g.Size.Width > 0 && g.Size.Height > 0 {
		w, h = g.Size.Width*mmToPt, g.Size.Height*mmToPt
	} else {
		w, h, err = settings.ParsePageSize(name)
		if err != nil {
			return 0, 0, err
		}
	}
	if g.Orientation == settings.OrientationLandscape {
		w, h = h, w
	}
	return w, h, nil
}

// collectSheets gathers <style> blocks and <link rel="stylesheet"> resources
// from the DOM. A failed stylesheet only logs a warning; the layout proceeds
// without it.
func collectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, idx int, log io.Writer) []*css.Stylesheet {
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
			if linkStylesheet(n) {
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
	return sheets
}

// styleText concatenates the raw text of a <style> element.
func styleText(n *html.Node) string {
	var sb strings.Builder
	for _, c := range n.Children {
		if c.Type == html.TextNode {
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

// linkStylesheet reports whether n is a stylesheet <link> whose media
// attribute allows print output: empty, or containing "print" or "all".
func linkStylesheet(n *html.Node) bool {
	if n.Name != "link" || !strings.Contains(strings.ToLower(n.Attribute("rel")), "stylesheet") {
		return false
	}
	if n.Attribute("href") == "" {
		return false
	}
	media := strings.ToLower(n.Attribute("media"))
	return media == "" || strings.Contains(media, "print") || strings.Contains(media, "all")
}

// DefaultTOCXSL returns the default TOC stylesheet. In pure Go the default
// TOC look is a built-in Go template; this returns a description of it for
// --dump-default-toc-xsl compatibility.
func DefaultTOCXSL() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!-- gowkhtmltopdf default TOC stylesheet.
     Upstream ships an XSLT here; the pure-Go implementation uses an
     equivalent built-in template (see internal/convert/toc.go). -->
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" indent="yes"/>
  <xsl:template match="/">
    <h1>Table of Contents</h1>
    <ul id="toc"/>
  </xsl:template>
</xsl:stylesheet>
`
}

// resolveRelativeLinkURIs rewrites non-absolute, non-fragment OpLinkURI
// values against the page base URL when --resolve-relative-links is on
// (default). Fragments (#id) and scheme URLs are left unchanged.
func resolveRelativeLinkURIs(ops []layout.Op, base string) {
	if base == "" {
		return
	}
	bu, err := url.Parse(base)
	if err != nil || bu == nil {
		return
	}
	for i := range ops {
		if ops[i].Kind != layout.OpLinkURI || ops[i].URI == "" {
			continue
		}
		u := ops[i].URI
		if strings.HasPrefix(u, "#") || strings.Contains(u, "://") || strings.HasPrefix(strings.ToLower(u), "mailto:") {
			continue
		}
		ref, err := url.Parse(u)
		if err != nil {
			continue
		}
		ops[i].URI = bu.ResolveReference(ref).String()
	}
}

// loadFontRegistry builds the opt-in font registry from --font-path and
// optional --use-system-fonts. Returns nil when nothing was configured.
func loadFontRegistry(cmd *cli.Command, log io.Writer) *pdf.Registry {
	var dirs []string
	dirs = append(dirs, cmd.Global.FontPaths...)
	if cmd.Global.UseSystemFonts {
		dirs = append(dirs, pdf.DefaultSystemFontDirs()...)
	}
	if len(dirs) == 0 {
		return nil
	}
	reg := pdf.ScanFontDirs(dirs)
	if log != nil && log != io.Discard && !cmd.Global.Quiet {
		fmt.Fprintf(log, "info: scanned %d font path(s)\n", len(dirs))
	}
	return reg
}

// MergeFontFaces loads local @font-face url(...) TTF/OTF/WOFF1 sources into
// the registry. WOFF2 (.woff2), EOT, data:, and remote https:// (non-file)
// src are skipped by product policy. ACL follows FetchSub. Shared by PDF
// convert and image mode so both honor the same local @font-face subset.
func MergeFontFaces(ctx context.Context, loader *load.Loader, reg *pdf.Registry, sheets []*css.Stylesheet, base string, lp settings.LoadPage, idx int, log io.Writer) *pdf.Registry {
	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}
		for _, ff := range sheet.FontFaces {
			for _, u := range css.FontFaceURLs(ff.Src) {
				low := strings.ToLower(u)
				if strings.HasSuffix(low, ".woff2") || strings.HasSuffix(low, ".eot") {
					fmt.Fprintf(log, "warning: object %d: @font-face src %q skipped (WOFF2/EOT unsupported; WOFF1/TTF/OTF only)\n", idx, u)
					continue
				}
				// data: would bypass the network:// gate; reject so we never
				// ParseTTF untrusted inline payloads from CSS.
				if strings.HasPrefix(low, "data:") {
					fmt.Fprintf(log, "warning: object %d: @font-face data: src skipped\n", idx)
					continue
				}
				// Remote network fonts are unsupported by design (ACL/network
				// policy): no auto-fetch of https:// webfont CDNs.
				if strings.Contains(low, "://") && !strings.HasPrefix(low, "file:") {
					fmt.Fprintf(log, "warning: object %d: @font-face network src %q skipped\n", idx, u)
					continue
				}
				r, err := loader.FetchSub(ctx, base, u, lp)
				if err != nil {
					fmt.Fprintf(log, "warning: object %d: @font-face src %q: %v\n", idx, u, err)
					continue
				}
				f, err := pdf.ParseFontBytes(r.Body)
				if err != nil {
					fmt.Fprintf(log, "warning: object %d: @font-face src %q: %v\n", idx, u, err)
					continue
				}
				if ff.Family != "" {
					f.PostScriptName = strings.ReplaceAll(ff.Family, " ", "")
				}
				if reg == nil {
					reg = pdf.NewRegistry()
				}
				reg.AddFont(f)
				if ff.Family != "" {
					reg.AddFamilyAlias(ff.Family, f)
				}
			}
		}
	}
	return reg
}
