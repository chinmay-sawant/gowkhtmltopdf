package convert

import "testing"

// TestOutputFixture23TheadRepeat checks that the table heading is painted on
// both the first and continuation pages.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture23TheadRepeat(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-23-thead-repeat.pdf")
	fresh := freshOps(t, "fixture-23-thead-repeat.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	assertOpsTextRun(t, committed, 1, "Statement ST-23-1001",
		28.346, 800.290, 14, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "Line",
		33.846, 777.473, 9, "LiberationSans-Bold", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 2, "Line",
		33.846, 800.273, 9, "LiberationSans-Bold", [3]float64{1, 1, 1})
	assertOpsFillRect(t, committed, 1,
		28.346, 770.444, 43.087, 20.300, blue)

	assertFixtureOpsShape(t, committed, 2, 249, 563, 8, 0, [4]float64{0, 0, 595.28, 841.89})
}
