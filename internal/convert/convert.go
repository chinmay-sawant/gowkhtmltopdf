package convert

import (
	"context"
	"errors"
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
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// mmToPt converts millimetres to PostScript points.
const mmToPt = 72.0 / 25.4

// Request is the PDF pipeline input, independent of the CLI parser. Both
// cmd mains (via RunPDFContext adapter) and the library API (wave 2) build it.
// Image is reserved for a future shared seam with imageout; PDF ignores it.
type Request struct {
	Global  settings.PdfGlobal
	Image   *settings.ImageGlobal
	Objects []settings.PdfObject
	// Output receives the finished PDF bytes. Run requires this sink to be
	// explicit; CLI adapters select stdout when the user asks for it.
	Output io.Writer
	// OutlineOutput receives --dump-outline XML. It is separate from Output so
	// diagnostics/document metadata can never be appended to a PDF stream.
	// It is only required when Global.DumpOutline is true.
	OutlineOutput io.Writer
}

// ErrMissingOutput reports a request that did not choose a document sink.
var ErrMissingOutput = errors.New("convert: output sink is required")

// ErrMissingOutlineOutput reports a dump-outline request without its metadata
// sink. Keeping this separate from Output prevents accidental mixed formats.
var ErrMissingOutlineOutput = errors.New("convert: outline output sink is required")

// ErrUnexpectedImageSettings reports an image-mode union member passed to the
// PDF engine. The shared Request remains the compatibility contract, while
// these constructors and validators make each mode's invariant explicit at
// its boundary.
var ErrUnexpectedImageSettings = errors.New("convert: image settings are not valid for PDF")

// ErrMissingImageSettings reports an image request sent through the image
// adapter without its mode-specific settings.
var ErrMissingImageSettings = errors.New("convert: image settings are required")

// NewPDFRequest builds the PDF side of the compatibility union. Callers that
// already have a writer should prefer this constructor over a partially filled
// Request literal.
func NewPDFRequest(global settings.PdfGlobal, objects []settings.PdfObject, output, outline io.Writer) *Request {
	return &Request{Global: global, Objects: objects, Output: output, OutlineOutput: outline}
}

// NewImageRequest builds the image side of the compatibility union. Image
// settings are copied so the request owns its mode configuration snapshot.
func NewImageRequest(global settings.PdfGlobal, image settings.ImageGlobal, objects []settings.PdfObject, output io.Writer) *Request {
	return &Request{Global: global, Image: &image, Objects: objects, Output: output}
}

// Validate checks the explicit output contract before any loading or font
// initialization occurs. This makes a missing sink deterministic and cheap to
// test through the engine seam.
func (r *Request) Validate() error {
	if r == nil {
		return errors.New("convert: nil request")
	}

	if r.Output == nil {
		return ErrMissingOutput
	}

	if r.Global.DumpOutline && r.OutlineOutput == nil {
		return ErrMissingOutlineOutput
	}

	return nil
}

// ValidatePDF checks the PDF-specific request invariant before running the
// document pipeline.
func (r *Request) ValidatePDF() error {
	if r != nil && r.Image != nil {
		return ErrUnexpectedImageSettings
	}

	return r.Validate()
}

// ValidateImage checks the image-specific request invariant. Image output is
// implemented by internal/imageout but shares this boundary contract.
func (r *Request) ValidateImage() error {
	if r == nil {
		return errors.New("convert: nil request")
	}

	if r.Image == nil {
		return ErrMissingImageSettings
	}

	return r.Validate()
}

// RunPDF executes the full pdf conversion with a background context and no
// progress callback. Thin adapter over RunPDFContext for existing callers.
func RunPDF(cmd *cli.Command, log io.Writer) error {
	return RunPDFContext(context.Background(), cmd, log, nil)
}

// RunPDFContext adapts a CLI parse result into a Request and runs the
// pipeline. Opening/closing the output path (or stdout) stays here so Run
// only sees an io.Writer. Prefer Run when the caller already has a writer.
func RunPDFContext(ctx context.Context, cmd *cli.Command, log io.Writer, progress func(phase string, percent int)) error {
	if cmd == nil {
		return errors.New("convert: nil command")
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return err
	}

	req := &Request{
		Global:        cmd.Global,
		Objects:       cmd.Objects,
		Output:        out,
		OutlineOutput: os.Stdout,
	}
	// CLI may still set the legacy Command.DumpOutline bit; OR into Global.
	if cmd.DumpOutline {
		req.Global.DumpOutline = true
	}

	runErr := Run(ctx, req, log, progress)
	if closeErr := closeOut(); closeErr != nil && runErr == nil {
		return closeErr
	}

	return runErr
}

// Run executes the full PDF conversion pipeline for req.
// ctx is threaded into every load; progress receives human-readable phase
// names and a 0-100 percentage as the conversion advances (nil disables it).
// Progress lines are also written to log unless req.Global.Quiet is set.
//
// Pipeline: every body object is loaded, laid out and painted (headings and
// locations are recorded); table-of-contents objects are generated from the
// collected outline and painted with a two-iteration fixed point on their
// page count; pages are reordered so all TOC pages come first; then the PDF
// outline, TOC link annotations and the per-page headers/footers are wired
// using the final page indices.
func Run(ctx context.Context, req *Request, log io.Writer, progress func(phase string, percent int)) error {
	if err := req.ValidatePDF(); err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	// load.NewLoader applies Allow / EnableLocalFileAccess from LoadGlobal.
	loader := load.NewLoader(req.Global.Load)
	loader.Log = log

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}

	registry := loadFontRegistry(req.Global, log)

	report := func(phase string, percent int) {
		if progress != nil {
			progress(phase, percent)
		}

		if log != nil && log != io.Discard && !req.Global.Quiet {
			fmt.Fprintf(log, "%s\n", phase)
		}
	}

	doc := pdf.NewDocument()
	n := len(req.Objects)

	var bodies []*objectState

	var tocs []*objectState

	for i := range req.Objects {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("object %d: %w", i+1, err)
		}

		report(fmt.Sprintf("Loading pages (%d/%d)", i+1, n), percent(i+1, n))

		obj := &req.Objects[i]
		if obj.IsTableOfContent {
			st, err := initTOCState(ctx, loader, font, registry, req, obj, i, log)
			if err != nil {
				return err
			}

			tocs = append(tocs, st)

			continue
		}

		st, err := renderObject(ctx, loader, font, registry, doc, req, obj, i, log)
		if err != nil {
			return err
		}

		if st != nil {
			bodies = append(bodies, st)
		}
	}

	headings := flatHeadings(bodies)

	// Exclude selectors are applied in outline.BuildTree (not at collect time)
	// so TOC and PDF outline share one filter path.
	exclude := parseExcludeSelectors(req.Global.ExcludeFromOutline, log)

	tocTotal := 0

	if len(tocs) > 0 {
		// Real phase: TOC layout + paint (page count unknown until finished).
		report("Building table of contents", percent(n, n+1))
		// The TOC lists the full outline (all levels); the PDF outline
		// applies outline-depth separately below.
		// Use the explicit document-page ordering contract; keep Heading.Page
		// object-local for headers, links, and page ownership.
		tocTree := outline.BuildTreeBy(headings, outline.Options{Exclude: exclude}, outline.DocumentPage)

		tocTotal, err = renderTOCObjects(ctx, font, doc, req, tocs, tocTree.Flatten(), log)
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

	if req.Global.Outline {
		outTree := outline.BuildTreeBy(headings, outline.Options{
			MaxDepth: req.Global.OutlineDepth,
			Exclude:  exclude,
		}, outline.DocumentPage)
		// --dump-outline uses its dedicated metadata sink. The engine never
		// reaches into os.Stdout; CLI adapters own stdout selection.
		if req.Global.DumpOutline {
			if _, err := req.OutlineOutput.Write(outline.DumpOutlineXMLBy(outTree, tocTotal, outline.DocumentPage)); err != nil {
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

	applyInternalLinks(doc, bodies, tocTotal)

	plan := newPagePlan(tocs, bodies, req.Global.Copies, req.Global.Collate)
	ranges := plan.Ranges()

	if req.Global.Title != "" {
		doc.SetInfo("Title", req.Global.Title)
	}

	doc.SetInfo("Producer", "gowkhtmltopdf")
	doc.SetCompression(req.Global.UseCompression)
	// Grayscale is the sole color bit (settings maps colormode → Grayscale).
	doc.SetGrayscale(req.Global.Grayscale)
	doc.SetCreationTime(time.Now())

	if plan.copies > 1 {
		if err := materializeCopies(doc, ranges, plan.copies); err != nil {
			return err
		}

		if !plan.collate {
			order := nonCollateOrder(ranges, plan.copies)
			if err := doc.ReorderPages(order); err != nil {
				return fmt.Errorf("assemble copies: %w", err)
			}
		}
	}

	// Headers/footers after copies so [page]/[topage] reflect the final page set.
	drawHeadersFooters(ctx, loader, font, doc, req, plan, headings, log)

	report("Done", 100)

	if err := doc.Write(req.Output); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
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

// pageOwner is one logical (pre-copy) page and the object that owns it.
type pageOwner struct {
	st    *objectState
	local int // page index within the object
}

// pagePlan is the single owner of the document's page-index model: the
// logical (pre-copy) order, the TOC front-matter offset, and the
// copy/collate permutation onto final document pages.
type pagePlan struct {
	owners   []pageOwner // logical page -> owning object + local index
	tocTotal int
	copies   int
	collate  bool
}

// newPagePlan builds the logical owner list (TOC pages then body pages) and
// the copy/collate parameters used by HF drawing and link remapping.
func newPagePlan(tocs, bodies []*objectState, copies int, collate bool) *pagePlan {
	if copies < 1 {
		copies = 1
	}

	pp := &pagePlan{copies: copies, collate: collate}
	for _, st := range tocs {
		pp.tocTotal += st.tocPages
		for i := range st.tocPages {
			pp.owners = append(pp.owners, pageOwner{st, i})
		}
	}

	for _, st := range bodies {
		for i := range st.pages {
			pp.owners = append(pp.owners, pageOwner{st, i})
		}
	}

	return pp
}

// OwnerOf resolves the object that owns final page p (header/footer and
// link passes). ok is false for pages outside the logical set.
func (pp *pagePlan) OwnerOf(p int) (pageOwner, bool) {
	if pp == nil {
		return pageOwner{}, false
	}

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

	if i < 0 || i >= n {
		return pageOwner{}, false
	}

	return pp.owners[i], true
}

// Remap converts a logical (pre-copy) dest page to the final page in the
// same copy group as srcPage.
func (pp *pagePlan) Remap(logicalDest, srcPage int) int {
	if pp == nil {
		return logicalDest
	}

	n := len(pp.owners)
	if pp.copies <= 1 || n <= 0 {
		return logicalDest
	}

	if pp.collate {
		return (srcPage/n)*n + logicalDest
	}

	return logicalDest*pp.copies + srcPage%pp.copies
}

// LogicalN is the number of pre-copy pages.
func (pp *pagePlan) LogicalN() int {
	if pp == nil {
		return 0
	}

	return len(pp.owners)
}

// Ranges returns per-object page spans in final (post-TOC-reorder, pre-copy)
// document order for materializeCopies / nonCollateOrder.
func (pp *pagePlan) Ranges() []pageRange {
	if pp == nil || len(pp.owners) == 0 {
		return nil
	}

	var ranges []pageRange

	var cur *objectState

	start := 0
	count := 0

	for i, own := range pp.owners {
		if cur == nil {
			cur = own.st
			start = i
			count = 1

			continue
		}

		if own.st == cur {
			count++

			continue
		}

		ranges = append(ranges, pageRange{start: start, count: count})
		cur = own.st
		start = i
		count = 1
	}

	if cur != nil {
		ranges = append(ranges, pageRange{start: start, count: count})
	}

	return ranges
}

// tocFirstOrder builds the page permutation that puts every TOC object's
// pages (in object order) before every body object's pages.
func tocFirstOrder(tocs, bodies []*objectState) []int {
	order := make([]int, 0, len(tocs)+len(bodies))

	for _, tr := range tocs {
		for i := range tr.tocPages {
			order = append(order, tr.start+i)
		}
	}

	for _, bg := range bodies {
		for i := range bg.pages {
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
		for c := range copies {
			for i := r.start; i < r.start+r.count; i++ {
				order = append(order, i+c*origTotal)
			}
		}
	}

	return order
}

// newHFGeom is the single place page geometry is derived from settings.
// contentW/contentH are the layout viewport before auto-margin resolution.
func newHFGeom(g settings.PdfGlobal) (hfGeom, error) {
	pageW, pageH, err := pageGeometry(g)
	if err != nil {
		return hfGeom{}, err
	}

	geom := hfGeom{
		pageW:        pageW,
		pageH:        pageH,
		marginTop:    g.Margin.Top * mmToPt,
		marginBottom: g.Margin.Bottom * mmToPt,
		marginLeft:   g.Margin.Left * mmToPt,
		marginRight:  g.Margin.Right * mmToPt,
	}
	geom.recomputeContent()

	return geom, nil
}

// initTOCState builds the per-object state of a table-of-contents object:
// geometry (with auto margins resolved) and the effective TOC settings.
func initTOCState(ctx context.Context, loader *load.Loader, font *pdf.Font, registry *pdf.Registry, req *Request, obj *settings.PdfObject, idx int, log io.Writer) (*objectState, error) {
	geom, err := newHFGeom(req.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d: %w", idx+1, err)
	}

	st := &objectState{
		obj:      obj,
		idx:      idx,
		isTOC:    true,
		header:   obj.HeaderFor(req.Global),
		footer:   obj.FooterFor(req.Global),
		repl:     mergedReplaces(obj, req.Global),
		toc:      effectiveTOC(*obj, req.Global),
		registry: registry,
		media:    mediaFor(req.Global, obj),
		geom:     geom,
		lp:       obj.Load,
	}

	reg, err := effectiveMargins(ctx, loader, font, req.Global, st, log)
	if err != nil {
		return nil, fmt.Errorf("object %d: %w", idx+1, err)
	}

	st.registry = reg

	return st, nil
}

// renderObject loads, lays out and paints one body object into doc and
// returns the per-object state the later passes need (nil when the load
// policy skipped the object).
func renderObject(ctx context.Context, loader *load.Loader, font *pdf.Font, registry *pdf.Registry, doc *pdf.Document, req *Request, obj *settings.PdfObject, idx int, log io.Writer) (*objectState, error) {
	geom, err := newHFGeom(req.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	media := mediaFor(req.Global, obj)

	prep, err := PrepareDocument(ctx, loader, obj.Page, obj.Load, registry, PrepareOptions{
		ViewportW:       geom.contentW,
		ViewportH:       geom.contentH,
		MediaType:       media,
		ObjectIndex:     idx + 1,
		SimplifyDOM:     SimplifyDOMEnabled(req.Global.Web, obj.Web),
		SimplifyProfile: SimplifyDOMProfile(req.Global.Web, obj.Web),
	}, log)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	if prep.Resource.Skip {
		line.Emit(log, line.Warn, "object %d (%s): load error policy is skip, omitting", idx+1, obj.Page)

		return nil, nil
	}

	root := prep.Root
	registry = prep.Registry
	sheets := prep.Sheets

	imagesFn := func(src string) ([]byte, error) {
		if !req.Global.Web.Images {
			return nil, errors.New("images disabled")
		}

		r, err := prep.Resources.Fetch(ctx, src)
		if err != nil {
			return nil, err
		}

		return r.Body, nil
	}

	printUL := req.Global.Web.PrintLinkUnderline || obj.Web.PrintLinkUnderline
	st := &objectState{
		obj:           obj,
		idx:           idx,
		header:        obj.HeaderFor(req.Global),
		footer:        obj.FooterFor(req.Global),
		repl:          mergedReplaces(obj, req.Global),
		base:          prep.Resources.Base,
		lp:            obj.Load,
		registry:      registry,
		resources:     prep.Resources,
		imagesEnabled: req.Global.Web.Images,
		media:         media,
		geom:          geom,
		imagesFn:      imagesFn,
		doctitle:      docTitle(root),
	}

	reg, err := effectiveMargins(ctx, loader, font, req.Global, st, log)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}
	// Explicit handshake: body layout uses the HF-extended registry.
	st.registry = reg
	registry = reg

	lres, err := layout.LayoutContext(ctx, root, st.bodyLayoutOpts(font, registry, sheets, obj.Load.ZoomFactor, imagesFn, req.Global.Background, printUL))
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): layout: %w", idx+1, obj.Page, err)
	}

	if req.Global.SmartShrinking {
		contentW := st.geom.contentW
		if cw := measuredWidth(lres); cw > contentW {
			// Smart shrinking: scale-to-width re-layout. The layout engine
			// scales everything by Options.Zoom; the page geometry is
			// unchanged, so the content fits the content area. A user
			// zoom factor composes multiplicatively.
			zoom := contentW / cw
			if zoom > 0 && zoom < 1 {
				line.Emit(log, line.Info, "object %d (%s): content width %.1fpt exceeds the %.1fpt content area; smart shrinking with zoom %.3f",
					idx+1, obj.Page, cw, contentW, zoom)

				effZoom := zoom
				if zf := obj.Load.ZoomFactor; zf > 0 {
					effZoom = zoom * zf
				}

				lres, err = layout.LayoutContext(ctx, root, st.bodyLayoutOpts(font, registry, sheets, effZoom, imagesFn, req.Global.Background, printUL))
				if err != nil {
					return nil, fmt.Errorf("object %d (%s): smart-shrink layout: %w", idx+1, obj.Page, err)
				}
			}
		}
	}

	if req.Global.ResolveRelativeLinks {
		resolveRelativeLinkURIs(lres.Ops, st.base)
	}

	// --no-external-links strips URI link ops before painting (the object
	// flag is the CLI's --external-links target; it defaults on).
	if !obj.ExternalLinks {
		lres.Ops = stripLinkURIs(lres.Ops)
	}

	before := doc.PageCount()

	if err := layout.PaintContext(ctx, doc, lres, paintOptions(st.geom)); err != nil {
		return nil, fmt.Errorf("object %d (%s): paint: %w", idx+1, obj.Page, err)
	}

	st.pages = doc.PageCount() - before
	st.offset = before
	st.res = lres
	st.headings = collectObjectHeadings(root, lres, before, req.Global, *obj, log)

	return st, nil
}

// bodyLayoutOpts builds layout.Options for a body (or smart-shrink) pass
// from the object's resolved geometry and shared render knobs.
func (st *objectState) bodyLayoutOpts(font *pdf.Font, registry *pdf.Registry, sheets []*css.Stylesheet, zoom float64, imagesFn func(string) ([]byte, error), background, printLinkUnderline bool) layout.Options {
	media := st.media
	if media == "" {
		media = "print"
	}

	return layout.Options{
		Width:              st.geom.contentW,
		Height:             st.geom.contentH,
		Font:               font,
		Registry:           registry,
		Sheets:             sheets,
		Media:              media,
		Zoom:               zoom,
		Images:             imagesFn,
		Background:         background,
		PrintLinkUnderline: printLinkUnderline,
	}
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

// pageGeometry resolves the page size in points from the single size model:
// Size.Width/Height (mm) override a named PageSize / Size.PageSize.
// Landscape swaps the pair. Legacy PageWidth/PageHeight fields are gone.
func pageGeometry(g settings.PdfGlobal) (w, h float64, err error) {
	if g.Size.Width > 0 && g.Size.Height > 0 {
		w, h = g.Size.Width*mmToPt, g.Size.Height*mmToPt
	} else {
		name := g.PageSize
		if name == "" {
			name = g.Size.PageSize
		}

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

// mediaFor resolves layout CSS media for PDF mode via settings.ResolveMedia.
// Object media lives on Load (CLI --media-type / print-media-type object flags);
// it is projected onto a temporary Web for the shared resolver. PDF default is "print".
func mediaFor(g settings.PdfGlobal, obj *settings.PdfObject) string {
	var objWeb *settings.Web

	if obj != nil {
		w := settings.Web{PrintMediaType: obj.Load.PrintMediaType, MediaType: obj.Load.MediaType}
		objWeb = &w
	}

	return settings.ResolveMedia("print", g.Web, objWeb)
}

// SheetOptions configures CollectSheets viewport/media gating and log labels.
// imageout (wave 2) passes fixed screen viewport defaults; PDF passes the
// object's content box.
type SheetOptions struct {
	ViewportW, ViewportH float64 // pt, for <link media> feature queries
	MediaType            string  // "print" / "screen" / "" (treated as all)
	// ObjectIndex is 1-based for warning prefixes; 0 omits "object N:".
	ObjectIndex int
}

// CollectSheets gathers <style> blocks and <link rel="stylesheet"> resources
// from the DOM in document order. A failed stylesheet only logs a warning;
// the layout proceeds without it. Shared by PDF convert and (wave 2) imageout.
func CollectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, opts SheetOptions, log io.Writer) []*css.Stylesheet {
	warn := func(format string, args ...any) {
		if log == nil {
			return
		}

		if opts.ObjectIndex > 0 {
			line.Emit(log, line.Warn, "object %d: "+format, append([]any{opts.ObjectIndex}, args...)...)
		} else {
			line.Emit(log, line.Warn, format, args...)
		}
	}

	var sheets []*css.Stylesheet

	if root != nil {
		root.Walk(func(n *html.Node) {
			if n.Type != html.ElementNode {
				return
			}

			switch n.Name {
			case "style":
				sheet, err := css.Parse(styleText(n))
				if err != nil {
					warn("skipping <style>: %v", err)
				} else if sheet != nil {
					sheets = append(sheets, sheet)
				}
			case "link":
				if linkStylesheet(n, opts.ViewportW, opts.ViewportH, opts.MediaType) {
					href := n.Attribute("href")
					r, err := loader.FetchSub(ctx, base, href, lp)
					if err != nil {
						warn("skipping <link href=%q>: %v", href, err)

						return
					}

					sheet, err := css.Parse(string(r.Body))
					if err != nil {
						warn("skipping <link href=%q>: %v", href, err)

						return
					}

					sheets = append(sheets, sheet)
				}
			}
		})
	}

	nRules := 0
	for _, s := range sheets {
		nRules += len(s.Rules)
	}

	const softRuleWarn = 25000
	if nRules >= softRuleWarn {
		warn("large stylesheet volume (%d rules); print may be slow", nRules)
	}

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
// attribute matches the conversion mediaType (empty, all, print/screen, or
// feature queries that MediaMatches accepts for that type).
func linkStylesheet(n *html.Node, viewportW, viewportH float64, mediaType string) bool {
	if n.Name != "link" || !strings.Contains(strings.ToLower(n.Attribute("rel")), "stylesheet") {
		return false
	}

	if n.Attribute("href") == "" {
		return false
	}

	media := n.Attribute("media")
	if media == "" {
		return true
	}

	return css.MediaMatches(media, mediaType, viewportW, viewportH)
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
func loadFontRegistry(g settings.PdfGlobal, log io.Writer) *pdf.Registry {
	var dirs []string

	dirs = append(dirs, g.FontPaths...)
	if g.UseSystemFonts {
		dirs = append(dirs, pdf.DefaultSystemFontDirs()...)
	}

	if len(dirs) == 0 {
		return nil
	}

	reg := pdf.ScanFontDirs(dirs)

	if log != nil && log != io.Discard && !g.Quiet {
		line.Emit(log, line.Info, "scanned %d font path(s)", len(dirs))
	}

	return reg
}

// MergeFontFaces loads @font-face url(...) TTF/OTF/WOFF1 sources into the
// registry (local and remote https via FetchSub ACL/timeouts). WOFF2 (.woff2),
// EOT, and data: src are skipped until WOFF2 decode ships. Shared by PDF
// convert and image mode.
func MergeFontFaces(ctx context.Context, loader *load.Loader, reg *pdf.Registry, sheets []*css.Stylesheet, base string, lp settings.LoadPage, idx int, log io.Writer) *pdf.Registry {
	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, ff := range sheet.FontFaces {
			for _, u := range css.FontFaceURLs(ff.Src) {
				low := strings.ToLower(u)
				if strings.HasSuffix(low, ".woff2") || strings.HasSuffix(low, ".eot") {
					line.Emit(log, line.Warn, "object %d: @font-face src %q skipped (WOFF2/EOT unsupported; WOFF1/TTF/OTF only)", idx, u)

					continue
				}
				// data: would bypass the network:// gate; reject so we never
				// ParseTTF untrusted inline payloads from CSS.
				if strings.HasPrefix(low, "data:") {
					line.Emit(log, line.Warn, "object %d: @font-face data: src skipped", idx)

					continue
				}

				r, err := loader.FetchSub(ctx, base, u, lp)
				if err != nil {
					line.Emit(log, line.Warn, "object %d: @font-face src %q: %v", idx, u, err)

					continue
				}

				f, err := pdf.ParseFontBytes(r.Body)
				if err != nil {
					line.Emit(log, line.Warn, "object %d: @font-face src %q: %v", idx, u, err)

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
