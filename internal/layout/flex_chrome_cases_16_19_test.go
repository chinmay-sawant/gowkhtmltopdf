//nolint:wsl // direct fixture assertions keep the geometry visible
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestChromeFlexCase16AlignItemsStretch covers the source-faithful WPT input
// for align-items: stretch. The two un-sized spans use the full line cross
// size, and the first span keeps its 6em top margin.
func TestChromeFlexCase16AlignItemsStretch(t *testing.T) {
	t.Parallel()

	src := readChromeCase(t, "case-16-wpt-align-items-stretch.html")
	res := layoutChromeCase(t, src)
	container := findBox(t, res, "div")
	spans := chromeFlexElementBoxes1619(t, res, "span")
	assertChromeFlexCase16Geometry(t, container, spans)

	t.Run("margin-sensitive-stretch", func(t *testing.T) {
		t.Parallel()
		first := spans[0]
		if !near(first.w, 96) || !near(first.height, 0) {
			t.Errorf("margin span size = %.2fx%.2f, want %.2fx0", first.w, first.height, 96.0)
		}
		if !near(first.y, container.y+72) {
			t.Errorf("margin span y = %.2f, want %.2f", first.y, container.y+72)
		}
	})
}

func assertChromeFlexCase16Geometry(t *testing.T, container *box, spans []*box) {
	t.Helper()

	if len(spans) != 3 {
		t.Fatalf("span boxes = %d, want 3", len(spans))
	}

	const (
		containerWidth  = 360.0
		containerHeight = 72.0
		itemWidth       = 96.0
	)
	if !near(container.w, containerWidth) || !near(container.height, containerHeight) {
		t.Fatalf(
			"stretch container = %.2fx%.2f, want %.2fx%.2f",
			container.w, container.height, containerWidth, containerHeight,
		)
	}

	for index, span := range spans[1:] {
		if !near(span.w, itemWidth) || !near(span.height, containerHeight) {
			t.Errorf(
				"stretch span %d size = %.2fx%.2f, want %.2fx%.2f",
				index+2, span.w, span.height, itemWidth, containerHeight,
			)
		}
		if !near(span.y, container.y) {
			t.Errorf("stretch span %d y = %.2f, want container y %.2f", index+2, span.y, container.y)
		}
	}

	if !near(spans[1].x, container.x+itemWidth) || !near(spans[2].x, container.x+2*itemWidth) {
		t.Errorf(
			"stretch span x positions = %.2f/%.2f, want %.2f/%.2f",
			spans[1].x, spans[2].x, container.x+itemWidth, container.x+2*itemWidth,
		)
	}
}

// TestChromeFlexCase19MarginAuto covers the source-faithful WPT input for row
// main-axis and cross-axis auto margins.
func TestChromeFlexCase19MarginAuto(t *testing.T) {
	t.Parallel()

	src := readChromeCase(t, "case-19-wpt-flexbox-margin-auto.html")
	res := layoutChromeCase(t, src)
	container := findBox(t, res, "div")
	spans := chromeFlexElementBoxes1619(t, res, "span")
	if len(spans) != 2 {
		t.Fatalf("span boxes = %d, want 2", len(spans))
	}

	const itemWidth = 48.0
	t.Run("main-axis-auto-margins", func(t *testing.T) {
		t.Parallel()
		for index, span := range spans {
			if !near(span.w, itemWidth) {
				t.Errorf("margin span %d width = %.2f, want %.2f", index+1, span.w, itemWidth)
			}
		}

		if !near(spans[1].x-spans[0].x, 4*itemWidth) {
			t.Errorf("margin span separation = %.2f, want %.2f", spans[1].x-spans[0].x, 4*itemWidth)
		}

		groupCenter := (spans[0].x + spans[1].x + spans[1].w) / 2
		containerCenter := container.x + container.w/2
		if !near(groupCenter, containerCenter) {
			t.Errorf("auto-margin group center = %.2f, want container center %.2f", groupCenter, containerCenter)
		}
	})

	t.Run("cross-axis-fixed-margins", func(t *testing.T) {
		t.Parallel()
		const (
			emPixels   = 12.0
			itemHeight = 72.0
		)
		wantY := container.y + container.style.BorderTop.Width + emPixels
		for index, span := range spans {
			if !near(span.height, itemHeight) || !near(span.y, wantY) {
				t.Errorf(
					"margin span %d geometry = (%.2f, %.2f), want y=%.2f h=%.2f",
					index+1, span.y, span.height, wantY, itemHeight,
				)
			}
		}
	})
}

func chromeFlexElementBoxes1619(t *testing.T, res *Result, name string) []*box {
	t.Helper()

	var found []*box
	var walk func(*box)
	walk = func(node *box) {
		if node == nil {
			return
		}
		if node.node != nil && node.node.Type == html.ElementNode && node.node.Name == name {
			found = append(found, node)
		}
		for _, child := range node.children {
			walk(child)
		}
	}
	walk(res.root)

	return found
}
