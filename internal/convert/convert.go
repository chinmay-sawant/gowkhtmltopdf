package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"gowkhtmltopdf/internal/convert/render"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/errs"
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
// cmd mains (via internal/app) and the library API build it.
type Request struct {
	Global  settings.PdfGlobal
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
	// benchmarkPageIslands is an internal-only performance hook. It is never
	// inferred from HTML content; production and CLI requests always use the
	// generic document renderer. Benchmark tests opt in through the dedicated
	// constructor below.
	benchmarkPageIslands bool
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

// ErrInvalidCopies reports a request with a non-positive copy count.
var ErrInvalidCopies = errors.New("convert: copies must be at least one")

// ErrNoRenderableObjects reports a request that contains no body object that
// can be loaded. Table-of-contents objects are metadata, not renderable page
// input, and an object with neither a page nor inline HTML is empty.
var ErrNoRenderableObjects = settings.ErrNoRenderableObjects

// ErrUnexpectedImageSettings reports an image-mode union member passed to the
// PDF engine. The shared Request remains the compatibility contract, while
// these constructors and validators make each mode's invariant explicit at
// its boundary.
var ErrUnexpectedImageSettings = errors.New("convert: image settings are not valid for PDF")

// ErrMissingImageSettings reports an image request sent through the image
// adapter without its mode-specific settings.
var ErrMissingImageSettings = errors.New("convert: image settings are required")

// errNilRequest reports a nil Request at a method boundary.
var errNilRequest = errs.ErrNilRequest

// errNilContext reports a nil context at the conversion boundary.
var errNilContext = errs.ErrNilContext

// errImagesDisabled reports an image request made while images are disabled.
var errImagesDisabled = errs.ErrImagesDisabled

var (
	errTooManyObjects = errors.New("convert: object limit exceeded")
	errTooManyCopies  = errors.New("convert: copy limit exceeded")
	errTooManyPages   = errors.New("convert: page limit exceeded")
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

// NewBenchmarkPDFRequest builds the explicitly opted-in benchmark request.
// This constructor is intentionally internal (the package itself is under
// internal/) and keeps the benchmark-only page-island optimization separate
// from normal HTML rendering.
func NewBenchmarkPDFRequest(
	global settings.PdfGlobal,
	objects []settings.PdfObject,
	output, outline io.Writer,
) *Request {
	req := NewPDFRequest(global, objects, output, outline)
	req.benchmarkPageIslands = true

	return req
}

// Validate checks the explicit output contract before any loading or font
// initialization occurs. This makes a missing sink deterministic and cheap to
// test through the engine seam.
func (r *Request) Validate() error {
	if r == nil {
		return errNilRequest
	}

	if r.Output == nil {
		return ErrMissingOutput
	}

	if len(r.Objects) > maxConversionObjects {
		return fmt.Errorf("%w: got %d, limit %d", errTooManyObjects, len(r.Objects), maxConversionObjects)
	}

	if r.Global.Copies < 1 {
		return fmt.Errorf("%w: got %d", ErrInvalidCopies, r.Global.Copies)
	}

	if r.Global.Copies > maxConversionCopies {
		return fmt.Errorf("%w: got %d, limit %d", errTooManyCopies, r.Global.Copies, maxConversionCopies)
	}

	if r.Global.DumpOutline && r.OutlineOutput == nil {
		return ErrMissingOutlineOutput
	}

	if err := ValidateRenderableObjects(r.Objects); err != nil {
		return err
	}

	return nil
}

// ValidateRenderableObjects applies the shared input invariant used by both
// PDF and image requests. A request may contain TOC metadata, but it must
// also contain at least one body object with either a non-empty page source
// or inline HTML bytes.
//
//nolint:wrapcheck // delegating alias to shared settings package
func ValidateRenderableObjects(objects []settings.PdfObject) error {
	return settings.ValidateRenderableObjects(objects)
}

// ValidatePDF checks the PDF-specific request invariant before running the
// document pipeline.
func (r *Request) ValidatePDF() error {
	return r.Validate()
}

// runContext owns the dependencies for one conversion lifecycle.
type runContext struct {
	req      *Request
	loader   *load.Loader
	font     *pdf.Font
	registry *pdf.Registry
	doc      *pdf.Document
	log      io.Writer
	progress func(phase string, percent int)
	tocs     []*objectState
	bodies   []*objectState
	headings []*outline.Heading
	tocTotal int
	plan     *pagePlan
	exclude  []css.Selector
}

func (run *runContext) report(phase string, value int) {
	if run.progress != nil {
		run.progress(phase, value)
	}

	if run.log != nil && run.log != io.Discard && !run.req.Global.Quiet {
		fmt.Fprintf(run.log, "%s\n", phase)
	}
}

func (run *runContext) renderObjects(ctx context.Context) ([]*objectState, []*objectState, error) {
	var tocs, bodies []*objectState

	count := len(run.req.Objects)

	for idx := range run.req.Objects {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("object %d: %w", idx+1, err)
		}

		run.report(fmt.Sprintf("Loading pages (%d/%d)", idx+1, count), percent(idx+1, count))

		obj := &run.req.Objects[idx]
		if obj.IsTableOfContent {
			state, err := initTOCState(ctx, run, obj, idx)
			if err != nil {
				return nil, nil, err
			}

			tocs = append(tocs, state)

			continue
		}

		state, err := renderObject(ctx, run, obj, idx)
		if err != nil {
			return nil, nil, err
		}

		if state != nil {
			bodies = append(bodies, state)
		}
	}

	return tocs, bodies, nil
}

// Run executes the full PDF conversion pipeline for req. The lifecycle is
// delegated to render.Pipeline; this package supplies the PDF-specific adapter
// and keeps its private state out of the orchestration module.
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
		req:      req,
		loader:   loader,
		font:     font,
		registry: registry,
		doc:      pdf.NewDocument(),
		log:      log,
		progress: progress,
		tocs:     nil,
		bodies:   nil,
		headings: nil,
		tocTotal: 0,
		plan:     nil,
		exclude:  nil,
	}

	if err := render.Run(ctx, &pdfPipeline{run: run}); err != nil {
		return fmt.Errorf("render pipeline: %w", err)
	}

	return nil
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
func initTOCState(ctx context.Context, run *runContext, obj *settings.PdfObject, idx int) (*objectState, error) {
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

	reg, err := effectiveMargins(ctx, run.loader, run.font, run.req.Global, state, run.log)
	if err != nil {
		return nil, fmt.Errorf("object %d: %w", idx+1, err)
	}

	state.registry = reg

	return state, nil
}

// renderObject loads, lays out and paints one body object into doc and
// returns the per-object state the later passes need (nil when the load
// policy skipped the object).
//
//nolint:gocognit,cyclop,funlen // per-object rendering lifecycle
func renderObject(ctx context.Context, run *runContext, obj *settings.PdfObject, idx int) (*objectState, error) {
	geom, err := newHFGeom(run.req.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	media := mediaFor(run.req.Global, obj)

	prep, err := PrepareDocument(ctx, run.loader, obj.Page, obj.Load, run.registry, PrepareOptions{
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
	resources := prep.Resources
	sheets := prep.Sheets
	geom = applyCSSPageMargins(geom, sheets)

	imagesFn := func(src string) ([]byte, error) {
		if !run.req.Global.Web.Images {
			return nil, errImagesDisabled
		}

		r, err := resources.Fetch(ctx, src)
		if err != nil {
			return nil, fmt.Errorf("fetch image resource %q: %w", src, err)
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
		resources:     resources,
		imagesEnabled: run.req.Global.Web.Images,
		media:         media,
		geom:          geom,
		imagesFn:      imagesFn,
		doctitle:      docTitle(root),
	}

	reg, err := effectiveMargins(ctx, run.loader, run.font, run.req.Global, state, run.log)
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

	if run.req.benchmarkPageIslands {
		if plan, ok := benchmarkPageIslandPlan(root); ok {
			if err := renderBenchmarkPageIslands(ctx, run.doc, state, root, plan, objectRender, run.log); err != nil {
				return nil, fmt.Errorf("object %d (%s): certified page islands: %w", idx+1, obj.Page, err)
			}

			return state, nil
		}
	}

	lres, err := layout.LayoutContext(ctx, root, state.bodyLayoutOpts(objectRender))
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

				lres, err = layout.LayoutContext(ctx, root, state.bodyLayoutOpts(objectRender))
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

	if err := layout.PaintContext(ctx, run.doc, lres, paintOptions(state.geom)); err != nil {
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
