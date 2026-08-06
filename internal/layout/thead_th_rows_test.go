package layout

import (
	"fmt"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// Leading all-<th> rows without <thead> still repeat on continuation pages.
func TestLeadingTHRowsRepeatAsHeader(t *testing.T) {
	src := `<html><body>
<table>
<tr><th>ColA</th><th>ColB</th></tr>
`
	for i := 0; i < 40; i++ {
		src += fmt.Sprintf(`<tr><td>r%d</td><td>body row %d with text</td></tr>`, i, i)
	}
	src += `</table></body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{Width: 400, Height: 200, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	// headerRows must be detected from first all-th row.
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
		t.Fatal("no table")
	}
	if tb.headerRows != 1 {
		t.Fatalf("headerRows=%d, want 1 (leading th row)", tb.headerRows)
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
		t.Fatalf("leading-th header on %d page band(s), want ≥2: %v", len(pagesWithHeader), pagesWithHeader)
	}
}

func TestMixedFirstRowNotHeader(t *testing.T) {
	// Row headers (th+td) must not be treated as a repeating column header band.
	src := `<html><body><table>
<tr><th>Name</th><td>Alice</td></tr>
<tr><th>Age</th><td>30</td></tr>
</table></body></html>`
	res := layoutHTML(t, src)
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
		t.Fatal("no table")
	}
	if tb.headerRows != 0 {
		t.Fatalf("headerRows=%d, want 0 for th+td first row", tb.headerRows)
	}
}
