package layout

import "testing"

// TestAutoHeightIncludesBottomBorder: an auto-height bordered box must
// include its bottom border in the border-box height in every formatting
// context. Block and flex boxes used to add only padding-bottom while grid
// and table cells added padding plus border, so identical content produced
// different heights and the next sibling overlapped the missing edge.
func TestAutoHeightIncludesBottomBorder(t *testing.T) {
	t.Parallel()

	styleSheet := sheet(t, `
body { margin: 0 }
.wrap { padding: 4pt; border: 2pt solid #000; font-size: 10pt; line-height: 1; width: 100pt }
.grid { display: grid }
.flex { display: flex }
.next { height: 12pt }
`)

	layoutWith := func(class string) *Result {
		t.Helper()

		return layoutHTML(t, `<html><body>`+
			`<div class="wrap `+class+`"><span>x</span></div><div class="next">n</div>`+
			`</body></html>`, styleSheet)
	}

	blockRes := layoutWith("")
	gridRes := layoutWith("grid")
	flexRes := layoutWith("flex")

	blockBox := findBoxByClass(t, blockRes, "wrap")
	gridBox := findBoxByClass(t, gridRes, "wrap")
	flexBox := findBoxByClass(t, flexRes, "wrap")

	if !near(blockBox.height, gridBox.height) || !near(blockBox.height, flexBox.height) {
		t.Fatalf("auto-height bordered boxes disagree: block %.3f grid %.3f flex %.3f",
			blockBox.height, gridBox.height, flexBox.height)
	}

	// One 10pt line + 4pt padding both sides + 2pt border both sides = 22.
	if blockBox.height < 21.99 {
		t.Fatalf("block height %.3f does not include the 2pt bottom border", blockBox.height)
	}

	next := findBoxByClass(t, blockRes, "next")
	if !near(next.y, blockBox.y+blockBox.height) {
		t.Fatalf("next sibling top = %.3f, want box bottom %.3f", next.y, blockBox.y+blockBox.height)
	}
}
