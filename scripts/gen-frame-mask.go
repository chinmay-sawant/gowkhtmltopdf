package main

// One-off generator for testdata/golden/frame-mask.png: a 64x64 nine-slice
// mask with an opaque patterned frame band and a fully transparent center, so
// a mask-border/-webkit-mask-box-image frame is visible in rendered output.
// Run from the repo root: go run scripts/gen-frame-mask.go
import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

func main() {
	const side = 64
	const band = 16

	img := image.NewRGBA(image.Rect(0, 0, side, side))

	for y := range side {
		for x := range side {
			inFrame := x < band || x >= side-band || y < band || y >= side-band
			if !inFrame {
				continue
			}

			// A light diagonal stripe pattern keeps the frame visually
			// distinct from a plain border when masked in.
			stripe := ((x + y) / 8) % 2

			c := color.RGBA{R: 255, G: 255, B: 255, A: 255}
			if stripe == 1 {
				c = color.RGBA{R: 200, G: 220, B: 255, A: 255}
			}

			img.Set(x, y, c)
		}
	}

	f, err := os.Create("testdata/golden/frame-mask.png")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}
