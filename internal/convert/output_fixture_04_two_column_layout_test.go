package convert

import "testing"

// TestOutputFixture04TwoColumnLayout checks both independently styled columns
// and their shared table geometry against a fresh conversion.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title  "Product Brief - Widget 2.0" #1a3d6d at (28.346, 798.397), 16pt bold
//	left   "Executive summary" #1a3d6d at (44.346, 743.780), 11pt bold
//	right  "Key numbers" #1a3d6d at (309.890, 743.780), 11pt bold
//	left fill  #eef3f9 at (35.846, 621.494), 258.044x141.200
//	right fill #f7f4ee at (301.390, 621.494), 258.044x141.200
//	rule   #c5d5e8, 0.75pt, (35.846, 762.694) to (293.890, 762.694)
//	1 page, A4, 25 text runs, 8 strokes, 2 fills, no images
func TestOutputFixture04TwoColumnLayout(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-04-two-column-layout.pdf")
	fresh := freshOps(t, "fixture-04-two-column-layout.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	ink := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	leftFill := [3]float64{238.0 / 255, 243.0 / 255, 249.0 / 255}
	rightFill := [3]float64{247.0 / 255, 244.0 / 255, 238.0 / 255}
	leftBorder := [3]float64{197.0 / 255, 213.0 / 255, 232.0 / 255}

	assertOpsTextRun(t, committed, 1, "Product Brief - Widget 2.0",
		28.346, 798.397, 16, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "Executive summary",
		44.346, 743.780, 11, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "Key numbers",
		309.890, 743.780, 11, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1,
		"The two-column body above is a table with two fixed-width cells. "+
			"Float lite (`float`/`clear`) is also supported for invoice",
		28.346, 596.277, 10, "LiberationSans-Regular", ink)

	assertOpsFillRect(t, committed, 1, 35.846, 621.494, 258.044, 141.200, leftFill)
	assertOpsFillRect(t, committed, 1, 301.390, 621.494, 258.044, 141.200, rightFill)
	assertOpsStrokeSegment(t, committed, 1,
		35.846, 762.694, 293.890, 762.694, leftBorder, 0.75)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1", committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 25 {
		t.Errorf("text runs = %d, want 25", len(committed.Texts))
	}

	if len(committed.Strokes) != 8 {
		t.Errorf("strokes = %d, want 8", len(committed.Strokes))
	}

	if len(committed.Fills) != 2 {
		t.Errorf("fills = %d, want 2", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
