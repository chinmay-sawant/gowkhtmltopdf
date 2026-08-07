package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// TestTableRowspanColumnAlignment: cells after a rowspan must stay in their
// logical columns (wiki filmography Year rowspan is the motivating case).
func TestTableRowspanColumnAlignment(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 400pt; }
td, th { border: 1px solid #aaa; padding: 2pt; text-align: left; }
`)
	htmlSrc := `<html><body><table>
<tr><th>Year</th><th>Title</th><th>Role</th></tr>
<tr><td rowspan="2">2009</td><td>Sex Party</td><td>Carola</td></tr>
<tr><td>And for Dessert</td><td>Girl</td></tr>
<tr><td>2011</td><td>Blind Alley</td><td>Rosa</td></tr>
</table></body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: 400, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var yearX, titleX, dessertX, girlX, alleyX float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		switch {
		case paintOp.Text == "Year":
			yearX = paintOp.X
		case paintOp.Text == "Title":
			titleX = paintOp.X
		case paintOp.Text == "And for Dessert" || paintOp.Text == "And for Dessert ":
			dessertX = paintOp.X
		case paintOp.Text == "Girl" || paintOp.Text == "Girl ":
			girlX = paintOp.X
		case paintOp.Text == "Blind Alley" || paintOp.Text == "Blind Alley ":
			alleyX = paintOp.X
		}
	}

	t.Logf("Year=%.0f Title=%.0f Dessert=%.0f Girl=%.0f Alley=%.0f", yearX, titleX, dessertX, girlX, alleyX)

	if titleX <= yearX+10 {
		t.Fatalf("Title header x=%.0f not right of Year x=%.0f", titleX, yearX)
	}
	// Rowspan continuation row: title text must align under Title, not Year.
	if dessertX < titleX-5 {
		t.Fatalf("rowspan continuation title at x=%.0f, want under Title (>= %.0f)", dessertX, titleX)
	}

	if girlX <= dessertX {
		t.Fatalf("Role 'Girl' at x=%.0f should be right of Title text at %.0f", girlX, dessertX)
	}

	if alleyX < titleX-5 {
		t.Fatalf("Blind Alley at x=%.0f, want under Title", alleyX)
	}
}

// TestTableTallRowspanNoBlankPages: a Year-like cell with rowspan covering
// many short rows must not make rowsIntact treat the first row as multi-page
// (bottom border at full span height) and cascade blank pages.
func TestTableTallRowspanNoBlankPages(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 100%; }
td, th { border: 1px solid #333; padding: 3pt; }
`)

	var rows string

	rows += `<tr><td rowspan="20">2019</td><td>Award body zero</td><td>Category</td><td>Nominated</td></tr>`
	for i := 1; i < 20; i++ {
		rows += `<tr><td>Award body ` + string(rune('A'+i%26)) + `</td><td>Category</td><td>Nominated</td></tr>`
	}
	// More years so the table spans several pages.
	for y := range 3 {
		rows += `<tr><td rowspan="12">202` + string(rune('0'+y)) + `</td><td>More awards</td><td>Cat</td><td>Won</td></tr>`
		for i := 1; i < 12; i++ {
			rows += `<tr><td>More awards ` + string(rune('a'+i)) + `</td><td>Cat</td><td>Won</td></tr>`
		}
	}

	htmlSrc := `<html><body><table>
<tr><th>Year</th><th>Award</th><th>Category</th><th>Result</th></tr>` + rows + `</table></body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	const pageH = 400.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: pageH, Sheets: []*css.Stylesheet{cssSheet},
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
		}
	}

	t.Logf("pages=%d blank=%d", maxPage+1, blank)

	if blank > 0 {
		t.Fatalf("%d blank pages from rowspan rowsIntact", blank)
	}

	if maxPage+1 > 15 {
		t.Fatalf("unexpectedly many pages %d (rowspan height bug?)", maxPage+1)
	}
}
