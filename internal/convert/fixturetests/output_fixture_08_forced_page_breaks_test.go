package fixturetests

import "testing"

// TestOutputFixture08ForcedPageBreaks checks the intro page and all four
// forced section pages, including the repeated keep-box fill and border.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	intro page 1, "Facility Operations Manual" #1a3d6d at (28.346, 798.397), 16pt bold
//	section page 2, "1. Shift handover" #1a3d6d at (28.346, 802.183), 12pt bold
//	section page 3, "2. Weekly inspection" #1a3d6d at (28.346, 802.183), 12pt bold
//	section page 4, "3. Emergency response" #1a3d6d at (28.346, 802.183), 12pt bold
//	section page 5, "4. Documentation & forms" #1a3d6d at (28.346, 802.183), 12pt bold
//	keep box page 2, #f3f7fc at (28.346, 726.644), 538.587x34.000
//	rule page 2, #c5d5e8, 0.75pt, (28.346, 760.644) to (566.934, 760.644)
//	5 pages, A4, 25 text runs, 16 strokes, 4 fills, no images
func TestOutputFixture08ForcedPageBreaks(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-08-forced-page-breaks.pdf")
	fresh := freshOps(t, "fixture-08-forced-page-breaks.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	keepFill := [3]float64{243.0 / 255, 247.0 / 255, 252.0 / 255}
	border := [3]float64{197.0 / 255, 213.0 / 255, 232.0 / 255}

	assertOpsTextRun(t, committed, 1, "Facility Operations Manual",
		28.346, 798.397, 16, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 2, "1. Shift handover",
		28.346, 802.183, 12, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 3, "2. Weekly inspection",
		28.346, 802.183, 12, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 4, "3. Emergency response",
		28.346, 802.183, 12, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 5, "4. Documentation & forms",
		28.346, 802.183, 12, "LiberationSans-Bold", headingBlue)

	assertOpsFillRect(t, committed, 2, 28.346, 726.644, 538.587, 34.000, keepFill)
	assertOpsStrokeSegment(t, committed, 2,
		28.346, 760.644, 566.934, 760.644, border, 0.75)

	if committed.Pages != 5 || len(committed.MediaBoxes) != 5 {
		t.Errorf("pages = %d, mediaboxes = %d, want 5 pages and 5 mediaboxes",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 25 {
		t.Errorf("text runs = %d, want 25", len(committed.Texts))
	}

	if len(committed.Strokes) != 16 {
		t.Errorf("strokes = %d, want 16", len(committed.Strokes))
	}

	if len(committed.Fills) != 4 {
		t.Errorf("fills = %d, want 4", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
