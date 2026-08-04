package layout

// floatState tracks left/right floats inside one block formatting context
// (one flowChildren call). Coordinates are canvas-absolute for bottoms and
// edges; contentX/contentW define the containing block.
//
// Lite model (invoice chrome): floats leave normal flow, pack to a side at
// the current flow y (stacking vertically when multiple floats share a side),
// and in-flow content uses a simple side exclusion until clear or past the
// float bottoms. Not a full CSS2 float engine.
type floatState struct {
	leftBottom  float64 // canvas y of lowest left-float bottom; 0 = none
	rightBottom float64
	leftEdge    float64 // absolute x of right edge of active left float
	rightEdge   float64 // absolute x of left edge of active right float
	contentX    float64
	contentW    float64
	hasLeft     bool
	hasRight    bool
}

func newFloatState(contentX, contentW float64) floatState {
	return floatState{
		contentX:  contentX,
		contentW:  contentW,
		leftEdge:  contentX,
		rightEdge: contentX + contentW,
	}
}

// clear advances cy (content-relative) so the next in-flow box sits below
// the floats named by clear ("left"|"right"|"both").
func (f *floatState) clear(clear string, y, cy float64) float64 {
	need := y + cy
	switch clear {
	case "left":
		if f.hasLeft && f.leftBottom > need {
			need = f.leftBottom
		}
		f.hasLeft = false
		f.leftBottom = 0
		f.leftEdge = f.contentX
	case "right":
		if f.hasRight && f.rightBottom > need {
			need = f.rightBottom
		}
		f.hasRight = false
		f.rightBottom = 0
		f.rightEdge = f.contentX + f.contentW
	case "both":
		if f.hasLeft && f.leftBottom > need {
			need = f.leftBottom
		}
		if f.hasRight && f.rightBottom > need {
			need = f.rightBottom
		}
		f.hasLeft, f.hasRight = false, false
		f.leftBottom, f.rightBottom = 0, 0
		f.leftEdge = f.contentX
		f.rightEdge = f.contentX + f.contentW
	}
	if need > y+cy {
		return need - y
	}
	return cy
}

// place records a laid-out float box on the left or right side.
func (f *floatState) place(side string, b *box) {
	bottom := b.y + b.h
	switch side {
	case "left":
		if !f.hasLeft || bottom > f.leftBottom {
			f.leftBottom = bottom
		}
		edge := b.x + b.w
		if !f.hasLeft || edge > f.leftEdge {
			f.leftEdge = edge
		}
		f.hasLeft = true
	case "right":
		if !f.hasRight || bottom > f.rightBottom {
			f.rightBottom = bottom
		}
		if !f.hasRight || b.x < f.rightEdge {
			f.rightEdge = b.x
		}
		f.hasRight = true
	}
}

// exclusion returns the in-flow content origin and width at canvas y = y+cy
// after subtracting active float intrusion.
func (f *floatState) exclusion(y, cy float64) (x, w float64) {
	x, w = f.contentX, f.contentW
	top := y + cy
	if f.hasLeft && f.leftBottom > top {
		if f.leftEdge > x {
			w -= f.leftEdge - x
			x = f.leftEdge
		}
	}
	if f.hasRight && f.rightBottom > top {
		limit := f.rightEdge
		if limit < x+w {
			w = limit - x
		}
	}
	if w < 0 {
		w = 0
	}
	return x, w
}

// extentCy returns cy raised to cover any float that still protrudes below
// the in-flow content end (so the parent border box encloses floats).
func (f *floatState) extentCy(y, cy float64) float64 {
	end := y + cy
	if f.hasLeft && f.leftBottom > end {
		end = f.leftBottom
	}
	if f.hasRight && f.rightBottom > end {
		end = f.rightBottom
	}
	return end - y
}
