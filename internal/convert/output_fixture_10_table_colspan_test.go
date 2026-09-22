package convert

import "testing"

// TestOutputFixture10TableColspan checks the full-width colspan headers, the
// nested-table text, and the optional-items table against a fresh conversion.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title    "Quotation Q-2024-081" #1a3d6d at (28.346, 799.343), 15pt bold
//	header   "Position summary" #1a3d6d at (33.846, 760.223), 9pt bold
//	nested   "Wall mount kit" #222222 at (79.933, 680.723), 9pt regular
//	optional "Optional line items" #1a3d6d at (33.846, 612.823), 9pt bold
//	colspan  #e8eef5 at (28.346, 753.944), 538.587x18.800
//	optional fill #e8eef5 at (28.346, 606.544), 538.587x18.800
//	rule     #999999, 1pt, (28.346, 772.744) to (71.433, 772.744)
//	1 page, A4, 39 text runs, 345 strokes, 7 fills, no images
func TestOutputFixture10TableColspan(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-10-table-colspan.pdf")
	fresh := freshOps(t, "fixture-10-table-colspan.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	tableFill := [3]float64{232.0 / 255, 238.0 / 255, 245.0 / 255}
	border := [3]float64{153.0 / 255, 153.0 / 255, 153.0 / 255}

	assertOpsTextRun(t, committed, 1, "Quotation Q-2024-081",
		28.346, 799.343, 15, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "Position summary",
		33.846, 760.223, 9, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "Wall mount kit",
		79.933, 680.723, 9, "LiberationSans-Regular", bodyColor)
	assertOpsTextRun(t, committed, 1, "Optional line items",
		33.846, 612.823, 9, "LiberationSans-Bold", headingBlue)

	assertOpsFillRect(t, committed, 1, 28.346, 753.944, 538.587, 18.800, tableFill)
	assertOpsFillRect(t, committed, 1, 28.346, 606.544, 538.587, 18.800, tableFill)
	assertOpsStrokeSegment(t, committed, 1,
		28.346, 772.744, 71.433, 772.744, border, 1)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1", committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 39 {
		t.Errorf("text runs = %d, want 39", len(committed.Texts))
	}

	if len(committed.Strokes) != 345 {
		t.Errorf("strokes = %d, want 345", len(committed.Strokes))
	}

	if len(committed.Fills) != 7 {
		t.Errorf("fills = %d, want 7", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
