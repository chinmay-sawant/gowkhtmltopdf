//nolint:testpackage // tests exercise unexported package internals via shared helpers
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
	walk = func(node *html.Node) {
		if node.Name == "style" {
			var cssSheet string

			for _, c := range node.Children {
				if c.Type == html.TextNode {
					cssSheet += c.Text
				}
			}

			if sh, err := css.Parse(cssSheet); err == nil {
				sheets = append(sheets, sh)
			}
		}

		for _, c := range node.Children {
			walk(c)
		}
	}
	walk(root)

	return root, sheets
}

func TestLogoTitleGap(t *testing.T) { //nolint:cyclop
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-07-image-logo.html")

	logo, err := os.ReadFile("../../testdata/golden/logo.png")
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
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

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpImage && paintOp.Y < 50 {
			imgRight = paintOp.X + paintOp.W
			sawImg = true
		}

		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "Nordwind Industries GmbH") && paintOp.Y < 50 {
			textX = paintOp.X
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
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-19-margin-and-sizing.html")

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 595, Height: 842, Sheets: sheets, Background: true,
	})
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
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-10-table-colspan.html")
	contW := 595.0 - 2*28.346

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contW, Height: 842, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, paintOp := range res.Ops {
		right := paintOp.X + paintOp.W
		if paintOp.Kind == OpLine && paintOp.W == 0 {
			right = paintOp.X
		}

		if right > contW+1 {
			t.Errorf("op exceeds content: kind=%v X=%.1f W=%.1f right=%.1f cw=%.1f",
				paintOp.Kind, paintOp.X, paintOp.W, right, contW)
		}
	}
}

func TestPositionLiteFixtureReservesOverlaySpace(t *testing.T) { //nolint:cyclop
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-26-position-lite.html")

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 595 - 56.7, Height: 842, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	absBottom := 0.0
	flowY := 0.0
	flowBold := false
	relLeft := 0.0
	relTextX := 0.0

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpFillRect && paintOp.W > 400 && paintOp.H > 20 &&
			paintOp.R > 0.9 && paintOp.G > 0.9 && paintOp.B > 0.9 {
			relLeft = paintOp.X
		}

		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "Relatively offset block") {
			relTextX = paintOp.X
		}

		if paintOp.Kind == OpFillRect && paintOp.W > 100 && paintOp.H > 10 &&
			paintOp.R > 0.9 && paintOp.G > 0.85 && paintOp.G < 0.99 && paintOp.B > 0.8 && paintOp.B < 0.95 {
			absBottom = paintOp.Y + paintOp.H
		}

		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "In-flow text under") {
			flowY = paintOp.Y
			flowBold = paintOp.Bold
		}
	}

	if relLeft == 0 || relTextX == 0 || absBottom == 0 || flowY == 0 {
		t.Fatalf("relative left=%.1f text x=%.1f overlay bottom=%.1f flow y=%.1f", relLeft, relTextX, absBottom, flowY)
	}

	if relTextX < relLeft {
		t.Fatalf("relative text x=%.1f is outside green box left=%.1f", relTextX, relLeft)
	}

	if flowY <= absBottom {
		t.Fatalf("fixture flow text y=%.1f overlaps absolute overlay bottom=%.1f", flowY, absBottom)
	}

	if flowBold {
		t.Fatal("fixture flow text unexpectedly uses bold font weight")
	}
}

func TestLetterheadPaddingBeforeBorder(t *testing.T) { //nolint:cyclop
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-16-invoice-with-css.html")

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 595 - 56.7, Height: 842, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var addrY, borderY float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "Hafenstrasse") {
			addrY = paintOp.Y
		}

		if paintOp.Kind == OpLine && paintOp.Width >= 2 && paintOp.W > 400 && paintOp.Y < 80 {
			borderY = paintOp.Y
		}
	}

	if addrY == 0 || borderY == 0 {
		t.Fatalf("addrY=%.1f borderY=%.1f", addrY, borderY)
	}

	if borderY-addrY < 5 {
		t.Errorf("gap address→border = %.2f, want >= ~6pt", borderY-addrY)
	}
}
