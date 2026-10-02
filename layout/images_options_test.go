package layout_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// backgroundSource paints a 40x40 box whose background asks for the image
// source "bg". css.Apply fetches nothing, so the source resolves only through
// layout.Options.Images.
const backgroundSource = `<body style="margin:0">` +
	`<div style="width:40px;height:40px;background-image:url('bg');` +
	`background-size:100% 100%"></div></body>`

// redPNG builds a 2x2 opaque red PNG.
func redPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
		img.Pix[i+3] = 255
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

// imageResolver answers "bg" with the red PNG and rejects any other source.
func imageResolver(t *testing.T) func(string) ([]byte, error) {
	t.Helper()

	data := redPNG(t)

	return func(src string) ([]byte, error) {
		if src != "bg" {
			return nil, errors.New("unexpected src") //nolint:err113 // test-local sentinel
		}

		return data, nil
	}
}

func TestDisplayListOptionsResolvesImage(t *testing.T) {
	t.Parallel()

	display, err := layout.DisplayListOptions(
		t.Context(), styledOf(t, backgroundSource), layout.Options{Images: imageResolver(t)},
	)
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	for _, paintOp := range display.Ops {
		if paintOp.Kind != layout.DisplayOpImage {
			continue
		}

		data, width, height := paintOp.ImageBytes()
		if len(data) == 0 || width <= 0 || height <= 0 {
			t.Fatalf("image op %d bytes %dx%d", len(data), width, height)
		}

		if !bytes.HasPrefix(data, []byte("\x89PNG")) {
			t.Fatal("image payload is not a PNG")
		}

		return
	}

	t.Fatal("no image op for a resolved background")
}

func TestDisplayListSkipsImagesWithoutResolver(t *testing.T) {
	t.Parallel()

	for _, paintOp := range displayOf(t, backgroundSource).Ops {
		if paintOp.Kind == layout.DisplayOpImage {
			t.Fatal("image resolved without a resolver")
		}
	}
}

func TestLayOptionsPaintsResolvedImage(t *testing.T) {
	t.Parallel()

	placed, err := layout.LayOptions(
		t.Context(), styledOf(t, backgroundSource), layout.Options{Images: imageResolver(t)},
	)
	if err != nil {
		t.Fatalf("lay: %v", err)
	}

	image := color.NRGBAModel.Convert(placed.Image().At(20, 20))

	got, ok := image.(color.NRGBA)
	if !ok {
		t.Fatal("center pixel is not NRGBA")
	}

	if got.R < 200 || got.G > 60 || got.B > 60 {
		t.Fatalf("center pixel %+v, want red", got)
	}
}
