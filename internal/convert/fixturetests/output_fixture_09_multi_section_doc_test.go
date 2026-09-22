//nolint:dupl // fixture contracts intentionally keep their measured assertions local
package fixturetests

import "testing"

// TestOutputFixture09MultiSectionDoc checks the first page and the continuation
// on the second page, along with the report table header styling.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title page 1, "Engineering Report - Q3 2024" #1a3d6d at (28.346, 786.297), 16pt bold
//	section page 1, "1. Delivered work packages" #1a3d6d at (28.346, 726.583), 12pt bold
//	continuation page 2, "Capacity planning for early 2025 assumes a steady
//	2,100 engineering hours per quarter and a 4 % overtime ceiling." at
//	(28.346, 771.544), 10pt regular
//	header page 1, #e8eef5 at (28.346, 664.844), 104.500x20.500
//	rule page 1, #c9d4de, 1pt, (28.346, 685.344) to (132.847, 685.344)
//	2 pages, A4, 79 text runs, 94 strokes, 4 fills, no images
func TestOutputFixture09MultiSectionDoc(t *testing.T) {

	committed := readCommittedOps(t, "fixture-09-multi-section-doc.pdf")
	fresh := freshOps(t, "fixture-09-multi-section-doc.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	tableFill := [3]float64{232.0 / 255, 238.0 / 255, 245.0 / 255}
	border := [3]float64{201.0 / 255, 212.0 / 255, 222.0 / 255}

	assertOpsTextRun(t, committed, 1, "Engineering Report - Q3 2024",
		28.346, 786.297, 16, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "1. Delivered work packages",
		28.346, 726.583, 12, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 2,
		"Capacity planning for early 2025 assumes a steady 2,100 engineering hours per quarter and a 4 % overtime ceiling.",
		28.346, 771.544, 10, "LiberationSans-Regular", bodyColor)

	assertOpsFillRect(t, committed, 1, 28.346, 664.844, 104.500, 20.500, tableFill)
	assertOpsStrokeSegment(t, committed, 1,
		28.346, 685.344, 132.847, 685.344, border, 1)

	if committed.Pages != 2 || len(committed.MediaBoxes) != 2 {
		t.Errorf("pages = %d, mediaboxes = %d, want 2 pages and 2 mediaboxes",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 79 {
		t.Errorf("text runs = %d, want 79", len(committed.Texts))
	}

	if len(committed.Strokes) != 94 {
		t.Errorf("strokes = %d, want 94", len(committed.Strokes))
	}

	if len(committed.Fills) != 4 {
		t.Errorf("fills = %d, want 4", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
