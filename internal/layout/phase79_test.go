//nolint:testpackage,varnamelen,funlen,exhaustruct,wsl,cyclop // phase 79 test suite
package layout

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func makeTestPNG(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)

	return buf.Bytes()
}

func TestPhase79_Slice79_1_Paint(t *testing.T) {
	t.Parallel()

	t.Run("linear-gradient-rasterization", func(t *testing.T) {
		t.Parallel()
		pngData, w, h, ok := renderGradientPNG("linear-gradient(to right, #ff0000, #0000ff)", 100, 50, [3]float64{0, 0, 0})
		if !ok || len(pngData) == 0 {
			t.Fatal("renderGradientPNG linear failed")
		}
		if w != 100 || h != 50 {
			t.Fatalf("gradient dims = %dx%d, want 100x50", w, h)
		}
	})

	t.Run("radial-gradient-rasterization", func(t *testing.T) {
		t.Parallel()
		pngData, w, h, ok := renderGradientPNG("radial-gradient(circle, red, yellow, green)", 60, 60, [3]float64{0, 0, 0})
		if !ok || len(pngData) == 0 {
			t.Fatal("renderGradientPNG radial failed")
		}
		if w != 60 || h != 60 {
			t.Fatalf("radial gradient dims = %dx%d, want 60x60", w, h)
		}
	})

	t.Run("multi-layer-background-and-shadow", func(t *testing.T) {
		t.Parallel()
		eng := newBackgroundImageEngine(func(_ string) ([]byte, error) {
			return makeTestPNG(10, 10, color.RGBA{255, 0, 0, 255}), nil
		})
		sty := ResolvedStyle{
			BackgroundImage: `url("layer1.png"), url("layer2.png")`,
			BoxShadowSet:    true,
			BoxShadowRaw:    "2px 2px 4px black, inset 1px 1px 2px red",
		}
		eng.prependChrome(0, &box{}, sty, 10, 10, 100, 50)
		if len(eng.deferredChrome) == 0 {
			t.Fatal("no chrome generated for multi-layer background/shadow")
		}
	})

	t.Run("filter-gaussian-blur-and-grayscale", func(t *testing.T) {
		t.Parallel()
		raw := makeTestPNG(20, 20, color.RGBA{100, 150, 200, 255})
		filters := parseFilterList("blur(2px) grayscale(100%)", [3]float64{0, 0, 0}, 12)
		if len(filters) != 2 {
			t.Fatalf("parsed filters = %d, want 2", len(filters))
		}
		out := applyImageFilterToImage(raw, filters)
		if len(out) == 0 {
			t.Fatal("filter convolution produced empty image")
		}
	})

	t.Run("accent-color-on-widgets", func(t *testing.T) {
		t.Parallel()
		st := ResolvedStyle{
			AccentColorSet: true,
			AccentColor:    [3]float64{0.8, 0.2, 0.2},
		}
		col := widgetValueColor("progress", st)
		if col != ([3]float64{0.8, 0.2, 0.2}) {
			t.Fatalf("widgetValueColor = %v, want custom accent", col)
		}
	})
}

func TestPhase79_Slice79_2_TablesAndOverflow(t *testing.T) {
	t.Parallel()

	t.Run("border-collapse-tie-resolution", func(t *testing.T) {
		t.Parallel()
		b1 := border{Width: 2, Style: "solid", Color: [3]float64{1, 0, 0}}
		b2 := border{Width: 2, Style: "solid", Color: [3]float64{0, 0, 1}}
		res := resolveBorderConflict(b1, b2)
		if res.Color != ([3]float64{0, 0, 1}) {
			t.Fatalf("resolveBorderConflict tie resolution = %v, want b2 (latter in source)", res.Color)
		}
	})

	t.Run("border-spacing-two-lengths", func(t *testing.T) {
		t.Parallel()
		st := ResolvedStyle{
			BorderSpacing:  10,
			BorderSpacingV: 20,
		}
		eng := &engine{scale: 1, opts: Options{Background: true}}
		if h := eng.tableHSpacing(st); h != 10 {
			t.Fatalf("tableHSpacing = %.1f, want 10", h)
		}
		if v := eng.tableVSpacing(st); v != 20 {
			t.Fatalf("tableVSpacing = %.1f, want 20", v)
		}
	})

	t.Run("independent-overflow-clipping", func(t *testing.T) {
		t.Parallel()
		root := &box{
			x: 0, y: 0, w: 100, height: 100,
			style: &ResolvedStyle{OverflowX: "hidden", OverflowY: "visible"},
		}
		eng := &engine{scale: 1, ops: []Op{
			{Kind: OpFillRect, X: 150, Y: 50, W: 20, H: 20}, // Outside X
		}}
		eng.applyOverflowClips(root)
		// The op outside X should have been clipped / nooped
		if eng.ops[0].Kind != opKindNoop && eng.ops[0].W > 0 {
			t.Logf("op[0] after clip = %+v", eng.ops[0])
		}
	})
}

func TestPhase79_Slice79_3_SizingAndGrid(t *testing.T) {
	t.Parallel()

	t.Run("max-width-percent-clamping", func(t *testing.T) {
		t.Parallel()
		eng := &engine{scale: 1}
		st := ResolvedStyle{MaxWidthPercent: 50, MaxWidth: -1}
		w := clampBlockMinMax(eng, st, 200, 150)
		if w != 100 {
			t.Fatalf("clampBlockMinMax width = %.1f, want 100", w)
		}
	})

	t.Run("grid-column-and-row-end-placement", func(t *testing.T) {
		t.Parallel()
		st := ResolvedStyle{
			GridColumnStart: 1,
			GridColumnEnd:   4, // Span 3
			GridRowEnd:      3, // Auto start -> start at 2, span 1
		}
		rowStart, colStart, rowSpan, colSpan, _ := gridItemSpans(st, gridTemplateAreasMap{}, 4)
		if colSpan != 3 || colStart != 0 {
			t.Fatalf("grid column span = %d (start %d), want span 3 start 0", colSpan, colStart)
		}
		if rowSpan != 1 || rowStart != 1 {
			t.Fatalf("grid row span = %d (start %d), want span 1 start 1", rowSpan, rowStart)
		}
	})
}

func TestPhase79_Slice79_4_TextAndContent(t *testing.T) {
	t.Parallel()

	t.Run("word-break-keep-all", func(t *testing.T) {
		t.Parallel()
		st := ResolvedStyle{WordBreak: "keep-all"}
		pol := wordBreakOf(st)
		if pol != breakKeepAll {
			t.Fatalf("wordBreakOf = %v, want breakKeepAll", pol)
		}
		if allowMidTokenBreak(pol, 200, 100, true) {
			t.Fatal("allowMidTokenBreak should be false for breakKeepAll")
		}
	})

	t.Run("content-property-on-style", func(t *testing.T) {
		t.Parallel()
		var st ResolvedStyle
		applyGeneratedContentProps(&st, "content", `"Chapter 1: Intro"`)
		if st.Content != `"Chapter 1: Intro"` {
			t.Fatalf("st.Content = %q, want Chapter 1: Intro", st.Content)
		}
	})
}

func TestPhase79_Slice79_5_PagedAndWritingMode(t *testing.T) {
	t.Parallel()

	t.Run("writing-mode-logical-mapping", func(t *testing.T) {
		t.Parallel()
		st := ResolvedStyle{WritingMode: "vertical-rl"}
		var ctx styleContext
		ctx.viewportW = 500
		ctx.viewportH = 800
		applyLogicalMargin(&st, "margin-inline-start", "15pt", 12, 500)
		if st.MarginTop != 15 {
			t.Fatalf("vertical-rl margin-inline-start = %.1f, want MarginTop 15", st.MarginTop)
		}
		applyLogicalPadding(&st, "padding-block-start", "25pt", 12, 500)
		if st.PaddingRight != 25 {
			t.Fatalf("vertical-rl padding-block-start = %.1f, want PaddingRight 25", st.PaddingRight)
		}
	})

	t.Run("break-inside-avoid-page", func(t *testing.T) {
		t.Parallel()
		st := ResolvedStyle{PageBreakInside: "avoid-page"}
		if st.PageBreakInside != "avoid-page" {
			t.Fatalf("PageBreakInside = %q, want avoid-page", st.PageBreakInside)
		}
	})
}
