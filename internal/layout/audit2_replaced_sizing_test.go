package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// audit2 replaced-content sizing regressions. Each test asserts the painted
// geometry of the image op (X/Y/W/H), not implementation details.

// oneImageOp returns the only image op in the result.
func oneImageOp(t *testing.T, res *Result) Op {
	t.Helper()

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("image ops = %d, want exactly 1", len(imgs))
	}

	return imgs[0]
}

// tutorialspoint-cpp-11: a width:100% block image must resolve the percentage
// against its containing block (the card content column), not the page frame.
// The live banner is 920x250 inside .al-detail-card with padding 20px.
func TestBlockImagePercentWidthUsesContainingBlock(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.card { padding: 21pt 15pt; border: 0.75pt solid #e8ebf0 }
.wrap { max-width: 920px; margin: 0 auto }
.wrap img { width: 100%; height: auto; display: block }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="card"><p>column</p><div class="wrap">`+
			`<img src="x.png"></div></div></body></html>`,
		tinyPNG(920, 250), "", cssSheet)

	img := oneImageOp(t, res)
	// The wrapper is the image's containing block: its content box is the
	// percent base (the card border and padding sit outside it).
	wrap := findBoxByClass(t, res, "wrap")
	wantW := wrap.w
	wantH := wantW * 250.0 / 920.0

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("banner image = %.3fx%.3f at x=%.2f, want %.3fx%.3f "+
			"(percent width against the wrapper content box, ratio 920/250)",
			img.W, img.H, img.X, wantW, wantH)
	}

	if img.X+img.W > wrap.x+wrap.w+0.5 {
		t.Fatalf("banner right edge %.2f overflows the wrapper right edge %.2f",
			img.X+img.W, wrap.x+wrap.w)
	}
}

// programiz-cpp-12: the editor screenshot keeps its intrinsic 966/570 ratio
// inside a column flex card and honors the width attribute when no CSS width
// applies.
func TestFlexColumnImageWidthAttributeKeepsRatio(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.seg { display: flex; flex-direction: column }
img { max-width: 100% !important }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="seg"><img src="x.png" width="365"></div></body></html>`,
		tinyPNG(966, 570), "", cssSheet)

	img := oneImageOp(t, res)
	wantW := 365 * 0.75 // attribute is CSS px
	wantH := wantW * 570.0 / 966.0

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("width attribute image = %.3fx%.3f at x=%.2f, want %.3fx%.3f",
			img.W, img.H, img.X, wantW, wantH)
	}
}

// programiz-cpp-12 real shape: the site print CSS sets img{width:100%}, so the
// screenshot fills the flex column content box and still keeps the 966/570
// ratio (no overflow past the card).
func TestFlexColumnImagePercentWidthKeepsRatio(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.seg { display: flex; flex-direction: column; padding: 30pt }
img { width: 100%; max-width: 100% !important }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="seg"><img src="x.png"></div></body></html>`,
		tinyPNG(966, 570), "", cssSheet)

	img := oneImageOp(t, res)
	wantW := testViewport - 60 // flex container content box
	wantH := wantW * 570.0 / 966.0

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("percent image = %.3fx%.3f at x=%.2f, want %.3fx%.3f "+
			"(card content width, intrinsic ratio)",
			img.W, img.H, img.X, wantW, wantH)
	}
}

// programiz-cpp-13: img{page-break-inside:avoid} must move a whole image to the
// next page instead of painting a sliver at the boundary and dropping the rest.
func TestImageAvoidInsideMovesWholeToNextPage(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt }
div.filler { height: 700pt }
img { page-break-inside: avoid; display: block }
`)

	// 400x300 px = 300x225pt image that straddles the content bottom.
	res := layoutHTMLWithImages(t,
		`<html><body><div class="filler">filler</div><img src="x.png"></body></html>`,
		tinyPNG(400, 300), "", cssSheet)

	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	imgIdx := -1

	for i, op := range res.Ops {
		if op.Kind == OpImage {
			imgIdx = i

			break
		}
	}

	if imgIdx < 0 {
		t.Fatal("no image op emitted")
	}

	page := pageOfIdx(t, res, imgIdx)
	img := res.Ops[imgIdx]

	if page != 1 {
		t.Fatalf("image op on page %d at y=%.2f h=%.2f, want whole image on page 1 "+
			"(page-break-inside:avoid must move it off the boundary)", page, img.Y, img.H)
	}

	if img.Y < 841.5 {
		t.Fatalf("image op moved to page %d but y=%.2f is not the next page top; "+
			"the fixture never straddled the boundary", page, img.Y)
	}
}

// programiz-cpp-13 real shape: the illustration sits in a div that is a flex
// column item (the site's .code-segments + .col-sm-12 markup). The whole image
// must still move off the page boundary, not paint a top sliver and vanish.
func TestFlexItemImageAvoidInsideMovesWholeToNextPage(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt }
div.filler { height: 700pt }
.seg { display: flex; flex-direction: column }
.seg img { display: block; page-break-inside: avoid; object-fit: cover; width: 100%; height: 100% }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="filler">filler</div><div class="seg">`+
			`<div class="col"><img src="x.png"></div></div></body></html>`,
		tinyPNG(1152, 776), "", cssSheet)

	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	imgIdx := -1

	for i, op := range res.Ops {
		if op.Kind == OpImage {
			imgIdx = i

			break
		}
	}

	if imgIdx < 0 {
		t.Fatal("no image op emitted")
	}

	img := res.Ops[imgIdx]

	if page := pageOfIdx(t, res, imgIdx); page != 1 {
		t.Fatalf("nested flex image op on page %d at y=%.2f h=%.2f, want the whole "+
			"image moved to page 1", page, img.Y, img.H)
	}

	if img.Y < 841.5 {
		t.Fatalf("nested image op y=%.2f is not the next page top; the fixture never "+
			"straddled the boundary", img.Y)
	}
}

// w3schools-4: the cert-card hexagon is an inline svg with width/height 39
// attributes inside a 39x39 icon box whose CSS sizes the svg at 100%. It must
// paint 29.25pt square, not stretch across the card.
func TestInlineSVGPercentSizeStaysInIconBox(t *testing.T) {
	t.Parallel()

	const hexagon = `<svg xmlns="http://www.w3.org/2000/svg" width="39" height="39" ` +
		`fill="#00599C" viewBox="0 0 24 24"><path d="M12 2 2 7v10l10 5 10-5V7z"/></svg>`

	cssSheet := sheet(t, `
body { margin: 0 }
.header { display: flex; align-items: center }
.icon { width: 39px; height: 39px; flex-shrink: 0 }
.icon svg { width: 100%; height: 100% }
`)

	res := layoutHTML(t,
		`<html><body><div class="header"><div class="icon">`+hexagon+
			`</div><h2>Learn C++</h2></div></body></html>`, cssSheet)

	img := oneImageOp(t, res)
	want := 39 * 0.75

	if !near(img.W, want) || !near(img.H, want) {
		t.Fatalf("hexagon svg = %.3fx%.3f at x=%.2f, want %.3fx%.3f (icon box)",
			img.W, img.H, img.X, want, want)
	}
}

// w3schools-4 real shape: when the certificate preview's column is squeezed
// below max-width, the width follows the column and the height follows the
// 250:179 ratio.
func TestNarrowCertPreviewKeepsRatio(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.narrow { width: 78pt }
.narrow img { width: 100%; max-width: 250px; height: auto; display: block; margin: 0 0 0 auto }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="narrow">`+
			`<img src="x.png" width="250" height="179"></div></body></html>`,
		tinyPNG(250, 179), "", cssSheet)

	img := oneImageOp(t, res)
	wantW := 78.0
	wantH := wantW * 179.0 / 250.0

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("narrow cert preview = %.3fx%.3f, want %.3fx%.3f",
			img.W, img.H, wantW, wantH)
	}
}

// cplusplus-tutorial-8: a viewBox-only inline SVG takes its width from the
// containing block (SVG 2 width:auto is 100% of the CB) and derives the height
// from the viewBox ratio. Chrome: a 64x24px div yields a 64x19.2px svg.
func TestInlineSVGViewBoxUsesContainingBlockWidth(t *testing.T) {
	t.Parallel()

	const inlineSVG = `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 120 36'>` +
		`<text x='0' y='22' font-size='22px' font-family='sans-serif'>cplusplus</text></svg>`

	cssSheet := sheet(t, `
body { margin: 0 }
#site { width: 64px; height: 24px; display: inline-block; vertical-align: top }
`)

	res := layoutHTML(t,
		`<html><body><div id="site">`+inlineSVG+`</div></body></html>`, cssSheet)

	img := oneImageOp(t, res)
	wantW := 64 * 0.75            // 64px containing block
	wantH := wantW * 36.0 / 120.0 // viewBox ratio 120:36

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("viewBox-only inline svg = %.3fx%.3f, want %.3fx%.3f "+
			"(containing block width, viewBox ratio)", img.W, img.H, wantW, wantH)
	}
}

// cplusplus-tutorial-8 control: explicit width/height attributes stay
// authoritative over the containing block.
func TestInlineSVGAttributesWin(t *testing.T) {
	t.Parallel()

	const inlineSVG = `<svg xmlns='http://www.w3.org/2000/svg' width='64' height='24' viewBox='0 0 120 36'>` +
		`<text x='0' y='22' font-size='22px' font-family='sans-serif'>cplusplus</text></svg>`

	cssSheet := sheet(t, `body { margin: 0 } #site { width: 500pt }`)

	res := layoutHTML(t, `<html><body><div id="site">`+inlineSVG+`</div></body></html>`, cssSheet)

	img := oneImageOp(t, res)
	wantW := 64 * 0.75
	wantH := 24 * 0.75

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("svg with width/height attrs = %.3fx%.3f, want %.3fx%.3f",
			img.W, img.H, wantW, wantH)
	}
}

// w3schools-4: a square image with width:100%;height:auto in a flex row must
// paint square. Chrome: 400x400 in a 400pt row.
func TestFlexRowPercentImageStaysSquare(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.row { display: flex }
.row img { width: 100%; height: auto }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="row"><img src="x.png"></div></body></html>`,
		tinyPNG(96, 96), "", cssSheet)

	img := oneImageOp(t, res)
	if !near(img.W, img.H) {
		t.Fatalf("square flex image painted %.3fx%.3f, want square", img.W, img.H)
	}
}

// w3schools-4: the certificate preview (250x179) is width:100% capped at
// 250px with height:auto. The ratio must survive the cap and the flex column.
func TestFlexColumnCertPreviewKeepsRatio(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.col { display: flex; flex-direction: column; width: 250pt }
.col img { width: 100%; max-width: 250px; height: auto; display: block; margin: 0 0 0 auto }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="col">`+
			`<img src="x.png" width="250" height="179"></div></body></html>`,
		tinyPNG(250, 179), "", cssSheet)

	img := oneImageOp(t, res)
	wantW := 250 * 0.75 // max-width: 250px
	wantH := wantW * 179.0 / 250.0

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("cert preview = %.3fx%.3f, want %.3fx%.3f (max-width cap, ratio)",
			img.W, img.H, wantW, wantH)
	}
}

// w3schools-4: a 300x300 badge with width:auto inside a column flex container
// must stay square (Chrome scales both axes when it stretches).
func TestFlexColumnAutoImageStaysSquare(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.col { display: flex; flex-direction: column; width: 300pt }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="col">`+
			`<img src="x.png" style="width:auto;border-radius:5px"></div></body></html>`,
		tinyPNG(300, 300), "", cssSheet)

	img := oneImageOp(t, res)
	if !near(img.W, img.H) {
		t.Fatalf("300x300 badge painted %.3fx%.3f, want square", img.W, img.H)
	}
}

// w3schools-4: a plain image without dimensions in a flex item must keep its
// 995x786 ratio too.
func TestFlexItemPlainImageKeepsRatio(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.flex { display: flex; flex-direction: column; width: 300pt }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="flex"><img src="x.png"></div></body></html>`,
		tinyPNG(995, 786), "", cssSheet)

	img := oneImageOp(t, res)
	gotRatio := img.W / img.H
	wantRatio := 995.0 / 786.0

	if !near(gotRatio, wantRatio) {
		t.Fatalf("plain flex image ratio = %.4f (%.3fx%.3f), want %.4f",
			gotRatio, img.W, img.H, wantRatio)
	}
}

// w3schools-4: the right-rail badge is a width:auto image inside an anchor
// and a picture element. It keeps its intrinsic 300x300 size in the wide rail
// instead of stretching to the column.
func TestInlineAutoImageKeepsIntrinsicSize(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.side { width: 363pt }
.side img { width: auto; border-radius: 5px }
`)

	res := layoutHTMLWithImages(t,
		`<html><body><div class="side"><a href="#"><picture>`+
			`<img src="x.png" style="width:auto;border-radius:5px">`+
			`</picture></a></div></body></html>`,
		tinyPNG(300, 300), "", cssSheet)

	img := oneImageOp(t, res)
	wantW := 300 * 0.75
	wantH := wantW // square intrinsic

	if !near(img.W, wantW) || !near(img.H, wantH) {
		t.Fatalf("rail badge = %.3fx%.3f at x=%.2f, want %.3fx%.3f (intrinsic 300px)",
			img.W, img.H, img.X, wantW, wantH)
	}
}
