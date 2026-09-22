package convert

import "testing"

// TestOutputFixture07ImageLogo checks both authored image placements and the
// letterhead and signature rules against a fresh conversion.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title  "Nordwind Industries GmbH" #1a3d6d at (155.846, 777.544), 15pt bold
//	logo   page 1, (28.346, 775.644), 120x36
//	preview page 1, (28.346, 652.244), 54x24
//	header rule #1a3d6d, 2.25pt, (29.471, 765.869) to (565.809, 765.869)
//	signature rule #999999, 0.75pt, (28.721, 610.369) to (566.559, 610.369)
//	1 page, A4, 9 text runs, 2 strokes, no fills, 2 images
func TestOutputFixture07ImageLogo(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-07-image-logo.pdf")
	fresh := freshOps(t, "fixture-07-image-logo.html")

	assertPageOpsMatch(t, committed, fresh)

	blue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	bodyColor := [3]float64{34.0 / 255, 34.0 / 255, 34.0 / 255}
	gray := [3]float64{153.0 / 255, 153.0 / 255, 153.0 / 255}

	assertOpsTextRun(t, committed, 1, "Nordwind Industries GmbH",
		155.846, 777.544, 15, "LiberationSans-Bold", blue)
	assertOpsTextRun(t, committed, 1, "Dear Ms. Weber,",
		28.346, 744.777, 10, "LiberationSans-Regular", bodyColor)
	assertOpsTextRun(t, committed, 1, "Export Logistics, Nordwind Industries GmbH",
		28.346, 566.124, 9, "LiberationSans-Regular", [3]float64{85.0 / 255, 85.0 / 255, 85.0 / 255})

	assertOpsImageBox(t, committed, 1, 28.346, 775.644, 120, 36)
	assertOpsImageBox(t, committed, 1, 28.346, 652.244, 54, 24)
	assertOpsStrokeSegment(t, committed, 1,
		29.471, 765.869, 565.809, 765.869, blue, 2.25)
	assertOpsStrokeSegment(t, committed, 1,
		28.721, 610.369, 566.559, 610.369, gray, 0.75)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1", committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 9 {
		t.Errorf("text runs = %d, want 9", len(committed.Texts))
	}

	if len(committed.Strokes) != 2 {
		t.Errorf("strokes = %d, want 2", len(committed.Strokes))
	}

	if len(committed.Fills) != 0 {
		t.Errorf("fills = %d, want 0", len(committed.Fills))
	}

	if len(committed.Images) != 2 {
		t.Errorf("images = %d, want 2", len(committed.Images))
	}
}
