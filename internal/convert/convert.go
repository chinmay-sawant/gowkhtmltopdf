package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/render"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/outline"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdfprofile"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
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
	maxConversionCopies  = render.MaxCopies
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
var ErrInvalidCopies = render.ErrInvalidCopies

// ErrNoRenderableObjects reports a request that contains no body object that
// can be loaded. Table-of-contents objects are metadata, not renderable page
// input, and an object with neither a page nor inline HTML is empty.
var ErrNoRenderableObjects = settings.ErrNoRenderableObjects

// errNilRequest reports a nil Request at a method boundary.
// Primary definition (migrated from internal/errs, PT-GO-28).
var errNilRequest = errors.New("gowkhtmltopdf: nil request")

// errNilContext reports a nil context at the conversion boundary.
var errNilContext = errs.ErrNilContext

// errImagesDisabled reports an image request made while images are disabled.
// Primary definition (migrated from internal/errs, PT-GO-28).
var errImagesDisabled = errors.New("gowkhtmltopdf: images disabled")

var (
	errTooManyObjects = errors.New("convert: object limit exceeded")
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

	if err := render.ValidateCopies(r.Global.Copies); err != nil {
		return fmt.Errorf("copies: %w", err)
	}

	if r.Global.DumpOutline && r.OutlineOutput == nil {
		return ErrMissingOutlineOutput
	}

	if err := ValidateRenderableObjects(r.Objects); err != nil {
		return err
	}

	if _, err := PolicyForGlobal(r.Global); err != nil {
		return err
	}

	return nil
}

const (
	pdfVersion14 = "1.4"
	pdfVersion17 = "1.7"
	pdfVersion20 = "2.0"
)

// ErrProfileRequiresPDF17 indicates a 1.7-era compliance profile was requested
// with a PDF version other than 1.7 (1.4 or 2.0).
var ErrProfileRequiresPDF17 = pdf.ErrConformanceRequiresPDF17

// ErrProfileRequiresPDF20 indicates a 2.0-era compliance profile was requested
// with a PDF version other than 2.0 (1.4 or 1.7).
var ErrProfileRequiresPDF20 = pdf.ErrConformanceRequiresPDF20

// policyForProfile builds a WriterPolicy for a known compliance profile,
// rejecting explicit version strings that conflict with the profile base.
func policyForProfile(glob settings.PdfGlobal, canonicalProfile string) (pdf.WriterPolicy, error) {
	switch {
	case pdfprofile.IsPDFA3(canonicalProfile) || pdfprofile.IsPDFUA1(canonicalProfile):
		return compliancePolicy(
			glob, canonicalProfile, pdf.PDF17, pdfVersion17, pdf.ErrConformanceRequiresPDF17,
			pdfVersion14, pdfVersion20,
		)
	case pdfprofile.IsPDFA4(canonicalProfile) || pdfprofile.IsPDFUA2(canonicalProfile):
		return compliancePolicy(
			glob, canonicalProfile, pdf.PDF20, pdfVersion20, pdf.ErrConformanceRequiresPDF20,
			pdfVersion14, pdfVersion17,
		)
	default:
		return pdf.WriterPolicy{}, settings.ErrInvalidPDFProfile
	}
}

// compliancePolicy validates version conflict, builds the policy, and runs Validate.
func compliancePolicy(
	glob settings.PdfGlobal,
	canonicalProfile string,
	version pdf.PDFVersion,
	_ string,
	conflictErr error,
	forbiddenVersions ...string,
) (pdf.WriterPolicy, error) {
	if glob.PdfVersion != "" {
		parsedVersion, err := settings.ParsePDFVersion(glob.PdfVersion)
		if err != nil {
			return pdf.WriterPolicy{}, err //nolint:wrapcheck // sentinel error from settings
		}

		for _, forbidden := range forbiddenVersions {
			if parsedVersion == forbidden {
				return pdf.WriterPolicy{}, conflictErr
			}
		}
	}

	policy := pdf.WriterPolicy{ //nolint:exhaustruct // default feature flags
		Version:            version,
		ConformanceProfile: canonicalProfile,
	}
	if err := policy.Validate(); err != nil {
		return pdf.WriterPolicy{}, err //nolint:wrapcheck // delegating policy validation
	}

	return policy, nil
}

// PolicyForGlobal maps the requested PDF version and profile to a pdf.WriterPolicy.
// When PdfProfile is set, PDF 1.7 or PDF 2.0 is implied based on the profile unless explicitly conflicting.
// An empty version with no profile defaults to PDF 1.4.
func PolicyForGlobal(glob settings.PdfGlobal) (pdf.WriterPolicy, error) {
	if glob.PdfProfile != "" {
		canonicalProfile, err := settings.ParsePDFProfile(glob.PdfProfile)
		if err != nil {
			return pdf.WriterPolicy{}, err //nolint:wrapcheck // sentinel error from settings
		}

		return policyForProfile(glob, canonicalProfile)
	}

	version, err := settings.ParsePDFVersion(glob.PdfVersion)
	if err != nil {
		return pdf.WriterPolicy{}, err //nolint:wrapcheck // zero policy on error
	}

	switch version {
	case pdfVersion14:
		return pdf.WriterPolicy{Version: pdf.PDF14}, nil //nolint:exhaustruct // default feature flags
	case pdfVersion17:
		return pdf.WriterPolicy{Version: pdf.PDF17}, nil //nolint:exhaustruct // default feature flags
	case pdfVersion20:
		return pdf.WriterPolicy{Version: pdf.PDF20}, nil //nolint:exhaustruct // default feature flags
	default:
		return pdf.WriterPolicy{},
			fmt.Errorf("%w: %q", settings.ErrInvalidPDFVersion, version)
	}
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

// runContext owns the dependencies for one conversion lifecycle. It is the
// wiring that implements the narrow stage interfaces (objectRenderer,
// hfLoader); stage functions take the interface they need instead of the fat
// struct.
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

// objectRenderer is the narrow contract the object-loading stage needs: one
// body object and one TOC object, each producing its objectState. runContext
// implements it; the stage takes the interface so tests can drive the
// pipeline with a fake renderer instead of a full run.
type objectRenderer interface {
	renderObject(ctx context.Context, obj *settings.PdfObject, idx int) (*objectState, error)
	initTOC(ctx context.Context, obj *settings.PdfObject, idx int) (*objectState, error)
}

// hfLoader is the narrow contract the header/footer pass needs: the default
// fallback font for text bands plus lazy loading of one HTML band template.
// runContext implements it.
type hfLoader interface {
	defaultFont() *pdf.Font
	loadHF(ctx context.Context, state *objectState, rawOrURL string) (*htmlHFLayout, *pdf.Registry, error)
}

var (
	_ objectRenderer = (*runContext)(nil)
	_ hfLoader       = (*runContext)(nil)
)

// defaultFont returns the default face used by text header/footer bands
// (the hfLoader contract; see loadHTMLHF/drawTextHF).
func (run *runContext) defaultFont() *pdf.Font { return run.font }

// loadHF lazily loads one HTML header/footer band template with the run's
// loader, fallback font and log (the hfLoader contract).
func (run *runContext) loadHF(
	ctx context.Context, state *objectState, rawOrURL string,
) (*htmlHFLayout, *pdf.Registry, error) {
	return loadHTMLHF(ctx, run.loader, run.font, state, rawOrURL, run.log)
}

func (run *runContext) report(phase string, value int) {
	if run.progress != nil {
		run.progress(phase, value)
	}

	if run.log != nil && run.log != io.Discard && !run.req.Global.Quiet {
		fmt.Fprintf(run.log, "%s\n", phase)
	}
}

// renderObjects drives the per-object loop using only the narrow
// objectRenderer contract plus the objects slice and report callback.
// It exists as a standalone stage so tests can supply a fake renderer
// without constructing a full runContext (which owns 15 fields). The
// runContext method below wires the real run into this narrow stage.
func renderObjects(
	ctx context.Context,
	renderer objectRenderer,
	objects []settings.PdfObject,
	report func(string, int),
) ([]*objectState, []*objectState, error) {
	var tocs, bodies []*objectState

	count := len(objects)

	for idx := range objects {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("object %d: %w", idx+1, err)
		}

		if report != nil {
			report(fmt.Sprintf("Loading pages (%d/%d)", idx+1, count), percent(idx+1, count))
		}

		obj := &objects[idx]
		if obj.IsTableOfContent {
			state, err := renderer.initTOC(ctx, obj, idx)
			if err != nil {
				return nil, nil, err
			}

			tocs = append(tocs, state)

			continue
		}

		state, err := renderer.renderObject(ctx, obj, idx)
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
//
// The caller owns the overall conversion timeout: Run never imposes one, so
// ctx should carry a deadline (or be cancellable) when a bounded run is
// required. Cancellation is honored between stages and inside long passes;
// HTTP fetches are additionally bounded per request by the loader's
// LoadPage.Timeout policy (load.DefaultResponseTimeout when unset).
func Run(ctx context.Context, req *Request, log io.Writer, progress func(phase string, percent int)) error {
	if err := req.Validate(); err != nil {
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

	policy, err := PolicyForGlobal(req.Global)
	if err != nil {
		return fmt.Errorf("pdf policy: %w", err)
	}

	doc, err := pdf.NewDocumentWithPolicy(policy)
	if err != nil {
		return fmt.Errorf("initialize document: %w", err)
	}

	registry := pdf.RegistryFromGlobal(req.Global)
	pdf.LogFontRegistryScan(req.Global, log)
	run := &runContext{
		req:      req,
		loader:   loader,
		font:     font,
		registry: registry,
		doc:      doc,
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

// initTOC builds the per-object state of a table-of-contents object:
// geometry (with auto margins resolved) and the effective TOC settings.
// It is the TOC half of the objectRenderer contract.
func (run *runContext) initTOC(ctx context.Context, obj *settings.PdfObject, idx int) (*objectState, error) {
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
// policy skipped the object). It is the body half of the objectRenderer
// contract.
//
//nolint:cyclop,funlen,wsl // per-object rendering lifecycle
func (run *runContext) renderObject(ctx context.Context, obj *settings.PdfObject, idx int) (*objectState, error) {
	geom, err := newHFGeom(run.req.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	media := mediaFor(run.req.Global, obj)

	prep, err := prepare.Document(
		ctx,
		run.loader,
		obj.Page,
		obj.Load,
		run.registry,
		prepare.BuildOptions(
			geom.contentW,
			geom.contentH,
			media,
			idx+1,
			run.req.Global.Web,
			obj.Web,
		),
		run.log,
	)
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
	header, footer := applyPageMarginBoxes(obj.HeaderFor(run.req.Global), obj.FooterFor(run.req.Global), geom.pageBoxes)
	state := &objectState{ //nolint:exhaustruct // intentional zero-value fields
		obj:           obj,
		idx:           idx,
		header:        header,
		footer:        footer,
		repl:          mergedReplaces(obj, run.req.Global),
		base:          prep.Resource.Base,
		lp:            obj.Load,
		registry:      registry,
		resources:     resources,
		imagesEnabled: run.req.Global.Web.Images,
		media:         media,
		geom:          geom,
		imagesFn:      imagesFn,
		doctitle:      docTitle(root),
	}

	if run.doc.Policy().IsPDFUA1() || run.doc.Policy().IsPDFUA2() {
		if l := docLang(root); l != "" {
			run.doc.SetLanguage(l)
		}
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

	lres, objectRender, err := layoutBody(
		ctx,
		state,
		objectRender,
		run.log,
		func(options layout.Options) (*layout.Result, error) {
			return layout.LayoutContext(ctx, root, options)
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}
	if err := applyBodyPolicies(ctx, state, objectRender, lres); err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}

	before := run.doc.PageCount()

	if err := layout.PaintContext(ctx, run.doc, lres, paintOptions(state.geom)); err != nil {
		return nil, fmt.Errorf("object %d (%s): paint: %w", idx+1, obj.Page, err)
	}

	state.geom.pageNames = layout.PageNames(lres, state.geom.contentH)
	state.pages = run.doc.PageCount() - before
	state.offset = before
	state.headings = collectObjectHeadings(root, lres, *obj)
	state.navigation = collectBodyNavigation(lres)

	return state, nil
}

type layoutBodyFunc func(layout.Options) (*layout.Result, error)

type layoutReleaseFunc func(*layout.Result)

//nolint:cyclop,wsl // smart-shrink is one bounded second-pass policy.
func layoutBody(
	ctx context.Context,
	state *objectState,
	render objectRenderContext,
	log io.Writer,
	layoutFn layoutBodyFunc,
	release layoutReleaseFunc,
) (*layout.Result, objectRenderContext, error) {
	if err := ctx.Err(); err != nil {
		return nil, render, fmt.Errorf("layout: %w", err)
	}

	result, err := layoutFn(state.bodyLayoutOpts(render))
	if err != nil {
		return nil, render, fmt.Errorf("layout: %w", err)
	}

	if !render.global.SmartShrinking {
		return result, render, nil
	}

	contentW := state.geom.contentW
	contentW2 := measuredWidth(result)
	if contentW2 <= contentW+smartShrinkMinOverflow {
		return result, render, nil
	}

	zoom := contentW / contentW2
	if zoom <= 0 || zoom >= 1 {
		return result, render, nil
	}

	line.Emit(log, line.Info,
		"object %d (%s): content width %.1fpt exceeds the %.1fpt content area; smart shrinking with zoom %.3f",
		state.idx+1, state.obj.Page, contentW2, contentW, zoom)

	effZoom := zoom
	if zoomFactor := state.obj.Load.ZoomFactor; zoomFactor > 0 {
		effZoom = zoom * zoomFactor
	}
	render.zoom = effZoom
	if release != nil {
		release(result)
	}
	if err := ctx.Err(); err != nil {
		return nil, render, fmt.Errorf("smart-shrink layout: %w", err)
	}

	result, err = layoutFn(state.bodyLayoutOpts(render))
	if err != nil {
		return nil, render, fmt.Errorf("smart-shrink layout: %w", err)
	}

	return result, render, nil
}

func applyBodyPolicies(
	ctx context.Context,
	state *objectState,
	render objectRenderContext,
	result *layout.Result,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("body policy: %w", err)
	}

	if render.global.ResolveRelativeLinks {
		resolveRelativeLinkURIs(result.Ops, state.base)
	}

	// --no-external-links strips URI link ops before painting (the object
	// flag is the CLI's --external-links target; it defaults on).
	if !state.obj.ExternalLinks {
		result.Ops = stripLinkURIs(result.Ops)
	}

	return nil
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
