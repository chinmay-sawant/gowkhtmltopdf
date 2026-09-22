package fixturetests

import "testing"

// TestOutputFixture16InvoiceWithCSS checks the invoice letterhead, line-item
// table header, and the footer continuation on page two.
//
// Ground truth measured with scripts/inspect_pdf_ops.py (PyMuPDF 1.27.1):
//
//	title page 1, "Nordwind Industries GmbH" #1a3d6d at (28.346, 797.450), 17pt bold
//	line-item header fill page 1, #1a3d6d at (28.346, 642.644), 32.315x18.800
//	footer page 2, "Payment terms: net 30 days from issue date. Late payment
//	interest: 2 % per month. Please quote the invoice number with every payment.
//	Goods" #666666 at (28.346, 792.470), 8pt regular
//	footer rule page 2, #aab7c4, 0.75pt, (28.721, 813.169) to (566.559, 813.169)
//	2 pages, A4, 175 text runs, 333 strokes, 78 fills, no images
func TestOutputFixture16InvoiceWithCSS(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-16-invoice-with-css.pdf")
	fresh := freshOps(t, "fixture-16-invoice-with-css.html")

	assertPageOpsMatch(t, committed, fresh)

	headingBlue := [3]float64{26.0 / 255, 61.0 / 255, 109.0 / 255}
	white := [3]float64{1, 1, 1}
	footerColor := [3]float64{102.0 / 255, 102.0 / 255, 102.0 / 255}
	border := [3]float64{170.0 / 255, 183.0 / 255, 196.0 / 255}

	assertOpsTextRun(t, committed, 1, "Nordwind Industries GmbH",
		28.346, 797.450, 17, "LiberationSans-Bold", headingBlue)
	assertOpsTextRun(t, committed, 1, "#",
		33.846, 648.923, 9, "LiberationSans-Bold", white)
	assertOpsTextRun(t, committed, 2,
		"Payment terms: net 30 days from issue date. Late payment interest: 2 % per month. "+
			"Please quote the invoice number with every payment. Goods",
		28.346, 792.470, 8, "LiberationSans-Regular", footerColor)
	assertOpsFillRect(t, committed, 1,
		28.346, 642.644, 32.315, 18.800, headingBlue)
	assertOpsStrokeSegment(t, committed, 2,
		28.721, 813.169, 566.559, 813.169, border, 0.75)

	if committed.Pages != 2 || len(committed.MediaBoxes) != 2 {
		t.Errorf("pages = %d, mediaboxes = %d, want 2 pages and 2 mediaboxes",
			committed.Pages, len(committed.MediaBoxes))
	}

	if len(committed.Texts) != 175 {
		t.Errorf("text runs = %d, want 175", len(committed.Texts))
	}

	if len(committed.Strokes) != 333 {
		t.Errorf("strokes = %d, want 333", len(committed.Strokes))
	}

	if len(committed.Fills) != 78 {
		t.Errorf("fills = %d, want 78", len(committed.Fills))
	}

	if len(committed.Images) != 0 {
		t.Errorf("images = %d, want 0", len(committed.Images))
	}
}
