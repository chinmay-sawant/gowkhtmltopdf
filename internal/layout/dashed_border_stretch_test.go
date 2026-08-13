//nolint:testpackage // white-box: asserts dash segment geometry after Paint
package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func loadGoldenHTMLAndSheet(t *testing.T, name string) (*html.Node, *css.Stylesheet) {
	t.Helper()

	htmlBytes, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(string(htmlBytes))
	if err != nil {
		t.Fatal(err)
	}

	var styleText strings.Builder

	root.Walk(func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "style" {
			styleText.WriteString(n.TextContent())
		}
	})

	sheet, err := css.Parse(styleText.String())
	if err != nil {
		t.Fatal(err)
	}

	return root, sheet
}

// TestDashedBorderVerticalsStaySegmented guards fixture-40 (.abs-host) and
// fixture-48 (.tracking): stretchPaginatedChrome must not elongate the last
// left/right dash into a solid stub past the bottom edge.
func TestDashedBorderVerticalsStaySegmented(t *testing.T) {
	t.Parallel()

	cases := []struct {
		file  string
		class string
	}{
		{"fixture-40-transform-badge.html", "abs-host"},
		{"fixture-48-shipping-document.html", "tracking"},
	}

	for _, tc := range cases {
		t.Run(tc.class, func(t *testing.T) {
			t.Parallel()

			root, sheet := loadGoldenHTMLAndSheet(t, tc.file)

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

			if err := Paint(pdf.NewDocument(), res, PaintOptions{
				PageWidth: pageW, PageHeight: pageH,
				MarginTop: margin, MarginBottom: margin,
				MarginLeft: margin, MarginRight: margin,
			}); err != nil {
				t.Fatal(err)
			}

			host := findBoxByClass(t, res, tc.class)

			boxBottom := host.y + host.height
			var bottomEdgeY float64
			var hasBottomEdge bool
			var leftSegs, rightSegs int

			for i := host.opStart; i <= host.opEnd && i < len(res.Ops); i++ {
				op := res.Ops[i]
				if op.Kind != OpLine {
					continue
				}

				// Bottom horizontal dashes (or solid).
				if op.H == 0 && op.W > 0 && nearLayout(op.Y, boxBottom) {
					if !hasBottomEdge || op.Y > bottomEdgeY {
						bottomEdgeY = op.Y
						hasBottomEdge = true
					}
				}

				if op.W != 0 || op.H <= 0 {
					continue
				}

				onLeft := nearLayout(op.X, host.x)
				onRight := nearLayout(op.X, host.x+host.w)
				if !onLeft && !onRight {
					continue
				}

				if onLeft {
					leftSegs++
				}
				if onRight {
					rightSegs++
				}

				// Each vertical piece must stay dash-sized, not a rail stub.
				maxSeg := op.Width*three + 1
				if op.Width <= 0 {
					maxSeg = three + 1
				}
				if op.H > maxSeg {
					t.Fatalf("%s vertical dash elongated: x=%.2f y=%.2f h=%.2f width=%.2f (maxSeg=%.2f boxBottom=%.2f)",
						tc.class, op.X, op.Y, op.H, op.Width, maxSeg, boxBottom)
				}

				if op.Y+op.H > boxBottom+1 {
					t.Fatalf("%s vertical dash past box bottom: y+h=%.2f boxBottom=%.2f",
						tc.class, op.Y+op.H, boxBottom)
				}
			}

			if leftSegs < 2 || rightSegs < 2 {
				t.Fatalf("%s expected multi-segment dashed sides, got left=%d right=%d",
					tc.class, leftSegs, rightSegs)
			}

			if !hasBottomEdge {
				t.Fatalf("%s missing bottom border edge near y=%.2f", tc.class, boxBottom)
			}

			// Side rails must meet the bottom edge within one dash period, not
			// overshoot it with a solid stub (the original bug).
			var maxSideBottom float64
			for i := host.opStart; i <= host.opEnd && i < len(res.Ops); i++ {
				op := res.Ops[i]
				if op.Kind != OpLine || op.W != 0 || op.H <= 0 {
					continue
				}
				if !nearLayout(op.X, host.x) && !nearLayout(op.X, host.x+host.w) {
					continue
				}
				if bot := op.Y + op.H; bot > maxSideBottom {
					maxSideBottom = bot
				}
			}

			if maxSideBottom > bottomEdgeY+1 {
				t.Fatalf("%s side verticals end at %.2f past bottom edge %.2f",
					tc.class, maxSideBottom, bottomEdgeY)
			}
		})
	}
}
