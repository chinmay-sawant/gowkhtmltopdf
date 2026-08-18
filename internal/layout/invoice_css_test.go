//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"strings"
	"testing"
)

func TestBoxSizingBorderBox(t *testing.T) {
	t.Parallel()
	// content-box (default): width is content; border box grows by pad+border.
	s := sheet(t, `.a { width: 100pt; padding: 10pt; border: 5pt solid black }
.b { width: 100pt; padding: 10pt; border: 5pt solid black; box-sizing: border-box }`)
	res := layoutHTML(t, `<html><body>
<div class="a">a</div><div class="b">b</div>
</body></html>`, s)

	var boxes []*box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.node != nil && boxNode.node.Name == divElementName {
			if cls := boxNode.node.Attribute("class"); cls == "a" || cls == "b" {
				boxes = append(boxes, boxNode)
			}
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if len(boxes) != 2 {
		t.Fatalf("div boxes = %d, want 2", len(boxes))
	}
	// content-box: 100 + 10+10 + 5+5 = 130
	if !near(boxes[0].w, 130) {
		t.Errorf("content-box width = %v, want 130", boxes[0].w)
	}
	// border-box: 100
	if !near(boxes[1].w, 100) {
		t.Errorf("border-box width = %v, want 100", boxes[1].w)
	}
}

func TestInlineBlockBesideText(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.badge { display: inline-block; width: 40pt; padding: 2pt; border: 1pt solid black;
  box-sizing: border-box; text-align: center }
p { width: 300pt }`)
	res := layoutHTML(t, `<html><body>
<p>Hello <span class="badge">NEW</span> world</p>
</body></html>`, cssSheet)
	texts := opsOfKind(res, OpText)

	var hello, badge, world *Op

	for i := range texts {
		paintOp := &texts[i]

		switch {
		case strings.HasPrefix(strings.TrimSpace(paintOp.Text), "Hello"):
			hello = paintOp
		case strings.HasPrefix(strings.TrimSpace(paintOp.Text), "NEW"):
			badge = paintOp
		case strings.HasPrefix(strings.TrimSpace(paintOp.Text), "world"):
			world = paintOp
		}
	}

	if hello == nil || badge == nil || world == nil {
		t.Fatalf("missing text ops hello=%v badge=%v world=%v in %+v", hello != nil, badge != nil, world != nil, texts)
	}
	// Badge sits on the same baseline band as surrounding text.
	if math.Abs(hello.Y-world.Y) > 0.5 {
		t.Errorf("hello/world baselines diverge: %v vs %v", hello.Y, world.Y)
	}

	if math.Abs(hello.Y-badge.Y) > hello.H+2 {
		t.Errorf("badge baseline %v far from hello %v", badge.Y, hello.Y)
	}

	if badge.X < hello.X+hello.W-1 {
		t.Errorf("badge x=%v should be after hello (x=%v w=%v)", badge.X, hello.X, hello.W)
	}

	if world.X < badge.X {
		t.Errorf("world x=%v should be after badge x=%v", world.X, badge.X)
	}
}

func TestFloatLeftRightClear(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.logo { float: left; width: 60pt; height: 40pt; background-color: #ccc }
.meta { float: right; width: 80pt; height: 30pt; background-color: #eee }
.clear { clear: both }
body { width: 400pt }`)
	res := layoutHTML(t, `<html><body>
<div class="logo">L</div>
<div class="meta">M</div>
<div class="clear">Below</div>
</body></html>`, cssSheet)

	var logo, meta, clearBox *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.node != nil && boxNode.node.Name == divElementName {
			switch boxNode.node.Attribute("class") {
			case "logo":
				logo = boxNode
			case "meta":
				meta = boxNode
			case "clear":
				clearBox = boxNode
			}
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if logo == nil || meta == nil || clearBox == nil {
		t.Fatalf("missing boxes logo=%v meta=%v clear=%v", logo != nil, meta != nil, clearBox != nil)
	}
	// Same band: logo left, meta right.
	if logo.x >= meta.x {
		t.Errorf("logo x=%v should be left of meta x=%v", logo.x, meta.x)
	}

	if math.Abs(logo.y-meta.y) > 1 {
		t.Errorf("logo/meta should share y band: %v vs %v", logo.y, meta.y)
	}
	// Clear sits below both floats.
	below := logo.y + logo.height
	if meta.y+meta.height > below {
		below = meta.y + meta.height
	}

	if clearBox.y+0.01 < below {
		t.Errorf("clear y=%v should be >= float bottoms %v", clearBox.y, below)
	}
}

func TestTextAlignJustify(t *testing.T) {
	t.Parallel()

	s := sheet(t, `div { width: 200pt; text-align: justify }`)
	// Long enough to wrap into multiple lines.
	res := layoutHTML(t, `<html><body><div>alpha bravo charlie delta echo foxtrot golf hotel india</div></body></html>`, s)

	texts := opsOfKind(res, OpText)
	if len(texts) < 2 {
		t.Fatalf("expected wrapped lines, got %d ops", len(texts))
	}
	// First line should be stretched toward the content edge (not clustered left).
	// With justify, the last word on a non-final line sits near the right edge.
	firstLine := []Op{}

	y0 := texts[0].Y
	for _, op := range texts {
		if math.Abs(op.Y-y0) < 0.01 {
			firstLine = append(firstLine, op)
		}
	}

	if len(firstLine) < 2 {
		t.Fatalf("first line should have multiple words, got %d", len(firstLine))
	}

	last := firstLine[len(firstLine)-1]
	right := last.X + last.W
	// body margin 6 + width 200 = 206 content right edge
	if right < 180 {
		t.Errorf("justified first line ends at %v, want near right edge (~206)", right)
	}
}

func TestTableCellVerticalAlignMiddle(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
td { border: 1pt solid black; padding: 0 }
.tall { height: 60pt }
.mid { vertical-align: middle }`)
	res := layoutHTML(t, `<html><body>
<table><tr>
<td class="tall">Hi<br>there<br>row</td>
<td class="mid">X</td>
</tr></table>
</body></html>`, cssSheet)
	texts := opsOfKind(res, OpText)

	var xOp *Op

	for i := range texts {
		if texts[i].Text == "X" || texts[i].Text == "X " {
			xOp = &texts[i]

			break
		}
	}

	if xOp == nil {
		t.Fatal("missing X text op")

		return
	}
	// Row is tall; middle-aligned "X" should sit well below the top cell padding.
	if xOp.Y < 25 {
		t.Errorf("middle-aligned cell text y=%v looks top-aligned", xOp.Y)
	}
}
