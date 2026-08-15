//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// Reproduce blank page between short filmography tables and a tall awards
// table with rowspan + page-break-inside:avoid (wiki pattern).
func TestAwardsAfterFilmographyNoBlankPage(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
h2, h3 { font-size: 14pt; margin: 8pt 0 4pt; page-break-after: avoid; }
.wikitable { border-collapse: collapse; page-break-inside: avoid; margin: 1em 0; font-size: 10pt; }
td, th { border: 1px solid #aaa; padding: 3pt; }
`)

	var filmRows strings.Builder
	for i := range 8 {
		filmRows.WriteString(fmt.Sprintf(`<tr><td>%d</td><td>Film %d</td><td>Role</td><td>Notes here</td></tr>`, 2006+i, i))
	}

	var awardRows strings.Builder
	// rowspan=6 on year like wiki awards
	awardRows.WriteString(`<tr><td rowspan="6">2019</td><td>Award A</td><td>Cat</td><td>Work</td><td>Nom</td></tr>`)

	for i := range 5 {
		awardRows.WriteString(fmt.Sprintf(`<tr><td>Award %d</td><td>Cat</td><td>Work</td><td>Nom</td></tr>`, i))
	}

	for y := 2020; y <= 2024; y++ {
		awardRows.WriteString(fmt.Sprintf(
			`<tr><td rowspan="4">%d</td><td>Award</td><td>Cat</td><td>Work</td><td>Nom</td></tr>`, y))

		for range 3 {
			awardRows.WriteString(`<tr><td>Award</td><td>Cat</td><td>Work</td><td>Nom</td></tr>`)
		}
	}

	src := `<html><body>
<section>
<h2>Filmography</h2>
<h3>Television</h3>
<table class="wikitable"><tr><th>Year</th><th>Title</th><th>Role</th><th>Notes</th></tr>
` + filmRows.String() + `</table>
<h3>Video games</h3>
<table class="wikitable"><tr><th>Year</th><th>Title</th><th>Role</th><th>Notes</th></tr>
<tr><td>2025</td><td>Call of Duty</td><td>Eve</td><td>Playable</td></tr>
</table>
</section>
<section>
<h2>Awards and nominations</h2>
<table class="wikitable"><tr><th>Year</th><th>Award</th><th>Category</th><th>Work</th><th>Result</th></tr>
` + awardRows.String() + `</table>
</section>
</body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	const pageH = 750.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 538, Height: pageH, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	opPage := paginateOps(res, pageH)
	maxPage := 0
	pagesWithText := map[int]int{}

	var awardsY, videoY float64
	awardsY, videoY = -1, -1

	for paintOp, paintOp2 := range res.Ops {
		if paintOp2.Kind != OpText {
			continue
		}

		p := opPage[paintOp]
		if p > maxPage {
			maxPage = p
		}

		pagesWithText[p]++

		if paintOp2.Text == "Awards and nominations" {
			awardsY = paintOp2.Y
		}

		if strings.Contains(paintOp2.Text, "Call of Duty") {
			videoY = paintOp2.Y
		}
	}

	blank := 0

	for p := 0; p <= maxPage; p++ {
		if pagesWithText[p] == 0 {
			blank++

			t.Logf("blank page %d", p)
		}
	}

	t.Logf("pages=%d blank=%d videoY=%.0f awardsY=%.0f gap=%.0f", maxPage+1, blank, videoY, awardsY, awardsY-videoY)

	if blank > 0 {
		t.Fatalf("%d blank page(s)", blank)
	}

	if awardsY > 0 && videoY > 0 && awardsY-videoY > pageH*1.2 {
		t.Fatalf("gap video→awards = %.0f (>%.0f)", awardsY-videoY, pageH*1.2)
	}
}
