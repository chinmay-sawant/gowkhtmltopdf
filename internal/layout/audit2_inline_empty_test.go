package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// audit2LayoutImages renders src with a 1x1 PNG resolver so percentage-width
// <img> fixtures get real used sizes.
func audit2LayoutImages(t *testing.T, src string, sheets ...*css.Stylesheet) *Result {
	t.Helper()

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	opts := Options{
		Width: 500, Height: 800, Background: true, Sheets: sheets,
		Images: func(string) ([]byte, error) { return onePixelPNG(), nil },
	}

	res, err := Layout(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	return res
}

// learn-cpp-org-11: an inline formatting context whose only content is
// collapsible whitespace must produce a zero-height line box. CSS 2.1 9.4.2
// treats line boxes with no text, no preserved white space, and no inline
// boxes with non-zero margin/padding/border as zero-height. gowk emitted a
// full line box for <span> </span>, pushing the next block down by one
// line-height (the live page's GTM noscript moved the whole document 13pt).
func TestWhitespaceOnlyInlineCreatesNoLineBox(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0; font-family: sans-serif; font-size: 16px; line-height: 1.5 }
h1 { font-size: 32px; margin: 0 }`)

	control := audit2LayoutImages(t, `<html><body><h1>Welcome</h1></body></html>`, cssSheet)
	if control == nil {
		t.Fatal("no result")
	}

	controlH1 := findBox(t, control, "h1")

	for _, variant := range []string{
		`<span></span>`,
		`<noscript></noscript>`,
		`<wbr>`,
		`<span> </span>`,
		`<span>` + "\n" + ` </span>`,
		`<a href="#"></a>`,
		`<em></em>`,
	} {
		res := layoutHTML(t, `<html><body>`+variant+`<h1>Welcome</h1></body></html>`, cssSheet)
		h1 := findBox(t, res, "h1")

		if math.Abs(h1.y-controlH1.y) > 0.01 {
			t.Errorf("variant %q: h1.y = %.2fpt, control %.2fpt; a whitespace-only "+
				"inline run must not open a line box", variant, h1.y, controlH1.y)
		}
	}
}

// learn-cpp-org-12: the newline between <a> and a width:100% image is a
// collapsible space, but it must not occupy the line that the image wraps to
// by itself. Deleting only the newline moved the hero up 13pt in the live
// page; the image top must match the whitespace-free variant.
func TestWhitespaceBeforePercentImageDoesNotAddLine(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0 } .wrap { width: 930px }`)

	withWS := audit2LayoutImages(t,
		`<html><body><div class="wrap"><a href="#">`+"\n"+`<img src="hero.png" style="width:100%"></a></div></body></html>`,
		cssSheet)
	withoutWS := audit2LayoutImages(t,
		`<html><body><div class="wrap"><a href="#"><img src="hero.png" style="width:100%"></a></div></body></html>`,
		cssSheet)

	imgWith := firstImageOp(t, withWS)
	imgWithout := firstImageOp(t, withoutWS)

	if math.Abs(imgWith.Y-imgWithout.Y) > 0.01 {
		t.Errorf("image top with whitespace = %.2fpt, without = %.2fpt; the "+
			"collapsible space before a replaced element must not create its own line",
			imgWith.Y, imgWithout.Y)
	}

	if imgWith.Y > 0.01 {
		t.Errorf("image top = %.2fpt, want 0 (leading collapsible space)", imgWith.Y)
	}
}

func firstImageOp(t *testing.T, res *Result) Op {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpImage {
			return op
		}
	}

	t.Fatal("no image op")

	return Op{}
}
