package imageout

import (
	"errors"
	"image"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestRenderOptionsValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts RenderOptions
	}{
		{name: "negative width", opts: RenderOptions{Width: -1}},
		{name: "negative height", opts: RenderOptions{Height: -10}},
		{name: "negative padding", opts: RenderOptions{Padding: -1}},
		{name: "negative crop offset", opts: RenderOptions{Crop: image.Rect(-5, 0, 100, 100)}},
		{
			name: "negative crop width",
			opts: RenderOptions{Crop: image.Rectangle{
				Min: image.Point{X: 5, Y: 5}, Max: image.Point{X: 1, Y: 100},
			}},
		},
		{name: "unknown media", opts: RenderOptions{Media: "tv"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if err := testCase.opts.Validate(); err == nil {
				t.Fatal("Validate accepted invalid options")
			}
		})
	}
}

func TestRenderOptionsValidateAcceptsDefaults(t *testing.T) {
	t.Parallel()

	opts := []RenderOptions{
		{},
		{Width: 100, Height: 50, Media: "print"},
		{Width: 100, Height: 50, Padding: 12},
		{Media: "screen"},
		{Crop: image.Rect(10, 10, 50, 50)},
	}

	for _, option := range opts {
		if err := option.Validate(); err != nil {
			t.Fatalf("Validate(%+v) = %v, want nil", option, err)
		}
	}
}

func TestRenderContextRejectsNegativeWidthBeforeLayout(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><p>x</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Render(root, RenderOptions{Width: -1})
	if !errors.Is(err, errNegativeDimension) {
		t.Fatalf("Render error = %v, want errNegativeDimension", err)
	}

	_, err = Render(root, RenderOptions{Media: "tv"})
	if !errors.Is(err, errInvalidMediaType) {
		t.Fatalf("Render error = %v, want errInvalidMediaType", err)
	}
}
