//nolint:testpackage // white-box tests need raster internals (glyph atlas, downscale, format helpers)
package imageout

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/html"
)

// renderHTML parses src and renders it with defaults: 200 px viewport,
// backgrounds on. Tests that need more call renderHTMLOpts.
func renderHTML(t *testing.T, src string) image.Image {
	t.Helper()

	img, err := renderHTMLOpts(src, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 200, Background: true,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	return img
}

func renderHTMLOpts(src string, opts RenderOptions) (image.Image, error) {
	root, err := html.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("imageout: parse: %w", err)
	}

	if opts.Width <= 0 {
		opts.Width = 200
	}

	opts.Background = true

	return Render(root, opts)
}

// asNRGBA converts any color to color.NRGBA. The NRGBA model always yields
// NRGBA, so the checked assertion cannot fail in practice.
func asNRGBA(c color.Color) color.NRGBA {
	n, ok := color.NRGBAModel.Convert(c).(color.NRGBA)
	if !ok {
		return color.NRGBA{} //nolint:exhaustruct // intentional zero/partial fields
	}

	return n
}

// countPixels returns the number of pixels in b whose color matches want.
func countPixels(img image.Image, b image.Rectangle, want color.NRGBA) int {
	count := 0

	for row := b.Min.Y; row < b.Max.Y; row++ {
		for col := b.Min.X; col < b.Max.X; col++ {
			if asNRGBA(img.At(col, row)) == want {
				count++
			}
		}
	}

	return count
}

func redPNG(t *testing.T, w, h int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	drawSolid(img, color.NRGBA{R: 255, A: 255}) //nolint:exhaustruct // intentional zero/partial fields

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}

	return buf.Bytes()
}

func drawSolid(img *image.NRGBA, c color.NRGBA) {
	for row := img.Bounds().Min.Y; row < img.Bounds().Max.Y; row++ {
		for col := img.Bounds().Min.X; col < img.Bounds().Max.X; col++ {
			img.SetNRGBA(col, row, c)
		}
	}
}

// dataURIImages is a RenderOptions.Images that decodes data: URLs.
func dataURIImages(src string) ([]byte, error) {
	if !strings.HasPrefix(src, "data:") {
		return nil, os.ErrNotExist
	}

	rest := src[len("data:"):]

	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return nil, os.ErrNotExist
	}

	meta, data := rest[:comma], rest[comma+1:]
	if !strings.Contains(meta, "base64") {
		return nil, os.ErrNotExist
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("decode data URL: %w", err)
	}

	return decoded, nil
}

// TestRenderSolidColor checks canvas size and exact pixel colors (PNG path:
// the red fill and white background are exact).
func TestRenderSolidColor(t *testing.T) {
	t.Parallel()
	img := renderHTML(t, `<html><body><div style="background-color:#ff0000;width:100px;height:50px"></div></body></html>`)

	if got := img.Bounds().Dx(); got != 200 {
		t.Errorf("canvas width = %d, want 200", got)
	}

	if got := img.Bounds().Dy(); got < 50 {
		t.Errorf("canvas height = %d, want >= 50", got)
	}

	want := color.NRGBA{R: 255, A: 255} //nolint:exhaustruct // intentional zero/partial fields
	if count := countPixels(img, image.Rect(0, 0, 200, img.Bounds().Dy()), want); count == 0 {
		t.Error("no solid red pixels found")
	}

	if got := asNRGBA(img.At(0, 0)); got != (color.NRGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("corner pixel = %v, want white", got)
	}
}

// TestRenderTransparent checks that --transparent leaves the background
// alpha 0 while painted ops stay opaque.
func TestRenderTransparent(t *testing.T) {
	t.Parallel()

	src := `<html><body><div style="background-color:#ff0000;width:100px;height:50px"></div></body></html>`

	img, err := renderHTMLOpts(src,
		RenderOptions{Width: 200, Transparent: true}) //nolint:exhaustruct // intentional zero/partial fields
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if got := asNRGBA(img.At(0, 0)); got.A != 0 {
		t.Errorf("background alpha = %d, want 0 (transparent)", got.A)
	}

	want := color.NRGBA{R: 255, A: 255} //nolint:exhaustruct // intentional zero/partial fields
	if count := countPixels(img, image.Rect(0, 0, 200, img.Bounds().Dy()), want); count == 0 {
		t.Error("no opaque red pixels found")
	}
}

// TestRenderCrop checks the crop rect intersects the canvas: 200x100 canvas
// cropped at (50,50,150,100) yields 100x50 and preserves pixels.
func TestRenderCrop(t *testing.T) {
	t.Parallel()

	src := `<html><body><div style="background-color:#0000ff;width:120px;height:60px"></div></body></html>`

	full, err := renderHTMLOpts(src, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 200, Height: 100,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if full.Bounds().Dx() != 200 || full.Bounds().Dy() != 100 {
		t.Fatalf("full canvas = %v, want 200x100", full.Bounds())
	}

	crop := image.Rect(50, 50, 150, 100)

	img, err := renderHTMLOpts(src, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 200, Height: 100, Crop: crop,
	})
	if err != nil {
		t.Fatalf("Render crop: %v", err)
	}

	if img.Bounds().Dx() != 100 || img.Bounds().Dy() != 50 {
		t.Errorf("cropped canvas = %v, want 100x50", img.Bounds())
	}

	for _, p := range []image.Point{{0, 0}, {99, 49}, {50, 25}} {
		want := asNRGBA(full.At(p.X+50, p.Y+50))
		got := asNRGBA(img.At(p.X, p.Y))

		if got != want {
			t.Errorf("pixel %v = %v, want %v (matches uncropped)", p, got, want)
		}
	}
}

// TestRenderText checks that a bold text run paints non-background pixels.
func TestRenderText(t *testing.T) {
	t.Parallel()

	img := renderHTML(t, `<html><body><b>Hi</b></body></html>`)

	want := color.NRGBA{A: 255} //nolint:exhaustruct // intentional zero/partial fields
	if count := countPixels(img, img.Bounds(), want); count == 0 {
		t.Fatal("canvas is entirely background")
	}

	dark := 0

	for row := img.Bounds().Min.Y; row < img.Bounds().Max.Y; row++ {
		for col := img.Bounds().Min.X; col < img.Bounds().Max.X; col++ {
			c := asNRGBA(img.At(col, row))
			if c.R < 100 && c.G < 100 && c.B < 100 {
				dark++
			}
		}
	}

	if dark == 0 {
		t.Error("no dark (text) pixels found for <b>Hi</b>")
	}
}

// TestRenderImageDataURI checks an <img> with a data: PNG appears in output.
func TestRenderImageDataURI(t *testing.T) {
	t.Parallel()
	raw := redPNG(t, 16, 16)
	src := `<html><body><img src="data:image/png;base64,` +
		base64.StdEncoding.EncodeToString(raw) + `"></body></html>`

	img, err := renderHTMLOpts(src, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Images: dataURIImages,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// the image sits at the body margin (8 px) with natural size 16x16
	want := color.NRGBA{R: 255, A: 255} //nolint:exhaustruct // intentional zero/partial fields
	if count := countPixels(img, image.Rect(8, 8, 8+16, 8+16), want); count == 0 {
		t.Error("no red pixels in the <img> region")
	}
}

// TestScaleNearest checks the stdlib-free scaler preserves content.
func TestScaleNearest(t *testing.T) {
	t.Parallel()

	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255}) //nolint:exhaustruct // intentional zero/partial fields

	dst := scaleNearest(src, 4, 4)

	wantRed := color.NRGBA{R: 255, A: 255} //nolint:exhaustruct // intentional zero/partial fields
	if got := dst.At(0, 0); color.NRGBAModel.Convert(got) != wantRed {
		t.Errorf("scaled (0,0) = %v, want red", got)
	}

	wantEmpty := color.NRGBA{} //nolint:exhaustruct // intentional zero/partial fields
	if got := dst.At(3, 3); color.NRGBAModel.Convert(got) != wantEmpty {
		t.Errorf("scaled (3,3) = %v, want transparent", got)
	}
}

type imageWrapper struct {
	image.Image
}

func TestScaleNearestNRGBAMatchesGeneric(t *testing.T) {
	t.Parallel()

	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))

	for row := range 2 {
		for col := range 3 {
			src.SetNRGBA(col, row, color.NRGBA{
				R: uint8(20 + col*30),           //nolint:gosec // test pattern values stay in uint8 range
				G: uint8(40 + row*50),           //nolint:gosec // test pattern values stay in uint8 range
				B: uint8(60 + col*10 + row*5),   //nolint:gosec // test pattern values stay in uint8 range
				A: uint8(100 + col*40 + row*20), //nolint:gosec // test pattern values stay in uint8 range
			})
		}
	}

	got := scaleNearest(src, 7, 5)
	want := scaleNearestGeneric(imageWrapper{Image: src}, 7, 5)

	if string(got.Pix) != string(want.Pix) {
		t.Fatalf("NRGBA fast path differs from generic path")
	}
}

func TestDownscaleBoxUsesExactNRGBAAverages(t *testing.T) {
	t.Parallel()

	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
	src.SetNRGBA(1, 0, color.NRGBA{R: 20, G: 30, B: 40, A: 50})
	src.SetNRGBA(0, 1, color.NRGBA{R: 30, G: 40, B: 50, A: 60})
	src.SetNRGBA(1, 1, color.NRGBA{R: 40, G: 50, B: 60, A: 70})

	got := downscaleBox(src, 2)
	want := color.NRGBA{R: 25, G: 35, B: 45, A: 55}

	if pixel := got.NRGBAAt(0, 0); pixel != want {
		t.Fatalf("downscaled pixel = %v, want %v", pixel, want)
	}
}

// TestEncodeFormats checks PNG and JPEG encode with the expected dimensions.
func TestEncodeFormats(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	drawSolid(img, color.NRGBA{R: 200, G: 30, B: 30, A: 255})

	pngBytes, err := encode(img, "png", 94)
	if err != nil {
		t.Fatalf("png encode: %v", err)
	}

	jpgBytes, err := encode(img, "jpg", 94)
	if err != nil {
		t.Fatalf("jpg encode: %v", err)
	}

	if len(pngBytes) == 0 || len(jpgBytes) == 0 {
		t.Fatal("empty encoding")
	}

	dec, err := jpeg.Decode(bytes.NewReader(jpgBytes))
	if err != nil {
		t.Fatalf("jpeg decode: %v", err)
	}

	if dec.Bounds().Dx() != 64 || dec.Bounds().Dy() != 64 {
		t.Errorf("jpeg size = %v, want 64x64", dec.Bounds())
	}
}

// TestEncodeJPEGQualityChangesSize checks that --quality moves the compressed
// size (plumbing of --format/--quality) on noisy content.
func TestEncodeJPEGQualityChangesSize(t *testing.T) {
	t.Parallel()

	noise := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	rng := rand.New(rand.NewSource(7)) //nolint:gosec // deterministic test input, not security

	for row := range 80 {
		for col := range 100 {
			noise.SetNRGBA(col, row, color.NRGBA{
				R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), //nolint:gosec // rng.Intn(256) fits uint8
				B: uint8(rng.Intn(256)), //nolint:gosec // rng.Intn(256) fits uint8
				A: 255,
			})
		}
	}

	q10, err := encode(noise, "jpg", 10)
	if err != nil {
		t.Fatalf("jpg q10: %v", err)
	}

	q100, err := encode(noise, "jpg", 100)
	if err != nil {
		t.Fatalf("jpg q100: %v", err)
	}

	if len(q10) == len(q100) {
		t.Errorf("jpeg quality did not change output size (%d == %d)", len(q10), len(q100))
	}
}

// runCommand builds a parsed cli.Command pointing at input with output file.
func runCommand(t *testing.T, args ...string) *cli.Command {
	t.Helper()

	cmd, err := cli.Parse(args)
	if err != nil {
		t.Fatalf("cli.Parse(%v): %v", args, err)
	}

	cmd.Global.Load.EnableLocalFileAccess = true
	for i := range cmd.Objects {
		cmd.Objects[i].Load.BlockLocalFileAccess = false
	}

	return cmd
}

// TestRunEndToEnd drives Run through the CLI: local HTML file to PNG and to
// JPEG, checking flags (--width/--format/--quality) reach the output.
func TestRunEndToEnd(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	input := filepath.Join(dir, "in.html")
	inputHTML := `<html><body><div style="background-color:#ff0000;width:100px;height:50px"></div></body></html>`

	if err := os.WriteFile(input, []byte(inputHTML), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	pngOut := filepath.Join(dir, "out.png")

	cmd := runCommand(t, "--width", "200", "--format", "png", input, pngOut)
	if err := Run(t.Context(), cmd, io.Discard); err != nil {
		t.Fatalf("Run png: %v", err)
	}

	file, err := os.Open(pngOut)
	if err != nil {
		t.Fatalf("open png: %v", err)
	}

	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	if img.Bounds().Dx() != 200 {
		t.Errorf("png width = %d, want 200", img.Bounds().Dx())
	}

	jpgOut := filepath.Join(dir, "out.jpg")

	cmd = runCommand(t, "--width", "200", "--format", "jpg", "--quality", "30", input, jpgOut)
	if err := Run(t.Context(), cmd, io.Discard); err != nil {
		t.Fatalf("Run jpg: %v", err)
	}

	jpgFile, err := os.Open(jpgOut)
	if err != nil {
		t.Fatalf("open jpg: %v", err)
	}

	defer jpgFile.Close()

	jimg, err := jpeg.Decode(jpgFile)
	if err != nil {
		t.Fatalf("decode jpg: %v", err)
	}

	if jimg.Bounds().Dx() != 200 {
		t.Errorf("jpg width = %d, want 200", jimg.Bounds().Dx())
	}
}

// TestSmartWidth checks the viewport grows (1.5x) until fixed-width content
// fits, and stays at --width when smart width is off.
func TestSmartWidth(t *testing.T) {
	t.Parallel()

	src := `<html><body><div style="width:1500px;height:10px;background-color:#0000ff"></div></body></html>`

	smart, err := renderHTMLOpts(src, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 1024, SmartWidth: true,
	})
	if err != nil {
		t.Fatalf("Render smart: %v", err)
	}

	if got := smart.Bounds().Dx(); got != 1536 {
		t.Errorf("smart width canvas = %d, want 1536 (1024*1.5)", got)
	}

	fixed, err := renderHTMLOpts(src, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 1024, SmartWidth: false,
	})
	if err != nil {
		t.Fatalf("Render fixed: %v", err)
	}

	if got := fixed.Bounds().Dx(); got != 1024 {
		t.Errorf("fixed canvas = %d, want 1024", got)
	}

	defRoot, err := html.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	def, err := Render(defRoot, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Width: 0, SmartWidth: false, Background: true,
	})
	if err != nil {
		t.Fatalf("Render default: %v", err)
	}

	if got := def.Bounds().Dx(); got != 1024 {
		t.Errorf("default canvas = %d, want 1024", got)
	}
}
