package fixturetests

import "testing"

// TestOutputFixture60ImplementedPropsA checks the masthead, the fixture
// needle, the final page gallery label, and the first local image placement.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture60ImplementedPropsA(t *testing.T) {
	committed := readCommittedOps(t, "fixture-60-implemented-props-a.pdf")
	fresh := freshOps(t, "fixture-60-implemented-props-a.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{31.0 / 255, 75.0 / 255, 153.0 / 255}
	assertOpsTextRun(t, committed, 1, "Implemented CSS audit A",
		201.647, 790.852, 15.965, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "IMPLEMENTED-PROPS-A",
		40.003, 751.792, 9.48, "LiberationSans-Regular", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 8, "network images",
		34.016, 629.144, 7.982, "LiberationSans-Regular", [3]float64{85.0 / 255, 85.0 / 255, 102.0 / 255})
	assertOpsImageBox(t, committed, 1, 34.016, 776.443, 89.803, 26.941)

	assertFixtureOpsShape(t, committed, 8, 859, 1831, 472, 120, [4]float64{0, 0, 595.28, 841.89})
}
