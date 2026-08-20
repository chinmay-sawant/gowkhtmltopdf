//nolint:testpackage // white-box test exercises the PDF path directly
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestRoundedTopPDFStrokeEmitsCornerCurves(t *testing.T) {
	t.Parallel()

	content := pdf.NewContent()
	drawStroke(content, &Op{ //nolint:exhaustruct // focused rounded-border operation
		Kind: OpStrokeRect, X: 12, Y: 16, W: 120, H: 40,
		R: 0.1, G: 0.7, B: 0.4, Width: 4,
		Radius: 8, StrokeMask: StrokeMaskTop,
	}, 0, 200, PaintOptions{
		PageWidth: 0, PageHeight: 0, MarginTop: 0, MarginBottom: 0, MarginLeft: 0, MarginRight: 0,
	}, 220)

	if curves := strings.Count(string(content.Bytes()), " c\n"); curves != 2 {
		t.Fatalf("rounded top PDF path emitted %d corner curves, want 2: %q", curves, content.Bytes())
	}
}

func TestRoundedLeftPDFStrokeEmitsCornerCurves(t *testing.T) {
	t.Parallel()

	content := pdf.NewContent()
	drawStroke(content, &Op{ //nolint:exhaustruct // focused rounded-border operation
		Kind: OpStrokeRect, X: 12, Y: 16, W: 120, H: 40,
		R: 0.1, G: 0.7, B: 0.4, Width: 4,
		Radius: 8, StrokeMask: StrokeMaskLeft,
	}, 0, 200, PaintOptions{
		PageWidth: 0, PageHeight: 0, MarginTop: 0, MarginBottom: 0, MarginLeft: 0, MarginRight: 0,
	}, 220)

	data := string(content.Bytes())
	if curves := strings.Count(data, " c\n"); curves != 2 {
		t.Fatalf("rounded left PDF path emitted %d corner curves, want 2: %q", curves, content.Bytes())
	}

	if !strings.Contains(data, "20 164 m\n") || !strings.Contains(data, "14 198 l\n") {
		t.Fatalf("rounded left PDF path did not keep a straight vertical edge: %q", content.Bytes())
	}
}

// EXT-11: HF PaintBand origin mapping must honor StrokeMask radius and
// RotateDeg the same way body draw* does (no fork through bandStrokeRect/bandText).
func TestPaintBandMatchesBodyRoundedAndVerticalText(t *testing.T) {
	t.Parallel()

	stroke := Op{ //nolint:exhaustruct // focused band vs body policy probe
		Kind: OpStrokeRect, X: 10, Y: 20, W: 80, H: 30,
		R: 0.2, G: 0.4, B: 0.8, Width: 3,
		Radius: 6, StrokeMask: StrokeMaskTop,
	}
	text := Op{ //nolint:exhaustruct // vertical-rl glyph matrix probe
		Kind: OpText, X: 12, Y: 40, W: 10, H: 40,
		Text: "V", Size: 12, RotateDeg: -90,
		R: 0, G: 0, B: 0,
	}

	body := pdf.NewContent()
	margins := PaintOptions{ //nolint:exhaustruct // origin-aligned with band
		MarginLeft: 5,
	}

	const pageH = 200.0

	drawStroke(body, &stroke, 0, 0, margins, pageH)
	drawText(body, &text, 0, 0, margins, pageH, "F0")

	doc := pdf.NewDocument()
	page := doc.AddPage(300, 300)
	band := pdf.NewContent()

	err := PaintBand(page, band, []Op{stroke, text}, BandOptions{ //nolint:exhaustruct
		OriginX: 5, OriginY: pageH,
	})
	if err != nil {
		t.Fatal(err)
	}

	bodyData := string(body.Bytes())
	bandData := string(band.Bytes())
	bandCurves := strings.Count(bandData, " c\n")

	if bandCurves != 2 {
		t.Fatalf("PaintBand rounded top emitted %d curves, want 2 (body policy): %q", bandCurves, bandData)
	}

	if !strings.Contains(bandData, "0 -1 1 0") {
		t.Fatalf("PaintBand ignored RotateDeg=-90 text matrix: %q", bandData)
	}

	if strings.Count(bodyData, " c\n") != bandCurves {
		t.Fatalf("band/body curve count diverged: body=%q band=%q", bodyData, bandData)
	}
}
