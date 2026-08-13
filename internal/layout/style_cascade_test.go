//nolint:testpackage // tests exercise the resolved style cascade
package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
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
