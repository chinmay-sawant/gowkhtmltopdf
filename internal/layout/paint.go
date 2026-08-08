package layout

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"

	"gowkhtmltopdf/internal/pdf"
)

// Page-break keyword constants shared by the pagination passes.
const (
	pageBreakAvoid  = "avoid"
	pageBreakAlways = "always"
)

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
// always, page-break-inside: avoid (best-effort, box must fit one page), and
// table rows never split.
//
// After pagination Paint fills res.Pages (page → op indices) and res.Locations
// (element boxes in document order with their page and canvas rect).
// backgroundIfNil returns the caller's context, or a fresh background context
// when nil so derived contexts never panic on legacy callers.
func backgroundIfNil(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

func Paint(doc *pdf.Document, res *Result, opts PaintOptions) error {
	return PaintContext(context.Background(), doc, res, opts)
}

// PaintContext is the cancellation-aware form of Paint. The legacy Paint
// entrypoint remains a background-context adapter for package callers that do
// not have a request context.
func PaintContext(ctx context.Context, doc *pdf.Document, res *Result, opts PaintOptions) error {
	ctx, cancel := context.WithCancel(backgroundIfNil(ctx))
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

	if len(res.Ops) == 0 {
		doc.AddPage(opts.PageWidth, opts.PageHeight)
		populateLocations(res, contentH, nil)

		return nil
	}

	opPage := paginateOps(res, contentH)
	fixedIdx := fixedOpIndices(res)

	// Split rect ops at page boundaries first so sticky clamps the natural
	// fragment geometry that will actually be painted (fixture-31).
	splitCrossingRects(res, contentH, opPage)

	// Drop row shells left behind when text snapped to the next page
	// (fixture-31: empty white rows after Row 27 on page 1).
	stripOrphanRowChrome(res, contentH)

	// Close open tops on table continuations after rowspan/vertical splits.
	capTablePageBreaks(res, contentH)

	// Print-scoped sticky: clamp the natural fragment without fixed-style
	// continuation clones.
	applyStickyPrint(res, contentH)

	// Re-derive pages after splits and sticky (new ops / Y shifts).
	opPage = buildPagesAfterSplits(res, contentH, fixedIdx)

	populateLocations(res, contentH, opPage)

	return paintPages(ctx, doc, res, opts, contentH, fixedIdx)
}

// fixedOpIndices collects the indices of viewport-fixed ops, which are
// stamped on every page at viewport-relative coords.
func fixedOpIndices(res *Result) []int {
	var fixedIdx []int

	for i := range res.Ops {
		if res.Ops[i].Fixed {
			fixedIdx = append(fixedIdx, i)
		}
	}

	return fixedIdx
}

// buildPagesAfterSplits re-derives page buckets from the final op Y
// positions after rect splits and sticky shifts added or moved ops.
func buildPagesAfterSplits(res *Result, contentH float64, fixedIdx []int) []int {
	opPage := make([]int, len(res.Ops))
	perPage := map[int][]int{}

	for idx := range res.Ops {
		if res.Ops[idx].Fixed {
			continue
		}

		p := int(res.Ops[idx].Y / contentH)
		opPage[idx] = p
		perPage[p] = append(perPage[p], idx)
	}

	maxP := 0
	for p := range perPage {
		if p > maxP {
			maxP = p
		}
	}

	if len(fixedIdx) > 0 && maxP < 0 {
		maxP = 0
	}

	if len(perPage) == 0 && len(fixedIdx) > 0 {
		maxP = 0
		perPage[0] = nil
	}

	res.Pages = make([][]int, maxP+1)
	for p := 0; p <= maxP; p++ {
		res.Pages[p] = perPage[p]
	}

	return opPage
}

// paintPages paints every page: page content ops first, then the fixed layer
// with page-local coordinates.
func paintPages(
	ctx context.Context, doc *pdf.Document, res *Result, opts PaintOptions, contentH float64, fixedIdx []int,
) error {
	var paintErr error

	for pageIdx, idxs := range res.Pages {
		if err := paintPageOps(ctx, doc, res, opts, contentH, pageIdx, idxs, fixedIdx, &paintErr); err != nil {
			return err
		}
	}

	return paintErr
}

// paintPageOps paints one page's content ops and its fixed layer, returning
// the first context error.
func paintPageOps(
	ctx context.Context, doc *pdf.Document, res *Result, opts PaintOptions, contentH float64,
	pageIdx int, idxs, fixedIdx []int, paintErr *error,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("layout: paint context: %w", err)
	}

	page := doc.AddPage(opts.PageWidth, opts.PageHeight)
	child := page.Content()
	fontNames := map[*pdf.Font]string{}
	nextFont := 0
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
	nextImg := 0
	paintOp := func(paintOp *Op, pageN int) {
		paintOpOnPage(child, page, paintOp, pageN, contentH, opts, page.Height(), resName, &nextImg, paintErr)
	}

	sortPaintIndices(res.Ops, idxs)

	for _, idx := range idxs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: paint context: %w", err)
		}

		paintOp(&res.Ops[idx], pageIdx)
	}
	// Fixed layer: page-local coords (pageIdx 0 math on every page).
	sortPaintIndices(res.Ops, fixedIdx)

	for _, idx := range fixedIdx {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: paint context: %w", err)
		}

		paintOp(&res.Ops[idx], 0)
	}

	return nil
}

// paintOpOnPage dispatches one op onto a page's content stream: transforms,
// opacity, fill/stroke/line/text/image drawing, then restores the graphics
// state. The first image-embed error is recorded into paintErr.
func paintOpOnPage(
	child *pdf.Content, page *pdf.Page, paintOp *Op, pageN int, contentH float64, opts PaintOptions,
	pageH float64, resName func(*pdf.Font) string, nextImg *int, paintErr *error,
) {
	if paintOp.Kind == opKindNoop {
		return
	}

	if paintOp.Kind == OpLinkURI {
		drawLinkXform(page, paintOp, pageN, contentH, opts)

		return
	}

	needGS := paintOp.XformSet || (paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1)
	if needGS {
		child.Save()
	}

	if paintOp.XformSet {
		a, b, cc, d, e, f := pdfCTMFromCSS(paintOp.Xform, pageN, contentH, opts, page.Height())
		child.Transform(a, b, cc, d, e, f)
	}

	if paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1 {
		child.SetOpacity(paintOp.PaintOpacity)
	}

	drawPageOp(child, page, paintOp, pageN, contentH, opts, pageH, resName, nextImg, paintErr)

	if needGS {
		child.Restore()
	}
}

// drawPageOp dispatches one op to the shared fill/stroke/line/text/image
// drawing routines.
func drawPageOp(
	child *pdf.Content, page *pdf.Page, paintOp *Op, pageN int, contentH float64, opts PaintOptions,
	pageH float64, resName func(*pdf.Font) string, nextImg *int, paintErr *error,
) {
	switch paintOp.Kind {
	case OpFillRect:
		drawFill(child, paintOp, pageN, contentH, opts, pageH)
	case OpStrokeRect:
		drawStroke(child, paintOp, pageN, contentH, opts, pageH)
	case OpLine:
		drawLine(child, paintOp, pageN, contentH, opts, pageH)
	case OpText, OpBullet:
		drawText(child, paintOp, pageN, contentH, opts, pageH, resName(paintOp.Font))
	case OpImage:
		name := "I" + strconv.Itoa(*nextImg)
		*nextImg++

		if err := drawImage(page, child, paintOp, pageN, contentH, opts, name); err != nil && *paintErr == nil {
			*paintErr = err
		}
	case OpLinkURI, opKindNoop:
	}
}

func sortPaintIndices(ops []Op, idxs []int) {
	sort.SliceStable(idxs, func(idx, jdx int) bool {
		acc, boxN := ops[idxs[idx]], ops[idxs[jdx]]
		absZ, boxZ := 0, 0

		if acc.ZIndexSet {
			absZ = acc.ZIndex
		}

		if boxN.ZIndexSet {
			boxZ = boxN.ZIndex
		}

		if absZ != boxZ {
			return absZ < boxZ
		}

		if acc.Positioned != boxN.Positioned {
			return !acc.Positioned
		}
		// Same stacking context: backgrounds/borders under text & images so
		// page-split fill remnants cannot cover continuation-row ink
		// (fixture-31 Row 28 vs next-row white fill).
		la, lb := paintLayer(acc.Kind), paintLayer(boxN.Kind)
		if la != lb {
			return la < lb
		}

		return idxs[idx] < idxs[jdx]
	})
}

// paintLayer orders ops within a z-index band: chrome under content.
func paintLayer(k OpKind) int {
	switch k {
	case OpFillRect, OpStrokeRect, OpLine:
		return 0
	case OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
		return 1
	}

	return 1
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
// embedded fonts, fake-bold policy). Pagination, z-sorting and fixed stamps
// are skipped. Link ops are left to the caller (annotations need document
// context). Returns the first image-embed error, if any.
func PaintBand(p *pdf.Page, c *pdf.Content, ops []Op, opts BandOptions) error {
	return PaintBandContext(context.Background(), p, c, ops, opts)
}

// PaintBandContext is the cancellation-aware form of PaintBand used for
// HTML headers and footers.
func PaintBandContext(ctx context.Context, page *pdf.Page, chld *pdf.Content, ops []Op, opts BandOptions) error {
	ctx, cancel := context.WithCancel(backgroundIfNil(ctx))
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

// paintBandOps dispatches every op onto the band content stream, honoring the
// shared colors/opacity/transform/fake-bold policy. Returns the first
// image-embed error, if any.
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

	for i := range ops {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: paint band context: %w", err)
		}

		paintOp := &ops[i]
		if paintOp.Kind == OpLinkURI || paintOp.Kind == opKindNoop {
			continue
		}

		paintBandOp(chld, page, paintOp, opts, contentH, pageH, useSimple, resName, &nextImg, &firstErr)
	}

	return firstErr
}

// paintBandOp paints one band op: graphics-state save, transform, opacity,
// then the shared draw dispatch, and a final restore.
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

	drawBandOp(chld, page, paintOp, opts, contentH, pageH, useSimple, resName, nextImg, firstErr)

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

	if FakeBoldFor(paintOp) {
		chld.SetLineWidth(paintOp.Size * outlineStrokeRatio)
		chld.TextRenderMode(two)
	}

	chld.TextShow(paintOp.Text)

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

func isSplittable(op *Op) bool {
	return op.Kind == OpFillRect || op.Kind == OpStrokeRect || op.Kind == OpLine
}

// capTablePageBreaks draws a horizontal top edge on pages where a table
// continuation begins mid-grid (split vertical rules at the page top with no
// matching full-width horizontal). Without this, border-collapse rowspan
// tables leave open tops and orphan vertical stubs (wiki awards before
// "2024 Razzie").
func capTablePageBreaks(res *Result, contentH float64) {
	if res == nil || contentH <= 0 || len(res.Ops) == 0 {
		return
	}

	maxPage := capTableMaxPage(res, contentH)
	if maxPage == 0 {
		return
	}

	const eps = 2.0

	_, horiz, vertStarts, vertEnds, horizByY := collectTableBorderSegments(res)
	seal := func(gVal, minX, maxX, borderW, red, green, blue float64) {
		sealBorderGap(res, &horiz, horizByY, eps, gVal, minX, maxX, borderW, red, green, blue)
	}
	coverage := func(y, minX, maxX float64) bool {
		full, _, _, _ := hCoverage(horizByY, y, minX, maxX)

		return full
	}

	// (1) Classic page-top stubs.
	sealPageTopStubs(vertStarts, coverage, seal, maxPage, contentH, eps)

	// (2) Seal incomplete tops of multi-column vertical clusters that start a
	// continuation-page body band (under repeated thead or at page top).
	// Mid-table rowspan holes keep skipped tops so continuous year cells stay
	// unsplit; only the page-fragment open edge is closed.
	sealPageTopClusters(vertStarts, vertEnds, contentH, coverage, seal)

	// Row bottoms: seal when verticals end near a page bottom and no full
	// horizontal closes the strip (next row's top moved to the following page).
	sealPageBottomClusters(vertStarts, vertEnds, contentH, coverage, seal, eps)
}

// sealBorderGap appends a horizontal rule at gVal spanning [minX, maxX] unless
// a near-identical rule already exists.
func sealBorderGap(
	res *Result, horiz *[]hseg, horizByY map[int][]hseg, eps, gVal, minX, maxX, borderW, red, green, blue float64,
) {
	if maxX-minX < 20 || borderW < 0 {
		return
	}

	if borderW < minBorderWidthPt {
		borderW = 0.5
	}
	// Avoid exact duplicates.
	for _, h := range *horiz {
		if math.Abs(h.y-gVal) <= 0.5 && math.Abs(h.x0-minX) <= eps && math.Abs(h.x1-maxX) <= eps {
			return
		}
	}

	op := Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpLine, X: minX, Y: gVal, W: maxX - minX, H: 0,
		Width: borderW, R: red, G: green, B: blue,
	}
	res.Ops = append(res.Ops, op)
	sealed := hseg{minX, maxX, gVal, borderW, red, green, blue}
	*horiz = append(*horiz, sealed)
	horizByY[roundY(gVal)] = append(horizByY[roundY(gVal)], sealed)
}

// sealPageTopClusters closes vertical clusters that start near the top of a
// continuation page's body band.
func sealPageTopClusters(
	vertStarts, vertEnds map[int][]vseg, contentH float64,
	coverage func(y, minX, maxX float64) bool,
	seal func(gVal, minX, maxX, borderW, red, green, blue float64),
) {
	for _, child := range clusterVerticals(vertStarts, vertEnds, true) {
		if child.n < three || child.maxX-child.minX < 20 {
			continue
		}

		if coverage(child.y, child.minX, child.maxX) {
			continue
		}

		page := int(child.y / contentH)
		if page <= 0 {
			continue
		}

		pageTop := float64(page) * contentH
		// Body under thead typically starts within ~header+padding of page top.
		if child.y > pageTop+80 {
			continue
		}

		seal(child.y, child.minX, child.maxX, child.bw, child.r, child.g, child.b)
	}
}

// sealPageBottomClusters closes vertical clusters that end near a page bottom
// with no full horizontal rule (next row's top moved to the following page).
func sealPageBottomClusters(
	vertStarts, vertEnds map[int][]vseg, contentH float64,
	coverage func(y, minX, maxX float64) bool,
	seal func(gVal, minX, maxX, borderW, red, green, blue float64),
	eps float64,
) {
	for _, child := range clusterVerticals(vertStarts, vertEnds, false) {
		if child.n < three || child.maxX-child.minX < 20 {
			continue
		}

		page := int((child.y - layoutSlack) / contentH)
		pageBot := float64(page+1) * contentH
		// Only near the page boundary (row ended as last on page).
		if child.y < pageBot-40 || child.y > pageBot+eps {
			continue
		}

		if coverage(child.y, child.minX, child.maxX) {
			continue
		}

		if page >= 0 {
			seal(child.y, child.minX, child.maxX, child.bw, child.r, child.g, child.b)
		}
	}
}

// vseg is one vertical border segment.
type vseg struct{ x, y0, y1, w, r, g, b float64 }

// hseg is one horizontal border segment.
type hseg struct{ x0, x1, y, w, r, g, b float64 }

// borderCluster is a group of vertical segments sharing a start/end Y.
type borderCluster struct {
	y           float64
	minX, maxX  float64
	bw, r, g, b float64
	n           int
}

func capTableMaxPage(res *Result, contentH float64) int {
	maxPage := 0

	for i := range res.Ops {
		if res.Ops[i].Fixed {
			continue
		}

		p := int(res.Ops[i].Y / contentH)
		if p > maxPage {
			maxPage = p
		}
	}

	return maxPage
}

// collectTableBorderSegments gathers non-fixed vertical/horizontal line ops
// once and groups them by rounded Y.
func collectTableBorderSegments(res *Result) ([]vseg, []hseg, map[int][]vseg, map[int][]vseg, map[int][]hseg) {
	verts, horiz := collectBorderSegmentOps(res.Ops)

	// Group verticals that share a start Y (row top) or end Y (row bottom).
	vertStarts := map[int][]vseg{}
	vertEnds := map[int][]vseg{}
	horizByY := map[int][]hseg{}

	for _, v := range verts {
		vertStarts[roundY(v.y0)] = append(vertStarts[roundY(v.y0)], v)
		vertEnds[roundY(v.y1)] = append(vertEnds[roundY(v.y1)], v)
	}

	for _, h := range horiz {
		horizByY[roundY(h.y)] = append(horizByY[roundY(h.y)], h)
	}

	return verts, horiz, vertStarts, vertEnds, horizByY
}

// collectBorderSegmentOps gathers non-fixed line ops as vertical or horizontal
// border segments.
func collectBorderSegmentOps(ops []Op) ([]vseg, []hseg) {
	var verts []vseg

	var horiz []hseg

	for i := range ops {
		paintOp := &ops[i]
		if paintOp.Fixed || paintOp.Kind != OpLine {
			continue
		}

		if paintOp.H > 2 && (paintOp.W < 1 || paintOp.W < paintOp.H*0.05) {
			verts = append(verts, vseg{
				x: paintOp.X, y0: paintOp.Y, y1: paintOp.Y + paintOp.H,
				w: paintOp.Width, r: paintOp.R, g: paintOp.G, b: paintOp.B,
			})

			continue
		}

		if paintOp.W > 2 && paintOp.H < 1 {
			horiz = append(horiz, hseg{
				x0: paintOp.X, x1: paintOp.X + paintOp.W, y: paintOp.Y,
				w: paintOp.Width, r: paintOp.R, g: paintOp.G, b: paintOp.B,
			})
		}
	}

	return verts, horiz
}

// roundY bins a canvas Y into 0.5pt buckets.
func roundY(y float64) int { return int(math.Round(y * two)) }

// clusterVerticals merges vertical segments sharing a start (byStart) or end
// Y into clusters with the min/max x and dominant stroke.
func clusterVerticals(vertStarts, vertEnds map[int][]vseg, byStart bool) map[int]*borderCluster {
	out := map[int]*borderCluster{}

	groups := vertStarts
	if !byStart {
		groups = vertEnds
	}

	for _, group := range groups {
		for _, val := range group {
			keyY := val.y0
			if !byStart {
				keyY = val.y1
			}

			k := roundY(keyY)

			child := out[k]
			if child == nil {
				child = &borderCluster{y: keyY, minX: val.x, maxX: val.x, bw: val.w, r: val.r, g: val.g, b: val.b, n: 1}
				out[k] = child

				continue
			}

			child.n++
			if val.x < child.minX {
				child.minX = val.x
			}

			if val.x > child.maxX {
				child.maxX = val.x
			}
			// Prefer average y so we sit on the dominant edge.
			child.y = (child.y*float64(child.n-1) + keyY) / float64(child.n)
		}
	}

	return out
}

// hCoverage reports whether horizontal segments near posY span [minX, maxX].
func hCoverage(horizByY map[int][]hseg, posY, minX, maxX float64) (bool, float64, float64, bool) {
	const eps = 2.0

	var covMin, covMax float64

	has := false

	key := roundY(posY)
	for k := key - int(eps*two) - 1; k <= key+int(eps*two)+1; k++ {
		for _, height := range horizByY[k] {
			covMin, covMax, has = mergeCoverageSeg(height, posY, minX, maxX, eps, covMin, covMax, has)
		}
	}

	if !has {
		return false, 0, 0, false
	}

	full := covMin <= minX+eps && covMax >= maxX-eps

	return full, covMin, covMax, true
}

// mergeCoverageSeg extends the running coverage with one segment that lies at
// posY within eps and overlaps the vertical band.
func mergeCoverageSeg(
	height hseg, posY, minX, maxX, eps float64, covMin, covMax float64, has bool,
) (float64, float64, bool) {
	if math.Abs(height.y-posY) > eps {
		return covMin, covMax, has
	}
	// Only count segments that overlap the vertical band.
	if height.x1 < minX-eps || height.x0 > maxX+eps {
		return covMin, covMax, has
	}

	if !has {
		return height.x0, height.x1, true
	}

	if height.x0 < covMin {
		covMin = height.x0
	}

	if height.x1 > covMax {
		covMax = height.x1
	}

	return covMin, covMax, true
}

// sealPageTopStubs seals vertical stubs that start exactly at a page top with
// no closing horizontal rule.
func sealPageTopStubs(
	vertStarts map[int][]vseg,
	coverage func(y, minX, maxX float64) bool,
	seal func(gVal, minX, maxX, borderW, red, green, blue float64),
	maxPage int, contentH, eps float64,
) {
	for p := 1; p <= maxPage; p++ {
		pageTop := float64(p) * contentH

		minX, maxX, borderW, redN, green, blueN, node := pageTopStubBounds(vertStarts, pageTop, eps)

		if node < two {
			continue
		}

		if coverage(pageTop, minX, maxX) {
			continue
		}

		seal(pageTop, minX, maxX, borderW, redN, green, blueN)
	}
}

// pageTopStubBounds scans vertical segments at pageTop for the min/max x and
// dominant stroke of the stub cluster, returning how many stubs matched.
func pageTopStubBounds(
	vertStarts map[int][]vseg, pageTop, eps float64,
) (float64, float64, float64, float64, float64, float64, int) {
	var minX, maxX, borderW, red, green, blue float64

	node := 0

	key := roundY(pageTop)
	for k := key - int(eps*two) - 1; k <= key+int(eps*two)+1; k++ {
		for _, val := range vertStarts[k] {
			if val.y0 < pageTop-eps || val.y0 > pageTop+eps {
				continue
			}

			if node == 0 {
				minX, maxX, borderW, red, green, blue = val.x, val.x, val.w, val.r, val.g, val.b
			} else {
				if val.x < minX {
					minX = val.x
				}

				if val.x > maxX {
					maxX = val.x
				}
			}

			node++
		}
	}

	return minX, maxX, borderW, red, green, blue, node
}

// paginateOps assigns every op a page. Crossing text/image/link ops snap to
// the next page boundary (taking following flow with them so row spacing is
// preserved); then page-break policies are applied as canvas-Y shifts; finally
// pages derive from the final Y positions. Rect-type ops crossing a boundary
// are split by Paint.
func paginateOps(res *Result, contentH float64) []int {
	ensureFlowIndex(res, contentH)
	// Resolve forced section starts before snapping text to provisional page
	// boundaries. Otherwise a row near the boundary of the unbroken flow can
	// move its text alone; a later page-break-before shift then leaves the
	// collapsed-table chrome behind at the old row position.
	for range 10 {
		if !beforeAlways(res, contentH) {
			break
		}
	}

	snapCrossingTextOps(res, contentH)

	paginationFixpoint(res, contentH)

	// After flow has settled, clone <thead> onto continuation pages.
	// Blank avoid-list bands are controlled by preferSplitOverBlank during
	// the fixpoint above (former packAvoidGaps sibling packing was a no-op).
	repeatTableHeaders(res, contentH)
	// Sticky is applied in Paint after rect splitting (see splitCrossingRects).
	opPage := make([]int, len(res.Ops))
	for i := range res.Ops {
		opPage[i] = int(res.Ops[i].Y / contentH)
	}

	return opPage
}

// paginationFixpoint runs the page-break policies until none of them move
// anything or the iteration cap is reached.
func paginationFixpoint(res *Result, contentH float64) {
	for range 10 {
		changed := avoidInside(res, contentH)
		if beforeAlways(res, contentH) {
			changed = true
		}

		if afterBreaks(res, contentH) {
			changed = true
		}

		if rowsIntact(res, contentH) {
			changed = true
		}

		if keepHeadingWithNext(res, contentH) {
			changed = true
		}

		if orphansWidows(res, contentH) {
			changed = true
		}

		if !changed {
			break
		}
	}
}

// snapCrossingTextOps moves text/image/link ops that cross a page boundary to
// the next page (taking following flow with them so row spacing is preserved).
func snapCrossingTextOps(res *Result, contentH float64) {
	for idx := range len(res.Ops) {
		paintOp := &res.Ops[idx]
		if paintOp.Fixed {
			continue
		}

		switch paintOp.Kind {
		case OpText, OpBullet, OpImage, OpLinkURI:
			opH := paintOp.H
			if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
				opH = paintOp.Size * defaultLineHeightRatio
			}

			page := int(paintOp.Y / contentH)
			if page < 0 {
				page = 0
			}

			boundary := float64(page+1) * contentH
			if paintOp.Y+opH > boundary+1e-9 {
				snapOpToBoundary(res, idx, paintOp, boundary)
			}
		case OpFillRect, OpStrokeRect, OpLine, opKindNoop:
		}
	}
}

// snapOpToBoundary shifts one crossing op (and the row chrome under it) onto
// the next page, leaving ascender room above the boundary.
func snapOpToBoundary(res *Result, idx int, paintOp *Op, boundary float64) {
	if boundary-paintOp.Y > layoutEpsilon {
		snapOpForward(res, idx, paintOp, boundary)

		return
	}

	paintOp.Y = boundary
}

// snapOpForward snaps the op forward to the next page with the row chrome
// that belongs to it, keeping the same-row fill tops above the text.
func snapOpForward(res *Result, idx int, paintOp *Op, boundary float64) {
	// Snap text (+ following flow). Same-row fills sit above the
	// baseline; include them in deltaY via minY so their tops clear
	// onto this page with the text (fixture-31 Row 28 white bg).
	// Keep chrome matching tight (one row) so table reports do
	// not inflate deltaY. Never clamp fill tops to `boundary` alone
	// — that collapses them onto the text Y and leaves section
	// gray showing through the ascent/padding band.
	oldY := paintOp.Y
	chrome, minY := rowChromeAbove(res, idx, oldY)
	// Leave room for ascenders above the baseline so snapped
	// lines do not paint into the top margin (page-4/5 bleed).
	lead := 0.0
	if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
		lead = paintOp.Size * pxToPtFactor
		if lead < maxGlueEm {
			lead = 8
		}
	}

	deltaY := boundary + lead - minY
	shiftFlowY(res, idx, idx, oldY-layoutSlack, deltaY)

	for _, j := range chrome {
		o := &res.Ops[j]
		if o.Y < oldY-0.01 {
			o.Y += deltaY
		}
	}
}

// rowChromeAbove collects fill/stroke rects whose band touches oldY, with
// their minimum top Y.
func rowChromeAbove(res *Result, idx int, oldY float64) ([]int, float64) {
	chrome := make([]int, 0, len(res.Ops))

	minY := oldY

	for jdx := range res.Ops {
		obj := &res.Ops[jdx]
		if !rowChromeBandCandidate(obj, jdx, idx, oldY) {
			continue
		}

		chrome = append(chrome, jdx)

		if obj.Y < minY {
			minY = obj.Y
		}
	}

	return chrome, minY
}

// rowChromeBandCandidate reports whether obj is a one-row fill/stroke rect
// sitting on the same band as the op at oldY (not the op itself).
func rowChromeBandCandidate(obj *Op, jdx, idx int, oldY float64) bool {
	if obj.Fixed || jdx == idx {
		return false
	}

	if obj.Kind != OpFillRect && obj.Kind != OpStrokeRect {
		return false
	}

	if obj.H <= 0.5 || obj.H > 40 {
		return false
	}

	if obj.Y > oldY+0.5 || obj.Y+obj.H < oldY-0.5 {
		return false
	}

	return oldY-obj.Y <= obj.H+2
}

// splitCrossingRects truncates splittable ops at each page boundary and keeps
// remainders immediately after (document paint order). Built into a new slice
// rather than mid-slice insert+copy (O(n²) and float-edge infinite loops when
// Y sits on a page boundary — TestTenPageTableReportPerformance hang).
type opSpan struct{ start, end int }

func splitCrossingRects(res *Result, contentH float64, opPage []int) {
	_ = opPage

	if res == nil || contentH <= 0 {
		return
	}
	// Give legacy/test-constructed operations an identity before any rewrite.
	// Layout-generated operations already receive IDs from engine.add. A split
	// fragment keeps its source ID; the box-range remap below is what makes all
	// fragments remain owned by the same element.
	assignOpIDs(res)

	spans := make([]opSpan, len(res.Ops))
	out := make([]Op, 0, len(res.Ops)+maxGlueEm)

	for idx := range res.Ops {
		paintOp := res.Ops[idx]
		start := len(out)

		if paintOp.Fixed || !isSplittable(&paintOp) || paintOp.H <= 0 {
			out = append(out, paintOp)
			spans[idx] = opSpan{start: start, end: len(out) - 1}

			continue
		}

		out = append(out, splitOpFragments(paintOp, contentH)...)
		spans[idx] = opSpan{start: start, end: len(out) - 1}
	}

	res.Ops = out
	remapBoxOpRanges(res.root, spans)
}

// assignOpIDs gives legacy/test-constructed operations an identity, keeping
// every ID unique across the display list.
func assignOpIDs(res *Result) {
	var nextID uint64

	for i := range res.Ops {
		if res.Ops[i].ID > nextID {
			nextID = res.Ops[i].ID
		}
	}

	for i := range res.Ops {
		if res.Ops[i].ID == 0 {
			nextID++
			res.Ops[i].ID = nextID
		}
	}
}

// splitOpFragments truncates one splittable rect at each page boundary and
// returns the fragments in document paint order.
func splitOpFragments(paintOp Op, contentH float64) []Op {
	var frags []Op

	guard := 0
	for paintOp.H > 1e-9 {
		guard++
		if guard > paginationGuardMax {
			// Defensive: never hang the paint pipeline.
			frags = append(frags, paintOp)

			break
		}
		// Epsilon bump so Y exactly on a page top maps to that page, not
		// the previous one (int truncates 52.0-ε down to 51).
		page := int((paintOp.Y + layoutEpsilon) / contentH)
		if page < 0 {
			page = 0
		}

		boundary := float64(page+1) * contentH
		if paintOp.Y+paintOp.H <= boundary+1e-9 {
			frags = append(frags, paintOp)

			break
		}

		firstH := boundary - paintOp.Y
		if firstH <= layoutEpsilon {
			// Start is at/past boundary; advance to next page top via p++.
			paintOp.Y = float64(page+1) * contentH

			continue
		}

		frag := paintOp
		frag.H = firstH
		frags = append(frags, frag)
		paintOp.Y = boundary
		paintOp.H -= firstH
	}

	return frags
}

// remapBoxOpRanges updates the layout-owned operation ranges after a display
// list rewrite. In particular, a source rectangle can become two or more
// page fragments; mapping the box end to the final fragment keeps pagination,
// sticky/fixed stamping, and ElementLocation ownership aligned.
func remapBoxOpRanges(boxNode *box, spans []opSpan) {
	if boxNode == nil {
		return
	}

	if boxNode.opStart >= 0 && boxNode.opEnd >= boxNode.opStart &&
		boxNode.opStart < len(spans) && boxNode.opEnd < len(spans) {
		boxNode.opStart = spans[boxNode.opStart].start
		boxNode.opEnd = spans[boxNode.opEnd].end
	}

	for _, child := range boxNode.children {
		remapBoxOpRanges(child, spans)
	}
}

// stripOrphanRowChrome removes row-sized fills and horizontal rules that sit
// on a page with no overlapping text/bullet/image ink. Page-break snaps move
// the text but leave the previous row's trailing fill / the snapped row's
// background behind, which reads as empty rows (fixture-31 after Row 27).
func stripOrphanRowChrome(res *Result, contentH float64) {
	if res == nil || contentH <= 0 || len(res.Ops) == 0 {
		return
	}

	pageOps := pageIndexedOps(res, contentH)

	stickyTargets := stickySectionChromeTargets(res.root)

	for page := range pageOps {
		pageTop := float64(page) * contentH
		pageBot := pageTop + contentH

		lastInkBot, hasInk := lastInkBottom(res, pageOps[page], pageTop, pageBot)
		if !hasInk {
			continue
		}

		if stripOrphanRows(res, pageOps[page], pageTop, pageBot, lastInkBot) {
			tightenLastRowChrome(res, pageOps[page], pageTop, pageBot, lastInkBot)
		}
		// Pull section washes / borders up to the last row chrome / ink so grey
		// does not pad an empty band to the page bottom (fixture-31 page 1).
		// Only section-colored chrome is clipped — arbitrary tall fills
		// (TestBoundaryFillSplit) are left to the normal page-split remnant.
		contentBot := sectionContentBottom(res, pageOps[page], pageTop, pageBot, lastInkBot)

		if pageBot-contentBot < maxGlueEm {
			continue
		}

		clipSectionTrailingBand(res, pageOps[page], pageTop, pageBot, contentBot)
		// A sticky containing block may begin on this page and continue onto
		// the next one. Its page fragment must still end at the last real row;
		// otherwise the unsplit section wash fills the unused page tail.
		clipStickySectionChrome(res, pageOps[page], pageTop, pageBot, contentBot, stickyTargets)
	}

	closePageLeadingSectionChromeWithTargets(res, contentH, stickyTargets)
}

// pageIndexedOps buckets non-fixed ops by their canvas page.
func pageIndexedOps(res *Result, contentH float64) [][]int {
	maxPage := 0

	for i := range res.Ops {
		if res.Ops[i].Fixed {
			continue
		}

		p := int(res.Ops[i].Y / contentH)
		if p > maxPage {
			maxPage = p
		}
	}

	pageOps := make([][]int, maxPage+1)

	for idx := range res.Ops {
		if res.Ops[idx].Fixed {
			continue
		}

		page := int(res.Ops[idx].Y / contentH)
		if page < 0 || page > maxPage {
			continue
		}

		pageOps[page] = append(pageOps[page], idx)
	}

	return pageOps
}

// opInPageBand reports whether the op's top edge sits within [pageTop, pageBot).
func opInPageBand(op *Op, pageTop, pageBot float64) bool {
	return op.Y >= pageTop-1e-9 && op.Y < pageBot-1e-9
}

// lastInkBottom scans a page for the lowest text/bullet/image ink bottom.
func lastInkBottom(res *Result, idxs []int, pageTop, pageBot float64) (float64, bool) {
	lastInkBot := pageTop
	hasInk := false

	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		var bot float64

		switch paintOp.Kind {
		case OpText, OpBullet:
			height := paintOp.Size * defaultLineHeightRatio
			if paintOp.H > height {
				height = paintOp.H
			}

			if height < minBoxPt {
				height = 4
			}

			bot = paintOp.Y + height
		case OpImage:
			bot = paintOp.Y + paintOp.H
		case OpFillRect, OpStrokeRect, OpLine, OpLinkURI, opKindNoop:
			continue
		}

		hasInk = true

		if bot > lastInkBot {
			lastInkBot = bot
		}
	}

	return lastInkBot, hasInk
}

// stripOrphanRows zeros row-sized fills / rules that sit below the last ink.
// Returns whether anything was stripped.
func stripOrphanRows(res *Result, idxs []int, pageTop, pageBot, lastInkBot float64) bool {
	stripped := false

	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		if stripOrphanRowOp(paintOp, lastInkBot) {
			stripped = true
		}
	}

	return stripped
}

// stripOrphanRowOp zeros one row-sized fill or horizontal rule that sits
// below the last ink. Returns whether it was stripped.
func stripOrphanRowOp(paintOp *Op, lastInkBot float64) bool {
	switch paintOp.Kind {
	case OpFillRect, OpStrokeRect:
		// Row-sized shells whose center sits below the last ink are
		// empty trailing row backgrounds (not the cell that holds the
		// last text, whose center is at/above the baseline band).
		if paintOp.H <= 0.5 || paintOp.H > 40 {
			return false
		}

		if paintOp.Y+paintOp.H/2 > lastInkBot+0.5 {
			paintOp.H = 0

			return true
		}
	case OpLine:
		if paintOp.H >= 1 {
			return false
		}
		// Horizontal rule below the last ink (empty row separator).
		if paintOp.Y > lastInkBot+0.5 {
			paintOp.Width = 0

			return true
		}
	case OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}

	return false
}

// tightenLastRowChrome shortens the last row's fill so padding under the
// final baseline does not read as another empty row (fixture-31 Row 27 cell).
func tightenLastRowChrome(res *Result, idxs []int, pageTop, pageBot, lastInkBot float64) {
	const underPad = 8.0

	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		tightenLastRowOp(paintOp, lastInkBot, underPad)
	}
}

// tightenLastRowOp shortens the last row's fill and pulls the trailing rule
// up under the final baseline.
func tightenLastRowOp(paintOp *Op, lastInkBot, underPad float64) {
	if isLastRowFill(paintOp, lastInkBot, underPad) {
		paintOp.H = lastInkBot + underPad - paintOp.Y
		if paintOp.H < 1 {
			paintOp.H = 1
		}
	}

	if isTrailingRowRule(paintOp, lastInkBot, underPad) {
		paintOp.Y = lastInkBot + underPad
	}
}

// isLastRowFill reports whether paintOp is the row-sized fill that runs under
// the last ink plus more than a small padding.
func isLastRowFill(paintOp *Op, lastInkBot, underPad float64) bool {
	if paintOp.Kind != OpFillRect && paintOp.Kind != OpStrokeRect {
		return false
	}

	return paintOp.H > 0.5 && paintOp.H <= 40 &&
		paintOp.Y < lastInkBot && paintOp.Y+paintOp.H > lastInkBot+underPad+2
}

// isTrailingRowRule reports whether paintOp is the horizontal rule that sits
// just below the last ink's padding band.
func isTrailingRowRule(paintOp *Op, lastInkBot, underPad float64) bool {
	return paintOp.Kind == OpLine && paintOp.H < 1 && paintOp.Width > 0 &&
		paintOp.Y > lastInkBot+underPad+1 && paintOp.Y < lastInkBot+40
}

// sectionContentBottom extends the content bottom over the last row chrome.
func sectionContentBottom(res *Result, idxs []int, pageTop, pageBot, contentBot float64) float64 {
	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		contentBot = extendSectionContentBot(paintOp, contentBot)
	}

	return contentBot
}

// extendSectionContentBot extends the content bottom over one row-chrome op.
func extendSectionContentBot(paintOp *Op, contentBot float64) float64 {
	if (paintOp.Kind == OpFillRect || paintOp.Kind == OpStrokeRect) && paintOp.H > 0.5 && paintOp.H <= 40 {
		if bot := paintOp.Y + paintOp.H; bot > contentBot {
			contentBot = bot
		}
	}

	if paintOp.Kind == OpLine && paintOp.H < 1 && paintOp.Width > 0 && paintOp.Y > contentBot {
		contentBot = paintOp.Y
	}

	return contentBot
}

// clipSectionTrailingBand trims section-colored washes/borders that extend
// into the empty tail of the page.
func clipSectionTrailingBand(res *Result, idxs []int, pageTop, pageBot, contentBot float64) {
	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		clipTrailingBandOp(paintOp, pageTop, pageBot, contentBot)
	}
}

// clipTrailingBandOp trims one trailing section wash or border to contentBot.
func clipTrailingBandOp(paintOp *Op, pageTop, pageBot, contentBot float64) {
	switch paintOp.Kind {
	case OpFillRect:
		if isTrailingSectionWash(paintOp, pageTop, contentBot) {
			paintOp.H = contentBot - paintOp.Y
		}
	case OpLine:
		if isTrailingSectionBorder(paintOp, pageTop, contentBot) {
			paintOp.H = contentBot - paintOp.Y
		} else if isTrailingPageRule(paintOp, pageBot, contentBot) {
			paintOp.Y = contentBot
		}
	case OpStrokeRect, OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}
}

// isTrailingSectionWash reports a continuation-fragment wash that begins at
// the page top and runs past the content bottom. A normal block fill (for
// example a <pre> with bottom padding) must retain its full box height
// through its bottom border, so only the section color matches.
func isTrailingSectionWash(paintOp *Op, pageTop, contentBot float64) bool {
	return paintOp.Y <= pageTop+1 && paintOp.H > 40 && isSectionWashRGB(paintOp.R, paintOp.G, paintOp.B) &&
		paintOp.Y+paintOp.H > contentBot+1 && paintOp.Y < contentBot
}

// isTrailingSectionBorder reports a page-top section side border that runs
// past the content bottom.
func isTrailingSectionBorder(paintOp *Op, pageTop, contentBot float64) bool {
	return paintOp.Y <= pageTop+1 && paintOp.H > 40 && nearSectionBorderRGB(paintOp.R, paintOp.G, paintOp.B) &&
		paintOp.Y+paintOp.H > contentBot+1 && paintOp.Y < contentBot
}

// isTrailingPageRule reports a thin section-colored rule stranded near the
// page bottom, below the last content.
func isTrailingPageRule(paintOp *Op, pageBot, contentBot float64) bool {
	return paintOp.H < 1 && paintOp.Width > 0 && nearSectionBorderRGB(paintOp.R, paintOp.G, paintOp.B) &&
		paintOp.Y > contentBot+1 && paintOp.Y > pageBot-30
}

// clipStickySectionChrome ends sticky-section chrome at the last real row and
// seals the section bottom border when the section continues on the next page.
func clipStickySectionChrome(
	res *Result, idxs []int, pageTop, pageBot, contentBot float64, targets []stickySectionChromeTarget,
) {
	for _, target := range targets {
		for _, i := range idxs {
			paintOp := &res.Ops[i]
			if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) || paintOp.H <= 40 ||
				paintOp.Y+paintOp.H <= contentBot+1 {
				continue
			}

			clipStickySectionChromeOp(paintOp, target, contentBot)
		}

		sealStickySectionBottom(res, target, pageBot, contentBot)
	}
}

// clipStickySectionChromeOp trims one sticky-section wash or side border to
// the content bottom.
func clipStickySectionChromeOp(paintOp *Op, target stickySectionChromeTarget, contentBot float64) {
	switch paintOp.Kind {
	case OpFillRect:
		if target.hasBackground && sameRectFrame(paintOp, target) && sameRGB(paintOp, target.background) {
			paintOp.H = contentBot - paintOp.Y
		}
	case OpLine:
		if target.sideMatches(paintOp) {
			paintOp.H = contentBot - paintOp.Y
		}
	case OpStrokeRect, OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}
}

// sealStickySectionBottom appends the section's bottom border when the
// section continues on the next page and no closing border exists yet.
func sealStickySectionBottom(res *Result, target stickySectionChromeTarget, pageBot, contentBot float64) {
	if target.hasBottom && stickySectionContinuesAfterPage(res, target, pageBot) &&
		!hasStickySectionBottomBorder(res, target, contentBot) {
		res.Ops = append(res.Ops, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: target.x, Y: contentBot, W: target.w,
			Width: target.borderBottomWidth,
			R:     target.borderBottom[0], G: target.borderBottom[1], B: target.borderBottom[2],
		})
	}
}

// closePageLeadingSectionChrome keeps continuation-page section washes and
// side borders aligned with their horizontal closing border. Pagination can
// move the last child row without updating the original block chrome. Only
// sections that contain a sticky box are considered, and their own box
// geometry/colors identify the chrome; unrelated wide rules cannot match.
func closePageLeadingSectionChrome(res *Result, contentH float64) {
	if res == nil {
		return
	}

	closePageLeadingSectionChromeWithTargets(res, contentH, stickySectionChromeTargets(res.root))
}

func closePageLeadingSectionChromeWithTargets(res *Result, contentH float64, targets []stickySectionChromeTarget) {
	if res == nil || res.root == nil || contentH <= 0 {
		return
	}

	if len(targets) == 0 {
		return
	}

	maxPage := pageMaxOfOps(res, contentH)

	for page := 1; page <= maxPage; page++ {
		pageTop := float64(page) * contentH
		pageBottom := pageTop + contentH

		for _, target := range targets {
			closeY := closingBorderY(res, target, pageTop, pageBottom)
			if closeY < 0 {
				continue
			}

			clipSectionChromeToCloseY(res, target, pageTop, pageBottom, closeY)
		}
	}
}

// pageMaxOfOps returns the highest non-fixed page index in the display list.
func pageMaxOfOps(res *Result, contentH float64) int {
	maxPage := 0

	for _, op := range res.Ops {
		if op.Fixed {
			continue
		}

		if page := int(op.Y / contentH); page > maxPage {
			maxPage = page
		}
	}

	return maxPage
}

// closingBorderY finds the lowest horizontal border matching target on the
// page (or -1 when none).
func closingBorderY(res *Result, target stickySectionChromeTarget, pageTop, pageBottom float64) float64 {
	closeY := -1.0

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Fixed || paintOp.Kind != OpLine || paintOp.H >= 1 || paintOp.Y <= pageTop+1 || paintOp.Y >= pageBottom {
			continue
		}

		if !sameHorizontalFrame(paintOp, target) || !sameRGB(paintOp, target.borderBottom) {
			continue
		}

		if paintOp.Y > closeY {
			closeY = paintOp.Y
		}
	}

	return closeY
}

// clipSectionChromeToCloseY trims the target's page-leading washes and side
// borders to the closing border so no chrome runs past the last row.
func clipSectionChromeToCloseY(res *Result, target stickySectionChromeTarget, pageTop, pageBottom, closeY float64) {
	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Fixed || paintOp.Y < pageTop-1 || paintOp.Y > pageTop+1 || paintOp.Y >= pageBottom {
			continue
		}

		clipSectionChromeOp(paintOp, target, closeY)
	}
}

// clipSectionChromeOp trims one page-leading wash or side border to closeY.
func clipSectionChromeOp(paintOp *Op, target stickySectionChromeTarget, closeY float64) {
	switch paintOp.Kind {
	case OpFillRect:
		if isSectionChromeWash(paintOp, target, closeY) {
			paintOp.H = closeY - paintOp.Y
		}
	case OpLine:
		if isSectionChromeSideBorder(paintOp, target, closeY) {
			paintOp.H = closeY - paintOp.Y
		}
	case OpStrokeRect, OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}
}

// isSectionChromeWash reports a page-leading section background wash that
// runs past closeY.
func isSectionChromeWash(paintOp *Op, target stickySectionChromeTarget, closeY float64) bool {
	return target.hasBackground && paintOp.H > 40 && sameRectFrame(paintOp, target) &&
		sameRGB(paintOp, target.background) && paintOp.Y+paintOp.H < closeY
}

// isSectionChromeSideBorder reports a page-leading section side border that
// runs past closeY.
func isSectionChromeSideBorder(paintOp *Op, target stickySectionChromeTarget, closeY float64) bool {
	return paintOp.H > 40 && target.sideMatches(paintOp) && paintOp.Y+paintOp.H < closeY
}

type stickySectionChromeTarget struct {
	x, y, w           float64
	background        [3]float64
	hasBackground     bool
	borderLeft        [3]float64
	borderRight       [3]float64
	borderBottom      [3]float64
	borderBottomWidth float64
	hasBottom         bool
	hasLeft, hasRight bool
}

func stickySectionChromeTargets(root *box) []stickySectionChromeTarget {
	var targets []stickySectionChromeTarget

	var walk func(b, parent *box)
	walk = func(boxNode, parent *box) {
		if boxNode == nil {
			return
		}

		if boxNode.sticky && parent != nil {
			target := stickyTargetFor(parent)
			if !hasDuplicateTarget(targets, target) {
				targets = append(targets, target)
			}
		}

		for _, child := range boxNode.children {
			walk(child, boxNode)
		}
	}
	walk(root, nil)

	return targets
}

// stickyTargetFor builds the section-chrome match target for a sticky box's
// parent: geometry, border colors and the background wash.
func stickyTargetFor(parent *box) stickySectionChromeTarget {
	sty := parent.style
	target := stickySectionChromeTarget{ //nolint:exhaustruct // intentional zero fields
		x:                 parent.x,
		y:                 parent.y,
		w:                 parent.w,
		borderLeft:        sty.BorderLeft.Color,
		borderRight:       sty.BorderRight.Color,
		borderBottom:      sty.BorderBottom.Color,
		borderBottomWidth: sty.BorderBottom.Width,
		hasBottom:         sty.BorderBottom.Width > 0 && sty.BorderBottom.Style != cssDisplayNone,
		hasLeft:           sty.BorderLeft.Width > 0 && sty.BorderLeft.Style != cssDisplayNone,
		hasRight:          sty.BorderRight.Width > 0 && sty.BorderRight.Style != cssDisplayNone,
	}

	if sty.BGColor[3] > 0 {
		target.background = [3]float64{sty.BGColor[0], sty.BGColor[1], sty.BGColor[2]}
		target.hasBackground = true
	}

	return target
}

// hasDuplicateTarget reports whether targets already holds a target with the
// same frame geometry.
func hasDuplicateTarget(targets []stickySectionChromeTarget, target stickySectionChromeTarget) bool {
	for _, prior := range targets {
		if math.Abs(prior.x-target.x) < 0.01 && math.Abs(prior.y-target.y) < 0.01 && math.Abs(prior.w-target.w) < 0.01 {
			return true
		}
	}

	return false
}

func sameHorizontalFrame(op *Op, target stickySectionChromeTarget) bool {
	return target.hasBottom && math.Abs(op.X-target.x) < 1 && math.Abs(op.W-target.w) < 1
}

func stickySectionContinuesAfterPage(res *Result, target stickySectionChromeTarget, pageBottom float64) bool {
	if res == nil {
		return false
	}

	for _, op := range res.Ops {
		if op.Fixed || (op.Kind != OpText && op.Kind != OpBullet) || op.Y < pageBottom {
			continue
		}

		if op.X >= target.x-1 && op.X <= target.x+target.w+1 {
			return true
		}
	}

	return false
}

func hasStickySectionBottomBorder(res *Result, target stickySectionChromeTarget, posY float64) bool {
	if res == nil {
		return false
	}

	for _, op := range res.Ops {
		if op.Fixed || op.Kind != OpLine || op.H >= 1 || math.Abs(op.Y-posY) >= 1 {
			continue
		}

		if sameHorizontalFrame(&op, target) && sameRGB(&op, target.borderBottom) {
			return true
		}
	}

	return false
}

func sameRectFrame(op *Op, target stickySectionChromeTarget) bool {
	return sameHorizontalFrame(op, target)
}

func (target stickySectionChromeTarget) sideMatches(paintOp *Op) bool {
	if paintOp.W > 1 || math.Abs(paintOp.X-target.x) >= 1 && math.Abs(paintOp.X-(target.x+target.w)) >= 1 {
		return false
	}

	if math.Abs(paintOp.X-target.x) < 1 {
		return target.hasLeft && sameRGB(paintOp, target.borderLeft)
	}

	return target.hasRight && sameRGB(paintOp, target.borderRight)
}

func sameRGB(op *Op, rgb [3]float64) bool {
	return math.Abs(op.R-rgb[0]) < 0.01 && math.Abs(op.G-rgb[1]) < 0.01 && math.Abs(op.B-rgb[2]) < 0.01
}

// isSectionWashRGB reports near-neutral cool greys like fixture-31 .section
// (#eceff1). Chromatic washes (e.g. fixture-32 grid #f3e5f5) must not match
// or page-trailing clip steals their height.
func isSectionWashRGB(r, g, b float64) bool {
	if math.Abs(r-g) > 0.035 || math.Abs(g-b) > 0.035 || math.Abs(r-b) > 0.035 {
		return false
	}

	return r > 0.88 && g > 0.88 && b > 0.88 && r < 0.97 && g < 0.97 && b < 0.97
}

// shiftFlowY moves the ops of the target range [from,to] - plus every op
// strictly below fromY - down by deltaY canvas points. Ops of earlier boxes
// that touch fromY exactly (collapsed margins) are left alone so the
// page-break fixpoint converges instead of dragging boundary ops along each
// iteration. Box.y is kept in sync for boxes whose top moved.
func shiftFlowY(res *Result, from, toIdx int, fromY, deltaY float64) {
	if res == nil || len(res.Ops) == 0 || deltaY == 0 {
		return
	}

	ensureFlowIndex(res, flowIndexPageSize(res))

	shiftOpsRange(res, from, toIdx, deltaY)

	startPage := int(fromY / res.flowPageSize)
	if startPage < 0 {
		startPage = 0
	}

	shiftFlowOps(res, from, toIdx, fromY, deltaY, startPage)

	if res.root == nil {
		return
	}

	if len(res.flowBoxes) == 0 {
		ensureFlowBoxIndex(res, flowBoxList(res))
	}

	shiftFlowBoxes(res, from, toIdx, fromY, startPage, deltaY)
}

// shiftOpsRange shifts the non-fixed ops of [from,to] by deltaY.
func shiftOpsRange(res *Result, from, to int, deltaY float64) {
	for i := from; i <= to; i++ {
		if i < 0 || i >= len(res.Ops) || res.Ops[i].Fixed {
			continue
		}

		shiftIndexedOp(res, i, deltaY)
	}
}

// flowBoxList returns the flattened box list, caching it on res.
func flowBoxList(res *Result) []*box {
	boxes := res.boxes
	if len(boxes) == 0 {
		boxes = make([]*box, 0)
		flattenBoxes(res.root, &boxes)
		res.boxes = boxes
	}

	return boxes
}

// shiftFlowOps moves the ops of every page bucket in the direction of the
// shift: positive shifts process buckets from the end so an operation moved
// to another bucket is never visited twice; negative shifts go ascending so
// an operation moved backward is not revisited.
func shiftFlowOps(res *Result, from, toIdx int, fromY, deltaY float64, startPage int) {
	if deltaY > 0 {
		for p := len(res.flowPages) - 1; p >= startPage; p-- {
			shiftOpsBucket(res, p, from, toIdx, fromY, deltaY)
		}

		return
	}

	for p := startPage; p < len(res.flowPages); p++ {
		shiftOpsBucket(res, p, from, toIdx, fromY, deltaY)
	}
}

// shiftOpsBucket shifts the ops of one page bucket that sit strictly below
// fromY. Removing the current item in shiftIndexedOp swaps the bucket's last
// item into its place; re-read the live bucket each step so a stale slice
// header cannot re-process an op that already left the page (that bug caused
// double negative shifts and infinite positive-shift loops).
func shiftOpsBucket(res *Result, page, from, toIdx int, fromY, deltaY float64) {
	if page < 0 || page >= len(res.flowPages) {
		return
	}

	for jdx := 0; ; {
		bucket := res.flowPages[page]
		if jdx >= len(bucket) {
			return
		}

		idx := bucket[jdx]
		if (idx >= from && idx <= toIdx) || res.Ops[idx].Y <= fromY {
			jdx++

			continue
		}

		oldPage := res.flowPageOf[idx]
		shiftIndexedOp(res, idx, deltaY)

		if res.flowPageOf[idx] == oldPage {
			jdx++
		}
	}
}

// shiftFlowBoxes moves the box buckets in the direction of the shift.
func shiftFlowBoxes(res *Result, from, toIdx int, fromY float64, startPage int, deltaY float64) {
	if deltaY > 0 {
		for p := len(res.flowBoxes) - 1; p >= startPage; p-- {
			shiftBoxesBucket(res, p, from, toIdx, fromY, startPage, deltaY)
		}

		return
	}

	for p := startPage; p < len(res.flowBoxes); p++ {
		shiftBoxesBucket(res, p, from, toIdx, fromY, startPage, deltaY)
	}
}

// shiftBoxesBucket shifts the boxes of one page bucket whose top moved.
// Re-reads res.flowBoxes[page] each step (same swap-remove hazard as ops).
func shiftBoxesBucket(res *Result, page, from, toIdx int, fromY float64, startPage int, deltaY float64) {
	if page < 0 || page >= len(res.flowBoxes) {
		return
	}

	for jdx := 0; ; {
		bucket := res.flowBoxes[page]
		if jdx >= len(bucket) {
			return
		}

		boxIndex := bucket[jdx]
		if skipBoxShift(res, boxIndex, from, toIdx, fromY, startPage) {
			jdx++

			continue
		}

		oldPage := res.flowBoxPage[boxIndex]
		shiftIndexedBox(res, boxIndex, deltaY)

		if res.flowBoxPage[boxIndex] == oldPage {
			jdx++
		}
	}
}

// skipBoxShift reports whether a box on startPage should stay put during a
// flow shift (top at/above fromY, except the target op range sitting on fromY).
func skipBoxShift(res *Result, boxIndex, from, toIdx int, fromY float64, startPage int) bool {
	if startPage != res.flowBoxPage[boxIndex] {
		return false
	}

	b := res.boxes[boxIndex]
	if b.y > fromY {
		return false
	}

	if b.y == fromY && b.opStart >= from && b.opEnd <= toIdx {
		return false
	}

	return true
}

func flowIndexPageSize(res *Result) float64 {
	if res.flowPageSize > 0 {
		return res.flowPageSize
	}

	return 1
}

func ensureFlowIndex(res *Result, pageSize float64) {
	if res == nil || len(res.Ops) == 0 || pageSize <= 0 {
		return
	}

	if res.flowPageSize == pageSize && len(res.flowPageOf) == len(res.Ops) {
		return
	}

	res.flowPageSize = pageSize
	res.flowPages, res.flowPageOf, res.flowPos = buildFlowOpIndex(res.Ops, pageSize)

	boxes := res.boxes
	if len(boxes) == 0 && res.root != nil {
		boxes = make([]*box, 0)
		flattenBoxes(res.root, &boxes)
		res.boxes = boxes
	}

	ensureFlowBoxIndex(res, boxes)
}

// buildFlowOpIndex buckets non-fixed ops by their canvas page.
func buildFlowOpIndex(ops []Op, pageSize float64) ([][]int, []int, []int) {
	maxPage := 0

	for i := range ops {
		if ops[i].Fixed {
			continue
		}

		page := int(ops[i].Y / pageSize)
		if page < 0 {
			page = 0
		}

		if page > maxPage {
			maxPage = page
		}
	}

	pages := make([][]int, maxPage+1)
	pageOf := make([]int, len(ops))
	pos := make([]int, len(ops))

	for i := range pageOf {
		pageOf[i] = -1
		pos[i] = -1
	}

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page := int(ops[idx].Y / pageSize)
		if page < 0 {
			page = 0
		}

		pageOf[idx] = page
		pos[idx] = len(pages[page])
		pages[page] = append(pages[page], idx)
	}

	return pages, pageOf, pos
}

func ensureFlowBoxIndex(res *Result, boxes []*box) {
	if res == nil {
		return
	}

	if len(res.flowBoxPage) == len(boxes) && len(res.flowBoxes) > 0 {
		return
	}

	res.flowBoxes = make([][]int, len(res.flowPages))
	res.flowBoxPage = make([]int, len(boxes))
	res.flowBoxPos = make([]int, len(boxes))

	for idx, b := range boxes {
		b.flowIndex = idx

		page := int(b.y / res.flowPageSize)
		if page < 0 {
			page = 0
		}

		for len(res.flowBoxes) <= page {
			res.flowBoxes = append(res.flowBoxes, nil)
		}

		res.flowBoxPage[idx] = page
		res.flowBoxPos[idx] = len(res.flowBoxes[page])
		res.flowBoxes[page] = append(res.flowBoxes[page], idx)
	}
}

func shiftIndexedOp(res *Result, index int, deltaY float64) {
	if index < 0 || index >= len(res.Ops) || res.Ops[index].Fixed {
		return
	}

	oldPage := res.flowPageOf[index]
	res.Ops[index].Y += deltaY

	newPage := flowPageOfY(res.Ops[index].Y, res.flowPageSize)
	if oldPage == newPage {
		return
	}

	removeFromFlowBucket(&res.flowPages, res.flowPos, oldPage, index)
	appendToFlowBucket(&res.flowPages, &res.flowPageOf, &res.flowPos, index, newPage)
}

func shiftIndexedBox(res *Result, index int, deltaY float64) {
	if index < 0 || index >= len(res.boxes) {
		return
	}

	b := res.boxes[index]
	oldPage := res.flowBoxPage[index]
	b.y += deltaY

	newPage := flowPageOfY(b.y, res.flowPageSize)
	if oldPage == newPage {
		return
	}

	removeFromFlowBucket(&res.flowBoxes, res.flowBoxPos, oldPage, index)
	appendToFlowBucket(&res.flowBoxes, &res.flowBoxPage, &res.flowBoxPos, index, newPage)
}

// flowPageOfY maps a canvas Y to its page index.
func flowPageOfY(y, pageSize float64) int {
	page := int(y / pageSize)
	if page < 0 {
		return 0
	}

	return page
}

// removeFromFlowBucket swaps the entry out of its bucket (keeping cursor
// positions valid for shiftIndexedOp's in-place iteration).
func removeFromFlowBucket(buckets *[][]int, pos []int, page, index int) {
	if buckets == nil || page < 0 || page >= len(*buckets) {
		return
	}

	bucket := (*buckets)[page]
	slot := pos[index]

	if slot < 0 || slot >= len(bucket) {
		return
	}

	last := bucket[len(bucket)-1]
	bucket[slot] = last
	pos[last] = slot
	(*buckets)[page] = bucket[:len(bucket)-1]
}

// appendToFlowBucket registers the entry in its new page bucket.
// buckets is a pointer so growing the outer page slice is visible to the caller.
func appendToFlowBucket(buckets *[][]int, pageOf *[]int, pos *[]int, index, page int) {
	if buckets == nil {
		return
	}

	for len(*buckets) <= page {
		*buckets = append(*buckets, nil)
	}

	(*pageOf)[index] = page
	(*pos)[index] = len((*buckets)[page])
	(*buckets)[page] = append((*buckets)[page], index)
}

// shiftOpsOnly moves ops in [from,to] by dy without dragging later flow.
// Used when rejoining a page-break-after:avoid box to a following box that
// already sits on the next page.
func shiftOpsOnly(res *Result, from, tOrigin int, deltaY float64) {
	if res == nil || len(res.Ops) == 0 || deltaY == 0 {
		return
	}

	ensureFlowIndex(res, flowIndexPageSize(res))

	for i := from; i <= tOrigin; i++ {
		if i < 0 || i >= len(res.Ops) || res.Ops[i].Fixed {
			continue
		}

		shiftIndexedOp(res, i, deltaY)
	}
}

// avoidInside walks post-order and moves page-break-inside: avoid boxes wholly
// to the next page when they span multiple pages but fit one content height.
func avoidInside(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(boxNode *box) bool {
		changed := false

		for _, c := range boxNode.children {
			if walk(c) {
				changed = true
			}
		}

		if boxNode.style.PageBreakInside == pageBreakAvoid && boxNode.height > 0 {
			if keepTogetherForAvoid(res, boxNode, contentH) {
				changed = true
			}
		}

		return changed
	}

	return walk(res.root)
}

// keepTogetherForAvoid shifts one page-break-inside:avoid box wholly to the
// next page when it straddles the boundary but fits a full page. Returns
// whether a shift happened.
func keepTogetherForAvoid(res *Result, boxNode *box, contentH float64) bool {
	height := boxNode.height
	// Prefer ink extent when taller than the border box (rowspan /
	// deferred paint can make ops protrude past b.h — wiki awards).
	if ink := boxInkExtent(res, boxNode); ink > height {
		height = ink
	}

	layoutOut := int(boxNode.y / contentH)
	hi := int((boxNode.y + height) / contentH)

	if hi <= layoutOut || height > contentH+0.01 {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	// Prefer splitting over large empty bands. Use border-box
	// height (b.h), not ink: after line-snap, ink can span a
	// page gap while the box is still a short list item —
	// classifying by ink disabled the short-box guard and
	// cascaded 100–150pt gaps (wiki references).
	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}
	// Large boxes: also prefer split when less than half the box
	// fits (rowspan tables / tall avoid blocks).
	if remaining < boxNode.height*0.5 && boxNode.height > contentH*0.35 {
		return false
	}

	dy := float64(layoutOut+1)*contentH - boxNode.y
	if dy <= layoutSlack {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

// boxInkExtent returns the bottom edge of the box's ink ops (boxNode.y when
// the op range is invalid or holds no ink).
func boxInkExtent(res *Result, boxNode *box) float64 {
	if boxNode.opStart > boxNode.opEnd || boxNode.opStart < 0 || boxNode.opEnd >= len(res.Ops) {
		return boxNode.y
	}

	bot := boxNode.y

	for k := boxNode.opStart; k <= boxNode.opEnd; k++ {
		paintOp := res.Ops[k]
		outBox := paintOp.Y

		switch paintOp.Kind {
		case OpText, OpBullet:
			outBox += paintOp.Size * defaultLineHeightRatio
		case OpFillRect, OpStrokeRect, OpLine, OpImage, OpLinkURI, opKindNoop:
			if paintOp.H > 0 {
				outBox += paintOp.H
			}
		}

		if outBox > bot {
			bot = outBox
		}
	}

	return bot
}

// beforeAlways walks pre-order and moves page-break-before: always boxes to a
// fresh page after everything preceding them. Every matching box is handled in
// one walk so multi-section reports (benchmarks, long fixtures) are not capped
// by the outer fixpoint iteration budget. After each shift, prefix maxima are
// rebuilt because later ops and boxes have moved.
func beforeAlways(res *Result, contentH float64) bool {
	if res == nil || res.root == nil || contentH <= 0 {
		return false
	}

	prefixMaxY := prefixMaxOfOps(res.Ops)
	changed := false

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.style.PageBreakBefore == pageBreakAlways {
			if shiftForcedBreak(res, boxNode, prefixMaxY, contentH) {
				prefixMaxY = prefixMaxOfOps(res.Ops)
				changed = true
			}
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}

	walk(res.root)

	return changed
}

// prefixMaxOfOps returns prefixMax[i] = max Y of ops[0:i].
func prefixMaxOfOps(ops []Op) []float64 {
	prefixMaxY := make([]float64, len(ops)+1)
	for i, op := range ops {
		prefixMaxY[i+1] = prefixMaxY[i]
		if op.Y > prefixMaxY[i+1] {
			prefixMaxY[i+1] = op.Y
		}
	}

	return prefixMaxY
}

// shiftForcedBreak moves a page-break-before:always box below the last op
// that precedes it. Returns whether it shifted.
func shiftForcedBreak(res *Result, boxNode *box, prefixMaxY []float64, contentH float64) bool {
	lastBefore := 0.0
	if boxNode.opStart > 0 && boxNode.opStart < len(prefixMaxY) {
		lastBefore = prefixMaxY[boxNode.opStart]
	}

	loPage := int(boxNode.y / contentH)
	lastPage := int(lastBefore / contentH)

	if loPage <= lastPage {
		dy := float64(lastPage+1)*contentH - boxNode.y
		if dy > 0 {
			shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

			return true
		}
	}

	return false
}

// afterBreaks walks in document order and applies page-break-after:
// always|avoid to the box that follows each box in flow.
func afterBreaks(res *Result, contentH float64) bool {
	boxes := flattenAllBoxes(res.root)

	changed := false

	for i, boxNode := range boxes {
		next := nextFlowSibling(boxes, i)
		if next == nil || boxNode.opStart > boxNode.opEnd {
			continue
		}

		lastPage := int(boxMaxOpY(res, boxNode) / contentH)

		switch boxNode.style.PageBreakAfter {
		case pageBreakAlways:
			if shiftAfterAlways(res, next, lastPage, contentH) {
				changed = true
			}
		case pageBreakAvoid:
			if keepAfterAvoid(res, boxNode, next, lastPage, contentH) {
				changed = true
			}
		}
	}

	return changed
}

// flattenAllBoxes returns the box tree in document order.
func flattenAllBoxes(root *box) []*box {
	var boxes []*box

	var flatten func(b *box)
	flatten = func(b *box) {
		boxes = append(boxes, b)

		for _, c := range b.children {
			flatten(c)
		}
	}

	if root != nil {
		flatten(root)
	}

	return boxes
}

// nextFlowSibling returns the first box after i that emitted ops.
func nextFlowSibling(boxes []*box, i int) *box {
	for j := i + 1; j < len(boxes); j++ {
		if boxes[j].opStart <= boxes[j].opEnd {
			return boxes[j]
		}
	}

	return nil
}

// boxMaxOpY returns the lowest Y of any op in the box's range.
func boxMaxOpY(res *Result, boxNode *box) float64 {
	lastY := res.Ops[boxNode.opStart].Y
	for k := boxNode.opStart + 1; k <= boxNode.opEnd; k++ {
		if res.Ops[k].Y > lastY {
			lastY = res.Ops[k].Y
		}
	}

	return lastY
}

// shiftAfterAlways pushes the next box to the page after this box's last op.
func shiftAfterAlways(res *Result, next *box, lastPage int, contentH float64) bool {
	dy := float64(lastPage+1)*contentH - next.y
	if dy <= 0 {
		return false
	}

	shiftFlowY(res, next.opStart, next.opEnd, next.y, dy)

	return true
}

// keepAfterAvoid keeps a page-break-after:avoid box with the following box
// across page boundaries: it clears the landing band, then sits the box just
// above the (possibly pushed) sibling. Returns whether it moved.
func keepAfterAvoid(res *Result, boxNode, next *box, lastPage int, contentH float64) bool {
	// Do NOT collapse natural flow spacing when they already share a
	// page (that pulled .keep boxes up onto paragraph baselines —
	// fixture-08 Forms index overlap).
	nextPage := int(next.y / contentH)
	if nextPage <= lastPage {
		return false
	}
	// Place the heading on next's page without a full-page shiftFlowY
	// (that blanked pages after avoid-inside tables). Clear the
	// page-top band first: paginateOps may already have snapped a
	// prior paragraph's continuation to pageStart — that text is
	// NOT `next` (next is the following sibling), so we must push
	// every op in the landing band, not only next.
	pageStart := float64(nextPage) * contentH

	need := avoidKeepBand(res, boxNode)

	const gap = 10.0
	need += gap
	bandTop := pageStart + need
	minY, minIdx := minOpYOnPage(res, nextPage, contentH, bandTop)

	if minIdx >= 0 && minY < bandTop-0.01 {
		push := bandTop - minY
		shiftFlowY(res, minIdx, minIdx, minY-layoutSlack, push)
	}

	target := bandTop - need // == pageStart when band was cleared
	if target < pageStart {
		target = pageStart
	}
	// Prefer sitting just above the (possibly pushed) next sibling.
	if next.y-need > target {
		target = next.y - need
	}

	if target < pageStart {
		target = pageStart
	}

	deltaY := target - boxNode.y
	if deltaY <= layoutSlackFine {
		return false
	}

	shiftOpsOnly(res, boxNode.opStart, boxNode.opEnd, deltaY)
	boxNode.y += deltaY

	return true
}

// minOpYOnPage finds the lowest non-fixed op on the page, starting from
// bandTop (returns -1 when none sit above it).
func minOpYOnPage(res *Result, nextPage int, contentH, bandTop float64) (float64, int) {
	minY := bandTop
	minIdx := -1

	for idx := range res.Ops {
		paintOp := &res.Ops[idx]
		if paintOp.Fixed {
			continue
		}

		if int(paintOp.Y/contentH) != nextPage {
			continue
		}

		if paintOp.Y < minY {
			minY = paintOp.Y
			minIdx = idx
		}
	}

	return minY, minIdx
}

// avoidKeepBand measures how much room the box needs on next's page (ink
// extent, else border-box height).
func avoidKeepBand(res *Result, boxNode *box) float64 {
	need := boxNode.height
	if need < 1 {
		need = 12
	}

	if ink := avoidKeepInkExtent(res, boxNode); ink > need {
		need = ink
	}

	return need
}

// avoidKeepInkExtent returns the ink band height of the box's ops (0 when the
// op range is invalid).
func avoidKeepInkExtent(res *Result, boxNode *box) float64 {
	if boxNode.opStart > boxNode.opEnd || boxNode.opStart < 0 || boxNode.opEnd >= len(res.Ops) {
		return 0
	}

	top, bot := boxNode.y, boxNode.y

	for k := boxNode.opStart; k <= boxNode.opEnd; k++ {
		yStart, yEnd := opInkEdges(res.Ops[k])

		if yStart < top {
			top = yStart
		}

		if yEnd > bot {
			bot = yEnd
		}
	}

	return bot - top
}

// opInkEdges returns the top and bottom ink edges of one op.
func opInkEdges(paintOp Op) (float64, float64) {
	yStart, yEnd := paintOp.Y, paintOp.Y

	switch paintOp.Kind {
	case OpText, OpBullet:
		yStart = paintOp.Y - paintOp.Size*ascentRatio
		yEnd = paintOp.Y + paintOp.Size*bulletGapRatio
	case OpLine:
		if paintOp.H == 0 {
			yEnd = paintOp.Y + math.Max(paintOp.Width, 1)
		} else {
			yEnd = paintOp.Y + paintOp.H
		}
	case OpFillRect, OpStrokeRect, OpImage, OpLinkURI, opKindNoop:
		if paintOp.H > 0 {
			yEnd = paintOp.Y + paintOp.H
		}
	}

	return yStart, yEnd
}

// rowsIntact keeps each table row on a single page: a row spanning multiple
// pages moves wholly to the next.
func rowsIntact(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(boxNode *box) bool {
		changed := false

		for _, c := range boxNode.children {
			if walk(c) {
				changed = true
			}
		}

		for _, row := range boxNode.rows {
			if shiftRowToPage(res, row, contentH) {
				changed = true
			}
		}

		return changed
	}

	return walk(res.root)
}

// shiftRowToPage moves a table row wholly to the next page when it spans
// multiple pages. Returns whether it moved.
func shiftRowToPage(res *Result, row []*box, contentH float64) bool {
	if len(row) == 0 {
		return false
	}

	first, last, rowTop, rowBottom, ok := rowOpGeometry(row)
	if !ok {
		return false
	}

	layoutOut, hi := int(rowTop/contentH), int(rowBottom/contentH)
	if hi <= layoutOut {
		return false
	}
	// Move only to the next page start. Using hi*contentH when the
	// row's measured bottom spans multiple pages (e.g. rowspan
	// paint height leaking into rowBoxH) skipped blank pages
	// between filmography and awards on long wiki tables.
	deltaY := float64(layoutOut+1)*contentH - rowTop
	if deltaY <= layoutSlack {
		return false
	}
	// fromY slightly above rowTop so border-collapse grid
	// lines that sit exactly on the row edge (and later
	// rows / chrome below) shift with the cells — otherwise
	// content moves and the grid stays behind (gapped /
	// misaligned music-video tables across page breaks).
	shiftFlowY(res, first, last, rowTop-layoutSlack, deltaY)

	return true
}

// rowOpGeometry returns the op range and starting-row geometry of a row:
// opStart..opEnd across its cells and the row band top/bottom (rowspan cells
// use their starting-row height, not the full span paint extent).
func rowOpGeometry(row []*box) (int, int, float64, float64, bool) {
	first, last := -1, -1
	haveGeom := false

	var rowTop, rowBottom float64

	for _, cell := range row {
		first, last = extendRowOpRange(first, last, cell)
		// Use starting-row geometry, not full rowspan paint extent.
		// Rowspan cells emit bottom borders at y+h (full span); scanning
		// those ops made the first row look multi-page and cascaded
		// blank pages (wiki awards tables with rowspan=10+).
		top, bot := rowCellGeometry(cell)

		if !haveGeom {
			rowTop, rowBottom, haveGeom = top, bot, true

			continue
		}

		if top < rowTop {
			rowTop = top
		}

		if bot > rowBottom {
			rowBottom = bot
		}
	}

	return first, last, rowTop, rowBottom, first >= 0 && haveGeom
}

// extendRowOpRange widens the row's op range with one cell's ops.
func extendRowOpRange(first, last int, cell *box) (int, int) {
	if cell.opStart > cell.opEnd {
		return first, last
	}

	if first < 0 {
		first = cell.opStart
	}

	if cell.opEnd > last {
		last = cell.opEnd
	}

	return first, last
}

// rowCellGeometry returns the starting-row band of one cell: rowspan cells
// use their starting-row height, not the full span paint extent.
func rowCellGeometry(cell *box) (float64, float64) {
	top := cell.y

	h := cell.height
	if cell.rowSpan > 1 && cell.rowBoxH > 0 {
		h = cell.rowBoxH
	}

	return top, top + h
}

// keepHeadingWithNext moves a heading to the next page when it would sit alone
// near the bottom (less than ~2 line-heights of room for following content).
func keepHeadingWithNext(res *Result, contentH float64) bool {
	if res.root == nil {
		return false
	}

	boxes := flattenAllBoxes(res.root)

	changed := false

	for idx, boxNode := range boxes {
		if boxNode.node == nil || !isHeadingName(boxNode.node.Name) || boxNode.opStart > boxNode.opEnd {
			continue
		}

		page := int(boxNode.y / contentH)

		room := float64(page+1)*contentH - (boxNode.y + boxNode.height)
		if room >= twoLineRoomPt { // ~2 lines at 12pt
			continue
		}
		// Find next flow sibling with ops.
		next := nextBelowSibling(boxes, idx, boxNode.y)
		if next == nil {
			continue
		}

		nextPage := int(next.y / contentH)
		if nextPage > page {
			continue // already separated by a break
		}
		// Move heading + following content together so the heading does not
		// land on top of a line that already snapped to the next page.
		dy := float64(page+1)*contentH - boxNode.y
		if dy > 0 {
			shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

			changed = true
		}
	}

	return changed
}

// nextBelowSibling returns the first flow sibling with ops at or below y.
func nextBelowSibling(boxes []*box, from int, y float64) *box {
	for j := from + 1; j < len(boxes); j++ {
		if boxes[j].opStart <= boxes[j].opEnd && boxes[j].y >= y {
			return boxes[j]
		}
	}

	return nil
}

func isHeadingName(name string) bool {
	switch name {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}

	return false
}

// orphansWidows enforces CSS Fragmentation Level 3 Rule 3 (widows/orphans)
// when a leaf block has countable line boxes, and falls back to a geometric
// short-block heuristic when line counts are unavailable.
//
// Class B breaks are rejected when lines-before < orphans or lines-after <
// widows (or the block has fewer lines than orphans+widows can satisfy). If
// the whole block fits on the next page it is shifted; otherwise progress
// escape leaves the break (content taller than one page). Forced breaks
// (page-break-before/after: always) run earlier and are not undone here.
// break-inside: avoid remains higher priority via avoidInside.
func orphansWidows(res *Result, contentH float64) bool {
	if res.root == nil || contentH <= 0 {
		return false
	}

	changed := false

	var walk func(b *box)
	walk = func(boxNode *box) {
		for _, c := range boxNode.children {
			walk(c)
		}

		if orphansWidowsBox(res, boxNode, contentH) {
			changed = true
		}
	}
	walk(res.root)

	return changed
}

// orphansWidowsBox applies Rule 3 (or the geometric fallback) to one block
// box. Returns whether anything moved.
func orphansWidowsBox(res *Result, boxNode *box, contentH float64) bool {
	if boxNode.kind != displayBlock || boxNode.height <= 0 || boxNode.opStart > boxNode.opEnd {
		return false
	}
	// Nested block containers: children apply Rule 3; only heuristic on
	// short straddlers here.
	if hasNestedFlowChild(boxNode) {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}

	lines := countBlockLineYs(res, boxNode)
	if len(lines) == 0 {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}

	return enforceOrphansWidows(res, boxNode, lines, contentH)
}

// enforceOrphansWidows applies CSS Fragmentation Rule 3 (orphans/widows) to a
// leaf block that straddles a page boundary. Returns whether it moved.
func enforceOrphansWidows(res *Result, boxNode *box, lines []float64, contentH float64) bool {
	orphans, widows := resolveOrphansWidows(boxNode)

	layoutOut := int(boxNode.y / contentH)
	hIdx := int((boxNode.y + boxNode.height) / contentH)

	if hIdx <= layoutOut {
		return false
	}

	boundary := float64(layoutOut+1) * contentH
	before, after := countLinesAroundBoundary(lines, boundary)
	// Rule 3 applies to Class B breaks *between line boxes*. If all text
	// lines sit on one side of the boundary (only padding/bg straddles),
	// do not keep-together tall boxes — fall back to the short heuristic.
	if before == 0 || after == 0 {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}
	// Rule 3: legal Class B break only if both sides meet the minima.
	if before >= orphans && after >= widows {
		return false
	}
	// Keep the block together when it fits one page; else progress escape.
	// Same blank-band guard as avoidInside: do not open a large empty
	// region on the current page for a short keep-together.
	if boxNode.height > contentH+0.01 {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}

	dy := float64(hIdx)*contentH - boxNode.y
	if dy <= layoutEpsilon {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

// resolveOrphansWidows returns the effective orphans/widows minima.
func resolveOrphansWidows(boxNode *box) (int, int) {
	orphans := boxNode.style.Orphans
	if orphans < 1 {
		orphans = defaultOrphansWidows
	}

	widows := boxNode.style.Widows
	if widows < 1 {
		widows = defaultOrphansWidows
	}

	return orphans, widows
}

// countLinesAroundBoundary splits line baselines into before/after counts.
func countLinesAroundBoundary(lines []float64, boundary float64) (int, int) {
	before, after := 0, 0

	for _, y := range lines {
		if y < boundary-1e-6 {
			before++
		} else {
			after++
		}
	}

	return before, after
}

// orphansWidowsHeuristic is the phase-18 geometric fallback: short blocks
// (~2–4 lines) that straddle a page boundary move wholly when they fit.
func orphansWidowsHeuristic(res *Result, boxNode *box, contentH float64) bool {
	if boxNode.height < 14 || boxNode.height > 60 {
		return false
	}

	layoutOut := int(boxNode.y / contentH)
	hIdx := int((boxNode.y + boxNode.height) / contentH)

	if hIdx <= layoutOut || boxNode.height > contentH {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}

	dy := float64(hIdx)*contentH - boxNode.y
	if dy <= layoutEpsilon {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

// preferSplitOverBlank reports whether a keep-together shift would leave an
// unacceptably large empty band on the current page. Shared by avoidInside
// and orphans/widows so dense page-break-inside:avoid lists do not cascade
// expanding gaps between consecutive short blocks.
//
// h should be the border-box height (not ink): line-snap can inflate ink
// across a page boundary without making the box "tall".
func preferSplitOverBlank(remaining, height, contentH float64) bool {
	if contentH <= 0 {
		return false
	}
	// Never blank more than half a page to keep a box together.
	if remaining > contentH*0.5 {
		return true
	}
	// Short/medium boxes (list items, citations, cards ~1–4 lines): only
	// keep-together when nearly at the page end. Each keep-together does
	// shiftFlowY on following siblings; sequences of short avoid items
	// otherwise expand inter-item gaps by remaining on every fixpoint
	// iteration (wiki references left 26–38pt bands).
	if height > 0 && height < contentH*0.35 {
		// Allow at most ~1.2 line-heights of trailing blank (or half the
		// box), whichever is larger — true end-of-page overflow only.
		// Tighter than the prior 24pt/0.75h guard so modest remainders
		// never keep short avoid siblings apart.
		maxBlank := 14.0
		if height*0.5 > maxBlank {
			maxBlank = height * halfRatio
		}

		if remaining > maxBlank {
			return true
		}
	}

	return false
}

func hasNestedFlowChild(boxNode *box) bool {
	for _, c := range boxNode.children {
		if c.kind == displayBlock || c.kind == displayTable {
			return true
		}
	}

	return false
}

// countBlockLineYs returns distinct text/bullet baseline Y positions in the
// box's op range (approximate line boxes for an IFC leaf block).
func countBlockLineYs(res *Result, boxNode *box) []float64 {
	if res == nil || boxNode.opStart > boxNode.opEnd || boxNode.opStart < 0 {
		return nil
	}

	const eps = 0.5

	yCoords := make([]float64, 0, maxGlueEm)

	end := boxNode.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	for i := boxNode.opStart; i <= end; i++ {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText && paintOp.Kind != OpBullet {
			continue
		}

		if !hasLineY(yCoords, paintOp.Y, eps) {
			yCoords = append(yCoords, paintOp.Y)
		}
	}

	return yCoords
}

// hasLineY reports whether a baseline Y is already recorded within eps.
func hasLineY(yCoords []float64, y, eps float64) bool {
	for _, ey := range yCoords {
		if math.Abs(ey-y) <= eps {
			return true
		}
	}

	return false
}

// repeatTableHeaders clones thead row ops onto every page that continues a
// multi-page table body, shifting body content down by the header height.
// Nested tables: each table repeats only its own thead.
func repeatTableHeaders(res *Result, contentH float64) {
	if res.root == nil || contentH <= 0 {
		return
	}

	for _, tblBox := range tableBoxes(res.root) {
		repeatTableHeaderOnPages(res, tblBox, contentH)
	}
}

// tableBoxes collects every multi-page table with a header band.
func tableBoxes(root *box) []*box {
	var tables []*box

	var walk func(b *box)
	walk = func(b *box) {
		if b.kind == displayTable && b.headerRows > 0 && b.headerRows < len(b.rows) {
			tables = append(tables, b)
		}

		for _, c := range b.children {
			walk(c)
		}
	}
	walk(root)

	return tables
}

// repeatTableHeaderOnPages clones one table's thead onto its continuation
// pages, shifting the body rows down by the header height.
func repeatTableHeaderOnPages(res *Result, tblBox *box, contentH float64) {
	nHdr := tblBox.headerRows
	if nHdr > len(tblBox.rows) {
		nHdr = len(tblBox.rows)
	}

	hdrFirst, hdrLast, hdrTop, hdrH := rowSpan(tblBox.rows[:nHdr], res)
	if hdrFirst < 0 || hdrH <= 0 {
		return
	}

	firstPage := int(tblBox.y / contentH)
	pages := headerContinuationPages(tblBox, firstPage, res, contentH)

	for page := range pages {
		if page <= firstPage {
			continue
		}

		pageTop := float64(page) * contentH
		shiftFrom, shiftTo, bodyTop := tableBodyRange(tblBox, page, res, contentH)

		if shiftFrom >= 0 && bodyTop >= 0 && bodyTop < pageTop+hdrH-0.5 {
			dy := pageTop + hdrH - bodyTop
			if dy > 0 {
				shiftFlowY(res, shiftFrom, shiftTo, bodyTop-layoutSlack, dy)
			}
		}

		cloneHeaderOps(res, hdrFirst, hdrLast, hdrTop, pageTop)
	}
}

// headerContinuationPages is the set of pages holding table body rows.
func headerContinuationPages(tblBox *box, _ int, res *Result, contentH float64) map[int]bool {
	pages := map[int]bool{}

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		top, _ := rowYBounds(row, res)
		if top < 0 {
			continue
		}

		pages[int(top/contentH)] = true
	}

	return pages
}

// tableBodyRange returns the op range and top Y of the body rows that start
// on the given page.
func tableBodyRange(tblBox *box, page int, res *Result, contentH float64) (int, int, float64) {
	shiftFrom, shiftTo := -1, -1
	bodyTop := -1.0

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		face, lst := rowOpRange(row)
		if face < 0 {
			continue
		}

		top, _ := rowYBounds(row, res)
		if int(top/contentH) < page {
			continue
		}

		if bodyTop < 0 || top < bodyTop {
			bodyTop = top
		}

		if shiftFrom < 0 || face < shiftFrom {
			shiftFrom = face
		}

		if lst > shiftTo {
			shiftTo = lst
		}
	}

	return shiftFrom, shiftTo, bodyTop
}

// cloneHeaderOps copies the thead op range to the page top.
func cloneHeaderOps(res *Result, hdrFirst, hdrLast int, hdrTop, pageTop float64) {
	for k := hdrFirst; k <= hdrLast && k < len(res.Ops); k++ {
		op := res.Ops[k]
		op.Y = pageTop + (op.Y - hdrTop)
		res.Ops = append(res.Ops, op)
	}
}

func rowOpRange(row []*box) (int, int) {
	first, last := -1, -1

	for _, cell := range row {
		if cell.opStart <= cell.opEnd {
			if first < 0 || cell.opStart < first {
				first = cell.opStart
			}

			if cell.opEnd > last {
				last = cell.opEnd
			}
		}
	}

	return first, last
}

func rowYBounds(row []*box, res *Result) (float64, float64) {
	first, last := rowOpRange(row)
	if first < 0 || first >= len(res.Ops) {
		return -1, -1
	}

	top, bottom := opBottomEdge(res.Ops[first])

	for k := first + 1; k <= last && k < len(res.Ops); k++ {
		posY, bot := opBottomEdge(res.Ops[k])
		if posY < top {
			top = posY
		}

		if bot > bottom {
			bottom = bot
		}
	}

	return rowCellBounds(row, top, bottom)
}

// opBottomEdge returns an op's top Y and approximate bottom Y.
func opBottomEdge(paintOp Op) (float64, float64) {
	posY := paintOp.Y

	height := paintOp.H
	if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
		height = paintOp.Size * defaultLineHeightRatio
	}

	return posY, posY + height
}

// rowCellBounds widens the row band with the cells' own geometry.
func rowCellBounds(row []*box, top, bottom float64) (float64, float64) {
	for _, cell := range row {
		if cell.height > 0 && cell.y+cell.height > bottom {
			bottom = cell.y + cell.height
		}

		if cell.y < top && cell.y > 0 {
			top = cell.y
		}
	}

	return top, bottom
}

func rowSpan(rows [][]*box, res *Result) (int, int, float64, float64) {
	first, last, top, bottom, ok := rowsBandGeometry(rows, res)
	if !ok {
		return -1, -1, 0, 0
	}

	height := bottom - top
	if height >= 1 {
		return first, last, top, height
	}

	return first, last, top, sumRowHeights(rows)
}

// rowsBandGeometry returns the op range and Y band of the rows.
func rowsBandGeometry(rows [][]*box, res *Result) (int, int, float64, float64, bool) {
	first, last := -1, -1
	top, bottom := 0.0, 0.0
	set := false

	for _, row := range rows {
		rowFirst, rowLast, rowTop, rowBottom, rowOK := rowBandExtent(row, res)
		if !rowOK {
			continue
		}

		first, last = extendBandRange(first, last, rowFirst, rowLast)
		top, bottom, set = extendBandBounds(top, bottom, rowTop, rowBottom, set)
	}

	return first, last, top, bottom, first >= 0 && set
}

// rowBandExtent returns the op range and Y band of one row.
func rowBandExtent(row []*box, res *Result) (int, int, float64, float64, bool) {
	face, lst := rowOpRange(row)
	if face < 0 {
		return -1, -1, 0, 0, false
	}

	right, rowB := rowYBounds(row, res)
	if right < 0 {
		return -1, -1, 0, 0, false
	}

	return face, lst, right, rowB, true
}

// extendBandRange widens the band's op range with one row's range.
func extendBandRange(first, last, rowFirst, rowLast int) (int, int) {
	if first < 0 || rowFirst < first {
		first = rowFirst
	}

	if rowLast > last {
		last = rowLast
	}

	return first, last
}

// extendBandBounds widens the band's Y bounds with one row's band.
func extendBandBounds(top, bottom, rowTop, rowBottom float64, set bool) (float64, float64, bool) {
	if !set || rowTop < top {
		top = rowTop
	}

	if !set || rowBottom > bottom {
		bottom = rowBottom
	}

	return top, bottom, true
}

// sumRowHeights falls back to the sum of the cells' heights when the op band
// is too thin to measure (rows without op Y spread).
func sumRowHeights(rows [][]*box) float64 {
	height := 0.0

	for _, row := range rows {
		rowH := 0.0
		for _, cell := range row {
			if cell.height > rowH {
				rowH = cell.height
			}
		}

		height += rowH
	}

	return height
}

// populateLocations records the canvas rect and page of every element box in
// document order. A box's page is the page of its first op; boxes without ops
// (or before an op→page map exists) fall back to the page of their y position.
func populateLocations(res *Result, contentH float64, opPage []int) {
	res.Locations = nil
	if res.root == nil {
		return
	}

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

func drawFill(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	ps := StyleOf(op)
	c.SetFillColor(ps.FillR, ps.FillG, ps.FillB)
	c.Rect(x, y, op.W, op.H)
	c.Fill()
}

func drawStroke(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	c.SetStrokeColor(op.R, op.G, op.B)
	c.SetLineWidth(1)
	c.Rect(x, y, op.W, op.H)
	c.Stroke()
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
		chld.TextMatrix(0, 1, -1, 0, posX, posY)
	} else {
		chld.TextAt(posX, posY)
	}
	// Fake bold only for Latin when CSS wants bold but the face is not bold.
	// Stroking CJK/Type0 outlines creates horizontal streak artifacts.
	fakeBold := FakeBoldFor(paintOp)
	if fakeBold {
		chld.SetLineWidth(paintOp.Size * outlineStrokeRatio)
		chld.TextRenderMode(two) // fill + stroke
	}

	chld.TextShow(paintOp.Text)

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
func drawLinkXform(page *pdf.Page, paintOp *Op, pageIdx int, contentH float64, opts PaintOptions) {
	if len(paintOp.URI) > 0 && paintOp.URI[0] == '#' {
		return
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

	page.AddLinkURI([4]float64{llx, lly, urx, ury}, paintOp.URI)
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
