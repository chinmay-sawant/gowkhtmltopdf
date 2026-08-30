//nolint:testpackage // tests exercise unexported shorthand expansion
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func classStyle(t *testing.T, decl string) *ResolvedStyle {
	t.Helper()

	return classStyleViewport(t, decl, testViewport, 800)
}

func classStyleViewport(t *testing.T, decl string, vw, vh float64) *ResolvedStyle {
	t.Helper()

	root := mustParse(t, `<html><body><div class="x">x</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, ".x { "+decl+" }")}, "print", vw, vh)

	return styleByClass(t, styles, "x")
}

func TestFlexFlowShorthand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, decl, dir, wrap string
	}{
		{"both", "flex-flow: column wrap", fxCol, "wrap"},
		{"dir-only", "flex-flow: row-reverse", "row-reverse", "nowrap"},
		{"wrap-only", "flex-flow: wrap", fxRow, "wrap"},
		{"wrap-reverse", "flex-flow: wrap-reverse", fxRow, fxWrapRev},
		{"column-reverse", "flex-flow: column-reverse wrap", fxColRev, "wrap"},
		{"token-order", "flex-flow: wrap column", fxCol, "wrap"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sty := classStyle(t, testCase.decl)
			if sty.FlexDirection != testCase.dir || sty.FlexWrap != testCase.wrap {
				t.Fatalf("%s → dir=%q wrap=%q, want %q %q",
					testCase.decl, sty.FlexDirection, sty.FlexWrap, testCase.dir, testCase.wrap)
			}
		})
	}
}

func TestPlaceShorthands(t *testing.T) { //nolint:cyclop // one test covers all three place-* shorthands
	t.Parallel()

	twoTok := classStyle(t, "place-content: center space-between; place-items: start end; place-self: flex-end center")
	if twoTok.AlignContent != fxCenter || twoTok.JustifyContent != fxBetween {
		t.Fatalf("place-content two tokens → align=%q justify=%q, want center space-between",
			twoTok.AlignContent, twoTok.JustifyContent)
	}

	if twoTok.AlignItems != fxStart || twoTok.JustifyItems != fxEnd {
		t.Fatalf("place-items two tokens → align=%q justify=%q, want start end",
			twoTok.AlignItems, twoTok.JustifyItems)
	}

	if twoTok.AlignSelf != fxFlexEnd || twoTok.JustifySelf != fxCenter {
		t.Fatalf("place-self two tokens → align=%q justify=%q, want flex-end center",
			twoTok.AlignSelf, twoTok.JustifySelf)
	}

	oneTok := classStyle(t, "place-content: end; place-items: stretch; place-self: center")
	if oneTok.AlignContent != fxEnd || oneTok.JustifyContent != fxEnd {
		t.Fatalf("place-content one token → align=%q justify=%q, want align=end justify=end",
			oneTok.AlignContent, oneTok.JustifyContent)
	}

	if oneTok.AlignItems != fxStretch || oneTok.JustifyItems != fxStretch {
		t.Fatalf("place-items one token → align=%q justify=%q, want align=stretch justify=stretch",
			oneTok.AlignItems, oneTok.JustifyItems)
	}

	if oneTok.AlignSelf != fxCenter || oneTok.JustifySelf != fxCenter {
		t.Fatalf("place-self one token → align=%q justify=%q, want align=center justify=center",
			oneTok.AlignSelf, oneTok.JustifySelf)
	}
}

func TestGridTemplateShorthand(t *testing.T) { //nolint:cyclop,funlen // tracks, areas, none, auto-flow, masonry skip
	t.Parallel()

	tracks := classStyle(t, "grid-template: 100px 1fr / 50px 2fr")
	if tracks.GridTemplateRows != "100px 1fr" || tracks.GridTemplateColumns != "50px 2fr" {
		t.Fatalf("grid-template tracks → rows=%q cols=%q",
			tracks.GridTemplateRows, tracks.GridTemplateColumns)
	}

	if tracks.GridTemplateAreas != "" {
		t.Fatalf("grid-template tracks left areas %q", tracks.GridTemplateAreas)
	}

	areas := classStyle(t, `grid-template: "a a" 40px "b c" 1fr / 1fr 2fr`)
	if !strings.Contains(areas.GridTemplateAreas, `"a a"`) ||
		!strings.Contains(areas.GridTemplateAreas, `"b c"`) {
		t.Fatalf("grid-template areas = %q, want quoted a/b rows", areas.GridTemplateAreas)
	}

	if areas.GridTemplateRows != "40px 1fr" || areas.GridTemplateColumns != "1fr 2fr" {
		t.Fatalf("grid-template area tracks → rows=%q cols=%q, want 40px 1fr / 1fr 2fr",
			areas.GridTemplateRows, areas.GridTemplateColumns)
	}

	cleared := initialStyle()
	cleared.GridTemplateRows = "1fr"
	cleared.GridTemplateColumns = "2fr"
	cleared.GridTemplateAreas = `"a"`
	parseGridTemplateShorthand(&cleared, cssDisplayNone)

	if cleared.GridTemplateRows != "" || cleared.GridTemplateColumns != "" ||
		cleared.GridTemplateAreas != "" {
		t.Fatalf("grid-template:none → rows=%q cols=%q areas=%q",
			cleared.GridTemplateRows, cleared.GridTemplateColumns, cleared.GridTemplateAreas)
	}

	gridSty := classStyle(t, "grid: 80px / "+twoFrTracks)
	if gridSty.GridTemplateRows != "80px" || gridSty.GridTemplateColumns != twoFrTracks {
		t.Fatalf("grid tracks → rows=%q cols=%q", gridSty.GridTemplateRows, gridSty.GridTemplateColumns)
	}

	if gridSty.GridAutoFlow != fxRow {
		t.Fatalf("grid tracks auto-flow = %q, want row", gridSty.GridAutoFlow)
	}

	flow := classStyle(t, "grid: auto-flow dense / 1fr 2fr")
	if flow.GridAutoFlow != gridFlowDense {
		t.Fatalf("grid auto-flow dense → %q, want dense", flow.GridAutoFlow)
	}

	if flow.GridTemplateColumns != "1fr 2fr" || flow.GridTemplateRows != "" {
		t.Fatalf("grid auto-flow dense → rows=%q cols=%q, want empty / 1fr 2fr",
			flow.GridTemplateRows, flow.GridTemplateColumns)
	}

	colFlow := classStyle(t, "grid: 100px / auto-flow dense")
	if colFlow.GridAutoFlow != gridFlowColumnDense {
		t.Fatalf("grid column auto-flow → %q, want column dense", colFlow.GridAutoFlow)
	}

	if colFlow.GridTemplateRows != "100px" {
		t.Fatalf("grid column auto-flow rows = %q, want 100px", colFlow.GridTemplateRows)
	}

	masonry := initialStyle()
	masonry.GridTemplateColumns = twoFrTracks
	parseGridTemplateShorthand(&masonry, "masonry / 1fr")

	if masonry.GridTemplateColumns != twoFrTracks || masonry.GridTemplateRows != "" {
		t.Fatalf("grid-template masonry should skip, rows=%q cols=%q",
			masonry.GridTemplateRows, masonry.GridTemplateColumns)
	}

	masonryGrid := initialStyle()
	masonryGrid.GridAutoFlow = fxCol
	parseGridShorthand(&masonryGrid, "masonry")

	if masonryGrid.GridAutoFlow != fxCol {
		t.Fatalf("grid masonry should skip, flow=%q", masonryGrid.GridAutoFlow)
	}
}

func TestVminVmax(t *testing.T) {
	t.Parallel()

	const vw, vh = 400.0, 800.0

	box := classStyleViewport(t, "width: 10vmin; height: 10vmax; margin-left: 5vmin", vw, vh)
	if !near(box.Width, 40) {
		t.Fatalf("width 10vmin = %.2f, want 40 (10%% of min(400,800))", box.Width)
	}

	if !near(box.Height, 80) {
		t.Fatalf("height 10vmax = %.2f, want 80 (10%% of max(400,800))", box.Height)
	}

	if !near(box.MarginLeft, 20) {
		t.Fatalf("margin-left 5vmin = %.2f, want 20", box.MarginLeft)
	}

	got, ok := lengthBox("20vmin", defaultFontSizePt, 500, overflowAuto)
	if !ok || !near(got, 100) {
		t.Fatalf("lengthBox 20vmin against 500 = (%v, %v), want 100", got, ok)
	}

	if got := marginLen("10vmax", defaultFontSizePt, 200); !near(got, 20) {
		t.Fatalf("marginLen 10vmax against 200 = %.2f, want 20", got)
	}
}
