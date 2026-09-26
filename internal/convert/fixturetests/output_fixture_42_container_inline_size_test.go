package fixturetests

import "testing"

// TestOutputFixture42ContainerInlineSize checks the wide container query
// branch and its green matched result.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture42ContainerInlineSize(t *testing.T) {
	committed := readCommittedOps(t, "fixture-42-container-inline-size.pdf")
	fresh := freshOps(t, "fixture-42-container-inline-size.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "Wide card (matches @container",
		61.346, 745.503, 18, "LiberationSans-Bold", [3]float64{204.0 / 255, 0, 0})
	assertOpsTextRun(t, committed, 1, "matched.",
		60.346, 518.983, 12, "LiberationSans-Bold", [3]float64{0, 102.0 / 255, 0})
	assertOpsStrokeSegment(t, committed, 1,
		52.346, 789.544, 352.346, 789.544,
		[3]float64{34.0 / 255, 68.0 / 255, 102.0 / 255}, 1)

	assertFixtureOpsShape(t, committed, 1, 8, 8, 1, 0, [4]float64{0, 0, 595.28, 841.89})
}
