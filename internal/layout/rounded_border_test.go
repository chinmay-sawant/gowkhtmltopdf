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
