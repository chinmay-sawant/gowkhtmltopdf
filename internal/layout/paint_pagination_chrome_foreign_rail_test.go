package layout

import (
	"testing"
)

// TestChromeFixtureCase33ForeignRailsDoNotStretchRows pins the five closed
// frames of the transformed flex rows. translateY(56pt) splices each row's
// border chrome into the display list immediately while deferring its
// children's fills, so unionChildOpRanges widens every row's op range across
// its later siblings' rails and rules. Chrome repair must not read those
// foreign ops as this row's content ink (which stretched every row down to the
// last row's bottom) or let frame realignment swap a row's top and bottom
// rules.
func TestChromeFixtureCase33ForeignRailsDoNotStretchRows(t *testing.T) {
	t.Parallel()

	res := fixture21To40(t, "case-33-wpt-flex-item-compressible.html")

	const rowHeight = 32

	stretchPaginatedChrome(res)

	for _, caseID := range []string{"case-33-text", "case-33-range", "case-33-button", "case-33-calc", "case-33-wide"} {
		caseBox := fixtureBox(t, res, caseID)

		if !near(caseBox.height, rowHeight) {
			t.Errorf("%s height after stretch = %.2f, want %d", caseID, caseBox.height, rowHeight)
		}

		if !caseFrameRuleAt(res, caseBox, caseBox.y) {
			t.Errorf("%s: no top border rule at y=%.2f", caseID, caseBox.y)
		}

		if !caseFrameRuleAt(res, caseBox, caseBox.y+rowHeight) {
			t.Errorf("%s: no bottom border rule at y=%.2f", caseID, caseBox.y+rowHeight)
		}

		for _, side := range []struct {
			name string
			x    float64
		}{
			{"left", caseBox.x},
			{"right", caseBox.x + caseBox.w},
		} {
			rail := caseRailAt(res, caseBox, side.x)
			if rail == nil {
				t.Errorf("%s: no %s rail anchored at y=%.2f", caseID, side.name, caseBox.y)

				continue
			}

			if !near(rail.Y+rail.H, caseBox.y+rowHeight) {
				t.Errorf("%s: %s rail spans y=%.2f..%.2f, want %.2f..%.2f",
					caseID, side.name, rail.Y, rail.Y+rail.H, caseBox.y, caseBox.y+rowHeight)
			}
		}
	}
}

// caseFrameRuleAt reports a full-width horizontal rule of the box at y, the
// shape every box frame rule shares.
func caseFrameRuleAt(res *Result, caseBox *box, y float64) bool {
	for idx := caseBox.opStart; idx <= caseBox.opEnd && idx < len(res.Ops); idx++ {
		operation := &res.Ops[idx]
		if operation.Kind == OpLine && operation.H == 0 && operation.W > 0 &&
			near(operation.X, caseBox.x) && near(operation.W, caseBox.w) && near(operation.Y, y) {
			return true
		}
	}

	return false
}

// caseRailAt returns the side rail on the box's own edge x whose top sits on
// the box's own top edge. Later siblings' rails share the edge x but start at
// another row's top, so the anchor keeps them out of the check.
func caseRailAt(res *Result, caseBox *box, x float64) *Op {
	for idx := caseBox.opStart; idx <= caseBox.opEnd && idx < len(res.Ops); idx++ {
		operation := &res.Ops[idx]
		if operation.Kind == OpLine && operation.W == 0 && operation.H > 0 &&
			near(operation.X, x) && near(operation.Y, caseBox.y) {
			return operation
		}
	}

	return nil
}
