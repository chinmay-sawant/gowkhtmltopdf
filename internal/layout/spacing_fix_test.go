package layout

import (
	"os"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func loadFixture(t *testing.T, name string) (*html.Node, []*css.Stylesheet) {
	t.Helper()
	b, err := os.ReadFile("../../testdata/golden/" + name)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}
	var sheets []*css.Stylesheet
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Name == "style" {
			var s string
			for _, c := range n.Children {
				if c.Type == html.TextNode {
					s += c.Text
				}
			}
			if sh, err := css.Parse(s); err == nil {
				sheets = append(sheets, sh)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return root, sheets
}

func TestLogoTitleGap(t *testing.T) {
	root, sheets := loadFixture(t, "fixture-07-image-logo.html")
	logo, err := os.ReadFile("../../testdata/golden/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 595 - 56.7, Height: 842, Sheets: sheets, Background: true,
		Images: func(src string) ([]byte, error) {
			if strings.HasPrefix(src, "data:") {
				return nil, os.ErrNotExist
			}
			return logo, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var imgRight, textX float64
	sawImg, sawText := false, false
	for _, op := range res.Ops {
		if op.Kind == OpImage && op.Y < 50 {
			imgRight = op.X + op.W
			sawImg = true
		}
		if op.Kind == OpText && strings.Contains(op.Text, "Nordwind Industries GmbH") && op.Y < 50 {
			textX = op.X
			sawText = true
		}
	}
	if !sawImg || !sawText {
		t.Fatalf("img=%v text=%v", sawImg, sawText)
	}
	gap := textX - imgRight
	if gap < 6.5 || gap > 9 {
		t.Errorf("logo-title gap = %.2f, want ~7.5pt (10px margin-left)", gap)
	}
}

func TestBlockMarginBottomGap(t *testing.T) {
	root, sheets := loadFixture(t, "fixture-19-margin-and-sizing.html")
	res, err := Layout(root, Options{Width: 595, Height: 842, Sheets: sheets, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var fills []Op
	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.H > 15 {
			fills = append(fills, op)
		}
	}
	if len(fills) < 3 {
		t.Fatalf("fills=%d", len(fills))
	}
	gap := fills[1].Y - (fills[0].Y + fills[0].H)
	if gap < 6.5 || gap > 9 {
		t.Errorf("gap between first boxes = %.2f, want ~7.5pt", gap)
	}
}

func TestNestedTableStaysInCell(t *testing.T) {
	root, sheets := loadFixture(t, "fixture-10-table-colspan.html")
	cw := 595.0 - 2*28.346
	res, err := Layout(root, Options{Width: cw, Height: 842, Sheets: sheets, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range res.Ops {
		right := op.X + op.W
		if op.Kind == OpLine && op.W == 0 {
			right = op.X
		}
		if right > cw+1 {
			t.Errorf("op exceeds content: kind=%v X=%.1f W=%.1f right=%.1f cw=%.1f", op.Kind, op.X, op.W, right, cw)
		}
	}
}

func TestLetterheadPaddingBeforeBorder(t *testing.T) {
	root, sheets := loadFixture(t, "fixture-16-invoice-with-css.html")
	res, err := Layout(root, Options{Width: 595 - 56.7, Height: 842, Sheets: sheets, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	var addrY, borderY float64
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "Hafenstrasse") {
			addrY = op.Y
		}
		if op.Kind == OpLine && op.Width >= 2 && op.W > 400 && op.Y < 80 {
			borderY = op.Y
		}
	}
	if addrY == 0 || borderY == 0 {
		t.Fatalf("addrY=%.1f borderY=%.1f", addrY, borderY)
	}
	if borderY-addrY < 5 {
		t.Errorf("gap address→border = %.2f, want >= ~6pt", borderY-addrY)
	}
}
