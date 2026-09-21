package convert

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestReferenceMetrics is the operator-facing hard-metrics gate behind
// `make reference-metrics`. It converts a small fixture set through RunPDF,
// asserts the golden page envelope plus ParseSemantic needles, and logs
// the counts the PR template calls gopdfsuit hard metrics: pages, fonts,
// images, URI annots, PDF version, bytes. Detector-surface changes show
// up here as a missing needle or a page-count miss.
func TestReferenceMetrics(t *testing.T) {
	t.Parallel()

	fixtures := []string{
		"fixture-01-simple-invoice.html",
		"fixture-29-wpt-break-nested-float-print.html",
		"fixture-64-next-72-props.html",
	}

	for _, file := range fixtures {
		t.Run(file, func(t *testing.T) {
			t.Parallel()

			buf, ok := fixturePageBounds[file]
			if !ok {
				t.Fatalf("missing fixturePageBounds for %s", file)
			}

			data := runPDF(t, requestForFixture(t, file))
			assertPDFStructure(t, data)

			if !bytes.Contains(data, []byte("/FontFile2")) {
				t.Error("expected embedded subset font (/FontFile2)")
			}

			pages := pageCount(data)
			if pages < buf.minPages || (buf.maxPages > 0 && pages > buf.maxPages) {
				t.Errorf("pages = %d, want [%d, %d]", pages, buf.minPages, buf.maxPages)
			}

			doc, err := pdf.ParseSemantic(data)
			if err != nil {
				t.Fatalf("ParseSemantic: %v", err)
			}

			text := doc.DocumentText()
			pos := 0
			for _, needle := range buf.needles {
				idx := strings.Index(text[pos:], needle)
				if idx < 0 {
					t.Errorf("extracted text missing %q after offset %d", needle, pos)
					continue
				}
				pos += idx + len(needle)
			}

			fontCount := 0
			imageCount := 0
			uriCount := 0
			for _, page := range doc.Pages {
				fontCount += len(page.Fonts)
				imageCount += len(page.Images)
				for _, annot := range page.Annots {
					if annot.URI != "" {
						uriCount++
					}
				}
			}

			t.Logf(
				"pages=%d version=%s bytes=%d fonts=%d images=%d uris=%d fontfile2=%t needles=%v",
				pages, doc.Version, len(data), fontCount, imageCount, uriCount,
				bytes.Contains(data, []byte("/FontFile2")), buf.needles,
			)
		})
	}
}
