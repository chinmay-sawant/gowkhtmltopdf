// Package layout turns the parsed HTML tree plus resolved styles into a
// display list: absolute-positioned drawing operations in a continuous
// canvas (y grows downward from the top of the page content area). Painting
// into a pdf.Document is done by Paint (paint.go).
//
// Report-engine scope: block and inline flow, margin collapsing between
// siblings, tables (separate borders, colspan), images, lists, text wrapping
// with the embedded Liberation Sans font, float lite (left/right + clear),
// real inline-block, box-sizing, position relative/absolute/fixed lite,
// print-scoped sticky (page content box = scrollport; overflow boxes at
// scroll offset 0 — see sticky.go), and a
// partial flex (row/column) subset, CSS grid lite (tracks, gap, column span),
// and CSS multi-column lite (column-count/width/gap/span/fill).
package layout

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	htmlTbody    = "tbody"
	htmlThead    = "thead"
	htmlCaption  = "caption"
	htmlColgroup = "colgroup"
	htmlTfoot    = "tfoot"
)

// CSS keyword constants shared by the layout engine. Kept here so repeated
// string literals resolve through one named value (goconst).
const (
	positionAbsolute        = "absolute"
	positionFixed           = "fixed"
	positionRelative        = "relative"
	positionStatic          = "static"
	positionSticky          = "sticky"
	htmlMeter               = "meter"
	displayBlock            = "block"
	displayFlex             = "flex"
	displayGrid             = "grid"
	displayInlineFlex       = "inline-flex"
	displayInlineGrid       = "inline-grid"
	displaySubgrid          = "subgrid"
	displayFlowRoot         = "flow-root"
	displayTable            = "table"
	displayTableCell        = "table-cell"
	displayTableCaption     = "table-caption"
	displayTableRow         = "table-row"
	displayRowGroup         = "table-row-group"
	displayHeaderGroup      = "table-header-group"
	displayFooterGroup      = "table-footer-group"
	displayListItem         = "list-item"
	listStyleDisc           = "disc"
	listStyleSquare         = "square"
	listStyleDecimal        = "decimal"
	listPosInside           = "inside"
	listPosOutside          = "outside"
	htmlSection             = "section"
	pseudoBefore            = "before"
	pseudoAfter             = "after"
	bulletDisc              = "\u2022"
	borderCollapseValue     = "collapse"
	overflowWrapAnywhere    = "anywhere"
	overflowWrapBreakWord   = "break-word"
	borderStyleDashed       = "dashed"
	borderStyleDotted       = "dotted"
	textTransformNone       = "none"
	textTransformUppercase  = "uppercase"
	textTransformLowercase  = "lowercase"
	textTransformCapitalize = "capitalize"
	tableCellKind           = "cell"
)

// Options controls a Layout run.
type Options struct {
	Width      float64 // viewport/content width in points
	Height     float64 // viewport height in points (for % heights)
	Font       *pdf.Font
	Faces      *pdf.FaceSet  // optional Liberation family; defaults loaded when nil
	Registry   *pdf.Registry // optional discovered fonts (--font-path)
	Sheets     []*css.Stylesheet
	Media      string // "print" or "screen"; "" = apply "all" rules only
	Images     func(src string) ([]byte, error)
	Background bool    // paint background colors
	DebugBoxes bool    // outline every box for test/golden output
	Zoom       float64 // zoom factor; style lengths are scaled by it (any positive value, < 1 shrinks)
	// PrintLinkUnderline is an opt-in operator policy (--print-link-underline):
	// after cascade, force text-decoration:underline on a[href]. Default off
	// so author CSS (including inherit → none) is honored.
	PrintLinkUnderline bool
}

// Result is a display list plus the canvas bounds.
type Result struct {
	Ops    []Op
	Width  float64
	Height float64

	root  *box   // element box tree, kept for Paint (Locations)
	boxes []*box // flattened box list for repeated pagination updates

	// flowPages indexes non-fixed operations by their current canvas page while
	// pagination is settling. It is rebuilt when the display list is rewritten
	// and updated in place by shiftFlowY so flow shifts do not rescan operations
	// that are before the affected page.
	flowPages    [][]int
	flowPageOf   []int
	flowPos      []int
	flowPageSize float64
	flowBoxes    [][]int
	flowBoxPage  []int
	flowBoxPos   []int

	// Pages maps page index → indices into Ops of the ops painted on that
	// page. Filled by Paint using its pagination semantics (an op goes to
	// the page containing its top edge; ops crossing a page boundary move
	// wholly to the next page).
	Pages [][]int
	// Locations lists every element box in document order with the page its
	// first op landed on and its canvas rect. Filled by Paint; boxes without
	// ops use the page of their y position.
	Locations []ElementLocation
}

// CloneResult returns an independent pagination result. Display-list image
// bytes, page/index slices, and the layout-owned box graph are copied so a
// second paint or pagination pass cannot mutate the source result through a
// shallow alias.
func CloneResult(res *Result) *Result {
	if res == nil {
		return nil
	}

	clone := *res
	clone.Ops = cloneOps(res.Ops)
	clone.Pages = cloneIndexPages(res.Pages)
	clone.Locations = append([]ElementLocation(nil), res.Locations...)
	clone.flowPages = cloneIndexPages(res.flowPages)
	clone.flowPageOf = append([]int(nil), res.flowPageOf...)
	clone.flowPos = append([]int(nil), res.flowPos...)
	clone.flowBoxes = cloneIndexPages(res.flowBoxes)
	clone.flowBoxPage = append([]int(nil), res.flowBoxPage...)
	clone.flowBoxPos = append([]int(nil), res.flowBoxPos...)

	boxes := make(map[*box]*box, len(res.boxes))
	clone.root = cloneBoxGraph(res.root, boxes)
	clone.boxes = make([]*box, len(res.boxes))

	for i, source := range res.boxes {
		clone.boxes[i] = cloneBoxGraph(source, boxes)
	}

	return &clone
}

func cloneOps(src []Op) []Op {
	if src == nil {
		return nil
	}

	dst := make([]Op, len(src))
	for i := range src {
		dst[i] = src[i]
		dst[i].Image = append([]byte(nil), src[i].Image...)
	}

	return dst
}

func cloneIndexPages(src [][]int) [][]int {
	if src == nil {
		return nil
	}

	dst := make([][]int, len(src))
	for i := range src {
		dst[i] = append([]int(nil), src[i]...)
	}

	return dst
}

func cloneBoxGraph(src *box, seen map[*box]*box) *box {
	if src == nil {
		return nil
	}

	if clone, ok := seen[src]; ok {
		return clone
	}

	clone := *src
	clone.children = nil
	clone.rows = nil
	clone.stickyPort = nil
	seen[src] = &clone

	for _, child := range src.children {
		clone.children = append(clone.children, cloneBoxGraph(child, seen))
	}

	for _, row := range src.rows {
		clonedRow := make([]*box, 0, len(row))
		for _, cell := range row {
			clonedRow = append(clonedRow, cloneBoxGraph(cell, seen))
		}

		clone.rows = append(clone.rows, clonedRow)
	}

	clone.stickyPort = cloneBoxGraph(src.stickyPort, seen)

	return &clone
}

// Workspace owns reusable display-list storage for sequential internal
// layouts. It is deliberately separate from Result so the established Layout
// and LayoutContext APIs keep their independent-result contract.
//
// A Workspace is not safe for concurrent use. Call Release only after Paint
// and every consumer has copied the result metadata it needs.
type Workspace struct {
	ops []Op
}

// Release returns a painted Result's display-list backing storage to w and
// clears Result references that would otherwise retain its box tree and paint
// indexes. It is a no-op for nil inputs.
func (w *Workspace) Release(res *Result) {
	if w == nil || res == nil {
		return
	}

	w.ops = res.Ops[:0]
	res.Ops = nil
	res.root = nil
	res.boxes = nil
	res.flowPages = nil
	res.flowPageOf = nil
	res.flowPos = nil
	res.flowBoxes = nil
	res.flowBoxPage = nil
	res.flowBoxPos = nil
	res.Pages = nil
	res.Locations = nil
}

// ElementLocation describes where one element box landed after pagination.
// X/Y/W/H are in canvas coordinates (y down from the top of the page content
// area); Page is the page index the box's first op was painted on.
type ElementLocation struct {
	Node *html.Node
	Page int
	X, Y float64
	W, H float64
}

// NodeRef provides the neutral node projection used by outline.
func (loc ElementLocation) NodeRef() *html.Node { return loc.Node }

// PageIndex provides the neutral page projection used by outline.
func (loc ElementLocation) PageIndex() int { return loc.Page }

// Bounds provides the neutral rectangle projection used by outline.
func (loc ElementLocation) Bounds() (float64, float64, float64, float64) {
	return loc.X, loc.Y, loc.W, loc.H
}

// OpKind discriminates display-list operations.
type OpKind int

const (
	OpFillRect OpKind = iota
	OpStrokeRect
	OpLine
	OpText
	OpImage
	OpLinkURI
	OpBullet
)

// Stroke masks are used only by rounded border display-list operations.
const (
	StrokeMaskTop uint8 = 1 << iota
	StrokeMaskRight
	StrokeMaskBottom
	StrokeMaskLeft
)

// Op is one display-list operation. Coordinates are in canvas points; for
// OpText and OpBullet, Y is the baseline.
type Op struct {
	// ID is the stable logical identity of the operation. Pagination may split
	// one operation into several fragments; fragments retain this ID so
	// location/range ownership can be remapped without treating a fragment as
	// a new document operation.
	ID uint64

	Kind    OpKind
	X, Y    float64
	W, H    float64
	R, G, B float64 // 0..1
	Alpha   float64
	Width   float64 // stroke width for OpLine
	// StrokeMask selects sides for a rounded OpStrokeRect. Zero means the
	// complete rounded rectangle; non-zero masks are used for mixed CSS
	// borders whose accented side must retain its corner arcs.
	StrokeMask uint8

	Text string
	Font *pdf.Font
	Size float64
	// LetterSpacing is the CSS letter-spacing value in points for text paint.
	LetterSpacing float64
	// TextTransform is applied when the text operation is painted.
	TextTransform string
	Bold          bool

	URI string

	Image  []byte // PNG or JPEG bytes
	ImgW   int
	ImgH   int
	IsJPEG bool
	Alt    string // Alt text for Figure elements under PDF/UA-1

	// Fixed marks ops from position:fixed boxes; Paint stamps them on every
	// page at viewport-relative coordinates.
	Fixed bool

	// Pinned keeps canvas Y stable under later index-suffix flow shifts
	// (repeated thead clones appended after document ops). Unlike Fixed,
	// pinned ops paint only on their natural page.
	Pinned bool

	// StickyID links display-list ops to a position:sticky box after parent
	// prependChrome shifts op indices (0 = not sticky).
	StickyID int

	// ZIndex paints later (higher) above earlier ops when non-zero or set.
	ZIndex    int
	ZIndexSet bool

	// Positioned marks operations emitted by an absolute/fixed subtree. Within
	// the same z-index band, positioned descendants paint above in-flow text,
	// while their own backgrounds remain below their own content.
	Positioned bool

	// RotateDeg rotates the glyph around its baseline origin (PDF text matrix).
	// Independent of CSS transform CTM (which wraps the whole op via Xform).
	RotateDeg float64
	// InkDescent is the glyph descent below the baseline. H remains the line
	// box height; pagination uses this narrower metric for generated text so a
	// line is not moved merely because its leading crosses a page boundary.
	InkDescent float64

	// Xform is a baked canvas-space CSS 2D transform (identity if unset).
	// Applied at paint via PDF cm (see pdfCTMFromCSS). Sibling flow unaffected.
	Xform    Matrix2D
	XformSet bool

	// PaintOpacity is element opacity (CSS opacity / filter:opacity), 0..1.
	// 0 or unset (≥1) means fully opaque. Nested opacities are multiplied.
	PaintOpacity float64
	// Radius is the uniform border radius for rounded fill/stroke rectangles.
	// RadiusY is the vertical radius when corners are elliptical; 0 means ry=rx.
	Radius                                                                 float64
	RadiusTopLeft, RadiusTopRight, RadiusBottomRight, RadiusBottomLeft     float64
	RadiusY                                                                float64
	RadiusTopLeftY, RadiusTopRightY, RadiusBottomRightY, RadiusBottomLeftY float64

	// StructElem is the PDF/UA-1 logical structure element associated with this op.
	StructElem *pdf.StructElem
}

type engine struct {
	opts     Options
	ctx      context.Context //nolint:containedctx // ctx is checked at recursion boundaries (checkContext)
	err      error
	font     *pdf.Font // default/regular face (metrics fallback)
	faces    *pdf.FaceSet
	registry *pdf.Registry
	// styles holds immutable resolved styles per node (from resolveStylesCtx).
	// Transient layout sizes use styleOverrides; callers use stylePtr for
	// shared *ResolvedStyle without a second copy.
	styles         map[*html.Node]*ResolvedStyle
	styleOverrides []styleOverride
	ops            []Op
	noEmit         bool // measurement mode: compute geometry without emitting ops
	height         float64
	scale          float64 // zoom factor applied to style lengths (>= 1)
	zIndex         int
	zIndexSet      bool
	positioned     bool
	stickySeq      int // monotonically increasing sticky box IDs (for Op.StickyID)
	// transformCBDepth counts ancestors with transform≠none; fixed→absolute CB.
	transformCBDepth int
	// imgMaxW > 0 clamps replaced <img> boxes to this containing-block width
	// (table cell / float / inline formatting context).
	imgMaxW float64
	// inlineCBW is the inline formatting context's content width when > 0;
	// used so width:% on inline-blocks resolves against the real CB.
	inlineCBW float64
	// imgCache resolves each <img src> at most once per Layout run (measure
	// and build share the same decode).
	imgCache map[string]*imageRef
	// containers is the last size-container map used for cascade (nil if none);
	// pseudo-content reuses it so @container gates match cascadeRaw.
	containers map[*html.Node]sizeContainer
	// bfcFloats is the floatState of the nearest enclosing BFC. Ordinary
	// blocks reuse it so floats affect later siblings; BFC roots push a
	// fresh state (see pushBFCFloats).
	bfcFloats *floatState
	// bfcStack restores previous BFC float states without allocating a
	// closure per push (table cells establish a BFC each).
	bfcStack []*floatState
	// bfcPool recycles floatState values for pushBFCFloats.
	bfcPool []*floatState
	// absCBHeights carries the containing-block height to deferred absolute
	// children after their in-flow parent has finished determining its size.
	absCBHeights map[*html.Node]float64
	// inlineItemPool recycles temporary inline-item backing arrays. The pool is
	// engine-local because layout is single-threaded and nested inline layout
	// must retain each active caller's slice.
	inlineItemPool [][]inlineItem
	// deferredChrome holds background/border ops to splice in one linear
	// pass (finalizeChrome) for the common non-sticky/non-fixed/non-transform
	// path. Sticky/fixed/transform boxes still splice immediately.
	deferredChrome []chromeEntry
	nextOpID       uint64
	// faceByStyle caches faceFor results for this Layout run (family hash +
	// weight/italic). Avoids repeating registry lookups on every measure call.
	faceByStyle map[faceStyleKey]*pdf.Font
	// faceByRune caches faceForRune fallback results for this Layout run
	// (only glyphs missing from the primary face). Key uses a family hash so
	// lookups allocate no joined family string.
	faceByRune map[faceRuneKey]*pdf.Font
	// needsXformStamp is set when any built box has transform≠none or
	// opacity<1 so stampBoxTransforms can skip the full tree walk.
	needsXformStamp bool
}

// styleOverride temporarily substitutes one node's resolved style while that
// node is built. Overrides are engine-local and form a stack so nested builds
// always observe the most recent override for the same node.
type styleOverride struct {
	node  *html.Node
	style *ResolvedStyle
}

// faceStyleKey is the faceFor cache key for one CSS face identity.
type faceStyleKey struct {
	famHash uint64
	weight  int
	italic  bool
}

// faceRuneKey is the faceForRune fallback cache key for one (style face
// identity, rune). famHash is FNV-1a over FontFamily tokens (no Join alloc).
type faceRuneKey struct {
	famHash uint64
	weight  int
	italic  bool
	r       rune
}

// chromeEntry records one box's background/border ops for insertion before
// the content op at index `at`. Owner is the box that requested chrome so
// chrome-only ranges and geometry shifts can be applied after merge.
type chromeEntry struct {
	at  int
	ops []Op
	b   *box
}

// faceFor selects the TrueType face for a resolved style (bold/italic),
// preferring CSS font-family matches from the opt-in registry, then the
// bundled Liberation FaceSet.
func (e *engine) faceFor(sty ResolvedStyle) *pdf.Font {
	key := faceStyleKey{
		famHash: sty.famHash,
		weight:  sty.FontWeight,
		italic:  sty.FontItalic,
	}
	if e.faceByStyle != nil {
		if f, ok := e.faceByStyle[key]; ok {
			return f
		}
	}

	face := e.lookupFaceFor(sty)

	if e.faceByStyle == nil {
		e.faceByStyle = make(map[faceStyleKey]*pdf.Font)
	}

	e.faceByStyle[key] = face

	return face
}

// lookupFaceFor is the uncached faceFor path.
func (e *engine) lookupFaceFor(sty ResolvedStyle) *pdf.Font {
	if e.registry != nil {
		if f := e.registry.Lookup(sty.FontFamily, sty.FontWeight, sty.FontItalic); f != nil {
			return f
		}
	}

	if e.faces != nil {
		if f := e.faces.ResolveFamily(sty.FontFamily, sty.FontWeight, sty.FontItalic); f != nil {
			return f
		}

		if f := e.faces.Resolve(sty.FontWeight, sty.FontItalic); f != nil {
			return f
		}
	}

	return e.font
}

// faceForRune picks the first CSS font-family face (then defaults) that has a
// glyph for r — browser-like fallback so Hangul/Latin/CJK can come from
// different faces in one run.
//
// Fast path: when the primary face covers r (common for Latin/report text),
// return it without a map lookup. Fallback faces are cached under a hash key
// that does not allocate a joined family string.
func (e *engine) faceForRune(sty ResolvedStyle, runeValue rune) *pdf.Font {
	primary := e.faceFor(sty)
	if isRuneWhitespace(runeValue) {
		return primary
	}

	if primary != nil && primary.GlyphID(runeValue) != 0 {
		return primary
	}

	return e.faceForRuneFallback(sty, runeValue, primary)
}

// faceForRuneFallback resolves and caches a non-primary face for a missing glyph.
func (e *engine) faceForRuneFallback(sty ResolvedStyle, runeValue rune, primary *pdf.Font) *pdf.Font {
	key := faceRuneKey{
		famHash: sty.famHash,
		weight:  sty.FontWeight,
		italic:  sty.FontItalic,
		r:       runeValue,
	}
	if e.faceByRune != nil {
		if f, ok := e.faceByRune[key]; ok {
			return f
		}
	}

	face := e.lookupFaceForRune(sty, runeValue)
	if face == nil {
		face = primary
	}

	if e.faceByRune == nil {
		e.faceByRune = make(map[faceRuneKey]*pdf.Font)
	}

	e.faceByRune[key] = face

	return face
}

// lookupFaceForRune is the uncached face resolution path.
func (e *engine) lookupFaceForRune(sty ResolvedStyle, runeValue rune) *pdf.Font {
	if f := e.registryFamilyWithGlyph(sty, runeValue); f != nil {
		return f
	}

	if f := e.facesWithGlyph(sty, runeValue); f != nil {
		return f
	}

	if e.font != nil && e.font.GlyphID(runeValue) != 0 {
		return e.font
	}
	// Last resort: any opt-in registry face that covers this codepoint
	// (DejaVu/Noto when --font-path / --use-system-fonts scanned them).
	if f := e.registryGlyphFallback(sty, runeValue); f != nil {
		return f
	}

	return e.faceFor(sty)
}

// isRuneWhitespace reports whether r is a rune that inline layout trims.
func isRuneWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// registryGlyphFallback is the last-resort registry lookup: any opt-in face
// covering r, regardless of CSS font-family.
func (e *engine) registryGlyphFallback(st ResolvedStyle, r rune) *pdf.Font {
	if e.registry == nil {
		return nil
	}

	return e.registry.FindWithGlyph(r, st.FontWeight, st.FontItalic)
}

// registryFamilyWithGlyph looks up the first CSS font-family face that has a
// glyph for runeValue.
func (e *engine) registryFamilyWithGlyph(style ResolvedStyle, runeValue rune) *pdf.Font {
	if e.registry == nil {
		return nil
	}

	// Reuse a one-element slice header so Lookup does not allocate per family.
	var one [1]string

	for _, fam := range style.FontFamily {
		one[0] = fam

		f := e.registry.Lookup(one[:], style.FontWeight, style.FontItalic)
		if f != nil && f.GlyphID(runeValue) != 0 {
			return f
		}
	}

	return nil
}

// facesWithGlyph resolves the default faces and returns the first that has a
// glyph for runeValue.
//
//nolint:cyclop,lll // ordered fallback search is intentionally explicit
func (e *engine) facesWithGlyph(style ResolvedStyle, runeValue rune) *pdf.Font {
	if e.faces == nil {
		return nil
	}

	if f := e.faces.ResolveFamily(style.FontFamily, style.FontWeight, style.FontItalic); f != nil && f.GlyphID(runeValue) != 0 {
		return f
	}

	if f := e.faces.Resolve(style.FontWeight, style.FontItalic); f != nil && f.GlyphID(runeValue) != 0 {
		return f
	}

	if style.FontWeight >= fontWeightBold && e.faces.UnicodeFallbackBold != nil && e.faces.UnicodeFallbackBold.GlyphID(runeValue) != 0 {
		return e.faces.UnicodeFallbackBold
	}

	if e.faces.UnicodeFallback != nil && e.faces.UnicodeFallback.GlyphID(runeValue) != 0 {
		return e.faces.UnicodeFallback
	}

	return nil
}

// scalePt applies the engine zoom factor to a style length in points.
func (e *engine) scalePt(v float64) float64 { return v * e.scale }

// zoomScale returns the effective zoom factor: any positive opts.Zoom
// (values below 1 shrink; 0 means no zoom).
func zoomScale(z float64) float64 {
	if z > 0 {
		return z
	}

	return 1
}

func (e *engine) add(paintOp Op) {
	if e.checkContext() || e.noEmit {
		return
	}

	if paintOp.ID == 0 {
		e.nextOpID++
		paintOp.ID = e.nextOpID
	}

	if !e.noEmit {
		paintOp.ZIndex = e.zIndex
		paintOp.ZIndexSet = e.zIndexSet
		paintOp.Positioned = e.positioned
		e.ops = append(e.ops, paintOp)
	}
}

// checkContext is intentionally cheap and is called at recursive layout
// boundaries. Existing Layout callers keep the same behavior through the
// background-context wrapper, while LayoutContext can stop large documents
// during tree construction instead of waiting for painting.
func (e *engine) checkContext() bool {
	if e.err != nil {
		return true
	}

	if e.ctx == nil {
		return false
	}

	if err := e.ctx.Err(); err != nil {
		e.err = err

		return true
	}

	return false
}

func (e *engine) pushZ(style ResolvedStyle) (int, bool, bool) {
	prevZ, prevSet, prevPositioned := e.zIndex, e.zIndexSet, e.positioned

	if style.Position == positionAbsolute || style.Position == positionFixed {
		e.positioned = true
	}

	if style.ZIndexSet {
		e.zIndex = style.ZIndex
		e.zIndexSet = true
	} else if style.HasTransform || style.Opacity < 1 {
		// CSS: transform/opacity create a stacking context (like z-index:0).
		e.zIndex = 0
		e.zIndexSet = true
	}

	if style.HasTransform || style.Opacity < 1 {
		e.needsXformStamp = true
	}

	return prevZ, prevSet, prevPositioned
}

func (e *engine) popZ(prevZ int, prevSet bool, prevPositioned bool) {
	e.zIndex, e.zIndexSet, e.positioned = prevZ, prevSet, prevPositioned
}

// Layout renders the document into a display list.
func Layout(root *html.Node, opts Options) (*Result, error) {
	return LayoutContext(context.Background(), root, opts)
}

// LayoutContext renders the document into a display list and observes ctx at
// style-pass and recursive tree-construction checkpoints. Layout remains the
// compatibility entry point for callers that do not need cancellation.
func LayoutContext(ctx context.Context, //nolint:revive // legacy Layout adapter
	root *html.Node, opts Options,
) (*Result, error) {
	return layoutContext(ctx, root, opts, nil)
}

// WithWorkspace is the sequential internal layout form that
// borrows display-list storage from workspace. Call workspace.Release after
// Paint and metadata projection, before the next layout using it.
func WithWorkspace(ctx context.Context, root *html.Node, opts Options, workspace *Workspace) (*Result, error) {
	return layoutContext(ctx, root, opts, workspace)
}

//nolint:cyclop // layout preflight and staged style/container passes are explicit lifecycle gates.
func layoutContext(
	ctx context.Context,
	root *html.Node, opts Options, workspace *Workspace,
) (*Result, error) {
	if root == nil {
		return nil, errors.New("layout: nil root") //nolint:err113 // static sentinel-free message matches legacy behavior
	}

	if ctx == nil {
		return nil, errs.ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: context: %w", err)
	}

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		return nil, fmt.Errorf("layout: load default faces: %w", err)
	}

	if opts.Faces != nil {
		faces = opts.Faces
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: context: %w", err)
	}

	font := opts.Font
	if font == nil {
		font = faces.Regular
	}

	styles, containers, err := resolveStylesForLayoutContext(ctx, root, opts)
	if err != nil {
		return nil, fmt.Errorf("layout: style resolution: %w", err)
	}

	var ops []Op

	if workspace == nil || cap(workspace.ops) == 0 {
		ops = make([]Op, 0, estimateOpCapacity(root))
	} else {
		ops = workspace.ops[:0]
	}

	return finalizeResult(newEngine(ctx, opts, faces, font, styles, containers, ops), root, opts)
}

// newEngine constructs the layout engine state (extracted from LayoutContext
// for clarity).
func newEngine(
	ctx context.Context, opts Options, faces *pdf.FaceSet, font *pdf.Font,
	styles map[*html.Node]*ResolvedStyle, containers map[*html.Node]sizeContainer, ops []Op,
) *engine {
	return &engine{ //nolint:exhaustruct // intentional zero fields
		opts:       opts,
		ctx:        ctx,
		font:       font,
		faces:      faces,
		registry:   opts.Registry,
		styles:     styles,
		scale:      zoomScale(opts.Zoom),
		containers: containers,
		ops:        ops,
	}
}

// finalizeResult builds the display list and Result from the constructed
// engine (extracted from LayoutContext for clarity).
func finalizeResult(eng *engine, root *html.Node, opts Options) (*Result, error) {
	boxNode := eng.build(root, opts.Width, 0, 0)

	if eng.err != nil {
		return nil, eng.err
	}
	// Merge deferred background/border ops before stamping sticky/fixed flags
	// and CSS transforms (those passes need final op indices).
	eng.finalizeChrome(boxNode)

	boxes := make([]*box, 0)
	flattenBoxes(boxNode, &boxes)

	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Ops:    eng.ops,
		Width:  opts.Width,
		Height: opts.Height,
		root:   boxNode,
		boxes:  boxes,
	}
	if boxNode != nil {
		res.Height = boxNode.y + boxNode.height
	}

	if res.Height < eng.height {
		res.Height = eng.height
	}
	// Paint-time CSS transforms/opacity: stamp composed CTMs after geometry
	// is final so transform-origin % resolves against the border box.
	// Skip the full tree walk when no element had transform/opacity.
	if eng.needsXformStamp {
		stampBoxTransforms(boxNode, IdentityMatrix(), res.Ops)
	}

	return res, nil
}

// resolveStylesForLayout runs the cascade, re-cascading once when @container
// rules match measured size containers (a nested remount covers container-type
// changes). Returns the final styles and the container map used (nil when no
// size containers matched).
func resolveStylesForLayout(
	root *html.Node, opts Options,
) (map[*html.Node]*ResolvedStyle, map[*html.Node]sizeContainer) {
	styles, containers, _ := resolveStylesForLayoutContext(context.Background(), root, opts)

	return styles, containers
}

//nolint:wsl // container remount gates are intentionally kept in lifecycle order.
func resolveStylesForLayoutContext(
	ctx context.Context, root *html.Node, opts Options,
) (map[*html.Node]*ResolvedStyle, map[*html.Node]sizeContainer, error) {
	// Pass 1: cascade without @container (used sizes unknown).
	styles, err := resolveStylesWithContext(ctx, root, opts, nil)
	if err != nil {
		return nil, nil, err
	}

	if !css.HasContainerRules(opts.Sheets) {
		return styles, nil, nil
	}
	// After definite inline sizes of size containers are known, re-cascade so
	// matching @container rules apply, then lay out once with final styles.
	cinfo, err := measureSizeContainersContext(ctx, root, styles, opts.Width)
	if err != nil {
		return nil, nil, err
	}
	if len(cinfo) == 0 {
		return styles, nil, nil
	}

	styles, err = resolveStylesWithContext(ctx, root, opts, cinfo)
	if err != nil {
		return nil, nil, err
	}

	// One nested remount: @container may change nested container-type.
	cinfo2, err := measureSizeContainersContext(ctx, root, styles, opts.Width)
	if err != nil {
		return nil, nil, err
	}
	if len(cinfo2) != len(cinfo) || !sameSizeContainers(cinfo, cinfo2) {
		styles, err = resolveStylesWithContext(ctx, root, opts, cinfo2)
		if err != nil {
			return nil, nil, err
		}

		return styles, cinfo2, nil
	}

	return styles, cinfo, nil
}

// sameSizeContainers reports whether two container measurements agree for
// every node (a changed used inline size / font size forces another pass).
func sameSizeContainers(a, b map[*html.Node]sizeContainer) bool {
	for n, sa := range a {
		sb, ok := b[n]
		if !ok || !sameSizeContainerState(sa, sb) {
			return false
		}
	}

	return true
}

func (e *engine) stylePtr(node *html.Node) *ResolvedStyle {
	for idx := len(e.styleOverrides) - 1; idx >= 0; idx-- {
		override := e.styleOverrides[idx]
		if override.node == node {
			return override.style
		}
	}

	if p := e.styles[node]; p != nil {
		return p
	}

	// Missing node (should not happen for walked trees): stable empty style.
	return &zeroResolvedStyle
}

// styleVal returns a by-value ResolvedStyle for APIs that still take values.
// Prefer field access on stylePtr to avoid the copy when possible.
func (e *engine) styleVal(node *html.Node) ResolvedStyle {
	return *e.stylePtr(node)
}

// buildWithStyle builds node with an engine-local style override. The override
// remains live through the complete recursive build, so boxes created for node
// can safely retain its pointer while descendants continue to read their own
// resolved styles from e.styles.
func (e *engine) buildWithStyle(
	node *html.Node, override *ResolvedStyle, availW, posX, posY float64,
) *box {
	if override == nil {
		return e.build(node, availW, posX, posY)
	}

	e.styleOverrides = append(e.styleOverrides, styleOverride{node: node, style: override})
	defer e.popStyleOverride()

	return e.build(node, availW, posX, posY)
}

func (e *engine) popStyleOverride() {
	e.styleOverrides = e.styleOverrides[:len(e.styleOverrides)-1]
}

func flattenBoxes(boxNode *box, out *[]*box) {
	if boxNode == nil {
		return
	}

	*out = append(*out, boxNode)
	for _, child := range boxNode.children {
		flattenBoxes(child, out)
	}
}

func estimateOpCapacity(root *html.Node) int {
	if root == nil {
		return 0
	}

	nodes := 0

	root.Walk(func(*html.Node) { nodes++ })

	capacity := nodes * opsPerNodeHint / two
	if capacity < minOpsCapacity {
		capacity = 64
	}

	const maxCapacity = 1 << 20
	if capacity > maxCapacity {
		capacity = maxCapacity
	}

	return capacity
}

// box is one laid-out box.
type box struct {
	node *html.Node
	// style points at the shared cascade ResolvedStyle (engine.styles).
	// Using a pointer avoids embedding ~1.3KB of style into every box — table
	// cells dominate box count on multi-page reports.
	style     *ResolvedStyle
	x, y      float64 // border-box top-left
	w, height float64 // border-box size
	kind      string  // "block" | "table" | "cell" | "replaced"
	// opStart/opEnd bound the inclusive range of e.ops indices that this
	// box's subtree emitted. opEnd < opStart means the box emitted nothing
	// (e.g. boxes built during a noEmit measure pass).
	opStart, opEnd int
	children       []*box
	flowIndex      int // transient index in Result.boxes during pagination
	firstBaseline  float64
	// table cells
	col, span         int
	row               int  // owning table row index, set once at placement
	rowSpan           int  // vertical span (default 1) for <td rowspan>
	paginationShifted bool // row was moved by a table pagination fixpoint
	// hasInk is set at cell build time from nodeHasTableInk (any
	// non-whitespace text, br, img, svg, video or canvas in the subtree);
	// row collapse uses the flag instead of re-walking each cell's tree.
	hasInk bool
	// rowBoxH is the height of the cell's starting row only. For rowspan>1,
	// h covers the full span (background/borders) while rowBoxH is what
	// rowsIntact uses so bottom-edge paint ops do not make the first row
	// look multi-page.
	rowBoxH    float64
	contentW   float64 // max-content border-box (preferred column contribution)
	contentMin float64 // min-content border-box (shrink floor)
	contentH   float64
	// rows[i] holds the cell boxes of table row i, in document order. The
	// row's op range is from rows[i][0].opStart to rows[i][len-1].opEnd.
	rows [][]*box
	// headerRows is the number of leading rows that came from <thead> /
	// table-header-group (for repeating headers across pages).
	headerRows int
	// sticky: print-scoped position:sticky (see sticky.go). Insets are scaled
	// points; cb* is filled at pagination time from the parent box.
	// stickyPort is the nearest overflow:auto|scroll|hidden|clip ancestor
	// (scrollport at offset 0); nil means page content box is the scrollport.
	sticky                         bool
	stickyID                       int
	stickyTop, stickyRight         float64
	stickyBottom, stickyLeft       float64
	stickyTopSet, stickyRightSet   bool
	stickyBottomSet, stickyLeftSet bool
	stickyPort                     *box
	cbX, cbY, cbW, cbH             float64
	// replaced image (nil when missing/failed); shared decode via resolveImage.
	img *imageRef
}

func (e *engine) build(node *html.Node, availW, posX, posY float64) *box {
	if e.checkContext() {
		return nil
	}

	if node.Type == html.TextNode {
		return nil
	}

	sty := e.styleVal(node)

	if sty.Display == cssDisplayNone {
		return nil
	}

	prevZ, prevSet, prevPositioned := e.pushZ(sty)
	defer e.popZ(prevZ, prevSet, prevPositioned)
	// Ancestor transforms only (own transform does not change this box's CB).
	underXformCB := e.transformCBDepth > 0
	start := len(e.ops)

	var boxNode *box

	switch node.Name {
	case cssTagImg:
		boxNode = e.buildImage(node, sty, posX, posY, true)
	case "hr":
		boxNode = e.buildHR(node, sty, availW, posX, posY)
	}
	// Out-of-flow positioning wraps the display type so fixed/absolute flex
	// and grid containers still get the right formatting context.
	// Transformed ancestors establish a CB for fixed (treated as absolute).
	if boxNode == nil {
		boxNode = e.buildOutOfFlowIfPositioned(node, sty, availW, posX, posY, underXformCB)
	}
	// Descendants of a transformed box see this as a containing block.
	if sty.HasTransform {
		e.transformCBDepth++
		defer func() { e.transformCBDepth-- }()
	}

	if boxNode == nil {
		boxNode = e.buildInFlowDisplay(node, sty, availW, posX, posY)
	}

	if boxNode != nil {
		boxNode.opStart, boxNode.opEnd = start, len(e.ops)-1
		e.finishBuiltBox(boxNode, sty, underXformCB)
	}

	return boxNode
}

// finishBuiltBox applies position post-processing to a built box: relative
// offset, sticky tagging, and the viewport-fixed op stamp.
func (e *engine) finishBuiltBox(boxNode *box, sty ResolvedStyle, underXformCB bool) {
	if sty.Position == positionRelative {
		e.applyRelativeOffset(boxNode)
	}

	if sty.Position == positionSticky {
		e.tagSticky(boxNode)
	}

	if sty.Position == positionFixed && !underXformCB {
		// Only viewport-fixed when not under a transformed ancestor CB.
		e.markOpsFixed(boxNode.opStart, boxNode.opEnd)
	}
}

// buildOutOfFlowIfPositioned wraps fixed/absolute boxes in an out-of-flow
// formatting context (fixed under a transformed ancestor is absolute).
func (e *engine) buildOutOfFlowIfPositioned(
	node *html.Node, sty ResolvedStyle, availW, posX, posY float64, underXformCB bool,
) *box {
	if sty.Position == positionFixed {
		return e.buildOutOfFlow(node, sty, availW, posX, posY, !underXformCB)
	}

	if sty.Position == positionAbsolute {
		return e.buildOutOfFlow(node, sty, availW, posX, posY, false)
	}

	return nil
}

func isTableDisplay(d string) bool {
	switch d {
	case displayTable, displayTableRow, displayRowGroup, displayHeaderGroup,
		displayFooterGroup, displayTableCell, displayTableCaption:
		return true
	}

	return false
}

// useBlockForTableDisplay reports elements that set display:table only for
// shrink-to-fit (wiki figure thumbs) rather than real tabular structure.
func useBlockForTableDisplay(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}

	if node.Name == displayTable {
		return false
	}
	// Real tables nest tr/tbody/thead/tfoot/colgroup; figures nest a/img/figcaption.
	for _, c := range node.Children {
		if c.Type != html.ElementNode {
			continue
		}

		switch c.Name {
		case "tr", htmlTbody, htmlThead, htmlTfoot, htmlColgroup, "col", htmlCaption:
			return false
		}
	}

	return true
}

// buildBlock lays out a block-level box.
//
//nolint:cyclop // block layout owns ordered CSS flow phases
func (e *engine) buildBlock(node *html.Node, style ResolvedStyle, availW, posX, posY float64) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: displayBlock, x: posX, y: posY,
	}
	w, margL := resolveBlockWidth(e, style, availW)
	boxNode.w = w

	boxNode.x = posX + margL
	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, style)

	// Content ops are recorded first so we know the box height; background
	// and borders are then inserted *before* those ops so paint order is
	// bg → borders → children (otherwise fills cover text).
	contentStart := len(e.ops)

	curY := e.scalePt(style.PaddingTop) + e.scalePt(style.BorderTop.Width)
	enclose := e.pushBFCFloats(style, contentX, contentW)
	children := node.Children
	widget := node.Name == htmlMeter || node.Name == "progress"

	if widget {
		children = nil // fallback text is replaced by the native-style bar
	}

	if node.Name == "details" {
		_, open := node.Attrs["open"]
		if !open {
			children = closedDetailsChildren(node)
		}
	}

	curY = e.flowChildren(boxNode, children, style, contentW, contentX, posY, curY)
	if widget && style.Height < 0 {
		// Native value controls use their intrinsic font-sized control height
		// when auto-sized. Treating them as ordinary text blocks adds the
		// line-height and authored padding a second time, producing the
		// oversized meter/progress tracks in fixture-56.
		curY = e.nativeWidgetAutoContentBottom(style)
	}

	if enclose && e.bfcFloats != nil {
		curY = e.bfcFloats.extentCy(posY, curY)
	}

	e.popBFCFloats(enclose)
	// padding-bottom is inside the border box (space above border-bottom /
	// letterhead rules — fixture-07/16).
	curY += e.scalePt(style.PaddingBottom)
	if isVerticalWritingMode(style.WritingMode) {
		curY = e.verticalWritingHeight(contentStart, curY, style)
	}

	// list marker (outside the principal box content — in the marker area)
	if node.Name == "li" && boxNode.firstBaseline > 0 {
		e.emitListMarker(node, style, contentX, boxNode.firstBaseline)
	}

	boxNode.height = e.applyHeightConstraints(style, curY)
	if widget {
		e.paintValueWidget(node, style, boxNode.x, posY, boxNode.w, boxNode.height)
	}

	e.paintPositionedPseudo(node, style, boxNode, pseudoBefore)
	e.paintPositionedPseudo(node, style, boxNode, pseudoAfter)

	e.prependChrome(contentStart, boxNode, style, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

// nativeWidgetAutoContentBottom returns the content-flow endpoint for an
// auto-sized native value control whose border-box height is one scaled font
// size. Padding and the top border are already part of the flow coordinate;
// the caller adds bottom padding after this endpoint.
func (e *engine) nativeWidgetAutoContentBottom(style ResolvedStyle) float64 {
	targetHeight := e.scalePt(style.FontSize)
	topChrome := e.scalePt(style.BorderTop.Width + style.PaddingTop)
	contentHeight := targetHeight - topChrome - e.scalePt(style.PaddingBottom)

	if contentHeight < 0 {
		contentHeight = 0
	}

	return topChrome + contentHeight
}

// paintPositionedPseudo paints generated content whose used position takes it
// out of the host's normal inline flow. This covers block and flex-item hosts
// such as diagram cards; inline hosts continue through collectInlineSpan.
func (e *engine) paintPositionedPseudo( //nolint:cyclop
	node *html.Node, host ResolvedStyle, boxNode *box, pseudoElem string,
) {
	text := e.pseudoContent(node, pseudoElem)
	if text == "" || boxNode == nil {
		return
	}

	style := e.pseudoStyle(node, pseudoElem, host)
	if style.Position != positionAbsolute && style.Position != positionFixed {
		return
	}

	face := e.faceFor(*style)
	size := style.FontSize * e.scale

	if face == nil || size <= 0 {
		return
	}

	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, host)
	pseudoX := contentX + e.scalePt(style.MarginLeft)

	if !style.LeftAuto {
		pseudoX = contentX + e.scalePt(style.Left)
	} else if !style.RightAuto {
		pseudoX = contentX + contentW - e.measureTextFace(text, *style) - e.scalePt(style.Right)
	}

	staticAfter := pseudoElem == pseudoAfter && style.TopAuto && style.BottomAuto
	pseudoY := boxNode.y + e.scalePt(host.PaddingTop) + e.scalePt(host.BorderTop.Width) +
		e.scalePt(style.MarginTop)

	if staticAfter {
		// An absolutely positioned ::after without an inset uses the static
		// position after its block host, not the host's content origin. This is
		// the baseline position used by diagram connectors between stacked cards.
		pseudoY = boxNode.y + boxNode.height + e.scalePt(style.MarginTop)
	}

	if !style.TopAuto {
		pseudoY = boxNode.y + e.scalePt(style.Top)
	} else if !style.BottomAuto {
		pseudoY = boxNode.y + boxNode.height - e.scalePt(style.Bottom)
	}

	baseline := pseudoY + e.fontAscent(size)

	if staticAfter {
		baseline = pseudoY
	}

	prevZ, prevSet, prevPositioned := e.pushZ(*style)
	e.add(Op{ //nolint:exhaustruct // generated pseudo text has no DOM box
		Kind: OpText, X: pseudoX, Y: baseline, W: e.measureTextFace(text, *style),
		H: style.LineHeight * e.scale, Text: text, Font: face, Size: size,
		InkDescent: e.fontDescentFace(face, size),
		R:          style.Color[0], G: style.Color[1], B: style.Color[2],
		Bold: style.FontWeight >= fontWeightBold,
	})
	e.popZ(prevZ, prevSet, prevPositioned)
}

func (e *engine) verticalWritingHeight(contentStart int, current float64, style ResolvedStyle) float64 {
	textWidth := 0.0
	for _, op := range e.ops[contentStart:] {
		if op.Kind == OpText && op.RotateDeg != 0 && op.W > textWidth {
			textWidth = op.W
		}
	}

	if textWidth == 0 {
		return current
	}

	verticalChrome := e.scalePt(style.PaddingTop) + e.scalePt(style.PaddingBottom) +
		e.scalePt(style.BorderTop.Width) + e.scalePt(style.BorderBottom.Width)

	needed := textWidth + verticalChrome
	if needed > current {
		return needed
	}

	return current
}

func (e *engine) paintValueWidget(node *html.Node, style ResolvedStyle, leftX, topY, width, height float64) {
	minValue, maxValue := 0.0, 1.0
	if node.Name == htmlMeter {
		minValue = widgetNumber(node.Attribute("min"), 0)
		maxValue = widgetNumber(node.Attribute("max"), 1)
	}

	if maxValue <= minValue {
		return
	}

	value := widgetNumber(node.Attribute("value"), minValue)

	ratio := (value - minValue) / (maxValue - minValue)
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}

	contentX, contentW := e.contentBox(leftX, width, style)
	contentY := topY + e.scalePt(style.BorderTop.Width) + e.scalePt(style.PaddingTop)

	contentH := height - e.scalePt(style.BorderTop.Width+style.BorderBottom.Width) -
		e.scalePt(style.PaddingTop+style.PaddingBottom)
	if contentW <= 0 || contentH <= 0 || ratio <= 0 {
		return
	}

	// Native progress/meter fill: CSS accent-color, then an authored
	// background, then the widget default. Custom properties participate
	// only after they resolve into those properties (e.g. accent-color:
	// var(--token)); inherited document tokens are not scanned here.
	color := widgetValueColor(node.Name, style)

	const indicatorRatio = 0.3

	indicatorH := e.scalePt(style.FontSize) * indicatorRatio
	if indicatorH <= 0 || indicatorH > contentH {
		indicatorH = contentH
	}

	indicatorY := contentY + (contentH-indicatorH)/two

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpFillRect, X: contentX, Y: indicatorY, W: contentW * ratio, H: indicatorH,
		R: color[0], G: color[1], B: color[2], Alpha: 1,
		Radius: usedBorderRadius(style, contentW*ratio, indicatorH),
	})
}

func widgetValueColor(tag string, style ResolvedStyle) [3]float64 {
	if style.AccentColorSet {
		return style.AccentColor
	}

	if tag == htmlMeter {
		return style.Color
	}

	return [3]float64{0, 0.5, 0}
}

func widgetNumber(raw string, fallback float64) float64 {
	if value, err := strconv.ParseFloat(raw, 64); err == nil {
		return value
	}

	return fallback
}

// closedDetailsChildren implements the print-time disclosure rule: a closed
// details element paints its summary but does not lay out the hidden payload.
func closedDetailsChildren(node *html.Node) []*html.Node {
	for _, child := range node.Children {
		if child.Type == html.ElementNode && child.Name == "summary" {
			return []*html.Node{child}
		}
	}

	return nil
}

// applyHeightConstraints enforces the used/min/max-height constraints on the
// current content height (extracted so buildBlock stays readable).
func (e *engine) applyHeightConstraints(style ResolvedStyle, curY float64) float64 {
	if h, ok := resolveUsedHeight(style, -1, e); ok {
		if curY < h {
			curY = h
		}
	}

	minHeight := e.scalePt(style.MinHeight)
	if style.BoxSizing != borderBox {
		minHeight += e.scalePt(style.PaddingTop) + e.scalePt(style.PaddingBottom) +
			e.scalePt(style.BorderTop.Width) + e.scalePt(style.BorderBottom.Width)
	}

	if minHeight < 0 {
		minHeight = 0
	}

	if minHeight > 0 && curY < minHeight {
		curY = minHeight
	}

	maxHeight := e.scalePt(style.MaxHeight)
	if style.BoxSizing != borderBox {
		maxHeight += e.scalePt(style.PaddingTop) + e.scalePt(style.PaddingBottom) +
			e.scalePt(style.BorderTop.Width) + e.scalePt(style.BorderBottom.Width)
	}

	if style.MaxHeight >= 0 && curY > maxHeight {
		curY = maxHeight
	}

	return curY
}

// resolveBlockWidth computes a block's used border-box width and the scaled
// left margin. Horizontal auto margins center (or push) a definite-width box.
func resolveBlockWidth(eng *engine, style ResolvedStyle, availW float64) (float64, float64) {
	margR := eng.scalePt(style.MarginRight)
	margL := eng.scalePt(style.MarginLeft)
	// Default: fill remaining width after horizontal margins.
	width := availW - margL - margR
	if width < 0 {
		width = 0
	}

	definiteW := resolveDefiniteWidth(eng, style, availW, &width)
	// content-box (default): specified width is the content width, so the
	// border box grows by horizontal padding + border. border-box: specified
	// width already is the border-box size.
	if definiteW && style.BoxSizing != borderBox {
		width += eng.scalePt(style.PaddingLeft) + eng.scalePt(style.PaddingRight) +
			eng.scalePt(style.BorderLeft.Width) + eng.scalePt(style.BorderRight.Width)
	}

	width = clampBlockMinMax(eng, style, availW, width)
	margL = resolveAutoMargins(style, definiteW, width, availW, margL, margR)

	return width, margL
}

// resolveAutoMargins centers (or pushes) a definite-width block via auto
// horizontal margins (CSS2.1 §10.3.3).
func resolveAutoMargins(style ResolvedStyle, definiteW bool, width, availW, margL, margR float64) float64 {
	if definiteW && (style.MarginLeftAuto || style.MarginRightAuto) {
		free := availW - width
		if free < 0 {
			free = 0
		}

		switch {
		case style.MarginLeftAuto && style.MarginRightAuto:
			margL = free / two
		case style.MarginLeftAuto:
			margL = free - margR
			if margL < 0 {
				margL = 0
			}
		}
	}

	return margL
}

// resolveDefiniteWidth applies the width/width% to *w. Returns false when the
// width resolves to auto (cyclic % honesty: indefinite containing block).
func resolveDefiniteWidth(eng *engine, style ResolvedStyle, availW float64, width *float64) bool {
	definiteW := style.Width >= 0 || style.WidthPercent >= 0

	switch {
	case style.WidthPercent >= 0:
		// Cyclic % honesty: indefinite containing block → treat as auto.
		if availW > 0 && availW < 1e12 {
			*width = availW * style.WidthPercent / cssPercent
		} else {
			definiteW = false
		}
	case style.Width >= 0:
		*width = eng.scalePt(style.Width)
	}

	return definiteW
}

// clampBlockMinMax applies the min/max-width constraints to w.
func clampBlockMinMax(eng *engine, style ResolvedStyle, availW, width float64) float64 {
	if style.MinWidthPercent >= 0 && availW > 0 && availW < 1e12 {
		mn := availW * style.MinWidthPercent / cssPercent
		if width < mn {
			width = mn
		}
	} else if style.MinWidth > 0 && width < eng.scalePt(style.MinWidth) {
		width = eng.scalePt(style.MinWidth)
	}

	if style.MaxWidth >= 0 && width > eng.scalePt(style.MaxWidth) {
		width = eng.scalePt(style.MaxWidth)
	}

	return width
}

// resolveUsedHeight returns a definite border-box height when the style has a
// usable height. HeightPercent requires a definite containing-block height
// (cbH >= 0); otherwise the percentage is treated as auto (cyclic honesty).
func resolveUsedHeight(sty ResolvedStyle, cbH float64, engN *engine) (float64, bool) {
	if sty.HeightPercent >= 0 {
		if cbH < 0 {
			return 0, false
		}

		height := cbH * sty.HeightPercent / cssPercent
		if sty.BoxSizing != borderBox {
			height += engN.scalePt(sty.PaddingTop) + engN.scalePt(sty.PaddingBottom) +
				engN.scalePt(sty.BorderTop.Width) + engN.scalePt(sty.BorderBottom.Width)
		}

		return height, true
	}

	if sty.Height < 0 {
		return 0, false
	}

	height := engN.scalePt(sty.Height)
	if sty.BoxSizing != borderBox {
		height += engN.scalePt(sty.PaddingTop) + engN.scalePt(sty.PaddingBottom) +
			engN.scalePt(sty.BorderTop.Width) + engN.scalePt(sty.BorderBottom.Width)
	}

	return height, true
}

// buildOutOfFlow places absolute (viewportFixed=false) or fixed
// (viewportFixed=true) boxes. Fixed under a transformed ancestor is absolute
// (caller passes viewportFixed=false). Fixed ops are marked by build().
func (e *engine) buildOutOfFlow(node *html.Node, sty ResolvedStyle, availW, x, y float64, viewportFixed bool) *box {
	cbX, cbY, cbW := x, y, availW
	if viewportFixed {
		cbX, cbY = 0, 0

		cbW = e.opts.Width
		if cbW <= 0 {
			cbW = availW
		}
	}

	start := len(e.ops)

	buildW := e.absoluteBuildWidth(sty, cbW)

	boxNode := e.buildInFlowDisplay(node, sty, buildW, cbX, cbY)
	if boxNode == nil {
		return nil
	}

	boxNode.opStart, boxNode.opEnd = start, len(e.ops)-1

	absX := cbX
	if !sty.LeftAuto {
		absX = cbX + e.scalePt(sty.Left)
	} else if !sty.RightAuto {
		absX = cbX + cbW - boxNode.w - e.scalePt(sty.Right)
	}

	cbH := e.absCBHeights[node]
	absY := e.resolveAbsY(sty, boxNode, cbY, viewportFixed, cbH)

	dx, dy := absX-boxNode.x, absY-boxNode.y
	boxNode.x, boxNode.y = absX, absY
	e.shiftBoxOps(boxNode, dx, dy)

	return boxNode
}

func (e *engine) absoluteBuildWidth(sty ResolvedStyle, cbW float64) float64 {
	if sty.Width >= 0 || sty.WidthPercent >= 0 || sty.LeftAuto || sty.RightAuto {
		return cbW
	}

	buildW := cbW - e.scalePt(sty.Left+sty.Right)
	if buildW < 0 {
		return 0
	}

	return buildW
}

// resolveAbsY places the out-of-flow box vertically: top wins, then bottom
// (fixed resolves against the viewport bottom; absolute uses the containing
// block's bottom when its deferred height is known).
//
//nolint:wsl // ordered absolute-positioning cases mirror CSS precedence
func (e *engine) resolveAbsY(sty ResolvedStyle, boxNode *box, cbY float64, viewportFixed bool, cbH float64) float64 {
	if !sty.TopAuto {
		return cbY + e.scalePt(sty.Top)
	}

	if sty.BottomAuto {
		return cbY
	}

	if viewportFixed {
		absY := e.opts.Height - boxNode.height - e.scalePt(sty.Bottom)
		if absY < 0 {
			return e.scalePt(sty.Bottom)
		}

		return absY
	}
	if cbH > 0 {
		return cbY + cbH - boxNode.height - e.scalePt(sty.Bottom)
	}

	return cbY + e.scalePt(sty.Bottom)
}

// buildInFlowDisplay builds flex/grid/multicol/table/block ignoring position.
// Single display dispatch shared with build() so abspos/fixed get the same
// formatting context (including table display:table-as-block heuristic).
func (e *engine) buildInFlowDisplay(nodeN *html.Node, sty ResolvedStyle, availW, posX, posY float64) *box {
	if sty.Display == displayFlex || sty.Display == displayInlineFlex {
		return e.buildFlex(nodeN, sty, availW, posX, posY)
	}

	if sty.Display == displayGrid || sty.Display == displayInlineGrid || sty.Display == displaySubgrid {
		return e.buildGrid(nodeN, sty, availW, posX, posY)
	}

	if isMulticol(sty) {
		return e.buildMulticol(nodeN, sty, availW, posX, posY)
	}

	if isTableDisplay(sty.Display) {
		// MediaWiki thumbs use figure{display:table;float:right} purely for
		// shrink-to-fit. Routing those through buildTable drops nested <img>
		// (no table-row/cell structure) and leaves empty floated boxes that
		// clear following paragraphs with huge gaps. Treat non-table hosts
		// without table-structure children as ordinary blocks.
		if useBlockForTableDisplay(nodeN) {
			return e.buildBlock(nodeN, sty, availW, posX, posY)
		}

		return e.buildTable(nodeN, sty, availW, posX, posY)
	}

	return e.buildBlock(nodeN, sty, availW, posX, posY)
}
