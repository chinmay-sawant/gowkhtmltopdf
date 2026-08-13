//nolint:testpackage // tests inspect unexported box tree geometry
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

func extractStyleContent(root *html.Node) string {
	var styleText strings.Builder

	var walkStyle func(*html.Node)
	walkStyle = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Name == styleElement {
			for _, child := range node.Children {
				if child.Type == html.TextNode {
					styleText.WriteString(child.Text)
				}
			}
		}

		for _, child := range node.Children {
			walkStyle(child)
		}
	}
	walkStyle(root)

	return styleText.String()
}

func findBlockquoteBox(root *box) *box {
	var blockquoteBox *box

	var findBQ func(*box)
	findBQ = func(currentBox *box) {
		if currentBox == nil {
			return
		}

		if currentBox.node != nil && currentBox.node.Name == "blockquote" {
			blockquoteBox = currentBox
		}

		for _, child := range currentBox.children {
			findBQ(child)
		}
	}
	findBQ(root)

	return blockquoteBox
}

func findBorderLeftRail(ops []Op, boxX float64) (float64, float64) {
	for _, operation := range ops {
		if operation.Kind == OpLine && operation.W < 1 && operation.H > 10 && nearLayout(operation.X, boxX) {
			return operation.H, operation.Y
		}
	}

	return 0, 0
}

func findLastBlockquoteTextBottom(ops []Op) float64 {
	var lastTextBot float64

	for _, operation := range ops {
		if operation.Kind != OpText {
			continue
		}

		if !strings.Contains(operation.Text, "Blockquote") && !strings.Contains(operation.Text, "segment") &&
			!strings.Contains(operation.Text, "CEO") {
			continue
		}

		bot := operation.Y + opVisibleInkHeight(operation)
		if bot > lastTextBot {
			lastTextBot = bot
		}
	}

	return lastTextBot
}

func verifyPostPaintBlockquote(t *testing.T, res *Result, blockquoteBox *box, preH float64) {
	t.Helper()

	postH, postY := findBorderLeftRail(res.Ops, blockquoteBox.x)
	if postH <= 0 {
		t.Fatal("blockquote border-left missing after Paint")
	}

	// Must stay at layout height — not stretched by baseline+line-height.
	if postH > blockquoteBox.height+1 {
		t.Fatalf("border stretched after Paint: h=%.2f, box h=%.2f (pre=%.2f)", postH, blockquoteBox.height, preH)
	}

	if math.Abs(postH-preH) > 1 {
		t.Fatalf("border height changed in Paint: pre=%.2f post=%.2f", preH, postH)
	}

	lastTextBot := findLastBlockquoteTextBottom(res.Ops)
	if lastTextBot <= 0 {
		t.Fatal("blockquote text not found")
	}

	railBot := postY + postH
	if lastTextBot > railBot+1 {
		t.Fatalf("rail ends above text: railBot=%.2f textBot=%.2f", railBot, lastTextBot)
	}
}

// TestBlockquoteBorderLeftMatchesContentHeight guards fixture-18: after Paint,
// border-left on a blockquote must not stretch past the block's used height
// (stretchPaginatedChrome used to treat text Y+H as line-box bottom while Y is
// the baseline, overshooting by ~ascent into the following margin).
func TestBlockquoteBorderLeftMatchesContentHeight(t *testing.T) {
	t.Parallel()

	htmlBytes, err := os.ReadFile("../../testdata/golden/fixture-18-typography.html")
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(string(htmlBytes))
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(extractStyleContent(root))
	if err != nil {
		t.Fatal(err)
	}

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	const (
		pageW  = 595.28
		pageH  = 841.89
		margin = 28.35
	)

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: pageW - 2*margin, Height: pageH - 2*margin,
		Sheets: []*css.Stylesheet{sheet}, Background: true, Faces: faces,
	})
	if err != nil {
		t.Fatal(err)
	}

	blockquoteBox := findBlockquoteBox(res.root)
	if blockquoteBox == nil {
		t.Fatal("blockquote box missing")
	}

	preH, _ := findBorderLeftRail(res.Ops, blockquoteBox.x)
	if preH <= 0 {
		t.Fatal("blockquote border-left missing before Paint")
	}

	if math.Abs(preH-blockquoteBox.height) > 0.5 {
		t.Fatalf("pre-paint border h=%.2f, box h=%.2f", preH, blockquoteBox.height)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	verifyPostPaintBlockquote(t, res, blockquoteBox, preH)
}
