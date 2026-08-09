package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
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

// smartShrinkMinOverflow is the smallest content overflow (in points) that
// justifies a full smart-shrink re-layout. Sub-tenth-point overflows are
// float rounding in box geometry (the benchmark report overflows its 538.6pt
// content area by 0.00pt and re-laid out 500 pages at zoom 1.000); zooming
// a whole document for an invisible hair costs a second full layout pass.
const smartShrinkMinOverflow = 0.1

// progressComplete is the final progress percentage reported to the CLI.
const progressComplete = 100

// mediaPrint is the default CSS media type for PDF layout.
const mediaPrint = "print"

// These high safety bounds protect the conversion pipeline's slice and copy
// multipliers while leaving ordinary documents and the checked benchmark
// corpus well below the limit.
const (
	maxConversionObjects = 10_000
	maxConversionCopies  = 1_000
	maxConversionPages   = 100_000
	maxStylesheetRules   = 1_000_000
)

// Request is the PDF pipeline input, independent of the CLI parser. Both
// cmd mains (via RunPDFContext adapter) and the library API (wave 2) build it.
// Image is reserved for a future shared seam with imageout; PDF ignores it.
type Request struct {
	Global  settings.PdfGlobal
	Image   *settings.ImageGlobal
	Objects []settings.PdfObject
	// Now supplies conversion metadata time. A nil function uses the
	// production wall clock; tests and deterministic callers can inject a
	// stable value shared by PDF metadata and header/footer substitutions.
	Now func() time.Time
	// Output receives the finished PDF bytes. Run requires this sink to be
	// explicit; CLI adapters select stdout when the user asks for it.
	Output io.Writer
	// OutlineOutput receives --dump-outline XML. It is separate from Output so
	// diagnostics/document metadata can never be appended to a PDF stream.
	// It is only required when Global.DumpOutline is true.
	OutlineOutput io.Writer
}

func (r *Request) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}

	return time.Now()
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

// errNilRequest reports a nil Request at a method boundary.
var errNilRequest = errors.New("convert: nil request")

// errNilCommand reports a nil cli.Command to the CLI adapter.
var errNilCommand = errors.New("convert: nil command")

// errImagesDisabled reports an image request made while images are disabled.
var errImagesDisabled = errors.New("images disabled")

var (
	errTooManyObjects = errors.New("convert: object limit exceeded")
	errTooManyCopies  = errors.New("convert: copy limit exceeded")
	errTooManyPages   = errors.New("convert: page limit exceeded")
	errTooManyStyles  = errors.New("convert: stylesheet rule limit exceeded")
)

// NewPDFRequest builds the PDF side of the compatibility union. Callers that
// already have a writer should prefer this constructor over a partially filled
// Request literal.
func NewPDFRequest(global settings.PdfGlobal, objects []settings.PdfObject, output, outline io.Writer) *Request {
	return &Request{ //nolint:exhaustruct // intentional zero-value fields
		Global:        global,
		Objects:       objects,
		Output:        output,
		OutlineOutput: outline,
	}
}

// NewImageRequest builds the image side of the compatibility union. Image
// settings are copied so the request owns its mode configuration snapshot.
func NewImageRequest(global settings.PdfGlobal, image settings.ImageGlobal, objects []settings.PdfObject, output io.Writer) *Request { //nolint:lll // constructor signature
	return &Request{ //nolint:exhaustruct // intentional zero-value fields
		Global:  global,
		Image:   &image,
		Objects: objects,
		Output:  output,
	}
}

// Validate checks the explicit output contract before any loading or font
// initialization occurs. This makes a missing sink deterministic and cheap to
// test through the engine seam.
func (r *Request) Validate() error {
	if r == nil {
		return errNilRequest
	}

	// PageSize is the canonical geometry key. Keep the legacy Size.PageSize
	// mirror synchronized at the request boundary so direct settings literals
	// cannot make page geometry depend on which consumer reads first.
	if strings.TrimSpace(r.Global.PageSize) == "" {
		r.Global.PageSize = r.Global.Size.PageSize
	}
	r.Global.Size.PageSize = r.Global.PageSize

	if r.Output == nil {
		return ErrMissingOutput
	}

	if len(r.Objects) > maxConversionObjects {
		return fmt.Errorf("%w: got %d, limit %d", errTooManyObjects, len(r.Objects), maxConversionObjects)
	}

	if r.Global.Copies > maxConversionCopies {
		return fmt.Errorf("%w: got %d, limit %d", errTooManyCopies, r.Global.Copies, maxConversionCopies)
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
		return errNilRequest
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
func RunPDFContext(ctx context.Context, cmd *cli.Command, log io.Writer, progress func(phase string, percent int)) (err error) { //nolint:lll // CLI adapter signature
	if cmd == nil {
		return errNilCommand
	}
	if ctx == nil {
		return errNilContext
	}

	outline := cmd.OutlineWriter
	if outline == nil {
		outline = io.Discard
	}

	req := NewPDFRequest(cmd.Global, cmd.Objects, io.Discard, outline)
	// CLI may still set the legacy Command.DumpOutline bit; OR into Global.
	if cmd.DumpOutline {
		req.Global.DumpOutline = true
	}
	if err := req.ValidatePDF(); err != nil {
		return err
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("open output: %w", err)
	}
	defer func() {
		if closeErr := closeOut(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	req.Output = out

	return Run(ctx, req, log, progress)
}

// runContext owns the dependencies for one conversion lifecycle.
type runContext struct {
	ctx      context.Context
	req      *Request
	loader   *load.Loader
	font     *pdf.Font
	registry *pdf.Registry
	doc      *pdf.Document
	log      io.Writer
	progress func(phase string, percent int)
}

func (run *runContext) report(phase string, value int) {
	if run.progress != nil {
		run.progress(phase, value)
	}

	if run.log != nil && run.log != io.Discard && !run.req.Global.Quiet {
		fmt.Fprintf(run.log, "%s\n", phase)
	}
}

func (run *runContext) renderObjects() (tocs, bodies []*objectState, err error) {
	count := len(run.req.Objects)
	for idx := range run.req.Objects {
		if err := run.ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("object %d: %w", idx+1, err)
		}

		run.report(fmt.Sprintf("Loading pages (%d/%d)", idx+1, count), percent(idx+1, count))

		obj := &run.req.Objects[idx]
		if obj.IsTableOfContent {
			state, err := initTOCState(run, obj, idx)
			if err != nil {
				return nil, nil, err
			}

			tocs = append(tocs, state)
			continue
		}

		state, err := renderObject(run, obj, idx)
		if err != nil {
			return nil, nil, err
		}
		if state != nil {
			bodies = append(bodies, state)
		}
	}

	return tocs, bodies, nil
}

// Run executes the full PDF conversion pipeline for req.
// ctx is threaded into every load; progress receives human-readable phase
// names and a 0-100 percentage as the conversion advances (nil disables it).
// Progress lines are also written to log unless req.Global.Quiet is set.
//
// The lifecycle is intentionally explicit: object rendering, outline/TOC
// assembly, copy materialization, and final output each have one owner.
func Run(ctx context.Context, req *Request, log io.Writer, progress func(phase string, percent int)) error {
	if err := req.ValidatePDF(); err != nil {
		return err
	}

	if ctx == nil {
		return errNilContext
	}
	// Construct the loader at the request boundary so invalid proxy policy is
	// returned before fonts, layout state, or document output are initialized.
	loader, err := load.NewLoaderWithError(req.Global.Load)
	if err != nil {
		return fmt.Errorf("initialize loader: %w", err)
	}

	loader.Log = log

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}

	registry := loadFontRegistry(req.Global, log)
	run := &runContext{
		ctx:      ctx,
		req:      req,
		loader:   loader,
		font:     font,
		registry: registry,
		doc:      pdf.NewDocument(),
		log:      log,
		progress: progress,
	}

	tocs, bodies, err := run.renderObjects()
	if err != nil {
		return err
	}

	headings := flatHeadings(bodies)

	// Exclude selectors are applied in outline.BuildTree (not at collect time)
	// so TOC and PDF outline share one filter path.
	exclude := parseExcludeSelectors(req.Global.ExcludeFromOutline, log)

	tocTotal := 0

	if len(tocs) > 0 {
		// Real phase: TOC layout + paint (page count unknown until finished).
		run.report("Building table of contents", percent(len(req.Objects), len(req.Objects)+1))
		// The TOC lists the full outline (all levels); the PDF outline
		// applies outline-depth separately below.
		// Use the explicit document-page ordering contract; keep Heading.Page
		// object-local for headers, links, and page ownership.
		tocTree := outline.BuildTreeBy(headings, outline.Options{ //nolint:exhaustruct // intentional zero-value fields
			Exclude: exclude,
		}, outline.DocumentPage)

		tocTotal, err = renderTOCObjects(ctx, font, run.doc, req, tocs, tocTree.Flatten(), log)
		if err != nil {
			return err
		}

		order := tocFirstOrder(tocs, bodies)
		if err := run.doc.ReorderPages(order); err != nil {
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

		root := emitOutline(run.doc, outTree, bodies, tocTotal)
		if len(root.Children) > 0 {
			run.doc.SetOutline(root)
		}
	}

	if len(tocs) > 0 {
		applyTOCLinks(run.doc, tocs, bodies, tocTotal, headings)
	}

	applyInternalLinks(run.doc, bodies, tocTotal)

	plan, err := newPagePlan(tocs, bodies, req.Global.Copies, req.Global.Collate)
	if err != nil {
		return err
	}
	ranges := plan.Ranges()

	if req.Global.Title != "" {
		run.doc.SetInfo("Title", req.Global.Title)
	}

	run.doc.SetInfo("Producer", "gowkhtmltopdf")
	run.doc.SetCompression(req.Global.UseCompression)
	// Grayscale is the sole color bit (settings maps colormode → Grayscale).
	run.doc.SetGrayscale(req.Global.Grayscale)
	run.doc.SetCreationTime(req.now())

	if plan.copies > 1 {
		if err := materializeCopies(run.doc, ranges, plan.copies); err != nil {
			return err
		}

		if !plan.collate {
			order := nonCollateOrder(ranges, plan.copies)
			if err := run.doc.ReorderPages(order); err != nil {
				return fmt.Errorf("assemble copies: %w", err)
			}
		}
	}

	// Headers/footers after copies so [page]/[topage] reflect the final page
	// set. Missing required bands are conversion failures; silently producing a
	// PDF without requested header/footer content is not a successful render.
	hfResult := drawHeadersFootersResult(ctx, loader, font, run.doc, req, plan, headings, log)
	if err := hfResult.Err(); err != nil {
		return fmt.Errorf("header/footer: %w", err)
	}

	run.report("Done", progressComplete)

	if err := run.doc.Write(req.Output); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// percent rounds i/n to a 0-100 percentage.
func percent(i, n int) int {
	if n <= 0 {
		return progressComplete
	}

	return int(math.Round(float64(i) * float64(progressComplete) / float64(n)))
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
func newPagePlan(tocs, bodies []*objectState, copies int, collate bool) (*pagePlan, error) {
	if copies < 1 {
		copies = 1
	}
	if copies > maxConversionCopies {
		return nil, fmt.Errorf("%w: got %d, limit %d", errTooManyCopies, copies, maxConversionCopies)
	}

	logicalPages := 0
	for _, state := range tocs {
		logicalPages += state.tocPages
	}
	for _, state := range bodies {
		logicalPages += state.pages
	}
	if logicalPages > maxConversionPages {
		return nil, fmt.Errorf("%w: got %d, limit %d", errTooManyPages, logicalPages, maxConversionPages)
	}
	if logicalPages > 0 && copies > maxConversionPages/logicalPages {
		return nil, fmt.Errorf(
			"%w: %d pages x %d copies exceeds %d",
			errTooManyPages,
			logicalPages,
			copies,
			maxConversionPages,
		)
	}

	pagePlan := &pagePlan{
		owners:  make([]pageOwner, 0, logicalPages),
		copies:  copies,
		collate: collate,
	}
	for _, st := range tocs {
		pagePlan.tocTotal += st.tocPages
		for i := range st.tocPages {
			pagePlan.owners = append(pagePlan.owners, pageOwner{st, i})
		}
	}

	for _, st := range bodies {
		for i := range st.pages {
			pagePlan.owners = append(pagePlan.owners, pageOwner{st, i})
		}
	}

	return pagePlan, nil
}

// OwnerOf resolves the object that owns final page p (header/footer and
// link passes). ok is false for pages outside the logical set.
func (pp *pagePlan) OwnerOf(page int) (pageOwner, bool) {
	if pp == nil {
		return pageOwner{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	count := len(pp.owners)
	if count == 0 {
		return pageOwner{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	var idx int

	switch {
	case pp.copies <= 1:
		idx = page
	case pp.collate:
		idx = page % count
	default: // non-collate: copies of page i are contiguous
		idx = page / pp.copies
	}

	if idx < 0 || idx >= count {
		return pageOwner{}, false //nolint:exhaustruct // intentional zero-value fields
	}

	return pp.owners[idx], true
}

// Remap converts a logical (pre-copy) dest page to the final page in the
// same copy group as srcPage.
func (pp *pagePlan) Remap(logicalDest, srcPage int) int {
	if pp == nil {
		return logicalDest
	}

	count := len(pp.owners)
	if pp.copies <= 1 || count <= 0 {
		return logicalDest
	}

	if pp.collate {
		return (srcPage/count)*count + logicalDest
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

	ranges := make([]pageRange, 0, len(pp.owners))

	var cur *objectState

	start := 0
	count := 0

	for idx, own := range pp.owners {
		if cur == nil {
			cur = own.st
			start = idx
			count = 1

			continue
		}

		if own.st == cur {
			count++

			continue
		}

		ranges = append(ranges, pageRange{start: start, count: count})
		cur = own.st
		start = idx
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
	if copies < 1 {
		return nil
	}
	if copies > maxConversionCopies {
		return fmt.Errorf("%w: got %d, limit %d", errTooManyCopies, copies, maxConversionCopies)
	}

	origTotal := 0
	for _, r := range ranges {
		origTotal += r.count
	}
	if origTotal > maxConversionPages || (origTotal > 0 && copies > maxConversionPages/origTotal) {
		return fmt.Errorf(
			"%w: %d pages x %d copies exceeds %d",
			errTooManyPages,
			origTotal,
			copies,
			maxConversionPages,
		)
	}

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

	order := make([]int, 0, origTotal*copies)

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
func newHFGeom(glob settings.PdfGlobal) (hfGeom, error) {
	pageW, pageH, err := pageGeometry(glob)
	if err != nil {
		return hfGeom{}, err
	}

	geom := hfGeom{ //nolint:exhaustruct // intentional zero-value fields
		pageW:        pageW,
		pageH:        pageH,
		marginTop:    glob.Margin.Top * mmToPt,
		marginBottom: glob.Margin.Bottom * mmToPt,
		marginLeft:   glob.Margin.Left * mmToPt,
		marginRight:  glob.Margin.Right * mmToPt,
	}
	geom.recomputeContent()

	return geom, nil
}

// initTOCState builds the per-object state of a table-of-contents object:
// geometry (with auto margins resolved) and the effective TOC settings.
func initTOCState(run *runContext, obj *settings.PdfObject, idx int) (*objectState, error) {
	geom, err := newHFGeom(run.req.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d: %w", idx+1, err)
	}

	state := &objectState{ //nolint:exhaustruct // intentional zero-value fields
		obj:      obj,
		idx:      idx,
		isTOC:    true,
		header:   obj.HeaderFor(run.req.Global),
		footer:   obj.FooterFor(run.req.Global),
		repl:     mergedReplaces(obj, run.req.Global),
		toc:      effectiveTOC(*obj, run.req.Global),
		registry: run.registry,
		media:    mediaFor(run.req.Global, obj),
		geom:     geom,
		lp:       obj.Load,
	}

	reg, err := effectiveMargins(run.ctx, run.loader, run.font, run.req.Global, state, run.log)
	if err != nil {
		return nil, fmt.Errorf("object %d: %w", idx+1, err)
	}

	state.registry = reg

	return state, nil
}

// renderObject loads, lays out and paints one body object into doc and
// returns the per-object state the later passes need (nil when the load
// policy skipped the object).
func renderObject(run *runContext, obj *settings.PdfObject, idx int) (*objectState, error) {
	geom, err := newHFGeom(run.req.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	media := mediaFor(run.req.Global, obj)

	prep, err := PrepareDocument(run.ctx, run.loader, obj.Page, obj.Load, run.registry, PrepareOptions{
		ViewportW:       geom.contentW,
		ViewportH:       geom.contentH,
		MediaType:       media,
		ObjectIndex:     idx + 1,
		SimplifyDOM:     SimplifyDOMEnabled(run.req.Global.Web, obj.Web),
		SimplifyProfile: SimplifyDOMProfile(run.req.Global.Web, obj.Web),
	}, run.log)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	if prep.Resource.Skip {
		line.Emit(run.log, line.Warn, "object %d (%s): load error policy is skip, omitting", idx+1, obj.Page)

		return nil, nil //nolint:nilnil // skip is a valid (nil, nil) outcome handled by Run
	}

	root := prep.Root
	registry := prep.Registry
	sheets := prep.Sheets

	imagesFn := func(src string) ([]byte, error) {
		if !run.req.Global.Web.Images {
			return nil, errImagesDisabled
		}

		r, err := prep.Resources.Fetch(run.ctx, src)
		if err != nil {
			return nil, err
		}

		return r.Body, nil
	}

	printUL := run.req.Global.Web.PrintLinkUnderline || obj.Web.PrintLinkUnderline
	state := &objectState{ //nolint:exhaustruct // intentional zero-value fields
		obj:           obj,
		idx:           idx,
		header:        obj.HeaderFor(run.req.Global),
		footer:        obj.FooterFor(run.req.Global),
		repl:          mergedReplaces(obj, run.req.Global),
		base:          prep.Resources.Base,
		lp:            obj.Load,
		registry:      registry,
		resources:     prep.Resources,
		imagesEnabled: run.req.Global.Web.Images,
		media:         media,
		geom:          geom,
		imagesFn:      imagesFn,
		doctitle:      docTitle(root),
	}

	reg, err := effectiveMargins(run.ctx, run.loader, run.font, run.req.Global, state, run.log)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}
	// Explicit handshake: body layout uses the HF-extended registry.
	state.registry = reg
	registry = reg

	objectRender := objectRenderContext{
		global:             run.req.Global,
		obj:                obj,
		font:               run.font,
		registry:           registry,
		sheets:             sheets,
		zoom:               obj.Load.ZoomFactor,
		imagesFn:           imagesFn,
		printLinkUnderline: printUL,
	}

	if plan, ok := benchmarkPageIslandPlan(root); ok {
		if err := renderBenchmarkPageIslands(run.ctx, run.doc, state, root, plan, objectRender, run.log); err != nil {
			return nil, fmt.Errorf("object %d (%s): certified page islands: %w", idx+1, obj.Page, err)
		}

		return state, nil
	}

	lres, err := layout.LayoutContext(run.ctx, root, state.bodyLayoutOpts(objectRender))
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): layout: %w", idx+1, obj.Page, err)
	}

	if run.req.Global.SmartShrinking { //nolint:nestif // sequential width-check/zoom/relayout steps
		contentW := state.geom.contentW
		if contentW2 := measuredWidth(lres); contentW2 > contentW+smartShrinkMinOverflow {
			// Smart shrinking: scale-to-width re-layout. The layout engine
			// scales everything by Options.Zoom; the page geometry is
			// unchanged, so the content fits the content area. A user
			// zoom factor composes multiplicatively.
			zoom := contentW / contentW2
			if zoom > 0 && zoom < 1 {
				line.Emit(run.log, line.Info,
					"object %d (%s): content width %.1fpt exceeds the %.1fpt content area; smart shrinking with zoom %.3f",
					idx+1, obj.Page, contentW2, contentW, zoom)

				effZoom := zoom
				if zf := obj.Load.ZoomFactor; zf > 0 {
					effZoom = zoom * zf
				}

				objectRender.zoom = effZoom
				lres, err = layout.LayoutContext(run.ctx, root, state.bodyLayoutOpts(objectRender))
				if err != nil {
					return nil, fmt.Errorf("object %d (%s): smart-shrink layout: %w", idx+1, obj.Page, err)
				}
			}
		}
	}

	if run.req.Global.ResolveRelativeLinks {
		resolveRelativeLinkURIs(lres.Ops, state.base)
	}

	// --no-external-links strips URI link ops before painting (the object
	// flag is the CLI's --external-links target; it defaults on).
	if !obj.ExternalLinks {
		lres.Ops = stripLinkURIs(lres.Ops)
	}

	before := run.doc.PageCount()

	if err := layout.PaintContext(run.ctx, run.doc, lres, paintOptions(state.geom)); err != nil {
		return nil, fmt.Errorf("object %d (%s): paint: %w", idx+1, obj.Page, err)
	}

	state.pages = run.doc.PageCount() - before
	state.offset = before
	state.headings = collectObjectHeadings(root, lres, before, run.req.Global, *obj, run.log)
	state.navigation = collectBodyNavigation(lres)

	return state, nil
}

// bodyLayoutOpts builds layout.Options for a body (or smart-shrink) pass
// from the object's resolved geometry and shared render knobs.
type objectRenderContext struct {
	global             settings.PdfGlobal
	obj                *settings.PdfObject
	font               *pdf.Font
	registry           *pdf.Registry
	sheets             []*css.Stylesheet
	zoom               float64
	imagesFn           func(string) ([]byte, error)
	printLinkUnderline bool
}

func (st *objectState) bodyLayoutOpts(render objectRenderContext) layout.Options {
	media := st.media
	if media == "" {
		media = mediaPrint
	}

	return layout.Options{ //nolint:exhaustruct // intentional zero-value fields
		Width:              st.geom.contentW,
		Height:             st.geom.contentH,
		Font:               render.font,
		Registry:           render.registry,
		Sheets:             render.sheets,
		Media:              media,
		Zoom:               render.zoom,
		Images:             render.imagesFn,
		Background:         render.global.Background,
		PrintLinkUnderline: render.printLinkUnderline,
	}
}

// mergedReplaces merges the --replace maps of the global and object header
// and footer settings. The CLI stores --replace on the header only; merging
// all four surfaces keeps footer --replace working for library users.
func mergedReplaces(obj *settings.PdfObject, glob settings.PdfGlobal) map[string]string {
	out := map[string]string{}
	for k, v := range glob.Header.Replace {
		out[k] = v
	}

	for k, v := range obj.Header.Replace {
		out[k] = v
	}

	for k, v := range glob.Footer.Replace {
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
	width := res.Width

	for _, op := range res.Ops {
		switch op.Kind {
		case layout.OpFillRect, layout.OpStrokeRect, layout.OpImage:
			if ext := op.X + op.W; ext > width {
				width = ext
			}
		case layout.OpLine, layout.OpText, layout.OpLinkURI, layout.OpBullet:
			// Text and link ops never force a page wider; ignore.
			continue
		}
	}

	return width
}

// pageGeometry resolves the page size in points from the single size model:
// Size.Width/Height (mm) override a named PageSize / Size.PageSize.
// Landscape swaps the pair. Legacy PageWidth/PageHeight fields are gone.
func pageGeometry(glob settings.PdfGlobal) (float64, float64, error) {
	var width, height float64

	if glob.Size.Width > 0 && glob.Size.Height > 0 {
		width, height = glob.Size.Width*mmToPt, glob.Size.Height*mmToPt
	} else {
		name := glob.PageSize
		if name == "" {
			name = glob.Size.PageSize
		}

		var err error

		width, height, err = settings.ParsePageSize(name)
		if err != nil {
			return 0, 0, fmt.Errorf("parse page size %q: %w", name, err)
		}
	}

	if glob.Orientation == settings.OrientationLandscape {
		width, height = height, width
	}

	return width, height, nil
}

// mediaFor resolves layout CSS media for PDF mode via settings.ResolveMedia.
// Object media lives on Load (CLI --media-type / print-media-type object flags);
// it is projected onto a temporary Web for the shared resolver. PDF default is "print".
func mediaFor(glob settings.PdfGlobal, obj *settings.PdfObject) string {
	var objWeb *settings.Web

	if obj != nil {
		w := settings.Web{ //nolint:exhaustruct // intentional zero-value fields
			PrintMediaType: obj.Load.PrintMediaType,
			MediaType:      obj.Load.MediaType,
		}
		objWeb = &w
	}

	return settings.ResolveMedia(mediaPrint, glob.Web, objWeb)
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
func CollectSheets(
	ctx context.Context, loader *load.Loader, root *html.Node, base string,
	loadPage settings.LoadPage, opts SheetOptions, log io.Writer,
) []*css.Stylesheet {
	if loader == nil {
		return nil
	}

	resources := loader.ForResource(&load.Resource{Base: base}, loadPage)
	sheets, err := collectSheets(ctx, resources, root, opts, log)
	if err != nil && log != nil {
		line.Emit(log, line.Warn, "stylesheet collection: %v", err)
	}

	return sheets
}

type sheetCollector struct {
	ctx       context.Context
	resources load.ResourceContext
	opts      SheetOptions
	log       io.Writer
	sheets    []*css.Stylesheet
	rules     int
	err       error
}

func collectSheets(
	ctx context.Context, resources load.ResourceContext, root *html.Node,
	opts SheetOptions, log io.Writer,
) ([]*css.Stylesheet, error) {
	collector := sheetCollector{ctx: ctx, resources: resources, opts: opts, log: log}
	if root != nil {
		root.Walk(collector.visit)
	}

	const softRuleWarn = 25000
	if collector.rules >= softRuleWarn {
		collector.warn("large stylesheet volume (%d rules); print may be slow", collector.rules)
	}
	if collector.err != nil {
		return collector.sheets, collector.err
	}

	return collector.sheets, nil
}

func (collector *sheetCollector) visit(node *html.Node) {
	if collector.err != nil || node.Type != html.ElementNode {
		return
	}

	switch node.Name {
	case "style":
		collector.collectStyle(node)
	case "link":
		collector.collectLink(node)
	}
}

func (collector *sheetCollector) collectStyle(node *html.Node) {
	sheet, err := css.Parse(styleText(node))
	if err != nil {
		collector.warn("skipping <style>: %v", err)
		return
	}

	collector.add(sheet)
}

func (collector *sheetCollector) collectLink(node *html.Node) {
	if !linkStylesheet(node, collector.opts.ViewportW, collector.opts.ViewportH, collector.opts.MediaType) {
		return
	}

	href := node.Attribute("href")
	resource, err := collector.resources.Fetch(collector.ctx, href)
	if err != nil {
		collector.warn("skipping <link href=%q>: %v", href, err)
		return
	}

	sheet, err := css.Parse(string(resource.Body))
	if err != nil {
		collector.warn("skipping <link href=%q>: %v", href, err)
		return
	}

	collector.add(sheet)
}

func (collector *sheetCollector) add(sheet *css.Stylesheet) {
	if sheet == nil {
		return
	}

	collector.rules += len(sheet.Rules)
	if collector.rules > maxStylesheetRules {
		collector.err = fmt.Errorf("%w: got %d, limit %d", errTooManyStyles, collector.rules, maxStylesheetRules)
		return
	}

	collector.sheets = append(collector.sheets, sheet)
}

func (collector *sheetCollector) warn(format string, args ...any) {
	if collector.log == nil {
		return
	}

	if collector.opts.ObjectIndex > 0 {
		line.Emit(
			collector.log,
			line.Warn,
			"object %d: "+format,
			append([]any{collector.opts.ObjectIndex}, args...)...,
		)
		return
	}

	line.Emit(collector.log, line.Warn, format, args...)
}

// styleText concatenates the raw text of a <style> element.
func styleText(n *html.Node) string {
	var strB strings.Builder

	for _, c := range n.Children {
		if c.Type == html.TextNode {
			strB.WriteString(c.Text)
		}
	}

	return strB.String()
}

// linkStylesheet reports whether n is a stylesheet <link> whose media
// attribute matches the conversion mediaType (empty, all, print/screen, or
// feature queries that MediaMatches accepts for that type).
func linkStylesheet(num *html.Node, viewportW, viewportH float64, mediaType string) bool {
	if num.Name != "link" || !strings.Contains(strings.ToLower(num.Attribute("rel")), "stylesheet") {
		return false
	}

	if num.Attribute("href") == "" {
		return false
	}

	media := num.Attribute("media")
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

	bufU, err := url.Parse(base)
	if err != nil || bufU == nil {
		return
	}

	for idx := range ops {
		if newURI, ok := resolveRelativeLinkURI(ops[idx], bufU); ok {
			ops[idx].URI = newURI
		}
	}
}

// resolveRelativeLinkURI rewrites one OpLinkURI against base, reporting
// whether the op's URI should be replaced. Fragments (#id), scheme URLs,
// mailto links and unparsable references are left unchanged.
func resolveRelativeLinkURI(op layout.Op, base *url.URL) (string, bool) {
	if op.Kind != layout.OpLinkURI || op.URI == "" {
		return "", false
	}

	u := op.URI
	if strings.HasPrefix(u, "#") || strings.Contains(u, "://") || strings.HasPrefix(strings.ToLower(u), "mailto:") {
		return "", false
	}

	ref, err := url.Parse(u)
	if err != nil {
		return "", false
	}

	return base.ResolveReference(ref).String(), true
}

// loadFontRegistry builds the opt-in font registry from --font-path and
// optional --use-system-fonts. Returns nil when nothing was configured.
func loadFontRegistry(glob settings.PdfGlobal, log io.Writer) *pdf.Registry {
	var dirs []string

	dirs = append(dirs, glob.FontPaths...)
	if glob.UseSystemFonts {
		dirs = append(dirs, pdf.DefaultSystemFontDirs()...)
	}

	if len(dirs) == 0 {
		return nil
	}

	reg := pdf.ScanFontDirs(dirs)

	if log != nil && log != io.Discard && !glob.Quiet {
		line.Emit(log, line.Info, "scanned %d font path(s)", len(dirs))
	}

	return reg
}

// MergeFontFaces loads @font-face url(...) TTF/OTF/WOFF1 sources into the
// registry (local and remote https via FetchSub ACL/timeouts). WOFF2 (.woff2),
// EOT, and data: src are skipped until WOFF2 decode ships. Shared by PDF
// convert and image mode.
func MergeFontFaces(
	ctx context.Context, loader *load.Loader, reg *pdf.Registry, sheets []*css.Stylesheet,
	base string, loadPage settings.LoadPage, idx int, log io.Writer,
) *pdf.Registry {
	if loader == nil {
		return reg
	}

	resources := loader.ForResource(&load.Resource{Base: base}, loadPage)
	return mergeFontFaces(ctx, resources, reg, sheets, idx, log)
}

func mergeFontFaces(
	ctx context.Context, resources load.ResourceContext, reg *pdf.Registry,
	sheets []*css.Stylesheet, idx int, log io.Writer,
) *pdf.Registry {
	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, fontFace := range sheet.FontFaces {
			reg = mergeFontFace(ctx, resources, reg, fontFace, idx, log)
		}
	}

	return reg
}

func mergeFontFace(
	ctx context.Context, resources load.ResourceContext, reg *pdf.Registry,
	fontFace css.FontFace, idx int, log io.Writer,
) *pdf.Registry {
	for _, uri := range css.FontFaceURLs(fontFace.Src) {
		font, ok := fetchFontFace(ctx, resources, uri, idx, log)
		if !ok {
			continue
		}

		if fontFace.Family != "" {
			font.PostScriptName = strings.ReplaceAll(fontFace.Family, " ", "")
		}
		if reg == nil {
			reg = pdf.NewRegistry()
		}

		reg.AddFont(font)
		if fontFace.Family != "" {
			reg.AddFamilyAlias(fontFace.Family, font)
		}
	}

	return reg
}

func fetchFontFace(
	ctx context.Context, resources load.ResourceContext, uri string,
	idx int, log io.Writer,
) (*pdf.Font, bool) {
	low := strings.ToLower(uri)
	if strings.HasSuffix(low, ".woff2") || strings.HasSuffix(low, ".eot") {
		line.Emit(log, line.Warn,
			"object %d: @font-face src %q skipped (WOFF2/EOT unsupported; WOFF1/TTF/OTF only)",
			idx, uri)
		return nil, false
	}
	// data: would bypass the network:// gate; reject so we never ParseTTF
	// untrusted inline payloads from CSS.
	if strings.HasPrefix(low, "data:") {
		line.Emit(log, line.Warn, "object %d: @font-face data: src skipped", idx)
		return nil, false
	}

	resource, err := resources.Fetch(ctx, uri)
	if err != nil {
		line.Emit(log, line.Warn, "object %d: @font-face src %q: %v", idx, uri, err)
		return nil, false
	}

	font, err := pdf.ParseFontBytes(resource.Body)
	if err != nil {
		line.Emit(log, line.Warn, "object %d: @font-face src %q: %v", idx, uri, err)
		return nil, false
	}

	return font, true
}
