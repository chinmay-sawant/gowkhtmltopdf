package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestMixedBorderCalloutKeepsNaturalHeight guards fixture-43 page 2: a box
// whose left border is wider than the others gets a miter-shortened bottom
// rail (borderOpsSides), and chrome repair must still classify that rail as
// the box's own chrome. Before opMiteredHorizontalRail, the rail counted as
// content ink, padding-bottom was added a second time, and the callout
// background and coral left rail hung ~8pt below the bottom border.
func TestMixedBorderCalloutKeepsNaturalHeight(t *testing.T) {
	t.Parallel()

	const doc = `<!DOCTYPE html><html><body>
	  <div class="callout"><strong>Risk boundary:</strong> Atlas must never
	  imply that an unavailable integration has produced a clean result.
	  Empty, delayed, and disconnected states are shown distinctly in the
	  product and in the export.</div>
	</body></html>`

	res := layoutHTML(t, doc, sheet(t, `
	  body { margin: 0; font-size: 8.8pt; line-height: 1.35; }
	  .callout {
	    background: #fff7ed;
	    border: 1pt solid #e8c89d;
	    border-left: 3pt solid #d05a3c;
	    margin: 0;
	    padding: 7pt 9pt;
	  }
	`))

	callout := findBoxByClass(t, res, "callout")
	if callout == nil {
		t.Fatal("callout box missing")
	}

	preH := callout.height
	if preH <= 0 {
		t.Fatalf("callout pre-paint height = %.2f", preH)
	}

	preRailH, preRailY := findBorderLeftRail(res.Ops, callout.x)
	if preRailH <= 0 {
		t.Fatal("callout left rail missing before Paint")
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: testViewport, PageHeight: 800,
	}); err != nil {
		t.Fatal(err)
	}

	if math.Abs(callout.height-preH) > 0.5 {
		t.Fatalf("callout stretched after Paint: h=%.2f want %.2f", callout.height, preH)
	}

	railH, railY := findBorderLeftRail(res.Ops, callout.x)
	if railH <= 0 {
		t.Fatal("callout left rail missing after Paint")
	}

	if math.Abs(railY-preRailY) > 0.5 || math.Abs(railH-preRailH) > 0.5 {
		t.Fatalf("left rail changed in Paint: y=%.2f h=%.2f want y=%.2f h=%.2f", railY, railH, preRailY, preRailH)
	}

	bottom := callout.y + callout.height
	if math.Abs(railY+railH-bottom) > 0.5 {
		t.Fatalf("left rail ends at %.2f, box bottom at %.2f", railY+railH, bottom)
	}
}
