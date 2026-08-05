package layout

// Print-scoped position:sticky (CSS Positioned Layout Level 3 lite).
//
// Scrollport: each page's content box [pageY, pageY+contentH) — not a CSS
// overflow:auto scroller (PDF has no scroll). Sticky is not position:fixed:
// it does not stamp on every page; it only clamps (and clones onto
// continuation pages) where the containing block intersects the page.
//
// Overflow-scroll sticky inside overflow:auto/scroll boxes is unsupported
// and degrades to in-flow layout without page-edge sticking.
//
// Insets: non-auto top/right/bottom/left define sticky-view edges; auto means
// no constraint on that side. Containing-block limit: the sticky margin box
// stays inside its nearest block ancestor (same CB as relative).

// tagSticky records sticky insets and stamps StickyID on the box's ops so
// pagination can find them after parent prependChrome shifts op indices.
func (e *engine) tagSticky(b *box) {
	if b == nil || b.style.Position != "sticky" {
		return
	}
	b.sticky = true
	e.stickySeq++
	b.stickyID = e.stickySeq
	st := b.style
	if !st.TopAuto {
		b.stickyTopSet = true
		b.stickyTop = e.scalePt(st.Top)
	}
	if !st.RightAuto {
		b.stickyRightSet = true
		b.stickyRight = e.scalePt(st.Right)
	}
	if !st.BottomAuto {
		b.stickyBottomSet = true
		b.stickyBottom = e.scalePt(st.Bottom)
	}
	if !st.LeftAuto {
		b.stickyLeftSet = true
		b.stickyLeft = e.scalePt(st.Left)
	}
	if b.opEnd >= b.opStart && b.opStart >= 0 {
		for i := b.opStart; i <= b.opEnd && i < len(e.ops); i++ {
			e.ops[i].StickyID = b.stickyID
		}
	}
}

func (b *box) hasStickyInset() bool {
	return b != nil && b.sticky &&
		(b.stickyTopSet || b.stickyRightSet || b.stickyBottomSet || b.stickyLeftSet)
}

// applyStickyPrint clamps sticky boxes to each page's content box (scrollport)
// and clones stuck paint ops onto continuation pages where the containing
// block still intersects. Called after flow pagination has settled.
func applyStickyPrint(res *Result, contentH float64) {
	if res == nil || res.root == nil || contentH <= 0 {
		return
	}
	var stickies []*box
	var walk func(b, parent *box)
	walk = func(b, parent *box) {
		if b.sticky {
			if parent != nil {
				b.cbX, b.cbY, b.cbW, b.cbH = parent.x, parent.y, parent.w, parent.h
			} else {
				h := res.Height
				if h < b.y+b.h {
					h = b.y + b.h
				}
				b.cbX, b.cbY, b.cbW, b.cbH = 0, 0, res.Width, h
			}
			stickies = append(stickies, b)
		}
		for _, c := range b.children {
			walk(c, b)
		}
	}
	walk(res.root, nil)
	for _, b := range stickies {
		applyOneSticky(res, b, contentH)
	}
}

func applyOneSticky(res *Result, b *box, contentH float64) {
	if !b.hasStickyInset() || b.stickyID == 0 {
		return
	}

	origX, origY := b.x, b.y
	cbTop := b.cbY
	cbBottom := b.cbY + b.cbH
	if b.cbH <= 0 {
		cbBottom = b.cbY + b.h
	}
	firstPage := int(cbTop / contentH)
	if firstPage < 0 {
		firstPage = 0
	}
	lastPage := int((cbBottom - 1e-6) / contentH)
	if lastPage < firstPage {
		lastPage = firstPage
	}

	baseOps := make([]Op, 0, 8)
	for i := range res.Ops {
		if res.Ops[i].StickyID == b.stickyID {
			baseOps = append(baseOps, res.Ops[i])
		}
	}
	if len(baseOps) == 0 {
		return
	}
	baseX, baseY := origX, origY
	// Painted sticky height (box + ink). Continuation flow starts just below
	// the bar (thead-style). Add a modest ascent clearance so the first row's
	// text (baseline Y, ascenders above) is not drawn through the bar —
	// without the old ~20pt pad that left a large white band under the clone.
	reserve := b.h
	maxBot := baseY + b.h
	for _, op := range baseOps {
		bot := op.Y + op.H
		if op.Kind == OpText || op.Kind == OpBullet {
			h := op.H
			if h <= 0 {
				h = 12
			}
			bot = op.Y + h*0.35
		}
		if bot > maxBot {
			maxBot = bot
		}
	}
	if d := maxBot - baseY; d > reserve {
		reserve = d
	}
	// Clear body-line ascenders under the bar so Row 28 is not clipped by the
	// sticky clone (measured ~16–20pt for the fixture font bbox).
	const ascentClearance = 16.0
	const stickyFlowGap = 2.0
	reserve += ascentClearance + stickyFlowGap
	if reserve < 1 {
		reserve = 1
	}

	natPage := int(origY / contentH)
	if natPage < 0 {
		natPage = 0
	}
	pageTop := float64(natPage) * contentH
	x1 := clampStickyX(origX, b.w, b.cbX, b.cbW, 0, res.Width, b)
	y1 := clampStickyY(origY, b.h, b.cbY, b.cbH, pageTop, pageTop+contentH, b)
	dx, dy := x1-origX, y1-origY
	if dx != 0 || dy != 0 {
		shiftStickyOps(res, b.stickyID, dx, dy)
		b.x, b.y = x1, y1
	}

	for page := firstPage; page <= lastPage; page++ {
		if page == natPage {
			continue
		}
		pt := float64(page) * contentH
		pb := pt + contentH
		if pb <= cbTop+1e-9 || pt >= cbBottom-1e-9 {
			continue
		}
		x := clampStickyX(origX, b.w, b.cbX, b.cbW, 0, res.Width, b)
		y := clampStickyY(origY, b.h, b.cbY, b.cbH, pt, pb, b)
		if stickyNear(y, origY) && stickyNear(x, origX) {
			continue
		}
		if y+b.h <= pt+1e-9 || y >= pb-1e-9 {
			continue
		}
		// Reserve vertical space under the sticky clone so flow rows are not
		// painted underneath (thead-repeat style). Without this, continuation
		// pages overlap Row N+1 with the sticky bar (fixture-31 / row 28+).
		if b.stickyTopSet {
			// Push all in-CB flow on this page down from the page top so the
			// sticky band (box top → painted bottom) is clear of row chrome.
			shiftStickyPageFlow(res, b, pt, pb, pt, reserve)
			y = clampStickyY(origY, b.h, b.cbY, b.cbH, pt, pb, b)
		}
		for _, op := range baseOps {
			op.X = op.X - baseX + x
			op.Y = op.Y - baseY + y
			op.Fixed = false
			op.StickyID = 0 // clone is paint-only; not re-processed
			// Keep default z-index so continuation row text can paint over the
			// bar's bottom edge the same way as the natural sticky on page 1
			// (sortPaintIndices already draws fills under text).
			res.Ops = append(res.Ops, op)
		}
	}
}

// shiftStickyPageFlow moves non-sticky ops on [pageTop, pageBottom) down by
// reserve so a sticky clone at stickyY does not cover body content.
//
// Everything on the page below the sticky band is shifted — including
// following siblings outside the containing block (fixture-31 ".after") —
// except tall page-leading background/border rects. Those stay anchored at
// the page top so section chrome continues under the sticky clone the way
// CSS paints a scrolling containing block (avoids a white gap + border
// slicing through the first continuation row).
func shiftStickyPageFlow(res *Result, sticky *box, pageTop, pageBottom, stickyY, reserve float64) {
	if res == nil || sticky == nil || reserve <= 0 {
		return
	}
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Fixed || op.StickyID == sticky.stickyID {
			continue
		}
		if op.Y < pageTop-1e-9 || op.Y >= pageBottom-1e-9 {
			continue
		}
		if op.Y >= stickyY-0.5 {
			if isPageLeadingBackground(op, pageTop, reserve) {
				continue
			}
			op.Y += reserve
		}
	}
}

// isPageLeadingBackground reports tall fill/stroke rects that begin at the
// page top — typically split remnants of a section/containing-block background.
func isPageLeadingBackground(op *Op, pageTop, reserve float64) bool {
	if op == nil {
		return false
	}
	switch op.Kind {
	case OpFillRect, OpStrokeRect:
	default:
		return false
	}
	if op.Y > pageTop+1 {
		return false
	}
	// Row-sized fragments must still move; only large chrome stays put.
	return op.H > reserve+10
}

func shiftStickyOps(res *Result, stickyID int, dx, dy float64) {
	if dx == 0 && dy == 0 || stickyID == 0 {
		return
	}
	for i := range res.Ops {
		if res.Ops[i].StickyID == stickyID {
			res.Ops[i].X += dx
			res.Ops[i].Y += dy
		}
	}
}

// clampStickyY applies top/bottom sticky-view constraints then the containing-
// block limit (CSS Position 3 algorithm lite).
func clampStickyY(naturalY, h, cbY, cbH, pageTop, pageBottom float64, b *box) float64 {
	y := naturalY
	if b.stickyTopSet {
		edge := pageTop + b.stickyTop
		if y < edge {
			y = edge
		}
	}
	if b.stickyBottomSet {
		edge := pageBottom - b.stickyBottom
		if y+h > edge {
			y = edge - h
		}
	}
	if y < cbY {
		y = cbY
	}
	if cbH > 0 && y+h > cbY+cbH {
		y = cbY + cbH - h
		if y < cbY {
			y = cbY
		}
	}
	return y
}

// clampStickyX applies left/right sticky-view constraints then the containing-
// block limit. Page scrollport horizontal edges are [pageLeft, pageRight).
func clampStickyX(naturalX, w, cbX, cbW, pageLeft, pageRight float64, b *box) float64 {
	x := naturalX
	if b.stickyLeftSet {
		edge := pageLeft + b.stickyLeft
		if x < edge {
			x = edge
		}
	}
	if b.stickyRightSet {
		edge := pageRight - b.stickyRight
		if x+w > edge {
			x = edge - w
		}
	}
	if x < cbX {
		x = cbX
	}
	if cbW > 0 && x+w > cbX+cbW {
		x = cbX + cbW - w
		if x < cbX {
			x = cbX
		}
	}
	return x
}

func stickyNear(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.5
}
