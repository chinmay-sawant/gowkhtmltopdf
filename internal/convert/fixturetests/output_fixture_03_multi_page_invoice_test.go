package fixturetests

import "testing"

// TestOutputFixture03MultiPageInvoice checks the four-page statement against
// a fresh conversion, including anchors from the first, middle, and last
// pages and the transaction table header styling.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title   page 1, "Account Statement - Q2 2024" #1a3d6d at (34.346, 781.677), 16pt bold
//	summary page 2, "1. Summary" #1a3d6d at (34.346, 792.223), 12pt bold
//	notes   page 4, "3. Notes" #1a3d6d at (34.346, 792.223), 12pt bold
//	header  page 3, #e8eef5 at (34.346, 760.724), 77.761x18.500
//	rule    page 1, #1a3d6d, 1.5pt, (35.096, 773.374) to (560.184, 773.374)
//	4 pages, A4, 81 text runs, 198 strokes, 6 fills, no images
func TestOutputFixture03MultiPageInvoice(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-03-multi-page-invoice.pdf")
	fresh := freshOps(t, "fixture-03-multi-page-invoice.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	tableFill := [3]float64{232.0 / 255, 238.0 / 255, 245.0 / 255}

	assertOpsTextRun(t, committed, 1, "Account Statement - Q2 2024",
		34.346, 781.677, 16, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 2, "1. Summary",
		34.346, 792.223, 12, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 3, "2. Transactions",
		34.346, 792.223, 12, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 4, "3. Notes",
		34.346, 792.223, 12, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 4,
		"gowkhtmltopdf golden fixture 03 - multi-page with explicit page-break sections and avoid-break blocks.",
		34.346, 749.610, 8, "LiberationSans-Regular", [3]float64{102.0 / 255, 102.0 / 255, 102.0 / 255})

	assertOpsFillRect(t, committed, 3, 34.346, 760.724, 77.761, 18.500, tableFill)
	assertOpsStrokeSegment(t, committed, 1,
		35.096, 773.374, 560.184, 773.374, headingBlue, 1.5)

	if committed.Pages != 4 || len(committed.MediaBoxes) != 4 {
		t.Errorf("pages = %d, mediaboxes = %d, want 4 pages and 4 mediaboxes",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 81 {
		t.Errorf("text runs = %d, want 81", len(committed.Texts))
	}

	if len(committed.Strokes) != 198 {
		t.Errorf("strokes = %d, want 198", len(committed.Strokes))
	}

	if len(committed.Fills) != 6 {
		t.Errorf("fills = %d, want 6", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}

	for index, box := range committed.MediaBoxes {
		if !opsBoxClose(box, [4]float64{0, 0, 595.28, 841.89}) {
			t.Errorf("mediabox[%d] = %v, want A4", index, box)
		}
	}
}
