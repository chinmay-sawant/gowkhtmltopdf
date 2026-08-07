package layout

import (
	"math"
	"os"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// TestFixture13PreBackgroundTouchesBottomBorder locks the visual contract for
// the final pre block: its background must cover its bottom padding right up
// to the border instead of being mistaken for page-trailing section chrome.
func TestFixture13PreBackgroundTouchesBottomBorder(t *testing.T) {
	res, _, _ := paintGoldenFixture(t, "fixture-13-pre-code-block.html")

	var fill, bottom *Op

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind == OpFillRect && nearRGB(paintOp, 0.957, 0.957, 0.949) && paintOp.Y > 400 && (fill == nil || paintOp.Y > fill.Y) {
			fill = paintOp
		}

		if paintOp.Kind == OpLine && nearRGB(paintOp, 0.847, 0.847, 0.831) && paintOp.W > 500 && paintOp.H < 1 && paintOp.Y > 400 && (bottom == nil || paintOp.Y > bottom.Y) {
			bottom = paintOp
		}
	}

	if fill == nil || bottom == nil {
		t.Fatalf("final pre chrome missing: fill=%+v bottom=%+v", fill, bottom)
	}

	if gap := bottom.Y - (fill.Y + fill.H); math.Abs(gap) > 0.5 {
		t.Fatalf("final pre background stops %.2fpt before its bottom border", gap)
	}
}

func TestStickySectionChromeRepairUsesContainingBlock(t *testing.T) {
	t.Parallel()
	borderRGB := [3]float64{0.271, 0.353, 0.392}
	background := [4]float64{0.925, 0.937, 0.945, 1}
	section := &box{ //nolint:exhaustruct // intentional zero fields
		x: 10, y: 0, w: 100,
		style: ResolvedStyle{ //nolint:exhaustruct // intentional zero fields
			BGColor:      background,
			BorderLeft:   border{Width: 1, Style: "solid", Color: borderRGB},
			BorderRight:  border{Width: 1, Style: "solid", Color: borderRGB},
			BorderBottom: border{Width: 1, Style: "solid", Color: borderRGB},
		},
	}
	section.children = []*box{{sticky: true}}
	root := &box{children: []*box{section}}                                                                                             //nolint:exhaustruct // intentional zero fields
	priorFill := Op{Kind: OpFillRect, X: 10, Y: 0, W: 100, H: 50, R: background[0], G: background[1], B: background[2]}                 //nolint:exhaustruct // intentional zero fields
	targetFill := Op{Kind: OpFillRect, X: 10, Y: 100, W: 100, H: 50, R: background[0], G: background[1], B: background[2]}              //nolint:exhaustruct // intentional zero fields
	targetLeft := Op{Kind: OpLine, X: 10, Y: 100, H: 50, R: borderRGB[0], G: borderRGB[1], B: borderRGB[2]}                             //nolint:exhaustruct // intentional zero fields
	targetRight := Op{Kind: OpLine, X: 110, Y: 100, H: 50, R: borderRGB[0], G: borderRGB[1], B: borderRGB[2]}                           //nolint:exhaustruct // intentional zero fields
	targetBottom := Op{Kind: OpLine, X: 10, Y: 180, W: 100, R: borderRGB[0], G: borderRGB[1], B: borderRGB[2]}                          //nolint:exhaustruct // intentional zero fields
	unrelatedFill := Op{Kind: OpFillRect, X: 200, Y: 100, W: 300, H: 50, R: background[0], G: background[1], B: background[2]}          //nolint:exhaustruct // intentional zero fields
	unrelatedBottom := Op{Kind: OpLine, X: 200, Y: 180, W: 300, R: borderRGB[0], G: borderRGB[1], B: borderRGB[2]}                      //nolint:exhaustruct // intentional zero fields
	res := &Result{root: root, Ops: []Op{priorFill, targetFill, targetLeft, targetRight, targetBottom, unrelatedFill, unrelatedBottom}} //nolint:exhaustruct // intentional zero fields

	closePageLeadingSectionChrome(res, 100)

	if got := res.Ops[0].H; got != 50 {
		t.Errorf("prior-page fill height = %.1f, want unchanged 50", got)
	}

	if got := res.Ops[1].H; got != 80 {
		t.Errorf("target fill height = %.1f, want 80", got)
	}

	if got := res.Ops[2].H; got != 80 {
		t.Errorf("target left border height = %.1f, want 80", got)
	}

	if got := res.Ops[3].H; got != 80 {
		t.Errorf("target right border height = %.1f, want 80", got)
	}

	if got := res.Ops[5].H; got != 50 {
		t.Errorf("unrelated fill height = %.1f, want unchanged 50", got)
	}

	if got := res.Ops[6].W; got != 300 {
		t.Errorf("unrelated bottom width = %.1f, want unchanged 300", got)
	}
}

func paintGoldenFixture(t *testing.T, name string) (*Result, float64, *pdf.Document) {
	t.Helper()

	src, err := os.ReadFile("../../testdata/golden/" + name)
	if err != nil {
		t.Fatal(err)
	}

	htmlSrc := string(src)
	sidx := strings.Index(htmlSrc, "<style>")
	sjd := strings.Index(htmlSrc, "</style>")

	if sidx < 0 || sjd < sidx {
		t.Fatalf("%s: missing style", name)
	}

	sheet, err := css.Parse(htmlSrc[sidx+len("<style>") : sjd])
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	pageW, pageH := 595.28, 841.89
	mat := 28.35
	contentH := pageH - 2*mat
	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: pageW - 2*mat, Height: contentH, Background: true,
		Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: mat, MarginBottom: mat, MarginLeft: mat, MarginRight: mat,
	}); err != nil {
		t.Fatal(err)
	}

	return res, contentH, doc
}

func nearRGB(op *Op, r, g, b float64) bool {
	return math.Abs(op.R-r) < 0.01 && math.Abs(op.G-g) < 0.01 && math.Abs(op.B-b) < 0.01
}
