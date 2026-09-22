package convert

import "testing"

// TestOutputFixture32FlexGridFull checks the Stage A/B heading, flex labels,
// and the grid's final cell.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture32FlexGridFull(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-32-flex-grid-full.pdf")
	fresh := freshOps(t, "fixture-32-flex-grid-full.html")

	assertPageOpsMatch(t, committed, fresh)

	dark := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	assertOpsTextRun(t, committed, 1, "Flex & grid Stage A/B",
		38.346, 790.290, 14, "LiberationSans-Bold", dark)
	assertOpsTextRun(t, committed, 1, "Items A",
		38.346, 751.970, 8, "LiberationSans-Regular", [3]float64{85.0 / 255, 85.0 / 255, 85.0 / 255})
	assertOpsTextRun(t, committed, 1, "Grow (flex: 1 1 auto)",
		46.346, 614.223, 9, "LiberationSans-Regular", dark)
	assertOpsFillRect(t, committed, 1,
		38.346, 692.344, 250.000, 55.600,
		[3]float64{227.0 / 255, 242.0 / 255, 253.0 / 255})

	assertFixtureOpsShape(t, committed, 1, 21, 48, 15, 0, [4]float64{0, 0, 595.28, 841.89})
}
