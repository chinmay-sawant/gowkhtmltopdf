package layout

import (
	"context"
	"errors"
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// ResolveStyles runs the same cascade LayoutContext uses. Convert uses it
// once so independent blocks share interned style pointers.
func ResolveStyles(ctx context.Context, root *html.Node, opts Options) (map[*html.Node]*ResolvedStyle, error) {
	if root == nil {
		return nil, errors.New("layout: nil root") //nolint:err113 // matches LayoutContext
	}

	if err := opts.validate(); err != nil {
		return nil, err
	}

	if ctx == nil {
		return nil, errs.ErrNilContext
	}

	styles, _, err := resolveStylesForLayoutContext(ctx, root, opts)
	if err != nil {
		return nil, fmt.Errorf("layout: style resolution: %w", err)
	}

	return styles, nil
}

// NodeWithWorkspace builds one already-styled node into workspace.
// styles must be the document cascade (same map IndependentBlocks saw).
//
//nolint:cyclop // faces, workspace, and Result assembly are one constructor
func NodeWithWorkspace(
	ctx context.Context,
	node *html.Node,
	opts Options,
	styles map[*html.Node]*ResolvedStyle,
	workspace *Workspace,
) (*Result, error) {
	if node == nil {
		return nil, errors.New("layout: nil node") //nolint:err113 // matches LayoutContext
	}

	if styles == nil {
		return nil, errors.New("layout: nil styles") //nolint:err113 // required shared cascade
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

	font := opts.Font
	if font == nil {
		font = faces.Regular
	}

	var ops []Op
	if workspace == nil || cap(workspace.ops) == 0 {
		ops = make([]Op, 0, estimateOpCapacity(node))
	} else {
		ops = workspace.ops[:0]
	}

	eng := newEngine(ctx, opts, faces, font, styles, nil, ops)
	rootBox := eng.build(node, opts.Width, 0, 0)

	if eng.err != nil {
		return nil, eng.err
	}

	eng.finalizeChrome(rootBox)

	return assembleNodeResult(eng, rootBox, opts, workspace), nil
}

// assembleNodeResult turns the built tree into the caller-facing Result:
// flattened boxes, final height, transform stamps, and census counters.
func assembleNodeResult(eng *engine, rootBox *box, opts Options, workspace *Workspace) *Result {
	boxes := make([]*box, 0)
	flattenBoxes(rootBox, &boxes)

	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Ops:            eng.ops,
		Width:          opts.Width,
		Height:         opts.Height,
		pageSnapHeight: eng.pageSnapHeight,
		root:           rootBox,
		boxes:          boxes,
	}
	if rootBox != nil {
		res.Height = rootBox.y + rootBox.height
	}

	if res.Height < eng.height {
		res.Height = eng.height
	}

	if eng.needsXformStamp {
		stampBoxTransforms(rootBox, IdentityMatrix(), res.Ops)
	}

	res.MaxContentX, res.HasFragmentLinks = censusOps(res.Ops, opts.Width)
	res.skipInitialBeforeAlways = true

	if workspace != nil {
		workspace.ops = res.Ops
	}

	return res
}
