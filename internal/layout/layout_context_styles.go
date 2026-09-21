package layout

import (
	"context"
	"errors"
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// ContextWithStyles renders the document with a caller-provided cascade.
// styles must come from ResolveStyles over the same root and options; they are
// reused only when the sheets carry no @container rules, otherwise the full
// container remount runs and the argument is ignored. Zoom changes do not
// invalidate them because opts.Zoom only feeds engine.scale (layout.go:1174).
func ContextWithStyles(
	ctx context.Context, root *html.Node, opts Options,
	styles map[*html.Node]*ResolvedStyle,
) (*Result, error) {
	return layoutContextWithStyles(ctx, root, opts, nil, styles)
}

// layoutContextWithStyles is the shared body of LayoutContext, WithWorkspace,
// and ContextWithStyles. A non-nil styles map skips style resolution
// unless the sheets carry @container rules, in which case the map is ignored
// and containers are populated by the full remount.
//
//nolint:cyclop // layout preflight and staged style/container passes are explicit lifecycle gates.
func layoutContextWithStyles(
	ctx context.Context,
	root *html.Node, opts Options, workspace *Workspace,
	styles map[*html.Node]*ResolvedStyle,
) (*Result, error) {
	if root == nil {
		return nil, errors.New("layout: nil root") //nolint:err113 // static sentinel-free message matches legacy behavior
	}

	if err := opts.validate(); err != nil {
		return nil, err
	}

	if ctx == nil {
		return nil, errs.ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: context: %w", err)
	}

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		return nil, fmt.Errorf("layout: load default faces: %w", err)
	}

	if opts.Faces != nil {
		faces = opts.Faces
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: context: %w", err)
	}

	font := opts.Font
	if font == nil {
		font = faces.Regular
	}

	var containers map[*html.Node]sizeContainer

	if styles == nil || css.HasContainerRules(opts.Sheets) {
		styles, containers, err = resolveStylesForLayoutContext(ctx, root, opts)
		if err != nil {
			return nil, fmt.Errorf("layout: style resolution: %w", err)
		}
	}

	var ops []Op

	if workspace == nil || cap(workspace.ops) == 0 {
		ops = make([]Op, 0, estimateOpCapacity(root))
	} else {
		ops = workspace.ops[:0]
	}

	return finalizeResult(newEngine(ctx, opts, faces, font, styles, containers, ops), root, opts)
}
