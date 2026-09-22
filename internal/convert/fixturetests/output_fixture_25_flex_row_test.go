package fixturetests

import "testing"

// TestOutputFixture25FlexRow checks the flex row title and the right edge of
// the three-item row.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture25FlexRow(t *testing.T) {

	committed := readCommittedOps(t, "fixture-25-flex-row.pdf")
	fresh := freshOps(t, "fixture-25-flex-row.html")

	assertPageOpsMatch(t, committed, fresh)

	black := [3]float64{0, 0, 0}
	assertOpsTextRun(t, committed, 1, "Flex row lite",
		40.346, 769.210, 20, "LiberationSans-Bold", black)
	assertOpsTextRun(t, committed, 1, "Growing middle column",
		147.346, 730.277, 10, "LiberationSans-Regular", black)
	assertOpsTextRun(t, committed, 1, "Right",
		463.934, 730.277, 10, "LiberationSans-Regular", black)

	assertFixtureOpsShape(t, committed, 1, 4, 4, 3, 0, [4]float64{0, 0, 595.28, 841.89})
}
