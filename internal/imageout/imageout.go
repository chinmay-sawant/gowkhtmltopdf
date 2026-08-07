package imageout

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

const (
	channelMax        = 255
	fnvOffsetBasis    = 14695981039346656037
	boxFilterFactor2  = 2
	boxFilterStride   = 8
	boxFilterHalf     = 4
	boxFilterArea     = 4 // 2x2 block of pixels (boxFilterFactor2 squared)
	pixelCenter       = 0.5
	defaultViewportW  = 768.0
	defaultViewportH  = 576.0
	qualityMaxPercent = 100
	opaqueAlpha       = 255
	formatPNG         = "png"
	formatJPG         = "jpg"
)

var (
	errNilRoot          = errors.New("imageout: nil root")
	errCropNoIntersect  = errors.New("imageout: crop rectangle does not intersect the canvas")
	errNilCommand       = errors.New("imageout: nil command")
	errNilRequest       = errors.New("imageout: nil request")
	errNothingToRender  = errors.New("load-error policy is skip; nothing to render")
	errImagesDisabled   = errors.New("images disabled")
	errNilOutput        = errors.New("imageout: nil Output writer")
	errNoInputToConvert = errors.New("no input to convert")
	errUnsupportedFmt   = errors.New("unsupported format")
)

// ptToPx maps layout canvas points to output pixels. The layout engine works
// in points with CSS pixels at 96 dpi (1 px = 0.75 pt, see
// layout/style.go pxToPt), so rasterizing at 1 CSS px = 1 output px means
// multiplying every point by 96/72.
const cssPxToPt = 0.75 // CSS px to layout points (1px = 0.75pt at 96dpi)

const ptToPx = 96.0 / 72.0

// rasterSS is the supersample factor for the paint canvas. Ops are painted
// at rasterSS times the final resolution, then box-filtered down. This
// stabilises small-text baselines and edges (stdlib has no FreeType hinting).
const rasterSS = 2

// screenWidthDefault is the wkhtmltoimage default viewport width in pixels
// (settings.ImageGlobal.Width default is already 1024; this guards against
// 0-width RenderOptions).
const screenWidthDefault = 1024

// maxSmartViewport caps the smart-width viewport growth in pixels.
const maxSmartViewport = 4096

// RenderOptions controls one Render call. Width/Height and Crop are in
// output pixels; the layout viewport is Width CSS pixels at 96 dpi.
type RenderOptions struct {
	Width       int // viewport width in pixels; <= 0 means 1024
	Height      int // minimum canvas height in pixels; 0 = content height
	Font        *pdf.Font
	Registry    *pdf.Registry // optional --font-path / system faces (CJK)
	Sheets      []*css.Stylesheet
	Media       string // "screen" (default), "print" or ""
	Images      func(src string) ([]byte, error)
	Background  bool // paint background colors
	Transparent bool // PNG background: alpha 0 instead of white
	Crop        image.Rectangle
	SmartWidth  bool // grow the viewport until content fits (default on)
	// PrintLinkUnderline mirrors --print-link-underline (opt-in).
	PrintLinkUnderline bool
}

// Render lays out root and rasterizes the result. The canvas is the viewport
// (or, with SmartWidth, the smallest grown viewport that fits the content)
// wide and max(content height, Height) tall. A non-empty Crop is applied to
// the rasterized canvas.
func Render(root *html.Node, opts RenderOptions) (image.Image, error) {
	return RenderContext(context.Background(), root, opts)
}

// RenderContext lays out and rasterizes root while observing ctx. Render is
// retained as the source-compatible background-context adapter.
//
//nolint:contextcheck // background fallback for nil-context callers (Render adapter)
func RenderContext(ctx context.Context, root *html.Node, opts RenderOptions) (image.Image, error) {
	if root == nil {
		return nil, errNilRoot
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("imageout: context: %w", err)
	}

	font := opts.Font
	if font == nil {
		var err error

		font, err = pdf.DefaultFont()
		if err != nil {
			return nil, fmt.Errorf("imageout: default font: %w", err)
		}
	}

	res, err := layoutResult(ctx, root, opts, font)
	if err != nil {
		return nil, err
	}

	img, err := rasterizeContext(ctx, res, maxHeight(res, opts), opts.Transparent)
	if err != nil {
		return nil, err
	}

	out, err := applyCrop(img, opts.Crop)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// layoutResult lays out root at the SmartWidth-grown or fixed viewport.
func layoutResult(ctx context.Context, root *html.Node, opts RenderOptions, font *pdf.Font) (*layout.Result, error) {
	if opts.SmartWidth {
		return layoutSmartWidth(ctx, root, opts, font)
	}

	viewport := float64(opts.Width)
	if viewport <= 0 {
		viewport = screenWidthDefault
	}

	res, err := layout.LayoutContext(ctx, root, layoutOptions(opts, font, viewport))
	if err != nil {
		return nil, fmt.Errorf("imageout: layout: %w", err)
	}

	return res, nil
}

// applyCrop clips img to crop, re-origin to (0,0); a zero rectangle is a no-op.
func applyCrop(img *image.NRGBA, crop image.Rectangle) (image.Image, error) {
	if crop.Empty() {
		return img, nil
	}

	inter := crop.Intersect(img.Bounds())
	if inter.Empty() {
		return nil, errCropNoIntersect
	}

	// re-origin the crop to (0,0): SubImage keeps the canvas
	// coordinate system, which is awkward for library callers
	return reOrigin(img.SubImage(inter)), nil
}

// reOrigin copies src into a fresh image whose bounds start at (0,0).
func reOrigin(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)

	return dst
}

// layoutOptions builds layout.Options for a viewport of viewportPx pixels.
func layoutOptions(opts RenderOptions, font *pdf.Font, viewportPx float64) layout.Options {
	heightPt := float64(opts.Height) * cssPxToPt
	if heightPt <= 0 {
		heightPt = viewportPx * cssPxToPt
	}

	return layout.Options{ //nolint:exhaustruct // intentional zero/partial fields
		Width:              viewportPx * cssPxToPt,
		Height:             heightPt,
		Font:               font,
		Registry:           opts.Registry,
		Sheets:             opts.Sheets,
		Media:              opts.Media,
		Images:             opts.Images,
		Background:         opts.Background,
		PrintLinkUnderline: opts.PrintLinkUnderline,
	}
}

// layoutSmartWidth lays out repeatedly, growing the viewport by 1.5x while
// painted content overflows the right edge (the layout engine always fills
// the full viewport width, so overflow is measured from the display list:
// max op.X+op.W). Growth is capped at maxSmartViewport pixels.
func layoutSmartWidth(
	ctx context.Context, root *html.Node, opts RenderOptions, font *pdf.Font,
) (*layout.Result, error) {
	viewport := float64(opts.Width)
	if viewport <= 0 {
		viewport = screenWidthDefault
	}

	var res *layout.Result

	for range 8 {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("imageout: context: %w", err)
		}

		var err error

		res, err = layout.LayoutContext(ctx, root, layoutOptions(opts, font, viewport))
		if err != nil {
			return nil, fmt.Errorf("imageout: layout: %w", err)
		}

		if viewport >= maxSmartViewport || contentWidthPx(res) <= viewport+0.5 {
			return res, nil
		}

		viewport *= 1.5
		if viewport > maxSmartViewport {
			viewport = maxSmartViewport
		}
	}

	return res, nil
}

// contentWidthPx returns the rightmost painted edge of the display list,
// converted to pixels. Link annotations do not paint and are ignored.
func contentWidthPx(res *layout.Result) float64 {
	maxW := 0.0

	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == layout.OpLinkURI {
			continue
		}

		if e := op.X + op.W; e > maxW {
			maxW = e
		}
	}

	return maxW * ptToPx
}

// maxHeight resolves the canvas height: the larger of the laid-out content
// height and the requested minimum (--height). layout.Result.Height reports
// the content height only, so the minimum must be applied here.
func maxHeight(res *layout.Result, opts RenderOptions) float64 {
	h := res.Height
	if hp := float64(opts.Height) * cssPxToPt; hp > h {
		h = hp
	}

	return h
}

// rasterizeContext paints the display list into an NRGBA canvas. The canvas is
// white unless transparent is set, in which case it starts fully transparent
// and only painted ops become visible. Painting uses rasterSS supersampling
// then box-filters down to the final CSS-pixel size. Glyph bitmaps for this
// run live on a per-rasterize atlas (P5-05) so concurrent Renders do not share
// mutable cache state.
func rasterizeContext(ctx context.Context, res *layout.Result, height float64, transparent bool) (*image.NRGBA, error) {
	pxPerPt := ptToPx * float64(rasterSS)
	widthPx := int(math.Round(res.Width * pxPerPt))
	heightPx := int(math.Round(height * pxPerPt))

	if widthPx < 1 {
		widthPx = 1
	}

	if heightPx < 1 {
		heightPx = 1
	}

	img := image.NewNRGBA(image.Rect(0, 0, widthPx, heightPx))
	if !transparent {
		fillNRGBAOpaque(img, img.Bounds(), color.NRGBA{R: channelMax, G: channelMax, B: channelMax, A: opaqueAlpha})
	}

	atlas := newGlyphAtlas()
	imageCache := newRasterImageCache()

	for _, i := range rasterPaintOrder(res.Ops) {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("imageout: context: %w", err)
		}

		paint(img, &res.Ops[i], pxPerPt, atlas, imageCache)
	}

	if rasterSS <= 1 {
		return img, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("imageout: context: %w", err)
	}

	return downscaleBox(img, rasterSS), nil
}

// rasterPaintOrder mirrors layout's PDF display-list policy: z-index first,
// then chrome (backgrounds/borders/lines) below content (text/images), with
// stable source order as the final tie breaker. Raster output used to walk
// Result.Ops directly, so a list produced by layout could render differently
// from the PDF page even though both consumed the same display list.
// FIX-REVIEW: PAINT-01 PDF body/header/footer traversal remains owned by
// internal/layout.Paint and PaintBand; this package consumes the same ordering
// and StyleOf policy without duplicating pagination or annotation semantics.
func rasterPaintOrder(ops []layout.Op) []int {
	idx := make([]int, len(ops))
	for i := range ops {
		idx[i] = i
	}

	sort.SliceStable(idx, func(i, j int) bool {
		entryA, bucket := ops[idx[i]], ops[idx[j]]
		sortKey, zIndexB := 0, 0

		if entryA.ZIndexSet {
			sortKey = entryA.ZIndex
		}

		if bucket.ZIndexSet {
			zIndexB = bucket.ZIndex
		}

		if sortKey != zIndexB {
			return sortKey < zIndexB
		}

		if entryA.Positioned != bucket.Positioned {
			return !entryA.Positioned
		}

		la, lb := rasterPaintLayer(entryA.Kind), rasterPaintLayer(bucket.Kind)

		return la < lb
	})

	return idx
}

func rasterPaintLayer(k layout.OpKind) int {
	switch k {
	case layout.OpFillRect, layout.OpStrokeRect, layout.OpLine:
		return 0
	case layout.OpText, layout.OpImage, layout.OpLinkURI, layout.OpBullet:
		return 1
	}

	return 1
}

type decodedRasterImage struct {
	raw   []byte
	jpeg  bool
	image image.Image
}

type scaledRasterImageKey struct {
	decoded       *decodedRasterImage
	width, height int
}

type rasterImageCache struct {
	decoded map[uint64][]*decodedRasterImage
	scaled  map[scaledRasterImageKey]*image.NRGBA
}

func newRasterImageCache() *rasterImageCache {
	return &rasterImageCache{
		decoded: make(map[uint64][]*decodedRasterImage),
		scaled:  make(map[scaledRasterImageKey]*image.NRGBA),
	}
}

func rasterImageHash(data []byte, isJPEG bool) uint64 {
	// FNV-1a is sufficient as a lookup accelerator; bytes.Equal below keeps
	// collisions correct. Include the source kind because PNG and JPEG have
	// different decoders even if their payloads happen to match.
	hash := uint64(fnvOffsetBasis)
	if isJPEG {
		hash ^= 1
		hash *= 1099511628211
	}

	for _, b := range data {
		hash ^= uint64(b)
		hash *= 1099511628211
	}

	return hash
}

func (c *rasterImageCache) decode(paintOp *layout.Op) (*decodedRasterImage, error) {
	key := rasterImageHash(paintOp.Image, paintOp.IsJPEG)
	for _, entry := range c.decoded[key] {
		if entry.jpeg == paintOp.IsJPEG && bytes.Equal(entry.raw, paintOp.Image) {
			return entry, nil
		}
	}

	var (
		src image.Image
		err error
	)

	if paintOp.IsJPEG {
		src, err = jpeg.Decode(bytes.NewReader(paintOp.Image))
	} else {
		src, err = png.Decode(bytes.NewReader(paintOp.Image))
	}

	if err != nil {
		return nil, fmt.Errorf("imageout: decode image: %w", err)
	}
	// PNG decoders may return RGBA for some color types. Normalize once per
	// source so repeated tiles can use the direct NRGBA scaling path without
	// changing the generic decoder semantics.
	if !paintOp.IsJPEG {
		if nrgba, ok := src.(*image.NRGBA); !ok {
			normalized := image.NewNRGBA(src.Bounds())
			draw.Draw(normalized, normalized.Bounds(), src, src.Bounds().Min, draw.Src)
			src = normalized
		} else {
			src = nrgba
		}
	}

	entry := &decodedRasterImage{raw: paintOp.Image, jpeg: paintOp.IsJPEG, image: src}
	c.decoded[key] = append(c.decoded[key], entry)

	return entry, nil
}

func (c *rasterImageCache) scaledImage(src *decodedRasterImage, width, height int) *image.NRGBA {
	key := scaledRasterImageKey{decoded: src, width: width, height: height}
	if scaled, ok := c.scaled[key]; ok {
		return scaled
	}

	scaled := scaleNearest(src.image, width, height)
	c.scaled[key] = scaled

	return scaled
}

// downscaleBox averages factor×factor blocks of src into one output pixel.
func downscaleBox(src *image.NRGBA, factor int) *image.NRGBA {
	if factor <= 1 {
		return src
	}

	if factor == boxFilterFactor2 {
		return downscaleBox2(src)
	}

	srcBounds := src.Bounds()
	dstW := srcBounds.Dx() / factor
	dstH := srcBounds.Dy() / factor

	if dstW < 1 {
		dstW = 1
	}

	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	blockArea := uint32(factor * factor) //nolint:gosec // factor is a small constant (2..rasterSS)

	for row := range dstH {
		for col := range dstW {
			var sumR, sumG, sumB, sumA uint32

			for dy := range factor {
				for dx := range factor {
					srcOffset := src.PixOffset(
						srcBounds.Min.X+col*factor+dx,
						srcBounds.Min.Y+row*factor+dy,
					)
					sumR += uint32(src.Pix[srcOffset])
					sumG += uint32(src.Pix[srcOffset+1])
					sumB += uint32(src.Pix[srcOffset+2])
					sumA += uint32(src.Pix[srcOffset+3])
				}
			}

			dstOffset := dst.PixOffset(col, row)
			dst.Pix[dstOffset] = uint8(sumR / blockArea)   //nolint:gosec // average of byte channels stays in uint8 range
			dst.Pix[dstOffset+1] = uint8(sumG / blockArea) //nolint:gosec // average of byte channels stays in uint8 range
			dst.Pix[dstOffset+2] = uint8(sumB / blockArea) //nolint:gosec // average of byte channels stays in uint8 range
			dst.Pix[dstOffset+3] = uint8(sumA / blockArea) //nolint:gosec // average of byte channels stays in uint8 range
		}
	}

	return dst
}

func downscaleBox2(src *image.NRGBA) *image.NRGBA {
	srcBounds := src.Bounds()
	dstW := srcBounds.Dx() / boxFilterFactor2
	dstH := srcBounds.Dy() / boxFilterFactor2

	if dstW < 1 {
		dstW = 1
	}

	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))

	for y := range dstH {
		srcTop := src.PixOffset(srcBounds.Min.X, srcBounds.Min.Y+y*2)
		srcBottom := src.PixOffset(srcBounds.Min.X, srcBounds.Min.Y+y*2+1)
		dstOffset := dst.PixOffset(0, y)

		for x := range dstW {
			left := x * boxFilterStride
			right := left + boxFilterHalf
			sumR := uint32(src.Pix[srcTop+left])
			sumG := uint32(src.Pix[srcTop+left+1])
			sumB := uint32(src.Pix[srcTop+left+2])
			sumA := uint32(src.Pix[srcTop+left+3])
			sumR += uint32(src.Pix[srcTop+right]) + uint32(src.Pix[srcBottom+left]) + uint32(src.Pix[srcBottom+right])
			sumG += uint32(src.Pix[srcTop+right+1]) + uint32(src.Pix[srcBottom+left+1]) + uint32(src.Pix[srcBottom+right+1])
			sumB += uint32(src.Pix[srcTop+right+2]) + uint32(src.Pix[srcBottom+left+2]) + uint32(src.Pix[srcBottom+right+2])
			sumA += uint32(src.Pix[srcTop+right+3]) + uint32(src.Pix[srcBottom+left+3]) + uint32(src.Pix[srcBottom+right+3])
			dst.Pix[dstOffset] = uint8(sumR / boxFilterArea)   //nolint:gosec // average of 4 byte channels stays in uint8 range
			dst.Pix[dstOffset+1] = uint8(sumG / boxFilterArea) //nolint:gosec // average of 4 byte channels stays in uint8 range
			dst.Pix[dstOffset+2] = uint8(sumB / boxFilterArea) //nolint:gosec // average of 4 byte channels stays in uint8 range
			dst.Pix[dstOffset+3] = uint8(sumA / boxFilterArea) //nolint:gosec // average of 4 byte channels stays in uint8 range
			dstOffset += 4
		}
	}

	return dst
}

// paint draws one display-list operation onto the canvas. pxPerPt converts
// layout points into the (possibly supersampled) canvas pixel space.
//
// Paint semantics (fake-bold gate, stroke min-width) come from layout.StyleOf /
// layout.FakeBoldFor so PDF and raster stay on one table (P5-01). Fill alpha
// deliberately diverges: StyleOf pre-composites translucent fills against white
// (PDF paper); raster keeps raw op.R/G/B/Alpha and draw.Over onto NRGBA so a
// transparent canvas (Transparent) can show through. FakeBold + stroke width
// still follow the shared table.
//
// Page assembly: prologue already shares convert.CollectSheets +
// MergeFontFaces; multi-page PDF assembly remains convert-specific (P5-02).
func paint(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64, atlas *glyphAtlas, imageCache *rasterImageCache) {
	paintStyle := layout.StyleOf(paintOp)

	switch paintOp.Kind {
	case layout.OpFillRect:
		paintFillRect(img, paintOp, pxPerPt)

	case layout.OpStrokeRect:
		paintStrokeRect(img, paintOp, paintStyle, pxPerPt)

	case layout.OpLine:
		paintLine(img, paintOp, paintStyle, pxPerPt)

	case layout.OpText, layout.OpBullet:
		paintText(img, paintOp, pxPerPt, atlas)

	case layout.OpImage:
		paintImage(img, paintOp, pxPerPt, imageCache)

	case layout.OpLinkURI: // annotations do not paint
	}
}

// paintFillRect fills rect with the paintOp color, over-composited unless opaque.
func paintFillRect(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64) {
	// Raw alpha for Over compositing — see paint comment (PDF vs raster).
	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax),
		A: uint8(paintOp.Alpha * channelMax),
	}
	rect := ptRectScale(paintOp.X, paintOp.Y, paintOp.W, paintOp.H, pxPerPt).Intersect(img.Bounds())

	if !rect.Empty() {
		if col.A == opaqueAlpha {
			fillNRGBAOpaque(img, rect, col)
		} else {
			draw.Draw(
				img,
				rect,
				image.NewUniform(col),
				image.Point{}, //nolint:exhaustruct // intentional zero/partial fields
				draw.Over,
			)
		}
	}
}

// paintStrokeRect paints the four border strips of a stroked rectangle.
func paintStrokeRect(img *image.NRGBA, paintOp *layout.Op, paintStyle layout.PaintStyle, pxPerPt float64) {
	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax), A: opaqueAlpha,
	}
	lineWidth := strokeWidthScale(paintStyle.StrokeWidth, pxPerPt)
	rect := ptRectScale(paintOp.X, paintOp.Y, paintOp.W, paintOp.H, pxPerPt)

	rects := [4]image.Rectangle{
		image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+lineWidth),
		image.Rect(rect.Min.X, rect.Max.Y-lineWidth, rect.Max.X, rect.Max.Y),
		image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+lineWidth, rect.Max.Y),
		image.Rect(rect.Max.X-lineWidth, rect.Min.Y, rect.Max.X, rect.Max.Y),
	}
	for _, rr := range rects {
		if rr = rr.Intersect(img.Bounds()); !rr.Empty() {
			fillNRGBAOpaque(img, rr, col)
		}
	}
}

// paintLine paints a horizontal or vertical stroke centred on the paintOp.
func paintLine(img *image.NRGBA, paintOp *layout.Op, paintStyle layout.PaintStyle, pxPerPt float64) {
	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax), A: opaqueAlpha,
	}
	lineWidth := strokeWidthScale(paintStyle.StrokeWidth, pxPerPt)
	// centre the stroke on the line: half its width, in points
	half := float64(lineWidth) / boxFilterFactor2 / pxPerPt

	var rect image.Rectangle

	if paintOp.H <= 0 { // horizontal line
		rect = ptRectScale(paintOp.X, paintOp.Y-half, paintOp.W, boxFilterFactor2*half, pxPerPt)
	} else { // vertical line
		rect = ptRectScale(paintOp.X-half, paintOp.Y, boxFilterFactor2*half, paintOp.H, pxPerPt)
	}

	if rect = rect.Intersect(img.Bounds()); !rect.Empty() {
		fillNRGBAOpaque(img, rect, col)
	}
}

// paintText draws the run (and fake-bold pass) at fractional baselines.
func paintText(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64, atlas *glyphAtlas) {
	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax), A: opaqueAlpha,
	}
	// Keep fractional baselines so glyphs share one stable baseline
	// instead of independently rounded Y positions (bobbing text).
	baseX := paintOp.X * pxPerPt
	baseY := paintOp.Y * pxPerPt

	face := paintOp.Font
	if face == nil {
		// Layout always attaches a face when DefaultFont is available;
		// this is defensive only (no 5×7 bitmap dual path).
		var err error

		face, err = pdf.DefaultFont()
		if err != nil || face == nil {
			return
		}
	}

	ttfDrawString(img, baseX, baseY, paintOp.Text, paintOp.Size, face, col, pxPerPt, atlas)
	// Latin-only fake-bold (CJK gate lives in layout.FakeBoldFor).
	if layout.FakeBoldFor(paintOp) {
		ttfDrawString(img, baseX+float64(rasterSS), baseY, paintOp.Text, paintOp.Size, face, col, pxPerPt, atlas)
	}
}

// paintImage draws a decoded paintOp image, scaled via the per-run cache.
func paintImage(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64, imageCache *rasterImageCache) {
	decoded, err := imageCache.decode(paintOp)
	if err != nil || paintOp.W <= 0 || paintOp.H <= 0 {
		return // layout already validated the bytes; skip on failure
	}

	src := decoded.image
	rect := ptRectScale(paintOp.X, paintOp.Y, paintOp.W, paintOp.H, pxPerPt).Intersect(img.Bounds())

	if rect.Empty() {
		return
	}

	sb := src.Bounds()
	if rect.Dx() == sb.Dx() && rect.Dy() == sb.Dy() {
		if nrgba, ok := src.(*image.NRGBA); ok && nrgba.Opaque() {
			drawNRGBAOpaque(img, rect, nrgba, sb.Min)
		} else {
			draw.Draw(img, rect, src, sb.Min, draw.Over)
		}

		return
	}

	// Go 1.26 removed image/draw's scalers; nearest
	// neighbour keeps it stdlib-only.
	scaled := imageCache.scaledImage(decoded, rect.Dx(), rect.Dy())
	if scaled.Opaque() {
		drawNRGBAOpaque(img, rect, scaled, image.Point{}) //nolint:exhaustruct // intentional zero/partial fields
	} else {
		draw.Draw(img, rect, scaled, image.Point{}, draw.Over) //nolint:exhaustruct // intentional zero/partial fields
	}
}

// strokeWidthScale returns the stroke thickness in canvas pixels from a
// StyleOf-resolved width (already min-clamped to 1 when non-positive).
func strokeWidthScale(strokeWidth float64, pxPerPt float64) int {
	w := strokeWidth
	if w <= 0 {
		w = 1
	}

	if lw := int(math.Round(w * pxPerPt)); lw >= 1 {
		return lw
	}

	return 1
}

// ptRectScale converts a point-space rectangle into canvas pixels.
func ptRectScale(x, y, w, h, pxPerPt float64) image.Rectangle {
	return image.Rect(
		int(math.Round(x*pxPerPt)), int(math.Round(y*pxPerPt)),
		int(math.Round((x+w)*pxPerPt)), int(math.Round((y+h)*pxPerPt)),
	)
}

func fillNRGBAOpaque(dst *image.NRGBA, rect image.Rectangle, col color.NRGBA) {
	if rect.Empty() {
		return
	}

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		offset := dst.PixOffset(rect.Min.X, y)

		for x := rect.Min.X; x < rect.Max.X; x++ {
			dst.Pix[offset] = col.R
			dst.Pix[offset+1] = col.G
			dst.Pix[offset+2] = col.B
			dst.Pix[offset+3] = 255
			offset += 4
		}
	}
}

func drawNRGBAOpaque(dst *image.NRGBA, rect image.Rectangle, src *image.NRGBA, sp image.Point) {
	for y := range rect.Dy() {
		dstOffset := dst.PixOffset(rect.Min.X, rect.Min.Y+y)
		srcOffset := src.PixOffset(sp.X, sp.Y+y)
		copy(dst.Pix[dstOffset:dstOffset+4*rect.Dx()], src.Pix[srcOffset:srcOffset+4*rect.Dx()])
	}
}

// scaleNearest resizes src to w×h with nearest-neighbour sampling. Go 1.26
// removed image/draw's BiLinear/NearestNeighbor scalers, so a tiny scaler
// lives here; natural-size images take the draw.Draw fast path in paint.
func scaleNearest(src image.Image, w, h int) *image.NRGBA {
	if nrgba, ok := src.(*image.NRGBA); ok {
		return scaleNearestNRGBA(nrgba, w, h)
	}

	return scaleNearestGeneric(src, w, h)
}

func scaleNearestNRGBA(src *image.NRGBA, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()

	if srcBounds.Dx() == 0 || srcBounds.Dy() == 0 {
		return dst
	}

	scaleX := float64(srcBounds.Dx()) / float64(width)

	scaleY := float64(srcBounds.Dy()) / float64(height)
	for row := range height {
		srcY := srcBounds.Min.Y + int((float64(row)+pixelCenter)*scaleY)
		if srcY > srcBounds.Max.Y-1 {
			srcY = srcBounds.Max.Y - 1
		}

		for col := range width {
			srcX := srcBounds.Min.X + int((float64(col)+pixelCenter)*scaleX)
			if srcX > srcBounds.Max.X-1 {
				srcX = srcBounds.Max.X - 1
			}

			srcOffset := src.PixOffset(srcX, srcY)
			dstOffset := dst.PixOffset(col, row)
			copy(dst.Pix[dstOffset:dstOffset+4], src.Pix[srcOffset:srcOffset+4])
		}
	}

	return dst
}

func scaleNearestGeneric(src image.Image, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()

	if srcBounds.Dx() == 0 || srcBounds.Dy() == 0 {
		return dst
	}

	scaleX := float64(srcBounds.Dx()) / float64(width)

	scaleY := float64(srcBounds.Dy()) / float64(height)
	for row := range height {
		srcY := srcBounds.Min.Y + int((float64(row)+pixelCenter)*scaleY)
		if srcY > srcBounds.Max.Y-1 {
			srcY = srcBounds.Max.Y - 1
		}

		for col := range width {
			srcX := srcBounds.Min.X + int((float64(col)+pixelCenter)*scaleX)
			if srcX > srcBounds.Max.X-1 {
				srcX = srcBounds.Max.X - 1
			}

			nc, ok := color.NRGBAModel.Convert(src.At(srcX, srcY)).(color.NRGBA)
			if !ok {
				continue
			}

			dst.SetNRGBA(col, row, nc)
		}
	}

	return dst
}

// Run is the CLI-facing adapter (P1-1): opens the output sink, builds a
// convert.Request, and delegates to RunRequest. Existing cmd/tests keep this
// signature; the engine no longer depends on *cli.Command internals beyond
// OpenOutput and output-path format sniffing.
func Run(ctx context.Context, cmd *cli.Command, log io.Writer) error {
	if cmd == nil {
		return errNilCommand
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("imageout: open output: %w", err)
	}

	img := cmd.Image
	// Resolve --format / extension before the engine so Request only needs
	// Image.Format (library callers set Format explicitly or get PNG).
	format, err := resolveFormat(img.Format, cmd.Output)
	if err != nil {
		_ = closeOut()

		return err
	}

	img.Format = format
	req := convert.NewImageRequest(cmd.Global, img, cmd.Objects, out)

	runErr := RunRequest(ctx, req, log)
	if closeErr := closeOut(); closeErr != nil && runErr == nil {
		return closeErr
	}

	return runErr
}

// RunRequest drives image conversion from a CLI-independent convert.Request
// (P1-1). req.Image must be non-nil; req.Output receives encoded PNG/JPEG
// bytes (nil → os.Stdout is not auto-selected; callers must supply a writer).
func RunRequest(ctx context.Context, req *convert.Request, log io.Writer) error {
	if req == nil {
		return errNilRequest
	}

	if err := req.ValidateImage(); err != nil {
		return fmt.Errorf("imageout: validate: %w", err)
	}

	if log == nil {
		log = io.Discard
	}
	// Policy A: one quiet bit — CLI --quiet sets Global.Quiet (not Image.Quiet).
	if req.Global.Quiet {
		log = io.Discard
	}

	obj, err := firstObject(req.Objects, log)
	if err != nil {
		return err
	}

	// P2-07: full load policy at construction. Image.Load holds image-mode
	// Proxy; CLI/library ACL (--allow / --enable-local-file-access) lives on
	// Global.Load. Merge so NewLoader applies everything; no post-construction
	// field pokes on Loader.
	loader := load.NewLoader(imageLoadGlobal(req.Global, *req.Image))
	loader.Log = log

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}

	registry := fontRegistry(req.Global, log)

	return renderRequest(ctx, req, obj, loader, font, registry, log)
}

// renderRequest renders the image object and writes the encoded bytes to
// req.Output; the loader/font/registry prelude lives in RunRequest.
func renderRequest(
	ctx context.Context,
	req *convert.Request,
	obj *settings.PdfObject,
	loader *load.Loader,
	font *pdf.Font,
	registry *pdf.Registry,
	log io.Writer,
) error {
	imgSet := req.Image

	prep, media, err := prepareImageDocument(ctx, loader, obj, req.Global, imgSet, registry, log)
	if err != nil {
		return err
	}

	root := prep.Root
	sheets := prep.Sheets
	registry = prep.Registry

	cache := map[string][]byte{}
	imagesFn := makeImageFetcher(ctx, imgSet, prep, cache)

	// Policy A: Quiet is Global.Quiet; body paint background is Global.Background
	// only (single field for PDF + image; CLI --background / library Set).
	img, err := RenderContext(ctx, root, RenderOptions{
		Width:              imgSet.Width,
		Height:             imgSet.Height,
		Font:               font,
		Registry:           registry,
		Sheets:             sheets,
		Media:              media,
		Images:             imagesFn,
		Background:         req.Global.Background,
		Transparent:        imgSet.Transparent,
		Crop:               cropRect(imgSet.Crop),
		SmartWidth:         imgSet.SmartWidth,
		PrintLinkUnderline: imgSet.Web.PrintLinkUnderline || req.Global.Web.PrintLinkUnderline || obj.Web.PrintLinkUnderline,
	})
	if err != nil {
		return err
	}

	return writeEncodedOutput(req, img, log)
}

// writeEncodedOutput resolves the format, composites onto white for
// transparent JPEG, and writes the encoded bytes to req.Output.
func writeEncodedOutput(req *convert.Request, img image.Image, log io.Writer) error {
	imgSet := req.Image

	format, err := resolveFormat(imgSet.Format, "")
	if err != nil {
		return err
	}

	if format == formatJPG && imgSet.Transparent {
		line.Emit(log, line.Warn, "--transparent ignored for JPEG output (white background used)")

		img = onWhite(img)
	}

	data, err := encode(img, format, imgSet.Quality)
	if err != nil {
		return fmt.Errorf("encode %s: %w", format, err)
	}

	out := req.Output
	if out == nil {
		return errNilOutput
	}

	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// prepareImageDocument resolves the media/SimplifyDOM profile and runs
// PrepareDocument for image mode (single page).
func prepareImageDocument(
	ctx context.Context,
	loader *load.Loader,
	obj *settings.PdfObject,
	global settings.PdfGlobal,
	imgSet *settings.ImageGlobal,
	registry *pdf.Registry,
	log io.Writer,
) (*convert.PreparedDocument, string, error) {
	media := mediaFor(global, *imgSet, obj)
	enabled := convert.SimplifyDOMEnabled(imgSet.Web, obj.Web) || global.Web.SimplifyDOM

	profile := convert.SimplifyDOMProfile(imgSet.Web, obj.Web)
	if profile == "" {
		profile = convert.SimplifyDOMProfile(
			global.Web,
			settings.Web{}, //nolint:exhaustruct // intentional zero/partial fields
		)
	}

	prep, err := convert.PrepareDocument(ctx, loader, obj.Page, obj.Load, registry, convert.PrepareOptions{
		ViewportW:       defaultViewportW,
		ViewportH:       defaultViewportH,
		MediaType:       media,
		SimplifyDOM:     enabled,
		SimplifyProfile: profile,
		ObjectIndex:     1,
	}, log)
	if err != nil {
		return nil, "", fmt.Errorf("imageout: prepare: %w", err)
	}

	if prep.Resource.Skip {
		return nil, "", fmt.Errorf("load %q: %w", obj.Page, errNothingToRender)
	}

	return prep, media, nil
}

// makeImageFetcher wraps prep.Resources.Fetch with a per-run byte cache and
// the --no-images gate.
func makeImageFetcher(
	ctx context.Context,
	imgSet *settings.ImageGlobal,
	prep *convert.PreparedDocument,
	cache map[string][]byte,
) func(string) ([]byte, error) {
	return func(src string) ([]byte, error) {
		if !imgSet.Web.Images {
			return nil, errImagesDisabled
		}

		if b, ok := cache[src]; ok {
			return b, nil
		}

		res, err := prep.Resources.Fetch(ctx, src)
		if err != nil {
			return nil, fmt.Errorf("fetch %q: %w", src, err)
		}

		cache[src] = res.Body

		return res.Body, nil
	}
}

// fontRegistry builds a font registry from Global.FontPaths and system font
// dirs, logging what was scanned (nil when nothing to scan).
func fontRegistry(global settings.PdfGlobal, log io.Writer) *pdf.Registry {
	if len(global.FontPaths) == 0 && !global.UseSystemFonts {
		return nil
	}

	scan := append([]string{}, global.FontPaths...)
	if global.UseSystemFonts {
		scan = append(scan, pdf.DefaultSystemFontDirs()...)
	}

	if log != io.Discard && len(scan) > 0 {
		line.Emit(log, line.Info, "scanned %d font path(s)", len(scan))
	}

	return pdf.ScanFontDirs(scan)
}

// imageLoadGlobal builds the LoadGlobal for image mode: Proxy from Image.Load
// plus ACL (Allow / EnableLocalFileAccess) from Global.Load, where CLI and
// ImageConverter.Global set them. NewLoader applies the full policy.
func imageLoadGlobal(global settings.PdfGlobal, image settings.ImageGlobal) settings.LoadGlobal {
	loadGlobal := image.Load
	// Global.Load is the ACL home (enablelocalfileaccess / allow keys).
	// Prefer Global when set; keep any Image.Load ACL already present (OR).
	if len(global.Load.Allow) > 0 {
		loadGlobal.Allow = append(append([]string(nil), loadGlobal.Allow...), global.Load.Allow...)
	}

	loadGlobal.EnableLocalFileAccess = loadGlobal.EnableLocalFileAccess || global.Load.EnableLocalFileAccess

	return loadGlobal
}

// imageLoadGlobalCmd is a test helper: ACL merge from a parsed Command.
func imageLoadGlobalCmd(cmd *cli.Command) settings.LoadGlobal {
	return imageLoadGlobal(cmd.Global, cmd.Image)
}

// firstObject returns the first page-like object. Image mode renders a
// single page; extra page objects and TOC objects are ignored with warnings.
// An empty Page is accepted when Load.InlineHTML is set (P2-04 in-memory
// source kind via ObjectSettings.SetBody).
func firstObject(objects []settings.PdfObject, log io.Writer) (*settings.PdfObject, error) {
	var first *settings.PdfObject

	for idx := range objects {
		obj := &objects[idx]
		if obj.IsTableOfContent {
			continue
		}

		if obj.Page == "" && len(obj.Load.InlineHTML) == 0 {
			continue
		}

		if first == nil {
			first = obj

			continue
		}

		line.Emit(log, line.Warn, "image mode renders the first page only; ignoring object %d", idx+1)
	}

	if first == nil {
		return nil, errNoInputToConvert
	}

	return first, nil
}

// mediaFor resolves the layout media via settings.ResolveMedia (P1-4).
// Image mode defaults to "screen". CLI --print-media-type lands on
// Global.Web; object overrides live on LoadPage and are mapped into a
// temporary Web view. Image.Web is merged so library Set paths still apply.
func mediaFor(global settings.PdfGlobal, image settings.ImageGlobal, obj *settings.PdfObject) string {
	web := image.Web
	if global.Web.PrintMediaType {
		web.PrintMediaType = true
	}

	if web.MediaType == settings.MediaIgnore {
		web.MediaType = global.Web.MediaType
	}

	var objWeb *settings.Web

	if obj != nil {
		web := settings.Web{ //nolint:exhaustruct // intentional zero/partial fields
			PrintMediaType: obj.Load.PrintMediaType || obj.Web.PrintMediaType,
			MediaType:      obj.Load.MediaType,
		}
		if obj.Web.MediaType != settings.MediaIgnore {
			web.MediaType = obj.Web.MediaType
		}

		objWeb = &web
	}

	return settings.ResolveMedia("screen", web, objWeb)
}

// cropRect converts image settings into a pixel rectangle; returns the zero
// rectangle (no crop) when any value is unset (defaults are -1).
func cropRect(c settings.CropSettings) image.Rectangle {
	if c.Left < 0 || c.Top < 0 || c.Width <= 0 || c.Height <= 0 {
		return image.Rectangle{} //nolint:exhaustruct // intentional zero/partial fields
	}

	return image.Rect(c.Left, c.Top, c.Left+c.Width, c.Top+c.Height)
}

// resolveFormat picks the output format: the --format flag wins, otherwise
// the output file extension (.jpg/.jpeg), otherwise PNG.
func resolveFormat(flag, output string) (string, error) {
	fmtName := strings.ToLower(strings.TrimSpace(flag))
	if fmtName == "" {
		switch strings.ToLower(filepath.Ext(output)) {
		case ".jpg", ".jpeg":
			fmtName = formatJPG
		default:
			fmtName = formatPNG
		}
	}

	switch fmtName {
	case formatPNG:
		return formatPNG, nil
	case formatJPG, "jpeg":
		return formatJPG, nil
	}

	return "", fmt.Errorf("%w %q (supported: png, jpg)", errUnsupportedFmt, flag)
}

// encode serializes img as PNG or JPEG. quality applies to JPEG only
// (1..100); PNG is lossless and ignores it.
func encode(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer

	switch format {
	case formatPNG:
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("png encode: %w", err)
		}
	case formatJPG:
		if quality < 1 {
			quality = 1
		}

		if quality > qualityMaxPercent {
			quality = 100
		}

		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("jpeg encode: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w %q", errUnsupportedFmt, format)
	}

	return buf.Bytes(), nil
}

// onWhite composites img onto a white background (JPEG has no alpha).
func onWhite(img image.Image) image.Image {
	dst := image.NewNRGBA(img.Bounds())
	draw.Draw(
		dst,
		dst.Bounds(),
		image.NewUniform(color.White),
		image.Point{}, //nolint:exhaustruct // intentional zero/partial fields
		draw.Src,
	)
	draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Over)

	return dst
}
