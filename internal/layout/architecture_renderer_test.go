//nolint:testpackage // tests exercise display-list geometry and resolved styles
package layout

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
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

func TestFlexLegendKeepsSpecifiedWidthSwatchOnTheLabelLine(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<div class="legend">
<span><i class="swatch blue"></i>internal stage</span>
<span><i class="swatch orange"></i>caller decision</span>
<span><i class="swatch dark"></i>published contract</span>
</div>
</body></html>`, sheet(t, `
* { box-sizing: border-box; }
body { margin: 0; font-family: Arial, Helvetica, sans-serif; font-size: 10pt; }
.legend { display: flex; gap: 5mm; }
.legend span { color: #536672; font-size: 8pt; }
.swatch { display: inline-block; width: 4mm; height: 4mm; margin-right: 1mm; }
.swatch.blue { background: #3c7899; }
.swatch.orange { background: #e07848; }
.swatch.dark { background: #163b56; }
`))

	byText := map[string]Op{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			byText[strings.TrimSpace(op.Text)] = op
		}
	}

	for _, pair := range [][2]string{
		{"internal", "stage"},
		{"caller", "decision"},
		{"published", "contract"},
	} {
		if joined, ok := byText[pair[0]+" "+pair[1]]; ok && joined.W > 0 {
			continue
		}

		first, okFirst := byText[pair[0]]
		second, okSecond := byText[pair[1]]

		if !okFirst || !okSecond {
			t.Fatalf("missing %q / %q in %#v", pair[0], pair[1], byText)
		}

		if math.Abs(first.Y-second.Y) > 0.01 {
			t.Fatalf("%q wrapped under %q: first=%+v second=%+v", pair[1], pair[0], first, second)
		}

		if second.X <= first.X {
			t.Fatalf("%q did not continue after %q: first=%+v second=%+v", pair[1], pair[0], first, second)
		}
	}
}

func isInlineBlockSwatch(op Op) bool {
	return op.Kind == OpFillRect && op.W > 5 && op.W < 20 && op.H > 5 && op.H < 20
}

func TestInlineBlockHonorsLengthVerticalAlign(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<div class="legend"><span><i class="swatch"></i>internal stage</span></div>
</body></html>`, sheet(t, `
body { margin: 0; font-family: Arial, Helvetica, sans-serif; font-size: 10pt; }
.legend { display: flex; }
.legend span { font-size: 8pt; }
.swatch {
	display: inline-block; width: 4mm; height: 4mm; margin-right: 1mm;
	vertical-align: -1mm; background: #3c7899;
}
`))

	var swatch, label Op

	for _, op := range res.Ops {
		switch {
		case isInlineBlockSwatch(op):
			swatch = op
		case op.Kind == OpText && strings.Contains(op.Text, "internal"):
			label = op
		}
	}

	if swatch.H == 0 || label.W == 0 {
		t.Fatalf("missing swatch or label: swatch=%+v label=%+v", swatch, label)
	}

	// vertical-align:-1mm lowers the empty 4mm box from the text baseline
	// so the swatch sits optically with the 8pt label.
	const mm = 72.0 / 25.4

	got := (swatch.Y + swatch.H) - label.Y
	if got < 0.75*mm || got > 1.25*mm {
		t.Fatalf("swatch bottom-to-baseline = %.2fpt, want ~1mm (%.2fpt): swatch=%+v label=%+v",
			got, mm, swatch, label)
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

		return
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

func TestMixedRoundedBorderKeepsRoundedPaintGeometry(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div class="card">card</div></body></html>`, sheet(t, `
body { margin: 0; }
.card {
 width: 120pt;
 height: 40pt;
 border: 1pt solid #cbd5e1;
 border-left: 4pt solid #2563eb;
 border-radius: 6pt;
}
`))

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpStrokeRect && paintOp.Radius > 0 {
			return
		}
	}

	t.Fatal("mixed rounded border fell back to square edge lines")
}

// EXT-08: left-rail paint width follows CSS border width for any accent color,
// not only fixture-56's #2563eb.
func TestMixedRoundedLeftRailWidthIgnoresAccentHex(t *testing.T) {
	t.Parallel()

	leftWidth := func(color string) float64 {
		t.Helper()

		res := layoutHTML(t, `<html><body><div class="card">card</div></body></html>`, sheet(t, `
body { margin: 0; }
.card {
 width: 120pt;
 height: 40pt;
 border: 1pt solid #cbd5e1;
 border-left: 4pt solid `+color+`;
 border-radius: 6pt;
}
`))
		for _, paintOp := range res.Ops {
			if paintOp.Kind == OpStrokeRect && paintOp.StrokeMask == StrokeMaskLeft && paintOp.Width > 2 {
				return paintOp.Width
			}
		}

		t.Fatalf("missing left StrokeMask rail for accent %s", color)

		return 0
	}

	blue := leftWidth("#2563eb")
	red := leftWidth("#dc2626")

	if math.Abs(blue-red) > 1e-9 {
		t.Fatalf("left rail width gated by accent hex: #2563eb=%.4f #dc2626=%.4f", blue, red)
	}

	if blue < 3.9 || blue > 4.1 {
		t.Fatalf("left rail width=%.4f, want CSS 4pt (no color scale)", blue)
	}
}

func TestMixedRoundedBorderKeepsTopAccentOnTopEdge(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div class="card">card</div></body></html>`, sheet(t, `
body { margin: 0; }
.card {
 width: 120pt;
 height: 40pt;
 border: 1pt solid #cbd5e1;
 border-top: 4pt solid #b4532a;
 border-radius: 6pt;
}
`))

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpStrokeRect && paintOp.StrokeMask == StrokeMaskTop &&
			paintOp.Radius > 0 && paintOp.R > 0.5 && paintOp.G < 0.5 && paintOp.B < 0.3 && paintOp.Width > 2 {
			return
		}
	}

	t.Fatal("mixed rounded border lost the accented top StrokeMask edge")
}

//nolint:cyclop,wsl // regression keeps the complete page-fragment assertion together
func TestRoundedCalloutMovesTogetherAtPageBoundary(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="lead">lead</div>`+
		`<aside class="dom-notes note">Security: no script execution by construction</aside></body></html>`)
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
.lead { height: 80pt; }
.note {
 display: block; height: 30pt; padding: 4pt; background: #f2ecdf;
 border-left: 4pt solid #d97706; border-radius: 0 8pt 8pt 0;
}
`)
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 220, Height: 100, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{ //nolint:exhaustruct
		PageWidth: 220, PageHeight: 100,
	}); err != nil {
		t.Fatal(err)
	}

	note := findElementByClass(root, "note")
	noteBox := fixture56BoxByNode(res.root, note)
	if noteBox == nil {
		t.Fatal("rounded callout has no box")

		return
	}
	pageTop := math.Floor(noteBox.y/100+0.5) * 100
	if math.Abs(noteBox.y-pageTop) > 0.01 {
		t.Fatalf("rounded callout straddles page: box=%+v", noteBox)
	}

	railY := math.MaxFloat64
	for i := noteBox.opStart; i <= noteBox.opEnd && i < len(res.Ops); i++ {
		op := res.Ops[i]
		if (op.Kind == OpLine || (op.Kind == OpStrokeRect && op.StrokeMask&StrokeMaskLeft != 0)) &&
			op.H > 0 && op.R > 0.7 && op.G > 0.3 && op.B < 0.1 {
			railY = math.Min(railY, op.Y)
		}
	}
	if math.Abs(railY-noteBox.y) > 0.01 {
		t.Fatalf("rounded callout rail detached at page boundary: boxY=%.2f railY=%.2f", noteBox.y, railY)
	}
}

func TestAbsoluteLeftRightBoxUsesBothInsets(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="page"><div class="footer">right footer</div></div></body></html>`)
	cssSheet := sheet(t, `
body { margin: 0; }
.page { position: relative; width: 200pt; height: 100pt; }
.footer { position: absolute; left: 10pt; right: 10pt; bottom: 0; height: 10pt; }
`)
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 300, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})

	if err != nil {
		t.Fatal(err)
	}

	page := findElementByClass(root, "page")
	footer := findElementByClass(root, "footer")
	pageBox := fixture56BoxByNode(res.root, page)
	footerBox := fixture56BoxByNode(res.root, footer)

	if pageBox == nil || footerBox == nil {
		t.Fatalf("missing positioned boxes: page=%+v footer=%+v", pageBox, footerBox)
	}

	if footerBox.x+footerBox.w > pageBox.x+pageBox.w+0.01 {
		t.Fatalf("left/right absolute box clipped right edge: page=%+v footer=%+v", pageBox, footerBox)
	}
}

func TestAfterPseudoStaticPositionFollowsBlockHost(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="node">1 · Configure</div></body></html>`)
	cssSheet := sheet(t, `
body { margin: 0; }
.node { position: relative; width: 100pt; height: 30pt; }
.node::after { content: "→"; position: absolute; margin-top: 10pt; font-size: 16pt; font-weight: 700; }
`)
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 300, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})

	if err != nil {
		t.Fatal(err)
	}

	node := findElementByClass(root, "node")
	nodeBox := fixture56BoxByNode(res.root, node)

	if nodeBox == nil {
		t.Fatal("missing node box")
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && strings.TrimSpace(paintOp.Text) == "→" {
			if paintOp.Y <= nodeBox.y+nodeBox.height {
				t.Fatalf("after pseudo stayed inside block host: op=%+v node=%+v", paintOp, nodeBox)
			}

			return
		}
	}

	t.Fatal("missing generated arrow")
}

//nolint:cyclop,funlen,wsl // regression keeps the complete fixture flow assertion together
func TestAPIFixtureFlowMetricsDoNotOverlapPreviousFlexItems(t *testing.T) {
	t.Parallel()

	base := filepath.Join("..", "..", "testdata", "golden", "api")
	source, err := os.ReadFile(filepath.Join(base, "architecture-diagram.html"))
	if err != nil {
		t.Fatal(err)
	}

	root := mustParse(t, string(source))
	var cssText strings.Builder
	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode || node.Name != styleElement {
			return
		}

		for _, child := range node.Children {
			if child.Type == html.TextNode {
				cssText.WriteString(child.Text)
			}
		}
	})

	cssSheet := sheet(t, cssText.String())
	res, err := Layout(root, Options{ //nolint:exhaustruct // focused fixture regression
		Width: 595.28, Height: 841.89, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{ //nolint:exhaustruct // exercise pagination chrome repair
		PageWidth: 595.28, PageHeight: 841.89,
	}); err != nil {
		t.Fatal(err)
	}

	flow := findElementByClass(root, "flow")
	meta := findElementByClass(root, "meta-row")
	flowBox := fixture56BoxByNode(res.root, flow)
	metaBox := fixture56BoxByNode(res.root, meta)
	if flowBox == nil || metaBox == nil {
		t.Fatalf("missing flow boxes: flow=%+v meta=%+v", flowBox, metaBox)
	}

	const minClearGap = 11 * 2.834645669 // 11mm in CSS points
	if gap := metaBox.y - (flowBox.y + flowBox.height); gap < minClearGap-0.01 {
		t.Fatalf("metrics gap = %.2fpt, want at least %.2fpt: flow=%+v meta=%+v", gap, minClearGap, flowBox, metaBox)
	}

	var flowNodeHeight float64
	for nodeIndex, node := range fixture56Nodes(root, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Attribute("class") == "node"
	}) {
		boxNode := fixture56BoxByNode(res.root, node)
		if boxNode == nil {
			t.Fatalf("missing flow node box")
		}

		if nodeIndex == 0 {
			flowNodeHeight = boxNode.height
		} else if math.Abs(boxNode.height-flowNodeHeight) > 0.01 {
			t.Fatalf("flex item height changed after paint: first=%.2f current=%.2f", flowNodeHeight, boxNode.height)
		}

		if metaBox.y < boxNode.y+boxNode.height-0.01 {
			t.Fatalf("metrics overlap flow node: node=%+v meta=%+v", boxNode, metaBox)
		}
	}
}

func TestAbsoluteLeftOnlyShrinkToFit(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="page"><div class="pill">pos</div></div></body></html>`)
	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.page { position: relative; width: 200pt; height: 60pt; border: 1pt dashed #888; }
.pill { position: absolute; left: 4pt; background: #fd8; padding: 2pt 6pt; }
`)
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 300, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := findElementByClass(root, "page")
	pill := findElementByClass(root, "pill")
	pageBox := fixture56BoxByNode(res.root, page)
	pillBox := fixture56BoxByNode(res.root, pill)
	if pageBox == nil || pillBox == nil {
		t.Fatalf("missing boxes: page=%+v pill=%+v", pageBox, pillBox)
	}

	if pillBox.w > pageBox.w*0.5 {
		t.Fatalf("left-only abspos should shrink-to-fit, pill.w=%.1f page.w=%.1f", pillBox.w, pageBox.w)
	}
	if pillBox.x < pageBox.x+3 || pillBox.x > pageBox.x+6 {
		t.Fatalf("pill.x=%.1f, want ~page.x+4 (page.x=%.1f)", pillBox.x, pageBox.x)
	}
}

func findElementByClass(root *html.Node, className string) *html.Node {
	var found *html.Node

	root.Walk(func(node *html.Node) {
		if found != nil || node.Type != html.ElementNode {
			return
		}

		classes := " " + node.Attribute("class") + " "
		if strings.Contains(classes, " "+className+" ") {
			found = node
		}
	})

	return found
}
