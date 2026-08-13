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
		if boxNode.kind == displayTable {
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

func TestContinuationHeaderNotOverlappedByBodyRow(t *testing.T) { //nolint:cyclop
	t.Parallel()

	src := `<html><body><table>
<tr><th>Year</th><th>Organization</th><th>Category</th><th>Work</th><th>Result</th></tr>`
	for i := range 18 {
		src += `<tr><td>` + strconv.Itoa(2000+i) + `</td><td>Org ` + strconv.Itoa(i) +
			`</td><td>Category ` + strconv.Itoa(i) + `</td><td>Work</td><td>Nominated</td></tr>`
	}

	src += `<tr><td>2023</td><td>Golden Globe Awards</td>` +
		`<td>Best Actress in a Motion Picture Drama</td><td>Blonde</td><td>Nominated</td></tr>`
	src += `<tr><td>2023</td><td>London Film Critics Circle Awards</td>` +
		`<td>Actress of the Year</td><td></td><td>Nominated</td></tr>`
	src += `</table></body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	const contentH = 220.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // continuation header geometry
		Width: 500, Height: contentH, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: 540, PageHeight: contentH + 40,
		MarginTop: 20, MarginBottom: 20, MarginLeft: 20, MarginRight: 20,
	}); err != nil {
		t.Fatal(err)
	}

	type mark struct {
		y    float64
		kind string
	}

	marks := []mark{}
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		switch {
		case strings.Contains(op.Text, "Year") && !strings.Contains(op.Text, "Actress"):
			marks = append(marks, mark{op.Y, "header"})
		case strings.Contains(op.Text, "Blonde"):
			marks = append(marks, mark{op.Y, "blonde"})
		}
	}

	for _, body := range marks {
		if body.kind != "blonde" {
			continue
		}

		page := int(body.y / contentH)
		if page <= 0 {
			continue
		}

		pageTop := float64(page) * contentH
		if body.y < pageTop+2 {
			t.Fatalf("Blonde row y=%.2f sits in the page-%d margin, want below thead", body.y, page+1)
		}

		for _, hdr := range marks {
			if hdr.kind != "header" {
				continue
			}

			if int(hdr.y/contentH) != page {
				continue
			}

			if body.y < hdr.y+4 {
				t.Fatalf("Blonde y=%.2f overlaps header y=%.2f on page %d", body.y, hdr.y, page+1)
			}
		}
	}
}
