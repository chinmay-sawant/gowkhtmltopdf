package layout

// Rowspan Ref cells with <br> between cites (wiki awards pattern) must
// spread markers across the cell height instead of packing both at the top.

import (
	"math"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// Multi-cite nowrap markers in a narrow Ref td must not stack at the same X
// (wiki awards: [127][128] for one win). Prefer one horizontal line.
func TestMultiCiteInNarrowTDNotStacked(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 480pt; }
td, th { border: 1px solid #a2a9b1; padding: 2pt 3pt; font-size: 9pt; }
.reference { white-space: nowrap; font-size: 8pt; }
td.y, th.y { width: 28pt; }
td.o, th.o { width: 120pt; }
td.c, th.c { width: 180pt; }
td.w, th.w { width: 60pt; }
td.r, th.r { width: 48pt; }
td.ref, th.ref { width: 44pt; }
`)
	htmlSrc := `<html><body><table>
<tr>
<th class="y">Year</th><th class="o">Organization</th><th class="c">Category</th>
<th class="w">Work</th><th class="r">Result</th><th class="ref">Ref.</th>
</tr>
<tr>
<td class="y">2019</td>
<td class="o">EDA Awards</td>
<td class="c">She Deserves a New Agent Award</td>
<td class="w"></td>
<td class="r">Won</td>
<td class="ref"><sup class="reference"><a href="#c127"><span class="cite-bracket">[</span>127<span class="cite-bracket">]</span></a></sup><sup class="reference"><a href="#c128"><span class="cite-bracket">[</span>128<span class="cite-bracket">]</span></a></sup></td>
</tr>
</table></body></html>`
	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 520, Height: 400, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	type mark struct {
		text string
		x, y float64
	}
	var marks []mark
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		// Coalesced cite ops are "[127]" / hair+"[128]" or piece-wise.
		compact := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\u200a' || r == '\u2009' {
				return -1
			}
			return r
		}, op.Text)
		if strings.Contains(compact, "127") || strings.Contains(compact, "128") ||
			compact == "[" || compact == "]" {
			marks = append(marks, mark{op.Text, op.X, op.Y})
		}
	}
	if len(marks) < 2 {
		t.Fatalf("expected multi-cite text ops, got %v", marks)
	}

	// Collect distinct baselines and X positions for cite ink.
	ys := map[float64]bool{}
	minX, maxX := marks[0].x, marks[0].x
	for _, m := range marks {
		ys[math.Round(m.y*4)/4] = true // 0.25pt bins
		if m.x < minX {
			minX = m.x
		}
		if m.x > maxX {
			maxX = m.x
		}
	}
	// Prefer a single baseline (horizontal cluster). If wrapped, require
	// horizontal advance somewhere so markers are not stacked at the same X.
	if len(ys) == 1 {
		if maxX-minX < 6 {
			t.Fatalf("cite markers share one line but no horizontal advance: %v", marks)
		}
		return
	}
	// Wrapped: must not all share the same X (stacked glyphs).
	sameX := true
	for i := 1; i < len(marks); i++ {
		if math.Abs(marks[i].x-marks[0].x) > 1 {
			sameX = false
			break
		}
	}
	if sameX {
		t.Fatalf("multi-cite markers stacked at same X (ys=%v marks=%v)", ys, marks)
	}
}

// Ref cell min-content for [n][m] nowrap cluster must exceed a single marker
// so the column is not forced to one-marker width.
func TestMultiCiteRefMinContentWiderThanOneMarker(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; }
td { border: 1px solid #999; padding: 2pt; font-size: 9pt; }
.reference { white-space: nowrap; font-size: 8pt; }
`)
	one := `<html><body><table><tr><td><sup class="reference"><a href="#a"><span>[</span>127<span>]</span></a></sup></td></tr></table></body></html>`
	two := `<html><body><table><tr><td><sup class="reference"><a href="#a"><span>[</span>127<span>]</span></a></sup><sup class="reference"><a href="#b"><span>[</span>128<span>]</span></a></sup></td></tr></table></body></html>`
	wOne := refCellWidth(t, one, s)
	wTwo := refCellWidth(t, two, s)
	t.Logf("one=%.1f two=%.1f", wOne, wTwo)
	if wTwo < wOne+8 {
		t.Fatalf("multi-cite ref cell width %.1f not wider than single %.1f by ≥8pt", wTwo, wOne)
	}
}

func refCellWidth(t *testing.T, src string, s *css.Stylesheet) float64 {
	t.Helper()
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 400, Height: 200, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	tb := findBox(t, res, "table")
	if len(tb.rows) == 0 || len(tb.rows[0]) == 0 {
		t.Fatal("no cell")
	}
	return tb.rows[0][0].w
}

func TestRowspanBrCitesSpreadVertically(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { border-collapse: collapse; width: 480pt; }
td { border: 1px solid #aaa; padding: 2pt; font-size: 9pt; }
.reference { white-space: nowrap; font-size: 8pt; }
`)
	src := `<html><body><table>
<tr><td rowspan="2">2023</td><td>EDA Awards</td><td>She Deserves a New Agent Award</td><td>Won</td>
<td rowspan="2" style="text-align:center"><sup class="reference"><a href="#a"><span class="cite-bracket">[</span>127<span class="cite-bracket">]</span></a></sup><br/><sup class="reference"><a href="#b"><span class="cite-bracket">[</span>128<span class="cite-bracket">]</span></a></sup></td>
</tr>
<tr><td></td><td>Most Egregious Lovers Age Difference Award</td><td>Nominated</td></tr>
</table></body></html>`
	root := mustParse(t, src)
	res, err := Layout(root, Options{Width: 500, Height: 400, Sheets: []*css.Stylesheet{s}, Background: true, Zoom: 0.666667})
	if err != nil {
		t.Fatal(err)
	}
	var y127, y128 float64
	var n127, n128 int
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		if op.Text == "127" || op.Text == "[127]" || (len(op.Text) >= 3 && op.Text == "127") {
			// collect digit ops
		}
		if containsCite(op.Text, "127") {
			y127 += op.Y
			n127++
		}
		if containsCite(op.Text, "128") {
			y128 += op.Y
			n128++
		}
	}
	if n127 == 0 || n128 == 0 {
		// try full marker as single op
		for _, op := range res.Ops {
			if op.Kind == OpText {
				t.Logf("text %q y=%.1f", op.Text, op.Y)
			}
		}
		t.Fatalf("missing cites n127=%d n128=%d", n127, n128)
	}
	y127 /= float64(n127)
	y128 /= float64(n128)
	dy := y128 - y127
	if dy < 8 {
		t.Fatalf("cite baselines too close dy=%.1f (y127=%.1f y128=%.1f); want spread across rowspan", dy, y127, y128)
	}
}

func containsCite(text, num string) bool {
	if text == num || text == "["+num+"]" {
		return true
	}
	// digit-only piece of [ 127 ]
	return text == num
}
