package layout

import "testing"

type case14Rect struct {
	x, y, w, h float64
}

func TestChromeFlexCase14Geometry(t *testing.T) {
	t.Parallel()

	src := readChromeCase(t, "case-14-legacy-flex-align-baseline.html")
	res := layoutChromeCase(t, src)

	// Chromium reference geometry converted from CSS px to layout points.
	wants := map[string][2]case14Rect{
		"horizontal-ltr-row": {
			{x: 0, y: 15, w: 37.5, h: 82.5},
			{x: 37.5, y: 15, w: 37.5, h: 82.5},
		},
		"horizontal-rtl-column": {
			{x: -7.5, y: 0, w: 82.5, h: 30},
			{x: -7.5, y: 45, w: 82.5, h: 30},
		},
		"vertical-lr-ltr-row": {
			{x: 0, y: 0, w: 82.5, h: 30},
			{x: 0, y: 45, w: 82.5, h: 30},
		},
		"vertical-lr-rtl-column": {
			{x: 0, y: 15, w: 37.5, h: 82.5},
			{x: 37.5, y: 15, w: 37.5, h: 82.5},
		},
		"vertical-rl-ltr-row-reverse": {
			{x: -7.5, y: 15, w: 82.5, h: 30},
			{x: -7.5, y: 45, w: 82.5, h: 30},
		},
		"vertical-rl-rtl-column-reverse": {
			{x: 0, y: 15, w: 37.5, h: 82.5},
			{x: 37.5, y: 15, w: 37.5, h: 82.5},
		},
	}

	for dataCase, want := range wants {
		container := findBoxByDataCase(t, res, dataCase)
		children := directElementChildren(container)

		if len(children) != len(want) {
			t.Fatalf("case 14 %s children = %d, want %d", dataCase, len(children), len(want))
		}

		for idx, child := range children {
			got := case14Rect{
				x: child.x - container.x,
				y: child.y - container.y,
				w: child.w,
				h: child.height,
			}
			if !near(got.x, want[idx].x) || !near(got.y, want[idx].y) ||
				!near(got.w, want[idx].w) || !near(got.h, want[idx].h) {
				t.Errorf("case 14 %s child %d = (%.2f, %.2f, %.2f, %.2f), want (%.2f, %.2f, %.2f, %.2f)",
					dataCase, idx, got.x, got.y, got.w, got.h,
					want[idx].x, want[idx].y, want[idx].w, want[idx].h)
			}
		}
	}
}

func findBoxByDataCase(t *testing.T, res *Result, dataCase string) *box {
	t.Helper()

	var found *box

	var walk func(*box)
	walk = func(boxNode *box) {
		if boxNode == nil || found != nil {
			return
		}

		if boxNode.node != nil && boxNode.node.Attribute("data-case") == dataCase {
			found = boxNode

			return
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}
	walk(res.root)

	if found == nil {
		t.Fatalf("data-case %q not found", dataCase)
	}

	return found
}

func directElementChildren(parent *box) []*box {
	children := make([]*box, 0, len(parent.children))
	for _, child := range parent.children {
		if child.node != nil && child.node.Parent == parent.node {
			children = append(children, child)
		}
	}

	return children
}
