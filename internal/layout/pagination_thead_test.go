//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strconv"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestTheadRepeatOnContinuationPages(t *testing.T) { //nolint:cyclop
	t.Parallel()

	src := `<html><body>
<table>
<thead><tr><th>ColA</th><th>ColB</th></tr></thead>
<tbody>`
	for i := range 40 {
		src += `<tr><td>r` + strconv.Itoa(i) + `</td><td>body row ` + strconv.Itoa(i) + `</td></tr>`
	}

	src += `</tbody></table></body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Background: true,
	})
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

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "ColA") || strings.Contains(paintOp.Text, "ColB") {
			pagesWithHeader[int(paintOp.Y/contentH)] = true
		}
	}

	if len(pagesWithHeader) < 2 {
		t.Fatalf("thead text on %d page band(s), want ≥2: %v", len(pagesWithHeader), pagesWithHeader)
	}
}

func TestTheadUADisplay(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<table><thead><tr><th>H</th></tr></thead><tbody><tr><td>B</td></tr></tbody></table>`+
		`</body></html>`)
	res := layoutHTML(t, `<html><body>`+
		`<table><thead><tr><th>H</th></tr></thead><tbody><tr><td>B</td></tr></tbody></table>`+
		`</body></html>`)
	_ = root

	var tblBox *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.kind == "table" {
			tblBox = boxNode

			return
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if tblBox == nil {
		t.Fatal("no table box")
	}

	if tblBox.headerRows != 1 {
		t.Errorf("headerRows = %d, want 1", tblBox.headerRows)
	}
}
