//nolint:testpackage,wsl // caption-side layout probes
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
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

func layoutCaptionSide(t *testing.T, src, _ string, sheets ...*css.Stylesheet) *Result {
	t.Helper()

	return layoutHTML(t, src, sheets...)
}
