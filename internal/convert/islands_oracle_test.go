package convert

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
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

func TestCertifiedIslandsApplyGenericLinkPolicies(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<!-- report.html.tmpl: paginated benchmark report -->
<html><head><title>Benchmark report</title></head><body>
<section class="benchmark-page"><a href="https://external.test/item">external</a></section>
</body></html>`
	global := settings.DefaultPdfGlobal()
	obj := settings.DefaultPdfObject()
	obj.Page = ""
	obj.Load.InlineHTML = []byte(html)
	obj.Load.InlineBase = "https://example.test/reports/index.html"

	var genericOut, islandOut bytes.Buffer
	if err := Run(t.Context(), NewPDFRequest(global, []settings.PdfObject{obj}, &genericOut, nil), nil, nil); err != nil {
		t.Fatalf("generic: %v", err)
	}

	islandRequest := NewBenchmarkPDFRequest(global, []settings.PdfObject{obj}, &islandOut, nil)
	if err := Run(t.Context(), islandRequest, nil, nil); err != nil {
		t.Fatalf("islands: %v", err)
	}

	for name, data := range map[string][]byte{"generic": genericOut.Bytes(), "islands": islandOut.Bytes()} {
		text := string(data)
		if !strings.Contains(text, "https://external.test/item") {
			t.Errorf("%s output missing external link", name)
		}
	}
}

func TestRelativeHTMLAnchorsResolveForGenericAndCertifiedRendering(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<!-- report.html.tmpl: paginated benchmark report -->
<html><body><section class="benchmark-page"><a href="docs/item.html">relative</a> ` +
		`<a href="//cdn.example/item">protocol relative</a></section></body></html>`
	global := settings.DefaultPdfGlobal()
	obj := settings.DefaultPdfObject()
	obj.Page = ""
	obj.Load.InlineHTML = []byte(html)
	obj.Load.InlineBase = "https://example.test/reports/index.html"

	var genericOut, islandOut bytes.Buffer
	if err := Run(t.Context(), NewPDFRequest(global, []settings.PdfObject{obj}, &genericOut, nil), nil, nil); err != nil {
		t.Fatalf("convert: %v", err)
	}

	islandRequest := NewBenchmarkPDFRequest(global, []settings.PdfObject{obj}, &islandOut, nil)
	if err := Run(t.Context(), islandRequest, nil, nil); err != nil {
		t.Fatalf("certified islands: %v", err)
	}

	for name, output := range map[string]*bytes.Buffer{
		"generic": &genericOut,
		"islands": &islandOut,
	} {
		if !strings.Contains(output.String(), "https://example.test/reports/docs/item.html") {
			t.Fatalf("%s relative HTML anchor missing resolved URI annotation", name)
		}

		if !strings.Contains(output.String(), "https://cdn.example/item") {
			t.Fatalf("%s protocol-relative HTML anchor missing resolved URI annotation", name)
		}
	}
}
