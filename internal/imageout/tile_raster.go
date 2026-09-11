package imageout

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

const (
	// rasterStripBytes is the target backing array for one paint strip. At the
	// public 1024px width that is 256 rows (1 MiB), so a 4040-row canvas reuses
	// one buffer instead of allocating 1024x4040x4 = 16.5 MiB.
	rasterStripBytes = 1 << 20
	nrgbaBytes       = 4
)

var errStripNeedsDirect = errors.New("imageout: strip raster requires the direct branch")

// rasterCanvasPlan is the shared geometry rasterizeContextPolicy and the
// strip PNG path both paint from. Dimensions match the shipped supersample
// budget checks; the direct flag is the same pixel-area rule as auto policy.
type rasterCanvasPlan struct {
	ssWidth, ssHeight       int
	finalWidth, finalHeight int
	paddingPt               float64
	direct                  bool
}

func (p rasterCanvasPlan) paintScale() float64 {
	if p.direct {
		return ptToPx
	}

	return ptToPx * float64(rasterSS)
}

// planRasterCanvas computes supersampled and final canvas size the same way
// rasterizeContextPolicy used to inline. The width check has no zoom remedy
// because the viewport is fixed; height and pixel area do.
func planRasterCanvas(
	res *layout.Result, height float64, padding int, zoom float64, policy rasterPolicy,
) (rasterCanvasPlan, error) {
	paddingPt := float64(padding) * cssPxToPt
	ssPxPerPt := ptToPx * float64(rasterSS)
	ssPaddingPx := paddingPt * ssPxPerPt

	widthPx, err := rasterDimension(
		res.Width*ssPxPerPt+ssPaddingPx*2,
		"maxRasterWidth",
		rasterBudget{}, //nolint:exhaustruct // no zoom remedy for a viewport-fixed width
	)
	if err != nil {
		return rasterCanvasPlan{}, err
	}

	heightBudget := rasterBudget{zoom: zoom, remedy: true}

	heightPx, err := rasterDimension(
		height*ssPxPerPt+ssPaddingPx*2, "maxRasterHeight", heightBudget,
	)
	if err != nil {
		return rasterCanvasPlan{}, err
	}

	if err := validateRasterCanvas(widthPx, heightPx, heightBudget); err != nil {
		return rasterCanvasPlan{}, err
	}

	// downscaleBox splits the supersampled canvas with floor division, so the
	// direct branch paints the exact final dimensions the 2x path would
	// return. rasterSS <= 1 folds into the direct branch: there is nothing to
	// downscale and the supersample cache would otherwise recycle the
	// returned canvas.
	finalWidthPx := widthPx / rasterSS
	finalHeightPx := heightPx / rasterSS
	direct := rasterSS <= 1 || policy == rasterPolicyDirect ||
		(policy == rasterPolicyAuto && directRaster(finalWidthPx*finalHeightPx))

	return rasterCanvasPlan{
		ssWidth:     widthPx,
		ssHeight:    heightPx,
		finalWidth:  finalWidthPx,
		finalHeight: finalHeightPx,
		paddingPt:   paddingPt,
		direct:      direct,
	}, nil
}

// stripRasterApplies is the large-PNG gate: same pixel-area line as
// encodeFastPNG / directRaster, and only PNG because JPEG still needs a full
// canvas (or MCU rows this package does not stream).
func stripRasterApplies(plan rasterCanvasPlan, format string) bool {
	return format == formatPNG && plan.direct
}

func rasterStripRows(width, height, stripBytes int) int {
	if width < 1 || height < 1 {
		return 1
	}

	if stripBytes <= 0 {
		stripBytes = rasterStripBytes
	}

	rows := stripBytes / (width * nrgbaBytes)
	if rows < 1 {
		rows = 1
	}

	if rows > height {
		rows = height
	}

	return rows
}

func nrgbaStride(width int) int {
	return width * nrgbaBytes
}

func rasterOutputRect(plan rasterCanvasPlan, crop image.Rectangle) (image.Rectangle, error) {
	canvas := image.Rect(0, 0, plan.finalWidth, plan.finalHeight)
	if crop.Empty() {
		return canvas, nil
	}

	out := crop.Intersect(canvas)
	if out.Empty() {
		return image.Rectangle{}, errCropNoIntersect
	}

	return out, nil
}

// stripPainter paints one horizontal window of a direct-resolution canvas
// into a reused NRGBA whose Rect is the window in canvas coordinates. Paint
// already clips to img.Bounds(), so ops that straddle a strip edge write only
// the pixels that belong in that window.
type stripPainter struct {
	ops         []layout.Op
	transparent bool
	pxPerPt     float64
	out         image.Rectangle
	pix         []byte
	stride      int
	stripH      int
	atlas       *glyphAtlas
	imageCache  *rasterImageCache
}

func newStripPainter(
	res *layout.Result, transparent bool, plan rasterCanvasPlan, crop image.Rectangle, stripBytes int,
) (*stripPainter, error) {
	if !plan.direct {
		return nil, errStripNeedsDirect
	}

	out, err := rasterOutputRect(plan, crop)
	if err != nil {
		return nil, err
	}

	width := out.Dx()
	height := out.Dy()
	stripH := rasterStripRows(width, height, stripBytes)
	stride := nrgbaStride(width)
	ops := offsetOps(res.Ops, plan.paddingPt)

	return &stripPainter{
		ops:         ops,
		transparent: transparent,
		pxPerPt:     plan.paintScale(),
		out:         out,
		pix:         make([]byte, stride*stripH),
		stride:      stride,
		stripH:      stripH,
		atlas:       newGlyphAtlas(),
		imageCache:  newRasterImageCache(),
	}, nil
}

// offsetOps copies ops and applies padding once. Xform lives on a shared
// extra pointer, so a per-strip offset would add padding on every window.
func offsetOps(ops []layout.Op, paddingPt float64) []layout.Op {
	if len(ops) == 0 {
		return nil
	}

	out := make([]layout.Op, len(ops))
	copy(out, ops)

	if paddingPt == 0 {
		return out
	}

	for i := range out {
		offsetPaintOp(&out[i], paddingPt)
	}

	return out
}

func (s *stripPainter) paintWindow(ctx context.Context, top, bottom int) (*image.NRGBA, error) {
	used := (bottom - top) * s.stride
	clear(s.pix[:used])

	strip := &image.NRGBA{
		Pix:    s.pix[:used],
		Stride: s.stride,
		Rect:   image.Rect(s.out.Min.X, top, s.out.Max.X, bottom),
	}

	if !s.transparent {
		fillNRGBAOpaque(strip, strip.Bounds(), color.NRGBA{
			R: channelMax, G: channelMax, B: channelMax, A: opaqueAlpha,
		})
	}

	if err := paintDisplayList(ctx, strip, s.ops, s.pxPerPt, s.atlas, s.imageCache); err != nil {
		return nil, err
	}

	return strip, nil
}

func paintDisplayList(
	ctx context.Context,
	img *image.NRGBA,
	ops []layout.Op,
	pxPerPt float64,
	atlas *glyphAtlas,
	imageCache *rasterImageCache,
) error {
	clip := img.Bounds()

	for _, opIndex := range rasterPaintOrder(ops) {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("imageout: context: %w", err)
		}

		paintOp := ops[opIndex]

		if paintOpBounds(&paintOp, pxPerPt).Intersect(clip).Empty() {
			continue
		}

		paint(img, &paintOp, pxPerPt, atlas, imageCache)
	}

	return nil
}

func (s *stripPainter) forEachWindow(ctx context.Context, visit func(*image.NRGBA) error) error {
	for top := s.out.Min.Y; top < s.out.Max.Y; top += s.stripH {
		bottom := min(top+s.stripH, s.out.Max.Y)

		strip, err := s.paintWindow(ctx, top, bottom)
		if err != nil {
			return err
		}

		if err := visit(strip); err != nil {
			return err
		}
	}

	return nil
}

// encodeStripPNG paints the display list in horizontal strips and streams
// filter-None PNG rows without allocating the full NRGBA canvas.
func encodeStripPNG(
	ctx context.Context, writer io.Writer, job pendingImageRaster, plan rasterCanvasPlan, stripBytes int,
) error {
	painter, err := newStripPainter(job.res, job.transparent, plan, job.crop, stripBytes)
	if err != nil {
		return err
	}

	encoder, err := startFastPNG(writer, painter.out.Dx(), painter.out.Dy(), !job.transparent)
	if err != nil {
		return err
	}

	row := 0

	if err := painter.forEachWindow(ctx, func(strip *image.NRGBA) error {
		for stripRow := range strip.Rect.Dy() {
			offset := strip.PixOffset(strip.Rect.Min.X, strip.Rect.Min.Y+stripRow)
			if err := encoder.writeNRGBARow(strip.Pix[offset:]); err != nil {
				return fmt.Errorf("png deflate row %d: %w", row, err)
			}

			row++
		}

		return nil
	}); err != nil {
		return err
	}

	return encoder.close()
}

// rasterizeStrips paints via the same reused strip buffer as encodeStripPNG
// and copies each window into a full canvas. Tests use it to compare pixels
// against rasterizeContextPolicy; production WriteImage never calls it.
func rasterizeStrips(
	ctx context.Context, res *layout.Result, height float64, transparent bool, padding int, zoom float64,
	crop image.Rectangle, policy rasterPolicy, stripBytes int,
) (*image.NRGBA, error) {
	plan, err := planRasterCanvas(res, height, padding, zoom, policy)
	if err != nil {
		return nil, err
	}

	painter, err := newStripPainter(res, transparent, plan, crop, stripBytes)
	if err != nil {
		return nil, err
	}

	dst, err := newRasterImage(painter.out.Dx(), painter.out.Dy())
	if err != nil {
		return nil, err
	}

	err = painter.forEachWindow(ctx, func(strip *image.NRGBA) error {
		dstY := strip.Rect.Min.Y - painter.out.Min.Y

		for stripRow := range strip.Rect.Dy() {
			srcOff := strip.PixOffset(strip.Rect.Min.X, strip.Rect.Min.Y+stripRow)
			dstOff := dst.PixOffset(0, dstY+stripRow)
			copy(dst.Pix[dstOff:dstOff+painter.stride], strip.Pix[srcOff:srcOff+painter.stride])
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return dst, nil
}
