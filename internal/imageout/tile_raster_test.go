package imageout

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

const (
	tileRasterSmallW = 80
	tileRasterSmallH = 64
	tileRasterStripH = 8
)

func TestStripRasterMatchesFullCanvas(t *testing.T) {
	t.Parallel()

	res := smallStripFixture(t)

	full, err := rasterizeContextPolicy(
		t.Context(), res, res.Height, false, 0, 0, rasterPolicyDirect,
	)
	if err != nil {
		t.Fatal(err)
	}

	stripBytes := tileRasterSmallW * tileRasterStripH * nrgbaBytes

	tiled, err := rasterizeStrips(
		t.Context(), res, res.Height, false, 0, 0,
		image.Rectangle{}, rasterPolicyDirect, stripBytes,
	)
	if err != nil {
		t.Fatal(err)
	}

	assertSameNRGBA(t, tiled, full)
}

func TestStripRasterPublicTileDimensions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		width  int
		height int
	}{
		{name: "250-tile-canvas", width: 1024, height: 2056},
		{name: "500-tile-canvas", width: 1024, height: 4040},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertPublicStripCase(t, testCase.width, testCase.height)
		})
	}
}

func assertPublicStripCase(t *testing.T, width, height int) {
	t.Helper()

	res := publicSizeStripFixture(width, height)

	full, err := rasterizeContextPolicy(
		t.Context(), res, res.Height, false, 0, 0, rasterPolicyAuto,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := full.Bounds(); got.Dx() != width || got.Dy() != height {
		t.Fatalf("full canvas = %dx%d, want %dx%d", got.Dx(), got.Dy(), width, height)
	}

	tiled, err := rasterizeStrips(
		t.Context(), res, res.Height, false, 0, 0,
		image.Rectangle{}, rasterPolicyAuto, rasterStripBytes,
	)
	if err != nil {
		t.Fatal(err)
	}

	assertSameNRGBA(t, tiled, full)
	assertStripPNGRoundTrip(t, res, full)
}

func assertStripPNGRoundTrip(t *testing.T, res *layout.Result, full *image.NRGBA) {
	t.Helper()

	plan, err := planRasterCanvas(res, res.Height, 0, 0, rasterPolicyAuto)
	if err != nil {
		t.Fatal(err)
	}

	if !stripRasterApplies(plan, formatPNG) {
		t.Fatal("public tile canvas did not take the strip PNG path")
	}

	job := pendingImageRaster{
		res:         res,
		height:      res.Height,
		transparent: false,
		padding:     0,
		zoom:        0,
		crop:        image.Rectangle{},
	}

	var encoded bytes.Buffer
	if err := encodeStripPNG(t.Context(), &encoded, job, plan, rasterStripBytes); err != nil {
		t.Fatal(err)
	}

	decoded, err := png.Decode(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	assertPNGPixelsEqual(t, full, decoded)
}

func TestStripRasterBufferSmallerThanFullCanvas(t *testing.T) {
	t.Parallel()

	const width, height = 1024, 4040

	rows := rasterStripRows(width, height, rasterStripBytes)
	stripBytes := nrgbaStride(width) * rows
	fullBytes := nrgbaStride(width) * height

	if stripBytes > rasterStripBytes {
		t.Fatalf("strip backing = %d bytes, want <= %d", stripBytes, rasterStripBytes)
	}

	if stripBytes >= fullBytes {
		t.Fatalf("strip backing %d bytes is not smaller than the %d-byte canvas", stripBytes, fullBytes)
	}

	if fullBytes != 1024*4040*nrgbaBytes {
		t.Fatalf("500-tile canvas bytes = %d, want %d", fullBytes, 1024*4040*nrgbaBytes)
	}
}

func TestStripRasterAppliesOnlyLargePNG(t *testing.T) {
	t.Parallel()

	large := rasterCanvasPlan{direct: true}
	small := rasterCanvasPlan{direct: false}

	if !stripRasterApplies(large, formatPNG) {
		t.Error("direct PNG should take the strip path")
	}

	if stripRasterApplies(large, formatJPG) {
		t.Error("JPEG must keep the full-canvas path")
	}

	if stripRasterApplies(small, formatPNG) {
		t.Error("small PNG must keep image/png")
	}
}

func smallStripFixture(t *testing.T) *layout.Result {
	t.Helper()

	return &layout.Result{
		Width:  float64(tileRasterSmallW) * cssPxToPt,
		Height: float64(tileRasterSmallH) * cssPxToPt,
		Ops: []layout.Op{
			cssFill(4, 4, 72, 56, 0.85, 0.9, 0.95),
			cssFill(10, 6, 20, 10, 1, 0, 0),
			cssFill(10, tileRasterStripH-2, 24, 12, 0, 0.6, 0),
			{
				Kind: layout.OpFillRect,
				X:    40 * cssPxToPt, Y: 10 * cssPxToPt,
				W: 28 * cssPxToPt, H: 28 * cssPxToPt,
				R: 0.2, G: 0.3, B: 0.8, Alpha: 1, Radius: 6 * cssPxToPt,
			},
			{
				Kind: layout.OpText,
				X:    12 * cssPxToPt, Y: 36 * cssPxToPt,
				Text: "Strip", Size: 12,
				R: 0.1, G: 0.1, B: 0.1, Alpha: 1,
			},
		},
	}
}

func publicSizeStripFixture(width, height int) *layout.Result {
	return &layout.Result{
		Width:  float64(width) * cssPxToPt,
		Height: float64(height) * cssPxToPt,
		Ops: []layout.Op{
			cssFill(8, 8, 200, 300, 0.85, 0.89, 0.93),
			cssFill(8, float64(height-80), 160, 48, 0.8, 0.2, 0.2),
			{
				Kind: layout.OpFillRect,
				X:    240 * cssPxToPt, Y: 240 * cssPxToPt,
				W: 80 * cssPxToPt, H: 40 * cssPxToPt,
				R: 0.1, G: 0.55, B: 0.3, Alpha: 1, Radius: 8 * cssPxToPt,
			},
			{
				Kind: layout.OpText,
				X:    16 * cssPxToPt, Y: 40 * cssPxToPt,
				Text: "Tile raster", Size: 14,
				R: 0.09, G: 0.2, B: 0.3, Alpha: 1,
			},
		},
	}
}

func cssFill(left, top, width, height, red, green, blue float64) layout.Op {
	return layout.Op{
		Kind: layout.OpFillRect,
		X:    left * cssPxToPt, Y: top * cssPxToPt,
		W: width * cssPxToPt, H: height * cssPxToPt,
		R: red, G: green, B: blue, Alpha: 1,
	}
}

func assertSameNRGBA(t *testing.T, got, want *image.NRGBA) {
	t.Helper()

	gotBounds := got.Bounds()
	wantBounds := want.Bounds()

	if gotBounds.Dx() != wantBounds.Dx() || gotBounds.Dy() != wantBounds.Dy() {
		t.Fatalf("tiled canvas = %dx%d, want %dx%d",
			gotBounds.Dx(), gotBounds.Dy(), wantBounds.Dx(), wantBounds.Dy())
	}

	if gotBounds != wantBounds {
		t.Fatalf("tiled bounds = %v, want %v", gotBounds, wantBounds)
	}

	if bytes.Equal(got.Pix, want.Pix) {
		return
	}

	for row := wantBounds.Min.Y; row < wantBounds.Max.Y; row++ {
		for col := wantBounds.Min.X; col < wantBounds.Max.X; col++ {
			if got.NRGBAAt(col, row) != want.NRGBAAt(col, row) {
				t.Fatalf("pixel (%d,%d) = %v, want %v", col, row, got.NRGBAAt(col, row), want.NRGBAAt(col, row))
			}
		}
	}
}
