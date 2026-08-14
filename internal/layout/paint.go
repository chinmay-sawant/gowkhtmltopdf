package layout

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	"gowkhtmltopdf/internal/errs"
	"gowkhtmltopdf/internal/pdf"
)

var errNilContext = errs.ErrNilContext

// Page-break keyword constants shared by the pagination passes.
const (
	pageBreakAvoid  = "avoid"
	pageBreakAlways = "always"
)

// rowChromeCap bounds the initial row-chrome candidate slice; a snap touches
// one row's fills (a handful) regardless of display-list size.
const rowChromeCap = 4

// rowChromeBandTolerance is the widest band row chrome can sit in above a
// snapped op: candidate fills are at most 40pt tall and may reach 2pt past
// the op's top.
const rowChromeBandTolerance = 42

// splitSlackPerCrossing is the per-op headroom reserved for page-split
// fragments; a crossing rect yields at least two fragments.
const splitSlackPerCrossing = 2

// PaintOptions describes the destination page geometry, in points.
type PaintOptions struct {
	PageWidth    float64
	PageHeight   float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
}

// Paint paginates the display list across pages and paints it into doc.
//
// Pagination is box-aware (phase 05): ops are placed on the page containing
// their top edge; rect-type ops that cross a page boundary are split at the
// boundary; text, images and links move wholly (text ops are already
// line-level). Page-break policies are honored: page-break-before/after:
// always, page-break-inside: avoid (best-effort, box must fit one page),
// aside callouts that overflow remaining Y are lifted to the next page, and
// table rows never split.
//
// After pagination Paint fills res.Pages (page → op indices) and res.Locations
// (element boxes in document order with their page and canvas rect).
// beginPaintContext is the single context-normalization boundary owned by
// this painting slice. Both legacy and cancellation-aware entrypoints use it,
// so nil-context compatibility does not leak into the paint implementation.
func beginPaintContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func Paint(doc *pdf.Document, res *Result, opts PaintOptions) error {
	return PaintContext(context.Background(), doc, res, opts)
}

// PaintContext is the cancellation-aware form of Paint. The legacy Paint
// entrypoint remains a background-context adapter for package callers that do
// not have a request context.
//
//nolint:cyclop,funlen // paint initialization and pagination coordination
func PaintContext(ctx context.Context, doc *pdf.Document, res *Result, opts PaintOptions) error {
	if ctx == nil {
		return errNilContext
	}

	ctx, cancel := beginPaintContext(ctx)
	defer cancel()

	if doc == nil || res == nil {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("layout: paint context: %w", err)
	}

	contentH := opts.PageHeight - opts.MarginTop - opts.MarginBottom
	if contentH <= 0 {
		contentH = opts.PageHeight
	}

	if err := validatePaintPageIndices(res.Ops, contentH); err != nil {
		return err
	}

	if len(res.Ops) == 0 {
		doc.AddPage(opts.PageWidth, opts.PageHeight)
		populateLocations(res, contentH, nil)

		return nil
	}

	opPage := paginateOps(res, contentH)
	stretchPaginatedChrome(res)

	if err := validatePaintPageIndices(res.Ops, contentH); err != nil {
		return err
	}

	fixedIdx := fixedOpIndices(res)

	// Split rect ops at page boundaries first so sticky clamps the natural
	// fragment geometry that will actually be painted (fixture-31).
	splitCrossingRects(res, contentH, opPage)

	// Drop row shells left behind when text snapped to the next page
	// (fixture-31: empty white rows after Row 27 on page 1).
	stripOrphanRowChrome(res, contentH)
	stretchPaginatedChrome(res)

	// Close open tops on table continuations after rowspan/vertical splits.
	capTablePageBreaks(res, contentH)

	// Print-scoped sticky: clamp the natural fragment without fixed-style
	// continuation clones.
	applyStickyPrint(res, contentH)
	stretchPaginatedChrome(res)
	stripThumbImageHairlines(res)

	if err := validatePaintPageIndices(res.Ops, contentH); err != nil {
		return err
	}

	// Re-derive pages after splits and sticky (new ops / Y shifts).
	opPage = buildPagesAfterSplits(res, contentH, fixedIdx)

	populateLocations(res, contentH, opPage)

	opMap, err := buildStructureTree(doc, res)
	if err != nil {
		return err
	}

	return paintPages(ctx, doc, res, opts, contentH, fixedIdx, opMap)
}

// fixedOpIndices collects the indices of viewport-fixed ops, which are
// stamped on every page at viewport-relative coords.
func fixedOpIndices(res *Result) []int {
	fixedIdx := make([]int, 0, len(res.Ops))

	for i := range res.Ops {
		if res.Ops[i].Fixed {
			fixedIdx = append(fixedIdx, i)
		}
	}

	return fixedIdx
}

// buildPagesAfterSplits re-derives page buckets from the final op Y
// positions after rect splits and sticky shifts added or moved ops.
func buildPagesAfterSplits(res *Result, contentH float64, _ []int) []int {
	opPage, counts := pageBuckets(res.Ops, contentH)
	res.Pages = make([][]int, len(counts))

	for p := range counts {
		if counts[p] > 0 {
			res.Pages[p] = make([]int, 0, counts[p])
		}
	}

	for idx, p := range opPage {
		if p >= 0 && p < len(counts) {
			res.Pages[p] = append(res.Pages[p], idx)
		}
	}

	return opPage
}

var (
	errInvalidContentHeight = errors.New("layout: invalid content height")
	errOutOfRangePageIndex  = errors.New("layout: operation has an out-of-range page index")
)

// validatePaintPageIndices rejects coordinates that cannot be represented by
// the bounded pagination index. Returning an error keeps an oversized page
// distinct from page zero or the final index bucket.
func validatePaintPageIndices(ops []Op, contentH float64) error {
	if contentH <= 0 || math.IsNaN(contentH) || math.IsInf(contentH, 0) {
		return fmt.Errorf("%w: %v", errInvalidContentHeight, contentH)
	}

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		if _, ok := checkedFlowPageOfY(ops[idx].Y, contentH); !ok {
			return fmt.Errorf("layout: operation %d %w", idx, errOutOfRangePageIndex)
		}
	}

	return nil
}

// pageBuckets maps every op to its canvas page in one pass, with per-page
// non-fixed op counts for exact-capacity buckets. Fixed ops leave pageOf at
// its zero value; callers decide whether their fill pass includes them.
//
// The Y+layoutEpsilon bump matches appendOpFragments: a rect fragment ends
// exactly at the next page top (Y = k*contentH), and float division of that
// exact product can round just below k (e.g. (21*785.197)/785.197 =
// 20.9999…). Without the bump the fragment is bucketed to the previous page,
// which then paints two background bands while the intended page paints
// none. The epsilon keeps a boundary-aligned op on the page it starts.
func pageBuckets(ops []Op, contentH float64) ([]int, []int) {
	// Page numbers are dense from 0..maxP, so counts index directly instead
	// of a per-page map (page buckets below are exact-capacity, no growth).
	maxPage := 0

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		pageVal, ok := checkedFlowPageOfY(ops[idx].Y+layoutEpsilon, contentH)
		if !ok {
			return nil, nil
		}

		if pageVal > maxPage {
			maxPage = pageVal
		}
	}

	pageOf := make([]int, len(ops))
	counts := make([]int, maxPage+1)

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		pageVal, ok := checkedFlowPageOfY(ops[idx].Y+layoutEpsilon, contentH)
		if !ok {
			return nil, nil
		}

		pageOf[idx] = pageVal
		counts[pageVal]++
	}

	return pageOf, counts
}

// contentSizeHint estimates the content-stream bytes a page's ops will emit:
// a fixed per-op operator budget plus the escaped text payload (most runes
// emit one byte, escaped to at most four).
func contentSizeHint(ops []Op, groups ...[]int) int {
	const opBudget = 64

	const avgEscapedRuneBytes = 2

	size := 0

	for _, group := range groups {
		for _, idx := range group {
			op := &ops[idx]
			size += opBudget

			if op.Kind == OpText || op.Kind == OpBullet {
				size += len(op.Text) * avgEscapedRuneBytes
			}
		}
	}

	return size
}

// paintPages paints every page: page content ops first, then the fixed layer
// with page-local coordinates. The font-name map and paint closures are
// allocated once and reused across pages; names still re-register per page via
// UseEmbeddedFont after the map is cleared.
//
//nolint:funlen // one pass per page; shared paint/resName closures cover content and fixed layers
func paintPages(
	ctx context.Context, doc *pdf.Document, res *Result, opts PaintOptions,
	contentH float64, fixedIdx []int, opMap map[int]*opTagInfo,
) error {
	var paintErr error

	// fixedIdx is page-independent and never mutated between pages, so apply
	// the shared PaintOrder policy once instead of once per page.
	fixedOrder := fixedIdx
	sortPaintIndices(res.Ops, fixedOrder)

	fontNames := map[*pdf.Font]string{}
	nextFont := 0

	var child *pdf.Content

	var page *pdf.Page

	pageOrder := make([]int, 0)

	resName := func(face *pdf.Font) string {
		if face == nil {
			return "F0"
		}

		if n, ok := fontNames[face]; ok {
			return n
		}

		n := "F" + strconv.Itoa(nextFont)
		nextFont++
		fontNames[face] = n
		child.UseEmbeddedFont(n, face)

		return n
	}

	for pageIdx, idxs := range res.Pages {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: paint context: %w", err)
		}

		clear(fontNames)

		nextFont = 0

		page = doc.AddPage(opts.PageWidth, opts.PageHeight)
		child = page.Content()
		child.Grow(contentSizeHint(res.Ops, idxs, fixedOrder))

		pageOrder = append(pageOrder[:0], idxs...)
		sortPaintIndices(res.Ops, pageOrder)

		painter := pagePainter{
			child:    child,
			page:     page,
			pageN:    pageIdx,
			contentH: contentH,
			pageH:    page.Height(),
			opts:     opts,
			resName:  resName,
			nextImg:  0,
			err:      paintErr,
			opMap:    opMap,
		}

		for _, idx := range pageOrder {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("layout: paint context: %w", err)
			}

			painter.paintOp(idx, &res.Ops[idx])
		}
		// Fixed layer: page-local coords (pageIdx 0 math on every page).
		painter.pageN = 0

		for _, idx := range fixedOrder {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("layout: paint context: %w", err)
			}

			painter.paintOp(idx, &res.Ops[idx])
		}

		paintErr = painter.err
	}

	return paintErr
}

type pagePainter struct {
	child    *pdf.Content
	page     *pdf.Page
	pageN    int
	contentH float64
	pageH    float64
	opts     PaintOptions
	resName  func(*pdf.Font) string
	nextImg  int
	err      error
	opMap    map[int]*opTagInfo
}

//nolint:cyclop,funlen,nestif,wsl // marked content and opacity/transform wrapping for ops
func (p *pagePainter) paintOp(opIdx int, paintOp *Op) {
	if paintOp.Kind == opKindNoop {
		return
	}

	tagInfo := p.opMap[opIdx]

	if paintOp.Kind == OpLinkURI {
		ref := drawLinkXform(p.page, paintOp, p.pageN, p.contentH, p.opts)
		if ref != 0 {
			elem := paintOp.StructElem
			if elem == nil && tagInfo != nil {
				elem = tagInfo.elem
			}
			if elem != nil {
				elem.SetObjRef(ref, p.page)
			}
		}

		return
	}

	needGS := paintOp.XformSet || (paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1)
	if needGS {
		p.child.Save()
	}

	if paintOp.XformSet {
		a, b, cc, d, e, f := pdfCTMFromCSS(paintOp.Xform, p.pageN, p.contentH, p.opts, p.page.Height())
		p.child.Transform(a, b, cc, d, e, f)
	}

	if paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1 {
		p.child.SetOpacity(paintOp.PaintOpacity)
	}

	isUA1 := p.page != nil && p.page.Doc() != nil && p.page.Doc().Policy().IsPDFUA1()

	switch {
	case isUA1 && tagInfo != nil && paintOp.Kind != OpFillRect && paintOp.Kind != OpStrokeRect && paintOp.Kind != OpLine:
		mcid := p.page.AllocMCID(tagInfo.elem)
		p.child.BeginMarkedContent(string(tagInfo.tag), mcid)
		p.drawPageOp(paintOp)
		p.child.EndMarkedContent()
	case isUA1 && paintOp.Kind == OpFillRect:
		p.child.BeginArtifact("Background")
		p.drawPageOp(paintOp)
		p.child.EndArtifact()
	case isUA1 && (paintOp.Kind == OpStrokeRect || paintOp.Kind == OpLine):
		p.child.BeginArtifact("Layout")
		p.drawPageOp(paintOp)
		p.child.EndArtifact()
	case isUA1:
		p.child.BeginArtifact("Layout")
		p.drawPageOp(paintOp)
		p.child.EndArtifact()
	default:
		p.drawPageOp(paintOp)
	}

	if needGS {
		p.child.Restore()
	}
}

// drawPageOp dispatches one op to the shared fill/stroke/line/text/image
// drawing routines.
func (p *pagePainter) drawPageOp(paintOp *Op) {
	switch paintOp.Kind {
	case OpFillRect:
		drawFill(p.child, paintOp, p.pageN, p.contentH, p.opts, p.pageH)
	case OpStrokeRect:
		drawStroke(p.child, paintOp, p.pageN, p.contentH, p.opts, p.pageH)
	case OpLine:
		drawLine(p.child, paintOp, p.pageN, p.contentH, p.opts, p.pageH)
	case OpText, OpBullet:
		drawText(p.child, paintOp, p.pageN, p.contentH, p.opts, p.pageH, p.resName(paintOp.Font))
	case OpImage:
		name := "I" + strconv.Itoa(p.nextImg)
		p.nextImg++

		if err := drawImage(p.page, p.child, paintOp, p.pageN, p.contentH, p.opts, name); err != nil && p.err == nil {
			p.err = err
		}
	case OpLinkURI, opKindNoop:
	}
}

// PaintStyle is the resolved per-op appearance that PDF and image adapters
// share: translucent-fill pre-composition, stroke min-width, and the Latin-only
// fake-bold gate (stroking CJK/Type0 outlines creates horizontal streaks).
type PaintStyle struct {
	FillR, FillG, FillB float64 // final RGB; alpha pre-composited against white when translucent
	FillAlpha           float64 // 1 after pre-composite; raw Op.Alpha when opaque
	StrokeWidth         float64
	FakeBold            bool
}

// StyleOf resolves paint-semantics for op. Layout owns these decisions so
// convert HF and imageout adapters do not drift.
func StyleOf(paintOp *Op) PaintStyle {
	if paintOp == nil {
		return PaintStyle{FillAlpha: 1, StrokeWidth: 1} //nolint:exhaustruct // intentional zero fields
	}

	pstyle := PaintStyle{ //nolint:exhaustruct // intentional zero fields
		FillR: paintOp.R, FillG: paintOp.G, FillB: paintOp.B, FillAlpha: 1,
		StrokeWidth: paintOp.Width,
	}
	if pstyle.StrokeWidth <= 0 {
		pstyle.StrokeWidth = 1
	}
	// Pre-composite translucent fills against white paper (PDF path).
	if paintOp.Alpha > 0 && paintOp.Alpha < 1 {
		a := paintOp.Alpha
		pstyle.FillR = paintOp.R*a + (1 - a)
		pstyle.FillG = paintOp.G*a + (1 - a)
		pstyle.FillB = paintOp.B*a + (1 - a)
		pstyle.FillAlpha = 1
	} else if paintOp.Alpha > 0 {
		pstyle.FillAlpha = paintOp.Alpha
	}

	pstyle.FakeBold = FakeBoldFor(paintOp)

	return pstyle
}

// FakeBoldFor reports whether CSS bold should be synthesized for op (Latin
// only; CJK stroking produces streak artifacts).
func FakeBoldFor(op *Op) bool {
	if op == nil || !op.Bold || (op.Font != nil && op.Font.Bold()) {
		return false
	}

	for _, r := range op.Text {
		if r > byteMax {
			return false
		}
	}

	return true
}

// BandOptions configures PaintBand (shared op→PDF dispatch for body and HF).
type BandOptions struct {
	OriginX, OriginY float64 // canvas origin on the page (y-down canvas → PDF via OriginY)
	// ContentH and Page geometry for canvasToPDF when using pageIdx 0 math.
	// When ContentH is 0, OriginY is treated as the PDF y of canvas y=0 top
	// edge and coordinates are mapped as PDF_y = OriginY - canvas_y.
	ContentH float64
	PageH    float64
	Margins  PaintOptions // MarginLeft/Top used when ContentH > 0
}

// PaintBand paints ops onto an existing page's content stream. Same dispatch
// as Paint for fill/stroke/line/text/image (colors, opacity, transforms,
// embedded fonts, fake-bold policy). Pagination and fixed stamps are skipped;
// z-sorting uses the shared PaintOrder policy. Link ops are left to the caller
// (annotations need document context). Returns the first image-embed error, if
// any.
func PaintBand(p *pdf.Page, c *pdf.Content, ops []Op, opts BandOptions) error {
	return PaintBandContext(context.Background(), p, c, ops, opts)
}

// PaintBandContext is the cancellation-aware form of PaintBand used for
// HTML headers and footers.
func PaintBandContext(ctx context.Context, page *pdf.Page, chld *pdf.Content, ops []Op, opts BandOptions) error {
	if ctx == nil {
		return errNilContext
	}

	ctx, cancel := beginPaintContext(ctx)
	defer cancel()

	if page == nil || chld == nil {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("layout: paint band context: %w", err)
	}

	fontNames := map[*pdf.Font]string{}
	nextFont := 0
	resName := func(face *pdf.Font) string {
		if face == nil {
			return "F0"
		}

		if n, ok := fontNames[face]; ok {
			return n
		}

		n := "B" + strconv.Itoa(nextFont)
		nextFont++
		fontNames[face] = n
		chld.UseEmbeddedFont(n, face)

		return n
	}

	return paintBandOps(ctx, page, chld, ops, opts, resName)
}

// paintBandOps dispatches every op onto the band content stream in the same
// PaintOrder used by the paginated body and raster adapter. Link operations
// are still skipped here because annotations need document context and are
// wired by the caller in display-list order.
func paintBandOps(
	ctx context.Context, page *pdf.Page, chld *pdf.Content, ops []Op, opts BandOptions,
	resName func(*pdf.Font) string,
) error {
	nextImg := 0
	contentH := opts.ContentH

	pageH := opts.PageH
	if pageH <= 0 {
		pageH = page.Height()
	}
	// Band mode without full page geometry: map canvas (y down) to PDF using
	// OriginY as the top edge and OriginX as left origin.
	useSimple := contentH <= 0

	var firstErr error

	for _, idx := range PaintOrder(ops) {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: paint band context: %w", err)
		}

		paintOp := &ops[idx]
		if paintOp.Kind == OpLinkURI || paintOp.Kind == opKindNoop {
			continue
		}

		paintBandOp(chld, page, paintOp, opts, contentH, pageH, useSimple, resName, &nextImg, &firstErr)
	}

	return firstErr
}

// paintBandOp paints one band op: graphics-state save, transform, opacity,
// then the shared draw dispatch, and a final restore.
//
//nolint:cyclop,wsl // band op opacity, transform and artifact wrapping
func paintBandOp(
	chld *pdf.Content, page *pdf.Page, paintOp *Op, opts BandOptions, contentH, pageH float64,
	useSimple bool, resName func(*pdf.Font) string, nextImg *int, firstErr *error,
) {
	needGS := paintOp.XformSet || (paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1)
	if needGS {
		chld.Save()
	}

	if paintOp.XformSet && !useSimple {
		a, b, cc, d, e, f := pdfCTMFromCSS(paintOp.Xform, 0, contentH, opts.Margins, pageH)
		chld.Transform(a, b, cc, d, e, f)
	}

	if paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1 {
		chld.SetOpacity(paintOp.PaintOpacity)
	}

	isUA1 := page != nil && page.Doc() != nil && page.Doc().Policy().IsPDFUA1()
	needArtifact := isUA1 && chld.MarkedDepth() == 0
	if needArtifact {
		chld.BeginArtifact("Pagination")
	}

	drawBandOp(chld, page, paintOp, opts, contentH, pageH, useSimple, resName, nextImg, firstErr)

	if needArtifact {
		chld.EndArtifact()
	}

	if needGS {
		chld.Restore()
	}
}

// drawBandOp dispatches one band op to the simple band path or the shared
// draw routines.
func drawBandOp(
	chld *pdf.Content, page *pdf.Page, paintOp *Op, opts BandOptions, contentH, pageH float64,
	useSimple bool, resName func(*pdf.Font) string, nextImg *int, firstErr *error,
) {
	if useSimple {
		recordBandError(firstErr, paintOpBandSimple(chld, page, paintOp, opts, resName(paintOp.Font), nextImg))

		return
	}

	switch paintOp.Kind {
	case OpFillRect:
		drawFill(chld, paintOp, 0, contentH, opts.Margins, pageH)
	case OpStrokeRect:
		drawStroke(chld, paintOp, 0, contentH, opts.Margins, pageH)
	case OpLine:
		drawLine(chld, paintOp, 0, contentH, opts.Margins, pageH)
	case OpText, OpBullet:
		drawText(chld, paintOp, 0, contentH, opts.Margins, pageH, resName(paintOp.Font))
	case OpImage:
		name := "I" + strconv.Itoa(*nextImg)
		*nextImg++

		recordBandError(firstErr, drawImage(page, chld, paintOp, 0, contentH, opts.Margins, name))
	case OpLinkURI, opKindNoop:
	}
}

// recordBandError keeps the first image-embed error.
func recordBandError(firstErr *error, err error) {
	if err != nil && *firstErr == nil {
		*firstErr = err
	}
}

func paintOpBandSimple(
	chld *pdf.Content, _ *pdf.Page, paintOp *Op, opts BandOptions, fontName string, nextImg *int,
) error {
	posX := opts.OriginX + paintOp.X

	switch paintOp.Kind {
	case OpFillRect:
		bandFillRect(chld, opts, paintOp, posX)
	case OpStrokeRect:
		bandStrokeRect(chld, opts, paintOp, posX)
	case OpLine:
		bandStrokeLine(chld, opts, paintOp, posX)
	case OpText, OpBullet:
		bandText(chld, opts, paintOp, posX, fontName)

	case OpImage:
		return bandImage(chld, opts, paintOp, posX, nextImg)
	case OpLinkURI, opKindNoop:
	}

	return nil
}

func bandFillRect(chld *pdf.Content, opts BandOptions, paintOp *Op, posX float64) {
	ps := StyleOf(paintOp)
	y := opts.OriginY - (paintOp.Y + paintOp.H)

	chld.SetFillColor(ps.FillR, ps.FillG, ps.FillB)
	chld.Rect(posX, y, paintOp.W, paintOp.H)
	chld.Fill()
}

func bandStrokeRect(chld *pdf.Content, opts BandOptions, paintOp *Op, posX float64) {
	y := opts.OriginY - (paintOp.Y + paintOp.H)
	chld.SetStrokeColor(paintOp.R, paintOp.G, paintOp.B)
	chld.SetLineWidth(1)
	chld.Rect(posX, y, paintOp.W, paintOp.H)
	chld.Stroke()
}

func bandStrokeLine(chld *pdf.Content, opts BandOptions, paintOp *Op, posX float64) {
	yEnd := opts.OriginY - paintOp.Y
	yTwo := opts.OriginY - (paintOp.Y + paintOp.H)

	width := paintOp.Width
	if width <= 0 {
		width = 1
	}

	chld.SetStrokeColor(paintOp.R, paintOp.G, paintOp.B)
	chld.SetLineWidth(width)
	chld.MoveTo(posX, yEnd)
	chld.LineTo(opts.OriginX+paintOp.X+paintOp.W, yTwo)
	chld.Stroke()
}

func bandText(chld *pdf.Content, opts BandOptions, paintOp *Op, posX float64, fontName string) {
	posY := opts.OriginY - paintOp.Y
	chld.SetFillColor(paintOp.R, paintOp.G, paintOp.B)

	if fontName == "" {
		fontName = "F0"
	}

	chld.SetFont(fontName, paintOp.Size)
	chld.BeginText()
	chld.TextAt(posX, posY)
	chld.SetCharSpacing(paintOp.LetterSpacing)

	if FakeBoldFor(paintOp) {
		chld.SetLineWidth(paintOp.Size * outlineStrokeRatio)
		chld.TextRenderMode(two)
	}

	chld.TextShow(transformInlineText(paintOp.Text, paintOp.TextTransform))

	if FakeBoldFor(paintOp) {
		chld.TextRenderMode(0)
	}

	chld.EndText()
}

func bandImage(chld *pdf.Content, opts BandOptions, paintOp *Op, posX float64, nextImg *int) error {
	name := "I" + strconv.Itoa(*nextImg)
	*nextImg++

	posY := opts.OriginY - (paintOp.Y + paintOp.H)
	if paintOp.IsJPEG {
		if err := chld.AddJPEGImage(name, posX, posY, paintOp.W, paintOp.H, paintOp.Image); err != nil {
			return fmt.Errorf("layout: band image %s: %w", name, err)
		}

		return nil
	}

	if err := chld.AddPNGImage(name, posX, posY, paintOp.W, paintOp.H, paintOp.Image); err != nil {
		return fmt.Errorf("layout: band image %s: %w", name, err)
	}

	return nil
}

func populateLocations(res *Result, contentH float64, opPage []int) {
	res.Locations = nil
	if res.root == nil {
		return
	}

	// The flattened box list mirrors the preorder walk below, so the
	// element-location count is known in advance: one append, no growth.
	count := 0

	for _, b := range res.boxes {
		if b.node != nil {
			count++
		}
	}

	res.Locations = make([]ElementLocation, 0, count)

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.node != nil {
			page := -1
			if boxNode.opStart <= boxNode.opEnd && boxNode.opStart < len(opPage) {
				page = opPage[boxNode.opStart]
			}

			if page < 0 {
				page = int(boxNode.y / contentH)
				if page < 0 {
					page = 0
				}
			}

			res.Locations = append(res.Locations, ElementLocation{
				Node: boxNode.node, Page: page, X: boxNode.x, Y: boxNode.y, W: boxNode.w, H: boxNode.height,
			})
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)
}

// canvasToPDF converts a canvas point (y down, origin at content top-left of
// page 0) to PDF coordinates on the given page.
func canvasToPDF(opX, opY float64, pageIdx int, contentH float64, opts PaintOptions, pageH float64) (float64, float64) {
	x := opts.MarginLeft + opX
	y := pageH - opts.MarginTop - opY + float64(pageIdx)*contentH

	return x, y
}

//nolint:varnamelen,wsl // PDF path helpers use compact graphics-state names
func drawFill(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	ps := StyleOf(op)
	c.SetFillColor(ps.FillR, ps.FillG, ps.FillB)
	if op.Radius > 0 || opHasRoundedCorners(op) {
		roundedRectPathCorners(c, x, y, op.W, op.H, opRadii(op))
	} else {
		c.Rect(x, y, op.W, op.H)
	}
	c.Fill()
}

//nolint:varnamelen,wsl // PDF path helpers use compact graphics-state names
func drawMaskedStroke(c *pdf.Content, op *Op, x, y, width float64) {
	if op.StrokeMask&StrokeMaskTop != 0 {
		c.SetLineCap(1)
		roundedTopPath(c, x, y, op.W, op.H, width, opRadii(op))
		c.Stroke()
		c.SetLineCap(0)
	}
	if op.StrokeMask&StrokeMaskBottom != 0 {
		c.SetLineCap(1)
		roundedBottomPath(c, x, y, op.W, width, opRadii(op))
		c.Stroke()
		c.SetLineCap(0)
	}
	if op.StrokeMask&StrokeMaskLeft != 0 {
		if opHasRoundedCorners(op) {
			c.SetLineCap(1)
		}
		roundedLeftPath(c, x, y, op.H, width, opRadii(op))
		c.Stroke()
		c.SetLineCap(0)
	}
	if op.StrokeMask&StrokeMaskRight != 0 {
		if opHasRoundedCorners(op) {
			c.SetLineCap(1)
		}
		roundedRightPath(c, x, y, op.W, op.H, width, opRadii(op))
		c.Stroke()
		c.SetLineCap(0)
	}
}

//nolint:varnamelen,wsl // PDF path helpers use compact graphics-state names
func drawStroke(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	c.SetStrokeColor(op.R, op.G, op.B)
	width := op.Width
	if width <= 0 {
		width = 1
	}
	c.SetLineWidth(width)
	if op.StrokeMask != 0 {
		// Partial / open multi-page frames: paint each selected side.
		drawMaskedStroke(c, op, x, y, width)

		return
	}
	if op.Radius > 0 || opHasRoundedCorners(op) {
		roundedRectPathCorners(c, x, y, op.W, op.H, opRadii(op))
	} else {
		c.Rect(x, y, op.W, op.H)
	}
	c.Stroke()
}

// roundedTopPath emits the top edge and its two corner arcs in PDF
// coordinates. It is an open path because the remaining edges may use a
// different CSS border style.
//
//nolint:varnamelen // PDF path builder uses conventional short receiver names
func roundedTopPath(
	c *pdf.Content,
	originX, originY, boxWidth, boxHeight, strokeWidth float64,
	radii [4]float64,
) {
	const kappa = 0.5522847498

	strokeInset := strokeWidth / 2 //nolint:mnd // half the stroke width insets the centerline radius
	leftRadius := math.Max(radii[0]-strokeInset, 0)
	rightRadius := math.Max(radii[1]-strokeInset, 0)
	topY := originY + boxHeight
	rightX := originX + boxWidth

	if leftRadius <= 0 {
		c.MoveTo(originX, topY)
	} else {
		c.MoveTo(originX, topY-leftRadius)
		c.CurveTo(
			originX, topY-leftRadius+kappa*leftRadius,
			originX+leftRadius-kappa*leftRadius, topY,
			originX+leftRadius, topY,
		)
	}

	c.LineTo(rightX-rightRadius, topY)

	if rightRadius > 0 {
		c.CurveTo(
			rightX-rightRadius+kappa*rightRadius, topY,
			rightX, topY-rightRadius+kappa*rightRadius,
			rightX, topY-rightRadius,
		)
	} else {
		c.LineTo(rightX, topY)
	}
}

// roundedLeftPath emits the left edge and its two corner arcs in PDF
// coordinates. It is open because the remaining edges may use other paint.
//
//nolint:varnamelen // PDF path builder uses conventional geometry names
func roundedLeftPath(
	c *pdf.Content,
	originX, originY, boxHeight, strokeWidth float64,
	radii [4]float64,
) {
	const kappa = 0.5522847498

	strokeInset := strokeWidth / 2 //nolint:mnd // half the stroke width insets the centerline radius
	originX += strokeInset
	topRadius := math.Max(radii[0]-strokeInset, 0)
	bottomRadius := math.Max(radii[3]-strokeInset, 0)
	topY := originY + boxHeight

	if bottomRadius > 0 {
		c.MoveTo(originX+bottomRadius, originY)
		c.CurveTo(
			originX+bottomRadius-kappa*bottomRadius, originY,
			originX, originY+bottomRadius-kappa*bottomRadius,
			originX, originY+bottomRadius,
		)
	} else {
		c.MoveTo(originX, originY)
	}

	c.LineTo(originX, topY-topRadius)

	if topRadius > 0 {
		c.CurveTo(
			originX, topY-topRadius+kappa*topRadius,
			originX+topRadius-kappa*topRadius, topY,
			originX+topRadius, topY,
		)
	} else {
		c.LineTo(originX, topY)
	}
}

// roundedBottomPath emits the bottom edge and its two corner arcs (PDF coords).
//
//nolint:varnamelen // PDF path builder uses conventional short receiver names
func roundedBottomPath(
	c *pdf.Content,
	originX, originY, boxWidth, strokeWidth float64,
	radii [4]float64,
) {
	const kappa = 0.5522847498

	strokeInset := strokeWidth / 2 //nolint:mnd // half the stroke width insets the centerline radius
	leftRadius := math.Max(radii[3]-strokeInset, 0)
	rightRadius := math.Max(radii[2]-strokeInset, 0)
	rightX := originX + boxWidth

	if leftRadius <= 0 {
		c.MoveTo(originX, originY)
	} else {
		c.MoveTo(originX, originY+leftRadius)
		c.CurveTo(
			originX, originY+leftRadius-kappa*leftRadius,
			originX+leftRadius-kappa*leftRadius, originY,
			originX+leftRadius, originY,
		)
	}

	c.LineTo(rightX-rightRadius, originY)

	if rightRadius > 0 {
		c.CurveTo(
			rightX-rightRadius+kappa*rightRadius, originY,
			rightX, originY+rightRadius-kappa*rightRadius,
			rightX, originY+rightRadius,
		)
	} else {
		c.LineTo(rightX, originY)
	}
}

// roundedRightPath emits the right edge and its two corner arcs (PDF coords).
//
//nolint:varnamelen // PDF path builder uses conventional geometry names
func roundedRightPath(
	c *pdf.Content,
	originX, originY, boxWidth, boxHeight, strokeWidth float64,
	radii [4]float64,
) {
	const kappa = 0.5522847498

	strokeInset := strokeWidth / 2 //nolint:mnd // half the stroke width insets the centerline radius
	rightX := originX + boxWidth - strokeInset
	topRadius := math.Max(radii[1]-strokeInset, 0)
	bottomRadius := math.Max(radii[2]-strokeInset, 0)
	topY := originY + boxHeight

	if topRadius > 0 {
		c.MoveTo(rightX-topRadius, topY)
		c.CurveTo(
			rightX-topRadius+kappa*topRadius, topY,
			rightX, topY-topRadius+kappa*topRadius,
			rightX, topY-topRadius,
		)
	} else {
		c.MoveTo(rightX, topY)
	}

	c.LineTo(rightX, originY+bottomRadius)

	if bottomRadius > 0 {
		c.CurveTo(
			rightX, originY+bottomRadius-kappa*bottomRadius,
			rightX-bottomRadius+kappa*bottomRadius, originY,
			rightX-bottomRadius, originY,
		)
	} else {
		c.LineTo(rightX, originY)
	}
}

func opRadii(op *Op) [4]float64 {
	if op.RadiusTopLeft == 0 && op.RadiusTopRight == 0 && op.RadiusBottomRight == 0 && op.RadiusBottomLeft == 0 {
		return [4]float64{op.Radius, op.Radius, op.Radius, op.Radius}
	}

	return [4]float64{op.RadiusTopLeft, op.RadiusTopRight, op.RadiusBottomRight, op.RadiusBottomLeft}
}

func opHasRoundedCorners(op *Op) bool {
	for _, radius := range opRadii(op) {
		if radius > 0 {
			return true
		}
	}

	return false
}

// roundedRectPathCorners emits a PDF path for CSS order: top-left,
// top-right, bottom-right, bottom-left. PDF's origin is bottom-left here.
//
//nolint:varnamelen // PDF path helper mirrors the standard Bezier approximation
func roundedRectPathCorners(c *pdf.Content, originX, originY, width, height float64, radii [4]float64) {
	const (
		kappa = 0.5522847498
		half  = 2.0
	)

	if !opRadiiPositive(radii) {
		c.Rect(originX, originY, width, height)

		return
	}

	for i := range radii {
		if radii[i] < 0 {
			radii[i] = 0
		}
	}

	// Clamp each corner to the local box dimensions. Layout already applies
	// CSS's adjacent-radii scaling; this protects hand-built ops as well.
	for i := range radii {
		if radii[i] > width/half {
			radii[i] = width / half
		}

		if radii[i] > height/half {
			radii[i] = height / half
		}
	}

	topLeft, topRight := radii[0], radii[1]
	bottomRight, bottomLeft := radii[2], radii[3]
	c.MoveTo(originX+bottomLeft, originY)
	c.LineTo(originX+width-bottomRight, originY)
	c.CurveTo(
		originX+width-bottomRight+kappa*bottomRight, originY,
		originX+width, originY+bottomRight-kappa*bottomRight,
		originX+width, originY+bottomRight,
	)
	c.LineTo(originX+width, originY+height-topRight)
	c.CurveTo(
		originX+width, originY+height-topRight+kappa*topRight,
		originX+width-topRight+kappa*topRight, originY+height,
		originX+width-topRight, originY+height,
	)
	c.LineTo(originX+topLeft, originY+height)
	c.CurveTo(
		originX+topLeft-kappa*topLeft, originY+height,
		originX, originY+height-topLeft+kappa*topLeft,
		originX, originY+height-topLeft,
	)
	c.LineTo(originX, originY+bottomLeft)
	c.CurveTo(
		originX, originY+bottomLeft-kappa*bottomLeft,
		originX+bottomLeft-kappa*bottomLeft, originY,
		originX+bottomLeft, originY,
	)
}

func opRadiiPositive(radii [4]float64) bool {
	for _, radius := range radii {
		if radius > 0 {
			return true
		}
	}

	return false
}

func drawLine(chld *pdf.Content, paintOp *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	xEnd, yEnd := canvasToPDF(paintOp.X, paintOp.Y, pageIdx, contentH, opts, pageH)
	xTwo, yTwo := canvasToPDF(paintOp.X+paintOp.W, paintOp.Y+paintOp.H, pageIdx, contentH, opts, pageH)

	width := paintOp.Width
	if width <= 0 {
		width = 1
	}

	chld.SetStrokeColor(paintOp.R, paintOp.G, paintOp.B)
	chld.SetLineWidth(width)
	chld.MoveTo(xEnd, yEnd)
	chld.LineTo(xTwo, yTwo)
	chld.Stroke()
}

func drawText(
	chld *pdf.Content, paintOp *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64, fontName string,
) {
	posX, posY := canvasToPDF(paintOp.X, paintOp.Y, pageIdx, contentH, opts, pageH)
	chld.SetFillColor(paintOp.R, paintOp.G, paintOp.B)

	if fontName == "" {
		fontName = "F0"
	}

	chld.SetFont(fontName, paintOp.Size)
	chld.BeginText()

	if paintOp.RotateDeg == 90 || paintOp.RotateDeg == -90 {
		if paintOp.RotateDeg < 0 {
			// PDF's y-up text space reverses the screen-space direction.
			// A CSS -90deg vertical run must therefore advance toward
			// increasing canvas Y, not above its containing box.
			chld.TextMatrix(0, -1, 1, 0, posX, posY)
		} else {
			chld.TextMatrix(0, 1, -1, 0, posX, posY)
		}
	} else {
		chld.TextAt(posX, posY)
	}

	chld.SetCharSpacing(paintOp.LetterSpacing)
	// Fake bold only for Latin when CSS wants bold but the face is not bold.
	// Stroking CJK/Type0 outlines creates horizontal streak artifacts.
	fakeBold := FakeBoldFor(paintOp)
	if fakeBold {
		chld.SetLineWidth(paintOp.Size * outlineStrokeRatio)
		chld.TextRenderMode(two) // fill + stroke
	}

	chld.TextShow(transformInlineText(paintOp.Text, paintOp.TextTransform))

	if fakeBold {
		chld.TextRenderMode(0)
	}

	chld.EndText()
}

func drawImage(
	page *pdf.Page, chld *pdf.Content, paintOp *Op, pageIdx int, contentH float64, opts PaintOptions, name string,
) error {
	posX, posY := canvasToPDF(paintOp.X, paintOp.Y+paintOp.H, pageIdx, contentH, opts, page.Height())

	if name == "" {
		name = "I0"
	}

	if paintOp.IsJPEG {
		if err := chld.AddJPEGImage(name, posX, posY, paintOp.W, paintOp.H, paintOp.Image); err != nil {
			return fmt.Errorf("layout: embed jpeg %s: %w", name, err)
		}

		return nil
	}

	if err := chld.AddPNGImage(name, posX, posY, paintOp.W, paintOp.H, paintOp.Image); err != nil {
		return fmt.Errorf("layout: embed png %s: %w", name, err)
	}

	return nil
}

// drawLinkXform places a URI annotation. Annotations are page-space (not under
// content-stream CTM), so CSS transforms are applied to the canvas rect first.
func drawLinkXform(page *pdf.Page, paintOp *Op, pageIdx int, contentH float64, opts PaintOptions) pdf.ObjRef {
	if len(paintOp.URI) > 0 && paintOp.URI[0] == '#' {
		return 0
	}

	x1Val, yMin, xMax, y1Val := paintOp.X, paintOp.Y, paintOp.X+paintOp.W, paintOp.Y+paintOp.H
	if paintOp.XformSet {
		x1Val, yMin, xMax, y1Val = linkXformBounds(paintOp)
	}

	llx, lly := canvasToPDF(x1Val, y1Val, pageIdx, contentH, opts, page.Height())
	urx, ury := canvasToPDF(xMax, yMin, pageIdx, contentH, opts, page.Height())

	if llx > urx {
		llx, urx = urx, llx
	}

	if lly > ury {
		lly, ury = ury, lly
	}

	return page.AddLinkURI([4]float64{llx, lly, urx, ury}, paintOp.URI)
}

// linkXformBounds returns the axis-aligned canvas bounds of the op rect after
// its CSS transform (annotation rectangles are page-space, not CTM-space).
func linkXformBounds(paintOp *Op) (float64, float64, float64, float64) {
	corners := [4][2]float64{
		{paintOp.X, paintOp.Y}, {paintOp.X + paintOp.W, paintOp.Y},
		{paintOp.X, paintOp.Y + paintOp.H}, {paintOp.X + paintOp.W, paintOp.Y + paintOp.H},
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, pt := range corners {
		textX, typeY := paintOp.Xform.Apply(pt[0], pt[1])
		minX = math.Min(minX, textX)
		minY = math.Min(minY, typeY)
		maxX = math.Max(maxX, textX)
		maxY = math.Max(maxY, typeY)
	}

	return minX, minY, maxX, maxY
}
