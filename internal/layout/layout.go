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
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/svg"
)

// CSS keyword constants shared by the layout engine. Kept here so repeated
// string literals resolve through one named value (goconst).
const (
	positionAbsolute      = "absolute"
	positionFixed         = "fixed"
	positionRelative      = "relative"
	positionSticky        = "sticky"
	displayBlock          = "block"
	displayFlex           = "flex"
	displayGrid           = "grid"
	displayInlineFlex     = "inline-flex"
	displayInlineGrid     = "inline-grid"
	displaySubgrid        = "subgrid"
	displayFlowRoot       = "flow-root"
	displayTable          = "table"
	displayTableCell      = "table-cell"
	displayTableCaption   = "table-caption"
	displayTableRow       = "table-row"
	displayRowGroup       = "table-row-group"
	displayHeaderGroup    = "table-header-group"
	displayFooterGroup    = "table-footer-group"
	displayListItem       = "list-item"
	listStyleDisc         = "disc"
	bulletDisc            = "\u2022"
	borderCollapseValue   = "collapse"
	overflowWrapAnywhere  = "anywhere"
	overflowWrapBreakWord = "break-word"
	borderStyleDashed     = "dashed"
	borderStyleDotted     = "dotted"
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

// ElementLocation describes where one element box landed after pagination.
// X/Y/W/H are in canvas coordinates (y down from the top of the page content
// area); Page is the page index the box's first op was painted on.
type ElementLocation struct {
	Node *html.Node
	Page int
	X, Y float64
	W, H float64
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

	Text string
	Font *pdf.Font
	Size float64
	Bold bool

	URI string

	Image  []byte // PNG or JPEG bytes
	ImgW   int
	ImgH   int
	IsJPEG bool

	// Fixed marks ops from position:fixed boxes; Paint stamps them on every
	// page at viewport-relative coordinates.
	Fixed bool

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

	// Xform is a baked canvas-space CSS 2D transform (identity if unset).
	// Applied at paint via PDF cm (see pdfCTMFromCSS). Sibling flow unaffected.
	Xform    Matrix2D
	XformSet bool

	// PaintOpacity is element opacity (CSS opacity / filter:opacity), 0..1.
	// 0 or unset (≥1) means fully opaque. Nested opacities are multiplied.
	PaintOpacity float64
}

type engine struct {
	opts       Options
	ctx        context.Context //nolint:containedctx // ctx is checked at recursion boundaries (checkContext)
	err        error
	font       *pdf.Font // default/regular face (metrics fallback)
	faces      *pdf.FaceSet
	registry   *pdf.Registry
	// styles holds one heap ResolvedStyle per node (from resolveStylesCtx).
	// Callers use stylePtr for shared *ResolvedStyle without a second copy.
	styles map[*html.Node]*ResolvedStyle
	ops        []Op
	noEmit     bool // measurement mode: compute geometry without emitting ops
	height     float64
	scale      float64 // zoom factor applied to style lengths (>= 1)
	zIndex     int
	zIndexSet  bool
	positioned bool
	stickySeq  int // monotonically increasing sticky box IDs (for Op.StickyID)
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

// hashFontFamily fingerprints a CSS font-family list without allocating.
func hashFontFamily(fams []string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	h := uint64(offset64)
	for _, fam := range fams {
		for i := 0; i < len(fam); i++ {
			h ^= uint64(fam[i])
			h *= prime64
		}

		h ^= 0xff // token separator
		h *= prime64
	}

	return h
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
		famHash: hashFontFamily(sty.FontFamily),
		weight:  sty.FontWeight,
		italic:  sty.FontItalic,
	}
	if e.faceByStyle != nil {
		if f, ok := e.faceByStyle[key]; ok {
			return f
		}
	}

	f := e.lookupFaceFor(sty)
	if e.faceByStyle == nil {
		e.faceByStyle = make(map[faceStyleKey]*pdf.Font)
	}

	e.faceByStyle[key] = f

	return f
}

// lookupFaceFor is the uncached faceFor path.
func (e *engine) lookupFaceFor(sty ResolvedStyle) *pdf.Font {
	if e.registry != nil {
		if f := e.registry.Lookup(sty.FontFamily, sty.FontWeight, sty.FontItalic); f != nil {
			return f
		}
	}

	if e.faces != nil {
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
		famHash: hashFontFamily(sty.FontFamily),
		weight:  sty.FontWeight,
		italic:  sty.FontItalic,
		r:       runeValue,
	}
	if e.faceByRune != nil {
		if f, ok := e.faceByRune[key]; ok {
			return f
		}
	}

	f := e.lookupFaceForRune(sty, runeValue)
	if f == nil {
		f = primary
	}

	if e.faceByRune == nil {
		e.faceByRune = make(map[faceRuneKey]*pdf.Font)
	}

	e.faceByRune[key] = f

	return f
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
func (e *engine) facesWithGlyph(style ResolvedStyle, runeValue rune) *pdf.Font {
	if e.faces == nil {
		return nil
	}

	if f := e.faces.Resolve(style.FontWeight, style.FontItalic); f != nil && f.GlyphID(runeValue) != 0 {
		return f
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
func LayoutContext(ctx context.Context, //nolint:revive,contextcheck // legacy Layout adapter (stutter + nil ctx)
	root *html.Node, opts Options,
) (*Result, error) {
	if root == nil {
		return nil, errors.New("layout: nil root") //nolint:err113 // static sentinel-free message matches legacy behavior
	}

	if ctx == nil {
		// Legacy callers without a request context get a background context.
		ctx = context.Background()
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

	styles, containers := resolveStylesForLayout(root, opts)

	return finalizeResult(newEngine(ctx, root, opts, faces, font, styles, containers), root, opts)
}

// newEngine constructs the layout engine state (extracted from LayoutContext
// for clarity).
func newEngine(
	ctx context.Context, root *html.Node, opts Options,
	faces *pdf.FaceSet, font *pdf.Font, styles map[*html.Node]*ResolvedStyle, containers map[*html.Node]sizeContainer,
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
		ops:        make([]Op, 0, estimateOpCapacity(root)),
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
	// Pass 1: cascade without @container (used sizes unknown).
	styles := resolveStylesWith(root, opts, nil)

	if !css.HasContainerRules(opts.Sheets) {
		return styles, nil
	}
	// After definite inline sizes of size containers are known, re-cascade so
	// matching @container rules apply, then lay out once with final styles.
	cinfo := measureSizeContainers(root, styles, opts.Width)
	if len(cinfo) == 0 {
		return styles, nil
	}

	styles = resolveStylesWith(root, opts, cinfo)

	// One nested remount: @container may change nested container-type.
	cinfo2 := measureSizeContainers(root, styles, opts.Width)
	if len(cinfo2) != len(cinfo) || !sameSizeContainers(cinfo, cinfo2) {
		return resolveStylesWith(root, opts, cinfo2), cinfo2
	}

	return styles, cinfo
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
	if p := e.styles[node]; p != nil {
		return p
	}

	// Missing node (should not happen for walked trees): stable empty style.
	return &zeroResolvedStyle
}

// styleVal returns a by-value ResolvedStyle for APIs that still take values.
// Prefer field access on e.styles[node] / stylePtr to avoid the copy.
func (e *engine) styleVal(node *html.Node) ResolvedStyle {
	if p := e.styles[node]; p != nil {
		return *p
	}

	return ResolvedStyle{} //nolint:exhaustruct // intentional zero fields
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

	capacity := nodes * minBoxPt
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
	col, span int
	row       int // owning table row index, set once at placement
	rowSpan   int // vertical span (default 1) for <td rowspan>
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
		boxNode = e.buildImage(node, sty, posX, posY)
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
		case "tr", "tbody", "thead", "tfoot", "colgroup", "col", "caption":
			return false
		}
	}

	return true
}

// buildBlock lays out a block-level box.
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
	curY = e.flowChildren(boxNode, node.Children, style, contentW, contentX, posY, curY)

	if enclose && e.bfcFloats != nil {
		curY = e.bfcFloats.extentCy(posY, curY)
	}

	e.popBFCFloats(enclose)
	// padding-bottom is inside the border box (space above border-bottom /
	// letterhead rules — fixture-07/16).
	curY += e.scalePt(style.PaddingBottom)

	// list marker (outside the principal box content — in the marker area)
	if node.Name == "li" && boxNode.firstBaseline > 0 {
		e.emitListMarker(node, style, contentX, boxNode.firstBaseline)
	}

	boxNode.height = e.applyHeightConstraints(style, curY)

	e.prependChrome(contentStart, boxNode, style, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

// applyHeightConstraints enforces the used/min/max-height constraints on the
// current content height (extracted so buildBlock stays readable).
func (e *engine) applyHeightConstraints(style ResolvedStyle, curY float64) float64 {
	if h, ok := resolveUsedHeight(style, -1, e); ok {
		if curY < h {
			curY = h
		}
	}

	if style.MinHeight > 0 && curY < e.scalePt(style.MinHeight) {
		curY = e.scalePt(style.MinHeight)
	}

	if style.MaxHeight >= 0 && curY > e.scalePt(style.MaxHeight) {
		curY = e.scalePt(style.MaxHeight)
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

	boxNode := e.buildInFlowDisplay(node, sty, cbW, cbX, cbY)
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

	absY := e.resolveAbsY(sty, boxNode, cbY, viewportFixed)

	dx, dy := absX-boxNode.x, absY-boxNode.y
	boxNode.x, boxNode.y = absX, absY
	e.shiftBoxOps(boxNode, dx, dy)

	return boxNode
}

// resolveAbsY places the out-of-flow box vertically: top wins, then bottom
// (fixed resolves against the viewport bottom; absolute against the CB top).
func (e *engine) resolveAbsY(sty ResolvedStyle, boxNode *box, cbY float64, viewportFixed bool) float64 {
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
	// Absolute bottom: offset from CB top (lite; not height−bottom).
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

func (e *engine) markOpsFixed(start, end int) {
	if end < start {
		return
	}

	for i := start; i <= end && i < len(e.ops); i++ {
		e.ops[i].Fixed = true
	}
}

// borderLineOps expands solid/dashed/dotted borders into the line operations
// consumed by both PDF and raster painting. Keeping the pattern as segments
// avoids adding a second stroke-style protocol to Op.
// Prefer appendBorderLineOps / emitBorderLine to avoid per-edge slice headers.
func borderLineOps(posX, posY, boxW, boxH, width float64, style string, red, green, blue float64) []Op {
	return appendBorderLineOps(nil, posX, posY, boxW, boxH, width, style, red, green, blue)
}

// appendBorderLineOps appends border segment ops into dst (may be nil).
func appendBorderLineOps(
	dst []Op, posX, posY, boxW, boxH, width float64, style string, red, green, blue float64,
) []Op {
	if width <= 0 || style == cssDisplayNone || (boxW <= 0 && boxH <= 0) {
		return dst
	}

	if style != borderStyleDashed && style != borderStyleDotted {
		return append(dst, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: posX, Y: posY, W: boxW, H: boxH, Width: width, R: red, G: green, B: blue,
		})
	}

	return appendDashedLineSegments(dst, posX, posY, boxW, boxH, width, style == borderStyleDotted, red, green, blue)
}

// appendDashedLineSegments expands a dashed/dotted border edge into segment ops.
func appendDashedLineSegments(
	dst []Op, posX, posY, boxW, boxH, width float64, dotted bool, red, green, blue float64,
) []Op {
	horizontal := boxW > 0
	length := boxW

	if !horizontal {
		length = boxH
	}

	drawLen, gap := width*three, width*two
	if dotted {
		drawLen, gap = width, width*dashGapMul
	}

	if drawLen < halfRatio {
		drawLen = 0.5
	}

	if gap < halfRatio {
		gap = 0.5
	}

	if n := int(length/(drawLen+gap)) + 1; cap(dst)-len(dst) < n {
		// Grow once for the expected segment count.
		grown := make([]Op, len(dst), len(dst)+n)
		copy(grown, dst)
		dst = grown
	}

	for pos := 0.0; pos < length-0.001; pos += drawLen + gap {
		seg := math.Min(drawLen, length-pos)
		if seg <= 0 {
			break
		}

		segX, segY, segW, segH := posX+pos, posY, seg, 0.0
		if !horizontal {
			segX, segY, segW, segH = posX, posY+pos, 0.0, seg
		}

		dst = append(dst, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: segX, Y: segY, W: segW, H: segH,
			Width: width, R: red, G: green, B: blue,
		})
	}

	return dst
}

// emitBorderLine appends one edge's border segments straight onto e.ops —
// no intermediate []Op (hot path for collapsed table grids).
func (e *engine) emitBorderLine(posX, posY, boxW, boxH, width float64, style string, red, green, blue float64) {
	if width <= 0 || style == cssDisplayNone || (boxW <= 0 && boxH <= 0) {
		return
	}

	if style != borderStyleDashed && style != borderStyleDotted {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: posX, Y: posY, W: boxW, H: boxH, Width: width, R: red, G: green, B: blue,
		})

		return
	}

	// Dashed/dotted: append into a tiny stack buffer then emit.
	var buf [8]Op
	segs := appendDashedLineSegments(buf[:0], posX, posY, boxW, boxH, width, style == borderStyleDotted, red, green, blue)
	for i := range segs {
		e.add(segs[i])
	}
}

// borderOps returns the four border line ops for the given border box.
func (e *engine) borderOps(sty ResolvedStyle, posX, posY, wid, height float64) []Op {
	ops := make([]Op, 0, 4)

	ops = appendBorderLineOps(ops, posX, posY, wid, 0, e.scalePt(sty.BorderTop.Width), sty.BorderTop.Style,
		sty.BorderTop.Color[0], sty.BorderTop.Color[1], sty.BorderTop.Color[2])
	ops = appendBorderLineOps(ops, posX+wid, posY, 0, height, e.scalePt(sty.BorderRight.Width), sty.BorderRight.Style,
		sty.BorderRight.Color[0], sty.BorderRight.Color[1], sty.BorderRight.Color[2])
	ops = appendBorderLineOps(ops, posX, posY+height, wid, 0, e.scalePt(sty.BorderBottom.Width), sty.BorderBottom.Style,
		sty.BorderBottom.Color[0], sty.BorderBottom.Color[1], sty.BorderBottom.Color[2])
	ops = appendBorderLineOps(ops, posX, posY, 0, height, e.scalePt(sty.BorderLeft.Width), sty.BorderLeft.Style,
		sty.BorderLeft.Color[0], sty.BorderLeft.Color[1], sty.BorderLeft.Color[2])

	return ops
}

// chromeMustSpliceImmediately reports boxes whose chrome must land in e.ops
// during the build: sticky (StickyID stamp), fixed (Fixed stamp), and
// transform (op-range exclusive CTM stamp). Common static/relative boxes defer.
func chromeMustSpliceImmediately(st ResolvedStyle) bool {
	if st.HasTransform {
		return true
	}

	switch st.Position {
	case positionSticky, positionFixed:
		return true
	}

	return false
}

// prependChrome inserts background + border ops at insertAt so they paint
// under any content ops already appended for this box.
//
// Common path defers the splice until finalizeChrome (one linear merge).
// Sticky/fixed/transform keep an immediate splice so mid-build StickyID/Fixed
// stamps and transform exclusive ranges stay correct without re-derivation.
func (e *engine) prependChrome(insertAt int, boxNode *box, sty ResolvedStyle, posX, posY, width, height float64) {
	if e.noEmit {
		return
	}

	var chrome []Op
	if sty.BGColor[3] > 0 && e.opts.Background {
		chrome = append(chrome, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: posX, Y: posY, W: width, H: height,
			R: sty.BGColor[0], G: sty.BGColor[1], B: sty.BGColor[2], Alpha: sty.BGColor[3],
		})
	}

	chrome = append(chrome, e.borderOps(sty, posX, posY, width, height)...)
	if len(chrome) == 0 {
		return
	}

	for i := range chrome {
		chrome[i].ZIndex = e.zIndex
		chrome[i].ZIndexSet = e.zIndexSet
		chrome[i].Positioned = e.positioned
	}

	if !chromeMustSpliceImmediately(sty) {
		e.deferredChrome = append(e.deferredChrome, chromeEntry{at: insertAt, ops: chrome, b: boxNode})

		return
	}
	// Immediate splice (sticky/fixed/transform).
	tail := append([]Op(nil), e.ops[insertAt:]...)
	e.ops = e.ops[:insertAt]
	e.ops = append(e.ops, chrome...)
	e.ops = append(e.ops, tail...)
	// Keep deferred insert indices valid after this mid-build splice.
	n := len(chrome)

	for i := range e.deferredChrome {
		if e.deferredChrome[i].at >= insertAt {
			e.deferredChrome[i].at += n
		}
	}
}

// finalizeChrome merges deferred background/border ops into e.ops in one
// linear pass and reindexes box op ranges. Paint order for multiple entries
// at the same index matches immediate-splice nesting: later (outer) entries
// paint first.
func (e *engine) finalizeChrome(root *box) {
	if len(e.deferredChrome) == 0 {
		return
	}

	entries := e.deferredChrome
	e.deferredChrome = nil

	out, oldToNew, ownerChrome := mergeDeferredChrome(e.ops, entries)

	e.ops = out

	// Remap content op ranges, expand owners with their chrome, then union
	// parent ranges over children so ancestor ranges still cover nested chrome.
	remapBoxRangesWithChrome(root, oldToNew, ownerChrome)
	unionChildOpRanges(root)

	// Deferred chrome under sticky ancestors never received StickyID at build
	// time; re-stamp from the box tree. Fixed content already marked Fixed —
	// expand Fixed onto chrome in the same range when any op is Fixed.
	restampStickyFixed(root, e.ops)
}

// chromeSpan is an inclusive op range owned by one box's chrome.
type chromeSpan struct{ start, end int }

// mergeDeferredChrome splices deferred background/border ops into oldOps in
// one linear pass. Paint order for multiple entries at the same index matches
// immediate-splice nesting: later (outer) entries paint first.
func mergeDeferredChrome(
	oldOps []Op, entries []chromeEntry,
) ([]Op, []int, map[*box]chromeSpan) {
	// Sort by insert index ascending; same index → reverse registration order
	// (parent registered after child, paints under content first).
	type indexed struct {
		ord int
		ent chromeEntry
	}

	order := make([]indexed, len(entries))
	for i, ent := range entries {
		order[i] = indexed{ord: i, ent: ent}
	}

	sort.SliceStable(order, func(i, j int) bool {
		if order[i].ent.at != order[j].ent.at {
			return order[i].ent.at < order[j].ent.at
		}
		// Higher ord (later register) first within the same at.
		return order[i].ord > order[j].ord
	})

	totalChrome := 0
	for _, it := range order {
		totalChrome += len(it.ent.ops)
	}

	out := make([]Op, 0, len(oldOps)+totalChrome)
	oldToNew := make([]int, len(oldOps))
	ownerChrome := map[*box]chromeSpan{}

	oidx := 0
	for idx, paintOp := range oldOps {
		for oidx < len(order) && order[oidx].ent.at == idx {
			ent := order[oidx].ent
			cs := len(out)
			out = append(out, ent.ops...)
			recordOwnerChrome(ownerChrome, ent.b, cs, len(out)-1)

			oidx++
		}

		oldToNew[idx] = len(out)
		out = append(out, paintOp)
	}
	// Trailing chrome (chrome-only boxes with insertAt == len(ops)).
	for oidx < len(order) {
		ent := order[oidx].ent
		cs := len(out)
		out = append(out, ent.ops...)
		recordOwnerChrome(ownerChrome, ent.b, cs, len(out)-1)

		oidx++
	}

	return out, oldToNew, ownerChrome
}

// recordOwnerChrome widens the chrome op span recorded for a box so a later
// (outer) entry keeps the owner's range covering all of its chrome.
func recordOwnerChrome(ownerChrome map[*box]chromeSpan, boxNode *box, start, endIdx int) {
	if boxNode == nil || endIdx < start {
		return
	}

	if prev, ok := ownerChrome[boxNode]; ok {
		if start < prev.start {
			prev.start = start
		}

		if endIdx > prev.end {
			prev.end = endIdx
		}

		ownerChrome[boxNode] = prev

		return
	}

	ownerChrome[boxNode] = chromeSpan{start: start, end: endIdx}
}

// remapBoxRangesWithChrome rewrites box op ranges through the old→new index
// map, then expands each owner with its recorded chrome span.
func remapBoxRangesWithChrome(root *box, oldToNew []int, ownerChrome map[*box]chromeSpan) {
	var remap func(b *box)
	remap = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.opEnd >= boxNode.opStart && boxNode.opStart >= 0 && boxNode.opEnd < len(oldToNew) {
			boxNode.opStart = oldToNew[boxNode.opStart]
			boxNode.opEnd = oldToNew[boxNode.opEnd]
		}

		if span, ok := ownerChrome[boxNode]; ok {
			mergeOwnerChromeSpan(boxNode, span)
		}

		for _, child := range boxNode.children {
			remap(child)
		}
	}
	remap(root)
}

// mergeOwnerChromeSpan unions a box's content range with its chrome span.
func mergeOwnerChromeSpan(boxNode *box, span chromeSpan) {
	if boxNode.opEnd < boxNode.opStart {
		boxNode.opStart, boxNode.opEnd = span.start, span.end

		return
	}

	if span.start < boxNode.opStart {
		boxNode.opStart = span.start
	}

	if span.end > boxNode.opEnd {
		boxNode.opEnd = span.end
	}
}

// unionChildOpRanges widens every box range over its children's ranges so
// ancestor ranges still cover nested chrome.
func unionChildOpRanges(root *box) {
	var unionChildren func(b *box)
	unionChildren = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		for _, child := range boxNode.children {
			unionChildren(child)

			if child.opEnd < child.opStart {
				continue
			}

			if boxNode.opEnd < boxNode.opStart {
				boxNode.opStart, boxNode.opEnd = child.opStart, child.opEnd

				continue
			}

			if child.opStart < boxNode.opStart {
				boxNode.opStart = child.opStart
			}

			if child.opEnd > boxNode.opEnd {
				boxNode.opEnd = child.opEnd
			}
		}
	}
	unionChildren(root)
}

// restampStickyFixed re-applies StickyID from sticky boxes and expands Fixed
// onto the full op range when the box was viewport-fixed (any op already Fixed).
func restampStickyFixed(boxNode *box, ops []Op) {
	if boxNode == nil {
		return
	}

	reapplyStickyID(boxNode, ops)
	expandFixedOps(boxNode, ops)

	for _, c := range boxNode.children {
		restampStickyFixed(c, ops)
	}
}

// reapplyStickyID stamps StickyID onto a sticky box's whole op range.
func reapplyStickyID(boxNode *box, ops []Op) {
	if !boxNode.sticky || boxNode.stickyID == 0 || boxNode.opEnd < boxNode.opStart || boxNode.opStart < 0 {
		return
	}

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		ops[i].StickyID = boxNode.stickyID
	}
}

// expandFixedOps spreads the Fixed mark over a viewport-fixed box's op range
// when any op in it is already Fixed (chrome added after build is included).
func expandFixedOps(boxNode *box, ops []Op) {
	if boxNode.style.Position != positionFixed || boxNode.opEnd < boxNode.opStart || boxNode.opStart < 0 {
		return
	}

	hasFixed := false

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		if ops[i].Fixed {
			hasFixed = true

			break
		}
	}

	if !hasFixed {
		return
	}

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		ops[i].Fixed = true
	}
}

// contentBox returns the content-box origin and width for a border box.
// Single home for "content = border-box − scaled padding − scaled border".
func (e *engine) contentBox(posX, boxW float64, style ResolvedStyle) (float64, float64) {
	contentW := boxW - e.scalePt(style.PaddingLeft) - e.scalePt(style.PaddingRight) -
		e.scalePt(style.BorderLeft.Width) - e.scalePt(style.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}

	return posX + e.scalePt(style.BorderLeft.Width) + e.scalePt(style.PaddingLeft), contentW
}

// imageRef is the resolved form of one <img src>: bytes + intrinsic size,
// resolved at most once per Layout run (measure and build share it).
type imageRef struct {
	src    string
	data   []byte
	w, h   int
	isJPEG bool
}

// resolveImage fetches (once) and decodes (once) src; nil on any failure.
func (e *engine) resolveImage(src string) *imageRef {
	if src == "" || e.opts.Images == nil {
		return nil
	}

	if e.imgCache == nil {
		e.imgCache = map[string]*imageRef{}
	}

	if ref, ok := e.imgCache[src]; ok {
		return ref
	}

	data, err := e.opts.Images(src)
	if err != nil {
		// Cache nil-miss? Store a sentinel empty ref so we do not re-fetch.
		e.imgCache[src] = nil

		return nil
	}

	ref := &imageRef{src: src, data: data} //nolint:exhaustruct // intentional zero fields
	if png, pw, ph, err := svg.Rasterize(data, svgRasterMax); err == nil {
		ref.data, ref.w, ref.h = png, pw, ph
	} else if w, h, jpeg, ok := imageDims(data); ok {
		ref.w, ref.h, ref.isJPEG = w, h, jpeg
	}

	e.imgCache[src] = ref

	return ref
}

// isInlineChild reports whether n participates in an inline formatting context.
func (e *engine) isInlineChild(node *html.Node) bool {
	if node.Type == html.TextNode {
		return true
	}

	if node.Type != html.ElementNode {
		return false
	}

	cstate := e.styles[node]
	if cstate.Display == cssDisplayNone || cstate.Float != cssDisplayNone ||
		cstate.Position == positionAbsolute || cstate.Position == positionFixed {
		return false
	}
	// <img> is replaced and UA-default inline-block, but author CSS may set
	// display:block (wiki .mw-logo-wordmark / .mw-logo-tagline stack).
	if node.Name == cssTagImg {
		return !blockishDisplay(cstate.Display)
	}

	return cstate.Display == cssDisplayInline || cstate.Display == cssDisplayInlineBlock ||
		cstate.Display == displayInlineFlex
}

// blockishDisplay reports display values that force a block formatting
// context for <img> and inline-level replaced elements.
func blockishDisplay(display string) bool {
	switch display {
	case displayBlock, displayFlex, displayGrid, displayTable, displayListItem, displayFlowRoot:
		return true
	default:
		return false
	}
}

// onlyCollapsibleWS reports whether every node is a text node of only
// whitespace (collapses between blocks and must not kill margin collapse).
func onlyCollapsibleWS(nodes []*html.Node) bool {
	if len(nodes) == 0 {
		return true
	}

	for _, n := range nodes {
		if n.Type != html.TextNode || strings.TrimSpace(n.Text) != "" {
			return false
		}
	}

	return true
}

// flowChildren lays out children in document order: runs of inlines, then
// block boxes, alternating as they appear. Floated children are positioned
// out of flow with a lite exclusion model; clear advances past them.
// Returns the advanced content height (cy end − cy start contribution is
// encoded as the final cy relative to start; callers pass starting cy).
// Float enclosure (extentCy) is the caller's job when it owns a BFC.
func (e *engine) flowChildren(
	parent *box, children []*html.Node, sty ResolvedStyle,
	contentW, contentX, posY, curY float64,
) float64 {
	prevBottom := 0.0

	var local floatState

	floats := e.bfcFloats
	if floats == nil {
		// Defensive: callers should pushBFCFloats first. Keep a local state
		// so isolated measure passes (layoutCell) still work.
		local = newFloatState(contentX, contentW)
		floats = &local
	}

	var deferred []*html.Node
	// Absolute/fixed containing-block origin is the content edge at entry.
	// Do not use the post-flow cy or deferred boxes sit below in-flow siblings.
	absOriginY := posY + curY

	absCBX, absCBW := contentX, contentW
	if sty.HasTransform {
		// Transformed element: padding box is the CB for abs/fixed descendants.
		absCBX = contentX - e.scalePt(sty.PaddingLeft)
		absOriginY -= e.scalePt(sty.PaddingTop)
		absCBW = contentW + e.scalePt(sty.PaddingLeft) + e.scalePt(sty.PaddingRight)
	}

	idx := 0
	for idx < len(children) {
		if e.checkContext() {
			return curY
		}

		curY, prevBottom, idx, deferred = e.flowOneChild(parent, children, idx, sty,
			contentW, contentX, posY, curY, prevBottom, floats, deferred)
	}

	for _, n := range deferred {
		ab := e.build(n, absCBW, absCBX, absOriginY)
		if ab != nil && parent != nil {
			parent.children = append(parent.children, ab)
		}
	}

	return curY
}

// flowOneChild advances one flow child (inline run, block, float or
// out-of-flow deferral), returning the updated cy, prevBottom, loop index and
// deferred list (extracted from flowChildren to keep each piece focused).
func (e *engine) flowOneChild(
	parent *box, children []*html.Node, idx int, sty ResolvedStyle,
	contentW, contentX, posY, curY, prevBottom float64, floats *floatState, deferred []*html.Node,
) (float64, float64, int, []*html.Node) {
	node := children[idx]

	switch {
	case isSkippableFlowNode(node, e):
		idx++
	case isOutOfFlowNode(node, e):
		// Defer out-of-flow boxes so they paint above in-flow content
		// (absolute overlays sit on top of later siblings' text).
		deferred = append(deferred, node)
		idx++
	case isFlowFloat(node, e):
		cs := e.styleVal(node)
		curY = floats.clearFloats(cs.Clear, posY, curY)
		attachFlowBox(parent, e.placeFloat(node, cs, floats, contentW, contentX, posY, curY), e)

		prevBottom = 0
		idx++
	case e.isInlineChild(node):
		run, next := collectInlineRun(children, idx, e)
		idx = next
		curY, prevBottom = e.layoutInlineRun(parent, sty, run, contentW, contentX, posY, curY, floats, prevBottom)
	case node.Type == html.ElementNode:
		var cblock *box
		curY, prevBottom, cblock = e.layoutBlockChild(node, floats, contentW, contentX, posY, curY, prevBottom)
		attachFlowBox(parent, cblock, e)

		idx++
	default:
		idx++
	}

	return curY, prevBottom, idx, deferred
}

// layoutInlineRun lays out one maximal inline run, returning the advanced cy
// and the margin accumulator.
func (e *engine) layoutInlineRun(
	parent *box, sty ResolvedStyle, run []*html.Node, contentW, contentX, posY, curY float64,
	floats *floatState, prevBottom float64,
) (float64, float64) {
	if onlyCollapsibleWS(run) {
		return curY, prevBottom
	}

	if len(run) > 0 {
		// Pass the real parent only. Synthetic measure-only boxes used to
		// allocate a full ResolvedStyle per inline run (hot on table cells);
		// emitLine tolerates a nil box (skips firstBaseline / text-align).
		h := e.layoutInlineFloats(parent, run, contentW, contentX, posY+curY, floats)
		curY += h

		if h > 0 {
			prevBottom = 0
		}
	}

	return curY, prevBottom
}

// isSkippableFlowNode reports nodes that are dropped from flow: display:none
// elements and pure-whitespace text (so margin collapse between block
// siblings is not interrupted — fixture-19 margin-bottom between divs).
func isSkippableFlowNode(node *html.Node, engine *engine) bool {
	if node.Type == html.ElementNode {
		if st := engine.styles[node]; st != nil && st.Display == cssDisplayNone {
			return true
		}
	}

	return node.Type == html.TextNode && strings.TrimSpace(node.Text) == ""
}

// isOutOfFlowNode reports absolute/fixed children (deferred to paint above
// the in-flow content of the current box).
func isOutOfFlowNode(node *html.Node, engine *engine) bool {
	if node.Type != html.ElementNode {
		return false
	}

	st := engine.styles[node]
	if st == nil {
		return false
	}

	return st.Position == positionAbsolute || st.Position == positionFixed
}

// isFlowFloat reports floated element children.
func isFlowFloat(node *html.Node, engine *engine) bool {
	if node.Type != html.ElementNode {
		return false
	}

	st := engine.styles[node]

	return st != nil && st.Float != cssDisplayNone
}

// collectInlineRun gathers a maximal run of inline children starting at idx,
// skipping display:none elements and keeping interior whitespace.
func collectInlineRun(children []*html.Node, idx int, engine *engine) ([]*html.Node, int) {
	var run []*html.Node

	for idx < len(children) {
		child := children[idx]
		if child.Type == html.ElementNode && engine.styles[child].Display == cssDisplayNone {
			idx++

			continue
		}

		if child.Type == html.ElementNode && engine.styles[child].Float != cssDisplayNone {
			break
		}

		if child.Type == html.TextNode && strings.TrimSpace(child.Text) == "" {
			// keep interior whitespace inside an inline run, but a run that
			// is only WS is dropped below.
			run = append(run, child)
			idx++

			continue
		}

		if !engine.isInlineChild(child) {
			break
		}

		run = append(run, child)
		idx++
	}

	return run, idx
}

// attachFlowBox appends a built child to its parent and draws the debug
// outline when DebugBoxes is on.
func attachFlowBox(parent *box, child *box, engine *engine) {
	if child == nil || parent == nil {
		return
	}

	parent.children = append(parent.children, child)

	if engine.opts.DebugBoxes {
		engine.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpStrokeRect, X: child.x, Y: child.y, W: child.w, H: child.height, R: 1, G: 0, B: 0,
		})
	}
}

// layoutBlockChild builds one block-level child: it clears floats, collapses
// margins with the previous sibling, applies the BFC float exclusion, and
// returns the advanced cy, the next margin accumulator, and the built box.
func (e *engine) layoutBlockChild(
	node *html.Node, floats *floatState, contentW, contentX, posY, curY, prevBottom float64,
) (float64, float64, *box) {
	cstate := e.styleVal(node)
	// In-flow tables always clear below preceding floats (deterministic
	// report policy). Shrink-to-fit / squeeze-beside is unsupported.
	clearVal := cstate.Clear
	if cstate.Display == displayTable {
		clearVal = clearBoth
	}

	curY = floats.clearFloats(clearVal, posY, curY)
	curY += collapseMargins(prevBottom, e.scalePt(cstate.MarginTop))
	// CSS2.1 §9.5: line boxes next to floats are shortened, not the
	// block box — so normal paragraphs get full content width and
	// re-query exclusion per line (wiki "Leading roles" reclaim).
	// §9.5 / BFC: flow-root, overflow≠visible, flex, etc. must not
	// overlap float margin boxes — otherwise heading border-bottom
	// paints through the infobox (wiki .mw-heading{display:flow-root}).
	bx, bw := contentX, contentW
	if establishesBFC(cstate) {
		bx, bw = floats.exclusion(contentX, contentW, posY, curY)
	}

	cblock := e.build(node, bw, bx, posY+curY)
	if cblock == nil {
		return curY, 0, nil
	}

	return curY + cblock.height, e.scalePt(cstate.MarginBottom), cblock
}

// pushBFCFloats installs a floatState for the current box. When the box
// establishes a BFC (or is the root), a fresh state is used and enclose is
// true so the caller should extend height with extentCy. Otherwise the
// parent BFC's state is reused and floats may protrude.
//
// Pair every push with popBFCFloats(enclose). No per-call closure is allocated.
func (e *engine) pushBFCFloats(style ResolvedStyle, contentX, contentW float64) bool {
	if e.bfcFloats != nil && !establishesBFC(style) {
		return false
	}

	e.bfcStack = append(e.bfcStack, e.bfcFloats)

	var fs *floatState
	if n := len(e.bfcPool); n > 0 {
		fs = e.bfcPool[n-1]
		e.bfcPool = e.bfcPool[:n-1]
	} else {
		fs = new(floatState)
	}

	*fs = newFloatState(contentX, contentW)
	e.bfcFloats = fs

	return true
}

// popBFCFloats restores the BFC float state installed by a matching
// pushBFCFloats that returned enclose=true.
func (e *engine) popBFCFloats(enclose bool) {
	if !enclose {
		return
	}

	if cur := e.bfcFloats; cur != nil {
		*cur = floatState{} //nolint:exhaustruct // clear before pool reuse
		e.bfcPool = append(e.bfcPool, cur)
	}

	n := len(e.bfcStack)
	if n == 0 {
		e.bfcFloats = nil

		return
	}

	e.bfcFloats = e.bfcStack[n-1]
	e.bfcStack = e.bfcStack[:n-1]
}

// emitListMarker paints the list marker in the marker area to the left of
// the content edge so it does not overlap the principal text.
func (e *engine) emitListMarker(node *html.Node, style ResolvedStyle, contentX, baseline float64) {
	typ := style.ListStyleType
	if typ == "" {
		typ = listStyleDisc
	}

	if typ == cssDisplayNone {
		return
	}

	size := e.scalePt(style.FontSize)
	face := e.faceFor(style)

	text := markerText(node, typ)

	minW := 0.0

	if face != nil {
		for _, r := range text {
			minW += face.AdvanceInPoints(r, size)
		}
	}

	if minW <= 0 {
		minW = size * float64(len([]rune(text))) * halfRatio
	}
	// Outside marker: sit in the padding/margin gutter left of contentX.
	gap := size * bulletGapRatio

	posX := contentX - gap - minW
	if posX < 0 {
		posX = 0
	}

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpBullet, X: posX, Y: baseline, Text: text, Font: face, Size: size,
		R: style.Color[0], G: style.Color[1], B: style.Color[2],
	})
}

// markerText returns the glyph/string for a list-style-type keyword.
func markerText(node *html.Node, typ string) string {
	switch typ {
	case listStyleDisc:
		return bulletDisc
	case "circle":
		return "\u25E6"
	case "square":
		return "\u25AA"
	case "decimal", "decimal-leading-zero":
		return strconv.Itoa(listItemIndex(node)) + "."
	case "lower-alpha", "lower-latin":
		return alphaMarker(listItemIndex(node), false) + "."
	case "upper-alpha", "upper-latin":
		return alphaMarker(listItemIndex(node), true) + "."
	case "lower-roman":
		return romanMarker(listItemIndex(node), false) + "."
	case "upper-roman":
		return romanMarker(listItemIndex(node), true) + "."
	default:
		return bulletDisc
	}
}

// listItemIndex is the 1-based index among element siblings that are list items.
func listItemIndex(node *html.Node) int {
	if node == nil || node.Parent == nil {
		return 1
	}

	idx := 0

	for _, child := range node.Parent.Children {
		if child.Type != html.ElementNode {
			continue
		}

		if !strings.EqualFold(child.Name, "li") {
			continue
		}

		idx++
		if child == node {
			return idx
		}
	}

	return 1
}

func alphaMarker(node int, upper bool) string {
	if node < 1 {
		node = 1
	}

	var chars []byte

	for node > 0 {
		node--

		ch := byte('a' + (node % alphabetLen))
		if upper {
			ch = byte('A' + (node % alphabetLen))
		}

		chars = append(chars, ch)
		node /= 26
	}

	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}

	return string(chars)
}

func romanMarker(node int, upper bool) string {
	if node < 1 {
		node = 1
	}

	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}

	var boxNode strings.Builder

	for i, v := range vals {
		for node >= v {
			boxNode.WriteString(syms[i])

			node -= v
		}
	}

	s := boxNode.String()
	if upper {
		return strings.ToUpper(s)
	}

	return s
}

// placeFloat lays out n as a float:left|right box and records it in floats.
// Consecutive same-side floats pack horizontally when width remains;
// otherwise they stack below the previous float bottom.
func (e *engine) placeFloat(
	node *html.Node, cstate ResolvedStyle, floats *floatState, contentW, contentX, posY, curY float64,
) *box {
	avail := contentW
	if cstate.Width < 0 && cstate.WidthPercent < 0 {
		avail = e.floatIntrinsicAvail(node, cstate, avail)
	}

	flowY := posY + curY

	fixX, fromY := contentX, flowY

	switch cstate.Float {
	case floatLeft, floatRight:
		fixX, fromY, avail = packFloatPosition(floats, contentX, contentW, flowY, avail, cstate.Float == floatLeft)
	}

	oldMax := e.imgMaxW
	e.setFloatImgMaxW(cstate, contentW, avail)

	fbox := e.build(node, avail, fixX, fromY)
	e.imgMaxW = oldMax

	if fbox == nil {
		return nil
	}

	if cstate.Float == floatLeft && floats.hasLeft && fbox.x+fbox.w > contentX+contentW {
		// Overflowed the pack attempt — stack below.
		fromY = maxY(floats.leftBottom, flowY)
		dx, dy := contentX-fbox.x, fromY-fbox.y
		fbox.x, fbox.y = contentX, fromY
		e.shiftBoxOps(fbox, dx, dy)
	}

	margL := e.scalePt(cstate.MarginLeft)
	margR := e.scalePt(cstate.MarginRight)

	if cstate.Float == floatRight {
		wantX := contentX + contentW - fbox.w - margR
		dx := wantX - fbox.x
		fbox.x = wantX
		e.shiftBoxOps(fbox, dx, 0)
	}

	floats.place(cstate.Float, fbox, margL, margR)

	return fbox
}

// setFloatImgMaxW clamps replaced images inside the float to its used width
// (extracted from placeFloat for clarity).
func (e *engine) setFloatImgMaxW(cs ResolvedStyle, contentW, avail float64) {
	switch {
	case cs.Width >= 0:
		e.imgMaxW = e.scalePt(cs.Width)
	case cs.WidthPercent >= 0 && contentW > 0:
		e.imgMaxW = contentW * cs.WidthPercent / cssPercent
	case avail > 0 && avail < contentW:
		e.imgMaxW = avail
	}
}

// packFloatPosition resolves where a new float starts: beside the existing
// same-side floats when there is room, otherwise below their bottom edge.
func packFloatPosition(
	floats *floatState, contentX, contentW, flowY, avail float64, isLeft bool,
) (float64, float64, float64) {
	fixX := contentX
	fromY := flowY
	packedAvail := avail

	if isLeft {
		if !floats.hasLeft {
			return fixX, fromY, packedAvail
		}

		room := floatsPackRoom(floats, contentX, contentW, true)
		if room >= avail*halfRatio { // enough room to attempt side-by-side
			fixX = floats.leftEdge
			fromY = maxY(floats.leftTop, flowY)
			packedAvail = minY(avail, room)

			return fixX, fromY, packedAvail
		}

		fromY = maxY(floats.leftBottom, flowY)

		return fixX, fromY, packedAvail
	}

	if !floats.hasRight {
		return fixX, fromY, packedAvail
	}

	room := floatsPackRoom(floats, contentX, contentW, false)
	if room >= avail*halfRatio {
		fromY = maxY(floats.rightTop, flowY)
		packedAvail = minY(avail, room)

		return fixX, fromY, packedAvail
	}

	fromY = maxY(floats.rightBottom, flowY)

	return fixX, fromY, packedAvail
}

// floatIntrinsicAvail measures the shrink-to-fit width of a float without a
// definite width: size containment, the widest descendant image, or the
// cell content max-content (plus chrome and margins).
func (e *engine) floatIntrinsicAvail(node *html.Node, style ResolvedStyle, avail float64) float64 {
	var intr float64
	if isSizeContainer(style) {
		// Size containment: intrinsic inline size as-if-empty (padding+border
		// only) so used size does not depend on descendants.
		intr = e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
			e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
			e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	} else if imgW := e.measureLargestImageWidth(node); imgW > 0 {
		// Wiki thumbs: size the float to the image, not the unwrapped
		// figcaption max-content (which letterboxed images in a wide frame).
		intr = imgW +
			e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
			e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
			e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	} else {
		intr = e.measureCellContent(node, style)
		intr += e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
			e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
			e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	}

	if intr > 0 && intr < avail {
		return intr
	}

	return avail
}

// floatsPackRoom is the horizontal room left beside existing floats for a
// new float on side isLeft.
func floatsPackRoom(floats *floatState, contentX, contentW float64, isLeft bool) float64 {
	room := contentX + contentW - floats.leftEdge
	if !isLeft {
		room = floats.rightEdge - contentX
	}

	if other := floats.rightEdge - floats.leftEdge; other < room {
		room = other
	}

	return room
}

func maxY(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

func minY(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

// shiftBoxOps translates every op in b's op range by (dx, dy).
// Deferred chrome owned by b's subtree is shifted too so finalizeChrome
// places backgrounds/borders at the post-move geometry.
func (e *engine) shiftBoxOps(boxNode *box, deltaX, deltaY float64) {
	if deltaX == 0 && deltaY == 0 || boxNode == nil {
		return
	}

	if boxNode.opEnd >= boxNode.opStart {
		for k := boxNode.opStart; k <= boxNode.opEnd && k < len(e.ops); k++ {
			e.ops[k].X += deltaX
			e.ops[k].Y += deltaY
		}
	}

	shiftDeferredChrome(e.deferredChrome, boxNode, deltaX, deltaY)
}

// shiftDeferredChrome translates the chrome ops of every deferred entry owned
// by b's subtree so finalizeChrome places backgrounds/borders at the new
// geometry.
func shiftDeferredChrome(entries []chromeEntry, boxNode *box, deltaX, deltaY float64) {
	if len(entries) == 0 {
		return
	}

	inSubtree := markBoxSubtree(boxNode)

	for idx := range entries {
		if _, ok := inSubtree[entries[idx].b]; !ok {
			continue
		}

		for j := range entries[idx].ops {
			entries[idx].ops[j].X += deltaX
			entries[idx].ops[j].Y += deltaY
		}
	}
}

// markBoxSubtree returns the set of boxes in b's subtree (b included).
func markBoxSubtree(boxNode *box) map[*box]struct{} {
	inSubtree := map[*box]struct{}{}

	var mark func(*box)
	mark = func(posX *box) {
		if posX == nil {
			return
		}

		inSubtree[posX] = struct{}{}

		for _, c := range posX.children {
			mark(c)
		}
	}
	mark(boxNode)

	return inSubtree
}

func collapseMargins(acc, boxN float64) float64 {
	if acc <= 0 && boxN <= 0 {
		return acc + boxN
	}

	if acc > boxN {
		return acc
	}

	return boxN
}

func (e *engine) emitBorders(st ResolvedStyle, x, y, w, h float64) {
	e.emitBorderLine(x, y, w, 0, e.scalePt(st.BorderTop.Width), st.BorderTop.Style,
		st.BorderTop.Color[0], st.BorderTop.Color[1], st.BorderTop.Color[2])
	e.emitBorderLine(x+w, y, 0, h, e.scalePt(st.BorderRight.Width), st.BorderRight.Style,
		st.BorderRight.Color[0], st.BorderRight.Color[1], st.BorderRight.Color[2])
	e.emitBorderLine(x, y+h, w, 0, e.scalePt(st.BorderBottom.Width), st.BorderBottom.Style,
		st.BorderBottom.Color[0], st.BorderBottom.Color[1], st.BorderBottom.Color[2])
	e.emitBorderLine(x, y, 0, h, e.scalePt(st.BorderLeft.Width), st.BorderLeft.Style,
		st.BorderLeft.Color[0], st.BorderLeft.Color[1], st.BorderLeft.Color[2])
}

// --- replaced elements ---

type imageUsedSize struct {
	w, h float64
}

// imageContainingWidth is the width used by percentage/max-width image
// constraints. imgMaxW is set by inline, float, and table-cell layout; the
// viewport is the fallback for ordinary block images.
func (e *engine) imageContainingWidth() float64 {
	if e.imgMaxW > 0 {
		return e.imgMaxW
	}

	return e.opts.Width
}

// usedImageSize is the single sizing policy for replaced images. It starts
// from intrinsic dimensions, applies HTML attributes, then CSS dimensions and
// finally max constraints while preserving the intrinsic aspect ratio for a
// one-dimensional constraint. The same helper is used by block, inline,
// float, and table intrinsic measurement paths.
func (e *engine) usedImageSize(
	node *html.Node, style ResolvedStyle, ref *imageRef,
) imageUsedSize {
	var size imageUsedSize
	if ref != nil {
		size.w = e.scalePt(pxToPt(float64(ref.w)))
		size.h = e.scalePt(pxToPt(float64(ref.h)))
	}

	attrW, attrH := e.imageAttrDims(node)
	if attrW > 0 {
		size.w = attrW
	}

	if attrH > 0 {
		size.h = attrH
	}

	size = applyImageAttrRatio(size, attrW, attrH, ref)

	cssW, cssH := style.Width >= 0, style.Height >= 0

	if style.WidthPercent >= 0 {
		if cb := e.imageContainingWidth(); cb > 0 {
			size.w = cb * style.WidthPercent / cssPercent
			cssW = true
		}
	} else if cssW {
		size.w = e.scalePt(style.Width)
	}

	if cssH {
		size.h = e.scalePt(style.Height)
	}

	size = applyImageCSSRatio(size, cssW, cssH, ref)
	size = clampImageWidth(size, e.imageMaxWidth(style, cssW))
	size = clampImageHeight(e, size, style)

	return size
}

// imageAttrDims reads scaled width/height attributes as used pixel dims.
func (e *engine) imageAttrDims(node *html.Node) (float64, float64) {
	if node == nil {
		return 0, 0
	}

	attrW := 0.0
	if v, err := strconv.Atoi(strings.TrimSpace(node.Attribute("width"))); err == nil && v > 0 {
		attrW = e.scalePt(pxToPt(float64(v)))
	}

	attrH := 0.0
	if v, err := strconv.Atoi(strings.TrimSpace(node.Attribute("height"))); err == nil && v > 0 {
		attrH = e.scalePt(pxToPt(float64(v)))
	}

	return attrW, attrH
}

// hasIntrinsic reports whether the image ref carries pixel dimensions.
func hasIntrinsic(ref *imageRef) bool {
	return ref != nil && ref.w > 0 && ref.h > 0
}

// applyImageAttrRatio fills the missing attribute dimension from the other
// attribute via the intrinsic aspect ratio.
func applyImageAttrRatio(size imageUsedSize, attrW, attrH float64, ref *imageRef) imageUsedSize {
	if !hasIntrinsic(ref) {
		return size
	}

	switch {
	case attrW > 0 && attrH == 0:
		size.h = attrW * float64(ref.h) / float64(ref.w)
	case attrH > 0 && attrW == 0:
		size.w = attrH * float64(ref.w) / float64(ref.h)
	}

	return size
}

// applyImageCSSRatio fills the missing CSS dimension from the other one via
// the intrinsic aspect ratio.
func applyImageCSSRatio(size imageUsedSize, cssW, cssH bool, ref *imageRef) imageUsedSize {
	if !hasIntrinsic(ref) {
		return size
	}

	switch {
	case cssW && !cssH:
		size.h = size.w * float64(ref.h) / float64(ref.w)
	case cssH && !cssW:
		size.w = size.h * float64(ref.w) / float64(ref.h)
	}

	return size
}

// clampImageWidth scales the size down to maxW preserving the aspect ratio.
func clampImageWidth(size imageUsedSize, maxW float64) imageUsedSize {
	if maxW >= 0 && size.w > maxW && size.w > 0 {
		factor := maxW / size.w
		size.w = maxW
		size.h *= factor
	}

	return size
}

// clampImageHeight scales the size down to max-height preserving the ratio.
func clampImageHeight(e *engine, size imageUsedSize, style ResolvedStyle) imageUsedSize {
	if style.MaxHeight < 0 {
		return size
	}

	maxH := e.scalePt(style.MaxHeight)
	if maxH >= 0 && size.h > maxH && size.h > 0 {
		factor := maxH / size.h
		size.w *= factor
		size.h = maxH
	}

	return size
}

// imageMaxWidth resolves the effective max-width constraint: CSS max-width,
// then max-width %, then the float/table/inline containing block for
// auto-sized images (a definite image width stays authoritative).
func (e *engine) imageMaxWidth(style ResolvedStyle, cssW bool) float64 {
	maxW := -1.0
	if style.MaxWidth >= 0 {
		maxW = e.scalePt(style.MaxWidth)
	}

	if style.MaxWidthPercent >= 0 {
		if cb := e.imageContainingWidth(); cb > 0 {
			pct := cb * style.MaxWidthPercent / cssPercent
			if maxW < 0 || pct < maxW {
				maxW = pct
			}
		}
	}

	if !cssW && e.imgMaxW > 0 && (maxW < 0 || e.imgMaxW < maxW) {
		maxW = e.imgMaxW
	}

	return maxW
}

func (e *engine) buildImage(n *html.Node, sty ResolvedStyle, posX, posY float64) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: n, style: e.stylePtr(n), kind: "replaced", x: posX, y: posY,
	}
	boxNode.img = e.resolveImage(n.Attribute("src"))
	size := e.usedImageSize(n, sty, boxNode.img)
	boxNode.w, boxNode.height = size.w, size.h
	// Paint replaced images that are not deferred to the inline line box.
	// Inline/inline-block <img> is collected by collectInline and painted in
	// emitLine; block-level and floated images paint here (wiki logo tagline
	// uses display:block and must stack under the wordmark).
	if boxNode.img != nil && boxNode.img.data != nil {
		inlineLevel := sty.Display == cssDisplayInline || sty.Display == cssDisplayInlineBlock ||
			sty.Display == displayInlineFlex || sty.Display == ""
		if sty.Float != cssDisplayNone || !inlineLevel {
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind:  OpImage,
				X:     posX,
				Y:     posY,
				W:     boxNode.w,
				H:     boxNode.height,
				Image: boxNode.img.data, ImgW: boxNode.img.w, ImgH: boxNode.img.h, IsJPEG: boxNode.img.isJPEG,
			})
		}
	}

	return boxNode
}

func (e *engine) buildHR(n *html.Node, sty ResolvedStyle, availW, posX, posY float64) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: n, style: e.stylePtr(n), kind: "replaced", x: posX, y: posY, w: availW,
	}
	if sty.Width >= 0 {
		boxNode.w = e.scalePt(sty.Width)
	}

	boxNode.height = e.scalePt(sty.BorderTop.Width) + e.scalePt(sty.BorderBottom.Width)
	if boxNode.height <= 0 {
		boxNode.height = 1
	}

	child := [3]float64{0, 0, 0}
	if sty.BorderTop.Style != cssDisplayNone {
		child = sty.BorderTop.Color
	}

	if boxNode.height > 0 {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: posX, Y: posY, W: boxNode.w, H: boxNode.height,
			R: child[0], G: child[1], B: child[2],
		})
	}

	return boxNode
}

// imageDims extracts pixel dimensions from PNG or JPEG bytes.
func imageDims(data []byte) (int, int, bool, bool) {
	if len(data) >= 24 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return int(binary.BigEndian.Uint32(data[16:20])), int(binary.BigEndian.Uint32(data[20:24])), false, true
	}

	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return jpegDims(data)
	}

	return 0, 0, false, false
}

// jpegDims scans JPEG segment markers for a SOF segment carrying dimensions.
// Layout matches pdf/images.go jpegScan SOF field order: after the marker and
// 2-byte length, precision (1), height (2), width (2).
func jpegDims(data []byte) (int, int, bool, bool) {
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != byteMax {
			pos++

			continue
		}

		marker := data[pos+1]
		if marker == 0xD9 || marker == 0xDA {
			return 0, 0, false, false
		}

		if isSOFMarker(marker) {
			// SOF layout: marker, 2-byte length, precision, height, width.
			// Need through width (pos+8 inclusive).
			if pos+9 > len(data) {
				return 0, 0, false, false
			}

			height := int(binary.BigEndian.Uint16(data[pos+5 : pos+7]))
			width := int(binary.BigEndian.Uint16(data[pos+7 : pos+9]))

			if width <= 0 || height <= 0 {
				return 0, 0, false, false
			}

			return width, height, true, true
		}

		segLen := int(data[pos+2])<<byteShift | int(data[pos+3])
		if segLen < jpegSegHeaderLen {
			return 0, 0, false, false
		}

		pos += jpegSegHeaderLen + segLen
	}

	return 0, 0, false, false
}

// isSOFMarker reports whether marker is a JPEG start-of-frame segment that
// carries image dimensions (skips DHT/DAC/DNL in the 0xC0..0xCF range).
func isSOFMarker(marker byte) bool {
	return marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC
}

// --- tables ---

func (e *engine) buildTable(node *html.Node, style ResolvedStyle, availW, posX, posY float64) *box {
	// flatten row groups into rows; count leading header-group rows
	rows, headerRows := e.collectTableRows(node)
	rows = stripEmptyTableRows(rows)
	headerRows = resolveHeaderRows(rows, headerRows)

	tableBox := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: displayTable, x: posX, y: posY, headerRows: headerRows,
	}
	if len(rows) == 0 {
		return tableBox
	}

	placed, nCols := placeTableCells(rows)
	if nCols == 0 {
		return tableBox
	}

	colW, colMin, colPct, colAbs, cellData := e.measureTableColumns(placed, nCols)

	// table width
	// border-collapse: collapse suppresses the separate-border gap so colspan
	// header rows and body cells share edges instead of looking double-lined.
	spacing := e.tableSpacing(style)
	chrome := spacing*float64(nCols+1) +
		e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
		e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight)

	colW, tableW := sizeTableColumns(tableColumnEnv{
		colMin: colMin, colW: colW, colPct: colPct, colAbs: colAbs,
		chrome: chrome, availW: availW, tableW: e.tableWidthHint(style, availW),
	})
	tableBox.w = tableW

	padL := e.scalePt(style.PaddingLeft) + e.scalePt(style.BorderLeft.Width)
	rowHeights, rowTops, curY := e.measureTableRows(tableBox, rows, cellData, colW, spacing, nCols, posX, posY, padL)

	tableBox.rows = cellData
	tableBox.height = curY + e.scalePt(style.PaddingBottom) + e.scalePt(style.BorderBottom.Width)

	if style.BGColor[3] > 0 && e.opts.Background {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: posX, Y: posY, W: tableBox.w, H: tableBox.height,
			R: style.BGColor[0], G: style.BGColor[1], B: style.BGColor[2], Alpha: style.BGColor[3],
		})
	}

	e.emitTableCells(tableBox, style, posX, posY, padL, colW, rowTops, rowHeights, cellData)

	return tableBox
}

// tableSpacing is the inter-cell gap: border-collapse suppresses it.
func (e *engine) tableSpacing(st ResolvedStyle) float64 {
	spacing := e.scalePt(st.BorderSpacing)
	if st.BorderCollapse != borderCollapseValue {
		return spacing
	}

	return 0
}

// emitTableCells paints the cell backgrounds/borders and the collapsed grid
// row-by-row so a row's grid segments land in the same op index span as its
// cells (pagination moves them together).
func (e *engine) emitTableCells(
	tableBox *box, sty ResolvedStyle, posX, posY, padL float64,
	colW, rowTops, rowHeights []float64, cellData [][]*box,
) {
	collapse := sty.BorderCollapse == borderCollapseValue
	// Separate borders: stroke the table box. Collapsed grids include the
	// outer perimeter — stroking both doubles the outer edge and leaves the
	// table chrome behind when only cell ops shift across pages.
	if !collapse {
		e.emitBorders(sty, posX, posY, tableBox.w, tableBox.height)
	}

	lastNonEmpty := lastNonEmptyRow(rowHeights)

	if collapse {
		for rowIdx, cells := range cellData {
			for _, cell := range cells {
				// Skip paint for collapsed empty rows (h≈0); content was
				// ink-less and would only re-inflate phantom bands.
				if cell.height > layoutSlack {
					e.emitCell(cell, true)
				}
			}

			if rowHeights[rowIdx] > layoutSlack {
				e.emitCollapsedRowGrid(tableBox, rowIdx, rowIdx == lastNonEmpty, padL, colW, rowTops, rowHeights)
			}
		}

		return
	}

	for _, cell := range tableBox.children {
		if cell.height > layoutSlack {
			e.emitCell(cell, false)
		}
	}
}

// tcell is one placed table cell: its source node, grid position and spans.
type tcell struct {
	node         *html.Node
	row, col     int
	cSpan, rSpan int
}

// collectTableRows flattens row groups into rows and counts leading
// header-group rows.
func (e *engine) collectTableRows(node *html.Node) ([][]*html.Node, int) {
	var rows [][]*html.Node

	headerRows := 0

	var collect func(n *html.Node, inHeader bool)
	collect = func(n *html.Node, inHeader bool) {
		for _, child := range n.Children {
			if child.Type != html.ElementNode {
				continue
			}

			cstate := e.styles[child]
			if cstate.Display == cssDisplayNone {
				continue
			}

			switch {
			case cstate.Display == displayTableRow:
				rows = append(rows, rowCellNodes(child, e))

				if inHeader {
					headerRows++
				}
			case cstate.Display == displayHeaderGroup:
				collect(child, true)
			case strings.HasSuffix(cstate.Display, "row-group"):
				collect(child, false)
			}
		}
	}
	collect(node, false)

	return rows, headerRows
}

// rowCellNodes returns the table-cell children of a <tr>.
func rowCellNodes(tr *html.Node, e *engine) []*html.Node {
	var cells []*html.Node

	for _, cell := range tr.Children {
		if cell.Type == html.ElementNode && e.styles[cell].Display == displayTableCell {
			cells = append(cells, cell)
		}
	}

	return cells
}

// resolveHeaderRows fixes up the thead-derived header count after empty rows
// were stripped, falling back to a leading band of <th> cells.
func resolveHeaderRows(rows [][]*html.Node, headerRows int) int {
	if headerRows > len(rows) {
		headerRows = len(rows)
	}
	// If thead contributed only empty rows, fall through to leading-th.
	if headerRows > 0 {
		// Verify leading rows still look like headers (all th); otherwise
		// the count was empty-thead noise.
		if countLeadingTHRows(rows[:headerRows]) != headerRows {
			headerRows = 0
		}
	}

	// Tables without <thead> often still use a leading row of <th> cells as
	// column headers (common HTML pattern). Treat consecutive leading all-th
	// rows as repeating headers for multi-page tables — generic, not site CSS.
	if headerRows == 0 {
		headerRows = countLeadingTHRows(rows)
	}

	return headerRows
}

// placeTableCells assigns each cell a column index honoring rowspan holes and
// discovers the column count. Returns the placed cells and nCols.
func placeTableCells(rows [][]*html.Node) ([]tcell, int) {
	var placed []tcell

	nRows := len(rows)
	occupied := make([][]int, nRows) // per-row remaining coverage counts
	nCols := 0

	for rowI, runic := range rows {
		if occupied[rowI] == nil {
			occupied[rowI] = make([]int, nCols)
		}

		rowPlaced, rowCols := placeRowCells(occupied, rowI, runic, nRows)
		placed = append(placed, rowPlaced...)

		if rowCols > nCols {
			nCols = rowCols
		}

		if len(occupied[rowI]) > nCols {
			nCols = len(occupied[rowI])
		}
	}

	// Normalize occupied rows to nCols.
	for ri := range occupied {
		for len(occupied[ri]) < nCols {
			occupied[ri] = append(occupied[ri], 0)
		}
	}

	return placed, nCols
}

// placeRowCells assigns one row's cells to columns, honoring rowspan holes.
func placeRowCells(occupied [][]int, rowI int, runic []*html.Node, nRows int) ([]tcell, int) {
	placed := make([]tcell, 0, len(runic))

	nCols := 0
	cidx := 0

	for _, cellNode := range runic {
		cstate, rowS := colSpan(cellNode), cellRowSpan(cellNode)
		if cstate < 1 {
			cstate = 1
		}

		if rowS < 1 {
			rowS = 1
		}

		for cidx < len(occupied[rowI]) && occupied[rowI][cidx] > 0 {
			cidx++
		}

		for len(occupied[rowI]) < cidx+cstate {
			occupied[rowI] = append(occupied[rowI], 0)
		}

		for k := range cstate {
			occupied[rowI][cidx+k] = rowS // covered for rs rows including this one
		}

		markRowspanCoverage(occupied, rowI, cidx, cstate, rowS, nRows)

		placed = append(placed, tcell{node: cellNode, row: rowI, col: cidx, cSpan: cstate, rSpan: rowS})

		if end := cidx + cstate; end > nCols {
			nCols = end
		}

		cidx += cstate
	}

	return placed, nCols
}

// markRowspanCoverage records that a rowspan cell covers columns
// [cidx, cidx+cstate) for rowS rows below rowI.
func markRowspanCoverage(occupied [][]int, rowI, cidx, cstate, rowS, nRows int) {
	for rowR := 1; rowR < rowS && rowI+rowR < nRows; rowR++ {
		for len(occupied[rowI+rowR]) < cidx+cstate {
			occupied[rowI+rowR] = append(occupied[rowI+rowR], 0)
		}

		for k := range cstate {
			if occupied[rowI+rowR][cidx+k] < rowS-rowR {
				occupied[rowI+rowR][cidx+k] = rowS - rowR
			}
		}
	}
}

// measureTableColumns measures each cell's min/max-content width; colspan
// cells contribute their content width evenly across the spanned columns
// (min floor per col). Returns column hints and the per-row cell boxes.
func (e *engine) measureTableColumns(
	placed []tcell, nCols int,
) ([]float64, []float64, []float64, []float64, [][]*box) {
	colW := make([]float64, nCols)   // preferred = max-content
	colMin := make([]float64, nCols) // shrink floor = min-content
	colPct := make([]float64, nCols) // >=0 means width:% of table; -1 = auto
	colAbs := make([]float64, nCols) // >=0 means absolute width pt; -1 = auto

	for i := range colPct {
		colPct[i] = -1
		colAbs[i] = -1
	}

	nRows := 0
	for _, p := range placed {
		if p.row+1 > nRows {
			nRows = p.row + 1
		}
	}

	cellData := make([][]*box, nRows)

	for _, page := range placed {
		cell := e.buildCell(page.node, page.col, page.cSpan)
		cell.row, cell.rowSpan = page.row, page.rSpan
		cellData[page.row] = append(cellData[page.row], cell)
		cstate := e.styleVal(page.node)

		switch {
		case page.cSpan == 1:
			applySingleCellColumn(cell, cstate, colW, colMin, colPct, colAbs, page.col, e)
		case page.cSpan > 1:
			distributeSpanColumns(cell, page, colW, colMin, nCols)
		}
	}

	return colW, colMin, colPct, colAbs, cellData
}

// applySingleCellColumn folds one non-spanning cell's width contribution and
// width hints into its column.
func applySingleCellColumn(
	cell *box, cstate ResolvedStyle, colW, colMin, colPct, colAbs []float64, col int, eng *engine,
) {
	if cell.contentW > colW[col] {
		colW[col] = cell.contentW
	}

	if cell.contentMin > colMin[col] {
		colMin[col] = cell.contentMin
	}

	if cstate.WidthPercent >= 0 && colPct[col] < 0 {
		colPct[col] = cstate.WidthPercent
	}

	if cstate.Width >= 0 && colAbs[col] < 0 {
		colAbs[col] = eng.scalePt(cstate.Width)
	}
}

// distributeSpanColumns spreads a colspan cell's width evenly across the
// spanned columns (min floor per col).
func distributeSpanColumns(cell *box, page tcell, colW, colMin []float64, nCols int) {
	var sumMax, sumMin float64
	for k := 0; k < page.cSpan && page.col+k < nCols; k++ {
		sumMax += colW[page.col+k]
		sumMin += colMin[page.col+k]
	}

	if cell.contentW > sumMax {
		extra := (cell.contentW - sumMax) / float64(page.cSpan)
		for k := 0; k < page.cSpan && page.col+k < nCols; k++ {
			colW[page.col+k] += extra
		}
	}

	if cell.contentMin > sumMin {
		extra := (cell.contentMin - sumMin) / float64(page.cSpan)
		for k := 0; k < page.cSpan && page.col+k < nCols; k++ {
			colMin[page.col+k] += extra
		}
	}
}

// tableWidthHint resolves the definite table border-box width hint (-1 = auto).
func (e *engine) tableWidthHint(st ResolvedStyle, availW float64) float64 {
	var hint float64 = -1 // auto
	if st.WidthPercent >= 0 {
		hint = availW * st.WidthPercent / cssPercent
	} else if st.Width >= 0 {
		hint = e.scalePt(st.Width)
		if hint > availW && availW > 0 {
			hint = availW
		}
	}

	return hint
}

// measureTableRows lays out every cell at its final column width and resolves
// row heights: single-row cells first, then rowspan growth, then final tops
// and cell heights. Returns rowHeights, rowTops and the content height.
func (e *engine) measureTableRows(
	tableBox *box, rows [][]*html.Node, cellData [][]*box, colW []float64,
	spacing float64, nCols int, posX, posY, padL float64,
) ([]float64, []float64, float64) {
	nRows := len(cellData)
	rowHeights := make([]float64, nRows)
	rowTops := make([]float64, nRows)
	curY := e.scalePt(tableBox.style.PaddingTop) + e.scalePt(tableBox.style.BorderTop.Width)
	// Measure each cell at its final column width; row height from single-row
	// cells first. Rowspan cells enlarge the spanned rows afterward.
	// Rows with no local cells (rowspan holes) or only ink-less cells stay at
	// height 0 until rowspan growth — do not invent a 1pt phantom band.
	for rowIdx, cells := range cellData {
		rowTops[rowIdx] = posY + curY
		rowH := e.measureRowCells(tableBox, cells, rowIdx, colW, spacing, nCols, posX, padL, rowTops)
		// Collapse rows whose cells have no ink (only padding/borders of empty
		// th/td). Keep a hairline only when the row has cells that paint
		// borders in separate-border mode and measured some chrome — pure
		// empty content collapses to 0 so border-collapse grids do not draw
		// a phantom empty band above real headers.
		if rowH > 0 && rowCellsHaveNoInk(rows[rowIdx]) {
			rowH = 0
		}

		rowHeights[rowIdx] = rowH

		if rowH > 0 || spacing > 0 {
			curY += rowH + spacing
		}
	}

	growRowspanRows(tableBox, nRows, rowHeights, spacing)

	// Recompute tops and assign final cell heights after rowspan growth.
	curY = e.scalePt(tableBox.style.PaddingTop) + e.scalePt(tableBox.style.BorderTop.Width)
	for rowIdx := range rowHeights {
		rowTops[rowIdx] = posY + curY
		curY += rowHeights[rowIdx] + spacing
	}

	assignFinalCellHeights(tableBox, nRows, rowHeights, rowTops, spacing)

	return rowHeights, rowTops, curY
}

// measureRowCells sizes and measures the cells of one row at their final
// column widths, returning the row height (single-row cells only).
func (e *engine) measureRowCells(
	tableBox *box, cells []*box, rowIdx int, colW []float64,
	spacing float64, nCols int, posX, padL float64, rowTops []float64,
) float64 {
	rowH := 0.0

	for _, cell := range cells {
		if cell.rowSpan < 1 {
			cell.rowSpan = 1
		}

		cellW := 0.0
		for k := 0; k < cell.span && cell.col+k < nCols; k++ {
			cellW += colW[cell.col+k]
		}

		cellW += spacing * float64(cell.span-1)
		cell.w = cellW
		cell.x = posX + padL

		for c := 0; c < cell.col && c < nCols; c++ {
			cell.x += colW[c] + spacing
		}

		cell.y = rowTops[rowIdx]
		e.measureCellHeight(cell, cellW)

		if cell.rowSpan == 1 && cell.contentH > rowH {
			rowH = cell.contentH
		}

		tableBox.children = append(tableBox.children, cell)
	}

	return rowH
}

// growRowspanRows enlarges the spanned rows so rowspan cells fit their
// content across the whole band.
func growRowspanRows(tableBox *box, nRows int, rowHeights []float64, spacing float64) {
	for _, cell := range tableBox.children {
		if cell.rowSpan <= 1 {
			continue
		}

		start := cell.row
		if start < 0 {
			continue
		}

		end := start + cell.rowSpan
		if end > nRows {
			end = nRows
		}

		sum := 0.0
		for rowIdx := start; rowIdx < end; rowIdx++ {
			sum += rowHeights[rowIdx]
			if rowIdx+1 < end {
				sum += spacing
			}
		}

		if cell.contentH > sum {
			extra := (cell.contentH - sum) / float64(end-start)
			for rowIdx := start; rowIdx < end; rowIdx++ {
				rowHeights[rowIdx] += extra
			}
		}
	}
}

// assignFinalCellHeights sets cell.y/height/rowBoxH from the resolved rows.
func assignFinalCellHeights(tb *box, nRows int, rowHeights, rowTops []float64, spacing float64) {
	for _, cell := range tb.children {
		start := cell.row
		if start < 0 {
			continue
		}

		cell.y = rowTops[start]

		rs := cell.rowSpan
		if rs < 1 {
			rs = 1
		}

		end := start + rs
		if end > nRows {
			end = nRows
		}

		height := 0.0
		for ri := start; ri < end; ri++ {
			height += rowHeights[ri]
			if ri+1 < end {
				height += spacing
			}
		}

		cell.height = height
		cell.rowBoxH = rowHeights[start]
	}
}

// lastNonEmptyRow returns the index of the last row with nonzero height.
func lastNonEmptyRow(rowHeights []float64) int {
	last := -1

	for ri := range rowHeights {
		if rowHeights[ri] > layoutSlack {
			last = ri
		}
	}

	return last
}

// emitCollapsedRowGrid strokes the shared border-collapse grid for one table
// row (top edge + verticals; bottom edge when lastRow). Ops are appended
// immediately after that row's cells and folded into the row's op range.
func (e *engine) emitCollapsedRowGrid(
	tableBox *box, rowIdx int, lastRow bool, padL float64, colW, rowTops, rowHeights []float64,
) {
	if rowIdx < 0 || rowIdx >= len(rowHeights) || rowHeights[rowIdx] <= 0.01 || len(colW) == 0 {
		return
	}

	nCols := len(colW)
	xList := gridColumnEdges(tableBox.x+padL, colW)

	yStart := rowTops[rowIdx]
	yEnd := yStart + rowHeights[rowIdx]
	gridStart := len(e.ops)
	stroke := &rowGridStroker{e: e}
	// Top edge. Skip under rowspan continuations so a multi-row Year cell is
	// not bisected mid-table; paint.capTablePageBreaks re-seals full tops for
	// page fragments where those holes look open.
	emitGridTopEdges(stroke, tableBox, rowIdx, xList, yStart)
	// Verticals only exist where an adjacent cell declares a left/right side.
	emitGridVerticals(stroke, tableBox, rowIdx, nCols, xList, yStart, yEnd)

	if lastRow {
		emitGridBottomEdges(stroke, tableBox, rowIdx, xList, yEnd)
	}

	gridEnd := len(e.ops) - 1
	if gridEnd >= gridStart && rowIdx < len(tableBox.rows) {
		expandRowOpRange(tableBox.rows[rowIdx], gridStart, gridEnd)
	}
}

// gridColumnEdges returns the x positions of the nCols+1 column boundaries.
func gridColumnEdges(left float64, colW []float64) []float64 {
	nCols := len(colW)
	xList := make([]float64, nCols+1)
	xList[0] = left

	for i := range nCols {
		xList[i+1] = xList[i] + colW[i]
	}

	return xList
}

// emitGridTopEdges strokes the row's top edge, skipping rowspan
// continuations so a multi-row cell is not bisected mid-table.
func emitGridTopEdges(stroke *rowGridStroker, tableBox *box, rowIdx int, xList []float64, yStart float64) {
	for cidx := range len(xList) - 1 {
		if rowIdx > 0 && rowspanCovers(tableBox, rowIdx-1, rowIdx, cidx) {
			continue
		}

		if side, ok := horizontalTableBorder(tableBox, rowIdx, cidx); ok {
			stroke.hline(xList[cidx], xList[cidx+1], yStart, side)
		}
	}
}

// emitGridVerticals strokes the row's vertical edges.
func emitGridVerticals(
	stroke *rowGridStroker, tableBox *box, rowIdx, nCols int, xList []float64, yStart, yEnd float64,
) {
	for cidx := 0; cidx <= nCols; cidx++ {
		if cidx > 0 && cidx < nCols && colspanCovers(tableBox, rowIdx, cidx-1, cidx) {
			continue
		}

		if side, ok := verticalTableBorder(tableBox, rowIdx, cidx); ok {
			stroke.vline(xList[cidx], yStart, yEnd, side)
		}
	}
}

// emitGridBottomEdges strokes the bottom edge of the last row.
func emitGridBottomEdges(stroke *rowGridStroker, tableBox *box, rowIdx int, xList []float64, yEnd float64) {
	for ci := range len(xList) - 1 {
		if side, ok := horizontalTableBorder(tableBox, rowIdx+1, ci); ok {
			stroke.hline(xList[ci], xList[ci+1], yEnd, side)
		}
	}
}

// rowGridStroker appends horizontal/vertical grid border ops with the shared
// engine so collapsed rows stay in the row's op span.
type rowGridStroker struct{ e *engine }

func (s *rowGridStroker) hline(x0, x1, yy float64, side border) {
	if x1-x0 <= 0 || !borderVisible(side) {
		return
	}

	s.e.emitBorderLine(x0, yy, x1-x0, 0,
		s.e.scalePt(side.Width), side.Style, side.Color[0], side.Color[1], side.Color[2])
}

func (s *rowGridStroker) vline(xx, ya, yb float64, side border) {
	if yb-ya <= 0.01 || !borderVisible(side) {
		return
	}

	s.e.emitBorderLine(xx, ya, 0, yb-ya,
		s.e.scalePt(side.Width), side.Style, side.Color[0], side.Color[1], side.Color[2])
}

func borderVisible(side border) bool {
	return side.Width > 0 && side.Style != cssDisplayNone
}

func horizontalTableBorder(tb *box, boundary, col int) (border, bool) {
	for _, cell := range tb.children {
		if cell.col > col || cell.col+cell.span <= col {
			continue
		}

		if cell.row == boundary && borderVisible(cell.style.BorderTop) {
			return cell.style.BorderTop, true
		}

		if cell.row+cell.rowSpan == boundary && borderVisible(cell.style.BorderBottom) {
			return cell.style.BorderBottom, true
		}
	}

	return border{}, false //nolint:exhaustruct // intentional zero fields
}

func verticalTableBorder(tb *box, row, boundary int) (border, bool) {
	for _, cell := range tb.children {
		if cell.row > row || cell.row+cell.rowSpan <= row {
			continue
		}

		if cell.col == boundary && borderVisible(cell.style.BorderLeft) {
			return cell.style.BorderLeft, true
		}

		if cell.col+cell.span == boundary && borderVisible(cell.style.BorderRight) {
			return cell.style.BorderRight, true
		}
	}

	return border{}, false //nolint:exhaustruct // intentional zero fields
}

// expandRowOpRange includes [start,end] paint ops in every cell of the row so
// pagination shifts that use the row's op span also move the collapsed grid.
func expandRowOpRange(row []*box, start, end int) {
	if start > end || len(row) == 0 {
		return
	}

	for _, cell := range row {
		if cell == nil {
			continue
		}

		if cell.opStart > cell.opEnd {
			// Cell emitted nothing (empty); claim the grid ops alone.
			cell.opStart, cell.opEnd = start, end

			continue
		}

		if start < cell.opStart {
			cell.opStart = start
		}

		if end > cell.opEnd {
			cell.opEnd = end
		}
	}
}

// rowspanCovers reports whether some cell occupies column ci across the
// boundary between row above and row below (so the horizontal rule is omitted).
func rowspanCovers(tb *box, above, below, cidx int) bool {
	for _, cell := range tb.children {
		if cell.rowSpan <= 1 {
			continue
		}

		start := cell.row
		if start < 0 {
			continue
		}

		if start <= above && start+cell.rowSpan > below &&
			cell.col <= cidx && cell.col+cell.span > cidx {
			return true
		}
	}

	return false
}

func colspanCovers(tableBox *box, rowIdx, leftCol, rightCol int) bool {
	if rowIdx < 0 || rowIdx >= len(tableBox.rows) {
		return false
	}

	for _, cell := range tableBox.rows[rowIdx] {
		if cell.span > 1 && cell.col <= leftCol && cell.col+cell.span > rightCol {
			return true
		}
	}

	// Rowspan continuation rows have no local cell — find covering cell.
	return rowspanCellCovers(tableBox, rowIdx, leftCol, rightCol)
}

// rowspanCellCovers reports whether a rowspan>1 cell whose vertical range
// includes ri spans columns (leftCol, rightCol).
func rowspanCellCovers(tableBox *box, rowIdx, leftCol, rightCol int) bool {
	for _, cell := range tableBox.children {
		start := cell.row
		if start < 0 {
			continue
		}

		rs := cell.rowSpan
		if rs < 1 {
			rs = 1
		}

		if start <= rowIdx && start+rs > rowIdx &&
			cell.span > 1 && cell.col <= leftCol && cell.col+cell.span > rightCol {
			return true
		}
	}

	return false
}

// buildCell measures a table cell's min/max-content width (no ops emitted).
// Height is not final here: layoutCell must run again with the real column
// width after column sizing, or narrow max-content widths force false wraps
// and inflate row heights (empty bands under single-line cell text).
func (e *engine) buildCell(n *html.Node, col, span int) *box {
	st := e.stylePtr(n)
	b := &box{node: n, style: st, kind: "cell", col: col, span: span} //nolint:exhaustruct // intentional zero fields
	b.contentMin, b.contentW = e.measureCellMinMax(n, *st)

	return b
}

// measureCellHeight lays out the cell at width (border-box) without emitting
// paint ops, and stores the result on b.contentH.
func (e *engine) measureCellHeight(boxNode *box, width float64) {
	was := e.noEmit
	e.noEmit = true
	// Preserve the caller's noEmit flag. Nested tables call this during an
	// outer measure pass; restoring false mid-measure leaked ops at wrong
	// positions (fixture-10 nested table borders/text).
	boxNode.contentH = e.layoutCell(boxNode.node, *boxNode.style, width)
	e.noEmit = was
}

// cellBG returns the background to paint for a cell: the cell's own color,
// or the parent table-row's background when the cell is transparent (CSS
// does not inherit background, but row backgrounds show through empty
// cells in browsers — required for tr.good / tr.warn / tr.bad).
func (e *engine) cellBG(cell *box) (float64, float64, float64, float64, bool) {
	style := cell.style
	if style.BGColor[3] > 0 {
		return style.BGColor[0], style.BGColor[1], style.BGColor[2], style.BGColor[3], true
	}

	if cell.node != nil && cell.node.Parent != nil {
		ps, has := e.styles[cell.node.Parent]
		if has && ps.Display == displayTableRow && ps.BGColor[3] > 0 {
			return ps.BGColor[0], ps.BGColor[1], ps.BGColor[2], ps.BGColor[3], true
		}
	}

	return 0, 0, 0, 0, false
}

// emitCell paints a placed cell's background, borders and content.
// skipBorders is set for border-collapse tables whose grid is stroked once
// by the parent table (avoids doubled/gapped per-cell edges).
func (e *engine) emitCell(cell *box, skipBorders bool) {
	sty := *cell.style
	start := len(e.ops)

	if e.opts.Background {
		if r, g, bl, a, ok := e.cellBG(cell); ok {
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpFillRect, X: cell.x, Y: cell.y, W: cell.w, H: cell.height,
				R: r, G: g, B: bl, Alpha: a,
			})
		}
	}

	if !skipBorders {
		e.emitBorders(sty, cell.x, cell.y, cell.w, cell.height)
	}

	curX, contentW := e.contentBox(cell.x, cell.w, sty)
	curY := cell.y + e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)
	curY = cellVerticalAlignOffset(cell, curY)
	// flowChildren advances cy; cell content is rooted at absolute canvas y
	// (pass y=0, contentX=cx, cy=content top) so floats pack inside the cell
	// BFC. Pass the cell as parent so float/block children attach for tests.
	oldMax := e.imgMaxW

	if contentW > 0 {
		e.imgMaxW = contentW
	}

	enclose := e.pushBFCFloats(sty, curX, contentW)
	_ = e.flowChildren(cell, cell.node.Children, sty, contentW, curX, 0, curY)

	if enclose && e.bfcFloats != nil {
		// Cell border box already sized; floats are clipped to the cell BFC.
		_ = e.bfcFloats.extentCy(0, curY)
	}

	e.popBFCFloats(enclose)

	e.imgMaxW = oldMax
	// Rowspan cells with forced multi-line content (wiki Ref: [127]<br>[128]
	// in rowspan=2) pack lines at the top with normal line-height, so both
	// markers sit in the first row band and look overlapped. Spread line
	// boxes evenly across the full cell height when we have room.
	if cell.rowSpan > 1 {
		distributeRowspanLines(e.ops, start, len(e.ops), cell.y, cell.height,
			e.scalePt(sty.PaddingTop)+e.scalePt(sty.BorderTop.Width),
			e.scalePt(sty.PaddingBottom)+e.scalePt(sty.BorderBottom.Width))
	}

	cell.opStart, cell.opEnd = start, len(e.ops)-1
}

// cellVerticalAlignOffset shifts the content origin within the row box for
// vertical-align middle/bottom table cells.
func cellVerticalAlignOffset(cell *box, curY float64) float64 {
	extra := cell.height - cell.contentH
	if extra <= 0 {
		return curY
	}

	switch cell.style.VerticalAlign {
	case cssVerticalAlignMiddle:
		return curY + extra/two
	case cssVerticalAlignBottom:
		return curY + extra
	default:
		return curY
	}
}

// distributeRowspanLines remaps distinct text/bullet baselines in ops[start:end)
// so they span the cell's content box evenly (top line near top, bottom near
// bottom). Non-text ops (underlines, links) ride with the nearest baseline.
func distributeRowspanLines(ops []Op, start, end int, cellY, cellH, padTop, padBot float64) {
	if end <= start || cellH <= 0 || ops == nil {
		return
	}

	const yEps = 0.75

	bands := collectTextBands(ops, start, end, yEps)

	if len(bands) < two {
		return
	}
	// Sort bands top→bottom.
	sortBandsTopDown(bands)

	innerTop := cellY + padTop
	innerBot := cellY + cellH - padBot

	if innerBot-innerTop < minBoxPt {
		return
	}
	// Only redistribute when natural packing is much shorter than the cell
	// (typical rowspan>1 with few <br> lines).
	natural := bands[len(bands)-1].y - bands[0].y
	if natural >= (innerBot-innerTop)*0.55 {
		return
	}

	targets := interpolatedBandTargets(ops, bands, innerTop, innerBot)
	if targets == nil {
		return
	}
	// Map old baseline → dy, apply to all ops near that baseline.
	shifts := make([]bandShift, len(bands))
	for i, b := range bands {
		shifts[i] = bandShift{y0: b.y, dy: targets[i] - b.y}
	}

	applyBandShifts(ops, start, end, shifts, bandEmSize(ops, bands[0].idx))
}

// bandEmSize estimates the em size from the first band's text ops.
func bandEmSize(ops []Op, indices []int) float64 {
	emSize := 8.0

	for _, i := range indices {
		if ops[i].Size > 0 {
			emSize = ops[i].Size

			break
		}
	}

	return emSize
}

// applyBandShifts moves every op onto the baseline of the nearest band.
func applyBandShifts(ops []Op, start, end int, shifts []bandShift, emSize float64) {
	for idx := start; idx < end && idx < len(ops); idx++ {
		posY := ops[idx].Y
		// Nearest band baseline.
		best, bestD := 0, math.Abs(posY-shifts[0].y0)

		for si := 1; si < len(shifts); si++ {
			d := math.Abs(posY - shifts[si].y0)
			if d < bestD {
				bestD, best = d, si
			}
		}

		if bestD <= emSize*1.5 {
			ops[idx].Y += shifts[best].dy
		}
	}
}

// band is a group of op indices sharing a baseline Y (average kept coherent).
type band struct {
	y   float64
	idx []int
}

// bandShift maps an old baseline to the delta that lands it on its target.
type bandShift struct{ y0, dy float64 }

// collectTextBands groups text/bullet ops into baseline bands.
func collectTextBands(ops []Op, start, end int, yEps float64) []band {
	var bands []band

	for idx := start; idx < end && idx < len(ops); idx++ {
		paintOp := ops[idx]
		if paintOp.Kind != OpText && paintOp.Kind != OpBullet {
			continue
		}

		placed := false

		for bi := range bands {
			if math.Abs(bands[bi].y-paintOp.Y) <= yEps {
				bands[bi].idx = append(bands[bi].idx, idx)
				// Keep average Y so multi-glyph lines stay coherent.
				n := float64(len(bands[bi].idx))
				bands[bi].y = (bands[bi].y*(n-1) + paintOp.Y) / n
				placed = true

				break
			}
		}

		if !placed {
			bands = append(bands, band{y: paintOp.Y, idx: []int{idx}})
		}
	}

	return bands
}

// sortBandsTopDown sorts bands by Y ascending.
func sortBandsTopDown(bands []band) {
	for i := 0; i < len(bands); i++ {
		for j := i + 1; j < len(bands); j++ {
			if bands[j].y < bands[i].y {
				bands[i], bands[j] = bands[j], bands[i]
			}
		}
	}
}

// interpolatedBandTargets places the first baseline ~0.7em into the cell, the
// last near the bottom, and interpolates the rest; nil when the cell is too
// small to redistribute.
func interpolatedBandTargets(ops []Op, bands []band, innerTop, innerBot float64) []float64 {
	if len(bands) == 1 {
		return nil
	}
	// Use first text size as em estimate.
	emSize := 8.0

	for _, i := range bands[0].idx {
		if ops[i].Size > 0 {
			emSize = ops[i].Size

			break
		}
	}

	first := innerTop + emSize*firstLineEm
	last := innerBot - emSize*baselineInsetRatio

	if last <= first {
		return nil
	}

	targets := make([]float64, len(bands))
	for i := range bands {
		targets[i] = first + (last-first)*float64(i)/float64(len(bands)-1)
	}

	return targets
}

// measureCellContent returns the max-content border-box width of the cell
// (longest unwrapped line, not longest word). Using min-content here made
// auto tables shrink-wrap to a rivulet of columns and inflate row heights
// via forced wraps (wiki filmography / any dense multi-column table).
func (e *engine) measureCellContent(n *html.Node, st ResolvedStyle) float64 {
	minW, maxW := e.measureCellMinMax(n, st)
	_ = minW

	return maxW
}

// measureCellMinMax returns min-content and max-content border-box widths.
// min-content ≈ longest unbreakable word; max-content ≈ widest line if soft
// wraps are not taken (hard breaks from <br>/blocks still split lines).
// When word-break/overflow-wrap allow breaking long tokens, min-content uses
// the longest soft segment (or widest single rune) so URL-heavy cells can
// shrink instead of forcing the table past the page edge.
//
// Short nowrap-only lines (wiki Ref cells with [127][128]) use the full line
// as min-content so adjacent cite markers stay on one horizontal line instead
// of wrapping into a stacked, overlapping pair in a one-marker-wide column.
func (e *engine) measureCellMinMax(node *html.Node, style ResolvedStyle) (float64, float64) {
	cellMeas := &cellMeasure{ //nolint:exhaustruct // zero fields are the flushed-line state
		engine: e,
		em:     style.FontSize,
		style:  style,
	}
	cellMeas.walk(node, style, style.WhiteSpace == cssWhiteSpaceNowrap || style.WhiteSpace == cssWhiteSpacePre)
	cellMeas.flushLine()

	chrome := e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
		e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width)
	minW := cellMeas.longestWord + chrome
	maxW := cellMeas.maxW + chrome

	if maxW < minW {
		maxW = minW
	}

	return minW, maxW
}

// cellMeasure accumulates min/max-content width contributions while walking a
// table cell's subtree (see measureCellMinMax).
type cellMeasure struct {
	engine         *engine
	style          ResolvedStyle
	em             float64
	lineW          float64
	maxW           float64
	longestWord    float64
	lineOnlyNowrap bool
	lineHasInk     bool
}

// flushLine folds the current line into maxW and resets the line state.
func (m *cellMeasure) flushLine() {
	if m.lineW > m.maxW {
		m.maxW = m.lineW
	}

	if m.lineHasInk && m.lineOnlyNowrap && m.lineW > m.longestWord {
		// Cap so a pathological nowrap paragraph does not freeze the table
		// at max-content; multi-cite clusters are well under ~10em.
		em := m.em
		if em < 1 {
			em = 10
		}

		if m.lineW <= em*10*m.engine.scale {
			m.longestWord = m.lineW
		}
	}

	m.lineW = 0
	m.lineOnlyNowrap = true
	m.lineHasInk = false
}

// walk measures one node's contribution to the current line.
func (m *cellMeasure) walk(nodeN *html.Node, cstate ResolvedStyle, nowrap bool) {
	switch nodeN.Type {
	case html.TextNode:
		m.measureText(nodeN.Text, cstate, nowrap)
	case html.ElementNode:
		m.measureElement(nodeN, cstate, nowrap)
	case html.CommentNode, html.DoctypeNode:
		return
	}
}

// measureText accumulates a text run into the current line, using the same
// face selection as paint (measureTextFace) — mismatched metrics undersize
// columns and force emergency wraps on words that should fit.
func (m *cellMeasure) measureText(text string, cstate ResolvedStyle, nowrap bool) {
	eng := m.engine

	if !nowrap {
		// Walk words without strings.Fields: no []string or word copies.
		// Matching white-space:normal — runs of HTML space collapse to one gap.
		if !hasNonHTMLSpace(text) {
			return
		}

		m.lineOnlyNowrap = false
		m.lineHasInk = true

		spaceW := eng.measureTextFace(" ", cstate)

		// Leading space if original had leading WS and line already started.
		if m.lineW > 0 && len(text) > 0 && isHTMLSpace(text[0]) {
			m.lineW += spaceW
		}

		i := 0
		first := true

		for i < len(text) {
			for i < len(text) && isHTMLSpace(text[i]) {
				i++
			}

			if i >= len(text) {
				break
			}

			j := i
			for j < len(text) && !isHTMLSpace(text[j]) {
				j++
			}

			word := text[i:j]
			if !first {
				m.lineW += spaceW
			}

			first = false
			m.lineW += eng.measureTextFace(word, cstate)
			m.noteWord(eng.minContentWidth(word, cstate))
			i = j
		}

		return
	}

	m.lineW += eng.measureTextFace(text, cstate)
	m.noteWord(eng.minContentWidth(text, cstate))

	if hasNonHTMLSpace(text) {
		m.lineHasInk = true
	}
}

// noteWord records a token's min-content width.
func (m *cellMeasure) noteWord(uw float64) {
	if uw > m.longestWord {
		m.longestWord = uw
	}
}

// measureElement handles br, replaced images, and block-level in-cell boxes.
func (m *cellMeasure) measureElement(nodeN *html.Node, childCS ResolvedStyle, nowrap bool) {
	if childCS.Display == cssDisplayNone {
		return
	}

	if nodeN.Name == "br" {
		m.flushLine()

		return
	}
	// Replaced images contribute their used CSS-pixel width (wiki thumbs).
	if nodeN.Name == cssTagImg {
		innerW := m.engine.measureImageWidth(nodeN, childCS)
		m.noteWord(innerW)
		m.lineOnlyNowrap = false
		m.lineHasInk = true
		m.lineW += innerW

		return
	}
	// Block-level in-cell boxes start a new line (simplified).
	blockish := isCellBlockish(childCS.Display)
	m.walkBlockChildren(nodeN, childCS, nowrap, blockish)
}

// isCellBlockish reports displays that break the current measured line.
func isCellBlockish(display string) bool {
	switch display {
	case displayBlock, displayTable, displayListItem, displayFlex, displayGrid:
		return true
	default:
		return false
	}
}

// walkBlockChildren walks an element's children, flushing the line before and
// after block-level boxes.
func (m *cellMeasure) walkBlockChildren(nodeN *html.Node, childCS ResolvedStyle, nowrap, blockish bool) {
	if blockish {
		m.flushLine()
	}

	childNowrap := nowrap || childCS.WhiteSpace == cssWhiteSpaceNowrap || childCS.WhiteSpace == cssWhiteSpacePre
	for _, child := range nodeN.Children {
		m.walk(child, childCS, childNowrap)
	}

	if blockish {
		m.flushLine()
	}
}

// wordBreakPolicy is the single table for "how may a token split?" —
// white-space, word-break and overflow-wrap combine into one enum.
// Shared by intrinsic min-content measurement and inline overflow packing.
type wordBreakPolicy int

const (
	breakNormal wordBreakPolicy = iota
	breakAll                    // word-break:break-all / overflow-wrap:anywhere
	breakWord                   // overflow-wrap:break-word (soft only)
	breakNever                  // white-space:nowrap|pre
)

func wordBreakOf(sty ResolvedStyle) wordBreakPolicy {
	if sty.WhiteSpace == cssWhiteSpaceNowrap || sty.WhiteSpace == cssWhiteSpacePre {
		return breakNever
	}

	if sty.WordBreak == "break-all" || sty.OverflowWrap == overflowWrapAnywhere {
		return breakAll
	}

	if sty.OverflowWrap == overflowWrapBreakWord {
		return breakWord
	}

	return breakNormal
}

// softModeOf maps a break policy to the soft-wrap rune table used by
// breakToken / splitTextToWidth. Emergency (breakNormal) uses URL-ish
// opportunities only so ordinary hyphenated words stay intact.
func softModeOf(pol wordBreakPolicy) softBreakMode {
	switch pol {
	case breakAll:
		return softBreakNone
	case breakWord:
		return softBreakWord
	case breakNever:
		return softBreakURL
	case breakNormal:
		return softBreakURL
	}

	return softBreakURL
}

// minContentWidth is the min-content contribution of a single token under
// the element's word-break / overflow-wrap policy (CSS min-content).
// Emergency print wrapping (tokens wider than the used line) is layout-only
// and must not shrink table column mins to a single rune.
func (e *engine) minContentWidth(cssSheet string, sty ResolvedStyle) float64 {
	if cssSheet == "" {
		return 0
	}

	full := e.measureTextFace(cssSheet, sty)

	switch wordBreakOf(sty) {
	case breakNever:
		return full
	case breakAll:
		return e.maxRuneWidth(cssSheet, sty)
	case breakWord:
		// Soft opportunities (/, ?, &, …) split the token for min-content.
		return e.maxSoftSegmentWidth(cssSheet, sty)
	case breakNormal:
		return full
	}

	return full
}

func (e *engine) maxRuneWidth(s string, st ResolvedStyle) float64 {
	var widest float64

	for _, r := range s {
		w := e.measureRuneFace(r, st)
		if w > widest {
			widest = w
		}
	}

	return widest
}

func (e *engine) maxSoftSegmentWidth(cssS string, sty ResolvedStyle) float64 {
	if cssS == "" {
		return 0
	}

	var widest, cur float64

	for _, r := range cssS {
		cur += e.measureRuneFace(r, sty)
		// Soft-break runes end a min-content segment; residual after the
		// last break is flushed below (covers tokens with no soft points).
		if isSoftWrapRune(r, softBreakWord) {
			if cur > widest {
				widest = cur
			}

			cur = 0
		}
	}

	if cur > widest {
		widest = cur
	}

	if widest <= 0 {
		return e.maxRuneWidth(cssS, sty)
	}

	return widest
}

// countLeadingTHRows returns how many consecutive leading rows are composed
// entirely of <th> cells (column header band without an explicit <thead>).
// Empty rows (no cells) are skipped so a leading blank tr does not block
// detection of the real header band.
func countLeadingTHRows(rows [][]*html.Node) int {
	node := 0

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		allTH := true

		for _, cell := range row {
			if cell == nil || cell.Name != "th" {
				allTH = false

				break
			}
		}

		if !allTH {
			break
		}

		node++
	}

	return node
}

// stripEmptyTableRows removes rows that have no table-cell elements.
// Safe: rowspan placement only creates geometry for rows that exist in the
// source list; empty tr nodes never start cells and only produced phantom
// min-height bands in the border-collapse grid.
func stripEmptyTableRows(rows [][]*html.Node) [][]*html.Node {
	if len(rows) == 0 {
		return rows
	}

	out := rows[:0]

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		out = append(out, row)
	}
	// If every row was empty, return a fresh empty slice (not a shared buf).
	if len(out) == 0 {
		return nil
	}

	return out
}

// rowCellsHaveNoInk reports whether every cell in the row is free of text,
// images, and other non-whitespace content (padding-only empty th/td).
func rowCellsHaveNoInk(cells []*html.Node) bool {
	if len(cells) == 0 {
		return true
	}

	for _, c := range cells {
		if c == nil {
			continue
		}

		if nodeHasTableInk(c) {
			return false
		}
	}

	return true
}

func nodeHasTableInk(node *html.Node) bool {
	if node == nil {
		return false
	}

	switch node.Type {
	case html.TextNode:
		return strings.TrimSpace(node.Text) != ""
	case html.ElementNode:
		if isTableInkElement(node.Name) {
			return true
		}

		for _, child := range node.Children {
			if nodeHasTableInk(child) {
				return true
			}
		}
	case html.CommentNode, html.DoctypeNode:
		return false
	}

	return false
}

// isTableInkElement reports element names that carry ink without text.
func isTableInkElement(name string) bool {
	switch name {
	case cssTagImg, "svg", "video", "canvas", "br":
		return true
	}

	return false
}

func isHTMLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// hasNonHTMLSpace reports that s contains at least one non-HTML-whitespace byte.
// Used instead of strings.TrimSpace(s) != "" to avoid the TrimSpace string header.
func hasNonHTMLSpace(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isHTMLSpace(s[i]) {
			return true
		}
	}

	return false
}

// measureImageWidth returns the same used content width that buildImage and
// inline painting use. Keeping intrinsic/attribute/CSS/max/containing-block
// policy in usedImageSize prevents float/table shrink-to-fit from disagreeing
// with the eventually painted image.
func (e *engine) measureImageWidth(n *html.Node, st ResolvedStyle) float64 {
	if n == nil {
		return 0
	}

	return e.usedImageSize(n, st, e.resolveImage(n.Attribute("src"))).w
}

// measureLargestImageWidth walks n for the widest descendant <img>.
func (e *engine) measureLargestImageWidth(node *html.Node) float64 {
	if node == nil {
		return 0
	}

	var best float64

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Name == cssTagImg {
				st := e.styleVal(node)
				if w := e.measureImageWidth(node, st); w > best {
					best = w
				}
			}

			for _, c := range node.Children {
				walk(c)
			}
		}
	}
	walk(node)

	return best
}

// layoutCell measures the height of a cell's content (no ops emitted).
func (e *engine) layoutCell(n *html.Node, sty ResolvedStyle, width float64) float64 {
	_, contentW := e.contentBox(0, width, sty)
	curY := e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)
	enclose := e.pushBFCFloats(sty, 0, contentW)
	curY = e.flowChildren(nil, n.Children, sty, contentW, 0, 0, curY)

	if enclose && e.bfcFloats != nil {
		curY = e.bfcFloats.extentCy(0, curY)
	}

	e.popBFCFloats(enclose)

	return curY + e.scalePt(sty.PaddingBottom) + e.scalePt(sty.BorderBottom.Width)
}

func colSpan(n *html.Node) int {
	if v, err := strconv.Atoi(strings.TrimSpace(n.Attribute("colspan"))); err == nil && v > 1 {
		return v
	}

	return 1
}

func cellRowSpan(n *html.Node) int {
	if v, err := strconv.Atoi(strings.TrimSpace(n.Attribute("rowspan"))); err == nil && v > 1 {
		return v
	}

	return 1
}

// tableColumnEnv is everything the column-sizing pass needs; no DOM, no ops.
type tableColumnEnv struct {
	colMin []float64 // min-content per column (content only)
	colW   []float64 // max-content per column (content only); mutated to used widths
	colPct []float64 // width:% hints, -1 = auto
	colAbs []float64 // width:pt hints, -1 = auto
	chrome float64   // spacing + border + padding
	availW float64
	// tableW is a definite table border-box width when >= 0; -1 means auto.
	tableW float64
}

// sizeTableColumns resolves used column widths and the table border-box width
// (CSS2.1-lite auto/fixed: sum, % and abs hints, min floors, definite scaling).
func sizeTableColumns(env tableColumnEnv) ([]float64, float64) {
	colW := env.colW
	colMin := env.colMin
	colPct := env.colPct
	colAbs := env.colAbs
	chrome := env.chrome
	availW := env.availW

	nCols := len(colW)
	if nCols == 0 {
		if env.tableW >= 0 {
			return colW, env.tableW
		}

		return colW, availW
	}

	sumMax, sumMin := columnWidthSums(colW, colMin, chrome)

	definiteTable := env.tableW >= 0
	tableW := env.tableW

	if !definiteTable {
		tableW = availW
		if sumMax < availW {
			// width:auto — shrink-wrap to max-content (not min-content).
			tableW = sumMax
		}
	}

	distributeColumnWidths(colW, colMin, colPct, colAbs, tableW, chrome, definiteTable, sumMax, sumMin, nCols)

	// Auto tables: border box covers used columns. Definite width keeps tableW.
	if !definiteTable {
		if sumCols := chrome + sumColWidths(colW); sumCols > tableW {
			tableW = sumCols
		}
	}

	return colW, tableW
}

// distributeColumnWidths applies the hint/extra/shrink strategy to the used
// column widths (extracted from sizeTableColumns for clarity).
func distributeColumnWidths(
	colW, colMin, colPct, colAbs []float64, tableW, chrome float64,
	definiteTable bool, sumMax, sumMin float64, nCols int,
) {
	switch {
	case definiteTable && hasColumnHints(colPct, colAbs):
		distributeColumnHints(colW, colMin, colPct, colAbs, tableW, chrome, nCols)
	case tableW > sumMax:
		distributeColumnExtra(colW, tableW, sumMax, nCols)
	case tableW < sumMax:
		distributeColumnShrink(colW, colMin, sumMax, sumMin, tableW, chrome, definiteTable)
	}
}

// columnWidthSums returns the max-content and min-content sums plus chrome.
func columnWidthSums(colW, colMin []float64, chrome float64) (float64, float64) {
	sumMax, sumMin := 0.0, 0.0

	for i := range colW {
		sumMax += colW[i]
		sumMin += colMin[i]
	}

	return sumMax + chrome, sumMin + chrome
}

// sumColWidths returns the sum of the used column widths.
func sumColWidths(colW []float64) float64 {
	sum := 0.0
	for i := range colW {
		sum += colW[i]
	}

	return sum
}

// hasColumnHints reports whether any column has a % or absolute width hint.
func hasColumnHints(colPct, colAbs []float64) bool {
	for i := range colPct {
		if colPct[i] >= 0 || colAbs[i] >= 0 {
			return true
		}
	}

	return false
}

// distributeColumnHints resolves a definite table width with % / absolute
// column hints: hinted columns take their share first, then the leftover is
// spread over auto columns (or by % share when every column is hinted).
func distributeColumnHints(colW, colMin, colPct, colAbs []float64, tableW, chrome float64, nCols int) {
	inner := tableW - chrome
	if inner < 0 {
		inner = 0
	}

	used, autoMax := applyHintedColumns(colW, colMin, colPct, colAbs, inner)

	remain := inner - used
	if remain < 0 {
		remain = 0
	}

	switch {
	case autoMax > 0 && remain > 0:
		spreadRemainderOverAuto(colW, colMin, colPct, colAbs, remain, autoMax)
	case autoMax == 0 && remain > 0:
		// All columns hinted — distribute leftover by % share, else evenly.
		spreadRemainderOverHinted(colW, colPct, remain, nCols)
	}
}

// applyHintedColumns sizes the hinted columns and returns the used width plus
// the total max-content of the auto columns (extracted from
// distributeColumnHints for readability).
func applyHintedColumns(colW, colMin, colPct, colAbs []float64, inner float64) (float64, float64) {
	used, autoMax := 0.0, 0.0

	for idx := range colW {
		switch {
		case colPct[idx] >= 0:
			colW[idx] = maxF(inner*colPct[idx]/cssPercent, colMin[idx])
			used += colW[idx]
		case colAbs[idx] >= 0:
			colW[idx] = maxF(colAbs[idx], colMin[idx])
			used += colW[idx]
		default:
			autoMax += colW[idx]
		}
	}

	return used, autoMax
}

// spreadRemainderOverAuto distributes leftover width over auto columns
// proportionally to their current share (min floor per column).
func spreadRemainderOverAuto(colW, colMin, colPct, colAbs []float64, remain, autoMax float64) {
	for i := range colW {
		if colPct[i] < 0 && colAbs[i] < 0 {
			colW[i] = maxF(remain*(colW[i]/autoMax), colMin[i])
		}
	}
}

// spreadRemainderOverHinted distributes leftover width by % share, else evenly.
func spreadRemainderOverHinted(colW, colPct []float64, remain float64, nCols int) {
	pctTotal := 0.0

	for i := range colPct {
		if colPct[i] > 0 {
			pctTotal += colPct[i]
		}
	}

	for i := range colW {
		if pctTotal > 0 && colPct[i] > 0 {
			colW[i] += remain * (colPct[i] / pctTotal)
		} else {
			colW[i] += remain / float64(nCols)
		}
	}
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

// distributeColumnExtra spreads a surplus evenly across all columns.
func distributeColumnExtra(colW []float64, tableW, sumMax float64, nCols int) {
	extra := (tableW - sumMax) / float64(nCols)
	for i := range colW {
		colW[i] += extra
	}
}

// distributeColumnShrink squeezes columns when the used max-content width
// overflows the table: grow from min toward max, scale mins into a definite
// box, or honor mins for auto tables (which may overflow).
func distributeColumnShrink(colW, colMin []float64, sumMax, sumMin, tableW, chrome float64, definiteTable bool) {
	innerAvail := tableW - chrome
	if innerAvail < 0 {
		innerAvail = 0
	}

	innerMax := sumMax - chrome
	innerMin := sumMin - chrome

	switch {
	case innerAvail >= innerMin && innerMax > innerMin:
		growFromMinTowardMax(colW, colMin, innerAvail, innerMin, innerMax)
	case innerMin > 0 && innerAvail < innerMin:
		// Narrower than min-content.
		if definiteTable {
			scaleMinsIntoBox(colW, colMin, innerAvail, innerMin)
		} else {
			// width:auto — honor mins (table may overflow) rather than
			// crushing text into emergency mid-word wraps.
			copy(colW, colMin)
		}
	case innerMax > 0:
		scaleToInnerWidth(colW, colMin, innerAvail, innerMax, definiteTable)
	}
}

// growFromMinTowardMax grows each column from min toward max proportionally
// to its free space.
func growFromMinTowardMax(colW, colMin []float64, innerAvail, innerMin, innerMax float64) {
	free := innerAvail - innerMin
	span := innerMax - innerMin

	for idx := range colW {
		grow := colW[idx] - colMin[idx]
		if grow < 0 {
			grow = 0
		}

		colW[idx] = colMin[idx] + free*(grow/span)
	}
}

// scaleMinsIntoBox scales the column mins into a definite, narrower box so
// max-width:100% images in a 22em float still shrink.
func scaleMinsIntoBox(colW, colMin []float64, innerAvail, innerMin float64) {
	scale := innerAvail / innerMin
	if scale < 0 {
		scale = 0
	}

	for i := range colW {
		colW[i] = colMin[i] * scale
	}
}

// scaleToInnerWidth scales columns down to the available inner width.
func scaleToInnerWidth(colW, colMin []float64, innerAvail, innerMax float64, definiteTable bool) {
	scale := innerAvail / innerMax
	if scale < 0 {
		scale = 0
	}

	for i := range colW {
		colW[i] *= scale
		if !definiteTable && colW[i] < colMin[i] {
			colW[i] = colMin[i]
		}
	}
}

// DeactivateOp marks an op so every painter (Paint, PaintBand) and every
// pagination helper ignores it while keeping its slot in Ops: the box tree
// stores op indices (opStart/opEnd) that Paint relies on, so entries must
// not be removed.
const opKindNoop OpKind = 255

func DeactivateOp(paintOp *Op) {
	if paintOp == nil {
		return
	}

	paintOp.Kind = opKindNoop
	paintOp.URI = ""
}
