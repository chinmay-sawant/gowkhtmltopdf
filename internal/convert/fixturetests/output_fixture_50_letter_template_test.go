package fixturetests

import "testing"

// TestOutputFixture50LetterTemplate checks the letterhead, signature role,
// and both local image placements.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture50LetterTemplate(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-50-letter-template.pdf")
	fresh := freshOps(t, "fixture-50-letter-template.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "Northstar Studio",
		73.701, 735.182, 20, "LiberationSans-Bold", [3]float64{16.0 / 255, 43.0 / 255, 67.0 / 255})
	assertOpsTextRun(t, committed, 1, "Editorial director",
		73.701, 217.790, 10.5, "LiberationSans-Regular", [3]float64{36.0 / 255, 51.0 / 255, 59.0 / 255})
	assertOpsImageBox(t, committed, 1, 368.847, 694.488, 116.107, 79.370)
	assertOpsImageBox(t, committed, 1, 73.701, 428.785, 2.250, 17.008)

	assertFixtureOpsShape(t, committed, 1, 20, 1, 3, 2, [4]float64{0, 0, 595.28, 841.89})
}
