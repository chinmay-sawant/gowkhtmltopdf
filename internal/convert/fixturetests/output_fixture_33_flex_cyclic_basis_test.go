package fixturetests

import "testing"

// TestOutputFixture33FlexCyclicBasis checks the definite-basis heading and
// the content-sized sibling in the indefinite column case.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture33FlexCyclicBasis(t *testing.T) {

	committed := readCommittedOps(t, "fixture-33-flex-cyclic-basis.pdf")
	fresh := freshOps(t, "fixture-33-flex-cyclic-basis.html")

	assertPageOpsMatch(t, committed, fresh)

	dark := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	assertOpsTextRun(t, committed, 1, "Flex cyclic percentage basis",
		38.346, 790.290, 14, "LiberationSans-Bold", dark)
	assertOpsTextRun(t, committed, 1, "Definite row ",
		38.346, 766.330, 11, "LiberationSans-Bold", [3]float64{51.0 / 255, 51.0 / 255, 51.0 / 255})
	assertOpsTextRun(t, committed, 1, "Auto basis sibling",
		49.346, 484.023, 9, "LiberationSans-Regular", dark)

	assertFixtureOpsShape(t, committed, 1, 23, 13, 9, 0, [4]float64{0, 0, 595.28, 841.89})
}
