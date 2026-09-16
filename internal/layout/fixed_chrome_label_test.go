package layout

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// learn-cpp.org dock (real-sites evidence 2026-09-16, learn-cpp-org row 4):
// the "Expected Output" label vanished when the position:fixed dock painted
// on more than one page, leaving an empty button bar. A crossed non-fixed
// background inserts a display-list fragment ahead of the fixed label, so the
// fixed index collected before the split pointed one op short. The fixed layer
// paints on every page, so the label text op must survive for every page.
func TestFixedFooterLabelPaintsOnEveryPage(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.filler { height: 1200pt; background: #e0e0e0; }
footer.dock { position: fixed; bottom: 0; left: 0; right: 0; background: #ffffff; }
footer.dock button { font-size: 12pt; }
`)

	res := layoutHTML(t, `<html><body><div class="filler">filler</div>`+
		`<footer class="dock"><button>LABEL-TEXT</button></footer></body></html>`, cssSheet)

	doc := pdf.NewDocument()

	// Layout viewport height 800 and paint content height 800 put the dock's
	// background and label across the first page boundary, matching the real
	// dock whose bottom sits in the page margin.
	if err := Paint(doc, res, PaintOptions{PageWidth: 595, PageHeight: 800}); err != nil {
		t.Fatal(err)
	}

	if doc.PageCount() < 2 {
		t.Fatalf("pages = %d, want at least 2 so the fixed layer repeats", doc.PageCount())
	}

	var buf bytes.Buffer

	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	sem, err := pdf.ParseSemantic(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	for p := range sem.Pages {
		if !strings.Contains(sem.Pages[p].Text, "LABEL") {
			t.Fatalf("fixed label missing on page %d; page text = %q", p, sem.Pages[p].Text)
		}
	}
}
