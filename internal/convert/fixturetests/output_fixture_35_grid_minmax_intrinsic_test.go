package fixturetests

import "testing"

// TestOutputFixture35GridMinmaxIntrinsic checks the minmax and masonry
// headings that bookend the intrinsic sizing demonstrations.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture35GridMinmaxIntrinsic(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-35-grid-minmax-intrinsic.pdf")
	fresh := freshOps(t, "fixture-35-grid-minmax-intrinsic.html")

	assertPageOpsMatch(t, committed, fresh)

	dark := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	label := [3]float64{51.0 / 255, 51.0 / 255, 51.0 / 255}

	assertOpsTextRun(t, committed, 1, "Grid minmax / intrinsic / subgrid / masonry",
		38.346, 791.237, 13, "LiberationSans-Bold", dark)
	assertOpsTextRun(t, committed, 1, "minmax() + fr floors",
		38.346, 770.477, 10, "LiberationSans-Bold", label)
	assertOpsTextRun(t, committed, 1, "Masonry packing",
		38.346, 566.277, 10, "LiberationSans-Bold", label)

	assertFixtureOpsShape(t, committed, 1, 26, 60, 18, 0, [4]float64{0, 0, 595.28, 841.89})
}
