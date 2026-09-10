package imageout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestRenderZoom proves RenderOptions.Zoom reaches layout.Options.Zoom: the
// same text op is twice as wide at zoom 2.
func TestRenderZoom(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><p style="font-size:20px">zoom</p></body></html>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	font, err := pdf.DefaultFont()
	if err != nil {
		t.Fatalf("default font: %v", err)
	}

	plain, err := layoutResult(t.Context(), root, RenderOptions{Width: 400}, font)
	if err != nil {
		t.Fatalf("plain layout: %v", err)
	}

	zoomed, err := layoutResult(t.Context(), root, RenderOptions{Width: 400, Zoom: 2}, font)
	if err != nil {
		t.Fatalf("zoomed layout: %v", err)
	}

	plainW := firstTextWidth(t, plain)
	zoomedW := firstTextWidth(t, zoomed)

	if math.Abs(zoomedW-2*plainW) > 1e-6 {
		t.Fatalf("text op width at zoom 2 = %g, want %g (2 x %g)", zoomedW, 2*plainW, plainW)
	}
}

func TestRenderOptionsValidateZoom(t *testing.T) {
	t.Parallel()

	for _, zoom := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1} {
		if err := (RenderOptions{Zoom: zoom}).Validate(); err == nil {
			t.Errorf("RenderOptions{Zoom: %v}.Validate() = nil, want error", zoom)
		}
	}

	if err := (RenderOptions{Zoom: 0}).Validate(); err != nil {
		t.Errorf("Zoom 0 must keep the layout default: %v", err)
	}

	if err := (RenderOptions{Zoom: 0.5}).Validate(); err != nil {
		t.Errorf("Zoom 0.5 must validate: %v", err)
	}
}

// firstTextWidth returns the width of the first text op in res.
func firstTextWidth(t *testing.T, res *layout.Result) float64 {
	t.Helper()

	for i := range res.Ops {
		if res.Ops[i].Kind == layout.OpText {
			return res.Ops[i].W
		}
	}

	t.Fatal("no text op in layout result")

	return 0
}
