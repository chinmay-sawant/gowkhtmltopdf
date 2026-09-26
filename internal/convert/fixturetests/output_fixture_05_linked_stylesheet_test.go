package fixturetests

import "testing"

// TestOutputFixture05LinkedStylesheet checks the output driven by the relative
// stylesheet, including its notice box and table header colors.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title   "Release Notes - Widget Suite 2.4" #0f3a5f at (28.346, 795.750), 17pt bold
//	notice  first line #233043 at (36.846, 741.677), 10pt regular
//	section "New features" #0f3a5f at (28.346, 694.083), 12pt bold
//	footer  #8a94a0 at (28.346, 506.070), 8pt regular
//	box     #fff3d6 at (28.346, 717.144), 538.587x42.000
//	header  #e8eef5 at (28.346, 663.244), 129.506x20.600
//	rule    #e5c46a, 0.75pt, (28.346, 759.144) to (566.934, 759.144)
//	1 page, A4, 21 text runs, 36 strokes, 4 fills, no images
func TestOutputFixture05LinkedStylesheet(t *testing.T) {
	committed := readCommittedOps(t, "fixture-05-linked-stylesheet.pdf")
	fresh := freshOps(t, "fixture-05-linked-stylesheet.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{15.0 / 255, 58.0 / 255, 95.0 / 255}
	bodyColor := [3]float64{35.0 / 255, 48.0 / 255, 67.0 / 255}
	muted := [3]float64{138.0 / 255, 148.0 / 255, 160.0 / 255}
	noticeFill := [3]float64{1, 243.0 / 255, 214.0 / 255}
	tableFill := [3]float64{232.0 / 255, 238.0 / 255, 245.0 / 255}
	boxBorder := [3]float64{229.0 / 255, 196.0 / 255, 106.0 / 255}

	assertOpsTextRun(t, committed, 1, "Release Notes - Widget Suite 2.4",
		28.346, 795.750, 17, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1,
		"Important: this release changes the on-disk format of configuration files. Backup existing configurations before",
		36.846, 741.677, 10, "LiberationSans-Regular", bodyColor)
	assertOpsTextRun(t, committed, 1, "New features",
		28.346, 694.083, 12, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1,
		"All styling for this document comes from the linked stylesheet style-05.css - there is no inline style block.",
		28.346, 506.070, 8, "LiberationSans-Regular", muted)

	assertOpsFillRect(t, committed, 1, 28.346, 717.144, 538.587, 42.000, noticeFill)
	assertOpsFillRect(t, committed, 1, 28.346, 663.244, 129.506, 20.600, tableFill)
	assertOpsStrokeSegment(t, committed, 1,
		28.346, 759.144, 566.934, 759.144, boxBorder, 0.75)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1", committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 21 {
		t.Errorf("text runs = %d, want 21", len(committed.Texts))
	}

	if len(committed.Strokes) != 36 {
		t.Errorf("strokes = %d, want 36", len(committed.Strokes))
	}

	if len(committed.Fills) != 4 {
		t.Errorf("fills = %d, want 4", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
