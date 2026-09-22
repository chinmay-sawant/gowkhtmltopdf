//nolint:dupl // fixture contracts intentionally keep their measured assertions local
package fixturetests

import "testing"

// TestOutputFixture61ImplementedPropsB checks the masthead, the fixture
// needle, a code-property row, the final page gallery label, and one image.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture61ImplementedPropsB(t *testing.T) {

	committed := readCommittedOps(t, "fixture-61-implemented-props-b.pdf")
	fresh := freshOps(t, "fixture-61-implemented-props-b.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{31.0 / 255, 75.0 / 255, 153.0 / 255}
	assertOpsTextRun(t, committed, 1, "Implemented CSS audit B",
		202.016, 790.815, 16, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "IMPLEMENTED-PROPS-B",
		40.016, 751.668, 9.5, "LiberationSans-Regular", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 7, "image-orientation",
		67.266, 722.450, 8.5, "LiberationMono-Regular", [3]float64{16.0 / 255, 58.0 / 255, 122.0 / 255})
	assertOpsTextRun(t, committed, 8, "network images",
		34.016, 41.289, 8, "LiberationSans-Regular", [3]float64{85.0 / 255, 85.0 / 255, 102.0 / 255})
	assertOpsImageBox(t, committed, 1, 34.016, 776.374, 90, 27)

	assertFixtureOpsShape(t, committed, 8, 989, 4356, 488, 8, [4]float64{0, 0, 595.28, 841.89})
}
