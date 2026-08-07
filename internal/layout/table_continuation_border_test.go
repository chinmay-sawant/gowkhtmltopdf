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
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 400pt; }
td, th { border: 1px solid #333; padding: 3pt; }
`)
	var rows string
	// Enough plain rows to push the rowspan pair onto a later page.
	for i := 0; i < 14; i++ {
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
	res, err := Layout(root, Options{
		Width: pageW, Height: pageH, Sheets: []*css.Stylesheet{s},
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
	for i, op := range res.Ops {
		if op.Fixed {
			opPage[i] = -1
			continue
		}
		opPage[i] = int(op.Y / contentH)
	}
	type pageInfo struct {
		hasCombo, has2024 bool
		bodyTop           float64
		hasBodyTop        bool
	}
	pages := map[int]*pageInfo{}
	for i, op := range res.Ops {
		p := opPage[i]
		if p < 0 {
			continue
		}
		info := pages[p]
		if info == nil {
			info = &pageInfo{}
			pages[p] = info
		}
		if op.Kind == OpText {
			if strings.Contains(op.Text, "Combo") || strings.Contains(op.Text, "Screen") {
				info.hasCombo = true
			}
			if op.Text == "2024" {
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
	for _, op := range res.Ops {
		if op.Fixed || op.Kind != OpLine {
			continue
		}
		if op.Y < pageTop-1 || op.Y > pageBot+1 {
			// verticals may start in-band
			if !(op.H > 2 && op.Y+op.H > pageTop && op.Y < pageBot) {
				continue
			}
		}
		if op.H > 2 && (op.W < 1 || op.W < op.H*0.05) {
			y0, y1 := op.Y, op.Y+op.H
			if y1 < pageTop+5 {
				continue // header-only
			}
			verts = append(verts, struct{ x, y0, y1 float64 }{op.X, y0, y1})
			continue
		}
		if op.W > 10 && op.H < 1 && op.Y >= pageTop && op.Y <= pageBot {
			horiz = append(horiz, struct{ x0, x1, y float64 }{op.X, op.X + op.W, op.Y})
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
	for _, v := range verts {
		if v.y0 < pageTop+15 {
			continue // skip pure header verticals at page top
		}
		if v.y0 < minBodyY {
			minBodyY = v.y0
		}
		if v.y1 > maxBodyY1 {
			maxBodyY1 = v.y1
		}
		if v.x < minX {
			minX = v.x
		}
		if v.x > maxX {
			maxX = v.x
		}
	}
	if maxX-minX < 50 {
		t.Fatalf("page %d: vertical span too narrow [%.1f,%.1f]", contPage, minX, maxX)
	}

	// Require a (near-)full-width horizontal at the body strip top.
	const eps = 3.0
	bestCov := 0.0
	var topY float64
	for _, h := range horiz {
		if math.Abs(h.y-minBodyY) > eps {
			continue
		}
		// Intersection with [minX,maxX].
		lo := math.Max(h.x0, minX)
		hi := math.Min(h.x1, maxX)
		if hi > lo {
			bestCov += hi - lo
			topY = h.y
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
