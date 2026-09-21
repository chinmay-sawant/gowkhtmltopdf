package layout

import (
	"math"
	"testing"
)

// TestChromeFixtureCase32BorderBoxesKeepSpecifiedHeight pins case 32's chrome
// repair: every .container is a 75pt-tall bordered flex box, so its 77pt
// border box must survive stretchPaginatedChrome even when the shrink
// branches' 150pt children overflow it, and even though each container's op
// range swallows the later siblings' border ops. Chromium keeps every border
// box at 77pt and lets the children paint outside.
func TestChromeFixtureCase32BorderBoxesKeepSpecifiedHeight(t *testing.T) {
	t.Parallel()

	res := fixture21To40(t, "case-32-wpt-flex-factor-less-than-one.html")
	containers := []string{"row-grow", "column-grow", "row-grow-basis", "row-shrink", "column-shrink", "vertical-grow"}

	for i, containerID := range containers {
		boxNode := fixtureBox(t, res, containerID)
		if !near(boxNode.y, float64(i)*77) || !near(boxNode.height, 77) {
			t.Fatalf("%s layout rect y=%.2f h=%.2f, want y=%.2f h=77",
				containerID, boxNode.y, boxNode.height, float64(i)*77)
		}
	}

	stretchPaginatedChrome(res)

	for i, containerID := range containers {
		boxNode := fixtureBox(t, res, containerID)
		if !near(boxNode.y, float64(i)*77) || !near(boxNode.height, 77) {
			t.Fatalf("%s after chrome repair y=%.2f h=%.2f, want y=%.2f h=77",
				containerID, boxNode.y, boxNode.height, float64(i)*77)
		}

		assertCase32Frame(t, res, containerID, boxNode)
	}
}

// assertCase32Frame requires the container's own border lines to sit on its
// box rect. Lines are filtered to the box's Y span so a later sibling's top
// border, which lands exactly on this box's bottom border line and shares its
// X/W, cannot leak in through the swallowed op range.
//
//nolint:cyclop // horizontal and vertical edge classification read best together
func assertCase32Frame(t *testing.T, res *Result, label string, boxNode *box) {
	t.Helper()

	boxBottom := boxNode.y + boxNode.height
	topY, bottomY := math.Inf(1), math.Inf(-1)
	leftTop, leftBottom := math.Inf(1), math.Inf(-1)
	rightTop, rightBottom := math.Inf(1), math.Inf(-1)

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(res.Ops); idx++ {
		lineOp := res.Ops[idx]
		if lineOp.Kind != OpLine {
			continue
		}

		if lineOp.H == 0 && lineOp.W > 0 && near(lineOp.X, boxNode.x) && near(lineOp.W, boxNode.w) &&
			(near(lineOp.Y, boxNode.y) || near(lineOp.Y, boxBottom)) {
			topY = math.Min(topY, lineOp.Y)
			bottomY = math.Max(bottomY, lineOp.Y)

			continue
		}

		if lineOp.W != 0 || lineOp.H <= 0 || lineOp.Y < boxNode.y-layoutCoordEpsilon ||
			lineOp.Y+lineOp.H > boxBottom+layoutCoordEpsilon {
			continue
		}

		switch {
		case near(lineOp.X, boxNode.x):
			leftTop = math.Min(leftTop, lineOp.Y)
			leftBottom = math.Max(leftBottom, lineOp.Y+lineOp.H)
		case near(lineOp.X, boxNode.x+boxNode.w):
			rightTop = math.Min(rightTop, lineOp.Y)
			rightBottom = math.Max(rightBottom, lineOp.Y+lineOp.H)
		}
	}

	if !near(topY, boxNode.y) || !near(bottomY, boxBottom) {
		t.Errorf("%s: border rules span y=%.2f..%.2f, want %.2f..%.2f", label, topY, bottomY, boxNode.y, boxBottom)
	}

	if !near(leftTop, boxNode.y) || !near(leftBottom, boxBottom) {
		t.Errorf("%s: left rail spans y=%.2f..%.2f, want %.2f..%.2f", label, leftTop, leftBottom, boxNode.y, boxBottom)
	}

	if !near(rightTop, boxNode.y) || !near(rightBottom, boxBottom) {
		t.Errorf("%s: right rail spans y=%.2f..%.2f, want %.2f..%.2f", label, rightTop, rightBottom, boxNode.y, boxBottom)
	}
}
