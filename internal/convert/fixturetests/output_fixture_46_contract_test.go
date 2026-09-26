package fixturetests

import "testing"

// TestOutputFixture46Contract checks the agreement heading, contract metadata,
// and the two signature labels.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture46Contract(t *testing.T) {
	committed := readCommittedOps(t, "fixture-46-contract.pdf")
	fresh := freshOps(t, "fixture-46-contract.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{40.0 / 255, 59.0 / 255, 82.0 / 255}
	black := [3]float64{32.0 / 255, 32.0 / 255, 32.0 / 255}

	assertOpsTextRun(t, committed, 1, "Professional Services Agreement",
		28.346, 795.153, 18, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "Northstar Labs GmbH",
		35.346, 726.888, 9.5, "LiberationSans-Regular", [3]float64{32.0 / 255, 32.0 / 255, 32.0 / 255})
	assertOpsTextRun(t, committed, 1, "For Meridian Systems AG",
		298.390, 372.763, 9.5, "LiberationSans-Regular", black)

	assertFixtureOpsShape(t, committed, 1, 34, 11, 1, 0, [4]float64{0, 0, 595.28, 841.89})
}
