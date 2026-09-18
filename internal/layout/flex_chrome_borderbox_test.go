package layout

import "testing"

// TestChromeFlexCrossSizeBorderBox covers case wpt-flex-cross-size-border-box
// (test/Chrome/manifest.json goTarget layout-unit).
//
// Source: third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-cross-size-border-box-001.html
// Expected: a border-box container and a content-box container that both leave
// the same 180pt content cross size stretch their children to that same height.
// The border-box container gets there with height:200pt minus 10pt border on
// each side; the content-box container with height:180pt plus the same border.
// The content-box container's border box is therefore taller than its own
// content box by exactly its padding plus border (20pt).
func TestChromeFlexCrossSizeBorderBox(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.flex { display: flex; width: 400pt }
.borderbox { height: 200pt; box-sizing: border-box; border: 10pt solid #000 }
.contentbox { height: 180pt; border: 10pt solid #000 }
`)
	res := layoutHTML(t, `<html><body>
<div class="flex borderbox"><div class="borderchild"></div></div>
<div class="flex contentbox"><div class="contentchild"></div></div>
</body></html>`, cssSheet)

	borderBox := findBoxByClass(t, res, "borderbox")
	contentBox := findBoxByClass(t, res, "contentbox")
	borderChild := findBoxByClass(t, res, "borderchild")
	contentChild := findBoxByClass(t, res, "contentchild")

	// 180 = border-box 200 - 2*10 border = content-box 180 (no subtraction).
	const (
		wantContentHeight = 180.0
		wantChrome        = 20.0
	)

	if !near(borderBox.height, wantContentHeight+wantChrome) {
		t.Fatalf("border-box container height = %.2f, want %.2f", borderBox.height, wantContentHeight+wantChrome)
	}

	if !near(contentBox.height, wantContentHeight+wantChrome) {
		t.Fatalf("content-box container height = %.2f, want %.2f", contentBox.height, wantContentHeight+wantChrome)
	}

	// Both stretched children must fill the container content cross size.
	if !near(borderChild.height, wantContentHeight) || !near(contentChild.height, wantContentHeight) {
		t.Fatalf("stretched child heights = %.2f/%.2f, want %.2f",
			borderChild.height, contentChild.height, wantContentHeight)
	}

	if !near(borderChild.y, borderBox.y+10) || !near(contentChild.y, contentBox.y+10) {
		t.Fatalf("stretched child tops = %.2f/%.2f, want content top %.2f/%.2f",
			borderChild.y, contentChild.y, borderBox.y+10, contentBox.y+10)
	}

	// The content-box container is taller than its content box by its
	// padding plus border, which is why it needs height:180pt to match.
	if !near(contentBox.height-contentChild.height, wantChrome) {
		t.Fatalf("content-box chrome = %.2f, want %.2f", contentBox.height-contentChild.height, wantChrome)
	}
}

// TestChromeFlexAutoMinSizeOverflowClip covers case
// wpt-min-size-auto-overflow-clip (test/Chrome/manifest.json goTarget
// chrome-reference).
//
// Source: third_party/blink/web_tests/external/wpt/css/css-flexbox/min-size-auto-overflow-clip.html
// Expected: overflow: clip must behave like overflow: visible for the
// automatic minimum size. The item's min-content width is the 150pt specified
// on its block child, so the item keeps 150pt and overflows the 100pt content
// box instead of shrinking below it. Chromium's reftest ref
// (min-size-auto-overflow-clip-ref.html) renders the same 150pt green block.
//
// Blocked: the engine caps an auto flex item's base width at the container
// content width (internal/layout/flex.go:615) and zeroes the automatic minimum
// size for overflow: clip (internal/layout/flex.go:747 via
// internal/layout/style_values.go:696). The strict assertions below stay so the
// case flips green when those two paths are fixed.
func TestChromeFlexAutoMinSizeOverflowClip(t *testing.T) {
	t.Parallel()

	t.Skip("blocked: capped auto base width and zeroed overflow:clip minimum")

	cssSheet := sheet(t, `
body { margin: 0 }
.flex { display: flex; width: 100pt; border: 1pt solid #000 }
.item { overflow: clip }
.inner { width: 150pt; height: 50pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="flex"><div class="item"><div class="inner"></div></div></div>
</body></html>`, cssSheet)

	flexBox := findBoxByClass(t, res, "flex")
	item := findBoxByClass(t, res, "item")
	inner := findBoxByClass(t, res, "inner")

	// 102 = 100pt content + 1pt border on each side.
	if !near(flexBox.w, 102) {
		t.Fatalf("container border-box width = %.2f, want 102", flexBox.w)
	}

	// Automatic minimum size: min-content width from the block child's 150pt.
	if !near(item.w, 150) {
		t.Fatalf("item width = %.2f, want 150 (min-content floor, no shrink)", item.w)
	}

	if !near(item.x, flexBox.x+1) {
		t.Fatalf("item x = %.2f, want %.2f (container content left)", item.x, flexBox.x+1)
	}

	if item.x+item.w <= flexBox.x+1+100 {
		t.Fatalf("item right = %.2f, want overflow past content right %.2f",
			item.x+item.w, flexBox.x+1+100)
	}

	if !near(inner.w, 150) || !near(inner.height, 50) {
		t.Fatalf("inner size = %.2fx%.2f, want 150x50", inner.w, inner.height)
	}
}

// TestChromeFlexScrollbarsRowReverseVRL covers case
// blink-scrollbars-row-reverse-vrl (test/Chrome/manifest.json goTarget
// chrome-reference).
//
// Source: third_party/blink/renderer/core/layout/layout_flexible_box_test.cc
// (TEST_F GeometriesWithScrollbarsRowReverseVRL, child expected at
// PhysicalOffset(-1525, -686)).
// Expected: writing-mode: vertical-rl puts a row's main axis on the vertical
// axis, so flex-direction: row-reverse anchors the item to the container's
// bottom edge; the block axis (cross axis) points right-to-left, so a
// 500pt-wide item in a 400pt container lands at negative physical coordinates.
//
// Blocked: flex layout ignores writing-mode and lays rows out on the physical
// horizontal axis (internal/layout/layout.go:1914 documents that vertical
// writing modes keep horizontal block flow). The strict assertion stays so the
// case flips green when logical-axis flex mapping lands.
func TestChromeFlexScrollbarsRowReverseVRL(t *testing.T) {
	t.Parallel()

	t.Skip("blocked: flex row ignores writing-mode, block flow stays horizontal (internal/layout/layout.go:1914)")

	cssSheet := sheet(t, `
body { margin: 0 }
.box {
  display: flex;
  writing-mode: vertical-rl;
  flex-direction: row-reverse;
  width: 400pt;
  height: 300pt;
}
.child { flex: none; width: 500pt; height: 500pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="box"><div class="child"></div></div>
</body></html>`, cssSheet)

	box := findBoxByClass(t, res, "box")
	child := findBoxByClass(t, res, "child")

	if !near(child.w, 500) || !near(child.height, 500) {
		t.Fatalf("child size = %.2fx%.2f, want 500x500", child.w, child.height)
	}

	// vertical-rl row main axis runs top-to-bottom, so row-reverse starts at
	// the container bottom: child top = container bottom - child main size.
	// Cross axis runs right-to-left, so cross-start is the right edge:
	// child left = container right - child cross size.
	wantX := box.x + box.w - child.w
	wantY := box.y + box.height - child.height

	if !near(child.x, wantX) || !near(child.y, wantY) {
		t.Fatalf("child position = (%.2f, %.2f), want (%.2f, %.2f)", child.x, child.y, wantX, wantY)
	}

	if child.y >= box.y || child.x >= box.x+box.w {
		t.Fatalf("child = (%.2f, %.2f), want overflow into negative physical coordinates from box (%.2f, %.2f)",
			child.x, child.y, box.x, box.y)
	}
}
