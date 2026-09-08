package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestOptionsValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts Options
	}{
		{name: "zero width", opts: Options{Width: 0, Height: 800}},
		{name: "negative width", opts: Options{Width: -10, Height: 800}},
		{name: "negative height", opts: Options{Width: 500, Height: -10}},
		{name: "nan width", opts: Options{Width: math.NaN(), Height: 800}},
		{name: "negative zoom", opts: Options{Width: 500, Height: 800, Zoom: -1}},
		{name: "nan zoom", opts: Options{Width: 500, Height: 800, Zoom: math.NaN()}},
		{name: "unknown media", opts: Options{Width: 500, Height: 800, Media: "tv"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if err := testCase.opts.validate(); err == nil {
				t.Fatal("validate accepted invalid options")
			}
		})
	}
}

func TestOptionsValidateAcceptsDefaults(t *testing.T) {
	t.Parallel()

	opts := []Options{
		{Width: 500, Height: 800},
		{Width: 500, Height: 0, Zoom: 0.95, Media: "print"},
		{Width: 400, Height: 600, Media: "screen"},
	}

	for _, option := range opts {
		if err := option.validate(); err != nil {
			t.Fatalf("validate(%+v) = %v, want nil", option, err)
		}
	}
}

func TestPaintOptionsValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts PaintOptions
	}{
		{name: "zero page", opts: PaintOptions{PageWidth: 0, PageHeight: 842}},
		{name: "negative height", opts: PaintOptions{PageWidth: 595, PageHeight: -842}},
		{name: "negative margin", opts: PaintOptions{PageWidth: 595, PageHeight: 842, MarginTop: -10}},
		{name: "nan margin", opts: PaintOptions{PageWidth: 595, PageHeight: 842, MarginLeft: math.NaN()}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if err := testCase.opts.validate(); err == nil {
				t.Fatal("validate accepted invalid paint options")
			}
		})
	}
}

func TestPaintOptionsValidateAcceptsDefaults(t *testing.T) {
	t.Parallel()

	opts := []PaintOptions{
		{PageWidth: 595, PageHeight: 842},
		{PageWidth: 595, PageHeight: 842, MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35},
	}

	for _, option := range opts {
		if err := option.validate(); err != nil {
			t.Fatalf("validate(%+v) = %v, want nil", option, err)
		}
	}
}

// TestLayoutContextRejectsNegativeZoom proves the fail-fast path at the
// entry point: the old zoomScale clamp would have silently treated a
// negative zoom as no zoom.
func TestLayoutContextRejectsNegativeZoom(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><p>x</p></body></html>`)

	_, err := Layout(root, Options{
		Width: 500, Height: 800, Zoom: -1,
	})
	if err == nil {
		t.Fatal("Layout accepted a negative zoom")
	}
}

func TestPaintContextRejectsMarginsThatSwallowPage(t *testing.T) {
	t.Parallel()

	doc := pdf.NewDocument()
	res := &Result{
		Ops: []Op{},
	}

	// Margins that swallow the page are not a validation error: header/footer
	// auto margins can legitimately produce this shape (tall HTML header) and
	// the engine clips such headers while the body fallback (contentH =
	// PageHeight) keeps conversion alive. Paint must not reject it.
	err := Paint(doc, res, PaintOptions{
		PageWidth: 595, PageHeight: 842, MarginTop: 500, MarginBottom: 500,
	})
	if err != nil {
		t.Fatalf("Paint rejected margins that swallow the page: %v", err)
	}
}
