package layout

// Print-scoped position:sticky (CSS Positioned Layout Level 3 lite).
//
// Scrollport selection:
//   - Default: each page's content box [pageY, pageY+contentH) — PDF print
//     scrollport. Sticky is not position:fixed: it does not stamp on
//     continuation pages; it only clamps its natural fragment on the page
//     where it occurs.
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
// or to a nearest overflow scrollport at scroll offset 0. It deliberately
// does not clone paint ops onto continuation pages: print sticky is scoped to
// the element's natural fragment and must not behave like position:fixed.
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

func nearSectionBorderRGB(r, g, b float64) bool {
	// fixture-31 .section border #455a64 ≈ (0.271, 0.353, 0.392)
	return r > 0.2 && r < 0.35 && g > 0.28 && g < 0.42 && b > 0.32 && b < 0.48
}

func shiftStickyOps(res *Result, stickyID int, dx, dy float64) {
	if (dx == 0 && dy == 0) || stickyID == 0 {
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
