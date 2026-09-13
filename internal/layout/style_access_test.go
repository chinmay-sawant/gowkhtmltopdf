package layout

import "testing"

// TestStyleOverrideReachesFlexMeasurement proves an engine-local override is
// visible to flex measurement readers. flexMinMainSize used to read the
// immutable cascade map directly, so a buildWithStyle override (grid stretch,
// flex grow/shrink) changed the built box but not the measured floor.
func TestStyleOverrideReachesFlexMeasurement(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="item">x</div></body></html>`)
	styles := resolveStyles(root, nil, "print", testViewport, 800)
	item := findElementByName(root, "div")

	stored := styles[item]
	if stored == nil {
		t.Fatal("item has no stored style")
	}

	if stored.MinWidthSet {
		t.Fatalf("stored min-width unexpectedly set: %+v", stored.MinWidth)
	}

	override := initialStyle()
	override.MinWidth = 50
	override.MinWidthSet = true
	override.MinWidthPercent = -1

	eng := &engine{
		scale:  1,
		opts:   Options{Width: testViewport},
		styles: styles,
	}
	eng.styleOverrides = append(eng.styleOverrides, styleOverride{node: item, style: &override})

	if got := eng.flexMinMainSize(flexMeas{n: item}, 200); !near(got, 50) {
		t.Fatalf("overridden min-width floor = %.3f, want 50", got)
	}
}

var resolvedStyleAccessSink float64 //nolint:gochecknoglobals // benchmark sink

// BenchmarkResolvedStyleAccess measures the cost of copying the full resolved
// style record at a stage boundary. The result controls whether a narrower
// stage view is worth introducing.
func BenchmarkResolvedStyleAccess(b *testing.B) {
	style := initialStyle()

	b.ReportAllocs()

	b.Run("pointer", func(b *testing.B) {
		for range b.N {
			resolvedStyleAccessSink += style.Width + style.FontSize
		}
	})

	b.Run("value-copy", func(b *testing.B) {
		for range b.N {
			snapshot := style
			resolvedStyleAccessSink += snapshot.Width + snapshot.FontSize
		}
	})
}
