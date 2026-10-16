package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestBoxlessForcedBreakStartsNewPage proves blink metadata enforcement for a
// boxless result: a forced break pushes every flow op at or below it onto the
// next page top, so the second rect leaves page zero.
func TestBoxlessForcedBreakStartsNewPage(t *testing.T) {
	t.Parallel()

	res := &Result{
		Ops: []Op{
			{Kind: OpFillRect, X: 10, Y: 20, W: 100, H: 10},
			{Kind: OpFillRect, X: 10, Y: 60, W: 100, H: 10},
		},
		Width:        400,
		Height:       120,
		ForcedBreaks: []float64{50},
	}

	doc := pdf.NewDocument()

	err := Paint(doc, res, PaintOptions{
		PageWidth: 500, PageHeight: 140,
		MarginTop: 20, MarginBottom: 20, MarginLeft: 20, MarginRight: 20,
	})
	if err != nil {
		t.Fatalf("Paint: %v", err)
	}

	if got := doc.PageCount(); got != 2 {
		t.Fatalf("pages = %d, want 2", got)
	}

	if got := pageOfIdx(t, res, 1); got != 1 {
		t.Fatalf("second op page = %d, want 1", got)
	}
}
