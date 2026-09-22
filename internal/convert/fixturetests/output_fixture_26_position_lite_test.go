package fixturetests

import "testing"

// TestOutputFixture26PositionLite checks the relative block and absolute
// overlay positions in the one-page sample.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture26PositionLite(t *testing.T) {

	committed := readCommittedOps(t, "fixture-26-position-lite.pdf")
	fresh := freshOps(t, "fixture-26-position-lite.html")

	assertPageOpsMatch(t, committed, fresh)

	black := [3]float64{0, 0, 0}
	assertOpsTextRun(t, committed, 1, "Position lite",
		44.346, 765.210, 20, "LiberationSans-Bold", black)
	assertOpsTextRun(t, committed, 1, "Relatively offset block",
		65.346, 722.277, 10, "LiberationSans-Regular", black)
	assertOpsTextRun(t, committed, 1, "Absolute overlay",
		74.346, 677.277, 10, "LiberationSans-Regular", black)

	assertFixtureOpsShape(t, committed, 1, 4, 320, 4, 0, [4]float64{0, 0, 595.28, 841.89})
}
