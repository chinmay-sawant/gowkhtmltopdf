package convert_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// headerFooterBenchmarkHTML makes page boundaries explicit so the benchmark
// measures placeholder substitution and per-page header/footer painting over
// a stable page-count workload.
func headerFooterBenchmarkHTML(pages int) []byte {
	var source strings.Builder

	const prefix = `<html><body style="font-family: sans-serif; font-size: 12pt">`

	const section = `<section style="height: 650pt; page-break-after: always">` +
		`<h1>Body page %d</h1><p>Stable header/footer benchmark content.</p></section>`

	source.WriteString(prefix)

	for page := 1; page <= pages; page++ {
		_, _ = fmt.Fprintf(&source, section, page)
	}

	source.WriteString(`</body></html>`)

	return []byte(source.String())
}

func BenchmarkHeaderFooterPlaceholders(b *testing.B) {
	for _, pages := range []int{2, 10, 50} {
		b.Run(fmt.Sprintf("%dPages", pages), func(b *testing.B) {
			global := settings.DefaultPdfGlobal()
			global.Quiet = true
			global.Header.Center = "Page [page] of [topage]"
			global.Footer.Right = "from [frompage]"
			object := settings.DefaultPdfObject()
			object.Load.InlineHTML = headerFooterBenchmarkHTML(pages)

			var output bytes.Buffer
			req := convert.NewPDFRequest(global, []settings.PdfObject{object}, &output, nil)

			b.ReportMetric(float64(pages), "requested_pages")

			b.ResetTimer()

			for range b.N {
				output.Reset()

				err := convert.Run(b.Context(), req, io.Discard, nil)
				if err != nil {
					b.Fatalf("header/footer conversion: %v", err)
				}
			}

			b.StopTimer()
			b.SetBytes(int64(output.Len()))
			b.ReportMetric(float64(benchmarkPDFPageCount(output.Bytes())), "actual_pages")
		})
	}
}

func TestHeaderFooterBenchmarkPageCounts(t *testing.T) {
	t.Parallel()

	for _, pages := range []int{2, 10} {
		global := settings.DefaultPdfGlobal()
		global.Quiet = true
		global.Header.Center = "Page [page] of [topage]"
		object := settings.DefaultPdfObject()
		object.Load.InlineHTML = headerFooterBenchmarkHTML(pages)

		var output bytes.Buffer
		req := convert.NewPDFRequest(global, []settings.PdfObject{object}, &output, nil)

		if err := convert.Run(t.Context(), req, io.Discard, nil); err != nil {
			t.Fatalf("header/footer conversion for %d pages: %v", pages, err)
		}

		got := benchmarkPDFPageCount(output.Bytes())
		if got != pages {
			t.Fatalf("page count for %d-page workload = %d", pages, got)
		}
	}
}
