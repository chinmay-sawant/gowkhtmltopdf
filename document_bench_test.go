package gowkhtmltopdf_test

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	gowkhtmltopdf "github.com/chinmay-sawant/gowkhtmltopdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
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

			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := document.WritePDF(b.Context(), &output); err != nil {
					b.Fatalf("Document.WritePDF: %v", err)
				}
			}

			b.StopTimer()

			if err := validateLibraryPDFOutput(output.Bytes(), pageCount); err != nil {
				b.Fatal(err)
			}

			// ResetTimer deletes metrics reported before it, so the page
			// count and output bytes are reported after the timed loop.
			b.ReportMetric(float64(pageCount), "pages")
			b.ReportMetric(float64(output.Len()), "output-bytes")
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

			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := document.WriteImage(b.Context(), &output); err != nil {
					b.Fatalf("ImageDocument.WriteImage: %v", err)
				}
			}

			b.StopTimer()

			if err := validateLibraryImageOutput(output.Bytes(), tileCount); err != nil {
				b.Fatal(err)
			}

			img, _, err := image.Decode(bytes.NewReader(output.Bytes()))
			if err != nil {
				b.Fatalf("decode rendered image: %v", err)
			}

			bounds := img.Bounds()

			// ResetTimer deletes metrics reported before it, so the tile
			// count, output bytes, and dimensions are reported after the
			// timed loop.
			b.ReportMetric(float64(tileCount), "tiles")
			b.ReportMetric(float64(output.Len()), "output-bytes")
			b.ReportMetric(float64(bounds.Dx()), "width")
			b.ReportMetric(float64(bounds.Dy()), "height")
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
		Zoom:             0,
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
		Zoom:            0,
		Allow:           nil,
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

// Library benchmark acceptance helpers. The public benchmarks validate their
// output through these functions, so a conversion that silently stops
// rendering pages, ordered text, image dimensions, opacity, or tiles fails
// the benchmark that measured it instead of publishing a fast wrong row.
const (
	libraryBenchmarkRowsPerPage = 20
	// libraryBenchmarkHeadingNeedle is the report heading shared by every
	// generated page. The template's em dash does not survive extraction with
	// the default font, so only the leading words are stable needles.
	libraryBenchmarkHeadingNeedle = "Benchmark report"
	libraryBenchmarkImageWidth    = 1024
	libraryBenchmarkTileWidth     = 120
	libraryBenchmarkTileHeight    = 56
	libraryBenchmarkTileGap       = 8
	libraryBenchmarkGridPadding   = 8
	// libraryBenchmarkTilesPerRow is the measured column count for the
	// 1024px grid. Eight 120px tiles plus seven 8px gaps overflow the 1008px
	// content box by 8px, so flex shrink trims every tile in a full row to
	// 119px (pitch 127px). Rows with fewer than eight tiles keep the unshrunk
	// 120px width and 128px pitch. Measured on 2026-09-11 with go1.26.4; the
	// geometry comes from libraryBenchmarkImageDocument's CSS above.
	libraryBenchmarkTilesPerRow      = 8
	libraryBenchmarkTilePitchFullRow = 127
	libraryBenchmarkTileWidthFullRow = 119
	libraryBenchmarkMinImageHeight   = 512
	// libraryBenchmarkMinTilePixels is far below the measured 6100+ exact
	// background pixels per rendered tile, but well above text coverage, so a
	// missing or misplaced tile fails while font fallback does not.
	libraryBenchmarkMinTilePixels = 4000
	libraryBenchmarkBackgroundR   = 0xd9
	libraryBenchmarkBackgroundG   = 0xe2
	libraryBenchmarkBackgroundB   = 0xec
)

// errLibraryBenchmarkValidation marks benchmark output that failed its
// semantic acceptance checks. Wrapping it keeps errors.Is usable while the
// message names the exact field that regressed.
var errLibraryBenchmarkValidation = errors.New("library benchmark validation failed")

// validateLibraryPDFOutput accepts a PDF only when it has the PDF header,
// exactly pageCount physical pages, the report heading on the first and last
// page, and the page-1 and last-page SKU needles in document order.
// libraryBenchmarkPages renders 20 rows per page, so SKU-001-001 exists only
// on page 1 and SKU-<pageCount>-020 only on the last page.
func validateLibraryPDFOutput(data []byte, pageCount int) error {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return fmt.Errorf("%w: output is missing the %%PDF- header", errLibraryBenchmarkValidation)
	}

	doc, err := pdf.ParseSemantic(data)
	if err != nil {
		return fmt.Errorf("%w: parse semantic PDF: %w", errLibraryBenchmarkValidation, err)
	}

	if doc.PageCount() != pageCount {
		return fmt.Errorf(
			"%w: page count = %d, want %d",
			errLibraryBenchmarkValidation, doc.PageCount(), pageCount,
		)
	}

	if pageCount == 0 {
		return fmt.Errorf("%w: no pages to validate", errLibraryBenchmarkValidation)
	}

	return validateLibraryPDFNeedles(doc, pageCount)
}

// validateLibraryPDFNeedles requires the report heading on the first and last
// page and the page-1 and last-page SKU needles in document order.
func validateLibraryPDFNeedles(doc *pdf.SemanticDoc, pageCount int) error {
	firstPageText := doc.Pages[0].Text
	lastPageText := doc.Pages[doc.PageCount()-1].Text

	if !strings.Contains(firstPageText, libraryBenchmarkHeadingNeedle) {
		return fmt.Errorf("%w: first page is missing the report heading", errLibraryBenchmarkValidation)
	}

	if !strings.Contains(lastPageText, libraryBenchmarkHeadingNeedle) {
		return fmt.Errorf("%w: last page is missing the report heading", errLibraryBenchmarkValidation)
	}

	firstNeedle := "SKU-001-001"
	lastNeedle := fmt.Sprintf("SKU-%03d-%03d", pageCount, libraryBenchmarkRowsPerPage)

	if !strings.Contains(firstPageText, firstNeedle) {
		return fmt.Errorf("%w: first page is missing %q", errLibraryBenchmarkValidation, firstNeedle)
	}

	if !strings.Contains(lastPageText, lastNeedle) {
		return fmt.Errorf("%w: last page is missing %q", errLibraryBenchmarkValidation, lastNeedle)
	}

	text := doc.DocumentText()

	firstAt := strings.Index(text, firstNeedle)
	lastAt := strings.Index(text, lastNeedle)

	if firstAt < 0 || lastAt < 0 || lastAt <= firstAt {
		return fmt.Errorf(
			"%w: text needles are not in order (%q at %d, %q at %d)",
			errLibraryBenchmarkValidation, firstNeedle, firstAt, lastNeedle, lastAt,
		)
	}

	return nil
}

// validateLibraryImageOutput accepts a PNG only when it decodes, has the
// requested width and the tiled-grid height for tileCount, is fully opaque
// (the benchmark document sets Transparent=false), and renders exactly
// tileCount tiles with no extra tile in the grid's trailing slots.
func validateLibraryImageOutput(data []byte, tileCount int) error {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: decode PNG: %w", errLibraryBenchmarkValidation, err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != libraryBenchmarkImageWidth {
		return fmt.Errorf(
			"%w: image width = %d, want %d",
			errLibraryBenchmarkValidation, bounds.Dx(), libraryBenchmarkImageWidth,
		)
	}

	if want := expectedLibraryImageHeight(tileCount); bounds.Dy() != want {
		return fmt.Errorf(
			"%w: image height = %d, want %d for %d tiles",
			errLibraryBenchmarkValidation, bounds.Dy(), want, tileCount,
		)
	}

	// Normalise to NRGBA once so the opacity check and the tile scan read
	// straight from Pix instead of paying the image.Image interface cost.
	raster := image.NewNRGBA(bounds)
	draw.Draw(raster, raster.Bounds(), img, bounds.Min, draw.Src)

	for offset := 3; offset < len(raster.Pix); offset += 4 {
		if raster.Pix[offset] != 0xff {
			return fmt.Errorf(
				"%w: transparent pixel at byte offset %d with Transparent=false",
				errLibraryBenchmarkValidation, offset,
			)
		}
	}

	return validateLibraryImageTiles(raster, tileCount)
}

// expectedLibraryImageHeight returns the PNG height for tileCount tiles: the
// larger of the benchmark document's 512px minimum and the grid content
// height. Each grid row is 56px tall with an 8px gap and 8px padding at the
// top and bottom, so content height is 64*rows + 8.
func expectedLibraryImageHeight(tileCount int) int {
	rows := (tileCount + libraryBenchmarkTilesPerRow - 1) / libraryBenchmarkTilesPerRow

	contentHeight := 2*libraryBenchmarkGridPadding +
		rows*libraryBenchmarkTileHeight +
		(rows-1)*libraryBenchmarkTileGap

	if contentHeight < libraryBenchmarkMinImageHeight {
		return libraryBenchmarkMinImageHeight
	}

	return contentHeight
}

// validateLibraryImageTiles counts exact tile-background pixels inside every
// expected tile interior and requires a fully rendered tile at each occupied
// slot plus no tile background in the trailing empty slots. That makes the
// check an exact tile count, not just "some tiles were drawn".
func validateLibraryImageTiles(raster *image.NRGBA, tileCount int) error {
	rows := (tileCount + libraryBenchmarkTilesPerRow - 1) / libraryBenchmarkTilesPerRow

	for row := range rows {
		items := min(libraryBenchmarkTilesPerRow, tileCount-row*libraryBenchmarkTilesPerRow)

		pitch := libraryBenchmarkTileWidth + libraryBenchmarkTileGap
		tileWidth := libraryBenchmarkTileWidth

		if items == libraryBenchmarkTilesPerRow {
			pitch = libraryBenchmarkTilePitchFullRow
			tileWidth = libraryBenchmarkTileWidthFullRow
		}

		for column := range libraryBenchmarkTilesPerRow {
			x0 := libraryBenchmarkGridPadding + column*pitch
			y0 := libraryBenchmarkGridPadding + row*(libraryBenchmarkTileHeight+libraryBenchmarkTileGap)

			count := libraryTileBackgroundPixels(raster, x0+1, y0+1, x0+tileWidth-1, y0+libraryBenchmarkTileHeight-1)

			if column < items {
				if count < libraryBenchmarkMinTilePixels {
					return fmt.Errorf(
						"%w: tile %d has %d background pixels, want >= %d",
						errLibraryBenchmarkValidation,
						row*libraryBenchmarkTilesPerRow+column+1,
						count,
						libraryBenchmarkMinTilePixels,
					)
				}

				continue
			}

			if count > 0 {
				return fmt.Errorf(
					"%w: unexpected tile background in empty slot row %d column %d",
					errLibraryBenchmarkValidation, row+1, column+1,
				)
			}
		}
	}

	return nil
}

// libraryTileBackgroundPixels counts pixels equal to the tile background
// inside [x0,x1) by [y0,y1). The caller has already proved the raster opaque,
// so only the RGB channels are compared.
func libraryTileBackgroundPixels(raster *image.NRGBA, x0, y0, x1, y1 int) int {
	count := 0

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			offset := raster.PixOffset(x, y)
			if raster.Pix[offset] == libraryBenchmarkBackgroundR &&
				raster.Pix[offset+1] == libraryBenchmarkBackgroundG &&
				raster.Pix[offset+2] == libraryBenchmarkBackgroundB {
				count++
			}
		}
	}

	return count
}
