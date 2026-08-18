package gowkhtmltopdf_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	gowkhtmltopdf "github.com/chinmay-sawant/gowkhtmltopdf"
)

var libraryBenchmarkPageSizes = []int{ //nolint:gochecknoglobals // fixed benchmark matrix
	2, 5, 10, 20, 50, 100, 200, 250, 500,
}

var libraryBenchmarkImageSizes = []int{ //nolint:gochecknoglobals // fixed benchmark matrix
	2, 5, 10, 20, 50, 100, 200, 250, 500,
}

// BenchmarkLibraryPDF measures the public Document.WritePDF API with one
// in-memory HTML source per public body page. Application-side HTML creation
// happens before the timer; validation, public-to-engine mapping, loading,
// layout, painting, and PDF encoding remain inside the measured call.
func BenchmarkLibraryPDF(b *testing.B) {
	for _, pageCount := range libraryBenchmarkPageSizes {
		b.Run(fmt.Sprintf("%dPages", pageCount), func(b *testing.B) {
			document := libraryBenchmarkPDFDocument(pageCount)

			var output bytes.Buffer

			b.ReportMetric(float64(pageCount), "pages")
			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := document.WritePDF(b.Context(), &output); err != nil {
					b.Fatalf("Document.WritePDF: %v", err)
				}
			}

			b.StopTimer()

			if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
				b.Fatalf("Document.WritePDF output is not a PDF")
			}

			b.SetBytes(int64(output.Len()))
		})
	}
}

// BenchmarkLibraryImage measures the public ImageDocument.WriteImage API with
// in-memory HTML containing a controlled number of rasterized tiles.
func BenchmarkLibraryImage(b *testing.B) {
	for _, tileCount := range libraryBenchmarkImageSizes {
		b.Run(fmt.Sprintf("%dTiles", tileCount), func(b *testing.B) {
			document := libraryBenchmarkImageDocument(tileCount)

			var output bytes.Buffer

			b.ReportMetric(float64(tileCount), "tiles")
			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := document.WriteImage(b.Context(), &output); err != nil {
					b.Fatalf("ImageDocument.WriteImage: %v", err)
				}
			}

			b.StopTimer()

			if !bytes.HasPrefix(output.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
				b.Fatalf("ImageDocument.WriteImage output is not a PNG")
			}

			b.SetBytes(int64(output.Len()))
		})
	}
}

func libraryBenchmarkPDFDocument(pageCount int) *gowkhtmltopdf.Document {
	pages := make([]gowkhtmltopdf.Page, pageCount)
	for index := range pages {
		pages[index] = gowkhtmltopdf.Page{
			Source:           gowkhtmltopdf.HTML(libraryBenchmarkPageHTML(index + 1)),
			Header:           nil,
			Footer:           nil,
			IncludeInOutline: nil,
			ExternalLinks:    nil,
			LocalLinks:       nil,
		}
	}

	return gowkhtmltopdf.NewDocument(pages...)
}

func libraryBenchmarkPageHTML(page int) []byte {
	var source strings.Builder

	source.WriteString(`<!doctype html><html><head><meta charset="utf-8"><style>
body { font-family: sans-serif; font-size: 11pt; margin: 24mm; }
h1 { color: #17324d; margin: 0 0 12pt; }
.row { border-bottom: 1px solid #d9e2ec; padding: 5pt 0; }
</style></head><body>`)
	fmt.Fprintf(&source, "<h1>Library benchmark page %d</h1>", page)

	for row := 1; row <= 12; row++ {
		fmt.Fprintf(
			&source,
			"<div class=\"row\">Item %d: renderer input, layout, and PDF output work</div>",
			row,
		)
	}

	source.WriteString("</body></html>")

	return []byte(source.String())
}

func libraryBenchmarkImageDocument(tileCount int) *gowkhtmltopdf.ImageDocument {
	var source strings.Builder

	source.WriteString(`<!doctype html><html><head><meta charset="utf-8"><style>
body { margin: 0; font-family: sans-serif; }
.grid { display: flex; flex-wrap: wrap; gap: 8px; padding: 8px; }
.tile { width: 120px; height: 56px; background: #d9e2ec; color: #17324d;
  border: 1px solid #52718d; padding: 8px; box-sizing: border-box; }
</style></head><body><div class="grid">`)

	for tile := 1; tile <= tileCount; tile++ {
		fmt.Fprintf(&source, "<div class=\"tile\">Tile %d</div>", tile)
	}

	source.WriteString("</div></body></html>")

	return &gowkhtmltopdf.ImageDocument{
		Source:          gowkhtmltopdf.HTML([]byte(source.String())),
		Width:           1024,
		Height:          512,
		Format:          "png",
		Quality:         0,
		SmartWidth:      nil,
		Transparent:     false,
		Crop:            nil,
		AllowLocalFiles: false,
		Background:      nil,
		FontPaths:       nil,
		UseSystemFonts:  false,
		Network:         nil,
		Now:             nil,
		OnInfo:          nil,
		OnWarn:          nil,
		OnError:         nil,
		OnPhase:         nil,
		OnProgress:      nil,
	}
}
