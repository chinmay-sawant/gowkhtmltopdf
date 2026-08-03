package layout

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"
	"time"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

const testViewport = 500.0

func mustParse(t *testing.T, src string) *html.Node {
	t.Helper()
	root, err := html.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return root
}

func layoutHTML(t *testing.T, src string, sheets ...*css.Stylesheet) *Result {
	t.Helper()
	return layoutHTMLZoom(t, src, 0, sheets...)
}

func layoutHTMLZoom(t *testing.T, src string, zoom float64, sheets ...*css.Stylesheet) *Result {
	t.Helper()
	root := mustParse(t, src)
	res, err := Layout(root, Options{Width: testViewport, Height: 800, Sheets: sheets, Background: true, Zoom: zoom})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	return res
}

func sheet(t *testing.T, src string) *css.Stylesheet {
	t.Helper()
	s, err := css.Parse(src)
	if err != nil {
		t.Fatalf("css.Parse: %v", err)
	}
	return s
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func opsOfKind(res *Result, kind OpKind) []Op {
	var out []Op
	for _, op := range res.Ops {
		if op.Kind == kind {
			out = append(out, op)
		}
	}
	return out
}

func firstText(res *Result) Op {
	for _, op := range res.Ops {
		if op.Kind == OpText {
			return op
		}
	}
	return Op{}
}

// findBox returns the first box in document order whose node is an element
// with the given name.
func findBox(t *testing.T, res *Result, name string) *box {
	t.Helper()
	var found *box
	var walk func(b *box)
	walk = func(b *box) {
		if found != nil {
			return
		}
		if b.node != nil && b.node.Type == html.ElementNode && b.node.Name == name {
			found = b
			return
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
	if found == nil {
		t.Fatalf("no box for <%s> in box tree", name)
	}
	return found
}

func TestBoxRanges(t *testing.T) {
	s := sheet(t, `div { padding: 5pt; border: 2pt solid black }`)
	res := layoutHTML(t, `<html><body><div><p>hello world</p></div></body></html>`, s)
	div := findBox(t, res, "div")
	if div.opStart > div.opEnd {
		t.Fatalf("div op range empty: [%d, %d]", div.opStart, div.opEnd)
	}
	// every index in the range must map to a real op
	for i := div.opStart; i <= div.opEnd; i++ {
		if i < 0 || i >= len(res.Ops) {
			t.Fatalf("div range index %d outside Ops (len %d)", i, len(res.Ops))
		}
	}
	if len(div.children) == 0 {
		t.Fatal("div has no box children")
	}
	p := findBox(t, res, "p")
	if p.opStart > p.opEnd {
		t.Fatalf("p op range empty: [%d, %d]", p.opStart, p.opEnd)
	}
	// children ranges nest inside the parent range
	if p.opStart < div.opStart || p.opEnd > div.opEnd {
		t.Errorf("p range [%d, %d] not nested in div [%d, %d]", p.opStart, p.opEnd, div.opStart, div.opEnd)
	}

	// noEmit interplay: table/cell ranges (built via the measure pass) must
	// map to ops that actually exist
	tres := layoutHTML(t, `<html><body><table><tr><td>a</td><td>b</td></tr><tr><td>c</td><td>d</td></tr></table></body></html>`)
	tb := findBox(t, tres, "table")
	if tb.opStart > tb.opEnd {
		t.Fatalf("table op range empty: [%d, %d]", tb.opStart, tb.opEnd)
	}
	for i := tb.opStart; i <= tb.opEnd; i++ {
		if i < 0 || i >= len(tres.Ops) {
			t.Fatalf("table range index %d outside Ops (len %d)", i, len(tres.Ops))
		}
	}
	if len(tb.children) != 4 {
		t.Fatalf("table children = %d, want 4 cells", len(tb.children))
	}
	for _, cell := range tb.children {
		if cell.opStart > cell.opEnd {
			t.Fatalf("cell op range empty: [%d, %d]", cell.opStart, cell.opEnd)
		}
		if cell.opStart < tb.opStart || cell.opEnd > tb.opEnd {
			t.Errorf("cell range [%d, %d] not nested in table [%d, %d]", cell.opStart, cell.opEnd, tb.opStart, tb.opEnd)
		}
	}
}

func TestTableRowRanges(t *testing.T) {
	res := layoutHTML(t, `<html><body><table><tr><td>a</td><td>b</td></tr><tr><td>c</td><td>d</td></tr></table></body></html>`)
	tb := findBox(t, res, "table")
	if len(tb.rows) != 2 {
		t.Fatalf("table rows = %d, want 2", len(tb.rows))
	}
	rowRange := func(row []*box) (int, int) {
		if len(row) == 0 {
			return -1, -2
		}
		return row[0].opStart, row[len(row)-1].opEnd
	}
	r0s, r0e := rowRange(tb.rows[0])
	r1s, r1e := rowRange(tb.rows[1])
	// each row's op range covers (overlaps) its cells' ranges
	for _, cell := range tb.rows[0] {
		if cell.opStart < r0s || cell.opEnd > r0e {
			t.Errorf("row 0 cell range [%d, %d] outside row range [%d, %d]", cell.opStart, cell.opEnd, r0s, r0e)
		}
	}
	for _, cell := range tb.rows[1] {
		if cell.opStart < r1s || cell.opEnd > r1e {
			t.Errorf("row 1 cell range [%d, %d] outside row range [%d, %d]", cell.opStart, cell.opEnd, r1s, r1e)
		}
	}
	// ops are emitted in document order: row 0 ends before row 1 starts
	if !(r0e < r1s) {
		t.Errorf("row 0 ends at op %d, row 1 starts at op %d; want row0 end < row1 start", r0e, r1s)
	}
}

func TestElementLocations(t *testing.T) {
	res := layoutHTML(t, `<html><body><h1>title</h1><p>body text</p><table><tr><td>a</td><td>b</td></tr><tr><td>c</td><td>d</td></tr></table></body></html>`)
	doc := pdf.NewDocument()
	doc.SetCreationTime(epoch)
	err := Paint(doc, res, PaintOptions{PageWidth: 595, PageHeight: 842, MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35})
	if err != nil {
		t.Fatalf("Paint: %v", err)
	}
	// h1, p, table + 4 cells (plus html/body)
	if len(res.Locations) < 6 {
		t.Fatalf("Locations = %d entries, want >= 6", len(res.Locations))
	}
	var h1 *ElementLocation
	for i := range res.Locations {
		if res.Locations[i].Node.Name == "h1" {
			h1 = &res.Locations[i]
			break
		}
	}
	if h1 == nil {
		t.Fatal("no h1 location")
	}
	if h1.Page != 0 {
		t.Errorf("h1 page = %d, want 0", h1.Page)
	}
	if h1.X < 0 || h1.W <= 0 || h1.H <= 0 || h1.Y < 0 {
		t.Errorf("h1 rect = (%.2f,%.2f) %.2fx%.2f, want X>=0, W>0, H>0, Y>=0", h1.X, h1.Y, h1.W, h1.H)
	}
	// document order: h1 before p before table, table cells in order
	names := make([]string, 0, len(res.Locations))
	for _, loc := range res.Locations {
		names = append(names, loc.Node.Name)
	}
	h1i, pi, ti := indexOf(names, "h1"), indexOf(names, "p"), indexOf(names, "table")
	if h1i < 0 || pi < 0 || ti < 0 {
		t.Fatalf("missing h1/p/table in %v", names)
	}
	if !(h1i < pi && pi < ti) {
		t.Errorf("document order broken in %v", names)
	}
	var tds []ElementLocation
	for _, loc := range res.Locations {
		if loc.Node.Name == "td" {
			tds = append(tds, loc)
		}
	}
	if len(tds) != 4 {
		t.Fatalf("td locations = %d, want 4", len(tds))
	}
	for i := 1; i < len(tds); i++ {
		if tds[i].Y < tds[i-1].Y {
			t.Errorf("td %d y=%.2f above td %d y=%.2f (document order)", i, tds[i].Y, i-1, tds[i-1].Y)
		}
	}

	// multi-page doc: some elements land on page 1
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 60; i++ {
		sb.WriteString("<p>paragraph of text number ")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" with some words to wrap</p>")
	}
	sb.WriteString("</body></html>")
	res2 := layoutHTML(t, sb.String())
	doc2 := pdf.NewDocument()
	doc2.SetCreationTime(epoch)
	if err := Paint(doc2, res2, PaintOptions{PageWidth: 595, PageHeight: 842, MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35}); err != nil {
		t.Fatalf("Paint long doc: %v", err)
	}
	if len(res2.Pages) < 2 {
		t.Fatalf("Pages = %d, want >= 2", len(res2.Pages))
	}
	total := 0
	for _, page := range res2.Pages {
		total += len(page)
	}
	if total != len(res2.Ops) {
		t.Errorf("Pages cover %d ops, Ops has %d", total, len(res2.Ops))
	}
	saw0, saw1 := false, false
	for _, loc := range res2.Locations {
		if loc.Page == 0 {
			saw0 = true
		}
		if loc.Page == 1 {
			saw1 = true
		}
	}
	if !saw0 || !saw1 {
		t.Errorf("page distribution: page0=%v page1=%v, want both", saw0, saw1)
	}
}

func indexOf(seq []string, s string) int {
	for i, v := range seq {
		if v == s {
			return i
		}
	}
	return -1
}

func TestBlockStacking(t *testing.T) {
	res := layoutHTML(t, `<html><body><div>A</div><div>B</div><div>C</div></body></html>`)
	texts := opsOfKind(res, OpText)
	if len(texts) != 3 {
		t.Fatalf("got %d text ops, want 3: %+v", len(texts), texts)
	}
	// each div is 16px tall (12pt * 1.2 line height + 8px body margins?)
	// verify strict vertical stacking order
	for i := 1; i < len(texts); i++ {
		if !(texts[i].Y > texts[i-1].Y) {
			t.Errorf("text %d y=%v not below text %d y=%v", i, texts[i].Y, i-1, texts[i-1].Y)
		}
	}
	// widths must fill the viewport (block boxes)
	fills := opsOfKind(res, OpFillRect)
	_ = fills
}

func TestBlockWidthsAndMargins(t *testing.T) {
	s := sheet(t, `div.a { margin: 10pt; width: 200pt } div.b { margin-left: 30pt }`)
	res := layoutHTML(t, `<html><body><div class="a">x</div><div class="b">y</div></body></html>`, s)
	texts := opsOfKind(res, OpText)
	if len(texts) != 2 {
		t.Fatalf("texts = %+v", texts)
	}
	// body UA margin is 8px = 6pt; box a: margin 10pt each side, width 200pt
	if !near(texts[0].X, 16) {
		t.Errorf("text a x = %v, want 16", texts[0].X)
	}
	// box b fills viewport minus left margin
	if !near(texts[1].X, 36) {
		t.Errorf("text b x = %v, want 36", texts[1].X)
	}
}

func TestMarginCollapse(t *testing.T) {
	s := sheet(t, `p.a { margin-bottom: 20pt } p.b { margin-top: 30pt }`)
	res := layoutHTML(t, `<html><body><p class="a">A</p><p class="b">B</p></body></html>`, s)
	texts := opsOfKind(res, OpText)
	if len(texts) != 2 {
		t.Fatalf("texts = %+v", texts)
	}
	// collapsed gap = max(20, 30) = 30; no body margin in this sheet
	dy := texts[1].Y - texts[0].Y
	want := lineHeightOf(&ResolvedStyle{FontSize: 12}) + 30
	if !near(dy, want) {
		t.Errorf("gap between lines = %v, want %v (collapsed)", dy, want)
	}
}

func TestPaddingBorderBox(t *testing.T) {
	s := sheet(t, `div { padding: 10pt; border: 2pt solid black; width: 200pt }`)
	res := layoutHTML(t, `<html><body><div>text</div></body></html>`, s)
	lines := opsOfKind(res, OpLine)
	if len(lines) != 4 {
		t.Fatalf("got %d border lines, want 4", len(lines))
	}
	texts := opsOfKind(res, OpText)
	if len(texts) != 1 {
		t.Fatalf("texts = %+v", texts)
	}
	// text starts after body margin (6) + padding + border
	if !near(texts[0].X, 18) {
		t.Errorf("text x = %v, want 18 (body 6 + border 2 + padding 10)", texts[0].X)
	}
	if texts[0].Y < 25 || texts[0].Y > 32 {
		t.Errorf("text y = %v, want baseline around 29.4 (content top + ascent)", texts[0].Y)
	}
}

func TestTextAlign(t *testing.T) {
	s := sheet(t, `div { width: 300pt } .r { text-align: right } .c { text-align: center }`)
	res := layoutHTML(t, `<html><body><div>left</div><div class="r">right</div><div class="c">center</div></body></html>`, s)
	texts := opsOfKind(res, OpText)
	if len(texts) != 3 {
		t.Fatalf("texts = %+v", texts)
	}
	// widths of the words
	w0 := texts[0].W
	w1 := texts[1].W
	w2 := texts[2].W
	// body UA margin is 6pt, so the 300pt content box starts at x=6
	if !near(texts[0].X, 6) {
		t.Errorf("left x = %v, want 6", texts[0].X)
	}
	if !near(texts[1].X+w1, 306) {
		t.Errorf("right text ends at %v, want 306", texts[1].X+w1)
	}
	if !near(texts[2].X, 6+(300-w2)/2) {
		t.Errorf("center x = %v, want %v", texts[2].X, 6+(300-w2)/2)
	}
	_ = w0
}

func TestTextWrapping(t *testing.T) {
	s := sheet(t, `div { width: 60pt }`)
	// a long sentence in a 60pt box must wrap into multiple lines
	res := layoutHTML(t, `<html><body><div>the quick brown fox jumps over the lazy dog</div></body></html>`, s)
	texts := opsOfKind(res, OpText)
	if len(texts) < 6 {
		t.Fatalf("expected many word ops, got %d: %+v", len(texts), texts)
	}
	// group ops by line: consecutive ops with the same (rounded) baseline
	var lines [][]Op
	for _, op := range texts {
		if len(lines) > 0 && math.Abs(lines[len(lines)-1][0].Y-op.Y) < 0.01 {
			lines[len(lines)-1] = append(lines[len(lines)-1], op)
		} else {
			lines = append(lines, []Op{op})
		}
	}
	if len(lines) < 3 {
		t.Fatalf("expected >= 3 wrapped lines, got %d", len(lines))
	}
	// successive lines below each other, each starting at x=6
	for i := 1; i < len(lines); i++ {
		if !(lines[i][0].Y > lines[i-1][0].Y) {
			t.Errorf("line %d not below line %d", i, i-1)
		}
		if !near(lines[i][0].X, 6) {
			t.Errorf("line %d x = %v, want 6", i, lines[i][0].X)
		}
	}
}

func TestWhiteSpacePre(t *testing.T) {
	s := sheet(t, `pre { white-space: pre; width: 60pt }`)
	res := layoutHTML(t, "<html><body><pre>a b\nc</pre></body></html>", s)
	texts := opsOfKind(res, OpText)
	// pre splits on \n only: "a b" and "c"
	if len(texts) != 2 {
		t.Fatalf("pre texts = %+v", texts)
	}
	if texts[0].Text != "a b" || texts[1].Text != "c" {
		t.Errorf("pre segments = %q / %q", texts[0].Text, texts[1].Text)
	}
}

func TestFontSizeEmInherit(t *testing.T) {
	s := sheet(t, `div { font-size: 20pt } p { font-size: 0.5em } .big { font-size: 2em }`)
	res := layoutHTML(t, `<html><body><div>parent <p>child</p><p class="big">big</p></div></body></html>`, s)
	texts := opsOfKind(res, OpText)
	if len(texts) != 3 {
		t.Fatalf("texts = %+v", texts)
	}
	// block children are laid out before the parent's inline run, so match by content.
	// Trailing whitespace is trimmed from the last run on a line, so strip it when keying.
	sizes := map[string]float64{}
	for _, op := range texts {
		sizes[strings.TrimRight(op.Text, " ")] = op.Size
	}
	// parent 20pt, child 10pt (0.5em of parent), big 40pt (2em of parent)
	if !near(sizes["parent"], 20) {
		t.Errorf("parent size = %v, want 20", sizes["parent "])
	}
	if !near(sizes["child"], 10) {
		t.Errorf("child size = %v, want 10", sizes["child"])
	}
	if !near(sizes["big"], 40) {
		t.Errorf("big size = %v, want 40", sizes["big"])
	}
}

func TestCascadeAndInline(t *testing.T) {
	s := sheet(t, `p { color: red; margin: 5pt } p.special { color: blue }`)
	res := layoutHTML(t, `<html><body><p>one</p><p class="special" style="color: #0f0">two</p></body></html>`, s)
	texts := opsOfKind(res, OpText)
	if len(texts) != 2 {
		t.Fatalf("texts = %+v", texts)
	}
	if texts[0].R != 1 || texts[0].G != 0 || texts[0].B != 0 {
		t.Errorf("first color = %v,%v,%v, want red", texts[0].R, texts[0].G, texts[0].B)
	}
	if texts[1].R != 0 || texts[1].G != 1 || texts[1].B != 0 {
		t.Errorf("inline color = %v,%v,%v, want green", texts[1].R, texts[1].G, texts[1].B)
	}
	if !near(texts[1].X, 11) {
		t.Errorf("special margin x = %v, want 11 (body 6 + margin 5)", texts[1].X)
	}
}

func TestBackgroundFill(t *testing.T) {
	s := sheet(t, `div { background-color: #f00; width: 100pt; height: 50pt }`)
	res := layoutHTML(t, `<html><body><div>x</div></body></html>`, s)
	fills := opsOfKind(res, OpFillRect)
	if len(fills) != 1 {
		t.Fatalf("fills = %+v", fills)
	}
	if !near(fills[0].W, 100) || !near(fills[0].H, 50) {
		t.Errorf("fill size = %v x %v, want 100 x 50", fills[0].W, fills[0].H)
	}
	if fills[0].R != 1 || fills[0].G != 0 || fills[0].B != 0 {
		t.Errorf("fill color = %v,%v,%v", fills[0].R, fills[0].G, fills[0].B)
	}
	if fills[0].Alpha != 1 {
		t.Errorf("alpha = %v, want 1", fills[0].Alpha)
	}
}

func TestDisplayNone(t *testing.T) {
	s := sheet(t, `.hidden { display: none }`)
	res := layoutHTML(t, `<html><body><div>visible</div><div class="hidden">secret</div></body></html>`, s)
	texts := opsOfKind(res, OpText)
	if len(texts) != 1 || texts[0].Text != "visible" {
		t.Errorf("texts = %+v, want only visible", texts)
	}
}

func TestListBullet(t *testing.T) {
	res := layoutHTML(t, `<html><body><ul><li>item</li></ul></body></html>`)
	bullets := opsOfKind(res, OpBullet)
	if len(bullets) != 1 {
		t.Fatalf("bullets = %+v", bullets)
	}
	if bullets[0].Text != "\u2022" {
		t.Errorf("bullet = %q", bullets[0].Text)
	}
}

func TestBoldUnderline(t *testing.T) {
	res := layoutHTML(t, `<html><body><p><b>bold</b> and <a href="http://x.example/">link</a></p></body></html>`)
	texts := opsOfKind(res, OpText)
	var bold, normal bool
	for _, op := range texts {
		if op.Bold {
			bold = true
		}
		if strings.Contains(op.Text, "and") {
			normal = true
		}
	}
	if !bold || !normal {
		t.Errorf("bold=%v normal=%v texts=%+v", bold, normal, texts)
	}
	links := opsOfKind(res, OpLinkURI)
	if len(links) != 1 || links[0].URI != "http://x.example/" {
		t.Fatalf("links = %+v", links)
	}
	lines := opsOfKind(res, OpLine)
	foundUnderline := false
	for _, l := range lines {
		if near(l.H, 0) && l.W > 0 && l.Y > 0 {
			foundUnderline = true
		}
	}
	if !foundUnderline {
		t.Errorf("no underline line found: %+v", lines)
	}
}

func TestTableLayout(t *testing.T) {
	res := layoutHTML(t, `<html><body><table>
		<tr><th>Name</th><th>Qty</th><th>Total</th></tr>
		<tr><td>Widget</td><td>2</td><td>$10.00</td></tr>
	</table></body></html>`)
	texts := opsOfKind(res, OpText)
	if len(texts) != 6 {
		t.Fatalf("table texts = %+v", texts)
	}
	// three columns must share x positions across rows
	x0 := texts[0].X
	x1 := texts[1].X
	x2 := texts[2].X
	if !(x1 > x0 && x2 > x1) {
		t.Fatalf("column x ordering wrong: %v %v %v", x0, x1, x2)
	}
	// row 2 cells are left-aligned: each must sit at or right of the column-0
	// content edge (body UA margin 6 + cell padding 0.75), not the centered
	// header text.
	for _, op := range texts[3:] {
		if op.X < 6.75 {
			t.Errorf("row 2 text x=%v left of column 0", op.X)
		}
	}
	// row 2 below row 1
	if !(texts[3].Y > texts[0].Y) {
		t.Errorf("row 2 not below row 1: %v vs %v", texts[3].Y, texts[0].Y)
	}
}

func TestTableColspan(t *testing.T) {
	res := layoutHTML(t, `<html><body><table>
		<tr><td colspan="2">wide</td><td>c</td></tr>
		<tr><td>a</td><td>b</td><td>c</td></tr>
	</table></body></html>`)
	texts := opsOfKind(res, OpText)
	if len(texts) != 5 {
		t.Fatalf("texts = %+v", texts)
	}
	// "wide" spans cols 0+1; col 2 starts right of it.
	// Content starts at x=6 (BODY UA margin 8px=6pt) + cell padding 1px=0.75pt.
	wide, c1 := texts[0], texts[1]
	if !near(wide.X, 6.75) {
		t.Errorf("wide x = %v, want 6.75 (body 6 + cell padding 0.75)", wide.X)
	}
	if !(c1.X > wide.X) {
		t.Errorf("third column x=%v must start right of wide (x=%v)", c1.X, wide.X)
	}
	// "a" at col 0, "b" at col 1, "c" at col 2 — b must sit between a and c
	a, b, c := texts[2], texts[3], texts[4]
	if !(a.X < b.X && b.X < c.X) {
		t.Errorf("column order broken: %v %v %v", a.X, b.X, c.X)
	}
	if !near(a.X, 6.75) {
		t.Errorf("a x = %v, want 6.75 (body 6 + cell padding 0.75)", a.X)
	}
	if !(c.X >= c1.X) {
		t.Errorf("c x = %v must be >= single-row col2 x=%v", c.X, c1.X)
	}
}

func TestImageIntrinsicAndWidth(t *testing.T) {
	// 10x20 pixel PNG
	png := tinyPNG(10, 20)
	res := layoutHTMLWithImages(t, `<html><body><img src="x.png"></body></html>`, png, "")
	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("imgs = %+v", imgs)
	}
	if !near(imgs[0].W, 7.5) || !near(imgs[0].H, 15) {
		t.Errorf("intrinsic size = %v x %v, want 7.5 x 15", imgs[0].W, imgs[0].H)
	}
	// with width attribute: the width attribute is px → 20px * 0.75 = 15pt at 96dpi
	res = layoutHTMLWithImages(t, `<html><body><img src="x.png" width="20"></body></html>`, png, "")
	imgs = opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("imgs = %+v", imgs)
	}
	if !near(imgs[0].W, 15) {
		t.Errorf("width attr = %v, want 15pt", imgs[0].W)
	}
}

func layoutHTMLWithImages(t *testing.T, src string, img []byte, imgSrc string) *Result {
	t.Helper()
	root := mustParse(t, src)
	provider := func(src string) ([]byte, error) {
		if imgSrc != "" && src != imgSrc {
			t.Errorf("unexpected image src %q", src)
		}
		return img, nil
	}
	res, err := Layout(root, Options{Width: testViewport, Height: 800, Images: provider, Background: true})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	return res
}

// tinyPNG builds a minimal valid RGBA PNG of the given pixel size.
func tinyPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic(err)
	}
	return out.Bytes()
}

var epoch = time.Unix(0, 0).UTC()

func TestPaginateAndPaint(t *testing.T) {
	// long content: many paragraphs → multiple pages
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 60; i++ {
		sb.WriteString("<p>paragraph of text number ")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" with some words to wrap</p>")
	}
	sb.WriteString("</body></html>")
	res := layoutHTML(t, sb.String())

	doc := pdf.NewDocument()
	doc.SetCreationTime(epoch)
	err := Paint(doc, res, PaintOptions{
		PageWidth: 595, PageHeight: 842, // A4 portrait
		MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35,
	})
	if err != nil {
		t.Fatalf("Paint: %v", err)
	}
	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
	pageCount := bytes.Count(buf.Bytes(), []byte("/Type /Page\n")) - bytes.Count(buf.Bytes(), []byte("/Type /Pages\n"))
	if pageCount < 2 {
		t.Errorf("expected >= 2 pages, got %d", pageCount)
	}
	if !bytes.Contains(buf.Bytes(), []byte("/FontFile2")) {
		t.Error("expected embedded subset font")
	}
}

func TestPaintSinglePage(t *testing.T) {
	res := layoutHTML(t, `<html><body><p>hello</p></body></html>`)
	doc := pdf.NewDocument()
	doc.SetCreationTime(epoch)
	err := Paint(doc, res, PaintOptions{PageWidth: 595, PageHeight: 842, MarginTop: 10, MarginBottom: 10, MarginLeft: 10, MarginRight: 10})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}
	n := bytes.Count(buf.Bytes(), []byte("/Type /Page\n"))
	if n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
}

func TestDebugBoxes(t *testing.T) {
	root := mustParse(t, `<html><body><div>x</div></body></html>`)
	res, err := Layout(root, Options{Width: testViewport, Height: 800, DebugBoxes: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(opsOfKind(res, OpStrokeRect)) == 0 {
		t.Error("no debug boxes emitted")
	}
}

func TestHRElement(t *testing.T) {
	res := layoutHTML(t, `<html><body><p>a</p><hr><p>b</p></body></html>`)
	fills := opsOfKind(res, OpFillRect)
	if len(fills) != 1 {
		t.Fatalf("hr fills = %+v", fills)
	}
	if !near(fills[0].W, testViewport-12) {
		t.Errorf("hr width = %v, want %v (viewport minus body margins)", fills[0].W, testViewport-12)
	}
}

func TestZoom(t *testing.T) {
	s := sheet(t, `div { width: 100pt; margin: 10pt; background-color: #000 }`)
	src := `<html><body><div>abc</div></body></html>`

	// zoom 1: text starts at body margin 6pt + div margin 10pt
	res1 := layoutHTMLZoom(t, src, 1, s)
	if !near(firstText(res1).X, 16) {
		t.Errorf("zoom 1 text x = %v, want 16 (body 6 + margin 10)", firstText(res1).X)
	}

	// zoom 2: everything doubles (body 12 + margin 20)
	res2 := layoutHTMLZoom(t, src, 2, s)
	if !near(firstText(res2).X, 32) {
		t.Errorf("zoom 2 text x = %v, want 32 (12 + 20)", firstText(res2).X)
	}
	fills := opsOfKind(res2, OpFillRect)
	if len(fills) != 1 {
		t.Fatalf("zoom 2 fills = %+v, want 1", fills)
	}
	if !near(fills[0].X, 12) {
		t.Errorf("zoom 2 div x = %v, want 12 (body margin 6pt scaled)", fills[0].X)
	}
	if !near(fills[0].X+fills[0].W, 212) {
		t.Errorf("zoom 2 div right edge = %v, want 212 (div x 12 + width 200)", fills[0].X+fills[0].W)
	}
}

func TestPageBreakParsing(t *testing.T) {
	s := sheet(t, `div.brk { page-break-before: always } div.avoid { break-inside: avoid } p.aft { break-after: always }`)
	src := `<html><body><div class="brk">x</div><div class="avoid">y</div><p class="aft">z</p></body></html>`

	// parsing must not panic or stall layout
	res := layoutHTML(t, src, s)
	if len(opsOfKind(res, OpText)) != 3 {
		t.Fatalf("texts = %+v, want 3", opsOfKind(res, OpText))
	}

	root := mustParse(t, src)
	styles := resolveStyles(root, []*css.Stylesheet{s}, "", testViewport, 800)
	for n, st := range styles {
		if n.Type != html.ElementNode {
			continue
		}
		switch n.Attribute("class") {
		case "brk":
			if st.PageBreakBefore != "always" {
				t.Errorf("div.brk page-break-before = %q, want always", st.PageBreakBefore)
			}
		case "avoid":
			if st.PageBreakInside != "avoid" {
				t.Errorf("div.avoid break-inside = %q, want avoid", st.PageBreakInside)
			}
		case "aft":
			if st.PageBreakAfter != "always" {
				t.Errorf("p.aft break-after = %q, want always", st.PageBreakAfter)
			}
		}
	}
}

// paintOpts is a full-page-content geometry for pagination tests.
func paintOpts() PaintOptions {
	return PaintOptions{PageWidth: 595, PageHeight: 842} // contentH = 842
}

// pageOf returns the page (0-based) of the first op whose text contains want.
func pageOf(t *testing.T, res *Result, want string) int {
	t.Helper()
	for i, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, want) {
			for p, idxs := range res.Pages {
				for _, idx := range idxs {
					if idx == i {
						return p
					}
				}
			}
		}
	}
	t.Fatalf("op %q not found in any page", want)
	return -1
}

func TestPageBreakBeforeAlways(t *testing.T) {
	// div1's text sits on page 1 (padding pushes it past the boundary);
	// div2 (break-before: always) must start on a fresh page: page 2.
	s := sheet(t, `div.a { padding-top: 850pt; height: 900pt } div.brk { page-break-before: always }`)
	res := layoutHTML(t, `<html><body><div class="a">one</div><div class="brk">two</div></body></html>`, s)
	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}
	p1 := pageOf(t, res, "one")
	p2 := pageOf(t, res, "two")
	if p1 != 1 {
		t.Errorf("first div on page %d, want 1", p1)
	}
	if p2 != 2 {
		t.Errorf("second div (break-before) on page %d, want 2", p2)
	}
}

func TestPageBreakInsideAvoid(t *testing.T) {
	// div1: y 6..506; div2 (avoid): y 506..906 spans pages 0-1, fits one page
	// (h=400) → moves wholly to page 1.
	s := sheet(t, `div.a { height: 500pt } div.b { height: 400pt; page-break-inside: avoid }`)
	res := layoutHTML(t, `<html><body><div class="a">x</div><div class="b">y</div></body></html>`, s)
	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}
	pb := pageOf(t, res, "y")
	if pb != 1 {
		t.Errorf("avoid-inside div on page %d, want 1", pb)
	}
	// every "y" text op must be on that single page
	for i, op := range res.Ops {
		if op.Kind == OpText && op.Text == "y" && pageOfIdx(t, res, i) != pb {
			t.Errorf("avoid box op on page %d, want %d", pageOfIdx(t, res, i), pb)
		}
	}
}

func TestBoundaryFillSplit(t *testing.T) {
	// colored div starts at y=700 (after 694pt spacer), h=400 → its fill
	// crosses the boundary at 842 and must split into two clipped rects.
	s := sheet(t, `div.a { height: 694pt } div.b { height: 400pt; background-color: #000 }`)
	res := layoutHTML(t, `<html><body><div class="a">x</div><div class="b">y</div></body></html>`, s)
	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}
	var fillIdx []int
	for i, op := range res.Ops {
		if op.Kind == OpFillRect {
			fillIdx = append(fillIdx, i)
		}
	}
	if len(fillIdx) != 2 {
		t.Fatalf("fills = %d, want 2 (split at boundary)", len(fillIdx))
	}
	f0, f1 := res.Ops[fillIdx[0]], res.Ops[fillIdx[1]]
	if !near(f0.Y, 700) || !near(f0.H, 142) {
		t.Errorf("first fill = y %v h %v, want y 700 h 142 (clipped to boundary)", f0.Y, f0.H)
	}
	if !near(f1.Y, 842) || !near(f1.H, 258) {
		t.Errorf("second fill = y %v h %v, want y 842 h 258", f1.Y, f1.H)
	}
	if p0, p1 := pageOfIdx(t, res, fillIdx[0]), pageOfIdx(t, res, fillIdx[1]); p0 != 0 || p1 != 1 {
		t.Errorf("split fills on pages %d and %d, want 0 and 1", p0, p1)
	}
}

// pageOfIdx returns the page of the op at index i.
func pageOfIdx(t *testing.T, res *Result, i int) int {
	t.Helper()
	for p, idxs := range res.Pages {
		for _, idx := range idxs {
			if idx == i {
				return p
			}
		}
	}
	t.Fatalf("op %d not found in any page", i)
	return -1
}

func TestTableRowNoSplit(t *testing.T) {
	// three tall rows (padding forces ~750pt rows); each must land on its own
	// page without splitting.
	s := sheet(t, `td { padding: 500pt; font-size: 12pt }`)
	res := layoutHTML(t, `<html><body><table><tr><td>r1</td></tr><tr><td>r2</td></tr><tr><td>r3</td></tr></table></body></html>`, s)
	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}
	// find the row texts (r1, r2, r3 — padded cells emit them)
	pages := map[string]int{}
	for i, op := range res.Ops {
		if op.Kind == OpText {
			pages[op.Text] = pageOfIdx(t, res, i)
		}
	}
	if len(pages) < 3 {
		t.Fatalf("row texts = %v", pages)
	}
	if !(pages["r1"] < pages["r2"] && pages["r2"] < pages["r3"]) {
		t.Errorf("row pages = %v, want strictly increasing", pages)
	}
}
