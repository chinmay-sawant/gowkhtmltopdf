package imageout

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"strings"

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
	errNilRequest      = errors.New("gowkhtmltopdf: nil request")
	errNothingToRender = errors.New("load-error policy is skip; nothing to render")
	errImagesDisabled  = errors.New("gowkhtmltopdf: images disabled")
	errNilOutput       = ErrMissingOutput
	errUnsupportedFmt  = errors.New("unsupported format")
	errRasterTooLarge  = errors.New("imageout: raster exceeds resource budget")
	errImageTooLarge   = errors.New("imageout: image exceeds resource budget")
	errEncodedTooLarge = errors.New("imageout: encoded image exceeds resource budget")
	// errNegativeDimension reports a RenderOptions width or height below
	// zero; 0 keeps the documented defaults (1024 viewport, content height).
	errNegativeDimension = errors.New("imageout: width and height must be non-negative")
	// errInvalidCropRect reports negative crop offsets or dimensions.
	errInvalidCropRect = errors.New("imageout: crop offsets and dimensions must be non-negative")
	// errInvalidMediaType reports a media value other than print, screen,
	// or empty (which applies only "all" rules).
	errInvalidMediaType = errors.New("imageout: media must be print, screen, or empty")
	// errInvalidZoom reports a zoom other than 0 or a finite positive value,
	// matching layout.Options' zoom contract.
	errInvalidZoom = errors.New("imageout: zoom must be zero or a finite positive value")
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

// directRasterPixels is the final-canvas pixel area at or above which
// rasterizeContext paints directly at final resolution instead of
// supersampling. One final pixel costs 4 bytes and the supersampled canvas
// costs rasterSS*rasterSS pixels per final pixel, so this is the largest final
// canvas whose 2x buffer still fits maxPooledRasterBytes (32 MiB / (4*2*2) =
// 2 Mi pixels). At or above the threshold the direct branch avoids the 2x
// canvas entirely and never enters supersamplePixCache (IMG-04). The public
// 250-tile canvas (1024x2056 = 2,105,344 px) and 500-tile canvas
// (1024x4040 = 4,136,960 px) both select the direct branch.
const directRasterPixels = maxPooledRasterBytes / (4 * rasterSS * rasterSS)

// directRaster reports whether a final canvas of finalPixels pixels takes the
// direct final-resolution branch. Keeping the rule a single pixel-area
// comparison makes the threshold boundary testable and auditable.
func directRaster(finalPixels int) bool {
	return finalPixels >= directRasterPixels
}

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
	Padding     int // output pixels added to every canvas edge
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
	// Zoom scales style lengths in layout; 0 keeps the layout default of 1,
	// matching layout.Options.Zoom. Values must be 0 or finite positive.
	Zoom float64
}

// Validate rejects caller mistakes before any layout or rasterization
// work: negative Width/Height (0 keeps the documented defaults), a crop
// with negative offsets or dimensions (which would silently no-op through
// applyCrop), a zoom other than 0 or finite positive, and media values other
// than print, screen, or empty. The dimension and crop halves share
// validateCanvasGeometry with Request.Validate so the app preflight covers
// the same numbers before opening the output.
func (o RenderOptions) Validate() error {
	if err := validateCanvasGeometry(o.Width, o.Height, o.Padding,
		[4]int{o.Crop.Min.X, o.Crop.Min.Y, o.Crop.Dx(), o.Crop.Dy()}, 0); err != nil {
		return err
	}

	if o.Zoom != 0 && !finitePositive(o.Zoom) {
		return fmt.Errorf("%w: got %g", errInvalidZoom, o.Zoom)
	}

	switch strings.ToLower(strings.TrimSpace(o.Media)) {
	case "", "print", "screen":
		return nil
	default:
		return fmt.Errorf("%w: got %q", errInvalidMediaType, o.Media)
	}
}

// validateCanvasGeometry rejects the negative canvas numbers shared by
// Request.Validate and RenderOptions.Validate. cropFloor is the lowest crop
// value the caller's layer accepts: -1 keeps CropSettings' "unset" sentinel,
// 0 requires an already-resolved crop rectangle.
func validateCanvasGeometry(width, height, padding int, crop [4]int, cropFloor int) error {
	if width < 0 || height < 0 || padding < 0 {
		return fmt.Errorf("%w: got %dx%d with %dpx padding", errNegativeDimension, width, height, padding)
	}

	for _, value := range crop {
		if value < cropFloor {
			return fmt.Errorf("%w: got left=%d top=%d width=%d height=%d",
				errInvalidCropRect, crop[0], crop[1], crop[2], crop[3])
		}
	}

	return nil
}

// finitePositive reports whether value is finite and greater than zero,
// mirroring internal/layout's zoom and viewport predicate.
func finitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
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

	if err := opts.Validate(); err != nil {
		return nil, err
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

	img, err := rasterizeContext(ctx, res, maxHeight(res, opts), opts.Transparent, opts.Padding, opts.Zoom)
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
		Zoom:               opts.Zoom,
	}
}

// imageViewport is one resolved image-mode layout viewport in the two units
// the pipeline needs: CSS pixels for the raster context (RenderOptions) and
// points for prepare.BuildOptions plus media matching. HeightPt falls back to
// WidthPt when the caller left Height unset, mirroring layoutOptions, so
// linked and imported sheet media queries evaluate against the geometry
// layout will actually use.
type imageViewport struct {
	WidthPx  float64
	HeightPx float64 // 0 keeps content height for the raster canvas
	WidthPt  float64
	HeightPt float64
}

// resolveImageViewport resolves image Width/Height settings into the layout
// viewport. WidthPx defaults to screenWidthDefault, matching RenderOptions'
// 0-width fallback, and HeightPt defaults to WidthPt when Height is 0.
func resolveImageViewport(width, height int) imageViewport {
	widthPx := float64(width)
	if widthPx <= 0 {
		widthPx = screenWidthDefault
	}

	heightPx := float64(height)
	if heightPx < 0 {
		heightPx = 0
	}

	widthPt := widthPx * cssPxToPt

	heightPt := heightPx * cssPxToPt
	if heightPt <= 0 {
		heightPt = widthPt
	}

	return imageViewport{
		WidthPx:  widthPx,
		HeightPx: heightPx,
		WidthPt:  widthPt,
		HeightPt: heightPt,
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

// rasterPolicy selects the supersample strategy for one rasterization.
// rasterPolicyAuto applies the documented directRasterPixels threshold; the
// other values force one branch so tests and the IMG-03 comparison can hold
// every other input constant.
type rasterPolicy int

const (
	rasterPolicyAuto rasterPolicy = iota
	rasterPolicyDirect
	rasterPolicySupersample
)

// rasterizeContext paints the display list into an NRGBA canvas. The canvas is
// white unless transparent is set, in which case it starts fully transparent
// and only painted ops become visible. Canvases at or above directRasterPixels
// paint directly at final resolution; smaller canvases use rasterSS
// supersampling then box-filter down to the final CSS-pixel size. Glyph
// bitmaps for this run live on a per-rasterize atlas (P5-05) so concurrent
// Renders do not share mutable cache state. zoom is the layout zoom in effect
// (0 means the default of 1) and only feeds budget error messages.
func rasterizeContext(
	ctx context.Context, res *layout.Result, height float64, transparent bool, padding int, zoom float64,
) (*image.NRGBA, error) {
	return rasterizeContextPolicy(ctx, res, height, transparent, padding, zoom, rasterPolicyAuto)
}

// rasterizeContextPolicy is rasterizeContext with an explicit raster policy.
// The supersampled geometry and budgets are computed first so every branch
// keeps the shipped dimension checks and error messages; the direct branch
// then paints the smaller final-size canvas those checks already bound.
//
//nolint:cyclop,funlen,mnd // supersampled rasterization pipeline
func rasterizeContextPolicy(
	ctx context.Context, res *layout.Result, height float64, transparent bool, padding int, zoom float64,
	policy rasterPolicy,
) (*image.NRGBA, error) {
	plan, err := planRasterCanvas(res, height, padding, zoom, policy)
	if err != nil {
		return nil, err
	}

	var img *image.NRGBA

	if plan.direct {
		// The returned canvas is the paint target: no 2x intermediate and no
		// downscale copy. It deliberately bypasses supersamplePixCache so a
		// large final canvas is never retained between conversions (IMG-04).
		img, err = newRasterImage(plan.finalWidth, plan.finalHeight)
		if err != nil {
			return nil, err
		}
	} else {
		neededBytes := plan.ssWidth * plan.ssHeight * 4
		pBuf := supersamplePixCache.get(neededBytes)

		switch {
		case pBuf == nil:
			pBuf = &pixBuffer{b: make([]byte, neededBytes)}

		case cap(pBuf.b) < neededBytes:
			// get only returns fitting buffers; keep a replaced buffer recyclable
			// so the retention policy stays in one place if that changes.
			supersamplePixCache.put(pBuf)
			pBuf = &pixBuffer{b: make([]byte, neededBytes)}

		default:
			pBuf.b = pBuf.b[:neededBytes]
			clear(pBuf.b)
		}

		defer func() {
			supersamplePixCache.put(pBuf)
		}()

		img = &image.NRGBA{
			Pix:    pBuf.b,
			Stride: plan.ssWidth * 4,
			Rect:   image.Rect(0, 0, plan.ssWidth, plan.ssHeight),
		}
	}

	if !transparent {
		fillNRGBAOpaque(img, img.Bounds(), color.NRGBA{R: channelMax, G: channelMax, B: channelMax, A: opaqueAlpha})
	}

	paintPxPerPt := plan.paintScale()
	atlas := newGlyphAtlas()
	imageCache := newRasterImageCache()

	if hasElementGroups(res.Ops) {
		if err := paintWithElementGroups(ctx, img, res.Ops, plan.paddingPt, paintPxPerPt, atlas, imageCache); err != nil {
			return nil, err
		}
	} else {
		for _, opIndex := range rasterPaintOrder(res.Ops) {
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("imageout: context: %w", err)
			}

			op := res.Ops[opIndex]
			offsetPaintOp(&op, plan.paddingPt)
			paint(img, &op, paintPxPerPt, atlas, imageCache)
		}
	}

	if plan.direct {
		return img, nil
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

// offsetPaintOp moves layout output into the padded canvas. Transformed ops
// need the same translation in their matrix so their visual position changes
// without changing the transform around the operation itself.
func offsetPaintOp(paintOp *layout.Op, paddingPt float64) {
	paintOp.X += paddingPt
	paintOp.Y += paddingPt

	if paintOp.XformSet {
		paintOp.Xform.E += paddingPt
		paintOp.Xform.F += paddingPt
	}
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
	hasher := fnv.New64a()
	if isJPEG {
		_, _ = hasher.Write([]byte{1})
	}

	_, _ = hasher.Write(data)

	return hasher.Sum64()
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

	paintOp.BindEmptyExtra()

	if paintOp.BlendMode != "" && paintOp.BlendMode != "normal" {
		paintBlended(img, paintOp, pxPerPt, atlas, imageCache)

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

	case layout.OpGridRun:
		paintGridRun(img, &opCopy, pxPerPt)

	case layout.OpText, layout.OpBullet:
		paintText(img, &opCopy, pxPerPt, atlas)

	case layout.OpImage:
		paintImage(img, &opCopy, pxPerPt, imageCache)

	case layout.OpLinkURI, layout.OpUnknown: // annotations and zero-value ops do not paint
	}
}

// paintTransformedOp renders an op with CSS 2D affine transform onto img.
//
//nolint:varnamelen,mnd,wsl,gosec,funlen // affine raster op transform and pixel compositing
func paintTransformedOp(
	img *image.NRGBA, op *layout.Op, pxPerPt float64, atlas *glyphAtlas, imageCache *rasterImageCache,
) {
	dstRect := paintOpBounds(op, pxPerPt).Intersect(img.Bounds())
	if dstRect.Empty() {
		return
	}

	minX, minY, maxX, maxY := opRectBounds(op)
	srcRect := ptRectScale(minX-2, minY-2, (maxX-minX)+4, (maxY-minY)+4, pxPerPt)
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
	// Shape geometry is the unclipped op rectangle. rect is only the paint
	// window (canvas edge or strip), so a rounded fill that crosses a strip
	// keeps its real corners instead of growing a new radius on the cut.
	fullRect := ptRectScale(paintOp.X, paintOp.Y, paintOp.W, paintOp.H, pxPerPt)
	mask := image.NewAlpha(rect)
	originX := float64(fullRect.Min.X)
	originY := float64(fullRect.Min.Y)
	width := float64(fullRect.Dx())
	height := float64(fullRect.Dy())

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if roundedContains(float64(x)+pixelCenter, float64(y)+pixelCenter,
				originX, originY, width, height, radiusX, radiusY) {
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

// scaledRadiiXY resolves the op's corner radii through layout.OpRadiiXY, the
// one owner of the shorthand, corner-longhand, and Y fallback rules, then
// converts them from layout points to raster pixels. The resolver stays in
// layout so a radii change lands in one package; this wrapper only scales.
func scaledRadiiXY(paintOp *layout.Op, pxPerPt float64) ([4]float64, [4]float64) {
	radiusX, radiusY := layout.OpRadiiXY(paintOp)

	for idx := range radiusX {
		radiusX[idx] *= pxPerPt
		radiusY[idx] *= pxPerPt
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
	opX, opY, opW, opH, _ := paintOp.PaintLineGeometry()
	// Centre the stroke on the line: half its width, in points. Extend past
	// each endpoint by that same half (square-cap equivalent) so meeting
	// axis-aligned borders fill the outer corner instead of leaving a notch.
	half := float64(lineWidth) / boxFilterFactor2 / pxPerPt

	var rect image.Rectangle

	if paintOp.H <= 0 { // horizontal line
		rect = ptRectScale(
			opX-half, opY-half, opW+boxFilterFactor2*half, boxFilterFactor2*half, pxPerPt,
		)
	} else { // vertical line
		rect = ptRectScale(
			opX-half, opY-half, boxFilterFactor2*half, opH+boxFilterFactor2*half, pxPerPt,
		)
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

// paintGridRun replays one batched table-row grid as individual lines, in
// emission order, so raster output matches the pre-batch op list.
func paintGridRun(img *image.NRGBA, runOp *layout.Op, pxPerPt float64) {
	if runOp.Grid == nil {
		return
	}

	for idx := range runOp.Grid.Segs {
		seg := &runOp.Grid.Segs[idx]
		line := layout.Op{ //nolint:exhaustruct // intentional zero fields
			ID: runOp.ID, Kind: layout.OpLine,
			X: seg.X, Y: seg.Y, W: seg.W, H: seg.H,
			Width: seg.Width, R: seg.R, G: seg.G, B: seg.B, LineInset: seg.LineInset,
		}

		paintLine(img, &line, layout.StyleOf(&line), pxPerPt)
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
		img, baseX, baseY, paintOp.Text, paintOp.TextLanguage(), paintOp.Size,
		paintOp.LetterSpacing, float64(paintOp.RotateDeg), face, col, pxPerPt, atlas,
	)
	// Latin-only fake-bold (CJK gate lives in layout.FakeBoldFor). The offset
	// is one final CSS pixel expressed in canvas pixels, so the direct branch
	// (pxPerPt = ptToPx) shifts by one canvas pixel rather than by rasterSS.
	if layout.FakeBoldFor(paintOp) {
		boldOffset := pxPerPt / ptToPx
		ttfDrawString(
			img, baseX+boldOffset, baseY, paintOp.Text, paintOp.TextLanguage(), paintOp.Size,
			paintOp.LetterSpacing, float64(paintOp.RotateDeg), face, col, pxPerPt, atlas,
		)
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

//nolint:mnd // bit packing for one NRGBA pixel; alpha is always opaque here
func fillNRGBAOpaque(dst *image.NRGBA, rect image.Rectangle, col color.NRGBA) {
	if rect.Empty() {
		return
	}

	// Pack one 4-byte NRGBA pixel; 64-bit stores fill two pixels per write so
	// large canvas backgrounds and tile fills move at memory speed. A is
	// always opaque here (every caller checks opaqueAlpha first).
	pattern := uint32(col.R) | uint32(col.G)<<8 | uint32(col.B)<<16 | uint32(opaqueAlpha)<<24
	word := uint64(pattern) | uint64(pattern)<<32
	rowBytes := rect.Dx() * 4 //nolint:mnd // 4 bytes per NRGBA pixel

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		offset := dst.PixOffset(rect.Min.X, y)
		row := dst.Pix[offset : offset+rowBytes]

		index := 0

		for ; index+32 <= rowBytes; index += 32 {
			binary.LittleEndian.PutUint64(row[index:], word)
			binary.LittleEndian.PutUint64(row[index+8:], word)
			binary.LittleEndian.PutUint64(row[index+16:], word)
			binary.LittleEndian.PutUint64(row[index+24:], word)
		}

		for ; index+8 <= rowBytes; index += 8 {
			binary.LittleEndian.PutUint64(row[index:], word)
		}

		if index < rowBytes {
			binary.LittleEndian.PutUint32(row[index:], pattern)
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

	pipeline, err := newImagePipeline(req, log)
	if err != nil {
		return err
	}

	if err := renderpipeline.Run(ctx, pipeline); err != nil {
		return fmt.Errorf("imageout: render pipeline: %w", err)
	}

	return nil
}

// newImagePipeline builds the image pipeline and its dependencies (loader,
// default font, registry) from the request. RunRequest is then just wiring:
// construct the pipeline, run it. The loader is built here so the full image
// load policy applies at construction (Image.Load proxy plus Global.Load
// ACL), with no post-construction field pokes on the Loader.
func newImagePipeline(req *Request, log io.Writer) (*imagePipeline, error) {
	obj := &req.Objects[0]

	loader, err := load.NewLoaderWithError(imageLoadGlobal(req.Global, req.Image))
	if err != nil {
		return nil, fmt.Errorf("imageout: loader: %w", err)
	}

	loader.SetLog(log)

	font, err := pdf.DefaultFont()
	if err != nil {
		return nil, fmt.Errorf("default font: %w", err)
	}

	registry := pdf.RegistryFromGlobal(req.Global)
	pdf.LogFontRegistryScan(req.Global, log)

	return &imagePipeline{ //nolint:exhaustruct // pending is populated during RenderObjects
		req:      req,
		obj:      obj,
		loader:   loader,
		font:     font,
		registry: registry,
		log:      log,
	}, nil
}

// imagePipeline adapts image-specific state to the shared render lifecycle.
// Rendering and encoding stay private to imageout; render owns sequencing and
// cancellation checks shared with the PDF pipeline.
// pendingImageRaster is the laid-out display list Finalize rasterizes. Large
// PNG output paints horizontal strips straight into the encoder; everything
// else still builds a full canvas and goes through writeEncodedOutput.
type pendingImageRaster struct {
	res         *layout.Result
	height      float64
	transparent bool
	padding     int
	zoom        float64
	crop        image.Rectangle
}

type imagePipeline struct {
	req      *Request
	obj      *settings.PdfObject
	loader   *load.Loader
	font     *pdf.Font
	registry *pdf.Registry
	log      io.Writer
	img      image.Image
	pending  pendingImageRaster
}

// Compile-time check: imagePipeline satisfies the shared render lifecycle seam.
var _ renderpipeline.Pipeline = (*imagePipeline)(nil)

func (p *imagePipeline) RenderObjects(ctx context.Context) error {
	imgSet := &p.req.Image

	prep, media, err := prepareImageDocument(ctx, p.loader, p.obj, p.req.Global, imgSet, p.registry, p.log)
	if err != nil {
		return err
	}

	root := prep.Root
	sheets := prep.Sheets
	p.registry = prep.Registry

	viewport := resolveImageViewport(imgSet.Width, imgSet.Height)
	imagesEnabled := settings.ResolveImages(p.req.Global.Web, imgSet, p.obj)

	cache := map[string][]byte{}
	imagesFn := makeImageFetcher(ctx, imagesEnabled, prep, cache)
	printLinkUnderline := imgSet.Web.PrintLinkUnderline ||
		p.req.Global.Web.PrintLinkUnderline ||
		p.obj.Web.PrintLinkUnderline

	// Policy A: Quiet is Global.Quiet; body paint background is Global.Background
	// only (single field for PDF + image; CLI --background / library Set).
	opts := RenderOptions{
		Width:              int(viewport.WidthPx),
		Height:             int(viewport.HeightPx),
		Font:               p.font,
		Registry:           p.registry,
		Sheets:             sheets,
		Media:              media,
		Images:             imagesFn,
		Background:         p.req.Global.Background,
		Transparent:        imgSet.Transparent,
		Crop:               cropRect(imgSet.Crop),
		SmartWidth:         imgSet.SmartWidth,
		Padding:            imgSet.Padding,
		PrintLinkUnderline: printLinkUnderline,
		Zoom:               p.obj.Load.ZoomFactor,
	}

	res, err := layoutResult(ctx, root, opts, p.font)
	if err != nil {
		return err
	}

	p.pending = pendingImageRaster{
		res:         res,
		height:      maxHeight(res, opts),
		transparent: opts.Transparent,
		padding:     opts.Padding,
		zoom:        opts.Zoom,
		crop:        opts.Crop,
	}

	return nil
}

func (p *imagePipeline) Assemble(context.Context) error {
	return nil
}

func (p *imagePipeline) Finalize(ctx context.Context) error {
	if p.pending.res != nil && p.img == nil {
		return p.encodePending(ctx)
	}

	return writeEncodedOutput(ctx, p.req, p.img, p.log)
}

func (p *imagePipeline) encodePending(ctx context.Context) error {
	format, err := resolveFormat(p.req.Image.Format, "")
	if err != nil {
		return err
	}

	plan, err := planRasterCanvas(
		p.pending.res, p.pending.height, p.pending.padding, p.pending.zoom, rasterPolicyAuto,
	)
	if err != nil {
		return err
	}

	if stripRasterApplies(plan, format) {
		return p.encodePendingStrips(ctx, plan)
	}

	img, err := rasterizeContextPolicy(
		ctx, p.pending.res, p.pending.height, p.pending.transparent,
		p.pending.padding, p.pending.zoom, rasterPolicyAuto,
	)
	if err != nil {
		return err
	}

	cropped, err := applyCrop(img, p.pending.crop)
	if err != nil {
		return err
	}

	p.img = cropped

	return writeEncodedOutput(ctx, p.req, p.img, p.log)
}

func (p *imagePipeline) encodePendingStrips(ctx context.Context, plan rasterCanvasPlan) error {
	if ctx == nil {
		return errNilContext
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("imageout: context: %w", err)
	}

	if p.req.Output == nil {
		return errNilOutput
	}

	buf := acquireEncodeBuffer()
	defer releaseEncodeBuffer(buf)

	if err := encodeStripPNG(ctx, buf, p.pending, plan, rasterStripBytes); err != nil {
		return fmt.Errorf("encode %s: %w", formatPNG, err)
	}

	return writeOutputBytes(ctx, p.req, buf)
}

// writeEncodedOutput resolves the format, composites onto white for
// transparent JPEG, and writes the encoded bytes to req.Output. The context
// can stop work before encoding and before the external write starts. An
// io.Writer cannot be interrupted by this context once Write has started.
func writeEncodedOutput(ctx context.Context, req *Request, img image.Image, log io.Writer) error {
	if ctx == nil {
		return errNilContext
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("imageout: context: %w", err)
	}

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

	// The pooled scratch is written to the sink before it returns to the pool,
	// so the encoded bytes are never copied out and a failed encode still
	// writes nothing.
	buf := acquireEncodeBuffer()
	defer releaseEncodeBuffer(buf)

	if err := encodeInto(buf, img, format, imgSet.Quality, !imgSet.Transparent); err != nil {
		return fmt.Errorf("encode %s: %w", format, err)
	}

	return writeOutputBytes(ctx, req, buf)
}

func writeOutputBytes(ctx context.Context, req *Request, buf *limitedImageBuffer) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("imageout: context: %w", err)
	}

	count, err := req.Output.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if count != buf.Len() {
		return fmt.Errorf("write output: %w", io.ErrShortWrite)
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

	// prepare/media matching is in points; imageSet.Width/Height are CSS
	// pixels, so both call sites go through the same resolved viewport.
	viewport := resolveImageViewport(imageSet.Width, imageSet.Height)

	prep, err := prepare.Document(
		ctx,
		loader,
		obj.Page,
		obj.Load,
		registry,
		prepare.BuildOptions(
			viewport.WidthPt,
			viewport.HeightPt,
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
// the resolved web.images gate. imagesEnabled must come from
// settings.ResolveImages so global, image, and object layers all reach the
// gate.
func makeImageFetcher(
	ctx context.Context,
	imagesEnabled bool,
	prep *prepare.Prepared,
	cache map[string][]byte,
) func(string) ([]byte, error) {
	var cacheBytes int

	return func(src string) ([]byte, error) {
		if !imagesEnabled {
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
