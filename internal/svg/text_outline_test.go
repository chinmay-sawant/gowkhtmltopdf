package svg

import (
	"bytes"
	"fmt"
	"testing"
)

// countDarkInk counts pixels dark enough to be text ink. Bold strokes add ink
// and italic slants move it, so the wordmark raster can be compared against an
// unstyled control without decoding glyph outlines.
func countDarkInk(t *testing.T, pngBytes []byte) int {
	t.Helper()

	img := decodeRaster(t, pngBytes)
	bounds := img.Bounds()
	dark := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a > 0 && r>>8 < 128 && g>>8 < 128 && b>>8 < 128 {
				dark++
			}
		}
	}

	return dark
}

// countVisiblePixels counts pixels with any coverage. White-on-transparent
// rasters (the live wordmark) have no dark pixels but still paint.
func countVisiblePixels(t *testing.T, pngBytes []byte) int {
	t.Helper()

	img := decodeRaster(t, pngBytes)
	bounds := img.Bounds()
	visible := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				visible++
			}
		}
	}

	return visible
}

// wordmarkSVGTemplate mirrors cplusplus.com's inline wordmark: viewBox-only
// root, Roboto with a sans-serif fallback, 22px text.
const wordmarkSVGTemplate = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 36">` +
	`<text x="0" y="22" font-family="Roboto,sans-serif" font-size="22"%s>cplusplus</text></svg>`

// TestRasterizeKeepsBoldItalicTextStyle is the cplusplus-tutorial-9
// regression: tdewolff/canvas's SVG parser ignores font-style/font-weight, so
// styled text used to rasterize as upright regular. Bold italic must add ink.
func TestRasterizeKeepsBoldItalicTextStyle(t *testing.T) {
	t.Parallel()

	if !fontFamilyResolves("sans-serif") {
		t.Skip("no system fonts available for canvas font loading")
	}

	plainPNG, _, _, err := Rasterize([]byte(fmt.Sprintf(wordmarkSVGTemplate, "")), 256)
	if err != nil {
		t.Fatal(err)
	}

	styledPNG, _, _, err := Rasterize(
		[]byte(fmt.Sprintf(wordmarkSVGTemplate, ` font-style="italic" font-weight="bold"`)), 256)
	if err != nil {
		t.Fatal(err)
	}

	plainInk := countDarkInk(t, plainPNG)
	styledInk := countDarkInk(t, styledPNG)

	if plainInk == 0 {
		t.Fatal("control raster painted no ink")
	}

	// True bold italic and faux bold both add roughly 20 percent more ink.
	if styledInk <= plainInk*6/5 {
		t.Fatalf("bold italic ink = %d, want > 1.2x regular ink %d (font style dropped in the rasterizer)",
			styledInk, plainInk)
	}
}

// TestRasterizeKeepsBoldItalicStyleOnFallbackFamily covers the live family
// list "Roboto,arial", where neither family is installed and
// withFontFallbacks appends ",sans-serif". The style must survive the fallback.
func TestRasterizeKeepsBoldItalicStyleOnFallbackFamily(t *testing.T) {
	t.Parallel()

	if !fontFamilyResolves("sans-serif") {
		t.Skip("no system fonts available for canvas font loading")
	}

	const fallbackTemplate = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 36">` +
		`<text x="0" y="22" font-family="Roboto,arial" font-size="22"%s>.com</text></svg>`

	plainPNG, _, _, err := Rasterize([]byte(fmt.Sprintf(fallbackTemplate, "")), 256)
	if err != nil {
		t.Fatal(err)
	}

	styledPNG, _, _, err := Rasterize(
		[]byte(fmt.Sprintf(fallbackTemplate, ` font-style="italic" font-weight="bold"`)), 256)
	if err != nil {
		t.Fatal(err)
	}

	plainInk := countDarkInk(t, plainPNG)
	styledInk := countDarkInk(t, styledPNG)

	if plainInk == 0 {
		t.Fatal("control raster painted no ink")
	}

	if styledInk <= plainInk {
		t.Fatalf("fallback bold italic ink = %d, want more than regular ink %d", styledInk, plainInk)
	}
}

// TestRasterizeSynthesizesStyleWhenFamilyLacksBoldItalic covers the second
// branch of face selection: a family installed with a regular face only must
// still get canvas's faux bold and italic instead of upright regular.
func TestRasterizeSynthesizesStyleWhenFamilyLacksBoldItalic(t *testing.T) {
	t.Parallel()

	if !fontFamilyResolves("Cousine") {
		t.Skip("Cousine fixture font not installed")
	}

	const template = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 36">` +
		`<text x="0" y="22" font-family="Cousine" font-size="22"%s>wordmark</text></svg>`

	plainPNG, _, _, err := Rasterize([]byte(fmt.Sprintf(template, "")), 256)
	if err != nil {
		t.Fatal(err)
	}

	styledPNG, _, _, err := Rasterize(
		[]byte(fmt.Sprintf(template, ` font-style="italic" font-weight="bold"`)), 256)
	if err != nil {
		t.Fatal(err)
	}

	plainInk := countDarkInk(t, plainPNG)
	styledInk := countDarkInk(t, styledPNG)

	if plainInk == 0 {
		t.Fatal("control raster painted no ink")
	}

	if styledInk <= plainInk {
		t.Fatalf("faux bold italic ink = %d, want more than regular ink %d", styledInk, plainInk)
	}
}

// TestPrepareCanvasInputOutlinesStyledTextOnly is the scope guard: plain text
// must stay byte-for-byte untouched (canvas renders it), and only text with a
// non-regular weight or style may be rewritten to an outline path.
func TestPrepareCanvasInputOutlinesStyledTextOnly(t *testing.T) {
	t.Parallel()

	if !fontFamilyResolves("sans-serif") {
		t.Skip("no system fonts available for canvas font loading")
	}

	regular := []byte(fmt.Sprintf(wordmarkSVGTemplate, ""))
	if got := prepareCanvasInput(regular); !bytes.Equal(got, regular) {
		t.Fatalf("unstyled text was rewritten:\n%s", got)
	}

	styled := []byte(fmt.Sprintf(wordmarkSVGTemplate, ` font-style="italic" font-weight="bold"`))

	got := prepareCanvasInput(styled)
	if bytes.Contains(got, []byte("<text")) {
		t.Fatalf("styled <text> survived preprocessing:\n%s", got)
	}

	if !bytes.Contains(got, []byte("<path")) {
		t.Fatalf("styled <text> was not outlined:\n%s", got)
	}
}

// TestPrepareCanvasInputInheritsStyledTextFromGroup pins the tree walk: a
// font-weight or font-style set on an ancestor <g> must reach its <text>.
func TestPrepareCanvasInputInheritsStyledTextFromGroup(t *testing.T) {
	t.Parallel()

	if !fontFamilyResolves("sans-serif") {
		t.Skip("no system fonts available for canvas font loading")
	}

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 36">` +
		`<g font-family="Roboto,sans-serif" font-size="22" font-style="italic" font-weight="bold">` +
		`<text x="0" y="22">cplusplus</text></g></svg>`)

	got := prepareCanvasInput(src)
	if bytes.Contains(got, []byte("<text")) {
		t.Fatalf("inherited styled <text> survived preprocessing:\n%s", got)
	}

	if !bytes.Contains(got, []byte("<path")) {
		t.Fatalf("inherited styled <text> was not outlined:\n%s", got)
	}
}

// TestPrepareCanvasInputOutlinesLiveWordmarkMarkup runs the exact markup from
// the cplusplus.com header (single-quoted attributes, inline style, textLength
// on both text nodes, a family list that needs the fallback) through the
// preprocessing entry.
func TestPrepareCanvasInputOutlinesLiveWordmarkMarkup(t *testing.T) {
	t.Parallel()

	if !fontFamilyResolves("sans-serif") {
		t.Skip("no system fonts available for canvas font loading")
	}

	live := []byte(`<svg xmlns='http://www.w3.org/2000/svg' style='fill:#fff' viewBox='0 0 120 36'>` +
		`<text x='0' y='22' textLength='120' lengthAdjust='spacingAndGlyphs' font-family='Roboto,sans-serif' ` +
		`font-size='22px' font-style='italic' font-weight='bold' style='fill:#fff'>cplusplus</text>` +
		`<text id='tld' x='80' y='34' textLength='38' lengthAdjust='spacingAndGlyphs' font-family='Roboto,arial' ` +
		`font-size='12px' font-style='italic' font-weight='bold' style='fill:#cde'>.com</text></svg>`)

	got := prepareCanvasInput(live)
	if bytes.Contains(got, []byte("<text")) {
		t.Fatalf("styled <text> survived preprocessing:\n%s", got)
	}

	if count := bytes.Count(got, []byte("<path")); count != 2 {
		t.Fatalf("outlined paths = %d, want 2:\n%s", count, got)
	}

	// The inline fill styles must survive onto the outline paths.
	if !bytes.Contains(got, []byte(`style='fill:#fff'`)) || !bytes.Contains(got, []byte(`style='fill:#cde'`)) {
		t.Fatalf("inline fill styles were dropped:\n%s", got)
	}

	pngBytes, _, _, err := Rasterize(live, 256)
	if err != nil {
		t.Fatal(err)
	}

	if countVisiblePixels(t, pngBytes) == 0 {
		t.Fatal("live wordmark raster has no ink")
	}
}
