package layout

// Float/clear and overflow keyword constants (goconst) kept next to the
// float engine that uses them.
const (
	floatLeft      = "left"
	floatRight     = "right"
	clearBoth      = "both"
	overflowHidden = "hidden"
	overflowScroll = "scroll"
	overflowAuto   = "auto"
	overflowClip   = "clip"
)

// floatState tracks left/right floats inside one block formatting context.
// Coordinates are canvas-absolute for bottoms and edges; contentX/contentW
// define the BFC's content box (exclusion starts from those edges).
//
// Lite model: floats leave normal flow, pack to a side at the current flow y
// (stacking vertically when multiple floats share a side), and in-flow
// content uses a simple side exclusion until clear or past the float bottoms.
// Not a full CSS2 float engine.
//
// BFC policy: only formatting-context roots enclose floats in their height
// (extentCy). Ordinary blocks share the parent BFC's floatState so a float
// inside <section> still affects following sibling sections — matching CSS
// (overflow:visible parents do not grow around floats).
//
// Table policy (tier-2-pending-3): in-flow display:table boxes always clear
// below floats in flowChildren — no shrink-beside. Floated tables keep
// display:table (CSS2.1 §9.7) and participate via placeFloat (fixture-29).
type floatState struct {
	leftBottom  float64 // canvas y of lowest left-float bottom; 0 = none
	rightBottom float64
	leftTop     float64 // canvas y of the top of the current left pack row
	rightTop    float64
	leftEdge    float64 // absolute x of right edge of active left float
	rightEdge   float64 // absolute x of left edge of active right float
	contentX    float64
	contentW    float64
	hasLeft     bool
	hasRight    bool
}

func newFloatState(contentX, contentW float64) floatState {
	return floatState{ //nolint:exhaustruct // intentional zero fields
		contentX:  contentX,
		contentW:  contentW,
		leftEdge:  contentX,
		rightEdge: contentX + contentW,
	}
}

// clearFloats advances cy (content-relative) so the next in-flow box sits
// below the floats named by clear ("left"|"right"|"both").
func (f *floatState) clearFloats(clearVal string, posY, curY float64) float64 {
	need := posY + curY

	switch clearVal {
	case floatLeft:
		need = f.clearLeft(need)
	case floatRight:
		need = f.clearRight(need)
	case clearBoth:
		need = f.clearLeft(need)
		need = f.clearRight(need)
	}

	if need > posY+curY {
		return need - posY
	}

	return curY
}

// clearLeft drops the left float and raises need past its bottom.
func (f *floatState) clearLeft(need float64) float64 {
	if f.hasLeft && f.leftBottom > need {
		need = f.leftBottom
	}

	f.hasLeft = false
	f.leftBottom = 0
	f.leftTop = 0
	f.leftEdge = f.contentX

	return need
}

// clearRight drops the right float and raises need past its bottom.
func (f *floatState) clearRight(need float64) float64 {
	if f.hasRight && f.rightBottom > need {
		need = f.rightBottom
	}

	f.hasRight = false
	f.rightBottom = 0
	f.rightTop = 0
	f.rightEdge = f.contentX + f.contentW

	return need
}

// place records a laid-out float box on the left or right side.
// ml/mr are the floated box's horizontal margins (scaled pt); exclusion uses
// the margin box so in-flow text clears the gap before the border (e.g.
// float:right; margin-left:1em), instead of painting flush against the frame.
func (f *floatState) place(side string, fbox *box, margL, margR float64) {
	bottom := fbox.y + fbox.height

	switch side {
	case floatLeft:
		f.placeLeft(fbox, bottom, margR)
	case floatRight:
		f.placeRight(fbox, bottom, margL)
	}
}

// placeLeft records a left float's bottom/top and the edge of its margin box.
func (f *floatState) placeLeft(fbox *box, bottom, margR float64) {
	if !f.hasLeft || bottom > f.leftBottom {
		f.leftBottom = bottom
	}

	if !f.hasLeft || fbox.y < f.leftTop {
		f.leftTop = fbox.y
	}

	edge := fbox.x + fbox.w + margR
	if !f.hasLeft || edge > f.leftEdge {
		f.leftEdge = edge
	}

	f.hasLeft = true
}

// placeRight records a right float's bottom/top and the edge of its margin box.
func (f *floatState) placeRight(fbox *box, bottom, margL float64) {
	if !f.hasRight || bottom > f.rightBottom {
		f.rightBottom = bottom
	}

	if !f.hasRight || fbox.y < f.rightTop {
		f.rightTop = fbox.y
	}

	edge := fbox.x - margL
	if !f.hasRight || edge < f.rightEdge {
		f.rightEdge = edge
	}

	f.hasRight = true
}

// exclusion returns the in-flow content origin and width at canvas y = y+cy
// after subtracting active float intrusion from the caller's content box
// (contentX/contentW). Float edges are canvas-absolute.
func (f *floatState) exclusion(contentX, contentW, y, cy float64) (float64, float64) {
	outX, outW := contentX, contentW
	top := y + cy

	if f.hasLeft && f.leftBottom > top {
		if f.leftEdge > outX {
			outW -= f.leftEdge - outX
			outX = f.leftEdge
		}
	}

	if f.hasRight && f.rightBottom > top {
		limit := f.rightEdge
		if limit < outX+outW {
			outW = limit - outX
		}
	}

	if outW < 0 {
		outW = 0
	}

	return outX, outW
}

// clearY returns the canvas Y just past any float that still shortens the
// line at top (so a line box that cannot fit can restart at full width).
func (f *floatState) clearY(top float64) float64 {
	next := top
	if f.hasLeft && f.leftBottom > next {
		next = f.leftBottom
	}

	if f.hasRight && f.rightBottom > next {
		next = f.rightBottom
	}

	return next
}

// extentCy returns cy raised to cover any float that still protrudes below
// the in-flow content end. Only BFC roots should apply this (CSS2.1 §10.6.7);
// ordinary blocks leave floats protruding so later siblings can wrap.
func (f *floatState) extentCy(posY, cy float64) float64 {
	end := posY + cy
	if f.hasLeft && f.leftBottom > end {
		end = f.leftBottom
	}

	if f.hasRight && f.rightBottom > end {
		end = f.rightBottom
	}

	return end - posY
}

// establishesBFC reports whether st creates a new block formatting context
// that traps floats (CSS2.1 / Display 3). Descendants' floats do not affect
// the parent BFC; the box's used height encloses its floats.
func establishesBFC(sty ResolvedStyle) bool {
	if sty.Float != cssDisplayNone {
		return true
	}

	switch sty.Display {
	case displayFlowRoot, cssDisplayInlineBlock, displayTableCell, displayTableCaption,
		displayFlex, displayInlineFlex, displayGrid, displayInlineGrid:
		return true
	}

	switch sty.Overflow {
	case overflowHidden, overflowScroll, overflowAuto, overflowClip:
		return true
	}

	return false
}
