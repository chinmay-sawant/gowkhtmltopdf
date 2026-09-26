package fixturetests

import "testing"

// TestOutputFixture63PageLevelDemos checks the dark-scheme title, the fixture
// needle, and the split-box text on the later page.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture63PageLevelDemos(t *testing.T) {
	committed := readCommittedOps(t, "fixture-63-page-level-demos.pdf")
	fresh := freshOps(t, "fixture-63-page-level-demos.html")

	assertPageOpsMatch(t, committed, fresh)

	darkBlue := [3]float64{156.0 / 255, 194.0 / 255, 255.0 / 255}
	assertOpsTextRun(t, committed, 1, "Page-level demos: fragmentation, footnotes, and color-scheme dark",
		34.016, 792.549, 15, "LiberationSans-Bold", darkBlue)
	assertOpsTextRun(t, committed, 1, "FIXTURE-63-PAGE-LEVEL",
		40.016, 776.418, 9.5, "LiberationSans-Regular", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 6, "break-inside: auto: this 450pt box splits across the boundary.",
		41.016, 283.818, 9.5, "LiberationSans-Regular", [3]float64{232.0 / 255, 232.0 / 255, 232.0 / 255})

	assertFixtureOpsShape(t, committed, 7, 35, 1495, 18, 0, [4]float64{0, 0, 595.28, 841.89})
}
