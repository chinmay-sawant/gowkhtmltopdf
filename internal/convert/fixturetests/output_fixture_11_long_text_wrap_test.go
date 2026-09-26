package fixturetests

import "testing"

// TestOutputFixture11LongTextWrap checks the first-page heading and the final
// appendix page of the long prose flow.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title page 1, "How Long Documents Paginate" #1a3d6d at (28.346, 798.397), 16pt bold
//	final page 3, low-level run "exists " #1c1c1c at (28.346, 598.180), 10pt regular
//	3 pages, A4, 1698 low-level text runs, no strokes, no fills, no images
func TestOutputFixture11LongTextWrap(t *testing.T) {
	committed := readCommittedOps(t, "fixture-11-long-text-wrap.pdf")
	fresh := freshOps(t, "fixture-11-long-text-wrap.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{28.0 / 255, 28.0 / 255, 28.0 / 255}

	assertOpsTextRun(t, committed, 1, "How Long Documents Paginate",
		28.346, 798.397, 16, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 3, "exists ",
		28.346, 598.180, 10, "LiberationSans-Regular", bodyColor)

	if committed.Pages != 3 || len(committed.MediaBoxes) != 3 {
		t.Errorf("pages = %d, mediaboxes = %d, want 3 pages and 3 mediaboxes",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 1698 {
		t.Errorf("text runs = %d, want 1698", len(committed.Texts))
	}

	if len(committed.Strokes) != 0 {
		t.Errorf("strokes = %d, want 0", len(committed.Strokes))
	}

	if len(committed.Fills) != 0 {
		t.Errorf("fills = %d, want 0", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
