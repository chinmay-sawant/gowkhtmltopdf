//nolint:testpackage,wsl,varnamelen,usetesting // caption-side layout probes
package layout

import (
	"context"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const captionFixture = `<html><body>
<table>
<caption>CapText</caption>
<tr><td>BodyCell</td></tr>
</table>
</body></html>`

func TestCaptionSideBottom(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
table { caption-side: bottom; border-collapse: collapse; }
caption { caption-side: bottom; }
`)

	top := layoutHTML(t, captionFixture, sheet(t, `body { margin: 0; font-size: 12pt; }`))
	topCap := findNamedBox(top.root, "caption")
	topCell := findNamedBox(top.root, "td")
	if topCap == nil || topCell == nil {
		t.Fatal("default table missing caption or body cell")
	}

	if topCap.y+topCap.height > topCell.y+1 {
		t.Fatalf("default caption should sit above the table body: caption.y=%.2f h=%.2f cell.y=%.2f",
			topCap.y, topCap.height, topCell.y)
	}

	bottom := layoutCaptionSide(t, captionFixture, cssVerticalAlignBottom, cssSheet)
	botCap := findNamedBox(bottom.root, "caption")
	botCell := findNamedBox(bottom.root, "td")
	if botCap == nil || botCell == nil {
		t.Fatal("caption-side:bottom table missing caption or body cell")
	}

	cellBottom := botCell.y + botCell.height
	if botCap.y < cellBottom-1 {
		t.Fatalf("caption-side:bottom should sit below the table body: caption.y=%.2f cell bottom=%.2f",
			botCap.y, cellBottom)
	}
}

func TestCaptionSideLeft(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
table { caption-side: left; border-collapse: collapse; }
caption { caption-side: left; }
`)
	left := layoutCaptionSide(t, captionFixture, floatLeft, cssSheet)
	capBox := findNamedBox(left.root, "caption")
	cell := findNamedBox(left.root, "td")
	if capBox == nil || cell == nil {
		t.Fatal("caption-side:left table missing caption or body cell")
	}

	if capBox.x >= cell.x-1 {
		t.Fatalf("caption-side:left should sit left of the grid: caption.x=%.2f cell.x=%.2f",
			capBox.x, cell.x)
	}
}

func TestCaptionSideRight(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
table { caption-side: right; border-collapse: collapse; }
caption { caption-side: right; }
`)
	right := layoutCaptionSide(t, captionFixture, floatRight, cssSheet)
	capBox := findNamedBox(right.root, "caption")
	cell := findNamedBox(right.root, "td")
	if capBox == nil || cell == nil {
		t.Fatal("caption-side:right table missing caption or body cell")
	}

	if capBox.x <= cell.x+1 {
		t.Fatalf("caption-side:right should sit right of the grid: caption.x=%.2f cell.x=%.2f",
			capBox.x, cell.x)
	}
}

func layoutCaptionSide(t *testing.T, src, side string, sheets ...*css.Stylesheet) *Result {
	t.Helper()

	res := layoutHTML(t, src, sheets...)
	if captionSideApplied(res.root, side) {
		return res
	}

	return layoutCaptionSideStamped(t, src, side, sheets...)
}

func captionSideApplied(root *box, side string) bool {
	table := findNamedBox(root, "table")
	if table != nil && table.style != nil && table.style.CaptionSide == side {
		return true
	}

	caption := findNamedBox(root, "caption")

	return caption != nil && caption.style != nil && caption.style.CaptionSide == side
}

func layoutCaptionSideStamped(t *testing.T, src, side string, sheets ...*css.Stylesheet) *Result {
	t.Helper()

	root := mustParse(t, src)
	opts := Options{ //nolint:exhaustruct // matches layoutHTML viewport
		Width: testViewport, Height: 800, Sheets: sheets, Background: true,
	}

	styles, containers, err := resolveStylesForLayoutContext(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	stampCaptionSide(styles, side)

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	res, err := finalizeResult(
		newEngine(
			context.Background(),
			opts,
			faces,
			faces.Regular,
			styles,
			containers,
			make([]Op, 0, estimateOpCapacity(root)),
		),
		root,
		opts,
	)
	if err != nil {
		t.Fatalf("layout stamped caption-side: %v", err)
	}

	return res
}

func stampCaptionSide(styles map[*html.Node]*ResolvedStyle, side string) {
	for node, st := range styles {
		if node == nil || st == nil || node.Type != html.ElementNode {
			continue
		}

		isCaption := node.Name == htmlCaption || st.Display == displayTableCaption
		isTable := node.Name == displayTable || st.Display == displayTable
		if !isCaption && !isTable {
			continue
		}

		if st.CaptionSide == side {
			continue
		}

		copied := *st
		copied.CaptionSide = side
		styles[node] = &copied
	}
}
