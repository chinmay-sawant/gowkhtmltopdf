//nolint:testpackage // tests exercise display-list geometry and resolved styles
package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
)

func TestPositionedBlockPseudoAfterIsPaintedWithPseudoStyle(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div class="card">card</div></body></html>`, sheet(t, `
body { margin: 0; font-size: 10pt; }
.card { position: relative; width: 120pt; height: 40pt; }
.card::after { content: "→"; position: absolute; left: 80pt; top: 8pt; font-size: 16pt; font-weight: 700; color: red; }
`))

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.TrimSpace(op.Text) == "→" {
			if !op.Bold || op.Size < 15 || op.X < 79 || op.Y < 20 {
				t.Fatalf("pseudo style/position not applied: %+v", op)
			}

			return
		}
	}

	t.Fatal("positioned block pseudo content was not painted")
}

func TestInlineBlockWhitespaceKeepsGap(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div class="row"><span class="tag">A</span> `+
		`<span class="tag">B</span></div></body></html>`, sheet(t, `
body { margin: 0; font-size: 10pt; }
.tag { display: inline-block; padding: 2pt 4pt; border: 1pt solid #123456; }
`))

	var labels []Op

	for _, op := range res.Ops {
		if op.Kind == OpText && (strings.TrimSpace(op.Text) == "A" || strings.TrimSpace(op.Text) == "B") {
			labels = append(labels, op)
		}
	}

	if len(labels) != 2 {
		t.Fatalf("tag labels = %d, want two: %+v", len(labels), res.Ops)
	}

	if gap := labels[1].X - (labels[0].X + labels[0].W); gap <= 0.1 {
		t.Fatalf("inline-block whitespace collapsed: first=%+v second=%+v gap=%.2f", labels[0], labels[1], gap)
	}
}

func TestUnitlessLineHeightRecomputesForChildFontSize(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div class="child">text</div></body></html>`, sheet(t, `
body { margin: 0; font-size: 10pt; line-height: 1.5; }
.child { font-size: 20pt; }
`))

	// The layout result intentionally does not expose styles; the line box is
	// observable through the text op height and should be 1.5em, not the
	// parent's 15pt absolute line height.
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.TrimSpace(op.Text) == "text" {
			if op.H < 29 || op.H > 31 {
				t.Fatalf("unitless line-height inherited as absolute value: op=%+v", op)
			}

			return
		}
	}

	t.Fatal("missing child text operation")
}

func TestInlineChromeDoesNotOverflowListItem(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><section class="panel"><ul><li><code>AddObject</code> `+
		`copies public settings before conversion.</li></ul></section></body></html>`)
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
.panel { width: 220pt; padding: 10pt; border: 1pt solid #ccc; }
.panel ul { margin: 2pt 0 0; padding-left: 14pt; }
.panel li, .panel code { font-size: 8.5pt; }
.panel code { padding: 1pt 2pt; border: 1pt solid #ccc; }
`)
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 400, Height: 400, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})

	if err != nil {
		t.Fatal(err)
	}

	li := findElementByName(root, "li")
	boxNode := fixture56BoxByNode(res.root, li)

	if boxNode == nil {
		t.Fatal("missing list-item box")
	}

	for _, op := range res.Ops[boxNode.opStart : boxNode.opEnd+1] {
		if op.Kind == OpText && op.X+op.W > boxNode.x+boxNode.w+0.01 {
			t.Fatalf("inline chrome overflowed list item: op=%+v box=%+v", op, boxNode)
		}
	}
}

func TestRegularTextAndOnePixelBorderKeepTheirPaintMetrics(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div class="box">regular</div></body></html>`, sheet(t, `
body { margin: 0; font-family: Arial, sans-serif; font-size: 10pt; }
.box { font-weight: 400; border: 1px solid #123456; padding: 2pt; }
`))

	borderOps := 0

	for _, paintOp := range res.Ops {
		switch paintOp.Kind {
		case OpText:
			if strings.TrimSpace(paintOp.Text) == "regular" && paintOp.Bold {
				t.Fatalf("regular text selected bold paint: %+v", paintOp)
			}
		case OpLine:
			borderOps++

			if paintOp.Width < 0.74 || paintOp.Width > 0.76 {
				t.Fatalf("one CSS pixel border lost its 0.75pt metric: %+v", paintOp)
			}
		case OpFillRect, OpStrokeRect, OpImage, OpLinkURI, OpBullet, opKindNoop:
		}
	}

	if borderOps == 0 {
		t.Fatal("missing border paint operations")
	}
}
