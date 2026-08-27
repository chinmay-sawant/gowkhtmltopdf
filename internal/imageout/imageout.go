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
	"strings"
	"sync"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	renderpipeline "github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/render"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const (
	channelMax         = 255
	fnvOffsetBasis     = 14695981039346656037
	boxFilterFactor2   = 2
	boxFilterStride    = 8
	boxFilterHalf      = 4
	boxFilterArea      = 4 // 2x2 block of pixels (boxFilterFactor2 squared)
	pixelCenter        = 0.5
	defaultViewportW   = 768.0
	defaultViewportH   = 576.0
	qualityMaxPercent  = 100
	opaqueAlpha        = 255
	formatPNG          = "png"
	formatJPG          = "jpg"
	maxRasterWidth     = 16_384
	maxRasterHeight    = 16_384
	maxRasterPixels    = 64 * 1024 * 1024
	maxRasterBytes     = 256 << 20
	maxImageDimension  = 16_384
	maxImagePixels     = 16 * 1024 * 1024
	maxImageEncoded    = 32 << 20
	maxImageDecoded    = 128 << 20
	imageDecodeBPP     = 8 // decoder and normalized/scaled working-set estimate
	maxImageFetches    = 64
	maxImageFetchBytes = 32 << 20
)

var (
	errNilRoot         = errors.New("imageout: nil root")
	errNilContext      = errs.ErrNilContext
	errCropNoIntersect = errors.New("imageout: crop rectangle does not intersect the canvas")
	errNilRequest      = errs.ErrNilRequest
	errNothingToRender = errors.New("load-error policy is skip; nothing to render")
	errImagesDisabled  = errs.ErrImagesDisabled
	errNilOutput       = ErrMissingOutput
	errUnsupportedFmt  = errors.New("unsupported format")
	errRasterTooLarge  = errors.New("imageout: raster exceeds resource budget")
	errImageTooLarge   = errors.New("imageout: image exceeds resource budget")
	errEncodedTooLarge = errors.New("imageout: encoded image exceeds resource budget")
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

// maxSmartWidthLayouts bounds complete layout passes for one image render.
// The final layout result is returned when content still exceeds the cap.
const maxSmartWidthLayouts = 8

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
// retained as the source-compatible background-context adapter. It rejects
// nil contexts at the cancellation-aware boundary.
func RenderContext(ctx context.Context, root *html.Node, opts RenderOptions) (image.Image, error) {
	if root == nil {
		return nil, errNilRoot
	}

	if ctx == nil {
		return nil, errNilContext
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
	return reOrigin(img.SubImage(inter))
}

// reOrigin copies src into a fresh image whose bounds start at (0,0).
func reOrigin(src image.Image) (*image.NRGBA, error) {
	srcBounds := src.Bounds()

	dst, err := newRasterImage(srcBounds.Dx(), srcBounds.Dy())
	if err != nil {
		return nil, err
	}

	draw.Draw(dst, dst.Bounds(), src, srcBounds.Min, draw.Src)

	return dst, nil
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
// max op.X+op.W). Growth is capped at maxSmartViewport pixels and at
// maxSmartWidthLayouts complete layout passes. The latter makes the bounded
// fallback explicit without changing the normal fitting result.
func layoutSmartWidth(
	ctx context.Context, root *html.Node, opts RenderOptions, font *pdf.Font,
) (*layout.Result, error) {
	viewport := float64(opts.Width)
	if viewport <= 0 {
		viewport = screenWidthDefault
	}

	var res *layout.Result

	for range maxSmartWidthLayouts {
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
type pixBuffer struct {
	b []byte
}

//nolint:gochecknoglobals // supersample pixel buffer recycling
var supersamplePixPool sync.Pool

//nolint:cyclop,funlen,mnd // supersampled rasterization pipeline
func rasterizeContext(ctx context.Context, res *layout.Result, height float64, transparent bool) (*image.NRGBA, error) {
	pxPerPt := ptToPx * float64(rasterSS)

	widthPx, err := rasterDimension(res.Width*pxPerPt, maxRasterWidth)
	if err != nil {
		return nil, err
	}

	heightPx, err := rasterDimension(height*pxPerPt, maxRasterHeight)
	if err != nil {
		return nil, err
	}

	if err := validateRasterSize(widthPx, heightPx); err != nil {
		return nil, err
	}

	neededBytes := widthPx * heightPx * 4
	pBuf, ok := supersamplePixPool.Get().(*pixBuffer)

	if !ok || pBuf == nil {
		pBuf = &pixBuffer{b: make([]byte, 0, neededBytes)}
	}

	if cap(pBuf.b) < neededBytes {
		pBuf.b = make([]byte, neededBytes)
	} else {
		pBuf.b = pBuf.b[:neededBytes]
		clear(pBuf.b)
	}

	defer supersamplePixPool.Put(pBuf)

	img := &image.NRGBA{
		Pix:    pBuf.b,
		Stride: widthPx * 4,
		Rect:   image.Rect(0, 0, widthPx, heightPx),
	}

	if !transparent {
		fillNRGBAOpaque(img, img.Bounds(), color.NRGBA{R: channelMax, G: channelMax, B: channelMax, A: opaqueAlpha})
	}

	atlas := newGlyphAtlas()
	imageCache := newRasterImageCache()

	for _, opIndex := range rasterPaintOrder(res.Ops) {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("imageout: context: %w", err)
		}

		paint(img, &res.Ops[opIndex], pxPerPt, atlas, imageCache)
	}

	if rasterSS <= 1 {
		outImg := image.NewNRGBA(image.Rect(0, 0, widthPx, heightPx))
		copy(outImg.Pix, img.Pix)

		return outImg, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("imageout: context: %w", err)
	}

	downscaled := downscaleBox(img, rasterSS)
	if downscaled == nil {
		return nil, errRasterTooLarge
	}

	return downscaled, nil
}

func rasterDimension(value float64, maxVal int) (int, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%w: non-finite dimension", errRasterTooLarge)
	}

	if value <= 0 {
		return 1, nil
	}

	rounded := math.Round(value)
	if rounded < 1 {
		return 1, nil
	}

	if rounded > float64(maxVal) {
		return 0, fmt.Errorf("%w: dimension %.0f exceeds %d", errRasterTooLarge, rounded, maxVal)
	}

	return int(rounded), nil
}

func validateRasterSize(width, height int) error {
	if width < 1 || height < 1 || width > maxRasterWidth || height > maxRasterHeight {
		return fmt.Errorf(
			"%w: dimensions %dx%d exceed %dx%d",
			errRasterTooLarge,
			width,
			height,
			maxRasterWidth,
			maxRasterHeight,
		)
	}

	pixels := int64(width) * int64(height)
	if pixels > maxRasterPixels || pixels*4 > maxRasterBytes {
		return fmt.Errorf("%w: %d pixels exceed budget", errRasterTooLarge, pixels)
	}

	return nil
}

func newRasterImage(width, height int) (*image.NRGBA, error) {
	if err := validateRasterSize(width, height); err != nil {
		return nil, err
	}

	return image.NewNRGBA(image.Rect(0, 0, width, height)), nil
}

func validateImageInput(data []byte, isJPEG bool) (int, int, error) {
	if len(data) > maxImageEncoded {
		return 0, 0, fmt.Errorf("%w: %d bytes, limit %d", errEncodedTooLarge, len(data), maxImageEncoded)
	}

	var (
		cfg image.Config
		err error
	)

	if isJPEG {
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(data))
	} else {
		cfg, err = png.DecodeConfig(bytes.NewReader(data))
	}

	if err != nil {
		return 0, 0, fmt.Errorf("imageout: decode config: %w", err)
	}

	width, height := cfg.Width, cfg.Height
	if width <= 0 || height <= 0 || width > maxImageDimension || height > maxImageDimension {
		return 0, 0, fmt.Errorf(
			"%w: dimensions %dx%d exceed %d",
			errImageTooLarge,
			width,
			height,
			maxImageDimension,
		)
	}

	pixels := int64(width) * int64(height)
	if pixels > maxImagePixels || pixels*imageDecodeBPP > maxImageDecoded {
		return 0, 0, fmt.Errorf("%w: %d pixels exceed budget", errImageTooLarge, pixels)
	}

	return width, height, nil
}

// rasterPaintOrder delegates to layout's canonical display-list policy:
// z-index first, then chrome below content, with stable source order last.
// Keeping this adapter as a wrapper preserves the package-local test seam
// without maintaining a second comparator.
// FIX-REVIEW: PAINT-01 PDF body/header/footer traversal remains owned by
// internal/layout.Paint and PaintBand; this package consumes the same ordering
// and StyleOf policy without duplicating pagination or annotation semantics.
func rasterPaintOrder(ops []layout.Op) []int {
	return layout.PaintOrder(ops)
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
	decoded            map[uint64][]*decodedRasterImage
	decodedRawBytes    int64
	decodedMemoryBytes int64
	decodedEntries     int
	scaled             map[scaledRasterImageKey]*image.NRGBA
	scaledBytes        int64
	scaledEntries      int
}

const (
	maxDecodedCacheEntries     = 64
	maxDecodedCacheRawBytes    = 32 << 20
	maxDecodedCacheMemoryBytes = 64 << 20
	maxScaledCacheEntries      = 128
	maxScaledCacheBytes        = 64 << 20
)

func newRasterImageCache() *rasterImageCache {
	return &rasterImageCache{ //nolint:exhaustruct // intentional zero fields
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

//nolint:cyclop // raster image decoding pipeline
func (c *rasterImageCache) decode(paintOp *layout.Op) (*decodedRasterImage, error) {
	if _, _, err := validateImageInput(paintOp.Image, paintOp.IsJPEG); err != nil {
		return nil, err
	}

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
			bounds := src.Bounds()

			normalized, normalizeErr := newRasterImage(bounds.Dx(), bounds.Dy())
			if normalizeErr != nil {
				return nil, normalizeErr
			}

			draw.Draw(normalized, normalized.Bounds(), src, src.Bounds().Min, draw.Src)
			src = normalized
		} else {
			src = nrgba
		}
	}

	entry := &decodedRasterImage{raw: paintOp.Image, jpeg: paintOp.IsJPEG, image: src}
	decodedMemory := decodedImageMemory(src)

	if c.decodedEntries < maxDecodedCacheEntries &&
		c.decodedRawBytes+int64(len(paintOp.Image)) <= maxDecodedCacheRawBytes &&
		c.decodedMemoryBytes+decodedMemory <= maxDecodedCacheMemoryBytes {
		c.decoded[key] = append(c.decoded[key], entry)
		c.decodedEntries++
		c.decodedRawBytes += int64(len(paintOp.Image))
		c.decodedMemoryBytes += decodedMemory
	}

	return entry, nil
}

func decodedImageMemory(src image.Image) int64 {
	bounds := src.Bounds()

	return int64(bounds.Dx()) * int64(bounds.Dy()) * imageDecodeBPP
}

func (c *rasterImageCache) scaledImage(src *decodedRasterImage, width, height int) *image.NRGBA {
	key := scaledRasterImageKey{decoded: src, width: width, height: height}
	if scaled, ok := c.scaled[key]; ok {
		return scaled
	}

	scaled := scaleNearest(src.image, width, height)
	if scaled == nil {
		return nil
	}

	if c.scaledEntries < maxScaledCacheEntries && c.scaledBytes+int64(len(scaled.Pix)) <= maxScaledCacheBytes {
		c.scaled[key] = scaled
		c.scaledEntries++
		c.scaledBytes += int64(len(scaled.Pix))
	}

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

	dst, err := newRasterImage(dstW, dstH)
	if err != nil {
		return nil
	}

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

	dst, err := newRasterImage(dstW, dstH)
	if err != nil {
		return nil
	}

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
// Page assembly: prologue already shares prepare.CollectSheets +
// MergeFontFaces; multi-page PDF assembly remains convert-specific (P5-02).
//
//nolint:cyclop // raster op paint dispatcher
func paint(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64, atlas *glyphAtlas, imageCache *rasterImageCache) {
	if paintOp == nil || paintOp.Kind == layout.OpLinkURI {
		return
	}

	if paintOp.XformSet && !paintOp.Xform.IsIdentity() {
		paintTransformedOp(img, paintOp, pxPerPt, atlas, imageCache)

		return
	}

	opCopy := *paintOp
	if opCopy.TextTransform != "" {
		opCopy.Text = layout.TransformInlineText(opCopy.Text, opCopy.TextTransform)
	}

	if opCopy.PaintOpacity > 0 && opCopy.PaintOpacity < 1 {
		alpha := opCopy.Alpha
		if alpha <= 0 || alpha > 1 {
			alpha = 1
		}

		opCopy.Alpha = alpha * opCopy.PaintOpacity
	}

	paintStyle := layout.StyleOf(&opCopy)

	switch opCopy.Kind {
	case layout.OpFillRect:
		paintFillRect(img, &opCopy, pxPerPt)

	case layout.OpStrokeRect:
		paintStrokeRect(img, &opCopy, paintStyle, pxPerPt)

	case layout.OpLine:
		paintLine(img, &opCopy, paintStyle, pxPerPt)

	case layout.OpText, layout.OpBullet:
		paintText(img, &opCopy, pxPerPt, atlas)

	case layout.OpImage:
		paintImage(img, &opCopy, pxPerPt, imageCache)

	case layout.OpLinkURI: // annotations do not paint
	}
}

// paintTransformedOp renders an op with CSS 2D affine transform onto img.
//
//nolint:cyclop,varnamelen,mnd,wsl,gosec,funlen // affine raster op transform and pixel compositing
func paintTransformedOp(
	img *image.NRGBA, op *layout.Op, pxPerPt float64, atlas *glyphAtlas, imageCache *rasterImageCache,
) {
	opX, opY, opW, opH := op.X, op.Y, op.W, op.H
	if op.Kind == layout.OpText || op.Kind == layout.OpBullet {
		opY -= op.Size
		opH = op.Size * 1.5
		if opW <= 0 {
			opW = op.Size * float64(len(op.Text))
		}
	}

	if opW <= 0 {
		opW = 1
	}

	if opH <= 0 {
		opH = 1
	}

	corners := [4][2]float64{
		{opX, opY},
		{opX + opW, opY},
		{opX + opW, opY + opH},
		{opX, opY + opH},
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, c := range corners {
		tx, ty := op.Xform.Apply(c[0], c[1])
		minX = math.Min(minX, tx)
		minY = math.Min(minY, ty)
		maxX = math.Max(maxX, tx)
		maxY = math.Max(maxY, ty)
	}

	dstRect := ptRectScale(minX-2, minY-2, (maxX-minX)+4, (maxY-minY)+4, pxPerPt).Intersect(img.Bounds())
	if dstRect.Empty() {
		return
	}

	srcRect := ptRectScale(opX-2, opY-2, opW+4, opH+4, pxPerPt)
	if srcRect.Dx() <= 0 || srcRect.Dy() <= 0 {
		return
	}

	subImg := image.NewNRGBA(srcRect)

	untransformed := *op
	untransformed.XformSet = false
	paint(subImg, &untransformed, pxPerPt, atlas, imageCache)

	det := op.Xform.A*op.Xform.D - op.Xform.B*op.Xform.C
	if math.Abs(det) < 1e-12 {
		return
	}

	invA := op.Xform.D / det
	invB := -op.Xform.B / det
	invC := -op.Xform.C / det
	invD := op.Xform.A / det
	invE := (op.Xform.C*op.Xform.F - op.Xform.D*op.Xform.E) / det
	invF := (op.Xform.B*op.Xform.E - op.Xform.A*op.Xform.F) / det

	for dstY := dstRect.Min.Y; dstY < dstRect.Max.Y; dstY++ {
		ptY := float64(dstY) / pxPerPt

		for dstX := dstRect.Min.X; dstX < dstRect.Max.X; dstX++ {
			ptX := float64(dstX) / pxPerPt

			srcPtX := invA*ptX + invC*ptY + invE
			srcPtY := invB*ptX + invD*ptY + invF

			srcX := int(math.Round(srcPtX * pxPerPt))
			srcY := int(math.Round(srcPtY * pxPerPt))

			if image.Pt(srcX, srcY).In(subImg.Bounds()) {
				srcPixOff := subImg.PixOffset(srcX, srcY)
				srcA := uint32(subImg.Pix[srcPixOff+3])

				if srcA == 0 {
					continue
				}

				dstPixOff := img.PixOffset(dstX, dstY)
				dstR := uint32(img.Pix[dstPixOff+0])
				dstG := uint32(img.Pix[dstPixOff+1])
				dstB := uint32(img.Pix[dstPixOff+2])
				dstA := uint32(img.Pix[dstPixOff+3])

				invAlpha := channelMax - srcA
				img.Pix[dstPixOff+0] = uint8((uint32(subImg.Pix[srcPixOff+0])*srcA + dstR*invAlpha) / channelMax)
				img.Pix[dstPixOff+1] = uint8((uint32(subImg.Pix[srcPixOff+1])*srcA + dstG*invAlpha) / channelMax)
				img.Pix[dstPixOff+2] = uint8((uint32(subImg.Pix[srcPixOff+2])*srcA + dstB*invAlpha) / channelMax)
				img.Pix[dstPixOff+3] = uint8((srcA*channelMax + dstA*invAlpha) / channelMax)
			}
		}
	}
}

// paintFillRect fills rect with the paintOp color, over-composited unless opaque.
func paintFillRect(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64) {
	// Raw alpha for Over compositing — see paint comment (PDF vs raster).
	alpha := paintOp.Alpha
	if alpha <= 0 && paintOp.PaintOpacity == 0 {
		alpha = 1
	}

	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax),
		A: uint8(math.Round(alpha * channelMax)),
	}
	rect := ptRectScale(paintOp.X, paintOp.Y, paintOp.W, paintOp.H, pxPerPt).Intersect(img.Bounds())

	if !rect.Empty() {
		if paintOp.Radius > 0 || paintOp.RadiusTopLeft > 0 || paintOp.RadiusTopRight > 0 ||
			paintOp.RadiusBottomRight > 0 || paintOp.RadiusBottomLeft > 0 {
			paintRoundedFill(img, rect, paintOp, pxPerPt, col)

			return
		}

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

func paintRoundedFill(
	img *image.NRGBA,
	rect image.Rectangle,
	paintOp *layout.Op,
	pxPerPt float64,
	col color.NRGBA,
) {
	radiusX, radiusY := scaledRadiiXY(paintOp, pxPerPt)
	mask := image.NewAlpha(rect)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if roundedContains(float64(x)+pixelCenter, float64(y)+pixelCenter, float64(rect.Min.X),
				float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), radiusX, radiusY) {
				mask.SetAlpha(x, y, color.Alpha{A: opaqueAlpha})
			}
		}
	}

	draw.DrawMask(img, rect, image.NewUniform(col), image.Point{X: 0, Y: 0}, mask, rect.Min, draw.Over)
}

//nolint:varnamelen,mnd // raster geometry mirrors the compact PDF path
func paintRoundedTopStroke(
	img *image.NRGBA,
	paintOp *layout.Op,
	col color.NRGBA,
	lineWidth int,
	pxPerPt float64,
) {
	rx, ry := scaledRadiiXY(paintOp, pxPerPt)
	x := paintOp.X * pxPerPt
	y := paintOp.Y * pxPerPt
	w := paintOp.W * pxPerPt
	strokeInset := float64(lineWidth) / boxFilterFactor2
	leftRX := max(rx[0]-strokeInset, 0)
	leftRY := max(ry[0]-strokeInset, 0)
	rightRX := max(rx[1]-strokeInset, 0)
	rightRY := max(ry[1]-strokeInset, 0)

	points := make([]rasterPoint, 0, roundedArcSteps*2+3) //nolint:mnd // two arcs plus their joins
	points = append(points, rasterPoint{X: x, Y: y + leftRY})
	points = appendArc(points, x+leftRX, y+leftRY, leftRX, leftRY, math.Pi, 1.5*math.Pi)
	points = append(points, rasterPoint{X: x + w - rightRX, Y: y})
	points = appendArc(points, x+w-rightRX, y+rightRY, rightRX, rightRY, 1.5*math.Pi, 2*math.Pi)
	points = append(points, rasterPoint{X: x + w, Y: y + rightRY})

	paintPolyline(img, points, col, lineWidth)
}

//nolint:varnamelen,mnd // raster geometry mirrors the compact PDF path
func paintRoundedLeftStroke(
	img *image.NRGBA,
	paintOp *layout.Op,
	col color.NRGBA,
	lineWidth int,
	pxPerPt float64,
) {
	rx, ry := scaledRadiiXY(paintOp, pxPerPt)
	x := paintOp.X * pxPerPt
	y := paintOp.Y * pxPerPt
	h := paintOp.H * pxPerPt
	strokeInset := float64(lineWidth) / boxFilterFactor2
	x += strokeInset
	topRX := max(rx[0]-strokeInset, 0)
	topRY := max(ry[0]-strokeInset, 0)
	bottomRX := max(rx[3]-strokeInset, 0)
	bottomRY := max(ry[3]-strokeInset, 0)

	points := make([]rasterPoint, 0, roundedArcSteps*2+3) //nolint:mnd // two arcs plus their joins
	points = append(points, rasterPoint{X: x + bottomRX, Y: y + h})
	points = appendArc(points, x+bottomRX, y+h-bottomRY, bottomRX, bottomRY, 0.5*math.Pi, math.Pi)
	points = append(points, rasterPoint{X: x, Y: y + topRY})
	points = appendArc(points, x+topRX, y+topRY, topRX, topRY, math.Pi, 1.5*math.Pi)

	paintPolyline(img, points, col, lineWidth)
}

//nolint:varnamelen,mnd // raster geometry mirrors the compact PDF path
func paintRoundedBottomStroke(
	img *image.NRGBA,
	paintOp *layout.Op,
	col color.NRGBA,
	lineWidth int,
	pxPerPt float64,
) {
	rx, ry := scaledRadiiXY(paintOp, pxPerPt)
	x := paintOp.X * pxPerPt
	y := paintOp.Y * pxPerPt
	w := paintOp.W * pxPerPt
	h := paintOp.H * pxPerPt
	strokeInset := float64(lineWidth) / boxFilterFactor2
	leftRX := max(rx[3]-strokeInset, 0)
	leftRY := max(ry[3]-strokeInset, 0)
	rightRX := max(rx[2]-strokeInset, 0)
	rightRY := max(ry[2]-strokeInset, 0)
	bottomY := y + h

	points := make([]rasterPoint, 0, roundedArcSteps*2+3) //nolint:mnd // two arcs plus their joins
	points = append(points, rasterPoint{X: x, Y: bottomY - leftRY})
	points = appendArc(points, x+leftRX, bottomY-leftRY, leftRX, leftRY, math.Pi, 0.5*math.Pi)
	points = append(points, rasterPoint{X: x + w - rightRX, Y: bottomY})
	points = appendArc(points, x+w-rightRX, bottomY-rightRY, rightRX, rightRY, 0.5*math.Pi, 0)
	points = append(points, rasterPoint{X: x + w, Y: bottomY - rightRY})

	paintPolyline(img, points, col, lineWidth)
}

//nolint:varnamelen,mnd // raster geometry mirrors the compact PDF path
func paintRoundedRightStroke(
	img *image.NRGBA,
	paintOp *layout.Op,
	col color.NRGBA,
	lineWidth int,
	pxPerPt float64,
) {
	rx, ry := scaledRadiiXY(paintOp, pxPerPt)
	x := paintOp.X * pxPerPt
	y := paintOp.Y * pxPerPt
	w := paintOp.W * pxPerPt
	h := paintOp.H * pxPerPt
	strokeInset := float64(lineWidth) / boxFilterFactor2
	rightX := x + w - strokeInset
	topRX := max(rx[1]-strokeInset, 0)
	topRY := max(ry[1]-strokeInset, 0)
	bottomRX := max(rx[2]-strokeInset, 0)
	bottomRY := max(ry[2]-strokeInset, 0)

	points := make([]rasterPoint, 0, roundedArcSteps*2+3) //nolint:mnd // two arcs plus their joins
	points = append(points, rasterPoint{X: rightX - topRX, Y: y})
	points = appendArc(points, rightX-topRX, y+topRY, topRX, topRY, 1.5*math.Pi, 2*math.Pi)
	points = append(points, rasterPoint{X: rightX, Y: y + h - bottomRY})
	points = appendArc(points, rightX-bottomRX, y+h-bottomRY, bottomRX, bottomRY, 0, 0.5*math.Pi)

	paintPolyline(img, points, col, lineWidth)
}

//nolint:varnamelen,mnd // raster geometry mirrors the compact PDF path
func paintRoundedStroke(
	img *image.NRGBA,
	paintOp *layout.Op,
	col color.NRGBA,
	lineWidth int,
	pxPerPt float64,
) {
	rx, ry := scaledRadiiXY(paintOp, pxPerPt)
	x := paintOp.X * pxPerPt
	y := paintOp.Y * pxPerPt
	w := paintOp.W * pxPerPt
	h := paintOp.H * pxPerPt
	strokeInset := float64(lineWidth) / boxFilterFactor2

	for i := range rx {
		rx[i] = max(rx[i]-strokeInset, 0)
		ry[i] = max(ry[i]-strokeInset, 0)
	}

	points := make([]rasterPoint, 0, roundedArcSteps*4+4) //nolint:mnd // four arcs plus their joins
	points = append(points, rasterPoint{X: x + rx[0], Y: y})
	points = append(points, rasterPoint{X: x + w - rx[1], Y: y})
	points = appendArc(points, x+w-rx[1], y+ry[1], rx[1], ry[1], 1.5*math.Pi, 2*math.Pi)
	points = append(points, rasterPoint{X: x + w, Y: y + h - ry[2]})
	points = appendArc(points, x+w-rx[2], y+h-ry[2], rx[2], ry[2], 0, 0.5*math.Pi)
	points = append(points, rasterPoint{X: x + rx[3], Y: y + h})
	points = appendArc(points, x+rx[3], y+h-ry[3], rx[3], ry[3], 0.5*math.Pi, math.Pi)
	points = append(points, rasterPoint{X: x, Y: y + ry[0]})
	points = appendArc(points, x+rx[0], y+ry[0], rx[0], ry[0], math.Pi, 1.5*math.Pi)

	paintPolyline(img, points, col, lineWidth)
}

const roundedArcSteps = 8

type rasterPoint struct {
	X, Y float64
}

func appendArc(points []rasterPoint, centerX, centerY, radiusX, radiusY, start, end float64) []rasterPoint {
	if radiusX <= 0 || radiusY <= 0 {
		return points
	}

	for step := 1; step <= roundedArcSteps; step++ {
		angle := start + (end-start)*float64(step)/roundedArcSteps
		points = append(points, rasterPoint{
			X: centerX + radiusX*math.Cos(angle),
			Y: centerY + radiusY*math.Sin(angle),
		})
	}

	return points
}

func paintPolyline(img *image.NRGBA, points []rasterPoint, col color.NRGBA, lineWidth int) {
	for i := 1; i < len(points); i++ {
		paintStrokeSegment(img, points[i-1], points[i], col, lineWidth)
	}
}

func paintStrokeSegment(img *image.NRGBA, start, end rasterPoint, col color.NRGBA, lineWidth int) {
	half := float64(lineWidth) / boxFilterFactor2
	minX := max(int(math.Floor(math.Min(start.X, end.X)-half))-1, img.Bounds().Min.X)
	maxX := min(int(math.Ceil(math.Max(start.X, end.X)+half))+1, img.Bounds().Max.X)
	minY := max(int(math.Floor(math.Min(start.Y, end.Y)-half))-1, img.Bounds().Min.Y)
	maxY := min(int(math.Ceil(math.Max(start.Y, end.Y)+half))+1, img.Bounds().Max.Y)

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			if pointSegmentDistance(float64(x)+pixelCenter, float64(y)+pixelCenter, start, end) <= half {
				img.SetNRGBA(x, y, col)
			}
		}
	}
}

//nolint:varnamelen,wsl // segment math uses conventional compact names
func pointSegmentDistance(x, y float64, start, end rasterPoint) float64 {
	dx := end.X - start.X
	dy := end.Y - start.Y
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return math.Hypot(x-start.X, y-start.Y)
	}

	t := ((x-start.X)*dx + (y-start.Y)*dy) / lengthSquared
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	return math.Hypot(x-(start.X+t*dx), y-(start.Y+t*dy))
}

func scaledRadii(paintOp *layout.Op, pxPerPt float64) [4]float64 {
	radii := [4]float64{paintOp.RadiusTopLeft, paintOp.RadiusTopRight, paintOp.RadiusBottomRight, paintOp.RadiusBottomLeft}
	if radii == [4]float64{} {
		radii = [4]float64{paintOp.Radius, paintOp.Radius, paintOp.Radius, paintOp.Radius}
	}

	for i := range radii {
		radii[i] *= pxPerPt
	}

	return radii
}

func scaledRadiiXY(paintOp *layout.Op, pxPerPt float64) ([4]float64, [4]float64) {
	radiusX := scaledRadii(paintOp, pxPerPt)
	radiusY := [4]float64{
		paintOp.RadiusTopLeftY, paintOp.RadiusTopRightY,
		paintOp.RadiusBottomRightY, paintOp.RadiusBottomLeftY,
	}

	if radiusY == [4]float64{} {
		if paintOp.RadiusY > 0 {
			radiusY = [4]float64{paintOp.RadiusY, paintOp.RadiusY, paintOp.RadiusY, paintOp.RadiusY}
		} else {
			return radiusX, radiusX
		}
	}

	for idx := range radiusY {
		radiusY[idx] *= pxPerPt
		if radiusX[idx] <= 0 {
			radiusY[idx] = 0
		} else if radiusY[idx] <= 0 {
			radiusY[idx] = radiusX[idx]
		}
	}

	return radiusX, radiusY
}

func inEllipse(pointX, pointY, centerX, centerY, radiusX, radiusY float64) bool {
	if radiusX <= 0 || radiusY <= 0 {
		return false
	}

	deltaX := (pointX - centerX) / radiusX
	deltaY := (pointY - centerY) / radiusY

	return deltaX*deltaX+deltaY*deltaY <= 1
}

//nolint:cyclop,varnamelen,wsl // four corner regions are explicit
func roundedContains(
	x, y, originX, originY, width, height float64,
	rx, ry [4]float64,
) bool {
	if x < originX || x >= originX+width || y < originY || y >= originY+height {
		return false
	}

	if x < originX+rx[0] && y < originY+ry[0] {
		return inEllipse(x, y, originX+rx[0], originY+ry[0], rx[0], ry[0])
	}
	if x >= originX+width-rx[1] && y < originY+ry[1] {
		return inEllipse(x, y, originX+width-rx[1], originY+ry[1], rx[1], ry[1])
	}
	if x >= originX+width-rx[2] && y >= originY+height-ry[2] {
		return inEllipse(x, y, originX+width-rx[2], originY+height-ry[2], rx[2], ry[2])
	}
	if x < originX+rx[3] && y >= originY+height-ry[3] {
		return inEllipse(x, y, originX+rx[3], originY+height-ry[3], rx[3], ry[3])
	}

	return true
}

//nolint:cyclop // rounded and rectangular stroke painting
func paintStrokeRect(img *image.NRGBA, paintOp *layout.Op, paintStyle layout.PaintStyle, pxPerPt float64) {
	alpha := 1.0
	if paintOp.Alpha > 0 && paintOp.Alpha < 1 {
		alpha = paintOp.Alpha
	}

	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax),
		A: uint8(math.Round(alpha * channelMax)),
	}
	lineWidth := strokeWidthScale(paintStyle.StrokeWidth, pxPerPt)

	if paintOp.StrokeMask != 0 {
		// Partial multi-page frames: paint only the selected sides.
		if paintOp.StrokeMask&layout.StrokeMaskTop != 0 {
			paintRoundedTopStroke(img, paintOp, col, lineWidth, pxPerPt)
		}

		if paintOp.StrokeMask&layout.StrokeMaskBottom != 0 {
			paintRoundedBottomStroke(img, paintOp, col, lineWidth, pxPerPt)
		}

		if paintOp.StrokeMask&layout.StrokeMaskLeft != 0 {
			paintRoundedLeftStroke(img, paintOp, col, lineWidth, pxPerPt)
		}

		if paintOp.StrokeMask&layout.StrokeMaskRight != 0 {
			paintRoundedRightStroke(img, paintOp, col, lineWidth, pxPerPt)
		}

		return
	}

	if paintOp.Radius > 0 || paintOp.RadiusTopLeft > 0 || paintOp.RadiusTopRight > 0 ||
		paintOp.RadiusBottomRight > 0 || paintOp.RadiusBottomLeft > 0 {
		paintRoundedStroke(img, paintOp, col, lineWidth, pxPerPt)

		return
	}

	rect := ptRectScale(paintOp.X, paintOp.Y, paintOp.W, paintOp.H, pxPerPt)

	rects := [4]image.Rectangle{
		image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+lineWidth),
		image.Rect(rect.Min.X, rect.Max.Y-lineWidth, rect.Max.X, rect.Max.Y),
		image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+lineWidth, rect.Max.Y),
		image.Rect(rect.Max.X-lineWidth, rect.Min.Y, rect.Max.X, rect.Max.Y),
	}
	for _, rr := range rects {
		if r := rr.Intersect(img.Bounds()); !r.Empty() {
			if col.A == opaqueAlpha {
				fillNRGBAOpaque(img, r, col)
			} else {
				//nolint:exhaustruct // intentional zero/partial fields
				draw.Draw(img, r, image.NewUniform(col), image.Point{}, draw.Over)
			}
		}
	}
}

func paintLine(img *image.NRGBA, paintOp *layout.Op, paintStyle layout.PaintStyle, pxPerPt float64) {
	alpha := 1.0
	if paintOp.Alpha > 0 && paintOp.Alpha < 1 {
		alpha = paintOp.Alpha
	}

	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax),
		A: uint8(math.Round(alpha * channelMax)),
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
		if col.A == opaqueAlpha {
			fillNRGBAOpaque(img, rect, col)
		} else {
			//nolint:exhaustruct // intentional zero/partial fields
			draw.Draw(img, rect, image.NewUniform(col), image.Point{}, draw.Over)
		}
	}
}

// paintText draws the run (and fake-bold pass) at fractional baselines.
func paintText(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64, atlas *glyphAtlas) {
	alpha := 1.0
	if paintOp.Alpha > 0 && paintOp.Alpha < 1 {
		alpha = paintOp.Alpha
	}

	col := color.NRGBA{
		R: uint8(paintOp.R * channelMax), G: uint8(paintOp.G * channelMax), B: uint8(paintOp.B * channelMax),
		A: uint8(math.Round(alpha * channelMax)),
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

	ttfDrawString(
		img, baseX, baseY, paintOp.Text, paintOp.Size,
		paintOp.LetterSpacing, paintOp.RotateDeg, face, col, pxPerPt, atlas,
	)
	// Latin-only fake-bold (CJK gate lives in layout.FakeBoldFor).
	if layout.FakeBoldFor(paintOp) {
		ttfDrawString(
			img, baseX+float64(rasterSS), baseY, paintOp.Text, paintOp.Size,
			paintOp.LetterSpacing, paintOp.RotateDeg, face, col, pxPerPt, atlas,
		)
	}
}

// paintImage draws a decoded paintOp image, scaled via the per-run cache.
//
//nolint:cyclop // raster image painting pipeline
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
	if scaled == nil {
		return
	}

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
	dst, err := newRasterImage(width, height)
	if err != nil {
		return nil
	}

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
	dst, err := newRasterImage(width, height)
	if err != nil {
		return nil
	}

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

// RunRequest drives image conversion from an image-only request. req.Output
// receives encoded PNG/JPEG bytes; callers must supply a writer.
func RunRequest(ctx context.Context, req *Request, log io.Writer) error {
	if req == nil {
		return errNilRequest
	}

	if ctx == nil {
		return errNilContext
	}

	if err := req.Validate(); err != nil {
		return fmt.Errorf("imageout: validate: %w", err)
	}

	if log == nil {
		log = io.Discard
	}
	// Policy A: one quiet bit — CLI --quiet sets Global.Quiet (not Image.Quiet).
	if req.Global.Quiet {
		log = io.Discard
	}

	obj := &req.Objects[0]

	// P2-07: full load policy at construction. Image.Load holds image-mode
	// Proxy; CLI/library ACL (--allow / --enable-local-file-access) lives on
	// Global.Load. Merge so NewLoader applies everything; no post-construction
	// field pokes on Loader.
	loader := load.NewLoader(imageLoadGlobal(req.Global, req.Image))
	loader.Log = log

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}

	registry := pdf.RegistryFromGlobal(req.Global)
	logFontRegistryScan(req.Global, log)

	pipeline := &imagePipeline{ //nolint:exhaustruct // image is populated during RenderObjects
		req:      req,
		obj:      obj,
		loader:   loader,
		font:     font,
		registry: registry,
		log:      log,
	}
	if err := renderpipeline.Run(ctx, pipeline); err != nil {
		return fmt.Errorf("imageout: render pipeline: %w", err)
	}

	return nil
}

// imagePipeline adapts image-specific state to the shared render lifecycle.
// Rendering and encoding stay private to imageout; render owns sequencing and
// cancellation checks shared with the PDF pipeline.
type imagePipeline struct {
	req      *Request
	obj      *settings.PdfObject
	loader   *load.Loader
	font     *pdf.Font
	registry *pdf.Registry
	log      io.Writer
	img      image.Image
}

func (p *imagePipeline) RenderObjects(ctx context.Context) error {
	imgSet := &p.req.Image

	prep, media, err := prepareImageDocument(ctx, p.loader, p.obj, p.req.Global, imgSet, p.registry, p.log)
	if err != nil {
		return err
	}

	root := prep.Root
	sheets := prep.Sheets
	p.registry = prep.Registry

	cache := map[string][]byte{}
	imagesFn := makeImageFetcher(ctx, imgSet, prep, cache)
	printLinkUnderline := imgSet.Web.PrintLinkUnderline ||
		p.req.Global.Web.PrintLinkUnderline ||
		p.obj.Web.PrintLinkUnderline

	// Policy A: Quiet is Global.Quiet; body paint background is Global.Background
	// only (single field for PDF + image; CLI --background / library Set).
	img, err := RenderContext(ctx, root, RenderOptions{
		Width:              imgSet.Width,
		Height:             imgSet.Height,
		Font:               p.font,
		Registry:           p.registry,
		Sheets:             sheets,
		Media:              media,
		Images:             imagesFn,
		Background:         p.req.Global.Background,
		Transparent:        imgSet.Transparent,
		Crop:               cropRect(imgSet.Crop),
		SmartWidth:         imgSet.SmartWidth,
		PrintLinkUnderline: printLinkUnderline,
	})
	if err != nil {
		return err
	}

	p.img = img

	return nil
}

func (p *imagePipeline) Assemble(context.Context) error {
	return nil
}

func (p *imagePipeline) Finalize(context.Context) error {
	return writeEncodedOutput(p.req, p.img, p.log)
}

// writeEncodedOutput resolves the format, composites onto white for
// transparent JPEG, and writes the encoded bytes to req.Output.
func writeEncodedOutput(req *Request, img image.Image, log io.Writer) error {
	imgSet := &req.Image

	if req.Output == nil {
		return errNilOutput
	}

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

	if _, err := req.Output.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// prepareImageDocument resolves viewport/media/SimplifyDOM via prepare.BuildOptions
// and runs prepare.Document for image mode (single page).
func prepareImageDocument(
	ctx context.Context,
	loader *load.Loader,
	obj *settings.PdfObject,
	global settings.PdfGlobal,
	imgSet *settings.ImageGlobal,
	registry *pdf.Registry,
	log io.Writer,
) (*prepare.Prepared, string, error) {
	var imageSet settings.ImageGlobal
	if imgSet != nil {
		imageSet = *imgSet
	}

	media := mediaFor(global, imageSet, obj)

	viewportW := defaultViewportW
	if imageSet.Width > 0 {
		viewportW = float64(imageSet.Width)
	}

	viewportH := defaultViewportH
	if imageSet.Height > 0 {
		viewportH = float64(imageSet.Height)
	}

	prep, err := prepare.Document(
		ctx,
		loader,
		obj.Page,
		obj.Load,
		registry,
		prepare.BuildOptions(
			viewportW,
			viewportH,
			media,
			1,
			global.Web,
			imageSet.Web,
			obj.Web,
		),
		log,
	)
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
	prep *prepare.Prepared,
	cache map[string][]byte,
) func(string) ([]byte, error) {
	var cacheBytes int

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

		if len(cache) < maxImageFetches && cacheBytes <= maxImageFetchBytes-len(res.Body) {
			cache[src] = res.Body
			cacheBytes += len(res.Body)
		}

		return res.Body, nil
	}
}

// logFontRegistryScan emits the shared font-path scan notice after
// pdf.RegistryFromGlobal.
func logFontRegistryScan(global settings.PdfGlobal, log io.Writer) {
	if log == nil || log == io.Discard || global.Quiet {
		return
	}

	if len(global.FontPaths) == 0 && !global.UseSystemFonts {
		return
	}

	count := len(global.FontPaths)
	if global.UseSystemFonts {
		count += len(pdf.DefaultSystemFontDirs())
	}

	line.Emit(log, line.Info, "scanned %d font path(s)", count)
}

// imageLoadGlobal resolves the shared and image-owned load settings before
// constructing the loader. The shared policy remains authoritative for
// explicit network restrictions; image settings can supply an image proxy or
// additive local-file ACL values. NewLoader then receives one complete policy
// snapshot for primary and subresource loads.
func imageLoadGlobal(global settings.PdfGlobal, image settings.ImageGlobal) settings.LoadGlobal {
	return load.ResolveEffectiveLoadGlobal(global.Load, image.Load)
}

// mediaFor resolves the layout media via settings.ResolveImageMedia (P1-4).
func mediaFor(global settings.PdfGlobal, image settings.ImageGlobal, obj *settings.PdfObject) string {
	return settings.ResolveImageMedia(global, image, obj)
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

// ResolveFormat exposes the format policy to the application adapter without
// coupling the image engine to the CLI command type.
func ResolveFormat(flag, output string) (string, error) {
	return resolveFormat(flag, output)
}

// encode serializes img as PNG or JPEG. quality applies to JPEG only
// (1..100); PNG is lossless and ignores it.
func encode(img image.Image, format string, quality int) ([]byte, error) {
	buf := limitedImageBuffer{Buffer: bytes.Buffer{}, limit: maxImageEncoded}

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

type limitedImageBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedImageBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.Len() {
		return 0, errEncodedTooLarge
	}

	n, err := b.Buffer.Write(data)
	if err != nil {
		return n, fmt.Errorf("write buffer: %w", err)
	}

	return n, nil
}

// onWhite composites img onto a white background (JPEG has no alpha).
func onWhite(img image.Image) image.Image {
	bounds := img.Bounds()
	if err := validateRasterSize(bounds.Dx(), bounds.Dy()); err != nil {
		return img
	}

	dst := image.NewNRGBA(bounds)
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
