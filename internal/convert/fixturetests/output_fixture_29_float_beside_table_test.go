package fixturetests

import "testing"

// TestOutputFixture29FloatBesideTable checks the floated table heading, the
// post-clear row, and the table header fill.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture29FloatBesideTable(t *testing.T) {
	committed := readCommittedOps(t, "fixture-29-float-beside-table.pdf")
	fresh := freshOps(t, "fixture-29-float-beside-table.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	dark := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}

	assertOpsTextRun(t, committed, 1, "Float-beside-table quality sample",
		28.346, 800.290, 14, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "FLOAT-TABLE-INFOBOX",
		412.434, 785.170, 8, "LiberationSans-Bold", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 1, "Post-clear full-width row FLOAT-AFTER-CLEAR-ROW",
		202.126, 597.923, 9, "LiberationSans-Regular", dark)
	assertOpsFillRect(t, committed, 1,
		407.934, 780.144, 158.000, 15.600, blue)

	assertFixtureOpsShape(t, committed, 1, 28, 34, 3, 0, [4]float64{0, 0, 595.28, 841.89})
}
