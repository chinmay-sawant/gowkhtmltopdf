package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// Filmography-like auto table: columns sized from longest-word (min-content)
// leave a narrow table and tall wrapped rows. Max-content sizing should give
// a wide table filling most of the containing block with short rows.
func TestAutoTableUsesMaxContentColumnWidths(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; }
td, th { border: 1px solid #aaa; padding: 0.4em 0.6em; }
`)

	const contentW = 538.0

	htmlSrc := `<html><body>
<table>
<tr><th>Year</th><th>Title</th><th>Role</th><th>Notes</th></tr>
<tr><td>2006</td><td>Virgin Rose</td><td>Marie</td><td>Original Spanish title Una rosa de Francia</td></tr>
<tr><td>2009</td><td>Sex, Party and Lies</td><td>Carola</td><td>Original Spanish title Mentiras y gordas</td></tr>
<tr><td>2007</td><td>Madrigal</td><td>Stella Maris</td><td></td></tr>
</table>
</body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contentW, Height: 800, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Find Notes header x and rightmost text in first data row.
	var notesX, maxRight, year0Y, year1Y float64
	year0Y = -1

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && paintOp.Text == "Notes" {
			notesX = paintOp.X
		}

		if paintOp.Kind == OpText && paintOp.Text == "2006" {
			year0Y = paintOp.Y
		}

		if paintOp.Kind == OpText && paintOp.Text == "2009" {
			year1Y = paintOp.Y
		}

		if paintOp.Kind == OpText && paintOp.X+paintOp.W > maxRight {
			maxRight = paintOp.X + paintOp.W
		}
	}

	t.Logf("Notes header x=%.0f maxRight=%.0f row delta=%.0f", notesX, maxRight, year1Y-year0Y)

	if notesX < 200 {
		t.Fatalf("Notes column starts at x=%.0f; want further right (max-content distribution)", notesX)
	}

	if maxRight < contentW*0.65 {
		t.Fatalf("table only reaches x=%.0f; want >= %.0f (max-content shrink-wrap)", maxRight, contentW*0.65)
	}
	// With real max-content, filmography-like rows stay near one line (+ cell padding).
	if year0Y > 0 && year1Y > year0Y && year1Y-year0Y > 40 {
		t.Fatalf("row height delta=%.0fpt too tall (wrapping from narrow cols); want denser rows", year1Y-year0Y)
	}
	// "Virgin Rose" should sit on one line (not wrap after Virgin).
	virginY, roseY := -1.0, -1.0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if paintOp.Text == "Virgin" || paintOp.Text == "Virgin " {
			virginY = paintOp.Y
		}

		if paintOp.Text == "Rose" || paintOp.Text == "Rose " {
			roseY = paintOp.Y
		}

		if paintOp.Text == "Virgin Rose" || paintOp.Text == "Virgin Rose " {
			virginY, roseY = paintOp.Y, paintOp.Y
		}
	}

	if virginY >= 0 && roseY >= 0 && roseY-virginY > 2 {
		t.Fatalf("Title 'Virgin Rose' wrapped across lines (y=%.1f vs %.1f)", virginY, roseY)
	}
}
