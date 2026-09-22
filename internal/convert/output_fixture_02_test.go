package convert

import "testing"

// TestOutputFixture02TableHeavyInvoice checks the table-heavy committed sample
// against a fresh conversion: text, cell borders, alternating fills, and page
// geometry must stay in the same places with the same colors.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title   "Order 2024-0055 - Consolidated Line Items" #111111 at (34.346, 783.558), 13.5pt bold
//	item    "Hex nut M10 (zinc)" #111111 at (81.973, 557.513), 9pt regular
//	section "Payment terms" #1a3d6d at (34.346, 395.720), 11pt bold
//	header  #e8eef5 at (34.346, 701.634), 42.127x18.800
//	stripe  #f4f6f8 at (34.346, 664.034), 42.127x18.800
//	rule    #999999, 1pt, (34.346, 720.434) to (76.473, 720.434)
//	1 page, A4 595.28 x 841.89, 101 text runs, 214 strokes, 48 fills, no images
func TestOutputFixture02TableHeavyInvoice(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-02-table-heavy-invoice.pdf")
	fresh := freshOps(t, "fixture-02-table-heavy-invoice.html")

	assertPageOpsMatch(t, committed, fresh)

	ink := [3]float64{17.0 / 255, 17.0 / 255, 17.0 / 255}
	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	headerFill := [3]float64{232.0 / 255, 238.0 / 255, 245.0 / 255}
	stripeFill := [3]float64{244.0 / 255, 246.0 / 255, 248.0 / 255}
	borderGray := [3]float64{153.0 / 255, 153.0 / 255, 153.0 / 255}

	assertOpsTextRun(t, committed, 1, "Order 2024-0055 - Consolidated Line Items",
		34.346, 783.558, 13.5, "LiberationSans-Bold", ink)
	assertOpsTextRun(t, committed, 1, "Hex nut M10 (zinc)",
		81.973, 557.513, 9, "LiberationSans-Regular", ink)
	assertOpsTextRun(t, committed, 1, "Payment terms",
		34.346, 395.720, 11, "LiberationSans-Bold", headingBlue)

	assertOpsFillRect(t, committed, 1, 34.346, 701.634, 42.127, 18.800, headerFill)
	assertOpsFillRect(t, committed, 1, 34.346, 664.034, 42.127, 18.800, stripeFill)
	assertOpsStrokeSegment(t, committed, 1,
		34.346, 720.434, 76.473, 720.434, borderGray, 1)

	if committed.Pages != 1 {
		t.Errorf("pages = %d, want 1", committed.Pages)
	}

	if len(committed.Texts) != 101 {
		t.Errorf("text runs = %d, want 101", len(committed.Texts))
	}

	if len(committed.Strokes) != 214 {
		t.Errorf("strokes = %d, want 214", len(committed.Strokes))
	}

	if len(committed.Fills) != 48 {
		t.Errorf("fills = %d, want 48", len(committed.Fills))
	}

	if len(committed.MediaBoxes) != 1 || !opsBoxClose(committed.MediaBoxes[0], [4]float64{0, 0, 595.28, 841.89}) {
		t.Errorf("mediaboxes = %v, want [[0 0 595.28 841.89]]", committed.MediaBoxes)
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
