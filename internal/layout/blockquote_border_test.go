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

	var styleText strings.Builder

	var walkStyle func(*html.Node)
	walkStyle = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "style" {
			for _, c := range n.Children {
				if c.Type == html.TextNode {
					styleText.WriteString(c.Text)
				}
			}
		}

		for _, c := range n.Children {
			walkStyle(c)
		}
	}
	walkStyle(root)

	sheet, err := css.Parse(styleText.String())
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

	var bq *box

	var findBQ func(*box)
	findBQ = func(b *box) {
		if b == nil {
			return
		}

		if b.node != nil && b.node.Name == "blockquote" {
			bq = b
		}

		for _, c := range b.children {
			findBQ(c)
		}
	}
	findBQ(res.root)

	if bq == nil {
		t.Fatal("blockquote box missing")
	}

	preH := 0.0

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.W < 1 && op.H > 10 && nearLayout(op.X, bq.x) {
			preH = op.H

			break
		}
	}

	if preH <= 0 {
		t.Fatal("blockquote border-left missing before Paint")
	}

	if math.Abs(preH-bq.height) > 0.5 {
		t.Fatalf("pre-paint border h=%.2f, box h=%.2f", preH, bq.height)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{ //nolint:exhaustruct // intentional zero fields
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	postH := 0.0
	postY := 0.0

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.W < 1 && op.H > 10 && nearLayout(op.X, bq.x) {
			postH = op.H
			postY = op.Y

			break
		}
	}

	if postH <= 0 {
		t.Fatal("blockquote border-left missing after Paint")
	}

	// Must stay at layout height — not stretched by baseline+line-height.
	if postH > bq.height+1 {
		t.Fatalf("border stretched after Paint: h=%.2f, box h=%.2f (pre=%.2f)", postH, bq.height, preH)
	}

	if math.Abs(postH-preH) > 1 {
		t.Fatalf("border height changed in Paint: pre=%.2f post=%.2f", preH, postH)
	}

	// Last blockquote text must sit inside the rail.
	var lastTextBot float64

	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		if !strings.Contains(op.Text, "Blockquote") && !strings.Contains(op.Text, "segment") &&
			!strings.Contains(op.Text, "CEO") {
			continue
		}

		bot := op.Y + opVisibleInkHeight(op)
		if bot > lastTextBot {
			lastTextBot = bot
		}
	}

	if lastTextBot <= 0 {
		t.Fatal("blockquote text not found")
	}

	railBot := postY + postH
	if lastTextBot > railBot+1 {
		t.Fatalf("rail ends above text: railBot=%.2f textBot=%.2f", railBot, lastTextBot)
	}
}
