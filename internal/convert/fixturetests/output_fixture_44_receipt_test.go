package fixturetests

import "testing"

// TestOutputFixture44Receipt checks the centered store title, receipt
// metadata, and the green paid status.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture44Receipt(t *testing.T) {

	committed := readCommittedOps(t, "fixture-44-receipt.pdf")
	fresh := freshOps(t, "fixture-44-receipt.html")

	assertPageOpsMatch(t, committed, fresh)

	assertOpsTextRun(t, committed, 1, "Acme Market",
		244.727, 782.950, 17, "LiberationSans-Bold", [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255})
	assertOpsTextRun(t, committed, 1, "Notebook, recycled paper",
		142.390, 720.627, 10, "LiberationSans-Regular", [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255})
	assertOpsTextRun(t, committed, 1, "Reusable coffee cup",
		142.390, 659.127, 10, "LiberationSans-Regular", [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255})

	assertFixtureOpsShape(t, committed, 1, 18, 19, 0, 0, [4]float64{0, 0, 595.28, 841.89})
}
