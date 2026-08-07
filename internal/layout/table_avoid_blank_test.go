package layout

import (
	"fmt"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestTallTableAvoidInsideNoBlankPages(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; page-break-inside: avoid; }
td, th { border: 1px solid #aaa; padding: 4pt; }
`)

	var rows string
	for i := range 40 {
		rows += fmt.Sprintf(`<tr><td>%d</td><td>Title number %d with some words</td><td>Role %d</td><td>Original Spanish title Something Something</td></tr>`, 2000+i, i, i)
	}

	htmlSrc := `<html><body><h2>Film</h2><table><tr><th>Year</th><th>Title</th><th>Role</th><th>Notes</th></tr>` + rows + `</table></body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 538, Height: 700, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	contentH := 700.0
	opPage := paginateOps(res, contentH)
	maxPage := 0
	pagesWithText := map[int]int{}

	for i, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		p := opPage[i]
		if p > maxPage {
			maxPage = p
		}

		pagesWithText[p]++
	}

	blank := 0

	for p := 0; p <= maxPage; p++ {
		if pagesWithText[p] == 0 {
			blank++
		}
	}

	t.Logf("pages=%d blank=%d textPages=%d", maxPage+1, blank, len(pagesWithText))

	if blank > 0 {
		t.Fatalf("%d blank pages in pagination (table avoid-inside bug)", blank)
	}
}

// TestAvoidTableAfterContentNoBlankCascade: a mid-page-starting table that
// fits one page with page-break-inside:avoid, preceded by enough content and
// a heading — must not cascade blank pages via avoid+keep-heading.
func TestAvoidTableAfterContentNoBlankCascade(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 11pt; }
p { margin: 0.4em 0; }
table { border-collapse: collapse; page-break-inside: avoid; width: 100%; }
td, th { border: 1px solid #aaa; padding: 3pt; font-size: 10pt; }
h2 { font-size: 14pt; margin: 8pt 0 4pt; }
`)

	var paras string
	for i := range 25 {
		paras += fmt.Sprintf(`<p>Lead paragraph %d with enough words to fill lines beside the upcoming filmography table section.</p>`, i)
	}

	var rows string
	for i := range 18 {
		rows += fmt.Sprintf(`<tr><td>%d</td><td>Virgin Rose film title %d</td><td>Marie</td><td>Original Spanish title Una rosa de Francia long notes</td></tr>`, 2006+i, i)
	}

	htmlSrc := `<html><body>` + paras + `<h2>Filmography</h2><h3>Film</h3><table>
<tr><th>Year</th><th>Title</th><th>Role</th><th>Notes</th></tr>` + rows + `</table>
<h3>Television</h3><table>
<tr><th>Year</th><th>Title</th><th>Role</th><th>Notes</th></tr>
<tr><td>2007</td><td>El Internado</td><td>Carolina</td><td>56 episodes</td></tr>
</table>
</body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	const pageH = 750.0

	res, err := Layout(root, Options{
		Width: 538, Height: pageH, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	opPage := paginateOps(res, pageH)
	maxPage := 0
	pagesWithText := map[int]int{}

	for i, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		p := opPage[i]
		if p > maxPage {
			maxPage = p
		}

		pagesWithText[p]++
	}

	blank := 0

	for p := 0; p <= maxPage; p++ {
		if pagesWithText[p] == 0 {
			blank++

			t.Logf("blank page %d", p)
		}
	}

	t.Logf("pages=%d blank=%d", maxPage+1, blank)

	if blank > 1 {
		t.Fatalf("%d blank pages — avoid-inside/heading cascade", blank)
	}
}
