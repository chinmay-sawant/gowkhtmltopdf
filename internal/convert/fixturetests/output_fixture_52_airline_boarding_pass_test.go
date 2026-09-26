package fixturetests

import "testing"

// TestOutputFixture52AirlineBoardingPass checks the brand heading, boarding
// section heading, and the final travel note.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture52AirlineBoardingPass(t *testing.T) {
	committed := readCommittedOps(t, "fixture-52-airline-boarding-pass.pdf")
	fresh := freshOps(t, "fixture-52-airline-boarding-pass.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{11.0 / 255, 61.0 / 255, 145.0 / 255}

	assertOpsTextRun(t, committed, 1, "Atlas Airways",
		28.346, 793.500, 17, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "BOARDING PASSES",
		28.346, 637.630, 11, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1,
		" Arrive at the gate no later than 30 minutes before departure. Photo ID required. Gate and seat assignments may",
		102.605, 178.978, 8.5, "LiberationSans-Regular", [3]float64{58.0 / 255, 74.0 / 255, 90.0 / 255})

	assertFixtureOpsShape(t, committed, 1, 150, 356, 17, 0, [4]float64{0, 0, 595.28, 841.89})
}
