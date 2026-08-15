//nolint:testpackage // white-box: asserts dash segment geometry after Paint
package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
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

	root.Walk(func(node *html.Node) {
		if node.Type == html.ElementNode && node.Name == styleElement {
			styleText.WriteString(node.TextContent())
		}
	})

	sheet, err := css.Parse(styleText.String())
	if err != nil {
		t.Fatal(err)
	}

	return root, sheet
}

func findBottomEdgeY(host *box, ops []Op) (float64, bool) {
	boxBottom := host.y + host.height
	bottomEdgeY := 0.0
	hasBottomEdge := false

	for i := host.opStart; i <= host.opEnd && i < len(ops); i++ {
		operation := ops[i]
		if operation.Kind != OpLine {
			continue
		}

		if operation.H == 0 && operation.W > 0 && nearLayout(operation.Y, boxBottom) {
			if !hasBottomEdge || operation.Y > bottomEdgeY {
				bottomEdgeY = operation.Y
				hasBottomEdge = true
			}
		}
	}

	return bottomEdgeY, hasBottomEdge
}

func isSideVerticalDash(operation Op, host *box) (bool, bool) {
	if operation.Kind != OpLine || operation.W != 0 || operation.H <= 0 {
		return false, false
	}

	return nearLayout(operation.X, host.x), nearLayout(operation.X, host.x+host.w)
}

func verifyVerticalDashHeight(t *testing.T, operation Op, boxBottom float64, className string) {
	t.Helper()

	maxSeg := operation.Width*three + 1
	if operation.Width <= 0 {
		maxSeg = three + 1
	}

	if operation.H > maxSeg {
		t.Fatalf("%s vertical dash elongated: x=%.2f y=%.2f h=%.2f width=%.2f (maxSeg=%.2f boxBottom=%.2f)",
			className, operation.X, operation.Y, operation.H, operation.Width, maxSeg, boxBottom)
	}

	if operation.Y+operation.H > boxBottom+1 {
		t.Fatalf("%s vertical dash past box bottom: y+h=%.2f boxBottom=%.2f",
			className, operation.Y+operation.H, boxBottom)
	}
}

func verifyVerticalDashes(t *testing.T, host *box, ops []Op, className string) (int, int) {
	t.Helper()

	boxBottom := host.y + host.height
	leftSegs := 0
	rightSegs := 0

	for i := host.opStart; i <= host.opEnd && i < len(ops); i++ {
		operation := ops[i]

		onLeft, onRight := isSideVerticalDash(operation, host)
		if !onLeft && !onRight {
			continue
		}

		if onLeft {
			leftSegs++
		}

		if onRight {
			rightSegs++
		}

		verifyVerticalDashHeight(t, operation, boxBottom, className)
	}

	return leftSegs, rightSegs
}

func findMaxSideBottom(host *box, ops []Op) float64 {
	var maxSideBottom float64

	for i := host.opStart; i <= host.opEnd && i < len(ops); i++ {
		operation := ops[i]
		if operation.Kind != OpLine || operation.W != 0 || operation.H <= 0 {
			continue
		}

		if !nearLayout(operation.X, host.x) && !nearLayout(operation.X, host.x+host.w) {
			continue
		}

		if bot := operation.Y + operation.H; bot > maxSideBottom {
			maxSideBottom = bot
		}
	}

	return maxSideBottom
}

func verifyDashedBorderGolden(t *testing.T, file, className string) {
	t.Helper()

	root, sheet := loadGoldenHTMLAndSheet(t, file)

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

	host := findBoxByClass(t, res, className)
	bottomEdgeY, hasBottomEdge := findBottomEdgeY(host, res.Ops)
	leftSegs, rightSegs := verifyVerticalDashes(t, host, res.Ops, className)

	if leftSegs < 2 || rightSegs < 2 {
		t.Fatalf("%s expected multi-segment dashed sides, got left=%d right=%d",
			className, leftSegs, rightSegs)
	}

	if !hasBottomEdge {
		t.Fatalf("%s missing bottom border edge near y=%.2f", className, host.y+host.height)
	}

	maxSideBottom := findMaxSideBottom(host, res.Ops)
	if maxSideBottom > bottomEdgeY+1 {
		t.Fatalf("%s side verticals end at %.2f past bottom edge %.2f",
			className, maxSideBottom, bottomEdgeY)
	}
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

	for _, testCase := range cases {
		t.Run(testCase.class, func(t *testing.T) {
			t.Parallel()

			verifyDashedBorderGolden(t, testCase.file, testCase.class)
		})
	}
}
