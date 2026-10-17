package blinkengine

import (
	"bytes"
	"image"
	_ "image/png"
	"image/color"
	"image/png"
	"testing"
)

// TestLayoutMapsMaskBorderFrameRaster proves the mask-border bridge: an
// element with a mask-border source composes a nine-slice frame raster on its
// group, the bridge copies it to the host mask descriptor, and the raster
// carries an opaque frame band with a transparent center (the fill-less
// default), which is what the PDF painter turns into a soft mask.
func TestLayoutMapsMaskBorderFrameRaster(t *testing.T) {
	t.Parallel()

	frame := image.NewRGBA(image.Rect(0, 0, 60, 60))

	for y := range 60 {
		for x := range 60 {
			if x < 15 || x >= 45 || y < 15 || y >= 45 {
				frame.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}

	var frameBuf bytes.Buffer
	if err := png.Encode(&frameBuf, frame); err != nil {
		t.Fatalf("encode frame: %v", err)
	}

	source := []byte(`<!DOCTYPE html><html><head><style>
body { margin: 0 }
.framed { width: 120px; height: 60px; background: #1f4b99;
          mask-border-source: url(frame.png); mask-border-slice: 15; mask-border-width: 12px }
</style></head><body>
<div class="framed"></div>
</body></html>`)

	res, err := Layout(t.Context(), source, Options{
		WidthPt:  538.6,
		HeightPt: 785.2,
		Media:    "print",
		Images: func(src string) ([]byte, error) {
			if src == "frame.png" {
				return frameBuf.Bytes(), nil
			}

			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	var mask []byte

	for _, op := range res.Ops {
		group := op.Group()
		if group != nil && group.Mask != nil && len(group.Mask.PNGBytes) > 0 {
			mask = group.Mask.PNGBytes

			break
		}
	}

	if len(mask) == 0 {
		t.Fatal("no group carries a mask-border frame raster")
	}

	img, _, err := image.Decode(bytes.NewReader(mask))
	if err != nil {
		t.Fatalf("decode frame raster: %v", err)
	}

	// The frame band is opaque; the center stays transparent without fill.
	_, _, _, cornerAlpha := img.At(1, 1).RGBA()
	if cornerAlpha == 0 {
		t.Error("frame corner is transparent, want the opaque frame band")
	}

	_, _, _, centerAlpha := img.At(img.Bounds().Dx()/2, img.Bounds().Dy()/2).RGBA()
	if centerAlpha != 0 {
		t.Errorf("frame center alpha = %d, want 0 without the fill flag", centerAlpha)
	}
}
