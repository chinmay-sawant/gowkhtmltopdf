package css

import (
	"context"
	"fmt"

	icss "github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// Relayout restyles doc for a new viewport and pointer state without
// reparsing the HTML or the stylesheet text. It shares doc's tree, which must
// not change between Apply and Relayout, and reuses its parsed sheets and
// font registry. Only the viewport-dependent collection gating for <link
// media> and @import media runs again. The cascade itself still runs in
// layout, because @media rules, vw/vh lengths, and percentages resolve
// against the viewport stored here.
//
// Use the result for the next layout and hit test. A source or theme change
// needs a fresh Apply, because Relayout keeps doc's tree and extra sheets. A
// nil doc returns ErrNilDocument and a non-positive size returns ErrBadSize.
func Relayout(
	ctx context.Context, doc *Document,
	width, height int, focus, hover, active string,
) (*Document, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("css: context: %w", err)
	}

	if doc == nil {
		return nil, ErrNilDocument
	}

	if width <= 0 || height <= 0 {
		return nil, ErrBadSize
	}

	if doc.root == nil {
		return nil, errUnreadable
	}

	media, kind, err := mediaType(doc.media)
	if err != nil {
		return nil, err
	}

	opts := Options{
		WidthPx:  width,
		HeightPx: height,
		Media:    kind,
		Extra:    doc.extra,
		Focus:    focus,
		Hover:    hover,
		Active:   active,
	}

	sheets, registry, err := collect(ctx, doc.root, opts, kind, media, doc.cache, doc)
	if err != nil {
		return nil, err
	}

	return &Document{
		root:     doc.root,
		sheets:   sheets,
		registry: registry,
		media:    kind,
		widthPx:  width,
		heightPx: height,
		state:    icss.MatchState{Focus: focus, Hover: hover, Active: active},
		extra:    doc.extra,
		cache:    doc.cache,
	}, nil
}
