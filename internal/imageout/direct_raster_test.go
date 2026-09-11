package imageout

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"runtime"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestFakeBoldOffsetScalesWithEffectiveSupersample pins the paintText audit:
// the fake-bold pass must sit one final CSS pixel from the base pass at every
// effective supersample factor. The reference draws the plain run twice with
// an explicit one-CSS-pixel X offset, so using the rasterSS constant instead
// of the effective factor at the direct resolution differs by one canvas
// pixel and fails byte equality.
func TestFakeBoldOffsetScalesWithEffectiveSupersample(t *testing.T) {
	t.Parallel()

	base := layout.Op{
		Kind: layout.OpText, X: 30, Y: 40, Text: "Hamburg", Size: 12,
		Bold: true, R: 0, G: 0, B: 0, Alpha: 1,
	}

	for _, scale := range []float64{1, rasterSS} {
		t.Run(fmt.Sprintf("scale-%g", scale), func(t *testing.T) {
			t.Parallel()

			pxPerPt := ptToPx * scale

			fakeBold := qualityWhiteCanvas(200, 80)
			fakeBoldOp := base

			paint(fakeBold, &fakeBoldOp, pxPerPt, newGlyphAtlas(), newRasterImageCache())

			reference := qualityWhiteCanvas(200, 80)
			plainOp := base
			plainOp.Bold = false
			offsetOp := plainOp
			offsetOp.X += cssPxToPt // one final CSS pixel

			atlas := newGlyphAtlas()
			cache := newRasterImageCache()

			paint(reference, &plainOp, pxPerPt, atlas, cache)
			paint(reference, &offsetOp, pxPerPt, atlas, cache)

			if !bytes.Equal(fakeBold.Pix, reference.Pix) {
				t.Errorf("fake-bold pass at effective scale %g is not one final pixel from the base pass", scale)
			}
		})
	}
}

// TestDirectRasterThresholdPolicy pins the documented IMG-02 rule: the direct
// branch starts at directRasterPixels, the constant is derived from the 32 MiB
// supersample cache cap, and both public tile workloads sit above it.
func TestDirectRasterThresholdPolicy(t *testing.T) {
	t.Parallel()

	const twoMiPixels = 2 << 20

	if directRasterPixels != twoMiPixels {
		t.Fatalf("directRasterPixels = %d, want %d (32 MiB / (4*2*2))", directRasterPixels, twoMiPixels)
	}

	// The 2x canvas of a threshold-size final canvas is exactly the per-buffer
	// retention cap, which is why the threshold is a pixel area and not a byte
	// count of the final canvas.
	if directRasterPixels*rasterSS*rasterSS*4 != maxPooledRasterBytes {
		t.Fatalf("threshold 2x canvas = %d bytes, want maxPooledRasterBytes %d",
			directRasterPixels*rasterSS*rasterSS*4, maxPooledRasterBytes)
	}

	if !directRaster(directRasterPixels) {
		t.Error("threshold pixel area should take the direct branch")
	}

	if directRaster(directRasterPixels - 1) {
		t.Error("one pixel below the threshold should keep the supersampled branch")
	}

	if !directRaster(1024 * 2056) {
		t.Error("public 250-tile canvas (1024x2056) should take the direct branch")
	}

	if !directRaster(1024 * 4040) {
		t.Error("public 500-tile canvas (1024x4040) should take the direct branch")
	}
}

// TestDirectRasterThresholdBoundary renders the same fixture at exactly
// directRasterPixels and one final row below it. The auto policy must
// reproduce the forced direct bytes at the threshold and the forced
// supersampled bytes below it, with dimensions and the marker pixel correct
// in both. A fixture text run makes the two branches distinguishable, so
// neither byte comparison can pass vacuously.
func TestDirectRasterThresholdBoundary(t *testing.T) {
	t.Parallel()

	ops := thresholdBoundaryOps()

	// 1024x2048 final pixels is exactly directRasterPixels: the 2x canvas is
	// 2048x4096. 1024x2049 is one row above and 1024x2047 one row below.
	atThreshold := &layout.Result{Width: 768, Height: 1536, Ops: ops}
	aboveThreshold := &layout.Result{Width: 768, Height: 1536.75, Ops: ops}
	belowThreshold := &layout.Result{Width: 768, Height: 1535.25, Ops: ops}

	tests := []struct {
		name       string
		res        *layout.Result
		wantHeight int
		wantDirect bool
	}{
		{name: "one-row-above", res: aboveThreshold, wantHeight: 2049, wantDirect: true},
		{name: "exactly-at-threshold", res: atThreshold, wantHeight: 2048, wantDirect: true},
		{name: "one-row-below", res: belowThreshold, wantHeight: 2047, wantDirect: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertThresholdBoundaryCase(t, testCase.res, testCase.wantHeight, testCase.wantDirect)
		})
	}
}

// thresholdBoundaryOps is the fill plus text fixture that makes the direct and
// supersampled branches distinguishable.
func thresholdBoundaryOps() []layout.Op {
	return []layout.Op{
		{
			Kind: layout.OpFillRect, X: 10, Y: 10, W: 30, H: 30,
			R: 1, G: 0, B: 0, Alpha: 1,
		},
		{
			Kind: layout.OpText, X: 20, Y: 80, Text: "Threshold boundary", Size: 12,
			R: 0, G: 0, B: 0, Alpha: 1,
		},
	}
}

// assertThresholdBoundaryCase renders one boundary case through all three
// policies and requires the auto policy to match the branch the threshold
// predicts.
func assertThresholdBoundaryCase(t *testing.T, res *layout.Result, wantHeight int, wantDirect bool) {
	t.Helper()

	render := func(policy rasterPolicy) *image.NRGBA {
		t.Helper()

		img, err := rasterizeContextPolicy(t.Context(), res, res.Height, false, 0, 0, policy)
		if err != nil {
			t.Fatal(err)
		}

		return img
	}

	auto := render(rasterPolicyAuto)
	direct := render(rasterPolicyDirect)
	super := render(rasterPolicySupersample)

	if got := auto.Bounds(); got.Dx() != 1024 || got.Dy() != wantHeight {
		t.Fatalf("auto canvas = %v, want 1024x%d", got, wantHeight)
	}

	red := color.NRGBA{R: channelMax, A: opaqueAlpha}
	if got := auto.NRGBAAt(25, 25); got != red {
		t.Fatalf("marker pixel (25,25) = %v, want red", got)
	}

	if bytes.Equal(direct.Pix, super.Pix) {
		t.Fatal("forced direct and supersampled renders are byte-identical; the fixture cannot detect the policy")
	}

	if wantDirect {
		if !bytes.Equal(auto.Pix, direct.Pix) {
			t.Error("auto output at the threshold does not match the forced direct branch")
		}
	} else if !bytes.Equal(auto.Pix, super.Pix) {
		t.Error("auto output below the threshold does not match the forced supersampled branch")
	}
}

// TestDirectRasterDoesNotEnterSupersampleCache runs two conversions at each
// public tile size (250 and 500) and inspects the process-wide supersample
// cache and Go heap between them. The direct branch must not retain the final
// canvas (8.4 MB at 250 tiles, 16.5 MB at 500 tiles) or any other buffer
// between conversions.
//
//nolint:paralleltest // inspects the process-wide supersamplePixCache
func TestDirectRasterDoesNotEnterSupersampleCache(t *testing.T) {
	tests := []struct {
		name       string
		tiles      int
		wantHeight int
	}{
		{name: "250-tiles", tiles: 250, wantHeight: 2056},
		{name: "500-tiles", tiles: 500, wantHeight: 4040},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assertDirectRasterCacheNeutral(t, testCase.tiles, testCase.wantHeight)
		})
	}

	assertSupersampleCacheRetainsControlBuffer(t)
}

// assertDirectRasterCacheNeutral renders the tile workload twice and requires
// the process-wide cache to stay unchanged, the largest retained buffer to
// stay below one final canvas, and the heap not to grow across the second
// render.
func assertDirectRasterCacheNeutral(t *testing.T, tiles, wantHeight int) {
	t.Helper()

	res := &layout.Result{
		Width:  tileWorkloadWide * cssPxToPt,
		Height: float64(wantHeight) * cssPxToPt,
		Ops:    tileWorkloadOps(tiles),
	}

	count0, total0, largest0 := supersamplePixCache.stats()
	finalCanvasBytes := tileWorkloadWide * wantHeight * 4

	var before runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	bounds1 := renderTallDirect(t, res)

	var middle runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&middle)

	bounds2 := renderTallDirect(t, res)

	var after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&after)

	if bounds1 != bounds2 || bounds1.Dx() != tileWorkloadWide || bounds1.Dy() != wantHeight {
		t.Fatalf("direct renders = %v and %v, want %dx%d", bounds1, bounds2, tileWorkloadWide, wantHeight)
	}

	count1, total1, largest1 := supersamplePixCache.stats()

	assertCacheStatsUnchanged(t, count0, total0, largest0, count1, total1, largest1)
	assertHeapGrowthBounded(t, middle, after)

	if largest1 >= finalCanvasBytes {
		t.Errorf("cache retains a buffer of %d bytes >= direct canvas %d bytes", largest1, finalCanvasBytes)
	}

	t.Logf("direct canvas %d bytes; cache before: count=%d total=%d largest=%d",
		finalCanvasBytes, count0, total0, largest0)
	t.Logf("cache after: count=%d total=%d largest=%d", count1, total1, largest1)
	t.Logf("HeapAlloc before=%d middle=%d after=%d", before.HeapAlloc, middle.HeapAlloc, after.HeapAlloc)
	t.Logf("HeapInuse before=%d middle=%d after=%d", before.HeapInuse, middle.HeapInuse, after.HeapInuse)
	t.Logf("HeapSys before=%d middle=%d after=%d", before.HeapSys, middle.HeapSys, after.HeapSys)
}

// renderTallDirect renders res at its own height and returns the canvas bounds.
func renderTallDirect(t *testing.T, res *layout.Result) image.Rectangle {
	t.Helper()

	img, err := rasterizeContext(t.Context(), res, res.Height, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	return img.Bounds()
}

func assertCacheStatsUnchanged(t *testing.T, count0, total0, largest0, count1, total1, largest1 int) {
	t.Helper()

	if count1 != count0 || total1 != total0 || largest1 != largest0 {
		t.Errorf(
			"direct renders changed cache stats: before (count=%d total=%d largest=%d), "+
				"after (count=%d total=%d largest=%d)",
			count0, total0, largest0, count1, total1, largest1,
		)
	}
}

// assertHeapGrowthBounded allows normal runtime noise between the two renders
// but fails when a full canvas would have been retained. The slack is far
// below one canvas.
func assertHeapGrowthBounded(t *testing.T, middle, after runtime.MemStats) {
	t.Helper()

	const heapSlack = 4 << 20

	if after.HeapAlloc > middle.HeapAlloc+heapSlack {
		t.Errorf("HeapAlloc grew from %d to %d across the second direct render, want <= %d",
			middle.HeapAlloc, after.HeapAlloc, middle.HeapAlloc+heapSlack)
	}

	if after.HeapInuse > middle.HeapInuse+heapSlack {
		t.Errorf("HeapInuse grew from %d to %d across the second direct render, want <= %d",
			middle.HeapInuse, after.HeapInuse, middle.HeapInuse+heapSlack)
	}
}

// assertSupersampleCacheRetainsControlBuffer proves the stats probe can see
// retention: a 512x512 final canvas has a 1024x1024 2x buffer (4 MiB) that
// fits the per-buffer cap, so the supersampled path must retain it.
func assertSupersampleCacheRetainsControlBuffer(t *testing.T) {
	t.Helper()

	control := &layout.Result{
		Width: 384, Height: 384,
		Ops: []layout.Op{{
			Kind: layout.OpFillRect, X: 10, Y: 10, W: 100, H: 100,
			R: 0, G: 0, B: 1, Alpha: 1,
		}},
	}

	if _, err := rasterizeContext(t.Context(), control, control.Height, false, 0, 0); err != nil {
		t.Fatal(err)
	}

	count, total, largest := supersamplePixCache.stats()
	controlBufferBytes := 512 * 512 * 4 * rasterSS * rasterSS

	if largest < controlBufferBytes {
		t.Errorf("control supersampled render retained largest=%d, want >= %d", largest, controlBufferBytes)
	}

	if count < 1 {
		t.Errorf("control supersampled render retained %d buffers, want >= 1", count)
	}

	t.Logf("control cache: count=%d total=%d largest=%d (expected buffer %d)",
		count, total, largest, controlBufferBytes)
}
