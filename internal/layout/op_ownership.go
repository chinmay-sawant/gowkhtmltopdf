package layout

import "math"

// opOwnerPhase selects the position tolerance a pagination pass uses when it
// asks whether an op paints a box's own chrome.
type opOwnerPhase int

const (
	// opOwnerChrome classifies ink during chrome repair.
	opOwnerChrome opOwnerPhase = iota
	// opOwnerClip classifies paint during overflow clipping.
	opOwnerClip
	// opOwnerSeal classifies chrome during sticky-section sealing.
	opOwnerSeal
)

// opOwnerTolerance is the position tolerance each pass accepted before the
// predicates were unified: chrome repair and clipping used layout epsilon,
// the sticky seal used 1pt.
func opOwnerTolerance(phase opOwnerPhase) float64 {
	if phase == opOwnerSeal {
		return 1
	}

	return layoutCoordEpsilon
}

// opOwnedBy reports whether op paints boxNode's own frame chrome rather than
// its content: the border-box rect or a page fragment of it, a straight or
// masked border side, an outline stroke, or a box-shadow layer. Chrome
// repair, overflow clipping, and sticky-section sealing all route through
// this one predicate; before the unification only clipping knew outline
// shapes, so an outline counted as content ink and stretched the box.
func opOwnedBy(oper *Op, boxNode *box, phase opOwnerPhase) bool {
	if oper == nil || boxNode == nil {
		return false
	}

	switch oper.Kind {
	case OpFillRect, OpStrokeRect:
		return opOwnsBoxPaint(oper, boxNode, phase)
	case OpLine:
		if opOwnsBoxSide(oper, boxNode, phase) {
			return true
		}

		return opOwnsBoxOutline(oper, boxNode)
	case OpUnknown, OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
		return false
	}

	return false
}

// opOwnsBoxPaint reports whether a fill or stroke paints the box's own frame
// chrome: its rect, a shadow layer, a masked side, or the outline.
func opOwnsBoxPaint(oper *Op, boxNode *box, phase opOwnerPhase) bool {
	if opOwnsBoxRect(oper, boxNode, phase) || opOwnsBoxShadow(oper, boxNode, phase) {
		return true
	}

	if oper.Kind == OpStrokeRect && oper.StrokeMask != 0 {
		return opOwnsBoxSide(oper, boxNode, phase) || opOwnsBoxOutline(oper, boxNode)
	}

	return opOwnsBoxOutline(oper, boxNode)
}

// opOwnsBoxRect reports a fill/stroke of the box frame. Chrome repair also
// accepts page-split fragments; the seal accepts page fragments whose Y sits
// inside the box; overflow clipping keeps the exact-frame rule.
func opOwnsBoxRect(oper *Op, boxNode *box, phase opOwnerPhase) bool {
	if oper.Kind != OpFillRect && oper.Kind != OpStrokeRect {
		return false
	}

	tol := opOwnerTolerance(phase)
	if math.Abs(oper.X-boxNode.x) > tol || math.Abs(oper.W-boxNode.w) > tol {
		return false
	}

	exact := math.Abs(oper.Y-boxNode.y) <= tol && math.Abs(oper.H-boxNode.height) <= tol

	switch phase {
	case opOwnerClip:
		return exact
	case opOwnerSeal:
		// The section box height can be stale by the time a page-leading
		// fragment is repaired; the caller matches X/W and color, so a shape
		// with the right frame is owned regardless of Y.
		return true
	case opOwnerChrome:
		return opOwnsChromeRectFragment(oper, boxNode, exact, tol)
	}

	return false
}

// opOwnsChromeRectFragment reports whether a fill or stroke of the box frame
// stays inside the box vertically: chrome repair also accepts page-split
// fragments that start above the box but end within it.
func opOwnsChromeRectFragment(oper *Op, boxNode *box, exact bool, tol float64) bool {
	if exact {
		return true
	}

	return oper.Y >= boxNode.y-tol && oper.Y+oper.H <= boxNode.y+boxNode.height+1
}

// opOwnsBoxSide reports a straight or masked border side. Chrome repair keeps
// its border-existing and dash-shape checks; clipping and sealing match edge
// proximity and overlap.
func opOwnsBoxSide(oper *Op, boxNode *box, phase opOwnerPhase) bool {
	tol := opOwnerTolerance(phase)
	if opIsVerticalSide(oper) {
		return opOwnsVerticalSide(oper, boxNode, phase, tol)
	}

	return opOwnsHorizontalSide(oper, boxNode, phase, tol)
}

// opIsVerticalSide reports whether oper draws a vertical rail: a zero-width
// line or a masked left or right stroke.
func opIsVerticalSide(oper *Op) bool {
	if oper.Kind == OpLine && oper.W == 0 {
		return true
	}

	return oper.Kind == OpStrokeRect &&
		(oper.StrokeMask == StrokeMaskLeft || oper.StrokeMask == StrokeMaskRight)
}

// opOwnsVerticalSide reports a vertical rail on the box's left or right edge.
func opOwnsVerticalSide(oper *Op, boxNode *box, phase opOwnerPhase, tol float64) bool {
	if oper.H <= 0 {
		return false
	}

	onLeft := math.Abs(oper.X-boxNode.x) <= tol
	onRight := math.Abs(oper.X-(boxNode.x+boxNode.w)) <= tol

	if !onLeft && !onRight {
		return false
	}

	// The seal matches rails by edge and color; its target box height can be
	// stale after pagination, so no vertical overlap check.
	if phase == opOwnerSeal {
		return true
	}

	if oper.Y > boxNode.y+boxNode.height+tol || oper.Y+oper.H < boxNode.y-tol {
		return false
	}

	if phase != opOwnerChrome {
		return true
	}

	return chromeVerticalBorderExists(oper, boxNode, onLeft, onRight)
}

// opOwnsHorizontalSide reports a horizontal rail on the box's top or bottom
// edge.
func opOwnsHorizontalSide(oper *Op, boxNode *box, phase opOwnerPhase, tol float64) bool {
	if oper.Kind != OpLine || oper.H != 0 || oper.W <= 0 {
		return false
	}

	onTop := math.Abs(oper.Y-boxNode.y) <= tol
	onBottom := math.Abs(oper.Y-(boxNode.y+boxNode.height)) <= tol

	if !onTop && !onBottom {
		return false
	}

	if oper.X+oper.W < boxNode.x-tol || oper.X > boxNode.x+boxNode.w+tol {
		return false
	}

	if phase != opOwnerChrome {
		return true
	}

	return opOwnsChromeHorizontalSide(oper, boxNode, tol, onTop, onBottom)
}

// opOwnsChromeHorizontalSide reports whether a horizontal rail matches the
// chrome-repair shape checks for the box's top or bottom border.
func opOwnsChromeHorizontalSide(oper *Op, boxNode *box, tol float64, onTop, onBottom bool) bool {
	style, ok := paintChromeStyleOf(boxNode)
	if !ok {
		return false
	}

	if math.Abs(oper.W-boxNode.w) <= tol {
		return true
	}

	if onTop && isDashedOrDottedStyle(style.borderTop.Style) {
		return true
	}

	if onBottom && isDashedOrDottedStyle(style.borderBottom.Style) {
		return true
	}

	return looksLikeDashSegmentLength(oper.W, oper.Width)
}

// chromeVerticalBorderExists reports the box side a vertical rail can belong
// to for the chrome-repair pass.
func chromeVerticalBorderExists(oper *Op, boxNode *box, onLeft, onRight bool) bool {
	if oper.H > boxNode.height+1e-6 {
		return false
	}

	style, ok := paintChromeStyleOf(boxNode)
	if !ok {
		return false
	}

	if onLeft && !(style.borderLeft.Width > 0 && style.borderLeft.Style != cssDisplayNone) {
		return false
	}

	if onRight && !(style.borderRight.Width > 0 && style.borderRight.Style != cssDisplayNone) {
		return false
	}

	return true
}

// opOwnsBoxOutline reports an op that traces the outline frame around boxNode.
// The scaled inflate is stamped on the box where outline ops are emitted, so
// paint-time checks need no engine scale.
func opOwnsBoxOutline(oper *Op, boxNode *box) bool {
	inflate := boxNode.outlineInflate
	if inflate <= 0 {
		return false
	}

	if oper.Kind == OpStrokeRect && oper.StrokeMask == 0 {
		return withinTol(oper.X, boxNode.x-inflate) &&
			withinTol(oper.Y, boxNode.y-inflate) &&
			withinTol(oper.W, boxNode.w+2*inflate) &&
			withinTol(oper.H, boxNode.height+2*inflate)
	}

	// Dashed/dotted outlines are many short segments: use edge overlap, not a
	// full-frame match.
	return lineOnRectEdges(
		oper,
		boxNode.x-inflate, boxNode.y-inflate,
		boxNode.w+2*inflate, boxNode.height+2*inflate,
	)
}

// withinTol reports whether two layout coordinates agree within epsilon.
func withinTol(a, b float64) bool {
	return math.Abs(a-b) <= layoutCoordEpsilon
}

// opOwnsBoxShadow reports a box-shadow layer: a fill shaped like the border
// box grown uniformly on both axes (spread and blur only grow it; offsets
// translate it). Gated on the box's own shadow so an unrelated translated
// fill of the same size cannot match.
func opOwnsBoxShadow(oper *Op, boxNode *box, phase opOwnerPhase) bool {
	if oper.Kind != OpFillRect || boxNode.style == nil || !boxNode.style.BoxShadowSet {
		return false
	}

	tol := opOwnerTolerance(phase)

	deltaW := oper.W - boxNode.w
	deltaH := oper.H - boxNode.height

	if deltaW < -tol || deltaH < -tol || math.Abs(deltaW-deltaH) > tol {
		return false
	}

	exactFrame := math.Abs(deltaW) <= tol && math.Abs(deltaH) <= tol &&
		math.Abs(oper.X-boxNode.x) <= tol && math.Abs(oper.Y-boxNode.y) <= tol

	return !exactFrame
}
