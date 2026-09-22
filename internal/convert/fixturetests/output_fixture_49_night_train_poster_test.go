package fixturetests

import "testing"

// TestOutputFixture49NightTrainPoster checks the poster title, edition year,
// and the full-width local illustration.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture49NightTrainPoster(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-49-night-train-poster.pdf")
	fresh := freshOps(t, "fixture-49-night-train-poster.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "The Night Line",
		65.197, 184.315, 32, "LiberationSerif-Bold", [3]float64{251.0 / 255, 244.0 / 255, 230.0 / 255})
	assertOpsTextRun(t, committed, 1, "2026",
		508.480, 57.530, 8, "LiberationMono-Regular", [3]float64{183.0 / 255, 200.0 / 255, 202.0 / 255})
	assertOpsImageBox(t, committed, 1, 28.346, 261.281, 538.587, 552.262)

	assertFixtureOpsShape(t, committed, 1, 7, 1, 3, 1, [4]float64{0, 0, 595.28, 841.89})
}
