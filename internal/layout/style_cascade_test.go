//nolint:testpackage // tests exercise the resolved style cascade
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestCascadeShorthandRespectsSourceOrder(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><h1>heading</h1><p>paragraph</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	sheetAfter := sheet(t, `h1, p { margin-top: 0 } h1 { margin: 25mm 0 7mm }`)
	styles := resolveStyles(root, []*css.Stylesheet{sheetAfter}, "print", testViewport, 800)
	heading := findElementByName(root, "h1")

	if got := styles[heading].MarginTop; got < 70 || got > 72 {
		t.Fatalf("later margin shorthand lost to earlier longhand: got %.2fpt", got)
	}

	root, err = html.Parse(`<html><body><h1>heading</h1></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	sheetBefore := sheet(t, `h1 { margin: 25mm 0 7mm } h1 { margin-top: 0 }`)
	styles = resolveStyles(root, []*css.Stylesheet{sheetBefore}, "print", testViewport, 800)
	heading = findElementByName(root, "h1")

	if got := styles[heading].MarginTop; got != 0 {
		t.Fatalf("later margin-top lost to earlier shorthand: got %.2fpt", got)
	}
}

// TestCascadeBorderShorthandOverridesEarlierBorderTop: a later border
// shorthand must beat an earlier border-top longhand (fixture-56 domain-03
// accent top vs .domains > section frame).
func TestCascadeBorderShorthandOverridesEarlierBorderTop(t *testing.T) {
	t.Parallel()

	// Equal specificity (type+class): later border shorthand must win.
	root, err := html.Parse(`<html><body><div class="domains"><section class="d03">x</section></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	cssSheet := sheet(t, `
section.d03 { border-top: 3px solid #2563eb }
.domains > section { border: 1px solid #cbd5e1; border-left: 4px solid #0f766e }
`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)
	sec := findElementByName(root, "section")
	sty := styles[sec]

	if sty.BorderTop.Width > 1.5 {
		t.Fatalf("border-top width = %.2f, want ≤1.5 after later border shorthand", sty.BorderTop.Width)
	}

	// #2563eb accent must not survive; #cbd5e1 frame is ~ (0.80, 0.84, 0.88).
	if sty.BorderTop.Color[0] < 0.5 && sty.BorderTop.Color[2] > 0.85 {
		t.Fatalf("border-top still accent blue (%.2f,%.2f,%.2f), want neutral frame",
			sty.BorderTop.Color[0], sty.BorderTop.Color[1], sty.BorderTop.Color[2])
	}

	if sty.BorderLeft.Width < 2.5 {
		t.Fatalf("border-left width = %.2f, want accent rail ≥2.5", sty.BorderLeft.Width)
	}
}

func TestWritingModeInherits(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><div style="writing-mode: vertical-rl"><p>child text</p></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)
	pElem := findElementByName(root, "p")

	if styles[pElem].WritingMode != "vertical-rl" {
		t.Fatalf("p WritingMode = %q, want %q", styles[pElem].WritingMode, "vertical-rl")
	}

	doc, err := html.Parse(`<html><body><div style="writing-mode: vertical-rl"><p>vertical</p></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(doc, Options{Width: 800, Height: 600}) //nolint:exhaustruct // test options
	if err != nil {
		t.Fatal(err)
	}

	var textOp *Op

	for i := range res.Ops {
		if res.Ops[i].Kind == OpText && res.Ops[i].Text == "vertical" {
			textOp = &res.Ops[i]

			break
		}
	}

	if textOp == nil {
		t.Fatal("no OpText found for 'vertical'")

		return
	}

	if textOp.RotateDeg != -90 {
		t.Fatalf("textOp.RotateDeg = %v, want -90", textOp.RotateDeg)
	}
}

//nolint:cyclop // test assertion helper
func TestTextIndentInheritsAndShiftsFirstLine(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><div style="text-indent: 2em"><p><span>indented</span></p></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)
	pNode := findElementByName(root, "p")
	spanNode := findElementByName(root, "span")

	if styles[pNode].TextIndent <= 0 {
		t.Fatalf("p TextIndent = %v, want > 0", styles[pNode].TextIndent)
	}

	if styles[spanNode].TextIndent != styles[pNode].TextIndent {
		t.Fatalf("span TextIndent = %v, want %v (inherited)", styles[spanNode].TextIndent, styles[pNode].TextIndent)
	}

	docPlain, err := html.Parse(`<html><body><p style="margin:0;font-size:16px">hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	resPlain, err := Layout(docPlain, Options{Width: 800, Height: 600}) //nolint:exhaustruct
	if err != nil {
		t.Fatal(err)
	}

	docIndent, err := html.Parse(`<html><body><p style="margin:0;font-size:16px;text-indent:2em">hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	resIndent, err := Layout(docIndent, Options{Width: 800, Height: 600}) //nolint:exhaustruct
	if err != nil {
		t.Fatal(err)
	}

	var xPlain, xIndent float64

	for _, op := range resPlain.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "hello") {
			xPlain = op.X

			break
		}
	}

	for _, op := range resIndent.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "hello") {
			xIndent = op.X

			break
		}
	}

	if xIndent <= xPlain {
		t.Fatalf("xIndent (%v) must be greater than xPlain (%v)", xIndent, xPlain)
	}
}

//nolint:gocognit,cyclop,wsl,funlen // table-driven webkit aliases assertion
func TestWebkitPrefixAliases(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body>
		<div class="box-sizing" style="-webkit-box-sizing: border-box">x</div>
		<div class="radius" style="-webkit-border-radius: 12pt">x</div>
		<div class="flex" style="-webkit-flex-direction: column; -webkit-flex-wrap: wrap; ` +
		`-webkit-justify-content: center; -webkit-align-items: center; ` +
		`-webkit-align-content: space-between; -webkit-order: 3">x</div>
		<div class="transform" style="-webkit-transform: rotate(45deg)">x</div>
		<div class="shadow" style="-webkit-box-shadow: 2pt 2pt 4pt #000">x</div>
	</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)

	body := findElementByName(root, "body")
	if body == nil {
		t.Fatal("body element not found")
	}

	for _, child := range body.Children {
		if child.Type != html.ElementNode {
			continue
		}

		sty := styles[child]
		switch child.Attribute("class") {
		case "box-sizing":
			if sty.BoxSizing != "border-box" {
				t.Fatalf("-webkit-box-sizing: got %q, want border-box", sty.BoxSizing)
			}
		case "radius":
			if !near(sty.BorderRadius, 12) {
				t.Fatalf("-webkit-border-radius: got %.1f, want 12", sty.BorderRadius)
			}
		case "flex":
			if sty.FlexDirection != "column" {
				t.Fatalf("-webkit-flex-direction: got %q, want column", sty.FlexDirection)
			}
			if sty.FlexWrap != "wrap" {
				t.Fatalf("-webkit-flex-wrap: got %q, want wrap", sty.FlexWrap)
			}
			if sty.JustifyContent != fxCenter {
				t.Fatalf("-webkit-justify-content: got %q, want center", sty.JustifyContent)
			}
			if sty.AlignItems != fxCenter {
				t.Fatalf("-webkit-align-items: got %q, want center", sty.AlignItems)
			}
			if sty.AlignContent != "space-between" {
				t.Fatalf("-webkit-align-content: got %q, want space-between", sty.AlignContent)
			}
			if sty.FlexOrder != 3 {
				t.Fatalf("-webkit-order: got %d, want 3", sty.FlexOrder)
			}
		case "transform":
			if !sty.HasTransform {
				t.Fatal("-webkit-transform: HasTransform false, want true")
			}
		case "shadow":
			if !sty.BoxShadowSet {
				t.Fatal("-webkit-box-shadow: BoxShadowSet false, want true")
			}
		}
	}
}

//nolint:cyclop // SVG presentation test assertions
func TestSVGPresentationProps(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body>
		<svg style="fill: red; stroke: blue; stroke-width: 2pt; fill-opacity: 0.8; stroke-opacity: 0.5"></svg>
	</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)
	svgElem := findElementByName(root, "svg")

	if svgElem == nil {
		t.Fatal("svg element not found")
	}

	sty := styles[svgElem]
	if !sty.FillSet || sty.Fill != [3]float64{1, 0, 0} {
		t.Fatalf("Fill = %v, FillSet = %v, want red (1,0,0) and true", sty.Fill, sty.FillSet)
	}

	if !sty.StrokeSet || sty.Stroke != [3]float64{0, 0, 1} {
		t.Fatalf("Stroke = %v, StrokeSet = %v, want blue (0,0,1) and true", sty.Stroke, sty.StrokeSet)
	}

	if !sty.StrokeWidthSet || !near(sty.StrokeWidth, 2) {
		t.Fatalf("StrokeWidth = %.1f, StrokeWidthSet = %v, want 2 and true", sty.StrokeWidth, sty.StrokeWidthSet)
	}

	if !near(sty.FillOpacity, 0.8) {
		t.Fatalf("FillOpacity = %.2f, want 0.8", sty.FillOpacity)
	}

	if !near(sty.StrokeOpacity, 0.5) {
		t.Fatalf("StrokeOpacity = %.2f, want 0.5", sty.StrokeOpacity)
	}
}

func findElementByName(root *html.Node, name string) *html.Node {
	if root.Type == html.ElementNode && root.Name == name {
		return root
	}

	for _, child := range root.Children {
		if found := findElementByName(child, name); found != nil {
			return found
		}
	}

	return nil
}
