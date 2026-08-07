package layout

import (
	"context"
	"math"
	"sort"
	"strconv"

	"gowkhtmltopdf/internal/pdf"
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
func Paint(doc *pdf.Document, res *Result, opts PaintOptions) error {
	return PaintContext(context.Background(), doc, res, opts)
}

// PaintContext is the cancellation-aware form of Paint. The legacy Paint
// entrypoint remains a background-context adapter for package callers that do
// not have a request context.
func PaintContext(ctx context.Context, doc *pdf.Document, res *Result, opts PaintOptions) error {
	if doc == nil || res == nil {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return err
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

	// Fixed ops are stamped on every page at viewport-relative coords.
	var fixedIdx []int

	for i := range res.Ops {
		if res.Ops[i].Fixed {
			fixedIdx = append(fixedIdx, i)
		}
	}

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
	opPage = make([]int, len(res.Ops))
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

	populateLocations(res, contentH, opPage)

	var paintErr error

	for pageIdx, idxs := range res.Pages {
		if err := ctx.Err(); err != nil {
			return err
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

			switch paintOp.Kind {
			case OpFillRect:
				drawFill(child, paintOp, pageN, contentH, opts, page.Height())
			case OpStrokeRect:
				drawStroke(child, paintOp, pageN, contentH, opts, page.Height())
			case OpLine:
				drawLine(child, paintOp, pageN, contentH, opts, page.Height())
			case OpText, OpBullet:
				drawText(child, paintOp, pageN, contentH, opts, page.Height(), resName(paintOp.Font))
			case OpImage:
				name := "I" + strconv.Itoa(nextImg)
				nextImg++

				if err := drawImage(page, child, paintOp, pageN, contentH, opts, name); err != nil && paintErr == nil {
					paintErr = err
				}
			}

			if needGS {
				child.Restore()
			}
		}

		sortPaintIndices(res.Ops, idxs)

		for _, idx := range idxs {
			if err := ctx.Err(); err != nil {
				return err
			}

			paintOp(&res.Ops[idx], pageIdx)
		}
		// Fixed layer: page-local coords (pageIdx 0 math on every page).
		sortPaintIndices(res.Ops, fixedIdx)

		for _, idx := range fixedIdx {
			if err := ctx.Err(); err != nil {
				return err
			}

			paintOp(&res.Ops[idx], 0)
		}
	}

	return paintErr
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
	default:
		return 1
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
// embedded fonts, fake-bold policy). Pagination, z-sorting and fixed stamps
// are skipped. Link ops are left to the caller (annotations need document
// context). Returns the first image-embed error, if any.
func PaintBand(p *pdf.Page, c *pdf.Content, ops []Op, opts BandOptions) error {
	return PaintBandContext(context.Background(), p, c, ops, opts)
}

// PaintBandContext is the cancellation-aware form of PaintBand used for
// HTML headers and footers.
func PaintBandContext(ctx context.Context, p *pdf.Page, c *pdf.Content, ops []Op, opts BandOptions) error {
	if p == nil || c == nil {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return err
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
		c.UseEmbeddedFont(n, face)

		return n
	}
	nextImg := 0
	pos := opts.Margins
	contentH := opts.ContentH

	pageH := opts.PageH
	if pageH <= 0 {
		pageH = p.Height()
	}
	// Band mode without full page geometry: map canvas (y down) to PDF using
	// OriginY as the top edge and OriginX as left origin.
	useSimple := contentH <= 0

	var firstErr error

	for i := range ops {
		if err := ctx.Err(); err != nil {
			return err
		}

		paintOp := &ops[i]
		if paintOp.Kind == OpLinkURI || paintOp.Kind == opKindNoop {
			continue
		}

		needGS := paintOp.XformSet || (paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1)
		if needGS {
			c.Save()
		}

		if paintOp.XformSet && !useSimple {
			a, b, cc, d, e, f := pdfCTMFromCSS(paintOp.Xform, 0, contentH, pos, pageH)
			c.Transform(a, b, cc, d, e, f)
		}

		if paintOp.PaintOpacity > 0 && paintOp.PaintOpacity < 1 {
			c.SetOpacity(paintOp.PaintOpacity)
		}

		if useSimple {
			if err := paintOpBandSimple(c, p, paintOp, opts, resName(paintOp.Font), &nextImg); err != nil && firstErr == nil {
				firstErr = err
			}
		} else {
			switch paintOp.Kind {
			case OpFillRect:
				drawFill(c, paintOp, 0, contentH, pos, pageH)
			case OpStrokeRect:
				drawStroke(c, paintOp, 0, contentH, pos, pageH)
			case OpLine:
				drawLine(c, paintOp, 0, contentH, pos, pageH)
			case OpText, OpBullet:
				drawText(c, paintOp, 0, contentH, pos, pageH, resName(paintOp.Font))
			case OpImage:
				name := "I" + strconv.Itoa(nextImg)
				nextImg++

				if err := drawImage(p, c, paintOp, 0, contentH, pos, name); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}

		if needGS {
			c.Restore()
		}
	}

	return firstErr
}

func paintOpBandSimple(c *pdf.Content, _ *pdf.Page, op *Op, opts BandOptions, fontName string, nextImg *int) error {
	posX := opts.OriginX + op.X

	switch op.Kind {
	case OpFillRect:
		ps := StyleOf(op)
		y := opts.OriginY - (op.Y + op.H)

		c.SetFillColor(ps.FillR, ps.FillG, ps.FillB)
		c.Rect(posX, y, op.W, op.H)
		c.Fill()
	case OpStrokeRect:
		y := opts.OriginY - (op.Y + op.H)
		c.SetStrokeColor(op.R, op.G, op.B)
		c.SetLineWidth(1)
		c.Rect(posX, y, op.W, op.H)
		c.Stroke()
	case OpLine:
		yEnd := opts.OriginY - op.Y
		yTwo := opts.OriginY - (op.Y + op.H)

		width := op.Width
		if width <= 0 {
			width = 1
		}

		c.SetStrokeColor(op.R, op.G, op.B)
		c.SetLineWidth(width)
		c.MoveTo(posX, yEnd)
		c.LineTo(opts.OriginX+op.X+op.W, yTwo)
		c.Stroke()
	case OpText, OpBullet:
		posY := opts.OriginY - op.Y
		c.SetFillColor(op.R, op.G, op.B)

		if fontName == "" {
			fontName = "F0"
		}

		c.SetFont(fontName, op.Size)
		c.BeginText()
		c.TextAt(posX, posY)

		if FakeBoldFor(op) {
			c.SetLineWidth(op.Size * outlineStrokeRatio)
			c.TextRenderMode(two)
		}

		c.TextShow(op.Text)

		if FakeBoldFor(op) {
			c.TextRenderMode(0)
		}

		c.EndText()
	case OpImage:
		name := "I" + strconv.Itoa(*nextImg)
		*nextImg++

		y := opts.OriginY - (op.Y + op.H)
		if op.IsJPEG {
			return c.AddJPEGImage(name, posX, y, op.W, op.H, op.Image)
		}

		return c.AddPNGImage(name, posX, y, op.W, op.H, op.Image)
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

	const eps = 2.0

	type vseg struct{ x, y0, y1, w, r, g, b float64 }
	// Collect non-fixed vertical and horizontal line ops once.
	var verts []vseg

	type hseg struct{ x0, x1, y, w, r, g, b float64 }

	var horiz []hseg

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Fixed || paintOp.Kind != OpLine {
			continue
		}

		if paintOp.H > 2 && (paintOp.W < 1 || paintOp.W < paintOp.H*0.05) {
			verts = append(verts, vseg{paintOp.X, paintOp.Y, paintOp.Y + paintOp.H, paintOp.Width, paintOp.R, paintOp.G, paintOp.B})

			continue
		}

		if paintOp.W > 2 && paintOp.H < 1 {
			horiz = append(horiz, hseg{paintOp.X, paintOp.X + paintOp.W, paintOp.Y, paintOp.Width, paintOp.R, paintOp.G, paintOp.B})
		}
	}
	// Group verticals that share a start Y (row top) or end Y (row bottom).
	roundY := func(y float64) int { return int(math.Round(y * two)) } // 0.5pt bins
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

	type cluster struct {
		y           float64
		minX, maxX  float64
		bw, r, g, b float64
		n           int
	}

	clusterAt := func(byStart bool) map[int]*cluster {
		out := map[int]*cluster{}

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
					child = &cluster{y: keyY, minX: val.x, maxX: val.x, bw: val.w, r: val.r, g: val.g, b: val.b, n: 1}
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
	hCoverage := func(y, minX, maxX float64) (full bool, covMin, covMax float64, has bool) {
		key := roundY(y)
		for k := key - int(eps*two) - 1; k <= key+int(eps*two)+1; k++ {
			for _, height := range horizByY[k] {
				if math.Abs(height.y-y) > eps {
					continue
				}
				// Only count segments that overlap the vertical band.
				if height.x1 < minX-eps || height.x0 > maxX+eps {
					continue
				}

				if !has {
					covMin, covMax, has = height.x0, height.x1, true
				} else {
					if height.x0 < covMin {
						covMin = height.x0
					}

					if height.x1 > covMax {
						covMax = height.x1
					}
				}
			}
		}

		if !has {
			return false, 0, 0, false
		}

		full = covMin <= minX+eps && covMax >= maxX-eps

		return full, covMin, covMax, true
	}
	seal := func(gVal, minX, maxX, borderW, red, green, blue float64) {
		if maxX-minX < 20 || borderW < 0 {
			return
		}

		if borderW < minBorderWidthPt {
			borderW = 0.5
		}
		// Avoid exact duplicates.
		for _, h := range horiz {
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
		horiz = append(horiz, sealed)
		horizByY[roundY(gVal)] = append(horizByY[roundY(gVal)], sealed)
	}

	// (1) Classic page-top stubs.
	for p := 1; p <= maxPage; p++ {
		pageTop := float64(p) * contentH

		var bwVal, maxX, borderW, redN, green, blueN float64

		node := 0

		key := roundY(pageTop)
		for k := key - int(eps*two) - 1; k <= key+int(eps*two)+1; k++ {
			for _, val := range vertStarts[k] {
				if val.y0 < pageTop-eps || val.y0 > pageTop+eps {
					continue
				}

				if node == 0 {
					bwVal, maxX, borderW, redN, green, blueN = val.x, val.x, val.w, val.r, val.g, val.b
				} else {
					if val.x < bwVal {
						bwVal = val.x
					}

					if val.x > maxX {
						maxX = val.x
					}
				}

				node++
			}
		}

		if node < two {
			continue
		}

		if full, _, _, _ := hCoverage(pageTop, bwVal, maxX); full {
			continue
		}

		seal(pageTop, bwVal, maxX, borderW, redN, green, blueN)
	}

	// (2) Seal incomplete tops of multi-column vertical clusters that start a
	// continuation-page body band (under repeated thead or at page top).
	// Mid-table rowspan holes keep skipped tops so continuous year cells stay
	// unsplit; only the page-fragment open edge is closed.
	for _, child := range clusterAt(true) {
		if child.n < 3 || child.maxX-child.minX < 20 {
			continue
		}

		full, _, _, _ := hCoverage(child.y, child.minX, child.maxX)
		if full {
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
	// Row bottoms: seal when verticals end near a page bottom and no full
	// horizontal closes the strip (next row's top moved to the following page).
	for _, child := range clusterAt(false) {
		if child.n < 3 || child.maxX-child.minX < 20 {
			continue
		}

		page := int((child.y - layoutSlack) / contentH)
		pageBot := float64(page+1) * contentH
		// Only near the page boundary (row ended as last on page).
		if child.y < pageBot-40 || child.y > pageBot+eps {
			continue
		}

		if full, _, _, _ := hCoverage(child.y, child.minX, child.maxX); full {
			continue
		}

		if page >= 0 {
			seal(child.y, child.minX, child.maxX, child.bw, child.r, child.g, child.b)
		}
	}
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
	for iter := 0; iter < 10 && beforeAlways(res, contentH); iter++ {
	}

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
				if deltaY := boundary - paintOp.Y; deltaY > layoutEpsilon {
					// Snap text (+ following flow). Same-row fills sit above the
					// baseline; include them in dy via minY so their tops clear
					// onto this page with the text (fixture-31 Row 28 white bg).
					// Keep chrome matching tight (one row) so table reports do
					// not inflate dy. Never clamp fill tops to `boundary` alone
					// — that collapses them onto the text Y and leaves section
					// gray showing through the ascent/padding band.
					oldY := paintOp.Y
					minY := oldY

					var chrome []int

					for jdx := range res.Ops {
						obj := &res.Ops[jdx]
						if obj.Fixed || jdx == idx {
							continue
						}

						if obj.Kind != OpFillRect && obj.Kind != OpStrokeRect {
							continue
						}

						if obj.H <= 0.5 || obj.H > 40 {
							continue
						}

						if obj.Y > oldY+0.5 || obj.Y+obj.H < oldY-0.5 {
							continue
						}

						if oldY-obj.Y > obj.H+2 {
							continue
						}

						chrome = append(chrome, jdx)

						if obj.Y < minY {
							minY = obj.Y
						}
					}
					// Leave room for ascenders above the baseline so snapped
					// lines do not paint into the top margin (page-4/5 bleed).
					lead := 0.0
					if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
						lead = paintOp.Size * pxToPtFactor
						if lead < maxGlueEm {
							lead = 8
						}
					}

					deltaY = boundary + lead - minY
					shiftFlowY(res, idx, idx, oldY-layoutSlack, deltaY)

					for _, j := range chrome {
						o := &res.Ops[j]
						if o.Y < oldY-0.01 {
							o.Y += deltaY
						}
					}
				} else {
					paintOp.Y = boundary
				}
			}
		}
	}

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

		guard := 0
		for paintOp.H > 1e-9 {
			guard++
			if guard > paginationGuardMax {
				// Defensive: never hang the paint pipeline.
				out = append(out, paintOp)

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
				out = append(out, paintOp)

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
			out = append(out, frag)
			paintOp.Y = boundary
			paintOp.H -= firstH
		}

		spans[idx] = opSpan{start: start, end: len(out) - 1}
	}

	res.Ops = out
	remapBoxOpRanges(res.root, spans)
}

// remapBoxOpRanges updates the layout-owned operation ranges after a display
// list rewrite. In particular, a source rectangle can become two or more
// page fragments; mapping the box end to the final fragment keeps pagination,
// sticky/fixed stamping, and ElementLocation ownership aligned.
func remapBoxOpRanges(boxNode *box, spans []opSpan) {
	if boxNode == nil {
		return
	}

	if boxNode.opStart >= 0 && boxNode.opEnd >= boxNode.opStart && boxNode.opStart < len(spans) && boxNode.opEnd < len(spans) {
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

	stickyTargets := stickySectionChromeTargets(res.root)

	for page := 0; page <= maxPage; page++ {
		pageTop := float64(page) * contentH
		pageBot := pageTop + contentH
		lastInkBot := pageTop
		hasInk := false

		for _, i := range pageOps[page] {
			paintOp := &res.Ops[i]
			if paintOp.Y < pageTop-1e-9 || paintOp.Y >= pageBot-1e-9 {
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
			default:
				continue
			}

			hasInk = true

			if bot > lastInkBot {
				lastInkBot = bot
			}
		}

		if !hasInk {
			continue
		}

		stripped := false

		for _, i := range pageOps[page] {
			paintOp := &res.Ops[i]
			if paintOp.StickyID != 0 {
				continue
			}

			if paintOp.Y < pageTop-1e-9 || paintOp.Y >= pageBot-1e-9 {
				continue
			}

			switch paintOp.Kind {
			case OpFillRect, OpStrokeRect:
				// Row-sized shells whose center sits below the last ink are
				// empty trailing row backgrounds (not the cell that holds the
				// last text, whose center is at/above the baseline band).
				if paintOp.H <= 0.5 || paintOp.H > 40 {
					continue
				}

				if paintOp.Y+paintOp.H/2 > lastInkBot+0.5 {
					paintOp.H = 0
					stripped = true
				}
			case OpLine:
				if paintOp.H >= 1 {
					continue
				}
				// Horizontal rule below the last ink (empty row separator).
				if paintOp.Y > lastInkBot+0.5 {
					paintOp.Width = 0
					stripped = true
				}
			}
		}

		if stripped {
			// Tighten the last row fill so padding under the final baseline does
			// not read as another empty row (fixture-31 Row 27 cell).
			const underPad = 8.0

			for _, i := range pageOps[page] {
				paintOp := &res.Ops[i]
				if paintOp.StickyID != 0 {
					continue
				}

				if paintOp.Y < pageTop-1e-9 || paintOp.Y >= pageBot-1e-9 {
					continue
				}

				if (paintOp.Kind == OpFillRect || paintOp.Kind == OpStrokeRect) &&
					paintOp.H > 0.5 && paintOp.H <= 40 &&
					paintOp.Y < lastInkBot && paintOp.Y+paintOp.H > lastInkBot+underPad+2 {
					paintOp.H = lastInkBot + underPad - paintOp.Y
					if paintOp.H < 1 {
						paintOp.H = 1
					}
				}

				if paintOp.Kind == OpLine && paintOp.H < 1 && paintOp.Width > 0 &&
					paintOp.Y > lastInkBot+underPad+1 && paintOp.Y < lastInkBot+40 {
					paintOp.Y = lastInkBot + underPad
				}
			}
		}
		// Pull section washes / borders up to the last row chrome / ink so grey
		// does not pad an empty band to the page bottom (fixture-31 page 1).
		// Only section-colored chrome is clipped — arbitrary tall fills
		// (TestBoundaryFillSplit) are left to the normal page-split remnant.
		contentBot := lastInkBot

		for _, i := range pageOps[page] {
			paintOp := &res.Ops[i]
			if paintOp.StickyID != 0 {
				continue
			}

			if paintOp.Y < pageTop-1e-9 || paintOp.Y >= pageBot-1e-9 {
				continue
			}

			if (paintOp.Kind == OpFillRect || paintOp.Kind == OpStrokeRect) && paintOp.H > 0.5 && paintOp.H <= 40 {
				if bot := paintOp.Y + paintOp.H; bot > contentBot {
					contentBot = bot
				}
			}

			if paintOp.Kind == OpLine && paintOp.H < 1 && paintOp.Width > 0 && paintOp.Y > contentBot {
				contentBot = paintOp.Y
			}
		}

		if pageBot-contentBot < maxGlueEm {
			continue
		}

		for _, i := range pageOps[page] {
			paintOp := &res.Ops[i]
			if paintOp.StickyID != 0 {
				continue
			}

			if paintOp.Y < pageTop-1e-9 || paintOp.Y >= pageBot-1e-9 {
				continue
			}

			switch paintOp.Kind {
			case OpFillRect:
				// Only continuation fragments that begin at the page top are
				// eligible for this trailing-band cleanup. A normal block fill
				// (for example a <pre> with bottom padding) must retain its full
				// box height through its bottom border.
				if paintOp.Y <= pageTop+1 && paintOp.H > 40 && isSectionWashRGB(paintOp.R, paintOp.G, paintOp.B) &&
					paintOp.Y+paintOp.H > contentBot+1 && paintOp.Y < contentBot {
					paintOp.H = contentBot - paintOp.Y
				}
			case OpLine:
				if paintOp.Y <= pageTop+1 && paintOp.H > 40 && nearSectionBorderRGB(paintOp.R, paintOp.G, paintOp.B) &&
					paintOp.Y+paintOp.H > contentBot+1 && paintOp.Y < contentBot {
					paintOp.H = contentBot - paintOp.Y
				} else if paintOp.H < 1 && paintOp.Width > 0 && nearSectionBorderRGB(paintOp.R, paintOp.G, paintOp.B) &&
					paintOp.Y > contentBot+1 && paintOp.Y > pageBot-30 {
					paintOp.Y = contentBot
				}
			}
		}
		// A sticky containing block may begin on this page and continue onto
		// the next one. Its page fragment must still end at the last real row;
		// otherwise the unsplit section wash fills the unused page tail.
		for _, target := range stickyTargets {
			for _, i := range pageOps[page] {
				paintOp := &res.Ops[i]
				if paintOp.StickyID != 0 || paintOp.Y < pageTop-1e-9 || paintOp.Y >= pageBot-1e-9 || paintOp.H <= 40 ||
					paintOp.Y+paintOp.H <= contentBot+1 {
					continue
				}

				switch paintOp.Kind {
				case OpFillRect:
					if target.hasBackground && sameRectFrame(paintOp, target) && sameRGB(paintOp, target.background) {
						paintOp.H = contentBot - paintOp.Y
					}
				case OpLine:
					if target.sideMatches(paintOp) {
						paintOp.H = contentBot - paintOp.Y
					}
				}
			}

			if target.hasBottom && stickySectionContinuesAfterPage(res, target, pageBot) &&
				!hasStickySectionBottomBorder(res, target, contentBot) {
				res.Ops = append(res.Ops, Op{ //nolint:exhaustruct // intentional zero fields
					Kind: OpLine, X: target.x, Y: contentBot, W: target.w,
					Width: target.borderBottomWidth,
					R:     target.borderBottom[0], G: target.borderBottom[1], B: target.borderBottom[2],
				})
			}
		}
	}

	closePageLeadingSectionChromeWithTargets(res, contentH, stickyTargets)
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

	maxPage := 0

	for _, op := range res.Ops {
		if op.Fixed {
			continue
		}

		if page := int(op.Y / contentH); page > maxPage {
			maxPage = page
		}
	}

	for page := 1; page <= maxPage; page++ {
		pageTop := float64(page) * contentH
		pageBottom := pageTop + contentH

		for _, target := range targets {
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

			if closeY < 0 {
				continue
			}

			for i := range res.Ops {
				paintOp := &res.Ops[i]
				if paintOp.Fixed || paintOp.Y < pageTop-1 || paintOp.Y > pageTop+1 || paintOp.Y >= pageBottom {
					continue
				}

				switch paintOp.Kind {
				case OpFillRect:
					if target.hasBackground && paintOp.H > 40 && sameRectFrame(paintOp, target) && sameRGB(paintOp, target.background) && paintOp.Y+paintOp.H < closeY {
						paintOp.H = closeY - paintOp.Y
					}
				case OpLine:
					if paintOp.H > 40 && target.sideMatches(paintOp) && paintOp.Y+paintOp.H < closeY {
						paintOp.H = closeY - paintOp.Y
					}
				}
			}
		}
	}
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
			sty := parent.style
			target := stickySectionChromeTarget{ //nolint:exhaustruct // intentional zero fields
				x:                 parent.x,
				y:                 parent.y,
				w:                 parent.w,
				borderLeft:        sty.BorderLeft.Color,
				borderRight:       sty.BorderRight.Color,
				borderBottom:      sty.BorderBottom.Color,
				borderBottomWidth: sty.BorderBottom.Width,
				hasBottom:         sty.BorderBottom.Width > 0 && sty.BorderBottom.Style != "none",
				hasLeft:           sty.BorderLeft.Width > 0 && sty.BorderLeft.Style != "none",
				hasRight:          sty.BorderRight.Width > 0 && sty.BorderRight.Style != "none",
			}

			if sty.BGColor[3] > 0 {
				target.background = [3]float64{sty.BGColor[0], sty.BGColor[1], sty.BGColor[2]}
				target.hasBackground = true
			}

			duplicate := false

			for _, prior := range targets {
				if math.Abs(prior.x-target.x) < 0.01 && math.Abs(prior.y-target.y) < 0.01 && math.Abs(prior.w-target.w) < 0.01 {
					duplicate = true

					break
				}
			}

			if !duplicate {
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
// strictly below fromY - down by dy canvas points. Ops of earlier boxes that
// touch fromY exactly (collapsed margins) are left alone so the page-break
// fixpoint converges instead of dragging boundary ops along each iteration.
// Box.y is kept in sync for boxes whose top moved.
func shiftFlowY(res *Result, from, to int, fromY, dy float64) {
	if res == nil || len(res.Ops) == 0 || dy == 0 {
		return
	}

	ensureFlowIndex(res, flowIndexPageSize(res))

	for i := from; i <= to; i++ {
		if i < 0 || i >= len(res.Ops) || res.Ops[i].Fixed {
			continue
		}

		shiftIndexedOp(res, i, dy)
	}

	startPage := int(fromY / res.flowPageSize)
	if startPage < 0 {
		startPage = 0
	}

	if dy > 0 {
		// Positive flow shifts move an operation to the same or a later page.
		// Process buckets from the end so an operation moved to another bucket
		// is never visited twice. Removing the current item in shiftIndexedOp
		// swaps the bucket's last item into its place; keep the cursor in place
		// when that happens.
		for p := len(res.flowPages) - 1; p >= startPage; p-- {
			for jdx := 0; jdx < len(res.flowPages[p]); {
				idx := res.flowPages[p][jdx]
				if (idx >= from && idx <= to) || res.Ops[idx].Y <= fromY {
					jdx++

					continue
				}

				oldPage := res.flowPageOf[idx]
				shiftIndexedOp(res, idx, dy)

				if res.flowPageOf[idx] == oldPage {
					jdx++
				}
			}
		}
	} else {
		// Negative shifts move operations to the same or an earlier page.
		// Process buckets in ascending order so an operation moved backward is
		// not visited twice, while keeping the index in sync for later passes.
		for p := startPage; p < len(res.flowPages); p++ {
			for jdx := 0; jdx < len(res.flowPages[p]); {
				idx := res.flowPages[p][jdx]
				if (idx >= from && idx <= to) || res.Ops[idx].Y <= fromY {
					jdx++

					continue
				}

				oldPage := res.flowPageOf[idx]
				shiftIndexedOp(res, idx, dy)

				if res.flowPageOf[idx] == oldPage {
					jdx++
				}
			}
		}
	}

	if res.root == nil {
		return
	}

	if len(res.flowBoxes) == 0 {
		boxes := res.boxes
		if len(boxes) == 0 {
			boxes = make([]*box, 0)
			flattenBoxes(res.root, &boxes)
			res.boxes = boxes
		}

		ensureFlowBoxIndex(res, boxes)
	}

	for p := len(res.flowBoxes) - 1; p >= startPage; p-- {
		for jdx := 0; jdx < len(res.flowBoxes[p]); {
			boxIndex := res.flowBoxes[p][jdx]

			b := res.boxes[boxIndex]
			if p == startPage && !(b.y > fromY ||
				(b.y == fromY && b.opStart >= from && b.opEnd <= to)) {
				jdx++

				continue
			}

			oldPage := res.flowBoxPage[boxIndex]
			shiftIndexedBox(res, boxIndex, dy)

			if res.flowBoxPage[boxIndex] == oldPage {
				jdx++
			}
		}
	}
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
	maxPage := 0

	for i := range res.Ops {
		if res.Ops[i].Fixed {
			continue
		}

		page := int(res.Ops[i].Y / pageSize)
		if page < 0 {
			page = 0
		}

		if page > maxPage {
			maxPage = page
		}
	}

	res.flowPages = make([][]int, maxPage+1)
	res.flowPageOf = make([]int, len(res.Ops))
	res.flowPos = make([]int, len(res.Ops))

	for i := range res.flowPageOf {
		res.flowPageOf[i] = -1
		res.flowPos[i] = -1
	}

	for idx := range res.Ops {
		if res.Ops[idx].Fixed {
			continue
		}

		page := int(res.Ops[idx].Y / pageSize)
		if page < 0 {
			page = 0
		}

		res.flowPageOf[idx] = page
		res.flowPos[idx] = len(res.flowPages[page])
		res.flowPages[page] = append(res.flowPages[page], idx)
	}

	boxes := res.boxes
	if len(boxes) == 0 && res.root != nil {
		boxes = make([]*box, 0)
		flattenBoxes(res.root, &boxes)
		res.boxes = boxes
	}

	ensureFlowBoxIndex(res, boxes)
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

func shiftIndexedOp(res *Result, index int, dy float64) {
	if index < 0 || index >= len(res.Ops) || res.Ops[index].Fixed {
		return
	}

	oldPage := res.flowPageOf[index]
	res.Ops[index].Y += dy

	newPage := int(res.Ops[index].Y / res.flowPageSize)
	if newPage < 0 {
		newPage = 0
	}

	if oldPage == newPage {
		return
	}

	if oldPage >= 0 && oldPage < len(res.flowPages) {
		bucket := res.flowPages[oldPage]
		pos := res.flowPos[index]

		if pos >= 0 && pos < len(bucket) {
			last := bucket[len(bucket)-1]
			bucket[pos] = last
			res.flowPos[last] = pos
			res.flowPages[oldPage] = bucket[:len(bucket)-1]
		}
	}

	for len(res.flowPages) <= newPage {
		res.flowPages = append(res.flowPages, nil)
	}

	res.flowPageOf[index] = newPage
	res.flowPos[index] = len(res.flowPages[newPage])
	res.flowPages[newPage] = append(res.flowPages[newPage], index)
}

func shiftIndexedBox(res *Result, index int, deltaY float64) {
	if index < 0 || index >= len(res.boxes) {
		return
	}

	b := res.boxes[index]
	oldPage := res.flowBoxPage[index]
	b.y += deltaY

	newPage := int(b.y / res.flowPageSize)
	if newPage < 0 {
		newPage = 0
	}

	if oldPage == newPage {
		return
	}

	if oldPage >= 0 && oldPage < len(res.flowBoxes) {
		bucket := res.flowBoxes[oldPage]
		pos := res.flowBoxPos[index]

		if pos >= 0 && pos < len(bucket) {
			last := bucket[len(bucket)-1]
			bucket[pos] = last
			res.flowBoxPos[last] = pos
			res.flowBoxes[oldPage] = bucket[:len(bucket)-1]
		}
	}

	for len(res.flowBoxes) <= newPage {
		res.flowBoxes = append(res.flowBoxes, nil)
	}

	res.flowBoxPage[index] = newPage
	res.flowBoxPos[index] = len(res.flowBoxes[newPage])
	res.flowBoxes[newPage] = append(res.flowBoxes[newPage], index)
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

		if boxNode.style.PageBreakInside == "avoid" && boxNode.height > 0 {
			height := boxNode.height
			// Prefer ink extent when taller than the border box (rowspan /
			// deferred paint can make ops protrude past b.h — wiki awards).
			if boxNode.opStart <= boxNode.opEnd && boxNode.opStart >= 0 && boxNode.opEnd < len(res.Ops) {
				bot := boxNode.y

				for k := boxNode.opStart; k <= boxNode.opEnd; k++ {
					paintOp := res.Ops[k]
					outBox := paintOp.Y

					switch paintOp.Kind {
					case OpText, OpBullet:
						outBox += paintOp.Size * defaultLineHeightRatio
					default:
						if paintOp.H > 0 {
							outBox += paintOp.H
						}
					}

					if outBox > bot {
						bot = outBox
					}
				}

				if ink := bot - boxNode.y; ink > height {
					height = ink
				}
			}

			layoutOut := int(boxNode.y / contentH)
			hi := int((boxNode.y + height) / contentH)

			if hi > layoutOut && height <= contentH+0.01 {
				remaining := float64(layoutOut+1)*contentH - boxNode.y
				// Prefer splitting over large empty bands. Use border-box
				// height (b.h), not ink: after line-snap, ink can span a
				// page gap while the box is still a short list item —
				// classifying by ink disabled the short-box guard and
				// cascaded 100–150pt gaps (wiki references).
				if preferSplitOverBlank(remaining, boxNode.height, contentH) {
					return changed
				}
				// Large boxes: also prefer split when less than half the box
				// fits (rowspan tables / tall avoid blocks).
				if remaining < boxNode.height*0.5 && boxNode.height > contentH*0.35 {
					return changed
				}

				dy := float64(layoutOut+1)*contentH - boxNode.y
				if dy > layoutSlack {
					shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

					changed = true
				}
			}
		}

		return changed
	}

	return walk(res.root)
}

// beforeAlways walks pre-order and moves page-break-before: always boxes to a
// fresh page after everything preceding them.
func beforeAlways(res *Result, contentH float64) bool {
	if res == nil || res.root == nil || contentH <= 0 {
		return false
	}
	// The previous implementation recomputed the maximum Y of every prefix
	// once per forced-break box. Build the same mutation-safe metadata once per
	// pass instead; a successful shift returns immediately and the next pass
	// rebuilds the prefix from the updated operation coordinates.
	prefixMaxY := make([]float64, len(res.Ops)+1)
	for i, op := range res.Ops {
		prefixMaxY[i+1] = prefixMaxY[i]
		if op.Y > prefixMaxY[i+1] {
			prefixMaxY[i+1] = op.Y
		}
	}

	var walk func(b *box) bool
	walk = func(boxNode *box) bool {
		if boxNode.style.PageBreakBefore == "always" {
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
		}

		changed := false

		for _, c := range boxNode.children {
			if walk(c) {
				return true
			}
		}

		return changed
	}

	return walk(res.root)
}

// afterBreaks walks in document order and applies page-break-after:
// always|avoid to the box that follows each box in flow.
func afterBreaks(res *Result, contentH float64) bool {
	var boxes []*box

	var flatten func(b *box)

	flatten = func(b *box) {
		boxes = append(boxes, b)

		for _, c := range b.children {
			flatten(c)
		}
	}
	if res.root != nil {
		flatten(res.root)
	}

	changed := false

	for i, boxNode := range boxes {
		var next *box

		for j := i + 1; j < len(boxes); j++ {
			if boxes[j].opStart <= boxes[j].opEnd {
				next = boxes[j]

				break
			}
		}

		if next == nil || boxNode.opStart > boxNode.opEnd {
			continue
		}

		lastY := res.Ops[boxNode.opStart].Y
		for k := boxNode.opStart + 1; k <= boxNode.opEnd; k++ {
			if res.Ops[k].Y > lastY {
				lastY = res.Ops[k].Y
			}
		}

		lastPage := int(lastY / contentH)

		switch {
		case boxNode.style.PageBreakAfter == "always":
			dy := float64(lastPage+1)*contentH - next.y
			if dy > 0 {
				shiftFlowY(res, next.opStart, next.opEnd, next.y, dy)

				changed = true
			}
		case boxNode.style.PageBreakAfter == "avoid":
			// Keep this box with the following box across page boundaries.
			// Do NOT collapse natural flow spacing when they already share a
			// page (that pulled .keep boxes up onto paragraph baselines —
			// fixture-08 Forms index overlap).
			nextPage := int(next.y / contentH)
			if nextPage <= lastPage {
				break
			}
			// Place the heading on next's page without a full-page shiftFlowY
			// (that blanked pages after avoid-inside tables). Clear the
			// page-top band first: paginateOps may already have snapped a
			// prior paragraph's continuation to pageStart — that text is
			// NOT `next` (next is the following sibling), so we must push
			// every op in the landing band, not only next.
			pageStart := float64(nextPage) * contentH

			need := boxNode.height
			if need < 1 {
				need = 12
			}

			if boxNode.opStart <= boxNode.opEnd && boxNode.opStart >= 0 && boxNode.opEnd < len(res.Ops) {
				top, bot := boxNode.y, boxNode.y

				for k := boxNode.opStart; k <= boxNode.opEnd; k++ {
					paintOp := res.Ops[k]
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
					default:
						if paintOp.H > 0 {
							yEnd = paintOp.Y + paintOp.H
						}
					}

					if yStart < top {
						top = yStart
					}

					if yEnd > bot {
						bot = yEnd
					}
				}

				if ink := bot - top; ink > need {
					need = ink
				}

				if ink := bot - boxNode.y; ink > need {
					need = ink
				}
			}

			const gap = 10.0
			need += gap
			bandTop := pageStart + need
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

			dy := target - boxNode.y
			if dy > layoutSlackFine {
				shiftOpsOnly(res, boxNode.opStart, boxNode.opEnd, dy)
				boxNode.y += dy
				changed = true
			}
		}
	}

	return changed
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
			if len(row) == 0 {
				continue
			}

			first, last := -1, -1
			rowTop, rowBottom := 0.0, 0.0
			haveGeom := false

			for _, cell := range row {
				if cell.opStart <= cell.opEnd {
					if first < 0 {
						first = cell.opStart
					}

					if cell.opEnd > last {
						last = cell.opEnd
					}
				}
				// Use starting-row geometry, not full rowspan paint extent.
				// Rowspan cells emit bottom borders at y+h (full span); scanning
				// those ops made the first row look multi-page and cascaded
				// blank pages (wiki awards tables with rowspan=10+).
				top := cell.y

				h := cell.height
				if cell.rowSpan > 1 && cell.rowBoxH > 0 {
					h = cell.rowBoxH
				}

				bot := top + h
				if !haveGeom {
					rowTop, rowBottom, haveGeom = top, bot, true
				} else {
					if top < rowTop {
						rowTop = top
					}

					if bot > rowBottom {
						rowBottom = bot
					}
				}
			}

			if first < 0 || !haveGeom {
				continue
			}

			layoutOut, hi := int(rowTop/contentH), int(rowBottom/contentH)
			if hi > layoutOut {
				// Move only to the next page start. Using hi*contentH when the
				// row's measured bottom spans multiple pages (e.g. rowspan
				// paint height leaking into rowBoxH) skipped blank pages
				// between filmography and awards on long wiki tables.
				deltaY := float64(layoutOut+1)*contentH - rowTop
				if deltaY > layoutSlack {
					// fromY slightly above rowTop so border-collapse grid
					// lines that sit exactly on the row edge (and later
					// rows / chrome below) shift with the cells — otherwise
					// content moves and the grid stays behind (gapped /
					// misaligned music-video tables across page breaks).
					shiftFlowY(res, first, last, rowTop-layoutSlack, deltaY)

					changed = true
				}
			}
		}

		return changed
	}

	return walk(res.root)
}

// keepHeadingWithNext moves a heading to the next page when it would sit alone
// near the bottom (less than ~2 line-heights of room for following content).
func keepHeadingWithNext(res *Result, contentH float64) bool {
	if res.root == nil {
		return false
	}

	var boxes []*box

	var flatten func(b *box)
	flatten = func(b *box) {
		boxes = append(boxes, b)

		for _, c := range b.children {
			flatten(c)
		}
	}
	flatten(res.root)

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
		var next *box

		for j := idx + 1; j < len(boxes); j++ {
			if boxes[j].opStart <= boxes[j].opEnd && boxes[j].y >= boxNode.y {
				next = boxes[j]

				break
			}
		}

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

		if boxNode.kind != "block" || boxNode.height <= 0 || boxNode.opStart > boxNode.opEnd {
			return
		}
		// Nested block containers: children apply Rule 3; only heuristic on
		// short straddlers here.
		if hasNestedFlowChild(boxNode) {
			if orphansWidowsHeuristic(res, boxNode, contentH) {
				changed = true
			}

			return
		}

		lines := countBlockLineYs(res, boxNode)
		if len(lines) == 0 {
			if orphansWidowsHeuristic(res, boxNode, contentH) {
				changed = true
			}

			return
		}

		orphans := boxNode.style.Orphans
		if orphans < 1 {
			orphans = 2
		}

		widows := boxNode.style.Widows
		if widows < 1 {
			widows = 2
		}

		layoutOut := int(boxNode.y / contentH)
		hIdx := int((boxNode.y + boxNode.height) / contentH)

		if hIdx <= layoutOut {
			return
		}

		boundary := float64(layoutOut+1) * contentH
		before, after := 0, 0

		for _, y := range lines {
			if y < boundary-1e-6 {
				before++
			} else {
				after++
			}
		}
		// Rule 3 applies to Class B breaks *between line boxes*. If all text
		// lines sit on one side of the boundary (only padding/bg straddles),
		// do not keep-together tall boxes — fall back to the short heuristic.
		if before == 0 || after == 0 {
			if orphansWidowsHeuristic(res, boxNode, contentH) {
				changed = true
			}

			return
		}
		// Rule 3: legal Class B break only if both sides meet the minima.
		legal := before >= orphans && after >= widows
		if legal {
			return
		}
		// Keep the block together when it fits one page; else progress escape.
		// Same blank-band guard as avoidInside: do not open a large empty
		// region on the current page for a short keep-together.
		if boxNode.height <= contentH+0.01 {
			remaining := float64(layoutOut+1)*contentH - boxNode.y
			if preferSplitOverBlank(remaining, boxNode.height, contentH) {
				return
			}

			dy := float64(hIdx)*contentH - boxNode.y
			if dy > layoutEpsilon {
				shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

				changed = true
			}
		}
	}
	walk(res.root)

	return changed
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

func hasNestedFlowChild(b *box) bool {
	for _, c := range b.children {
		if c.kind == "block" || c.kind == "table" {
			return true
		}
	}

	return false
}

// countBlockLineYs returns distinct text/bullet baseline Y positions in b's
// op range (approximate line boxes for an IFC leaf block).
func countBlockLineYs(res *Result, b *box) []float64 {
	if res == nil || b.opStart > b.opEnd || b.opStart < 0 {
		return nil
	}

	const eps = 0.5

	yCoords := make([]float64, 0, maxGlueEm)

	end := b.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	for i := b.opStart; i <= end; i++ {
		op := &res.Ops[i]
		if op.Kind != OpText && op.Kind != OpBullet {
			continue
		}

		posY := op.Y
		found := false

		for _, ey := range yCoords {
			if math.Abs(ey-posY) <= eps {
				found = true

				break
			}
		}

		if !found {
			yCoords = append(yCoords, posY)
		}
	}

	return yCoords
}

// repeatTableHeaders clones thead row ops onto every page that continues a
// multi-page table body, shifting body content down by the header height.
// Nested tables: each table repeats only its own thead.
func repeatTableHeaders(res *Result, contentH float64) {
	if res.root == nil || contentH <= 0 {
		return
	}

	var tables []*box

	var walk func(b *box)
	walk = func(b *box) {
		if b.kind == "table" && b.headerRows > 0 && b.headerRows < len(b.rows) {
			tables = append(tables, b)
		}

		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)

	for _, tblBox := range tables {
		nHdr := tblBox.headerRows
		if nHdr > len(tblBox.rows) {
			nHdr = len(tblBox.rows)
		}

		hdrFirst, hdrLast, hdrTop, hdrH := rowSpan(tblBox.rows[:nHdr], res)
		if hdrFirst < 0 || hdrH <= 0 {
			continue
		}

		firstPage := int(tblBox.y / contentH)
		pages := map[int]bool{}

		for _, row := range tblBox.rows[nHdr:] {
			top, _ := rowYBounds(row, res)
			if top < 0 {
				continue
			}

			pages[int(top/contentH)] = true
		}

		for page := range pages {
			if page <= firstPage {
				continue
			}

			pageTop := float64(page) * contentH
			bodyTop := -1.0
			shiftFrom, shiftTo := -1, -1

			for _, row := range tblBox.rows[nHdr:] {
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

			if shiftFrom >= 0 && bodyTop >= 0 && bodyTop < pageTop+hdrH-0.5 {
				dy := pageTop + hdrH - bodyTop
				if dy > 0 {
					shiftFlowY(res, shiftFrom, shiftTo, bodyTop-layoutSlack, dy)
				}
			}

			baseY := hdrTop

			for k := hdrFirst; k <= hdrLast && k < len(res.Ops); k++ {
				op := res.Ops[k]
				op.Y = pageTop + (op.Y - baseY)
				res.Ops = append(res.Ops, op)
			}
		}
	}
}

func rowOpRange(row []*box) (first, last int) {
	first, last = -1, -1

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

func rowYBounds(row []*box, res *Result) (top, bottom float64) {
	first, last := rowOpRange(row)
	if first < 0 || first >= len(res.Ops) {
		return -1, -1
	}

	top, bottom = res.Ops[first].Y, res.Ops[first].Y

	for k := first; k <= last && k < len(res.Ops); k++ {
		posY := res.Ops[k].Y

		height := res.Ops[k].H
		if res.Ops[k].Kind == OpText || res.Ops[k].Kind == OpBullet {
			height = res.Ops[k].Size * defaultLineHeightRatio
		}

		if posY < top {
			top = posY
		}

		if posY+height > bottom {
			bottom = posY + height
		}
	}

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

func rowSpan(rows [][]*box, res *Result) (first, last int, top, height float64) {
	first, last = -1, -1
	top, bottom := 0.0, 0.0
	set := false

	for _, row := range rows {
		face, lst := rowOpRange(row)
		if face < 0 {
			continue
		}

		if first < 0 || face < first {
			first = face
		}

		if lst > last {
			last = lst
		}

		right, rowB := rowYBounds(row, res)
		if right < 0 {
			continue
		}

		if !set || right < top {
			top = right
		}

		if !set || rowB > bottom {
			bottom = rowB
		}

		set = true
	}

	if first < 0 || !set {
		return -1, -1, 0, 0
	}

	height = bottom - top
	if height < 1 {
		height = 0

		for _, row := range rows {
			rowH := 0.0
			for _, cell := range row {
				if cell.height > rowH {
					rowH = cell.height
				}
			}

			height += rowH
		}
	}

	return first, last, top, height
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
func canvasToPDF(opX, opY float64, pageIdx int, contentH float64, opts PaintOptions, pageH float64) (x, y float64) {
	x = opts.MarginLeft + opX
	y = pageH - opts.MarginTop - opY + float64(pageIdx)*contentH

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

func drawText(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64, fontName string) {
	posX, posY := canvasToPDF(op.X, op.Y, pageIdx, contentH, opts, pageH)
	c.SetFillColor(op.R, op.G, op.B)

	if fontName == "" {
		fontName = "F0"
	}

	c.SetFont(fontName, op.Size)
	c.BeginText()

	if op.RotateDeg == 90 || op.RotateDeg == -90 {
		c.TextMatrix(0, 1, -1, 0, posX, posY)
	} else {
		c.TextAt(posX, posY)
	}
	// Fake bold only for Latin when CSS wants bold but the face is not bold.
	// Stroking CJK/Type0 outlines creates horizontal streak artifacts.
	fakeBold := FakeBoldFor(op)
	if fakeBold {
		c.SetLineWidth(op.Size * outlineStrokeRatio)
		c.TextRenderMode(two) // fill + stroke
	}

	c.TextShow(op.Text)

	if fakeBold {
		c.TextRenderMode(0)
	}

	c.EndText()
}

func drawImage(p *pdf.Page, c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, name string) error {
	posX, posY := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, p.Height())

	if name == "" {
		name = "I0"
	}

	if op.IsJPEG {
		return c.AddJPEGImage(name, posX, posY, op.W, op.H, op.Image)
	}

	return c.AddPNGImage(name, posX, posY, op.W, op.H, op.Image)
}

func drawLink(p *pdf.Page, op *Op, pageIdx int, contentH float64, opts PaintOptions) {
	// Same-document fragments are resolved to GoTo annotations in convert
	// (applyInternalLinks) after all pages exist.
	if len(op.URI) > 0 && op.URI[0] == '#' {
		return
	}

	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, p.Height())
	p.AddLinkURI([4]float64{x, y, x + op.W, y + op.H}, op.URI)
}

// drawLinkXform places a URI annotation. Annotations are page-space (not under
// content-stream CTM), so CSS transforms are applied to the canvas rect first.
func drawLinkXform(p *pdf.Page, op *Op, pageIdx int, contentH float64, opts PaintOptions) {
	if len(op.URI) > 0 && op.URI[0] == '#' {
		return
	}

	x1Val, yMin, xMax, y1Val := op.X, op.Y, op.X+op.W, op.Y+op.H
	if op.XformSet {
		corners := [4][2]float64{
			{x1Val, yMin}, {xMax, yMin}, {x1Val, y1Val}, {xMax, y1Val},
		}
		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

		for _, pt := range corners {
			textX, typeY := op.Xform.Apply(pt[0], pt[1])
			if textX < minX {
				minX = textX
			}

			if typeY < minY {
				minY = typeY
			}

			if textX > maxX {
				maxX = textX
			}

			if typeY > maxY {
				maxY = typeY
			}
		}

		x1Val, yMin, xMax, y1Val = minX, minY, maxX, maxY
	}

	llx, lly := canvasToPDF(x1Val, y1Val, pageIdx, contentH, opts, p.Height())
	urx, ury := canvasToPDF(xMax, yMin, pageIdx, contentH, opts, p.Height())

	if llx > urx {
		llx, urx = urx, llx
	}

	if lly > ury {
		lly, ury = ury, lly
	}

	p.AddLinkURI([4]float64{llx, lly, urx, ury}, op.URI)
}
