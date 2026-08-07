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

	// Split rect ops at page boundaries first. Sticky must run after this:
	// continuation fragments are created at the page-top boundary and would
	// otherwise cover sticky clones / reserved flow (fixture-31).
	splitCrossingRects(res, contentH, opPage)

	// Drop row shells left behind when text snapped to the next page
	// (fixture-31: empty white rows after Row 27 on page 1).
	stripOrphanRowChrome(res, contentH)

	// Close open tops on table continuations after rowspan/vertical splits.
	capTablePageBreaks(res, contentH)

	// Print-scoped sticky: clamp + continuation clones + reserve flow space.
	applyStickyPrint(res, contentH)

	// Re-derive pages after splits and sticky (new ops / Y shifts).
	opPage = make([]int, len(res.Ops))
	perPage := map[int][]int{}
	for i := range res.Ops {
		if res.Ops[i].Fixed {
			continue
		}
		p := int(res.Ops[i].Y / contentH)
		opPage[i] = p
		perPage[p] = append(perPage[p], i)
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
		p := doc.AddPage(opts.PageWidth, opts.PageHeight)
		c := p.Content()
		fontNames := map[*pdf.Font]string{}
		nextFont := 0
		resName := func(f *pdf.Font) string {
			if f == nil {
				return "F0"
			}
			if n, ok := fontNames[f]; ok {
				return n
			}
			n := "F" + strconv.Itoa(nextFont)
			nextFont++
			fontNames[f] = n
			c.UseEmbeddedFont(n, f)
			return n
		}
		nextImg := 0
		paintOp := func(op *Op, pg int) {
			if op.Kind == opKindNoop {
				return
			}
			if op.Kind == OpLinkURI {
				drawLinkXform(p, op, pg, contentH, opts)
				return
			}
			needGS := op.XformSet || (op.PaintOpacity > 0 && op.PaintOpacity < 1)
			if needGS {
				c.Save()
			}
			if op.XformSet {
				a, b, cc, d, e, f := pdfCTMFromCSS(op.Xform, pg, contentH, opts, p.Height())
				c.Transform(a, b, cc, d, e, f)
			}
			if op.PaintOpacity > 0 && op.PaintOpacity < 1 {
				c.SetOpacity(op.PaintOpacity)
			}
			switch op.Kind {
			case OpFillRect:
				drawFill(c, op, pg, contentH, opts, p.Height())
			case OpStrokeRect:
				drawStroke(c, op, pg, contentH, opts, p.Height())
			case OpLine:
				drawLine(c, op, pg, contentH, opts, p.Height())
			case OpText, OpBullet:
				drawText(c, op, pg, contentH, opts, p.Height(), resName(op.Font))
			case OpImage:
				name := "I" + strconv.Itoa(nextImg)
				nextImg++
				if err := drawImage(p, c, op, pg, contentH, opts, name); err != nil && paintErr == nil {
					paintErr = err
				}
			}
			if needGS {
				c.Restore()
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
	sort.SliceStable(idxs, func(i, j int) bool {
		a, b := ops[idxs[i]], ops[idxs[j]]
		az, bz := 0, 0
		if a.ZIndexSet {
			az = a.ZIndex
		}
		if b.ZIndexSet {
			bz = b.ZIndex
		}
		if az != bz {
			return az < bz
		}
		// Same stacking context: backgrounds/borders under text & images so
		// page-split fill remnants cannot cover continuation-row ink
		// (fixture-31 Row 28 vs next-row white fill).
		la, lb := paintLayer(a.Kind), paintLayer(b.Kind)
		if la != lb {
			return la < lb
		}
		return idxs[i] < idxs[j]
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
func StyleOf(op *Op) PaintStyle {
	if op == nil {
		return PaintStyle{FillAlpha: 1, StrokeWidth: 1}
	}
	ps := PaintStyle{
		FillR: op.R, FillG: op.G, FillB: op.B, FillAlpha: 1,
		StrokeWidth: op.Width,
	}
	if ps.StrokeWidth <= 0 {
		ps.StrokeWidth = 1
	}
	// Pre-composite translucent fills against white paper (PDF path).
	if op.Alpha > 0 && op.Alpha < 1 {
		a := op.Alpha
		ps.FillR = op.R*a + (1 - a)
		ps.FillG = op.G*a + (1 - a)
		ps.FillB = op.B*a + (1 - a)
		ps.FillAlpha = 1
	} else if op.Alpha > 0 {
		ps.FillAlpha = op.Alpha
	}
	ps.FakeBold = FakeBoldFor(op)
	return ps
}

// FakeBoldFor reports whether CSS bold should be synthesized for op (Latin
// only; CJK stroking produces streak artifacts).
func FakeBoldFor(op *Op) bool {
	if op == nil || !op.Bold || (op.Font != nil && op.Font.Bold()) {
		return false
	}
	for _, r := range op.Text {
		if r > 0xFF {
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
	resName := func(f *pdf.Font) string {
		if f == nil {
			return "F0"
		}
		if n, ok := fontNames[f]; ok {
			return n
		}
		n := "B" + strconv.Itoa(nextFont)
		nextFont++
		fontNames[f] = n
		c.UseEmbeddedFont(n, f)
		return n
	}
	nextImg := 0
	po := opts.Margins
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
		op := &ops[i]
		if op.Kind == OpLinkURI || op.Kind == opKindNoop {
			continue
		}
		needGS := op.XformSet || (op.PaintOpacity > 0 && op.PaintOpacity < 1)
		if needGS {
			c.Save()
		}
		if op.XformSet && !useSimple {
			a, b, cc, d, e, f := pdfCTMFromCSS(op.Xform, 0, contentH, po, pageH)
			c.Transform(a, b, cc, d, e, f)
		}
		if op.PaintOpacity > 0 && op.PaintOpacity < 1 {
			c.SetOpacity(op.PaintOpacity)
		}
		if useSimple {
			if err := paintOpBandSimple(c, p, op, opts, resName(op.Font), &nextImg); err != nil && firstErr == nil {
				firstErr = err
			}
		} else {
			switch op.Kind {
			case OpFillRect:
				drawFill(c, op, 0, contentH, po, pageH)
			case OpStrokeRect:
				drawStroke(c, op, 0, contentH, po, pageH)
			case OpLine:
				drawLine(c, op, 0, contentH, po, pageH)
			case OpText, OpBullet:
				drawText(c, op, 0, contentH, po, pageH, resName(op.Font))
			case OpImage:
				name := "I" + strconv.Itoa(nextImg)
				nextImg++
				if err := drawImage(p, c, op, 0, contentH, po, name); err != nil && firstErr == nil {
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

func paintOpBandSimple(c *pdf.Content, p *pdf.Page, op *Op, opts BandOptions, fontName string, nextImg *int) error {
	x := opts.OriginX + op.X
	switch op.Kind {
	case OpFillRect:
		ps := StyleOf(op)
		y := opts.OriginY - (op.Y + op.H)
		c.SetFillColor(ps.FillR, ps.FillG, ps.FillB)
		c.Rect(x, y, op.W, op.H)
		c.Fill()
	case OpStrokeRect:
		y := opts.OriginY - (op.Y + op.H)
		c.SetStrokeColor(op.R, op.G, op.B)
		c.SetLineWidth(1)
		c.Rect(x, y, op.W, op.H)
		c.Stroke()
	case OpLine:
		y1 := opts.OriginY - op.Y
		y2 := opts.OriginY - (op.Y + op.H)
		w := op.Width
		if w <= 0 {
			w = 1
		}
		c.SetStrokeColor(op.R, op.G, op.B)
		c.SetLineWidth(w)
		c.MoveTo(x, y1)
		c.LineTo(opts.OriginX+op.X+op.W, y2)
		c.Stroke()
	case OpText, OpBullet:
		y := opts.OriginY - op.Y
		c.SetFillColor(op.R, op.G, op.B)
		if fontName == "" {
			fontName = "F0"
		}
		c.SetFont(fontName, op.Size)
		c.BeginText()
		c.TextAt(x, y)
		if FakeBoldFor(op) {
			c.SetLineWidth(op.Size * 0.06)
			c.TextRenderMode(2)
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
			return c.AddJPEGImage(name, x, y, op.W, op.H, op.Image)
		}
		return c.AddPNGImage(name, x, y, op.W, op.H, op.Image)
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
		op := &res.Ops[i]
		if op.Fixed || op.Kind != OpLine {
			continue
		}
		if op.H > 2 && (op.W < 1 || op.W < op.H*0.05) {
			verts = append(verts, vseg{op.X, op.Y, op.Y + op.H, op.Width, op.R, op.G, op.B})
			continue
		}
		if op.W > 2 && op.H < 1 {
			horiz = append(horiz, hseg{op.X, op.X + op.W, op.Y, op.Width, op.R, op.G, op.B})
		}
	}
	// Group verticals that share a start Y (row top) or end Y (row bottom).
	roundY := func(y float64) int { return int(math.Round(y * 2)) } // 0.5pt bins
	type cluster struct {
		y           float64
		minX, maxX  float64
		bw, r, g, b float64
		n           int
	}
	clusterAt := func(byStart bool) map[int]*cluster {
		out := map[int]*cluster{}
		for _, v := range verts {
			keyY := v.y0
			if !byStart {
				keyY = v.y1
			}
			k := roundY(keyY)
			c := out[k]
			if c == nil {
				c = &cluster{y: keyY, minX: v.x, maxX: v.x, bw: v.w, r: v.r, g: v.g, b: v.b, n: 1}
				out[k] = c
				continue
			}
			c.n++
			if v.x < c.minX {
				c.minX = v.x
			}
			if v.x > c.maxX {
				c.maxX = v.x
			}
			// Prefer average y so we sit on the dominant edge.
			c.y = (c.y*float64(c.n-1) + keyY) / float64(c.n)
		}
		return out
	}
	hCoverage := func(y, minX, maxX float64) (full bool, covMin, covMax float64, has bool) {
		for _, h := range horiz {
			if math.Abs(h.y-y) > eps {
				continue
			}
			// Only count segments that overlap the vertical band.
			if h.x1 < minX-eps || h.x0 > maxX+eps {
				continue
			}
			if !has {
				covMin, covMax, has = h.x0, h.x1, true
			} else {
				if h.x0 < covMin {
					covMin = h.x0
				}
				if h.x1 > covMax {
					covMax = h.x1
				}
			}
		}
		if !has {
			return false, 0, 0, false
		}
		full = covMin <= minX+eps && covMax >= maxX-eps
		return full, covMin, covMax, true
	}
	seal := func(y, minX, maxX, bw, r, g, b float64) {
		if maxX-minX < 20 || bw < 0 {
			return
		}
		if bw < 0.3 {
			bw = 0.5
		}
		// Avoid exact duplicates.
		for _, h := range horiz {
			if math.Abs(h.y-y) <= 0.5 && math.Abs(h.x0-minX) <= eps && math.Abs(h.x1-maxX) <= eps {
				return
			}
		}
		op := Op{
			Kind: OpLine, X: minX, Y: y, W: maxX - minX, H: 0,
			Width: bw, R: r, G: g, B: b,
		}
		res.Ops = append(res.Ops, op)
		horiz = append(horiz, hseg{minX, maxX, y, bw, r, g, b})
	}

	// (1) Classic page-top stubs.
	for p := 1; p <= maxPage; p++ {
		pageTop := float64(p) * contentH
		var minX, maxX, bw, r, g, b float64
		n := 0
		for _, v := range verts {
			if v.y0 >= pageTop-eps && v.y0 <= pageTop+eps {
				if n == 0 {
					minX, maxX, bw, r, g, b = v.x, v.x, v.w, v.r, v.g, v.b
				} else {
					if v.x < minX {
						minX = v.x
					}
					if v.x > maxX {
						maxX = v.x
					}
				}
				n++
			}
		}
		if n < 2 {
			continue
		}
		if full, _, _, _ := hCoverage(pageTop, minX, maxX); full {
			continue
		}
		seal(pageTop, minX, maxX, bw, r, g, b)
	}

	// (2) Seal incomplete tops of multi-column vertical clusters that start a
	// continuation-page body band (under repeated thead or at page top).
	// Mid-table rowspan holes keep skipped tops so continuous year cells stay
	// unsplit; only the page-fragment open edge is closed.
	for _, c := range clusterAt(true) {
		if c.n < 3 || c.maxX-c.minX < 20 {
			continue
		}
		full, _, _, _ := hCoverage(c.y, c.minX, c.maxX)
		if full {
			continue
		}
		page := int(c.y / contentH)
		if page <= 0 {
			continue
		}
		pageTop := float64(page) * contentH
		// Body under thead typically starts within ~header+padding of page top.
		if c.y > pageTop+80 {
			continue
		}
		seal(c.y, c.minX, c.maxX, c.bw, c.r, c.g, c.b)
	}
	// Row bottoms: seal when verticals end near a page bottom and no full
	// horizontal closes the strip (next row's top moved to the following page).
	for _, c := range clusterAt(false) {
		if c.n < 3 || c.maxX-c.minX < 20 {
			continue
		}
		page := int((c.y - 0.01) / contentH)
		pageBot := float64(page+1) * contentH
		// Only near the page boundary (row ended as last on page).
		if c.y < pageBot-40 || c.y > pageBot+eps {
			continue
		}
		if full, _, _, _ := hCoverage(c.y, c.minX, c.maxX); full {
			continue
		}
		if page >= 0 {
			seal(c.y, c.minX, c.maxX, c.bw, c.r, c.g, c.b)
		}
	}
}

// paginateOps assigns every op a page. Crossing text/image/link ops snap to
// the next page boundary (taking following flow with them so row spacing is
// preserved); then page-break policies are applied as canvas-Y shifts; finally
// pages derive from the final Y positions. Rect-type ops crossing a boundary
// are split by Paint.
func paginateOps(res *Result, contentH float64) []int {
	for i := 0; i < len(res.Ops); i++ {
		op := &res.Ops[i]
		if op.Fixed {
			continue
		}
		switch op.Kind {
		case OpText, OpBullet, OpImage, OpLinkURI:
			opH := op.H
			if op.Kind == OpText || op.Kind == OpBullet {
				opH = op.Size * 1.2
			}
			page := int(op.Y / contentH)
			if page < 0 {
				page = 0
			}
			boundary := float64(page+1) * contentH
			if op.Y+opH > boundary+1e-9 {
				if dy := boundary - op.Y; dy > 1e-6 {
					// Snap text (+ following flow). Same-row fills sit above the
					// baseline; include them in dy via minY so their tops clear
					// onto this page with the text (fixture-31 Row 28 white bg).
					// Keep chrome matching tight (one row) so table reports do
					// not inflate dy. Never clamp fill tops to `boundary` alone
					// — that collapses them onto the text Y and leaves section
					// gray showing through the ascent/padding band.
					oldY := op.Y
					minY := oldY
					var chrome []int
					for j := range res.Ops {
						o := &res.Ops[j]
						if o.Fixed || j == i {
							continue
						}
						if o.Kind != OpFillRect && o.Kind != OpStrokeRect {
							continue
						}
						if o.H <= 0.5 || o.H > 40 {
							continue
						}
						if o.Y > oldY+0.5 || o.Y+o.H < oldY-0.5 {
							continue
						}
						if oldY-o.Y > o.H+2 {
							continue
						}
						chrome = append(chrome, j)
						if o.Y < minY {
							minY = o.Y
						}
					}
					// Leave room for ascenders above the baseline so snapped
					// lines do not paint into the top margin (page-4/5 bleed).
					lead := 0.0
					if op.Kind == OpText || op.Kind == OpBullet {
						lead = op.Size * 0.75
						if lead < 8 {
							lead = 8
						}
					}
					dy = boundary + lead - minY
					shiftFlowY(res, i, i, oldY-0.01, dy)
					for _, j := range chrome {
						o := &res.Ops[j]
						if o.Y < oldY-0.01 {
							o.Y += dy
						}
					}
				} else {
					op.Y = boundary
				}
			}
		}
	}
	for iter := 0; iter < 10; iter++ {
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
	out := make([]Op, 0, len(res.Ops)+8)
	for i := range res.Ops {
		op := res.Ops[i]
		start := len(out)
		if op.Fixed || !isSplittable(&op) || op.H <= 0 {
			out = append(out, op)
			spans[i] = opSpan{start: start, end: len(out) - 1}
			continue
		}
		guard := 0
		for op.H > 1e-9 {
			guard++
			if guard > 10000 {
				// Defensive: never hang the paint pipeline.
				out = append(out, op)
				break
			}
			// Epsilon bump so Y exactly on a page top maps to that page, not
			// the previous one (int truncates 52.0-ε down to 51).
			p := int((op.Y + 1e-6) / contentH)
			if p < 0 {
				p = 0
			}
			boundary := float64(p+1) * contentH
			if op.Y+op.H <= boundary+1e-9 {
				out = append(out, op)
				break
			}
			firstH := boundary - op.Y
			if firstH <= 1e-6 {
				// Start is at/past boundary; advance to next page top via p++.
				op.Y = float64(p+1) * contentH
				continue
			}
			frag := op
			frag.H = firstH
			out = append(out, frag)
			op.Y = boundary
			op.H -= firstH
		}
		spans[i] = opSpan{start: start, end: len(out) - 1}
	}
	res.Ops = out
	remapBoxOpRanges(res.root, spans)
}

// remapBoxOpRanges updates the layout-owned operation ranges after a display
// list rewrite. In particular, a source rectangle can become two or more
// page fragments; mapping the box end to the final fragment keeps pagination,
// sticky/fixed stamping, and ElementLocation ownership aligned.
func remapBoxOpRanges(b *box, spans []opSpan) {
	if b == nil {
		return
	}
	if b.opStart >= 0 && b.opEnd >= b.opStart && b.opStart < len(spans) && b.opEnd < len(spans) {
		b.opStart = spans[b.opStart].start
		b.opEnd = spans[b.opEnd].end
	}
	for _, child := range b.children {
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
	for p := 0; p <= maxPage; p++ {
		pageTop := float64(p) * contentH
		pageBot := pageTop + contentH
		lastInkBot := pageTop
		hasInk := false
		for i := range res.Ops {
			op := &res.Ops[i]
			if op.Fixed || op.Y < pageTop-1e-9 || op.Y >= pageBot-1e-9 {
				continue
			}
			var bot float64
			switch op.Kind {
			case OpText, OpBullet:
				h := op.Size * 1.2
				if op.H > h {
					h = op.H
				}
				if h < 4 {
					h = 4
				}
				bot = op.Y + h
			case OpImage:
				bot = op.Y + op.H
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
		for i := range res.Ops {
			op := &res.Ops[i]
			if op.Fixed || op.StickyID != 0 {
				continue
			}
			if op.Y < pageTop-1e-9 || op.Y >= pageBot-1e-9 {
				continue
			}
			switch op.Kind {
			case OpFillRect, OpStrokeRect:
				// Row-sized shells whose center sits below the last ink are
				// empty trailing row backgrounds (not the cell that holds the
				// last text, whose center is at/above the baseline band).
				if op.H <= 0.5 || op.H > 40 {
					continue
				}
				if op.Y+op.H/2 > lastInkBot+0.5 {
					op.H = 0
					stripped = true
				}
			case OpLine:
				if op.H >= 1 {
					continue
				}
				// Horizontal rule below the last ink (empty row separator).
				if op.Y > lastInkBot+0.5 {
					op.Width = 0
					stripped = true
				}
			}
		}
		if stripped {
			// Tighten the last row fill so padding under the final baseline does
			// not read as another empty row (fixture-31 Row 27 cell).
			const underPad = 8.0
			for i := range res.Ops {
				op := &res.Ops[i]
				if op.Fixed || op.StickyID != 0 {
					continue
				}
				if op.Y < pageTop-1e-9 || op.Y >= pageBot-1e-9 {
					continue
				}
				if (op.Kind == OpFillRect || op.Kind == OpStrokeRect) &&
					op.H > 0.5 && op.H <= 40 &&
					op.Y < lastInkBot && op.Y+op.H > lastInkBot+underPad+2 {
					op.H = lastInkBot + underPad - op.Y
					if op.H < 1 {
						op.H = 1
					}
				}
				if op.Kind == OpLine && op.H < 1 && op.Width > 0 &&
					op.Y > lastInkBot+underPad+1 && op.Y < lastInkBot+40 {
					op.Y = lastInkBot + underPad
				}
			}
		}
		// Pull section washes / borders up to the last row chrome / ink so grey
		// does not pad an empty band to the page bottom (fixture-31 page 1).
		// Only section-colored chrome is clipped — arbitrary tall fills
		// (TestBoundaryFillSplit) are left to the normal page-split remnant.
		contentBot := lastInkBot
		for i := range res.Ops {
			op := &res.Ops[i]
			if op.Fixed || op.StickyID != 0 {
				continue
			}
			if op.Y < pageTop-1e-9 || op.Y >= pageBot-1e-9 {
				continue
			}
			if (op.Kind == OpFillRect || op.Kind == OpStrokeRect) && op.H > 0.5 && op.H <= 40 {
				if bot := op.Y + op.H; bot > contentBot {
					contentBot = bot
				}
			}
			if op.Kind == OpLine && op.H < 1 && op.Width > 0 && op.Y > contentBot {
				contentBot = op.Y
			}
		}
		if pageBot-contentBot < 8 {
			continue
		}
		for i := range res.Ops {
			op := &res.Ops[i]
			if op.Fixed || op.StickyID != 0 {
				continue
			}
			if op.Y < pageTop-1e-9 || op.Y >= pageBot-1e-9 {
				continue
			}
			switch op.Kind {
			case OpFillRect:
				if op.H > 40 && isSectionWashRGB(op.R, op.G, op.B) &&
					op.Y+op.H > contentBot+1 && op.Y < contentBot {
					op.H = contentBot - op.Y
				}
			case OpLine:
				if op.H > 40 && nearSectionBorderRGB(op.R, op.G, op.B) &&
					op.Y+op.H > contentBot+1 && op.Y < contentBot {
					op.H = contentBot - op.Y
				} else if op.H < 1 && op.Width > 0 && nearSectionBorderRGB(op.R, op.G, op.B) &&
					op.Y > contentBot+1 && op.Y > pageBot-30 {
					op.Y = contentBot
				}
			}
		}
	}
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
	for i := from; i <= to; i++ {
		if i >= 0 && i < len(res.Ops) && res.Ops[i].Fixed {
			continue
		}
		res.Ops[i].Y += dy
	}
	for i := range res.Ops {
		if i < from || i > to {
			if res.Ops[i].Y > fromY {
				res.Ops[i].Y += dy
			}
		}
	}
	if res.root == nil {
		return
	}
	var walk func(b *box)
	walk = func(b *box) {
		if b.y > fromY || (b.y == fromY && b.opStart >= from && b.opEnd <= to) {
			b.y += dy
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
}

// shiftOpsOnly moves ops in [from,to] by dy without dragging later flow.
// Used when rejoining a page-break-after:avoid box to a following box that
// already sits on the next page.
func shiftOpsOnly(res *Result, from, to int, dy float64) {
	for i := from; i <= to; i++ {
		if i >= 0 && i < len(res.Ops) && res.Ops[i].Fixed {
			continue
		}
		res.Ops[i].Y += dy
	}
}

// avoidInside walks post-order and moves page-break-inside: avoid boxes wholly
// to the next page when they span multiple pages but fit one content height.
func avoidInside(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(b *box) bool {
		changed := false
		for _, c := range b.children {
			if walk(c) {
				changed = true
			}
		}
		if b.style.PageBreakInside == "avoid" && b.h > 0 {
			h := b.h
			// Prefer ink extent when taller than the border box (rowspan /
			// deferred paint can make ops protrude past b.h — wiki awards).
			if b.opStart <= b.opEnd && b.opStart >= 0 && b.opEnd < len(res.Ops) {
				bot := b.y
				for k := b.opStart; k <= b.opEnd; k++ {
					op := res.Ops[k]
					ob := op.Y
					switch op.Kind {
					case OpText, OpBullet:
						ob += op.Size * 1.2
					default:
						if op.H > 0 {
							ob += op.H
						}
					}
					if ob > bot {
						bot = ob
					}
				}
				if ink := bot - b.y; ink > h {
					h = ink
				}
			}
			lo := int(b.y / contentH)
			hi := int((b.y + h) / contentH)
			if hi > lo && h <= contentH+0.01 {
				remaining := float64(lo+1)*contentH - b.y
				// Prefer splitting over large empty bands. Use border-box
				// height (b.h), not ink: after line-snap, ink can span a
				// page gap while the box is still a short list item —
				// classifying by ink disabled the short-box guard and
				// cascaded 100–150pt gaps (wiki references).
				if preferSplitOverBlank(remaining, b.h, contentH) {
					return changed
				}
				// Large boxes: also prefer split when less than half the box
				// fits (rowspan tables / tall avoid blocks).
				if remaining < b.h*0.5 && b.h > contentH*0.35 {
					return changed
				}
				dy := float64(lo+1)*contentH - b.y
				if dy > 0.01 {
					shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
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
	var walk func(b *box) bool
	walk = func(b *box) bool {
		if b.style.PageBreakBefore == "always" {
			lastBefore := 0.0
			for i := 0; i < b.opStart; i++ {
				if res.Ops[i].Y > lastBefore {
					lastBefore = res.Ops[i].Y
				}
			}
			loPage := int(b.y / contentH)
			lastPage := int(lastBefore / contentH)
			if loPage <= lastPage {
				dy := float64(lastPage+1)*contentH - b.y
				if dy > 0 {
					shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
					return true
				}
			}
		}
		changed := false
		for _, c := range b.children {
			if walk(c) {
				changed = true
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
	for i, b := range boxes {
		var next *box
		for j := i + 1; j < len(boxes); j++ {
			if boxes[j].opStart <= boxes[j].opEnd {
				next = boxes[j]
				break
			}
		}
		if next == nil || b.opStart > b.opEnd {
			continue
		}
		lastY := res.Ops[b.opStart].Y
		for k := b.opStart + 1; k <= b.opEnd; k++ {
			if res.Ops[k].Y > lastY {
				lastY = res.Ops[k].Y
			}
		}
		lastPage := int(lastY / contentH)
		switch {
		case b.style.PageBreakAfter == "always":
			dy := float64(lastPage+1)*contentH - next.y
			if dy > 0 {
				shiftFlowY(res, next.opStart, next.opEnd, next.y, dy)
				changed = true
			}
		case b.style.PageBreakAfter == "avoid":
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
			need := b.h
			if need < 1 {
				need = 12
			}
			if b.opStart <= b.opEnd && b.opStart >= 0 && b.opEnd < len(res.Ops) {
				top, bot := b.y, b.y
				for k := b.opStart; k <= b.opEnd; k++ {
					op := res.Ops[k]
					y0, y1 := op.Y, op.Y
					switch op.Kind {
					case OpText, OpBullet:
						y0 = op.Y - op.Size*0.8
						y1 = op.Y + op.Size*0.35
					case OpLine:
						if op.H == 0 {
							y1 = op.Y + math.Max(op.Width, 1)
						} else {
							y1 = op.Y + op.H
						}
					default:
						if op.H > 0 {
							y1 = op.Y + op.H
						}
					}
					if y0 < top {
						top = y0
					}
					if y1 > bot {
						bot = y1
					}
				}
				if ink := bot - top; ink > need {
					need = ink
				}
				if ink := bot - b.y; ink > need {
					need = ink
				}
			}
			const gap = 10.0
			need += gap
			bandTop := pageStart + need
			minY := bandTop
			minIdx := -1
			for i := range res.Ops {
				op := &res.Ops[i]
				if op.Fixed {
					continue
				}
				if int(op.Y/contentH) != nextPage {
					continue
				}
				if op.Y < minY {
					minY = op.Y
					minIdx = i
				}
			}
			if minIdx >= 0 && minY < bandTop-0.01 {
				push := bandTop - minY
				shiftFlowY(res, minIdx, minIdx, minY-0.01, push)
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
			dy := target - b.y
			if dy > 0.001 {
				shiftOpsOnly(res, b.opStart, b.opEnd, dy)
				b.y += dy
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
	walk = func(b *box) bool {
		changed := false
		for _, c := range b.children {
			if walk(c) {
				changed = true
			}
		}
		for _, row := range b.rows {
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
				h := cell.h
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
			lo, hi := int(rowTop/contentH), int(rowBottom/contentH)
			if hi > lo {
				// Move only to the next page start. Using hi*contentH when the
				// row's measured bottom spans multiple pages (e.g. rowspan
				// paint height leaking into rowBoxH) skipped blank pages
				// between filmography and awards on long wiki tables.
				dy := float64(lo+1)*contentH - rowTop
				if dy > 0.01 {
					// fromY slightly above rowTop so border-collapse grid
					// lines that sit exactly on the row edge (and later
					// rows / chrome below) shift with the cells — otherwise
					// content moves and the grid stays behind (gapped /
					// misaligned music-video tables across page breaks).
					shiftFlowY(res, first, last, rowTop-0.01, dy)
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
	for i, b := range boxes {
		if b.node == nil || !isHeadingName(b.node.Name) || b.opStart > b.opEnd {
			continue
		}
		page := int(b.y / contentH)
		room := float64(page+1)*contentH - (b.y + b.h)
		if room >= 24 { // ~2 lines at 12pt
			continue
		}
		// Find next flow sibling with ops.
		var next *box
		for j := i + 1; j < len(boxes); j++ {
			if boxes[j].opStart <= boxes[j].opEnd && boxes[j].y >= b.y {
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
		dy := float64(page+1)*contentH - b.y
		if dy > 0 {
			shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
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
	walk = func(b *box) {
		for _, c := range b.children {
			walk(c)
		}
		if b.kind != "block" || b.h <= 0 || b.opStart > b.opEnd {
			return
		}
		// Nested block containers: children apply Rule 3; only heuristic on
		// short straddlers here.
		if hasNestedFlowChild(b) {
			if orphansWidowsHeuristic(res, b, contentH) {
				changed = true
			}
			return
		}
		lines := countBlockLineYs(res, b)
		if len(lines) == 0 {
			if orphansWidowsHeuristic(res, b, contentH) {
				changed = true
			}
			return
		}
		orphans := b.style.Orphans
		if orphans < 1 {
			orphans = 2
		}
		widows := b.style.Widows
		if widows < 1 {
			widows = 2
		}
		lo := int(b.y / contentH)
		hi := int((b.y + b.h) / contentH)
		if hi <= lo {
			return
		}
		boundary := float64(lo+1) * contentH
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
			if orphansWidowsHeuristic(res, b, contentH) {
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
		if b.h <= contentH+0.01 {
			remaining := float64(lo+1)*contentH - b.y
			if preferSplitOverBlank(remaining, b.h, contentH) {
				return
			}
			dy := float64(hi)*contentH - b.y
			if dy > 1e-6 {
				shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
				changed = true
			}
		}
	}
	walk(res.root)
	return changed
}

// orphansWidowsHeuristic is the phase-18 geometric fallback: short blocks
// (~2–4 lines) that straddle a page boundary move wholly when they fit.
func orphansWidowsHeuristic(res *Result, b *box, contentH float64) bool {
	if b.h < 14 || b.h > 60 {
		return false
	}
	lo := int(b.y / contentH)
	hi := int((b.y + b.h) / contentH)
	if hi <= lo || b.h > contentH {
		return false
	}
	remaining := float64(lo+1)*contentH - b.y
	if preferSplitOverBlank(remaining, b.h, contentH) {
		return false
	}
	dy := float64(hi)*contentH - b.y
	if dy <= 1e-6 {
		return false
	}
	shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
	return true
}

// preferSplitOverBlank reports whether a keep-together shift would leave an
// unacceptably large empty band on the current page. Shared by avoidInside
// and orphans/widows so dense page-break-inside:avoid lists do not cascade
// expanding gaps between consecutive short blocks.
//
// h should be the border-box height (not ink): line-snap can inflate ink
// across a page boundary without making the box "tall".
func preferSplitOverBlank(remaining, h, contentH float64) bool {
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
	if h > 0 && h < contentH*0.35 {
		// Allow at most ~1.2 line-heights of trailing blank (or half the
		// box), whichever is larger — true end-of-page overflow only.
		// Tighter than the prior 24pt/0.75h guard so modest remainders
		// never keep short avoid siblings apart.
		maxBlank := 14.0
		if h*0.5 > maxBlank {
			maxBlank = h * 0.5
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
	ys := make([]float64, 0, 8)
	end := b.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}
	for i := b.opStart; i <= end; i++ {
		op := &res.Ops[i]
		if op.Kind != OpText && op.Kind != OpBullet {
			continue
		}
		y := op.Y
		found := false
		for _, ey := range ys {
			if math.Abs(ey-y) <= eps {
				found = true
				break
			}
		}
		if !found {
			ys = append(ys, y)
		}
	}
	return ys
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

	for _, tb := range tables {
		nHdr := tb.headerRows
		if nHdr > len(tb.rows) {
			nHdr = len(tb.rows)
		}
		hdrFirst, hdrLast, hdrTop, hdrH := rowSpan(tb.rows[:nHdr], res)
		if hdrFirst < 0 || hdrH <= 0 {
			continue
		}
		firstPage := int(tb.y / contentH)
		pages := map[int]bool{}
		for _, row := range tb.rows[nHdr:] {
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
			for _, row := range tb.rows[nHdr:] {
				f, l := rowOpRange(row)
				if f < 0 {
					continue
				}
				top, _ := rowYBounds(row, res)
				if int(top/contentH) < page {
					continue
				}
				if bodyTop < 0 || top < bodyTop {
					bodyTop = top
				}
				if shiftFrom < 0 || f < shiftFrom {
					shiftFrom = f
				}
				if l > shiftTo {
					shiftTo = l
				}
			}
			if shiftFrom >= 0 && bodyTop >= 0 && bodyTop < pageTop+hdrH-0.5 {
				dy := pageTop + hdrH - bodyTop
				if dy > 0 {
					shiftFlowY(res, shiftFrom, shiftTo, bodyTop-0.01, dy)
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
		y := res.Ops[k].Y
		h := res.Ops[k].H
		if res.Ops[k].Kind == OpText || res.Ops[k].Kind == OpBullet {
			h = res.Ops[k].Size * 1.2
		}
		if y < top {
			top = y
		}
		if y+h > bottom {
			bottom = y + h
		}
	}
	for _, cell := range row {
		if cell.h > 0 && cell.y+cell.h > bottom {
			bottom = cell.y + cell.h
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
		f, l := rowOpRange(row)
		if f < 0 {
			continue
		}
		if first < 0 || f < first {
			first = f
		}
		if l > last {
			last = l
		}
		rt, rb := rowYBounds(row, res)
		if rt < 0 {
			continue
		}
		if !set || rt < top {
			top = rt
		}
		if !set || rb > bottom {
			bottom = rb
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
				if cell.h > rowH {
					rowH = cell.h
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
	walk = func(b *box) {
		if b.node != nil {
			page := -1
			if b.opStart <= b.opEnd && b.opStart < len(opPage) {
				page = opPage[b.opStart]
			}
			if page < 0 {
				page = int(b.y / contentH)
				if page < 0 {
					page = 0
				}
			}
			res.Locations = append(res.Locations, ElementLocation{
				Node: b.node, Page: page, X: b.x, Y: b.y, W: b.w, H: b.h,
			})
		}
		for _, c := range b.children {
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

func drawLine(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x1, y1 := canvasToPDF(op.X, op.Y, pageIdx, contentH, opts, pageH)
	x2, y2 := canvasToPDF(op.X+op.W, op.Y+op.H, pageIdx, contentH, opts, pageH)
	w := op.Width
	if w <= 0 {
		w = 1
	}
	c.SetStrokeColor(op.R, op.G, op.B)
	c.SetLineWidth(w)
	c.MoveTo(x1, y1)
	c.LineTo(x2, y2)
	c.Stroke()
}

func drawText(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64, fontName string) {
	x, y := canvasToPDF(op.X, op.Y, pageIdx, contentH, opts, pageH)
	c.SetFillColor(op.R, op.G, op.B)
	if fontName == "" {
		fontName = "F0"
	}
	c.SetFont(fontName, op.Size)
	c.BeginText()
	if op.RotateDeg == 90 || op.RotateDeg == -90 {
		c.TextMatrix(0, 1, -1, 0, x, y)
	} else {
		c.TextAt(x, y)
	}
	// Fake bold only for Latin when CSS wants bold but the face is not bold.
	// Stroking CJK/Type0 outlines creates horizontal streak artifacts.
	fakeBold := FakeBoldFor(op)
	if fakeBold {
		c.SetLineWidth(op.Size * 0.06)
		c.TextRenderMode(2) // fill + stroke
	}
	c.TextShow(op.Text)
	if fakeBold {
		c.TextRenderMode(0)
	}
	c.EndText()
}

func drawImage(p *pdf.Page, c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, name string) error {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, p.Height())
	if name == "" {
		name = "I0"
	}
	if op.IsJPEG {
		return c.AddJPEGImage(name, x, y, op.W, op.H, op.Image)
	}
	return c.AddPNGImage(name, x, y, op.W, op.H, op.Image)
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
	x0, y0, x1, y1 := op.X, op.Y, op.X+op.W, op.Y+op.H
	if op.XformSet {
		corners := [4][2]float64{
			{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1},
		}
		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
		for _, pt := range corners {
			tx, ty := op.Xform.Apply(pt[0], pt[1])
			if tx < minX {
				minX = tx
			}
			if ty < minY {
				minY = ty
			}
			if tx > maxX {
				maxX = tx
			}
			if ty > maxY {
				maxY = ty
			}
		}
		x0, y0, x1, y1 = minX, minY, maxX, maxY
	}
	llx, lly := canvasToPDF(x0, y1, pageIdx, contentH, opts, p.Height())
	urx, ury := canvasToPDF(x1, y0, pageIdx, contentH, opts, p.Height())
	if llx > urx {
		llx, urx = urx, llx
	}
	if lly > ury {
		lly, ury = ury, lly
	}
	p.AddLinkURI([4]float64{llx, lly, urx, ury}, op.URI)
}
