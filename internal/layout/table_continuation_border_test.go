package layout

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// Multi-page border-collapse table with rowspan: the first body row on a
// continuation page (under repeated thead) must form a closed outer strip —
// full-width top edge across all columns, including Year/Org rowspan holes.
func TestRowspanContinuationPageClosedOuterBorders(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 400pt; }
td, th { border: 1px solid #333; padding: 3pt; }
`)

	var rows string
	// Enough plain rows to push the rowspan pair onto a later page.
	for i := range 14 {
		rows += fmt.Sprintf(
			`<tr><td>%d</td><td>Org %d</td><td>Category %d</td><td>Work</td><td>Res</td><td>R</td></tr>`,
			2000+i, i, i)
	}

	rows += `<tr><td rowspan="2">2024</td><td rowspan="2">Razzie Awards</td>` +
		`<td>Worst Actress</td><td>Ghosted</td><td>Nominated</td><td>[136]</td></tr>`
	rows += `<tr><td>Worst Screen Combo (shared with Chris Evans)</td>` +
		`<td></td><td>Nominated</td><td></td></tr>`

	htmlSrc := `<html><body><table>
<tr><th>Year</th><th>Organization</th><th>Category</th><th>Work</th><th>Result</th><th>Ref.</th></tr>
` + rows + `</table></body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	const (
		pageH = 220.0
		pageW = 500.0
	)

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: pageW, Height: pageH, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Drive the full paint pagination path (thead repeat + capTablePageBreaks).
	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH + 40,
		MarginTop: 20, MarginBottom: 20, MarginLeft: 20, MarginRight: 20,
	}); err != nil {
		t.Fatal(err)
	}

	contentH := pageH // Paint uses PageHeight - margins = pageH when margins 20+20 and height pageH+40

	// Find the continuation page that has "Worst Screen Combo" but not "2024"
	// as a body year on the same band.
	opPage := make([]int, len(res.Ops))

	for idx, paintOp := range res.Ops {
		if paintOp.Fixed {
			opPage[idx] = -1

			continue
		}

		opPage[idx] = int(paintOp.Y / contentH)
	}

	type pageInfo struct {
		hasCombo, has2024 bool
		bodyTop           float64
		hasBodyTop        bool
	}

	pages := map[int]*pageInfo{}

	for i, paintOp := range res.Ops {
		page := opPage[i]
		if page < 0 {
			continue
		}

		info := pages[page]
		if info == nil {
			info = &pageInfo{} //nolint:exhaustruct // intentional zero fields
			pages[page] = info
		}

		if paintOp.Kind == OpText {
			if strings.Contains(paintOp.Text, "Combo") || strings.Contains(paintOp.Text, "Screen") {
				info.hasCombo = true
			}

			if paintOp.Text == "2024" {
				info.has2024 = true
			}
		}
	}
	// Locate continuation: Combo text without 2024 on that page (rowspan hole).
	contPage := -1

	for p, info := range pages {
		if info.hasCombo && !info.has2024 && p > 0 {
			contPage = p

			break
		}
	}
	// Fallback: any page >0 with Combo.
	if contPage < 0 {
		for p, info := range pages {
			if info.hasCombo && p > 0 {
				contPage = p

				break
			}
		}
	}

	if contPage < 0 {
		t.Fatalf("no continuation page with Screen Combo; pages=%v", pages)
	}

	pageTop := float64(contPage) * contentH
	pageBot := pageTop + contentH

	// Verticals and horizontals on this page for the body band (below header).
	var verts []struct{ x, y0, y1 float64 }

	var horiz []struct{ x0, x1, y float64 }

	for _, paintOp := range res.Ops {
		if paintOp.Fixed || paintOp.Kind != OpLine {
			continue
		}

		if paintOp.Y < pageTop-1 || paintOp.Y > pageBot+1 {
			// verticals may start in-band
			if !(paintOp.H > 2 && paintOp.Y+paintOp.H > pageTop && paintOp.Y < pageBot) {
				continue
			}
		}

		if paintOp.H > 2 && (paintOp.W < 1 || paintOp.W < paintOp.H*0.05) {
			y0, y1 := paintOp.Y, paintOp.Y+paintOp.H
			if y1 < pageTop+5 {
				continue // header-only
			}

			verts = append(verts, struct{ x, y0, y1 float64 }{paintOp.X, y0, y1})

			continue
		}

		if paintOp.W > 10 && paintOp.H < 1 && paintOp.Y >= pageTop && paintOp.Y <= pageBot {
			horiz = append(horiz, struct{ x0, x1, y float64 }{paintOp.X, paintOp.X + paintOp.W, paintOp.Y})
		}
	}

	if len(verts) < 3 {
		t.Fatalf("page %d: expected multi-column verticals, got %d", contPage, len(verts))
	}

	// Body strip: first cluster of verticals below the header band.
	// Header is ~20pt; body starts later on continuation pages.
	minBodyY := pageTop + contentH
	maxBodyY1 := pageTop
	minX, maxX := verts[0].x, verts[0].x

	for _, val := range verts {
		if val.y0 < pageTop+15 {
			continue // skip pure header verticals at page top
		}

		if val.y0 < minBodyY {
			minBodyY = val.y0
		}

		if val.y1 > maxBodyY1 {
			maxBodyY1 = val.y1
		}

		if val.x < minX {
			minX = val.x
		}

		if val.x > maxX {
			maxX = val.x
		}
	}

	if maxX-minX < 50 {
		t.Fatalf("page %d: vertical span too narrow [%.1f,%.1f]", contPage, minX, maxX)
	}

	// Require a (near-)full-width horizontal at the body strip top.
	const eps = 3.0

	bestCov := 0.0

	var topY float64

	for _, height := range horiz {
		if math.Abs(height.y-minBodyY) > eps {
			continue
		}
		// Intersection with [minX,maxX].
		lo := math.Max(height.x0, minX)
		hi := math.Min(height.x1, maxX)

		if hi > lo {
			bestCov += hi - lo
			topY = height.y
		}
	}
	// Allow multiple segments; total coverage should span most of the width.
	need := (maxX - minX) * 0.9
	if bestCov < need {
		t.Fatalf("page %d continuation body top incomplete: coverage=%.1f need≥%.1f at y≈%.1f (table [%.1f,%.1f]); horiz=%v",
			contPage, bestCov, need, minBodyY, minX, maxX, horiz)
	}
	// Outer vertical edges present for the body band.
	hasLeft, hasRight := false, false

	for _, v := range verts {
		if math.Abs(v.x-minX) <= 1 && v.y1 > minBodyY+2 {
			hasLeft = true
		}

		if math.Abs(v.x-maxX) <= 1 && v.y1 > minBodyY+2 {
			hasRight = true
		}
	}

	if !hasLeft || !hasRight {
		t.Fatalf("page %d missing outer verticals left=%v right=%v (topY=%.1f)", contPage, hasLeft, hasRight, topY)
	}
}
