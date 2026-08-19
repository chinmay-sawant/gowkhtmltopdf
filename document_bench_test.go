package gowkhtmltopdf_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	gowkhtmltopdf "github.com/chinmay-sawant/gowkhtmltopdf"
)

var libraryBenchmarkPageSizes = []int{ //nolint:gochecknoglobals // fixed benchmark matrix
	2, 5, 10, 20, 50, 100, 200, 250, 500,
}

var libraryBenchmarkImageSizes = []int{ //nolint:gochecknoglobals // fixed benchmark matrix
	2, 5, 10, 20, 50, 100, 200, 250, 500,
}

// BenchmarkLibraryPDF measures the public Document.WritePDF API with the same
// paginated report HTML used by the external CLI comparisons. Application-side
// template expansion happens before the timer; validation, public-to-engine
// mapping, loading, layout, painting, and PDF encoding remain inside the call.
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

			if got := bytes.Count(output.Bytes(), []byte("/Type /Page\n")); got != pageCount {
				b.Fatalf("Document.WritePDF pages = %d, want %d", got, pageCount)
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
	return gowkhtmltopdf.NewDocument(gowkhtmltopdf.Page{
		Source:           gowkhtmltopdf.HTML(libraryBenchmarkReportHTML(pageCount)),
		Header:           nil,
		Footer:           nil,
		IncludeInOutline: nil,
		ExternalLinks:    nil,
		LocalLinks:       nil,
	})
}

type libraryBenchmarkTemplateData struct {
	Pages []libraryBenchmarkPage
}

type libraryBenchmarkPage struct {
	Number int
	First  bool
	Rows   []libraryBenchmarkRow
}

type libraryBenchmarkRow struct {
	Number      int
	SKU         string
	Description string
	Quantity    int
	Amount      string
}

func libraryBenchmarkReportHTML(pageCount int) []byte {
	path := filepath.Join("testdata", "golden", "benchmarks", "templates", "report.html.tmpl")
	source, err := os.ReadFile(path)

	if err != nil {
		panic(fmt.Sprintf("read benchmark template %s: %v", path, err))
	}

	tpl, err := template.New("report.html.tmpl").Parse(string(source))
	if err != nil {
		panic(fmt.Sprintf("parse benchmark template: %v", err))
	}

	var output bytes.Buffer
	if err := tpl.Execute(&output, libraryBenchmarkTemplateData{
		Pages: libraryBenchmarkPages(pageCount),
	}); err != nil {
		panic(fmt.Sprintf("execute benchmark template: %v", err))
	}

	return output.Bytes()
}

func libraryBenchmarkPages(pageCount int) []libraryBenchmarkPage {
	pages := make([]libraryBenchmarkPage, pageCount)
	for page := range pages {
		rows := make([]libraryBenchmarkRow, 20)
		for row := range rows {
			line := row + 1
			rows[row] = libraryBenchmarkRow{
				Number:      line,
				SKU:         fmt.Sprintf("SKU-%03d-%03d", page+1, line),
				Description: fmt.Sprintf("Platform operations and support service %d", line),
				Quantity:    (line+page)%7 + 1,
				Amount:      fmt.Sprintf("%d.%02d", (page+1)*line, (page+line)%100),
			}
		}

		pages[page] = libraryBenchmarkPage{
			Number: page + 1,
			First:  page == 0,
			Rows:   rows,
		}
	}

	return pages
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
