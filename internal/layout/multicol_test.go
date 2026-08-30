//nolint:all
//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestMulticolParseProps(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.a { column-count: 3; column-gap: 12pt; column-fill: auto }
.b { columns: 100pt 2; column-span: all; column-fill: balance }
.c { column-gap: normal; column-width: 80pt }
.d { break-before: column; break-inside: avoid-column }
`)
	root := mustParse(t, `<html><body>
<div class="a">A</div>
<div class="b">B</div>
<div class="c">C</div>
<div class="d">D</div>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)

	var nodes []*html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "div" {
			nodes = append(nodes, n)
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if len(nodes) < 4 {
		t.Fatalf("expected 4 divs, got %d", len(nodes))
	}

	a := styles[nodes[0]]
	if a.ColumnCount != 3 || a.ColumnGap != 12 || a.ColumnGapNormal || a.ColumnFill != overflowAuto {
		t.Fatalf("a: count=%d gap=%.1f normal=%v fill=%q", a.ColumnCount, a.ColumnGap, a.ColumnGapNormal, a.ColumnFill)
	}

	b := styles[nodes[1]]
	if b.ColumnCount != 2 || b.ColumnWidth < 99 || b.ColumnWidth > 101 ||
		b.ColumnSpan != "all" || b.ColumnFill != "balance" {
		t.Fatalf("b: count=%d width=%.1f span=%q fill=%q", b.ColumnCount, b.ColumnWidth, b.ColumnSpan, b.ColumnFill)
	}

	c := styles[nodes[2]]
	if !c.ColumnGapNormal || c.ColumnWidth < 79 || c.ColumnWidth > 81 || c.ColumnCount != 0 {
		t.Fatalf("c: normal=%v width=%.1f count=%d", c.ColumnGapNormal, c.ColumnWidth, c.ColumnCount)
	}

	decl := styles[nodes[3]]
	// break-before:column ≈ page always; break-inside:avoid-column is
	// column-only and must not set page-break-inside:avoid.
	if decl.PageBreakBefore != pageBreakAlways {
		t.Fatalf("d break-before: got %q, want always", decl.PageBreakBefore)
	}

	if decl.PageBreakInside == avoidKeyword {
		t.Fatalf("d break-inside:avoid-column must not map to page avoid (got %q)", decl.PageBreakInside)
	}
}

func TestUsedColumnCountWidth(t *testing.T) {
	t.Parallel()

	nodeN, width := usedColumnCountWidth(200, 10, -1, 2)
	if nodeN != 2 || math.Abs(width-95) > 0.01 {
		t.Fatalf("count-only: n=%d w=%.2f want 2 / 95", nodeN, width)
	}

	nodeN, width = usedColumnCountWidth(200, 10, 60, 0)
	if nodeN != 3 || math.Abs(width-60) > 0.01 {
		t.Fatalf("width-only: n=%d w=%.2f want 3 / 60", nodeN, width)
	}

	nodeN, width = usedColumnCountWidth(100, 10, -1, 0)
	if nodeN != 1 || math.Abs(width-100) > 0.01 {
		t.Fatalf("both auto: n=%d w=%.2f", nodeN, width)
	}
}

func TestMulticolTwoColumnEqualWidths(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.mc {
  column-count: 2;
  column-gap: 20pt;
  width: 220pt;
  column-fill: balance;
}
.mc p { margin: 0 0 4pt 0; font-size: 10pt; }
`)
	res := layoutHTML(t, `<html><body>
<div class="mc">
  <p>Alpha left column line one.</p>
  <p>Bravo more text for height.</p>
  <p>Charlie right column start.</p>
  <p>Delta trailing paragraph.</p>
</div>
</body></html>`, cssSheet)
	pos := map[string]float64{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			for _, key := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
				if len(op.Text) >= len(key) && op.Text[:len(key)] == key {
					pos[key] = op.X
				}
			}
		}
	}

	for _, k := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
		if _, ok := pos[k]; !ok {
			t.Fatalf("missing text %s in ops", k)
		}
	}

	left := math.Min(math.Min(pos["Alpha"], pos["Bravo"]), math.Min(pos["Charlie"], pos["Delta"]))
	right := math.Max(math.Max(pos["Alpha"], pos["Bravo"]), math.Max(pos["Charlie"], pos["Delta"]))
	dx := right - left

	if dx < 90 || dx > 130 {
		t.Fatalf("column dx=%.1f (left=%.1f right=%.1f), want ~100–120", dx, left, right)
	}
}

func TestMulticolColumnSpanAll(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.mc { column-count: 2; column-gap: 10pt; width: 210pt; column-fill: balance }
.mc p { margin: 0 0 2pt 0; font-size: 9pt }
.mc h2 { column-span: all; font-size: 12pt; margin: 4pt 0 }
`)
	res := layoutHTML(t,
		"<html><body>\n<div class=\"mc\">\n  <p>Before span AAA AAA.</p>\n  <p>Before span BBB BBB.</p>\n"+
			"  <h2>Spanner Heading</h2>\n  <p>After span CCC CCC.</p>"+
			"  <p>After span DDD DDD.</p>\n</div>\n</body></html>", cssSheet)

	var spanX, beforeX, afterX float64

	var gotSpan, gotBefore, gotAfter bool

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if len(paintOp.Text) >= 7 && paintOp.Text[:7] == "Spanner" {
			spanX = paintOp.X
			gotSpan = true
		}

		if len(paintOp.Text) >= 6 && paintOp.Text[:6] == "Before" {
			beforeX = paintOp.X
			gotBefore = true
		}

		if len(paintOp.Text) >= 5 && paintOp.Text[:5] == "After" {
			afterX = paintOp.X
			gotAfter = true
		}
	}

	if !gotSpan || !gotBefore || !gotAfter {
		t.Fatalf("missing texts span=%v before=%v after=%v", gotSpan, gotBefore, gotAfter)
	}

	if spanX > beforeX+40 && spanX > afterX+40 {
		t.Fatalf("spanner x=%.1f looks column-shifted (before=%.1f after=%.1f)", spanX, beforeX, afterX)
	}
}

func TestMulticolLinesDoNotStraddlePages(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.mc { column-count: 2; column-gap: 8pt; width: 200pt; column-fill: balance; font-size: 10pt }
.mc p { margin: 0 0 6pt 0 }
`)
	root := mustParse(t, `<html><body>
<div class="mc">
  <p>P01 line of multicol article text filler words here.</p>
  <p>P02 line of multicol article text filler words here.</p>
  <p>P03 line of multicol article text filler words here.</p>
  <p>P04 line of multicol article text filler words here.</p>
  <p>P05 line of multicol article text filler words here.</p>
  <p>P06 line of multicol article text filler words here.</p>
  <p>P07 line of multicol article text filler words here.</p>
  <p>P08 line of multicol article text filler words here.</p>
  <p>P09 line of multicol article text filler words here.</p>
  <p>P10 line of multicol article text filler words here.</p>
  <p>P11 line of multicol article text filler words here.</p>
  <p>P12 line of multicol article text filler words here.</p>
</div>
</body></html>`)
	pageH := 120.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 220, Height: pageH, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		opH := paintOp.Size * 1.2
		lo := int(paintOp.Y / pageH)
		hi := int((paintOp.Y + opH) / pageH)

		if hi > lo {
			t.Fatalf("text %q straddles page at y=%.1f h=%.1f (pages %d-%d)", paintOp.Text, paintOp.Y, opH, lo, hi)
		}
	}

	pages := map[int]bool{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			pages[int(op.Y/pageH)] = true
		}
	}

	if len(pages) < 2 {
		t.Fatalf("expected multi-page multicol, got pages %v", pages)
	}
}

//nolint:cyclop,funlen // table-driven CSS shorthand proof
func TestColumnRuleParse(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.a { column-rule: 4pt dashed red }
.outer { color: blue }
.b { column-rule-width: medium; column-rule-style: solid; column-rule-color: currentColor }
.c { column-rule-width: thin }
.d { column-rule-width: thick }
.e { column-rule-style: none }
.f { column-rule: dotted }
`)
	root := mustParse(t, `<html><body>
<div class="a">A</div>
<div class="outer"><div class="b">B</div></div>
<div class="c">C</div>
<div class="d">D</div>
<div class="e">E</div>
<div class="f">F</div>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)

	styleA := styleByClass(t, styles, "a")
	if !near(styleA.ColumnRuleWidth, 4) {
		t.Fatalf("a width=%.3f, want 4pt", styleA.ColumnRuleWidth)
	}

	if styleA.ColumnRuleStyle != borderStyleDashed {
		t.Fatalf("a style=%q, want dashed", styleA.ColumnRuleStyle)
	}

	if styleA.ColumnRuleColor != ([3]float64{1, 0, 0}) {
		t.Fatalf("a color=%v, want red", styleA.ColumnRuleColor)
	}

	styleB := styleByClass(t, styles, "b")
	if !near(styleB.ColumnRuleWidth, borderWidth(mediumKeyword, 12)) {
		t.Fatalf("b width:medium = %.3f", styleB.ColumnRuleWidth)
	}

	if styleB.ColumnRuleStyle != solidKeyword {
		t.Fatalf("b style=%q, want solid", styleB.ColumnRuleStyle)
	}

	if styleB.ColumnRuleColor != ([3]float64{0, 0, 1}) {
		t.Fatalf("b currentColor=%v, want blue", styleB.ColumnRuleColor)
	}

	if !near(styleByClass(t, styles, "c").ColumnRuleWidth, borderWidth(thinKeyword, 12)) {
		t.Fatalf("c thin width=%.3f", styleByClass(t, styles, "c").ColumnRuleWidth)
	}

	if !near(styleByClass(t, styles, "d").ColumnRuleWidth, borderWidth(thickKeyword, 12)) {
		t.Fatalf("d thick width=%.3f", styleByClass(t, styles, "d").ColumnRuleWidth)
	}

	if styleByClass(t, styles, "e").ColumnRuleStyle != cssDisplayNone {
		t.Fatalf("e style=%q, want none", styleByClass(t, styles, "e").ColumnRuleStyle)
	}

	styleF := styleByClass(t, styles, "f")
	if styleF.ColumnRuleStyle != borderStyleDotted {
		t.Fatalf("f shorthand style=%q, want dotted", styleF.ColumnRuleStyle)
	}

	if !near(styleF.ColumnRuleWidth, borderWidth(mediumKeyword, 12)) {
		t.Fatalf("f shorthand width=%.3f, want medium", styleF.ColumnRuleWidth)
	}
}

// paint operation proof covers rule variants
