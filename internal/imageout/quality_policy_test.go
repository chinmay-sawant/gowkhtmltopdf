package imageout

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// quality_policy_test.go pins the raster quality contract IMG-02 must keep
// when large canvases paint directly at final resolution instead of through
// rasterSS supersampling plus downscaleBox. Every case samples decoded pixels
// or geometry: flat regions must match channel values, thin and rounded edges
// must stay continuous, transforms must land where the matrix says, and
// transparent output must stay transparent through a PNG round-trip.
//
// The synthetic fixtures live in the test code next to the expected values, so
// the checked geometry and the probe points cannot drift apart silently.

// qualityOpaqueWhite is the canvas color rasterizeContext starts from when
// transparent is false.
func qualityOpaqueWhite() color.NRGBA {
	return color.NRGBA{R: channelMax, G: channelMax, B: channelMax, A: opaqueAlpha}
}

// qualityWhiteCanvas returns a width x height NRGBA canvas filled opaque white.
func qualityWhiteCanvas(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	drawSolid(img, qualityOpaqueWhite())

	return img
}

// qualityLuma averages the RGB channels; alpha is ignored because every
// sampled region is opaque.
func qualityLuma(c color.NRGBA) int {
	return (int(c.R) + int(c.G) + int(c.B)) / 3
}

// qualityCompositeOver composites src over dst with straight 8-bit alpha,
// mirroring image/draw's Over math closely enough for a +-2 channel tolerance.
func qualityCompositeOver(dst, src color.NRGBA) color.NRGBA {
	alpha := uint32(src.A)
	inverse := uint32(channelMax) - alpha

	return color.NRGBA{
		R: clampChannel((uint32(src.R)*alpha + uint32(dst.R)*inverse) / channelMax),
		G: clampChannel((uint32(src.G)*alpha + uint32(dst.G)*inverse) / channelMax),
		B: clampChannel((uint32(src.B)*alpha + uint32(dst.B)*inverse) / channelMax),
		A: clampChannel(alpha + uint32(dst.A)*inverse/channelMax),
	}
}

// clampChannel narrows a computed 0..255 channel, capping a rounding overshoot
// instead of wrapping it, so the uint32 to uint8 conversion cannot overflow.
func clampChannel(value uint32) uint8 {
	if value > channelMax {
		return channelMax
	}

	return uint8(value)
}

// channelDelta returns the absolute difference between two channel bytes.
func channelDelta(a, b uint8) int {
	if a > b {
		return int(a) - int(b)
	}

	return int(b) - int(a)
}

// requireChannelsNear fails when any channel of got differs from want by more
// than tolerance.
func requireChannelsNear(t *testing.T, label string, got, want color.NRGBA, tolerance int) {
	t.Helper()

	if channelDelta(got.R, want.R) > tolerance || channelDelta(got.G, want.G) > tolerance ||
		channelDelta(got.B, want.B) > tolerance || channelDelta(got.A, want.A) > tolerance {
		t.Errorf("%s = %v, want within %d of %v", label, got, tolerance, want)
	}
}

// TestSmallTextBaselineStableAtEightPx renders an 8px text run through the
// shipped 2x-then-downscale path. Every non-descender letter must put its
// lowest inked row within 1px of the run baseline and the run must carry real
// ink, so a broken supersample or downscale that shifts or thins the glyphs
// fails instead of passing on a washed-out rendering.
func TestSmallTextBaselineStableAtEightPx(t *testing.T) {
	t.Parallel()

	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	const (
		sample      = "Hamburgevons"
		sizePt      = 6.0  // 8 CSS px at the layout 0.75pt-per-px ratio
		baselinePt  = 19.5 // pixel row 26
		startXPt    = 6.0  // pixel column 8
		canvasWPt   = 150  // 200 px
		canvasHPt   = 30   // 40 px
		inkLumaMax  = 128
		descenders  = "gypqj"
		minInkCount = 60
		minLetters  = 6
	)

	res := &layout.Result{
		Width:  canvasWPt,
		Height: canvasHPt,
		Ops: []layout.Op{{
			Kind: layout.OpText, X: startXPt, Y: baselinePt, Text: sample, Size: sizePt,
			R: 0, G: 0, B: 0, Alpha: 1, Font: face,
		}},
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if got := img.Bounds(); got.Dx() != 200 || got.Dy() != 40 {
		t.Fatalf("canvas = %v, want 200x40", got)
	}

	baselinePx := baselinePt * ptToPx

	bottoms, inkCount := measureGlyphBottoms(
		t, img, sample, descenders, startXPt,
		func(runeVal rune) float64 { return face.AdvanceInPoints(runeVal, sizePt) * ptToPx },
		inkLumaMax,
	)

	requireBaselineStability(t, bottoms, baselinePx, minLetters)

	if inkCount < minInkCount {
		t.Errorf("ink pixels = %d, want >= %d (text missing or washed out)", inkCount, minInkCount)
	}
}

// measureGlyphBottoms returns the lowest inked row of every non-descender
// letter in sample and the total count of pixels darker than inkLumaMax. It
// fails the test when a letter paints no ink.
func measureGlyphBottoms(
	t *testing.T, img *image.NRGBA, sample, descenders string, startXPt float64,
	advanceFor func(rune) float64, inkLumaMax int,
) ([]int, int) {
	t.Helper()

	penX := startXPt * ptToPx

	var bottoms []int

	inkCount := 0

	for _, runeVal := range sample {
		advance := advanceFor(runeVal)
		leftX := int(math.Floor(penX))
		rightX := int(math.Ceil(penX + advance))
		bottom := -1

		for rowY := range img.Bounds().Dy() {
			for x := leftX; x < rightX; x++ {
				if qualityLuma(img.NRGBAAt(x, rowY)) < inkLumaMax {
					inkCount++

					if rowY > bottom {
						bottom = rowY
					}
				}
			}
		}

		if bottom < 0 {
			t.Fatalf("letter %q painted no ink in columns [%d,%d)", runeVal, leftX, rightX)
		}

		if !strings.ContainsRune(descenders, runeVal) {
			bottoms = append(bottoms, bottom)
		}

		penX += advance
	}

	return bottoms, inkCount
}

// requireBaselineStability checks that the measured bottoms have the expected
// letter count, do not jitter by more than 1px, and sit on the baseline.
func requireBaselineStability(t *testing.T, bottoms []int, baselinePx float64, minLetters int) {
	t.Helper()

	if len(bottoms) < minLetters {
		t.Fatalf("measured %d non-descender bottoms, want at least %d", len(bottoms), minLetters)
	}

	minBottom, maxBottom := bottoms[0], bottoms[0]

	for _, bottom := range bottoms[1:] {
		minBottom = min(minBottom, bottom)
		maxBottom = max(maxBottom, bottom)
	}

	if maxBottom-minBottom > 1 {
		t.Errorf("small-text baseline jitter: bottoms span %d px at 8px (want <= 1): %v",
			maxBottom-minBottom, bottoms)
	}

	// An 8px glyph sits on the baseline; its lowest inked row is baseline-1
	// plus at most one antialiased row. A wrong scale moves the whole run.
	if float64(minBottom) < baselinePx-2 || float64(maxBottom) > baselinePx {
		t.Errorf("baseline rows %d..%d, want within [%.0f, %.0f]",
			minBottom, maxBottom, baselinePx-2, baselinePx)
	}
}

// thinBorderCanvasW and the thinBorder* constants are the canvas and edge
// geometry of the thin-border continuity test.
const (
	thinBorderCanvasW  = 140
	thinBorderCanvasH  = 100
	thinBorderBoxXpt   = 15.0
	thinBorderBoxYpt   = 15.0
	thinBorderBoxWpt   = 75.0
	thinBorderBoxHpt   = 45.0
	thinBorderEdgeTop  = 20
	thinBorderEdgeLeft = 20
	thinBorderEdgeW    = 100
	thinBorderEdgeH    = 60
	thinBorderMidX     = 70
)

// TestThinBordersPaintContinuousEdgeAtFinalResolution pins 1px and 1.5pt
// rectangle borders to exact border-colored runs at the expected final rows
// and columns. A downscale that drops or averages a row shifts the run into
// the background and fails on the first mismatching pixel.
func TestThinBordersPaintContinuousEdgeAtFinalResolution(t *testing.T) {
	t.Parallel()

	border := color.NRGBA{R: 25, G: 51, B: 102, A: opaqueAlpha}

	tests := []struct {
		name    string
		widthPt float64
		thick   int // final pixels
	}{
		{name: "1px", widthPt: 0.75, thick: 1},
		{name: "1.5pt", widthPt: 1.5, thick: 2},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertThinBorderCase(t, testCase.widthPt, testCase.thick, border)
		})
	}
}

// assertThinBorderCase renders one stroke width and requires the border run to
// cover exactly the expected final rows and columns.
func assertThinBorderCase(t *testing.T, widthPt float64, thick int, border color.NRGBA) {
	t.Helper()

	res := &layout.Result{
		Width:  float64(thinBorderCanvasW) * cssPxToPt,
		Height: float64(thinBorderCanvasH) * cssPxToPt,
		Ops: []layout.Op{{
			Kind: layout.OpStrokeRect, X: thinBorderBoxXpt, Y: thinBorderBoxYpt,
			W: thinBorderBoxWpt, H: thinBorderBoxHpt,
			R: 0.1, G: 0.2, B: 0.4, Width: widthPt,
		}},
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	requireImageBounds(t, img, thinBorderCanvasW, thinBorderCanvasH)

	lastRow := thinBorderEdgeTop + thick - 1
	lastCol := thinBorderEdgeLeft + thick - 1

	// The rows just outside the stroke stay background.
	if got := img.NRGBAAt(thinBorderMidX, thinBorderEdgeTop-1); got != qualityOpaqueWhite() {
		t.Errorf("row above top edge = %v, want white", got)
	}

	if got := img.NRGBAAt(thinBorderMidX, lastRow+1); got != qualityOpaqueWhite() {
		t.Errorf("row below top edge = %v, want white", got)
	}

	assertBorderRectRun(t, img,
		image.Rect(thinBorderEdgeLeft, thinBorderEdgeTop, thinBorderEdgeLeft+thinBorderEdgeW, lastRow+1), border)
	assertBorderRectRun(t, img,
		image.Rect(thinBorderEdgeLeft, thinBorderEdgeTop, lastCol+1, thinBorderEdgeTop+thinBorderEdgeH), border)
}

// assertBorderRectRun fails on the first pixel inside region that is not
// exactly want, so a dropped or faded border row cannot pass.
func assertBorderRectRun(t *testing.T, img *image.NRGBA, region image.Rectangle, want color.NRGBA) {
	t.Helper()

	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			if got := img.NRGBAAt(x, y); got != want {
				t.Fatalf("border pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

// TestRoundedBorderArcPaintsAndOutsideCornerStaysClear samples a rounded
// stroke rectangle: the corner square inside the bounding box but outside the
// radius must stay background, the arc must carry ink, and the straight edges
// must stay exact border color. Squaring the corner or dropping the arc fails.
func TestRoundedBorderArcPaintsAndOutsideCornerStaysClear(t *testing.T) {
	t.Parallel()

	const (
		boxXpt  = 15.0
		boxYpt  = 15.0
		boxWpt  = 60.0
		boxHpt  = 30.0
		radius  = 7.5 // 10 px
		widthPt = 1.5
		arcLuma = 240
		minArc  = 4
	)

	border := color.NRGBA{R: 25, G: 51, B: 102, A: opaqueAlpha}

	res := &layout.Result{
		Width:  105,
		Height: 75,
		Ops: []layout.Op{{
			Kind: layout.OpStrokeRect, X: boxXpt, Y: boxYpt, W: boxWpt, H: boxHpt,
			R: 0.1, G: 0.2, B: 0.4, Width: widthPt, Radius: radius,
		}},
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, point := range []image.Point{
		{X: 20, Y: 20}, {X: 21, Y: 20}, {X: 20, Y: 21}, {X: 21, Y: 21},
	} {
		if got := img.NRGBAAt(point.X, point.Y); got != qualityOpaqueWhite() {
			t.Errorf("outside-corner pixel %v = %v, want white", point, got)
		}
	}

	arcInk := 0

	for y := 22; y < 28; y++ {
		for x := 22; x < 28; x++ {
			if qualityLuma(img.NRGBAAt(x, y)) < arcLuma {
				arcInk++
			}
		}
	}

	if arcInk < minArc {
		t.Errorf("rounded arc ink pixels = %d, want >= %d (arc missing)", arcInk, minArc)
	}

	if got := img.NRGBAAt(60, 20); got != border {
		t.Errorf("straight top edge = %v, want %v", got, border)
	}

	if got := img.NRGBAAt(20, 50); got != border {
		t.Errorf("straight left edge = %v, want %v", got, border)
	}
}

// TestTranslucentFillCompositesToExpectedChannel composites a 50% fill over
// white and checks the channel math, then repeats the same op with a direct
// final-resolution paint. The two paths must agree on the flat region because
// IMG-02's direct branch has to keep the same pixel contract; an alpha value
// silently dropped turns the translucent pixel opaque and fails both checks.
func TestTranslucentFillCompositesToExpectedChannel(t *testing.T) {
	t.Parallel()

	const channelTolerance = 2

	src := color.NRGBA{
		R: uint8(0.2 * channelMax),
		G: uint8(0.4 * channelMax),
		B: uint8(0.6 * channelMax),
		A: uint8(math.Round(0.5 * channelMax)),
	}
	want := qualityCompositeOver(qualityOpaqueWhite(), src)

	fillOp := layout.Op{
		Kind: layout.OpFillRect, X: 7.5, Y: 7.5, W: 15, H: 15,
		R: 0.2, G: 0.4, B: 0.6, Alpha: 0.5,
	}

	res := &layout.Result{Width: 30, Height: 30, Ops: []layout.Op{fillOp}}

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	got := img.NRGBAAt(20, 20)
	requireChannelsNear(t, "supersampled translucent pixel", got, want, channelTolerance)

	direct := qualityWhiteCanvas(40, 40)
	opCopy := fillOp

	paint(direct, &opCopy, ptToPx, newGlyphAtlas(), newRasterImageCache())

	directPixel := direct.NRGBAAt(20, 20)
	requireChannelsNear(t, "direct translucent pixel", directPixel, got, 1)

	opaque := fillOp
	opaque.Alpha = 1

	resOpaque := &layout.Result{Width: 30, Height: 30, Ops: []layout.Op{opaque}}

	imgOpaque, err := rasterizeContext(t.Context(), resOpaque, resOpaque.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if opaquePixel := imgOpaque.NRGBAAt(20, 20); opaquePixel == got {
		t.Errorf("opaque fill pixel equals translucent pixel %v; alpha was not applied", got)
	}
}

// TestTransformedOpsLandAtExpectedCanvasPositions samples the center of a
// scaled and a rotated fill plus a probe point that only paints when the
// transform is ignored. Ink at the wrong point or a missing transformed shape
// fails.
func TestTransformedOpsLandAtExpectedCanvasPositions(t *testing.T) {
	t.Parallel()

	red := color.NRGBA{R: channelMax, A: opaqueAlpha}

	tests := []struct {
		name    string
		op      layout.Op
		inside  image.Point
		outside image.Point
	}{
		{
			name: "scale-1.5",
			op: layout.Op{
				Kind: layout.OpFillRect, X: 15, Y: 15, W: 15, H: 15, R: 1, Alpha: 1,
				Xform: layout.Scale(1.5, 1.5), XformSet: true,
			},
			inside:  image.Point{X: 45, Y: 45}, // scaled box spans 30..60 px
			outside: image.Point{X: 25, Y: 25}, // only the untransformed box covers this
		},
		{
			name: "rotate-90-about-center",
			op: layout.Op{
				Kind: layout.OpFillRect, X: 30, Y: 30, W: 20, H: 10, R: 1, Alpha: 1,
				Xform: layout.BakeOrigin(layout.RotateDeg(90), 40, 35), XformSet: true,
			},
			inside:  image.Point{X: 53, Y: 47}, // rotated bar spans 46.7..60 x 33.3..60 px
			outside: image.Point{X: 63, Y: 42}, // only the unrotated bar covers this
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := &layout.Result{Width: 150, Height: 150, Ops: []layout.Op{testCase.op}}

			img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
			if err != nil {
				t.Fatal(err)
			}

			if got := img.NRGBAAt(testCase.inside.X, testCase.inside.Y); got != red {
				t.Errorf("transformed pixel %v = %v, want red", testCase.inside, got)
			}

			if got := img.NRGBAAt(testCase.outside.X, testCase.outside.Y); got != qualityOpaqueWhite() {
				t.Errorf("untransformed-only pixel %v = %v, want white", testCase.outside, got)
			}
		})
	}
}

// TestTransparentRasterKeepsUnpaintedAlphaZeroThroughPNG checks the
// transparent PNG contract end to end: unpainted areas must decode with alpha
// 0 and must not be opaque white, painted areas must stay opaque, and the
// encoded PNG must preserve both.
func TestTransparentRasterKeepsUnpaintedAlphaZeroThroughPNG(t *testing.T) {
	t.Parallel()

	red := color.NRGBA{R: channelMax, A: opaqueAlpha}

	res := &layout.Result{
		Width: 60, Height: 37.5,
		Ops: []layout.Op{{
			Kind: layout.OpFillRect, X: 7.5, Y: 7.5, W: 22.5, H: 15,
			R: 1, G: 0, B: 0, Alpha: 1,
		}},
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, true, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	requireImageBounds(t, img, 80, 50)
	assertUnpaintedCornerTransparent(t, img.NRGBAAt(70, 45), "transparent canvas corner")

	if got := img.NRGBAAt(20, 20); got != red {
		t.Errorf("painted pixel = %v, want red", got)
	}

	opaque, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if got := opaque.NRGBAAt(70, 45); got != qualityOpaqueWhite() {
		t.Errorf("opaque canvas corner = %v, want white", got)
	}

	decoded := roundTripPNG(t, img)

	requireImageBounds(t, decoded, 80, 50)
	assertUnpaintedCornerTransparent(t, asNRGBA(decoded.At(70, 45)), "decoded PNG corner")

	if got := asNRGBA(decoded.At(20, 20)); got != red {
		t.Errorf("decoded painted pixel = %v, want red", got)
	}
}

// assertUnpaintedCornerTransparent requires an unpainted pixel to decode with
// alpha 0 and not to be opaque white.
func assertUnpaintedCornerTransparent(t *testing.T, corner color.NRGBA, label string) {
	t.Helper()

	if corner.A != 0 {
		t.Errorf("%s alpha = %d, want 0", label, corner.A)
	}

	if corner == qualityOpaqueWhite() {
		t.Errorf("%s is opaque white; the transparent contract was lost", label)
	}
}

// requireImageBounds fails when img is not wantWidth x wantHeight.
func requireImageBounds(t *testing.T, img image.Image, wantWidth, wantHeight int) {
	t.Helper()

	if got := img.Bounds(); got.Dx() != wantWidth || got.Dy() != wantHeight {
		t.Fatalf("bounds = %v, want %dx%d", got, wantWidth, wantHeight)
	}
}

// roundTripPNG encodes img and decodes it again, so callers can assert that
// the PNG codec preserved pixels.
func roundTripPNG(t *testing.T, img *image.NRGBA) image.Image {
	t.Helper()

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}

	decoded, err := png.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("png decode: %v", err)
	}

	return decoded
}

// The synthetic 250-tile geometry mirrors the public library workload measured
// on 2026-09-11 (document_bench_test.go:231-259): flex shrink fits 8 tiles per
// row at 119px width and 127px pitch, with a 120px width on the trailing
// partial row. 250 tiles over 32 rows is exactly 1024x2056, the canvas IMG-02's
// direct-resolution threshold must cover. Tiles paint their text label so the
// direct and supersampled branches are distinguishable by decoded bytes.
const (
	tileWorkloadCount  = 250
	tileWorkloadW      = 119.0
	tileWorkloadH      = 56.0
	tileWorkloadGap    = 8.0
	tileWorkloadPad    = 8.0
	tileWorkloadRow    = 8
	tileWorkloadRows   = 32
	tileWorkloadWide   = 1024
	tileWorkloadTall   = 2056
	tileWorkloadTextPt = 12.0
)

// tileWorkloadFill is the tile background #d9e2ec as layout op channels.
func tileWorkloadFill() (float64, float64, float64) {
	return 217.0 / channelMax, 226.0 / channelMax, 236.0 / channelMax
}

// tileWorkloadText is the tile label #17324d as layout op channels.
func tileWorkloadText() (float64, float64, float64) {
	return 23.0 / channelMax, 50.0 / channelMax, 77.0 / channelMax
}

// tileWorkloadRowsFor returns the grid row count for count tiles at the public
// 8-tiles-per-row wrap.
func tileWorkloadRowsFor(count int) int {
	return (count + tileWorkloadRow - 1) / tileWorkloadRow
}

// tileWorkloadOps builds the synthetic tile grid for count tiles: one flat
// #d9e2ec fill and one text label per tile, aligned to integer CSS pixels.
func tileWorkloadOps(count int) []layout.Op {
	red, green, blue := tileWorkloadFill()
	textRed, textGreen, textBlue := tileWorkloadText()

	ops := make([]layout.Op, 0, count*2)

	for row := range tileWorkloadRowsFor(count) {
		rowTiles := min(count-row*tileWorkloadRow, tileWorkloadRow)

		for col := range rowTiles {
			tileX := tileWorkloadPad + float64(col)*(tileWorkloadW+tileWorkloadGap)
			tileY := tileWorkloadPad + float64(row)*(tileWorkloadH+tileWorkloadGap)
			number := row*tileWorkloadRow + col + 1

			ops = append(ops, layout.Op{
				Kind: layout.OpFillRect,
				X:    tileX * cssPxToPt, Y: tileY * cssPxToPt,
				W: tileWorkloadW * cssPxToPt, H: tileWorkloadH * cssPxToPt,
				R: red, G: green, B: blue, Alpha: 1,
			})
			ops = append(ops, layout.Op{
				Kind: layout.OpText,
				X:    (tileX + tileWorkloadPad) * cssPxToPt, Y: (tileY + 20) * cssPxToPt,
				Text: "Tile " + strconv.Itoa(number), Size: tileWorkloadTextPt,
				R: textRed, G: textGreen, B: textBlue, Alpha: 1,
			})
		}
	}

	return ops
}

// tileBandCenter returns the pixel at the center of the first tile in tile row
// row. The center sits below the text label, over flat fill.
func tileBandCenter(row int) image.Point {
	return image.Point{
		X: int(math.Round(tileWorkloadPad + tileWorkloadW/2)),
		Y: int(math.Round(tileWorkloadPad + float64(row)*(tileWorkloadH+tileWorkloadGap) + tileWorkloadH*3/4)),
	}
}

// TestTallTiledPageKeepsTileColorAtPublic250TileDimensions builds the
// synthetic equivalent of the public 250-tile canvas and requires the tile
// fill color #d9e2ec at the center of every tile band through the auto policy,
// then through a PNG encode/decode. Auto output is compared byte-for-byte with
// a forced direct render to prove the tall page takes the direct branch, and
// against a forced supersampled render to prove the fixture can tell the
// branches apart. Only flat interior pixels and geometry are asserted.
func TestTallTiledPageKeepsTileColorAtPublic250TileDimensions(t *testing.T) {
	t.Parallel()

	staticRed, staticGreen, staticBlue := tileWorkloadFill()
	want := color.NRGBA{
		R: uint8(staticRed * channelMax), G: uint8(staticGreen * channelMax),
		B: uint8(staticBlue * channelMax), A: opaqueAlpha,
	}

	ops := tileWorkloadOps(tileWorkloadCount)
	res := &layout.Result{
		Width:  tileWorkloadWide * cssPxToPt,
		Height: tileWorkloadTall * cssPxToPt,
		Ops:    ops,
	}

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	requireImageBounds(t, img, tileWorkloadWide, tileWorkloadTall)
	requireTileBands(t, "auto", func(point image.Point) color.NRGBA {
		return img.NRGBAAt(point.X, point.Y)
	}, want)

	forcedDirect, err := rasterizeContextPolicy(
		t.Context(), res, res.Height, false, 0, 0, rasterPolicyDirect,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(img.Pix, forcedDirect.Pix) {
		t.Error("auto tall page output differs from the forced direct branch; the direct policy did not select")
	}

	forcedSuper, err := rasterizeContextPolicy(
		t.Context(), res, res.Height, false, 0, 0, rasterPolicySupersample,
	)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(img.Pix, forcedSuper.Pix) {
		t.Error("auto tall page output equals the supersampled branch; the fixture cannot detect the policy")
	}

	decoded := roundTripPNG(t, img)

	requireImageBounds(t, decoded, tileWorkloadWide, tileWorkloadTall)
	requireTileBands(t, "decoded", func(point image.Point) color.NRGBA {
		return asNRGBA(decoded.At(point.X, point.Y))
	}, want)
}

// requireTileBands samples the center of every tile row in a 250-tile canvas
// and requires the flat fill color there.
func requireTileBands(t *testing.T, label string, pixelAt func(image.Point) color.NRGBA, want color.NRGBA) {
	t.Helper()

	for row := range tileWorkloadRows {
		point := tileBandCenter(row)
		if got := pixelAt(point); got != want {
			t.Errorf("%s tile band row %d pixel %v = %v, want %v", label, row, point, got, want)
		}
	}
}
