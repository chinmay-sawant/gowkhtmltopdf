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

// TestCascadeClampFallbackKeepsEarlierWidth: while clampLength is gated off,
// clamp() must stay out of the cascade so width:100% survives (fixture-56
// .d01-exit). Wave2 left clamp allowed in supportedDeclaration but made
// clampLength always fail, which discarded the fallback and shrink-wrapped
// the exit-code table.
func TestCascadeClampFallbackKeepsEarlierWidth(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><table class="d01-exit"><tr><td>x</td></tr></table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	cssSheet := sheet(t, `
table.d01-exit {
  width: 100%;
  width: clamp(24rem, 70%, 46rem);
}
`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)
	tbl := findElementByName(root, "table")
	sty := styles[tbl]

	if sty.WidthPercent < 99.5 || sty.WidthPercent > 100.5 {
		t.Fatalf("WidthPercent = %.2f, want 100 (clamp must not win cascade)", sty.WidthPercent)
	}

	if sty.Width > 0 {
		t.Fatalf("Width = %.2fpt set from clamp path, want percent-only fallback", sty.Width)
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

//nolint:gocognit,cyclop,funlen,gocyclo,lll,wsl // table-driven webkit aliases assertion
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
		<div class="box-align-start" style="-webkit-box-align: start">x</div>
		<div class="box-align-baseline" style="-webkit-box-align: baseline">x</div>
		<div class="box-flex" style="-webkit-box-flex: 2">x</div>
		<div class="box-ordinal" style="-webkit-box-ordinal-group: 3">x</div>
		<div class="box-orient-h" style="-webkit-box-orient: horizontal">x</div>
		<div class="box-orient-v" style="-webkit-box-orient: vertical">x</div>
		<div class="box-pack" style="-webkit-box-pack: justify">x</div>
		<div class="text-fill" style="-webkit-text-fill-color: #ff0000">x</div>
		<div class="webkit-box" style="display: -webkit-box">x</div>
		<div class="webkit-inline-box" style="display: -webkit-inline-box">x</div>
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
		case "box-align-start":
			if sty.AlignItems != flexStartKeyword {
				t.Fatalf("-webkit-box-align: start got %q, want %q", sty.AlignItems, flexStartKeyword)
			}
		case "box-align-baseline":
			if sty.AlignItems != "baseline" {
				t.Fatalf("-webkit-box-align: baseline got %q, want baseline", sty.AlignItems)
			}
		case "box-flex":
			if sty.FlexGrow != 2 {
				t.Fatalf("-webkit-box-flex: got %.1f, want 2", sty.FlexGrow)
			}
		case "box-ordinal":
			if sty.FlexOrder != 2 {
				t.Fatalf("-webkit-box-ordinal-group: 3 got %d, want 2 (order = group-1)", sty.FlexOrder)
			}
		case "box-orient-h":
			if sty.FlexDirection != fxRow {
				t.Fatalf("-webkit-box-orient: horizontal got %q, want %q", sty.FlexDirection, fxRow)
			}
		case "box-orient-v":
			if sty.FlexDirection != fxCol {
				t.Fatalf("-webkit-box-orient: vertical got %q, want %q", sty.FlexDirection, fxCol)
			}
		case "box-pack":
			if sty.JustifyContent != fxBetween {
				t.Fatalf("-webkit-box-pack: justify got %q, want %q", sty.JustifyContent, fxBetween)
			}
		case "text-fill":
			if sty.Color != [3]float64{1, 0, 0} {
				t.Fatalf("-webkit-text-fill-color: got %v, want red", sty.Color)
			}
		case "webkit-box":
			if sty.Display != displayFlex {
				t.Fatalf("display: -webkit-box got %q, want %q", sty.Display, displayFlex)
			}
			if !sty.IsWebkitBox {
				t.Fatal("display: -webkit-box IsWebkitBox=false, want true")
			}
		case "webkit-inline-box":
			if sty.Display != displayInlineFlex {
				t.Fatalf("display: -webkit-inline-box got %q, want %q", sty.Display, displayInlineFlex)
			}
		}
	}

	// Prefixed equals unprefixed equivalents for all 6 plus display.
	prefixed := map[string]string{
		"-webkit-box-align":         "center",
		"-webkit-box-flex":          "2.5",
		"-webkit-box-ordinal-group": "2",
		"-webkit-box-orient":        "vertical",
		"-webkit-box-pack":          "center",
		"-webkit-text-fill-color":   "blue",
	}
	unprefixed := map[string]string{
		"-webkit-box-align":         "align-items: center",
		"-webkit-box-flex":          "flex-grow: 2.5",
		"-webkit-box-ordinal-group": "order: 1",
		"-webkit-box-orient":        "flex-direction: column",
		"-webkit-box-pack":          "justify-content: center",
		"-webkit-text-fill-color":   "color: blue",
	}
	for prop, val := range prefixed {
		stylePrefixed := styleForDecl(t, prop+": "+val)
		decl := unprefixed[prop]
		parts := strings.SplitN(decl, ":", 2)
		styleCanon := styleForDecl(t, strings.TrimSpace(parts[0])+": "+strings.TrimSpace(parts[1]))
		if !stylesEqualForProp(prop, stylePrefixed, styleCanon) {
			t.Fatalf("%s: %q style not equal to canonical %q", prop, val, decl)
		}
	}

	// Legacy -webkit-box maps to flex so box-* longhands can lay out.
	if s1, s2 := styleForDecl(t, "display: -webkit-box"), styleForDecl(t, "display: flex"); s1.Display != s2.Display {
		t.Fatalf("display -webkit-box %q != flex %q", s1.Display, s2.Display)
	}
	if s1, s2 := styleForDecl(t, "display: -webkit-inline-box"), styleForDecl(t, "display: inline-flex"); s1.Display != s2.Display {
		t.Fatalf("display -webkit-inline-box %q != inline-flex %q", s1.Display, s2.Display)
	}
}

//nolint:wsl // test helper
func styleForDecl(t *testing.T, decl string) *ResolvedStyle {
	t.Helper()
	root, err := html.Parse(`<html><body><div style="` + decl + `">x</div></body></html>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	styles := resolveStyles(root, nil, "print", testViewport, 800)

	body := findElementByName(root, "body")
	for _, child := range body.Children {
		if child.Type == html.ElementNode {
			return styles[child]
		}
	}

	t.Fatal("div not found")

	return nil
}

func stylesEqualForProp(prop string, first, second *ResolvedStyle) bool {
	switch prop {
	case "-webkit-box-align":
		return first.AlignItems == second.AlignItems
	case "-webkit-box-flex":
		return first.FlexGrow == second.FlexGrow
	case "-webkit-box-ordinal-group":
		return first.FlexOrder == second.FlexOrder
	case "-webkit-box-orient":
		return first.FlexDirection == second.FlexDirection
	case "-webkit-box-pack":
		return first.JustifyContent == second.JustifyContent
	case "-webkit-text-fill-color":
		return first.Color == second.Color
	default:
		return false
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
