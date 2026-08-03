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
	root := mustParse(t, src)
	res, err := Layout(root, Options{Width: testViewport, Height: 800, Sheets: sheets, Background: true})
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
