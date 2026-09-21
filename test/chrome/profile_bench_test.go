package chrome_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// BenchmarkChromeCasePDFs renders every Chrome interaction case through the
// full HTML to PDF pipeline with the same request shape as
// TestChromeCasePDFOutputs. It exists so the 40-case corpus can be profiled
// with the allocation views (alloc_space, alloc_objects) and read alongside
// the golden benchmark matrix; by itself it only asserts that each case still
// converts and parses.
//
// Profile recipe:
//
//	go test -c -o /tmp/chrome.test ./test/chrome
//	cd test/chrome && /tmp/chrome.test -test.run '^$' \
//	  -test.bench '^BenchmarkChromeCasePDFs$' -test.benchtime=1x -test.count=10 \
//	  -test.benchmem -test.memprofile=/tmp/chrome-mem.pprof -test.memprofilerate=65536
func BenchmarkChromeCasePDFs(b *testing.B) {
	manifest := readManifest(b)

	for _, item := range manifest.Cases {
		b.Run(item.ID, func(b *testing.B) {
			var output bytes.Buffer

			request := chromeCasePDFRequest(b, item, &output)

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := convert.Run(b.Context(), request, io.Discard, nil); err != nil {
					b.Fatalf("convert case %q: %v", item.ID, err)
				}
			}

			b.StopTimer()

			document, err := pdf.ParseSemantic(output.Bytes())
			if err != nil {
				b.Fatalf("parse generated PDF for %q: %v", item.ID, err)
			}

			b.ReportMetric(float64(document.PageCount()), "pages")
			b.ReportMetric(float64(output.Len()), "output-bytes")
		})
	}
}
