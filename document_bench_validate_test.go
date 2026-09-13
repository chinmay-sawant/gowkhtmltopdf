package gowkhtmltopdf_test

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	libraryTestTinyDimension = 64
	libraryTestFirstNeedle   = "SKU-001-001"
	libraryTestLastNeedle    = "SKU-002-020"
	semanticTestPageWidth    = 612
	semanticTestPageHeight   = 792
	semanticTestTextX        = 40
	semanticTestTextY        = 740
	semanticTestFontSize     = 12
)

// TestValidateLibraryPDFOutputAcceptsBenchmarkOutput proves the PDF validator
// accepts the real public-library workload the benchmark measures.
func TestValidateLibraryPDFOutputAcceptsBenchmarkOutput(t *testing.T) {
	t.Parallel()

	document := libraryBenchmarkPDFDocument(2)

	var output bytes.Buffer

	if err := document.WritePDF(t.Context(), &output); err != nil {
		t.Fatalf("Document.WritePDF: %v", err)
	}

	if err := validateLibraryPDFOutput(output.Bytes(), 2); err != nil {
		t.Fatalf("validateLibraryPDFOutput: %v", err)
	}
}

func TestValidateLibraryPDFOutputRejectsInvalid(t *testing.T) {
	t.Parallel()

	t.Run("not a pdf", func(t *testing.T) {
		t.Parallel()

		if err := validateLibraryPDFOutput([]byte("not a pdf"), 2); err == nil {
			t.Fatal("validateLibraryPDFOutput accepted non-PDF bytes")
		}
	})

	t.Run("truncated pdf", func(t *testing.T) {
		t.Parallel()

		data := semanticTestPDF(t, []string{libraryTestFirstNeedle, libraryTestLastNeedle})

		startxref := bytes.Index(data, []byte("startxref"))
		if startxref < 0 {
			t.Fatal("test PDF has no startxref marker")
		}

		if err := validateLibraryPDFOutput(data[:startxref], 2); err == nil {
			t.Fatal("validateLibraryPDFOutput accepted a truncated PDF")
		}
	})

	t.Run("wrong page count", func(t *testing.T) {
		t.Parallel()

		data := semanticTestPDF(t, []string{libraryTestFirstNeedle})

		if err := validateLibraryPDFOutput(data, 2); err == nil {
			t.Fatal("validateLibraryPDFOutput accepted 1 page when 2 were requested")
		}
	})

	t.Run("wrong text order", func(t *testing.T) {
		t.Parallel()

		data := semanticTestPDF(t, []string{libraryTestLastNeedle, libraryTestFirstNeedle})

		if err := validateLibraryPDFOutput(data, 2); err == nil {
			t.Fatal("validateLibraryPDFOutput accepted reversed page needles")
		}
	})

	t.Run("missing needles", func(t *testing.T) {
		t.Parallel()

		data := semanticTestPDF(t, []string{"alpha", "beta"})

		if err := validateLibraryPDFOutput(data, 2); err == nil {
			t.Fatal("validateLibraryPDFOutput accepted missing report text")
		}
	})
}

// TestValidateLibraryImageOutputAcceptsBenchmarkOutput proves the image
// validator accepts the real public-library workload at a filter row. Fifty
// tiles also exercise the trailing partial grid row.
func TestValidateLibraryImageOutputAcceptsBenchmarkOutput(t *testing.T) {
	t.Parallel()

	document := libraryBenchmarkImageDocument(50)

	var output bytes.Buffer

	if err := document.WriteImage(t.Context(), &output); err != nil {
		t.Fatalf("ImageDocument.WriteImage: %v", err)
	}

	if err := validateLibraryImageOutput(output.Bytes(), 50); err != nil {
		t.Fatalf("validateLibraryImageOutput: %v", err)
	}
}

func TestValidateLibraryImageOutputRejectsInvalid(t *testing.T) {
	t.Parallel()

	t.Run("not a png", func(t *testing.T) {
		t.Parallel()

		data := semanticTestPDF(t, []string{"alpha"})

		if err := validateLibraryImageOutput(data, 2); err == nil {
			t.Fatal("validateLibraryImageOutput accepted non-PNG bytes")
		}
	})

	t.Run("wrong width", func(t *testing.T) {
		t.Parallel()

		data := encodeTestPNG(t, libraryTestTinyDimension, libraryTestTinyDimension, nil)

		if err := validateLibraryImageOutput(data, 2); err == nil {
			t.Fatal("validateLibraryImageOutput accepted a wrong-width image")
		}
	})

	t.Run("wrong height", func(t *testing.T) {
		t.Parallel()

		data := encodeTestPNG(t, libraryBenchmarkImageWidth, libraryTestTinyDimension, nil)

		if err := validateLibraryImageOutput(data, 2); err == nil {
			t.Fatal("validateLibraryImageOutput accepted a wrong-height image")
		}
	})

	t.Run("transparent pixel", func(t *testing.T) {
		t.Parallel()

		data := encodeTestPNG(t, libraryBenchmarkImageWidth, libraryBenchmarkMinImageHeight, func(img *image.NRGBA) {
			img.SetNRGBA(0, 0, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0})
		})

		if err := validateLibraryImageOutput(data, 2); err == nil {
			t.Fatal("validateLibraryImageOutput accepted transparency with Transparent=false")
		}
	})

	t.Run("too few tiles", func(t *testing.T) {
		t.Parallel()

		if err := validateLibraryImageOutput(missingTilePNG(t), 2); err == nil {
			t.Fatal("validateLibraryImageOutput accepted an image with a missing tile")
		}
	})
}

// missingTilePNG draws exactly one background-colored tile at the grid origin,
// so the validator must report the missing tiles.
func missingTilePNG(t *testing.T) []byte {
	t.Helper()

	tile := image.Rect(
		libraryBenchmarkGridPadding,
		libraryBenchmarkGridPadding,
		libraryBenchmarkGridPadding+libraryBenchmarkTileWidth,
		libraryBenchmarkGridPadding+libraryBenchmarkTileHeight,
	)
	background := color.NRGBA{
		R: libraryBenchmarkBackgroundR,
		G: libraryBenchmarkBackgroundG,
		B: libraryBenchmarkBackgroundB,
		A: 0xff,
	}

	return encodeTestPNG(t, libraryBenchmarkImageWidth, libraryBenchmarkMinImageHeight, func(img *image.NRGBA) {
		draw.Draw(img, tile, &image.Uniform{C: background}, image.Point{}, draw.Src)
	})
}

// encodeTestPNG builds a PNG fixture on an opaque white canvas. paint may
// override regions after the white fill; a nil paint keeps the canvas white.
func encodeTestPNG(tb testing.TB, width, height int, paint func(*image.NRGBA)) []byte {
	tb.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: white}, image.Point{}, draw.Src)

	if paint != nil {
		paint(img)
	}

	var output bytes.Buffer

	if err := png.Encode(&output, img); err != nil {
		tb.Fatalf("encode test PNG: %v", err)
	}

	return output.Bytes()
}

// semanticTestPDF builds a real PDF through the internal writer so validation
// tests can exercise page count and text-order failures without running the
// HTML pipeline. Each page shows one text run.
func semanticTestPDF(tb testing.TB, pageTexts []string) []byte {
	tb.Helper()

	font, err := pdf.DefaultFont()
	if err != nil {
		tb.Fatalf("pdf.DefaultFont: %v", err)
	}

	document := pdf.NewDocument()

	for _, text := range pageTexts {
		page := document.AddPage(semanticTestPageWidth, semanticTestPageHeight)
		content := page.Content()
		content.UseEmbeddedFont("F1", font)
		content.BeginText()
		content.SetFont("F1", semanticTestFontSize)
		content.TextAt(semanticTestTextX, semanticTestTextY)
		content.TextShow(text)
		content.EndText()
	}

	var output bytes.Buffer

	if err := document.Write(&output); err != nil {
		tb.Fatalf("write semantic test PDF: %v", err)
	}

	return output.Bytes()
}
