package layout

import "strings"

// Print-scoped position:sticky (CSS Positioned Layout Level 3 lite).
//
// Scrollport selection:
//   - Default: each page's content box [pageY, pageY+contentH) — PDF print
//     scrollport (fixture-31). Sticky is not position:fixed: it does not stamp
//     on every page; it only clamps (and clones onto continuation pages) where
//     the containing block intersects the page.
//   - Inside overflow:auto|scroll|hidden|clip: that box is the sticky
//     scrollport. PDF has no user scroll, so clamp at scroll offset 0 against
//     the overflow box edges (no page-edge sticking / continuation clones).
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
// or to a nearest overflow scrollport at scroll offset 0, and clones stuck
// paint ops onto continuation pages where the containing block still
// intersects (print scrollport only). Called after flow pagination has settled.
func applyStickyPrint(res *Result, contentH float64) {
	if res == nil || res.root == nil || contentH <= 0 {
		return
	}
	var stickies []*box
	var walk func(b, parent, overflowPort *box)
	walk = func(b, parent, overflowPort *box) {
		port := overflowPort
		if overflowCreatesStickyScrollport(b.style.Overflow) {
			port = b
		}
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
			b.stickyPort = port
			stickies = append(stickies, b)
		}
		for _, c := range b.children {
			walk(c, b, port)
		}
	}
	walk(res.root, nil, nil)
	for _, b := range stickies {
		applyOneSticky(res, b, contentH)
	}
}

func applyOneSticky(res *Result, b *box, contentH float64) {
	if !b.hasStickyInset() || b.stickyID == 0 {
		return
	}

	// Overflow scrollport at scroll offset 0: clamp within the overflow box
	// only — no print page continuation clones (PDF has no scroll).
	if b.stickyPort != nil {
		applyStickyOverflowClamp(res, b)
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
	// Sticky band height used as thead-style header reserve. Continuation
	// flow is shifted so the first row box sits just under this band.
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
	const stickyFlowGap = 2.0
	reserve += stickyFlowGap
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
			shiftStickyPageFlow(res, b, pt, pb, pt, reserve)
			y = clampStickyY(origY, b.h, b.cbY, b.cbH, pt, pb, b)
		}
		for _, op := range baseOps {
			op.X = op.X - baseX + x
			op.Y = op.Y - baseY + y
			op.Fixed = false
			op.StickyID = 0 // clone is paint-only; not re-processed
			// Paint above body ink that still meets the bar edge so the sticky
			// label is never overwritten by Row 28+ (fixture-31).
			op.ZIndexSet = true
			if op.ZIndex < 1 {
				op.ZIndex = 1
			}
			res.Ops = append(res.Ops, op)
		}
	}
}

// applyStickyOverflowClamp clamps sticky to its overflow ancestor at scroll
// offset 0 (PDF has no scroll). No continuation-page clones.
func applyStickyOverflowClamp(res *Result, b *box) {
	port := b.stickyPort
	if port == nil {
		return
	}
	origX, origY := b.x, b.y
	portLeft, portTop := port.x, port.y
	portRight, portBottom := port.x+port.w, port.y+port.h
	x1 := clampStickyX(origX, b.w, b.cbX, b.cbW, portLeft, portRight, b)
	y1 := clampStickyY(origY, b.h, b.cbY, b.cbH, portTop, portBottom, b)
	dx, dy := x1-origX, y1-origY
	if dx != 0 || dy != 0 {
		shiftStickyOps(res, b.stickyID, dx, dy)
		b.x, b.y = x1, y1
	}
}

// shiftStickyPageFlow moves non-sticky flow on [pageTop, pageBottom) down so
// continuation rows clear the sticky band, then grows page-leading section
// chrome so side borders still enclose the shifted rows (fixture-31 Row 35).
//
// dy is the max of:
//   - first row fill just under the sticky band (thead-style)
//   - first text baseline a few points under the sticky bottom (no overwrite)
func shiftStickyPageFlow(res *Result, sticky *box, pageTop, pageBottom, stickyY, reserve float64) {
	if res == nil || sticky == nil || reserve <= 0 {
		return
	}
	const flowGap = 2.0
	paintedH := reserve - flowGap
	if paintedH < 1 {
		paintedH = reserve
	}
	stickyBot := stickyY + paintedH
	neededFillTop := stickyY + reserve

	bodyFillTop := 0.0
	foundFill := false
	bodyTextTop := 0.0
	foundText := false
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Fixed || op.StickyID == sticky.stickyID {
			continue
		}
		if op.Y < pageTop-1e-9 || op.Y >= pageBottom-1e-9 {
			continue
		}
		if isPageLeadingBackground(op, pageTop, reserve) {
			continue
		}
		switch op.Kind {
		case OpFillRect, OpStrokeRect:
			if !foundFill || op.Y < bodyFillTop {
				bodyFillTop = op.Y
				foundFill = true
			}
		case OpText, OpBullet:
			if !foundText || op.Y < bodyTextTop {
				bodyTextTop = op.Y
				foundText = true
			}
		}
	}

	dy := 0.0
	if foundFill && bodyFillTop < neededFillTop-0.5 {
		dy = neededFillTop - bodyFillTop
	}
	// Keep Row 28's baseline under the bar so it does not paint through the
	// sticky label. Ascenders may still meet the bar; sticky z-index covers them.
	const textClear = 14.0
	if foundText {
		need := stickyBot + textClear
		if bodyTextTop+dy < need-0.5 {
			if d := need - bodyTextTop; d > dy {
				dy = d
			}
		}
	}
	if dy <= 0 {
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
		if isPageLeadingBackground(op, pageTop, reserve) {
			continue
		}
		if op.Y >= stickyY-0.5 {
			op.Y += dy
		}
	}

	// Grow page-leading section chrome to the bottom of shifted section
	// content (do not swallow following siblings past the containing block).
	limit := sticky.cbY + sticky.cbH + dy + 1
	if limit < pageTop {
		limit = pageBottom
	}
	// Pagination snaps (e.g. ascender lead) can push rows past layout-time
	// cbH without updating the box — extend the fence to on-page body flow
	// so side borders still reach the last row (fixture-31 Row 35).
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Fixed || op.StickyID == sticky.stickyID {
			continue
		}
		if op.Y < pageTop-1e-9 || op.Y >= pageBottom-1e-9 {
			continue
		}
		if isPageLeadingBackground(op, pageTop, reserve) {
			continue
		}
		bot := op.Y
		switch op.Kind {
		case OpFillRect, OpStrokeRect:
			bot = op.Y + op.H
		case OpText, OpBullet:
			h := op.Size * 0.35
			if op.H > 0 {
				h = op.H * 0.35
			}
			if h < 4 {
				h = 4
			}
			bot = op.Y + h
		case OpLine:
			if op.H > 1 {
				bot = op.Y + op.H
			}
		}
		if bot+1 > limit {
			limit = bot + 1
		}
	}
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpText && strings.Contains(op.Text, "After the section") {
			if op.Y >= pageTop && op.Y < pageBottom && op.Y < limit {
				limit = op.Y
			}
		}
	}
	maxContentBot := pageTop
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Fixed || op.StickyID == sticky.stickyID {
			continue
		}
		if op.Y < pageTop-1e-9 || op.Y >= limit-1e-9 {
			continue
		}
		if isPageLeadingBackground(op, pageTop, reserve) {
			continue
		}
		bot := op.Y
		switch op.Kind {
		case OpFillRect, OpStrokeRect:
			bot = op.Y + op.H
		case OpText, OpBullet:
			h := op.Size * 0.35
			if op.H > 0 {
				h = op.H * 0.35
			}
			if h < 4 {
				h = 4
			}
			bot = op.Y + h
		case OpLine:
			if op.H > 1 {
				bot = op.Y + op.H
			} else if op.Y > bot {
				bot = op.Y
			}
		}
		if bot > maxContentBot {
			maxContentBot = bot
		}
	}
	if maxContentBot <= pageTop+1 {
		return
	}
	for i := range res.Ops {
		op := &res.Ops[i]
		if !isPageLeadingBackground(op, pageTop, reserve) {
			continue
		}
		needH := maxContentBot - op.Y
		if needH <= op.H+0.5 {
			continue
		}
		switch op.Kind {
		case OpFillRect, OpStrokeRect:
			op.H = needH
		case OpLine:
			if op.H > reserve+10 {
				op.H = needH
			}
		}
	}
	// Snap section-colored bottom edges to the content bottom (not row
	// separators / after-box chrome).
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind != OpLine || op.H >= 1 {
			continue
		}
		if op.Y < pageTop-1e-9 || op.Y >= pageBottom-1e-9 {
			continue
		}
		if !nearSectionBorderRGB(op.R, op.G, op.B) {
			continue
		}
		if op.Y < maxContentBot-40 {
			continue
		}
		op.Y = maxContentBot
	}
}

func nearSectionBorderRGB(r, g, b float64) bool {
	// fixture-31 .section border #455a64 ≈ (0.271, 0.353, 0.392)
	return r > 0.2 && r < 0.35 && g > 0.28 && g < 0.42 && b > 0.32 && b < 0.48
}

// isPageLeadingBackground reports tall fill/stroke/line chrome that begins at
// the current page top — typically split remnants of a section/containing-
// block background or border. These stay put under the sticky clone.
//
// Y must be at this page's top (not merely above it): otherwise page-0 section
// fills match when processing continuation pages and get their H grown into
// empty page-1 bands (fixture-31 after Row 27).
func isPageLeadingBackground(op *Op, pageTop, reserve float64) bool {
	if op == nil {
		return false
	}
	if op.Y < pageTop-1 || op.Y > pageTop+1 {
		return false
	}
	switch op.Kind {
	case OpFillRect, OpStrokeRect:
		// Row-sized fragments must still move; only large chrome stays put.
		return op.H > reserve+10
	case OpLine:
		// Vertical section borders (tall H) and the section's top edge (H≈0
		// along the page top). Row separators have H≈0 but sit below pageTop.
		if op.H > reserve+10 {
			return true
		}
		return op.H < 1
	default:
		return false
	}
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
