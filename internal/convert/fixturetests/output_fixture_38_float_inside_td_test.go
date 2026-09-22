package fixturetests

import "testing"

// TestOutputFixture38FloatInsideTD checks the floated cell labels and the
// neighboring top-aligned text.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture38FloatInsideTD(t *testing.T) {

	committed := readCommittedOps(t, "fixture-38-float-inside-td.pdf")
	fresh := freshOps(t, "fixture-38-float-inside-td.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	dark := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}

	assertOpsTextRun(t, committed, 1, "Float-inside-td quality sample",
		28.346, 800.290, 14, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "FLOAT-TABLE-CLEARS-BELOW",
		404.621, 786.527, 10, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "FLOAT-INSIDE-TD-CLEAR",
		35.346, 674.277, 10, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "vertical-align:top so row packing stays report-friendly.",
		316.660, 722.277, 10, "LiberationSans-Regular", dark)

	assertFixtureOpsShape(t, committed, 1, 18, 7, 2, 0, [4]float64{0, 0, 595.28, 841.89})
}
