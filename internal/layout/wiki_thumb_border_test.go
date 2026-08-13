//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func wikiThumbPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0xc9, 0xfe, 0x92, 0xef, 0x00, 0x00, 0x00,
		0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func wikiThumbCSS() string {
	return `
body { margin: 0; font-size: 10pt; }
figure[typeof~="mw:File/Thumb"] {
  display: table;
  float: right;
  clear: right;
  margin: 0.5em 0 1.3em 1.4em;
  border: 1px solid #c8ccd1;
  border-bottom: 0;
  border-collapse: collapse;
  background-color: #f8f9fa;
}
figure[typeof~="mw:File/Thumb"] > figcaption {
  display: table-caption;
  caption-side: bottom;
  border: 1px solid #c8ccd1;
  border-top: 0;
  font-size: 94%;
  padding: 0 6px 6px 6px;
}
img { display: block; border: 1px solid #c8ccd1; }
p { margin: 0; }
`
}

func layoutWikiThumb(t *testing.T, extraLeading string, height float64) *Result {
	t.Helper()

	png := wikiThumbPNG()
	cssSheet := sheet(t, wikiThumbCSS())

	root, err := html.Parse(`<html><body>` + extraLeading + `
<figure typeof="mw:File/Thumb">
<a href="/wiki/File:x.jpg"><img class="mw-file-element" width="125" height="88" src="thumb.png"></a>
<figcaption>De Armas (standing at the center with number 4) with the cast of El internado in 2008</figcaption>
</figure>
<p>` + strings.Repeat("more ", 40) + `</p>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // probe uses the standard print layout path
		Width: 538, Height: height, Sheets: []*css.Stylesheet{cssSheet},
		Background: true,
		Images: func(src string) ([]byte, error) {
			if strings.Contains(src, "thumb.png") {
				return png, nil
			}

			return nil, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: 595.28, PageHeight: height + 2*28.35,
		MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35,
	}); err != nil {
		t.Fatal(err)
	}

	return res
}

func findNamedBox(root *box, name string) *box {
	if root == nil {
		return nil
	}

	if root.node != nil && root.node.Name == name {
		return root
	}

	for _, child := range root.children {
		if found := findNamedBox(child, name); found != nil {
			return found
		}
	}

	return nil
}

func isVerticalRail(op Op) bool {
	return (op.Kind == OpLine && op.W == 0 && op.H > 8) ||
		(op.Kind == OpStrokeRect && (op.StrokeMask == StrokeMaskLeft || op.StrokeMask == StrokeMaskRight) && op.H > 8)
}

//nolint:cyclop,gocognit,paralleltest // renderer fixture uses shared font state
func TestWikiThumbCollapsedFrameHasOneSideRail(t *testing.T) {
	res := layoutWikiThumb(t, `<p>`+strings.Repeat("word ", 20)+`</p>`, 785.19)
	figure := findNamedBox(res.root, "figure")
	caption := findNamedBox(res.root, "figcaption")

	if figure == nil || caption == nil {
		t.Fatal("missing figure or figcaption box")
	}

	captionTop := caption.y
	captionBot := caption.y + caption.height
	leftRails := 0
	rightRails := 0
	figureRailBottom := 0.0

	for _, paintOp := range res.Ops {
		if !isVerticalRail(paintOp) {
			continue
		}

		opBot := paintOp.Y + paintOp.H
		if opBot < captionTop+1 || paintOp.Y > captionBot+1 {
			continue
		}

		if math.Abs(paintOp.X-figure.x) < 1.5 {
			leftRails++

			if opBot > figureRailBottom {
				figureRailBottom = opBot
			}
		}

		if math.Abs(paintOp.X-(figure.x+figure.w)) < 1.5 {
			rightRails++
		}
	}

	if leftRails != 1 || rightRails != 1 {
		t.Fatalf("caption-band side rails left=%d right=%d, want 1 and 1 (collapsed frame)", leftRails, rightRails)
	}

	if figureRailBottom > captionBot+1 {
		t.Fatalf("figure side rail ends at %.2f, caption box ends at %.2f", figureRailBottom, captionBot)
	}

	for _, imageOp := range res.Ops {
		if imageOp.Kind != OpImage {
			continue
		}

		for _, line := range res.Ops {
			if line.Kind != OpLine || line.H != 0 || line.Width > 0.55 {
				continue
			}

			if math.Abs(line.Y-(imageOp.Y+imageOp.H)) < 1 && math.Abs(line.X-imageOp.X) < 2 && math.Abs(line.W-imageOp.W) < 4 {
				t.Fatalf("stray hairline under thumb image at y=%.2f w=%.2f", line.Y, line.W)
			}
		}
	}
}

//nolint:paralleltest // renderer fixture uses shared font state
func TestWikiThumbChromeDoesNotCloneOntoNextPage(t *testing.T) {
	// Page short enough that a line-box stretch would cross, but the
	// caption border box still fits on page 0.
	const contentH = 155.0

	res := layoutWikiThumb(t, "", contentH)
	figure := findNamedBox(res.root, "figure")
	caption := findNamedBox(res.root, "figcaption")

	if figure == nil || caption == nil {
		t.Fatal("missing figure or figcaption box")
	}

	if caption.y+caption.height > contentH {
		t.Fatalf("caption bottom %.2f already past page 0; test geometry is wrong", caption.y+caption.height)
	}

	for _, paintOp := range res.Ops {
		if !isVerticalRail(paintOp) {
			continue
		}

		if math.Abs(paintOp.X-figure.x) > 1.5 && math.Abs(paintOp.X-(figure.x+figure.w)) > 1.5 {
			continue
		}

		if paintOp.Y >= contentH-1 {
			t.Fatalf("thumb side rail cloned onto next page: x=%.2f y=%.2f h=%.2f", paintOp.X, paintOp.Y, paintOp.H)
		}
	}
}
