package fixturetests

import "testing"

// TestOutputFixture14ColorfulReport checks the banner, KPI header fill, and
// colored horizontal rule in the report.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	banner title "September 2024 - Executive Dashboard" #ffffff at (38.846, 794.683), 12pt bold
//	status "Outstanding escalations" #8c1f1f at (37.846, 646.877), 10pt regular
//	banner fill #0f3a5f at (28.346, 771.844), 538.587x41.700
//	KPI header fill #d9e6f2 at (28.346, 737.844), 183.570x25.000
//	hr rule #0f3a5f, 1.5pt, (29.096, 505.694) to (566.184, 505.694)
//	1 page, A4, 30 text runs, 50 strokes, 23 fills, no images
func TestOutputFixture14ColorfulReport(t *testing.T) {
	committed := readCommittedOps(t, "fixture-14-colorful-report.pdf")
	fresh := freshOps(t, "fixture-14-colorful-report.html")

	assertPageOpsMatch(t, committed, fresh)

	white := [3]float64{1, 1, 1}
	red := [3]float64{140.0 / 255, 31.0 / 255, 31.0 / 255}
	bannerBlue := [3]float64{15.0 / 255, 58.0 / 255, 95.0 / 255}
	headerFill := [3]float64{217.0 / 255, 230.0 / 255, 242.0 / 255}

	assertOpsTextRun(t, committed, 1, "September 2024 - Executive Dashboard",
		38.846, 794.683, 12, "LiberationSans-Bold", white)
	assertOpsTextRun(t, committed, 1, "Outstanding escalations",
		37.846, 646.877, 10, "LiberationSans-Regular", red)
	assertOpsFillRect(t, committed, 1,
		28.346, 771.844, 538.587, 41.700, bannerBlue)
	assertOpsFillRect(t, committed, 1,
		28.346, 737.844, 183.570, 25.000, headerFill)
	assertOpsStrokeSegment(t, committed, 1,
		29.096, 505.694, 566.184, 505.694, bannerBlue, 1.5)

	if committed.Pages != 1 || len(committed.MediaBoxes) != 1 {
		t.Errorf("pages = %d, mediaboxes = %d, want 1 page and 1 mediabox",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 30 {
		t.Errorf("text runs = %d, want 30", len(committed.Texts))
	}

	if len(committed.Strokes) != 50 {
		t.Errorf("strokes = %d, want 50", len(committed.Strokes))
	}

	if len(committed.Fills) != 23 {
		t.Errorf("fills = %d, want 23", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
