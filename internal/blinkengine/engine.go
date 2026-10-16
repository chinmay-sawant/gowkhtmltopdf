// Package blinkengine runs the blinkless layout engine
// (github.com/chinmay-sawant/blinkless) and adapts the drawing list it
// returns to this module's display-list operations, so the existing painter
// and PDF writer produce the PDF.
//
// The pipeline is blinkless's documented one: its html.Parse reads the
// document, its css.Apply applies the stylesheets the document carries, and
// its layout.DisplayList places the styled tree and stops at the drawing
// list. This package parses no HTML, cascades no CSS, and places no boxes of
// its own. It copies the drawing list onto this module's layout.Op records
// (kinds, geometry, colors, faces, and rare payloads) and hands the copy back
// as a layout.Result. Convert then paginates and paints it exactly like a
// result from the native engine, which stays the default and is untouched.
//
// Known limits against the native engine, stated rather than papered over:
// a blinkless result carries no element box tree, so page outline headings
// and table-of-contents entries stay empty; smart shrinking and @page margin
// boxes do not apply; mask groups currently use only Blinkless's first
// resolved layer; perspective image operations fail because they cannot be
// represented by the current vector display-list geometry. Caller-supplied
// link sheets apply after the document's own <style> blocks regardless of
// source order, because blinkless applies document sheets itself.
package blinkengine

import (
	"context"
	"errors"
	"fmt"
	"math"

	bcss "github.com/chinmay-sawant/blinkless/css"
	bhtml "github.com/chinmay-sawant/blinkless/html"
	blayout "github.com/chinmay-sawant/blinkless/layout"
	ilayout "github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// pxPerPoint converts canvas points to CSS pixels: 1pt = 96/72px at 96 dpi,
// the factor blinkless reports as Display.PixelPerPoint.
const pxPerPoint = 96.0 / 72.0

var (
	// errNilContext reports a nil context.
	errNilContext = errors.New("blinkengine: nil context")

	// errBadViewport reports a viewport that cannot be laid out.
	errBadViewport = errors.New("blinkengine: viewport must be positive")
)

// Options configures one blinkless layout pass.
type Options struct {
	// WidthPt and HeightPt are the page content box in points. They become
	// the CSS-pixel viewport blinkless lays out in.
	WidthPt, HeightPt float64

	// Media is the stylesheet media type: "print" or "screen". The PDF path
	// passes the resolved media of the object; an empty value means screen to
	// blinkless.
	Media string

	// Sheets are extra stylesheets as raw CSS text, applied after the
	// document's own <style> blocks (blinkless's Extra option). The caller
	// fetches, media-filters, and import-expands them; one unparseable sheet
	// fails the layout rather than being dropped silently.
	Sheets []string

	// Images resolves an image source to its encoded bytes (PNG, JPEG, or
	// SVG). Nil skips every image, exactly like blinkless's own nil resolver,
	// and a resolver error skips that one image without failing the layout.
	Images func(src string) ([]byte, error)
}

// parseSheets parses the caller-supplied extra stylesheets in order. The
// caller already validated the CSS with this module's parser; a blinkless
// parse failure here is a bridge defect, so it fails the layout instead of
// silently dropping styles.
func parseSheets(sheets []string) ([]*bcss.Sheet, error) {
	if len(sheets) == 0 {
		return nil, nil
	}

	out := make([]*bcss.Sheet, 0, len(sheets))

	for idx, text := range sheets {
		sheet, err := bcss.Parse(text)
		if err != nil {
			return nil, fmt.Errorf("blinkless: parse stylesheet %d: %w", idx, err)
		}

		out = append(out, sheet)
	}

	return out, nil
}

// Layout parses source with blinkless, applies its stylesheets, places the
// document, and returns the mapped drawing list as a layout.Result. Result
// height is the taller of the laid-out content and HeightPt, converted from
// the CSS pixels blinkless reports.
func Layout(ctx context.Context, source []byte, opts Options) (*ilayout.Result, error) {
	if ctx == nil {
		return nil, errNilContext
	}

	if opts.WidthPt <= 0 || opts.HeightPt <= 0 {
		return nil, fmt.Errorf("%w: %g x %gpt", errBadViewport, opts.WidthPt, opts.HeightPt)
	}

	tree, err := bhtml.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("blinkless: parse html: %w", err)
	}

	extra, err := parseSheets(opts.Sheets)
	if err != nil {
		return nil, err
	}

	styled, err := bcss.Apply(ctx, tree, bcss.Options{ //nolint:exhaustruct // viewport, media, and extra sheets only
		WidthPx:  int(math.Round(opts.WidthPt * pxPerPoint)),
		HeightPx: int(math.Round(opts.HeightPt * pxPerPoint)),
		Media:    opts.Media,
		Extra:    extra,
	})
	if err != nil {
		return nil, fmt.Errorf("blinkless: style: %w", err)
	}

	display, err := blayout.DisplayListOptions(ctx, styled, blayout.Options{
		Images: opts.Images,
	})
	if err != nil {
		return nil, fmt.Errorf("blinkless: layout: %w", err)
	}

	ops, err := newOpMapper().ops(display.Ops)
	if err != nil {
		return nil, fmt.Errorf("blinkless: adapt: %w", err)
	}

	return &ilayout.Result{ //nolint:exhaustruct // exported display fields only; the box tree is nil by design
		Ops:    ops,
		Width:  opts.WidthPt,
		Height: float64(display.Height) * display.PointsPerPixel,
	}, nil
}
