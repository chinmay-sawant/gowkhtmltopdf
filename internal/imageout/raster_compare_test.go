package imageout

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// raster_compare_test.go is the IMG-03 decoded-image comparison: the public
// 250-tile workload is rendered through the direct final-resolution branch and
// through the supersampled branch, pixel differences are reported per region
// (flat fill, text, border), and two PNG artifacts are written for native-scale
// inspection when IMG03_ARTIFACT_DIR names a directory.

// imageCompareTileCSS mirrors libraryBenchmarkImageDocument in
// document_bench_test.go:172-183.
const imageCompareTileCSS = `
body { margin: 0; font-family: sans-serif; }
.grid { display: flex; flex-wrap: wrap; gap: 8px; padding: 8px; }
.tile { width: 120px; height: 56px; background: #d9e2ec; color: #17324d;
  border: 1px solid #52718d; padding: 8px; box-sizing: border-box; }
`

// imageCompareFixture parses the public 250-tile workload markup and its
// stylesheet into the layout inputs the raster policy consumes.
func imageCompareFixture(t *testing.T) (*html.Node, []*css.Stylesheet) {
	t.Helper()

	var source strings.Builder

	source.WriteString(`<!doctype html><html><head><meta charset="utf-8"><style>`)
	source.WriteString(imageCompareTileCSS)
	source.WriteString(`</style></head><body><div class="grid">`)

	for tile := 1; tile <= tileWorkloadCount; tile++ {
		fmt.Fprintf(&source, `<div class="tile">Tile %d</div>`, tile)
	}

	source.WriteString(`</div></body></html>`)

	root, err := html.Parse(source.String())
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(imageCompareTileCSS)
	if err != nil {
		t.Fatal(err)
	}

	return root, []*css.Stylesheet{sheet}
}

// imageCompareLayout lays out the 250-tile fixture at the public 1024px
// viewport and returns the display list plus canvas height.
func imageCompareLayout(t *testing.T) (*layout.Result, float64) {
	t.Helper()

	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	root, sheets := imageCompareFixture(t)
	opts := RenderOptions{
		Width: 1024, SmartWidth: false, Background: true,
		Font: face, Sheets: sheets,
	}

	res, err := layoutResult(t.Context(), root, opts, face)
	if err != nil {
		t.Fatal(err)
	}

	return res, maxHeight(res, opts)
}

// imageCompareRegionDiff is the decoded pixel difference for one region.
type imageCompareRegionDiff struct {
	name       string
	pixels     int
	diffPixels int
	sumAbs     int64
	maxAbs     int
}

// meanAbs is the mean absolute channel difference over every RGBA channel in
// the region.
func (d imageCompareRegionDiff) meanAbs() float64 {
	if d.pixels == 0 {
		return 0
	}

	return float64(d.sumAbs) / float64(d.pixels*4)
}

func (d imageCompareRegionDiff) String() string {
	return fmt.Sprintf("%s pixels=%d diffPixels=%d (%.2f%%) meanAbs=%.3f maxAbs=%d",
		d.name, d.pixels, d.diffPixels, 100*float64(d.diffPixels)/float64(max(d.pixels, 1)),
		d.meanAbs(), d.maxAbs)
}

// diffImageRegion compares direct against super over region.
func diffImageRegion(name string, direct, super *image.NRGBA, region image.Rectangle) imageCompareRegionDiff {
	region = region.Intersect(direct.Bounds()).Intersect(super.Bounds())
	diff := imageCompareRegionDiff{name: name, pixels: region.Dx() * region.Dy()}

	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			directOffset := direct.PixOffset(x, y)
			superOffset := super.PixOffset(x, y)
			pixelDiffers := false

			for channel := range 4 {
				delta := channelDelta(direct.Pix[directOffset+channel], super.Pix[superOffset+channel])
				diff.sumAbs += int64(delta)

				if delta > diff.maxAbs {
					diff.maxAbs = delta
				}

				if delta > 0 {
					pixelDiffers = true
				}
			}

			if pixelDiffers {
				diff.diffPixels++
			}
		}
	}

	return diff
}

// imageCompareRegions picks one representative region per display-list class:
// the bottom strip of the first tile fill (below the label), the first text
// run, and the first straight border line.
func imageCompareRegions(
	t *testing.T, res *layout.Result, bounds image.Rectangle,
) (image.Rectangle, image.Rectangle, image.Rectangle) {
	t.Helper()

	fillOp, textOp, lineOp := firstScanlineOps(res.Ops)

	if fillOp == nil || textOp == nil || lineOp == nil {
		t.Fatalf("fixture ops missing: fill=%v text=%v line=%v", fillOp != nil, textOp != nil, lineOp != nil)
	}

	var flat, text, border image.Rectangle

	fillRect := ptRectScale(fillOp.X, fillOp.Y, fillOp.W, fillOp.H, ptToPx)
	flat = image.Rect(fillRect.Min.X+10, fillRect.Max.Y-10, fillRect.Max.X-10, fillRect.Max.Y-2)

	textRect := paintOpBounds(textOp, ptToPx)
	text = image.Rect(textRect.Min.X-2, textRect.Min.Y-2, textRect.Max.X+2, textRect.Max.Y+2)

	border = paintOpBounds(lineOp, ptToPx)

	flat = flat.Intersect(bounds)
	text = text.Intersect(bounds)
	border = border.Intersect(bounds)

	if flat.Empty() || text.Empty() || border.Empty() {
		t.Fatalf("fixture regions empty after clipping: flat=%v text=%v border=%v", flat, text, border)
	}

	return flat, text, border
}

// firstScanlineOps returns the first fill, text, and line/stroke op in the
// display list, which the IMG-03 region probe samples.
func firstScanlineOps(ops []layout.Op) (*layout.Op, *layout.Op, *layout.Op) {
	var fillOp, textOp, lineOp *layout.Op

	for index := range ops {
		candidate := &ops[index]

		switch {
		case candidate.Kind == layout.OpFillRect && fillOp == nil:
			fillOp = candidate
		case candidate.Kind == layout.OpText && textOp == nil:
			textOp = candidate
		case (candidate.Kind == layout.OpLine || candidate.Kind == layout.OpStrokeRect) && lineOp == nil:
			lineOp = candidate
		}
	}

	return fillOp, textOp, lineOp
}

// imageCompareDarkFootprint returns the number of pixels in region that are
// dark in img but have no dark pixel within one pixel in other. It separates
// edge antialiasing differences from moved glyphs: a moved glyph leaves
// unmatched cores, while coverage differences keep a matching neighbor.
func imageCompareDarkFootprint(img, other *image.NRGBA, region image.Rectangle, threshold int) int {
	inRegion := func(x, y int) bool {
		return image.Pt(x, y).In(region)
	}
	dark := func(target *image.NRGBA, x, y int) bool {
		return inRegion(x, y) && qualityLuma(target.NRGBAAt(x, y)) < threshold
	}
	hasDarkNeighbor := func(target *image.NRGBA, x, y int) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dark(target, x+dx, y+dy) {
					return true
				}
			}
		}

		return false
	}

	unmatched := 0

	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			if dark(img, x, y) && !hasDarkNeighbor(other, x, y) {
				unmatched++
			}
		}
	}

	return unmatched
}

// imageCompareBorderRun returns the longest run of border-colored pixels on the
// best row of region.
func imageCompareBorderRun(img *image.NRGBA, region image.Rectangle, want color.NRGBA, tolerance int) int {
	best := 0

	for y := region.Min.Y; y < region.Max.Y; y++ {
		run := 0

		for x := region.Min.X; x < region.Max.X; x++ {
			c := img.NRGBAAt(x, y)

			if channelDelta(c.R, want.R) <= tolerance &&
				channelDelta(c.G, want.G) <= tolerance &&
				channelDelta(c.B, want.B) <= tolerance {
				run++
			}
		}

		best = max(best, run)
	}

	return best
}

// writeImageComparePNG encodes img and writes it to path, reporting the byte
// size in the test log.
func writeImageComparePNG(t *testing.T, path string, img *image.NRGBA) {
	t.Helper()

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // test artifact
		t.Fatalf("write %s: %v", path, err)
	}

	t.Logf("artifact %s: %dx%d, %d bytes", path, img.Bounds().Dx(), img.Bounds().Dy(), buf.Len())
}

// TestIMG03DirectVersusSupersampledTileQuality is the IMG-03 evidence test.
// It renders the public 250-tile fixture through both raster policies, reports
// decoded pixel differences for flat fill, text, and border regions, and
// enforces the policy decision: flat fills stay byte-exact, text differences
// stay bounded stem-weight coverage with no moved glyph cores, and the direct
// branch keeps an aligned 1px border that the 2x path spreads over two rows.
// Set IMG03_ARTIFACT_DIR to write the two native-scale PNGs.
//
//nolint:paralleltest // holds a direct canvas plus a 2x intermediate
func TestIMG03DirectVersusSupersampledTileQuality(t *testing.T) {
	borderColor := color.NRGBA{R: 0x52, G: 0x71, B: 0x8d, A: opaqueAlpha}

	res, height := imageCompareLayout(t)

	direct, err := rasterizeContextPolicy(t.Context(), res, height, false, 0, 0, rasterPolicyDirect)
	if err != nil {
		t.Fatal(err)
	}

	super, err := rasterizeContextPolicy(t.Context(), res, height, false, 0, 0, rasterPolicySupersample)
	if err != nil {
		t.Fatal(err)
	}

	if direct.Bounds() != super.Bounds() {
		t.Fatalf("bounds differ: direct=%v supersampled=%v", direct.Bounds(), super.Bounds())
	}

	requireImageBounds(t, direct, tileWorkloadWide, tileWorkloadTall)

	flat, text, border := imageCompareRegions(t, res, direct.Bounds())

	assertIMG03FlatAndTextQuality(t, direct, super, flat, text)
	assertIMG03BorderQuality(t, direct, super, border, borderColor)

	if dir := os.Getenv("IMG03_ARTIFACT_DIR"); dir != "" {
		writeImageComparePNG(t, filepath.Join(dir, "img-03-direct.png"), direct)
		writeImageComparePNG(t, filepath.Join(dir, "img-03-supersampled.png"), super)
	}
}

// IMG-03 quality bounds. Measured on 2026-09-11, go1.26.4: text mean abs
// 3.312, max 81, 13.26% of text pixels differ. The bounds keep a margin
// without accepting a structural change.
const (
	img03TextMeanAbsMax = 8.0
	img03TextDiffPctMax = 25.0
	img03TextMaxAbsMax  = 128
	// The direct branch paints the 119px top border as a solid run on the
	// aligned row; the supersampled branch splits it over two rows, so its
	// best-row run is zero.
	img03BorderRunMin = 100
	// Border color is #52718d from the fixture stylesheet.
	img03BorderTolerance = 30
)

// assertIMG03FlatAndTextQuality enforces the IMG-03 flat and text contract:
// flat fills stay byte-exact, and text differences stay bounded stem-weight
// coverage with no moved glyph cores.
func assertIMG03FlatAndTextQuality(t *testing.T, direct, super *image.NRGBA, flat, text image.Rectangle) {
	t.Helper()

	flatDiff := diffImageRegion("flat", direct, super, flat)
	textDiff := diffImageRegion("text", direct, super, text)

	t.Logf("%v", flatDiff)
	t.Logf("%v", textDiff)

	if flatDiff.diffPixels != 0 || flatDiff.maxAbs != 0 {
		t.Errorf("flat fill differs between policies: %v", flatDiff)
	}

	if textDiff.diffPixels == 0 {
		t.Error("text region is byte-identical; the comparison cannot judge text quality")
	}

	if textDiff.meanAbs() > img03TextMeanAbsMax {
		t.Errorf("text mean abs diff %.3f > %.3f: %v", textDiff.meanAbs(), img03TextMeanAbsMax, textDiff)
	}

	if pct := 100 * float64(textDiff.diffPixels) / float64(textDiff.pixels); pct > img03TextDiffPctMax {
		t.Errorf("text differing pixels %.2f%% > %.2f%%: %v", pct, img03TextDiffPctMax, textDiff)
	}

	if textDiff.maxAbs > img03TextMaxAbsMax {
		t.Errorf("text max channel diff %d > %d: %v", textDiff.maxAbs, img03TextMaxAbsMax, textDiff)
	}

	if missed := imageCompareDarkFootprint(direct, super, text, 128); missed != 0 {
		t.Errorf("%d direct text pixels have no dark supersampled neighbor within 1px", missed)
	}

	if missed := imageCompareDarkFootprint(super, direct, text, 128); missed != 0 {
		t.Errorf("%d supersampled text pixels have no dark direct neighbor within 1px", missed)
	}
}

// assertIMG03BorderQuality requires the direct branch to keep an aligned 1px
// border run that the supersampled branch spreads over two rows.
func assertIMG03BorderQuality(
	t *testing.T, direct, super *image.NRGBA, border image.Rectangle, borderColor color.NRGBA,
) {
	t.Helper()

	borderDiff := diffImageRegion("border", direct, super, border)

	t.Logf("%v", borderDiff)

	directRun := imageCompareBorderRun(direct, border, borderColor, img03BorderTolerance)
	superRun := imageCompareBorderRun(super, border, borderColor, img03BorderTolerance)

	t.Logf("best border row run: direct=%d supersampled=%d pixels", directRun, superRun)

	if directRun < img03BorderRunMin {
		t.Errorf("direct border run = %d, want >= %d: the 1px border was dropped or faded", directRun, img03BorderRunMin)
	}

	if directRun <= superRun {
		t.Errorf("direct border run %d <= supersampled %d; the comparison cannot show the aligned edge", directRun, superRun)
	}
}
