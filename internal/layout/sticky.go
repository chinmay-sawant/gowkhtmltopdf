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

	b.stickyTopSet, b.stickyTop = stickyInset(e, b.style.TopAuto, b.style.Top)
	b.stickyRightSet, b.stickyRight = stickyInset(e, b.style.RightAuto, b.style.Right)
	b.stickyBottomSet, b.stickyBottom = stickyInset(e, b.style.BottomAuto, b.style.Bottom)
	b.stickyLeftSet, b.stickyLeft = stickyInset(e, b.style.LeftAuto, b.style.Left)

	if b.opEnd >= b.opStart && b.opStart >= 0 {
		for i := b.opStart; i <= b.opEnd && i < len(e.ops); i++ {
			e.ops[i].StickyID = b.stickyID
		}
	}
}

// stickyInset resolves a non-auto sticky inset; auto insets stay unset (0).
func stickyInset(e *engine, auto bool, v float64) (set bool, inset float64) {
	if auto {
		return false, 0
	}

	return true, e.scalePt(v)
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
	walk = func(boxNode, parent, overflowPort *box) {
		port := overflowPort
		if overflowCreatesStickyScrollport(boxNode.style.Overflow) {
			port = boxNode
		}

		if boxNode.sticky {
			if parent != nil {
				boxNode.cbX, boxNode.cbY, boxNode.cbW, boxNode.cbH = parent.x, parent.y, parent.w, parent.height
			} else {
				h := res.Height
				if h < boxNode.y+boxNode.height {
					h = boxNode.y + boxNode.height
				}

				boxNode.cbX, boxNode.cbY, boxNode.cbW, boxNode.cbH = 0, 0, res.Width, h
			}

			boxNode.stickyPort = port
			stickies = append(stickies, boxNode)
		}

		for _, c := range boxNode.children {
			walk(c, boxNode, port)
		}
	}
	walk(res.root, nil, nil)

	for _, b := range stickies {
		applyOneSticky(res, b, contentH)
	}
}

func applyOneSticky(res *Result, boxNode *box, contentH float64) {
	if !boxNode.hasStickyInset() || boxNode.stickyID == 0 {
		return
	}

	// Overflow scrollport at scroll offset 0: clamp within the overflow box
	// only — no print page continuation clones (PDF has no scroll).
	if boxNode.stickyPort != nil {
		applyStickyOverflowClamp(res, boxNode)

		return
	}

	origX, origY := boxNode.x, boxNode.y

	natPage := int(origY / contentH)
	if natPage < 0 {
		natPage = 0
	}

	pageTop := float64(natPage) * contentH
	xEnd := clampStickyX(origX, boxNode.w, boxNode.cbX, boxNode.cbW, 0, res.Width, boxNode)
	y1 := clampStickyY(origY, boxNode.height, boxNode.cbY, boxNode.cbH, pageTop, pageTop+contentH, boxNode)

	dx, dy := xEnd-origX, y1-origY
	if dx != 0 || dy != 0 {
		shiftStickyOps(res, boxNode.stickyID, dx, dy)
		boxNode.x, boxNode.y = xEnd, y1
	}
}

// applyStickyOverflowClamp clamps sticky to its overflow ancestor at scroll
// offset 0 (PDF has no scroll). No continuation-page clones.
func applyStickyOverflowClamp(res *Result, boxNode *box) {
	port := boxNode.stickyPort
	if port == nil {
		return
	}

	origX, origY := boxNode.x, boxNode.y
	portLeft, portTop := port.x, port.y
	portRight, portBottom := port.x+port.w, port.y+port.height
	xEnd := clampStickyX(origX, boxNode.w, boxNode.cbX, boxNode.cbW, portLeft, portRight, boxNode)
	y1 := clampStickyY(origY, boxNode.height, boxNode.cbY, boxNode.cbH, portTop, portBottom, boxNode)

	dx, dy := xEnd-origX, y1-origY
	if dx != 0 || dy != 0 {
		shiftStickyOps(res, boxNode.stickyID, dx, dy)
		boxNode.x, boxNode.y = xEnd, y1
	}
}

func nearSectionBorderRGB(r, g, b float64) bool {
	// fixture-31 .section border #455a64 ≈ (0.271, 0.353, 0.392)
	return r > 0.2 && r < 0.35 && g > 0.28 && g < 0.42 && b > 0.32 && b < 0.48
}

func shiftStickyOps(res *Result, stickyID int, dxVal, deltaY float64) {
	if (dxVal == 0 && deltaY == 0) || stickyID == 0 {
		return
	}

	for i := range res.Ops {
		if res.Ops[i].StickyID == stickyID {
			res.Ops[i].X += dxVal
			res.Ops[i].Y += deltaY
		}
	}
}

// clampStickyY applies top/bottom sticky-view constraints then the containing-
// block limit (CSS Position 3 algorithm lite).
func clampStickyY(naturalY, htX, cbY, cbH, pageTop, pageBottom float64, boxNode *box) float64 {
	posY := naturalY

	if boxNode.stickyTopSet {
		edge := pageTop + boxNode.stickyTop
		if posY < edge {
			posY = edge
		}
	}

	if boxNode.stickyBottomSet {
		edge := pageBottom - boxNode.stickyBottom
		if posY+htX > edge {
			posY = edge - htX
		}
	}

	if posY < cbY {
		posY = cbY
	}

	if cbH > 0 && posY+htX > cbY+cbH {
		posY = cbY + cbH - htX
		if posY < cbY {
			posY = cbY
		}
	}

	return posY
}

// clampStickyX applies left/right sticky-view constraints then the containing-
// block limit. Page scrollport horizontal edges are [pageLeft, pageRight).
func clampStickyX(naturalX, width, cbX, cbW, pageLeft, pageRight float64, boxN *box) float64 {
	posX := naturalX

	if boxN.stickyLeftSet {
		edge := pageLeft + boxN.stickyLeft
		if posX < edge {
			posX = edge
		}
	}

	if boxN.stickyRightSet {
		edge := pageRight - boxN.stickyRight
		if posX+width > edge {
			posX = edge - width
		}
	}

	if posX < cbX {
		posX = cbX
	}

	if cbW > 0 && posX+width > cbX+cbW {
		posX = cbX + cbW - width
		if posX < cbX {
			posX = cbX
		}
	}

	return posX
}
