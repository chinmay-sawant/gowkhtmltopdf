package layout

import "testing"

// TestChromeFlexAspectRatioCrossSizeFeedback covers case
// wpt-aspect-ratio-cross-size-002 (chrome-reference), source
// chromium/third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-aspect-ratio-cross-size-002.html.
//
// Expected Chromium behavior: nested flex sizing preserves the
// aspect-ratio-derived sizes instead of inflating the cross size through
// content min-size feedback. The 200pt wide container makes .outer
// (aspect-ratio 4) 200x50. The .inner flex item stretches to the definite
// 50pt cross size, and its nested aspect-ratio content contributes a 100pt
// main size. Units are pt instead of the source's px so 200/4 is exact.
//
//nolint:wsl // fixture assertions keep the expected geometry together
func TestChromeFlexAspectRatioCrossSizeFeedback(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.container { width: 200pt }
.outer { display: flex; aspect-ratio: 4 }
.inner { display: flex; aspect-ratio: 1 }
.box { height: 100%; aspect-ratio: 2 }
`)
	res := layoutHTML(t, `<html><body>
<div class="container">
  <div class="outer">
    <div class="inner">
      <div>
        <div class="box"></div>
      </div>
    </div>
  </div>
</div>
</body></html>`, cssSheet)

	container := findBoxByClass(t, res, "container")
	outer := findBoxByClass(t, res, "outer")
	inner := findBoxByClass(t, res, "inner")

	if !near(container.w, 200) || !near(container.height, 50) {
		t.Fatalf("container size = %.2fx%.2f, want 200x50 and not taller", container.w, container.height)
	}
	if !near(outer.w, 200) || !near(outer.height, 50) {
		t.Fatalf("outer size = %.2fx%.2f, want 200x50 from the 200pt width and ratio 4", outer.w, outer.height)
	}
	if !near(inner.w, 100) || !near(inner.height, 50) {
		t.Fatalf("inner size = %.2fx%.2f, want 100x50 from the definite 50pt cross size and nested ratio",
			inner.w, inner.height)
	}
}

// TestChromeFlexMinimumWidthAspectRatio covers case
// wpt-flex-minimum-width-aspect (chrome-reference), source
// chromium/third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-minimum-width-flex-items-010.html.
//
// Expected Chromium behavior: the automatic main-axis minimum of the img
// uses the transferred aspect-ratio size. The img is 200x200 with
// max-height 100pt, so its transferred minimum width is 100pt * (200/200)
// = 100pt. The 10pt wide flex container must not shrink the item to the
// container width. The width/height attributes stand in for the source's
// support/200x200-green.png intrinsic size; layout tests run without an
// image resolver.
//
//nolint:wsl // fixture assertions keep the expected geometry together
func TestChromeFlexMinimumWidthAspectRatio(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.constrained-flex { display: flex; width: 10pt }
.item { max-height: 100pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="constrained-flex">
  <img class="item" width="200" height="200" alt="">
</div>
</body></html>`, cssSheet)

	flexBox := findBoxByClass(t, res, "constrained-flex")
	item := findBoxByClass(t, res, "item")

	if !near(flexBox.w, 10) {
		t.Fatalf("flex container width = %.2f, want 10", flexBox.w)
	}
	if !near(item.height, 100) {
		t.Fatalf("item height = %.2f, want 100 from max-height", item.height)
	}
	if !near(item.w, 100) {
		t.Fatalf("item width = %.2f, want 100 (transferred minimum from max-height 100 x ratio 200/200)",
			item.w)
	}
}

// TestChromeFlexMinimumWidthInlineSVGAspectRatio is the inline SVG mirror of
// TestChromeFlexMinimumWidthAspectRatio: a 200x200 svg is the direct flex item
// with max-height 100pt, so its used size clamps to 100x100 and its transferred
// minimum main size is 100pt. The 10pt container cannot shrink it further, so
// the item overflows exactly like the img control.
//
//nolint:wsl // fixture assertions keep the expected geometry together
func TestChromeFlexMinimumWidthInlineSVGAspectRatio(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.constrained-flex { display: flex; width: 10pt }
.item { max-height: 100pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="constrained-flex">
  <svg class="item" width="200" height="200"><rect width="200" height="200"/></svg>
</div>
</body></html>`, cssSheet)

	flexBox := findBoxByClass(t, res, "constrained-flex")
	item := findBoxByClass(t, res, "item")

	if !near(flexBox.w, 10) {
		t.Fatalf("flex container width = %.2f, want 10", flexBox.w)
	}
	if !near(item.height, 100) {
		t.Fatalf("svg item height = %.2f, want 100 from max-height", item.height)
	}
	if !near(item.w, 100) {
		t.Fatalf("svg item width = %.2f, want 100 (transferred minimum from max-height 100 x ratio 200/200)",
			item.w)
	}

	for _, imageOp := range res.Ops {
		if imageOp.Kind != OpImage || len(imageOp.Image) == 0 {
			continue
		}

		if !near(imageOp.W, 100) || !near(imageOp.H, 100) {
			t.Fatalf("svg image op = %.2fx%.2f, want 100x100", imageOp.W, imageOp.H)
		}
	}
}
