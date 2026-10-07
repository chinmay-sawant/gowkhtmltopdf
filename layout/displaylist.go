package layout

import (
	"context"
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/imageout"
	ilayout "github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pubstate"
)

// DisplayOp is one display-list operation. It is the internal Op under an
// exported name so a caller outside this module can read the retained
// display list that layout produces and replay it onto its own canvas
// instead of consuming a flattened bitmap.
//
// Coordinates are in canvas points, y down, and for DisplayOpText and
// DisplayOpBullet Y is the baseline. Multiply by Display.PixelPerPoint to
// reach CSS pixels.
//
// Two kinds carry no paint and must be skipped: DisplayOpNoop, which the
// clipper leaves behind when it deactivates an operation, and DisplayOpUnknown,
// which is the boundary marker for a blend or isolation group. Treat any kind
// outside the list below as inert, so a kind added later cannot be mistaken for
// a fill.
//
// Use the accessor methods (LinkURI, ImageBytes, ImageAlt, Transform,
// BlendModeName, Opacity, Outline, FontFeatures, TextLanguage, TextAutospace,
// TextTransformValue, NoFakeBoldValue) for the rare payload rather than the
// promoted fields: the payload sits behind an embedded pointer to an
// unexported type, so a caller cannot build or inspect it, and every accessor
// answers safely when it is absent. The plain fields (Kind, X, Y, W, H, R, G,
// B, Alpha, Width, Size, LetterSpacing, Text, Font, Grid, the radii,
// StrokeMask, LineInset, RotateDeg, Bold, FakeOblique, IsJPEG, IsBackground,
// Fixed, Pinned, Positioned, StickyID, ZIndex, ZIndexSet, XformSet, ID) are
// always safe to read; do not write through them.
//
// A blend group is read through the existing Group, GroupBoundary,
// IsGroupBegin, and IsGroupEnd methods, which are also nil-safe.
//
// Font is a *pdf.Font from an internal package, so a caller cannot declare a
// variable of that type. It does not need to: the face exposes Ascent,
// UnitsPerEm, and Bytes, and a face cache can be keyed on the op.Font value
// itself. Bytes hands the raw SFNT face to an independent shaper.
type DisplayOp = ilayout.Op

// DisplayGroup is the CSS element group an operation belongs to, used by
// mix-blend-mode and isolation: isolate. Nil for every op outside a group.
type DisplayGroup = ilayout.BlendGroup

// DisplayKind is the operation kind. It is a distinct type, so a consumer
// working with kinds declares values of this type rather than plain integers.
type DisplayKind = ilayout.OpKind

// Display operation kinds. These mirror the internal OpKind values; an alias
// does not inherit the constants, so they are restated here.
const (
	// DisplayOpNoop is a deactivated operation. The clipper writes it when
	// overflow removes an operation from the page, and it paints nothing.
	// Skip it. It is 255, outside the sequence below.
	DisplayOpNoop = ilayout.OpKindNoop
	// DisplayOpUnknown is the zero kind. Layout emits it only as the boundary
	// marker that opens or closes a blend or isolation group, which carries no
	// paint; test GroupBoundary to tell a marker from an unset kind. No painter
	// matches it, so treat it as inert rather than as a fill.
	DisplayOpUnknown = ilayout.OpUnknown
	// DisplayOpFillRect is a filled rectangle, with per-corner radii for a
	// rounded or elliptical box.
	DisplayOpFillRect = ilayout.OpFillRect
	// DisplayOpStrokeRect is a stroked rectangle. StrokeMask selects sides
	// when non-zero; zero means the complete rounded rectangle.
	DisplayOpStrokeRect = ilayout.OpStrokeRect
	// DisplayOpLine is one stroked segment. Width is in points.
	DisplayOpLine = ilayout.OpLine
	// DisplayOpText is one shaped text run. Y is the baseline.
	DisplayOpText = ilayout.OpText
	// DisplayOpImage is one image with its encoded payload and pixel bounds.
	// Read the payload with ImageBytes, not the promoted fields.
	DisplayOpImage = ilayout.OpImage
	// DisplayOpLinkURI is a link target; it paints nothing itself.
	DisplayOpLinkURI = ilayout.OpLinkURI
	// DisplayOpBullet is a generated list marker. Y is the baseline.
	DisplayOpBullet = ilayout.OpBullet
	// DisplayOpGridRun is one table row's collapsed border grid, carried as
	// ordered segments in Grid. Replay the segments in order.
	DisplayOpGridRun = ilayout.OpGridRun
)

// Display is a retained display list plus the canvas it was laid out for.
type Display struct {
	// Ops are the operations in source order. Replay them in the order
	// DisplayOrder returns, not in this order, so z-index, the outline paint
	// layer, and the chrome-below-content rule are honored.
	Ops []DisplayOp

	// Order indexes Ops in paint order, ready to iterate.
	Order []int

	// Boxes are the element border boxes from the same placement, in document
	// order, in CSS pixels. They match the boxes Lay returns, so a replaying
	// caller gets hit testing without rasterizing.
	Boxes []Box

	// Width and Height are the canvas size in CSS pixels, taken from the same
	// placement Lay builds, and Height applies the same requested minimum.
	// Height is converted straight from points while Lay reads the size back
	// off the painted picture, so the two can differ by one pixel.
	Width, Height int

	// PointsPerPixel and PixelPerPoint convert between the op coordinate space
	// (points) and the canvas size above (CSS pixels).
	PointsPerPixel, PixelPerPoint float64
}

// DisplayList lays doc out and returns its display list without rasterizing
// it. Lay does the same placement and then paints it to an image; DisplayList
// stops before the paint, so a caller can replay the vector operations onto
// its own canvas (a GPU, a PDF, a display list of its own) and keep text as
// glyphs rather than as pixels.
//
// DisplayList returns no image. Callers that still want a bitmap should keep
// calling Lay; the two share one placement path, so boxes and geometry agree.
//
// One deliberate difference: Lay also applies the raster budget, so a canvas
// taller or wider than the engine's pixel limit makes Lay fail with a resource
// error. DisplayList never rasterizes, so it returns that canvas instead. A
// caller that hands the result to a GPU should apply its own size limit.
func DisplayList(ctx context.Context, doc *css.Document) (*Display, error) {
	return DisplayListOptions(ctx, doc, Options{Images: nil})
}

// DisplayListOptions is DisplayList with the optional image resolver.
func DisplayListOptions(ctx context.Context, doc *css.Document, options Options) (*Display, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: context: %w", err)
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return nil, ErrNilDocument
	}

	//nolint:exhaustruct // fixed viewport, backgrounds on, no crop, no smart width
	opts := imageout.RenderOptions{
		Width:      styled.WidthPx,
		Height:     styled.HeightPx,
		Sheets:     styled.Sheets,
		Media:      styled.Media,
		Background: true,
		Registry:   styled.Registry,
		State:      styled.State,
		Images:     options.Images,
	}

	res, err := imageout.LayoutResult(ctx, styled.Root, opts)
	if err != nil {
		return nil, fmt.Errorf("layout: display: %w", err)
	}

	// The canvas is as tall as the taller of the content and the requested
	// minimum, matching what RenderLayout would have produced for Lay.
	heightPt := imageout.CanvasHeight(res, opts)

	return &Display{
		Ops:            res.Ops,
		Order:          ilayout.PaintOrder(res.Ops),
		Boxes:          boxesFrom(ilayout.PlacedElements(res)),
		Width:          int(res.Width * ptToPx),
		Height:         int(heightPt * ptToPx),
		PointsPerPixel: ptToPx,
		PixelPerPoint:  1 / ptToPx,
	}, nil
}

// DisplayOrder returns the indices that put ops in paint order. It is the
// same policy the PDF writer and the rasterizer use, so a replay matches
// their layering.
func DisplayOrder(ops []DisplayOp) []int {
	return ilayout.PaintOrder(ops)
}

// DisplayFakeBold reports whether a text op must synthesize weight by
// double-striking, because its face is upright but the op is bold. The check
// is CJK-aware and honors the no-fake-bold gate, so a caller should ask rather
// than test op.Bold itself.
func DisplayFakeBold(op *DisplayOp) bool {
	return ilayout.FakeBoldFor(op)
}

// DisplayTransformText applies the CSS text-transform named by an op and
// returns the text to draw. It returns the input unchanged for "none" and for
// an unrecognized value.
func DisplayTransformText(text, transform string) string {
	return ilayout.TransformInlineText(text, transform)
}
