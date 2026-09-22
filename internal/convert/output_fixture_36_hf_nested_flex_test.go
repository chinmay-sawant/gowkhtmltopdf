package convert

import "testing"

// TestOutputFixture36HFNestedFlex checks the body, nested header, nested
// footer, and header logo. requestForFixture attaches both companions.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture36HFNestedFlex(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-36-hf-nested-flex.pdf")
	fresh := freshOps(t, "fixture-36-hf-nested-flex.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	body := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}

	assertOpsTextRun(t, committed, 1, "Nested HTML header/footer",
		28.346, 772.423, 16, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "Nested HF flex header",
		36.346, 820.370, 9, "LiberationSans-Regular", blue)
	assertOpsTextRun(t, committed, 1, "Target section",
		28.346, 697.643, 13, "LiberationSans-Bold", body)
	assertOpsTextRun(t, committed, 1, "Nested HF footer ",
		240.943, 2.027, 8, "LiberationSans-Regular", [3]float64{85.0 / 255, 85.0 / 255, 85.0 / 255})
	assertOpsImageBox(t, committed, 1, 36.346, 828.890, 30, 9)

	assertFixtureOpsShape(t, committed, 1, 16, 2, 0, 1, [4]float64{0, 0, 595.28, 841.89})
}
