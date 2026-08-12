package convert //nolint:testpackage // white-box constructors

import (
	"bytes"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

func TestGenericVsCertifiedIslandsDifferentiallyEqual(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<!-- report.html.tmpl: paginated benchmark report -->
<html><head><title>Benchmark report</title>
<style>.benchmark-page + .benchmark-page { page-break-before: always; }</style>
</head>
<body>
<section class="benchmark-page"><h1>One</h1><p>alpha page</p></section>
<section class="benchmark-page"><h1>Two</h1><p>beta page</p></section>
</body></html>`

	global := settings.DefaultPdfGlobal()
	obj := settings.DefaultPdfObject()
	obj.Page = ""
	obj.Load.InlineHTML = []byte(html)

	var genericOut, islandOut bytes.Buffer

	generic := NewPDFRequest(global, []settings.PdfObject{obj}, &genericOut, nil)
	island := NewBenchmarkPDFRequest(global, []settings.PdfObject{obj}, &islandOut, nil)

	if err := Run(t.Context(), generic, nil, nil); err != nil {
		t.Fatalf("generic: %v", err)
	}

	if err := Run(t.Context(), island, nil, nil); err != nil {
		t.Fatalf("islands: %v", err)
	}

	gDoc, err := pdf.ParseSemantic(genericOut.Bytes())
	if err != nil {
		t.Fatalf("generic parse: %v", err)
	}

	iDoc, err := pdf.ParseSemantic(islandOut.Bytes())
	if err != nil {
		t.Fatalf("island parse: %v", err)
	}

	if gDoc.PageCount() != iDoc.PageCount() {
		t.Fatalf("page count generic=%d islands=%d", gDoc.PageCount(), iDoc.PageCount())
	}

	gText := gDoc.DocumentText()
	iText := iDoc.DocumentText()

	if !strings.Contains(gText, "alpha") || !strings.Contains(iText, "alpha") {
		t.Fatalf("missing alpha text generic=%q islands=%q", gText, iText)
	}

	if !strings.Contains(gText, "beta") || !strings.Contains(iText, "beta") {
		t.Fatalf("missing beta text generic=%q islands=%q", gText, iText)
	}
}
