package layout

import (
	"strconv"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestTheadRepeatOnContinuationPages(t *testing.T) {
	src := `<html><body>
<table>
<thead><tr><th>ColA</th><th>ColB</th></tr></thead>
<tbody>`
	for i := 0; i < 40; i++ {
		src += `<tr><td>r` + strconv.Itoa(i) + `</td><td>body row ` + strconv.Itoa(i) + `</td></tr>`
	}
	src += `</tbody></table></body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 400, Height: 200, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: 500, PageHeight: 280,
		MarginTop: 20, MarginBottom: 20, MarginLeft: 20, MarginRight: 20,
	}); err != nil {
		t.Fatal(err)
	}
	if doc.PageCount() < 2 {
		t.Fatalf("expected multi-page table, got %d pages", doc.PageCount())
	}
	// Header text "ColA" should appear as ops on more than one page Y-band.
	contentH := 280.0 - 40.0
	pagesWithHeader := map[int]bool{}
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		if strings.Contains(op.Text, "ColA") || strings.Contains(op.Text, "ColB") {
			pagesWithHeader[int(op.Y/contentH)] = true
		}
	}
	if len(pagesWithHeader) < 2 {
		t.Fatalf("thead text on %d page band(s), want ≥2: %v", len(pagesWithHeader), pagesWithHeader)
	}
}

func TestTheadUADisplay(t *testing.T) {
	root := mustParse(t, `<html><body><table><thead><tr><th>H</th></tr></thead><tbody><tr><td>B</td></tr></tbody></table></body></html>`)
	res := layoutHTML(t, `<html><body><table><thead><tr><th>H</th></tr></thead><tbody><tr><td>B</td></tr></tbody></table></body></html>`)
	_ = root
	var tb *box
	var walk func(b *box)
	walk = func(b *box) {
		if b.kind == "table" {
			tb = b
			return
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
	if tb == nil {
		t.Fatal("no table box")
	}
	if tb.headerRows != 1 {
		t.Errorf("headerRows = %d, want 1", tb.headerRows)
	}
}
