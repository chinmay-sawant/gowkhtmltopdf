package fixturetests

import "testing"

// TestOutputFixture51AsteriaStorybook checks the first-page story title, the
// final-page asset note, and all three illustration placements.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture51AsteriaStorybook(t *testing.T) {
	committed := readCommittedOps(t, "fixture-51-asteria-storybook.pdf")
	fresh := freshOps(t, "fixture-51-asteria-storybook.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "The Lantern of Asteria",
		68.031, 196.384, 28, "LiberationSerif-Bold", [3]float64{243.0 / 255, 198.0 / 255, 107.0 / 255})
	assertOpsTextRun(t, committed, 4, "embedded as HTML text, while the illustrations are generated text-free assets.",
		68.031, 545.284, 8.5, "LiberationSans-Regular", [3]float64{96.0 / 255, 112.0 / 255, 120.0 / 255})
	assertOpsImageBox(t, committed, 1, 28.346, 261.281, 538.587, 552.262)
	assertOpsImageBox(t, committed, 2, 68.031, 443.715, 453.543, 310.039)
	assertOpsImageBox(t, committed, 3, 68.031, 443.715, 453.543, 310.039)

	assertFixtureOpsShape(t, committed, 4, 26, 1, 9, 3, [4]float64{0, 0, 595.28, 841.89})
}
