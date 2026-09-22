package fixturetests

import "testing"

// TestOutputFixture45PurchaseOrder checks the purchase-order heading, the
// delivery note, and the approval signature.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture45PurchaseOrder(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-45-purchase-order.pdf")
	fresh := freshOps(t, "fixture-45-purchase-order.html")

	assertPageOpsMatch(t, committed, fresh)

	textColor := [3]float64{32.0 / 255, 43.0 / 255, 54.0 / 255}

	assertOpsTextRun(t, committed, 1, "Purchase Order PO-2025-0045",
		28.346, 795.557, 19, "LiberationSans-Bold", [3]float64{23.0 / 255, 74.0 / 255, 115.0 / 255})
	assertOpsTextRun(t, committed, 1, "Reference: LAB-DOCK-02",
		304.640, 692.550, 9.5, "LiberationSans-Regular", textColor)
	assertOpsTextRun(t, committed, 1, "Approved by: Daniel Weber",
		298.390, 492.250, 9.5, "LiberationSans-Regular", textColor)
	assertOpsStrokeSegment(t, committed, 1,
		29.471, 763.969, 565.809, 763.969,
		[3]float64{23.0 / 255, 74.0 / 255, 115.0 / 255}, 2.25)

	assertFixtureOpsShape(t, committed, 1, 41, 70, 6, 0, [4]float64{0, 0, 595.28, 841.89})
}
