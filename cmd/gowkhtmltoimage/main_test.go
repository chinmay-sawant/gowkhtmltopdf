package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
)

func TestRunHTMLToPNGUsesExplicitOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")
	input := `<!DOCTYPE html><style>
#header-area { width: 100%; display: flex; flex-direction: column; align-items: center; }
#header-title { background-color: #e63946; padding: 2px; }
#header-subtitle { background-color: #457b9d; padding: 2px; }
</style><html><head><title>Test</title></head><body>
<div id="header-area"><div id="header-title">Test Header</div><div id="header-subtitle">Test Subheader</div></div>
</body></html>`
	code := run([]string{
		"--quiet",
		"--format", "png",
		"--html", input,
		"--output", output,
	})

	if code != cli.ExitOK {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitOK)
	}

	data, err := os.ReadFile(output)

	if err != nil {
		t.Fatalf("ReadFile(%q): %v", output, err)
	}

	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("output prefix = %q, want PNG magic", data[:min(len(data), 16)])
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}

	center := float64(img.Bounds().Min.X+img.Bounds().Max.X) / 2

	for _, testCase := range []struct {
		name string
		want color.NRGBA
	}{
		{name: "title", want: color.NRGBA{R: 230, G: 57, B: 70, A: 255}},
		{name: "subtitle", want: color.NRGBA{R: 69, G: 123, B: 157, A: 255}},
	} {
		bounds := exactColorBounds(img, testCase.want)
		if bounds.Empty() {
			t.Fatalf("%s background was not painted", testCase.name)
		}

		if bounds.Dx() >= img.Bounds().Dx()/2 {
			t.Fatalf("%s width = %d, want a non-stretched child", testCase.name, bounds.Dx())
		}

		childCenter := float64(bounds.Min.X+bounds.Max.X) / 2
		if math.Abs(childCenter-center) > 1 {
			t.Fatalf("%s center x = %.2f, want %.2f", testCase.name, childCenter, center)
		}
	}
}

func TestRunRequiresExplicitOutput(t *testing.T) {
	t.Parallel()

	if code := run([]string{"--html", "<html><body>missing output</body></html>"}); code != cli.ExitError {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitError)
	}
}

func exactColorBounds(img image.Image, want color.NRGBA) image.Rectangle {
	minX, minY := img.Bounds().Max.X, img.Bounds().Max.Y
	maxX, maxY := img.Bounds().Min.X, img.Bounds().Min.Y
	found := false

	for row := img.Bounds().Min.Y; row < img.Bounds().Max.Y; row++ {
		for col := img.Bounds().Min.X; col < img.Bounds().Max.X; col++ {
			got, ok := color.NRGBAModel.Convert(img.At(col, row)).(color.NRGBA)
			if !ok || got != want {
				continue
			}

			found = true
			minX = min(minX, col)
			minY = min(minY, row)
			maxX = max(maxX, col)
			maxY = max(maxY, row)
		}
	}

	if !found {
		return image.Rectangle{}
	}

	return image.Rect(minX, minY, maxX+1, maxY+1)
}
