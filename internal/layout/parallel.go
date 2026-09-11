package layout

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// ParallelLayout is the opt-in concurrency entry point. It runs the same
// cascade as LayoutContext once, certifies a run of independent
// page-break-before body blocks, builds those blocks on a bounded worker pool,
// and merges the parts into one un-paginated Result. Documents that are not
// certified, and all PDF/UA callers, use the serial path unchanged.
//
// The merged Result is handed to the existing PaintContext; pagination, page
// names, locations, links, outline, and headers/footers are computed once on
// the merged list, exactly as the serial path does.
func ParallelLayout(ctx context.Context, root *html.Node, opts Options, workers int) (*Result, error) {
	if root == nil {
		return nil, errors.New("layout: nil root") //nolint:err113 // matches the legacy sentinel-free message
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

	styles, containers, err := resolveStylesForLayoutContext(ctx, root, opts)
	if err != nil {
		return nil, fmt.Errorf("layout: style resolution: %w", err)
	}

	plan := detectParallelSections(root, styles, opts, containers)
	if plan == nil {
		// Serial fallback reuses the resolution already paid for.
		return finalizeResult(
			newEngine(ctx, opts, faces, font, styles, containers,
				make([]Op, 0, estimateOpCapacity(root))),
			root, opts,
		)
	}

	return parallelBuild(ctx, opts, faces, font, styles, containers, plan, workers)
}

// parallelPlan is the certified independent-block run.
type parallelPlan struct {
	sections []*html.Node
}

// detectParallelSections certifies an ordered, gap-free run of independent
// page-break-before body blocks. It returns nil when any condition fails; the
// caller then uses the serial path. The conditions are the design note's
// B-conditions for stage 1.
//
//nolint:cyclop,funlen // detector gates are a flat checklist
func detectParallelSections(
	root *html.Node, styles map[*html.Node]*ResolvedStyle, opts Options,
	containers map[*html.Node]sizeContainer,
) *parallelPlan {
	if root == nil || styles == nil {
		return nil
	}

	// @container gates are measured per document; a candidate that depends on
	// an external container is not isolated.
	if containers != nil || css.HasContainerRules(opts.Sheets) {
		return nil
	}

	htmlNode := firstElementChild(root, "html")
	if htmlNode == nil {
		return nil
	}

	body := firstElementChild(htmlNode, "body")
	if body == nil {
		return nil
	}

	if !parallelShellStyleOK(styles[htmlNode]) || !parallelShellStyleOK(styles[body]) {
		return nil
	}

	sections := make([]*html.Node, 0, len(body.Children))

	for _, child := range body.Children {
		switch child.Type {
		case html.NodeUnknown, html.DoctypeNode:
			return nil
		case html.TextNode:
			if strings.TrimSpace(child.Text) != "" {
				return nil
			}
		case html.CommentNode:
		case html.ElementNode:
			if !parallelCandidateStyleOK(styles[child]) {
				return nil
			}

			sections = append(sections, child)
		default:
			return nil
		}
	}

	if len(sections) < minParallelSections {
		return nil
	}

	for idx, section := range sections {
		if idx > 0 && styles[section].PageBreakBefore != pageBreakAlways {
			return nil
		}

		if !parallelSubtreeOK(section, styles) {
			return nil
		}
	}

	return &parallelPlan{sections: sections}
}

// minParallelSections is the smallest certified run worth fanning out.
const minParallelSections = 2

// firstElementChild returns the first direct element child with the given
// name.
func firstElementChild(node *html.Node, name string) *html.Node {
	if node == nil {
		return nil
	}

	for _, child := range node.Children {
		if child.Type == html.ElementNode && child.Name == name {
			return child
		}
	}

	return nil
}

// parallelShellStyleOK requires the html/body shell to contribute no geometry
// or chrome: zero margins, padding, and borders, a normal block box, and no
// stacking-context or formatting-context switch.
//
//nolint:cyclop // the shell checklist reads better as one flat guard
func parallelShellStyleOK(sty *ResolvedStyle) bool {
	if sty == nil {
		return false
	}

	if sty.MarginTop != 0 || sty.MarginRight != 0 || sty.MarginBottom != 0 || sty.MarginLeft != 0 {
		return false
	}

	if sty.PaddingTop != 0 || sty.PaddingRight != 0 || sty.PaddingBottom != 0 || sty.PaddingLeft != 0 {
		return false
	}

	if sty.BorderTop.Width != 0 || sty.BorderRight.Width != 0 ||
		sty.BorderBottom.Width != 0 || sty.BorderLeft.Width != 0 {
		return false
	}

	if sty.Float != cssDisplayNone || sty.Position != positionStatic || sty.Display != displayBlock {
		return false
	}

	if sty.WritingMode != writingModeHorizontalTB || sty.ColumnCount != 0 {
		return false
	}

	return !parallelStackingContextFree(sty)
}

// parallelCandidateStyleOK is the per-candidate slice of the detector: a
// block-level in-flow box with zero vertical margins and no boundary feature
// the merge cannot reproduce.
func parallelCandidateStyleOK(sty *ResolvedStyle) bool {
	if sty == nil {
		return false
	}

	if sty.Display != displayBlock || sty.Float != cssDisplayNone || sty.Position != positionStatic {
		return false
	}

	if sty.MarginTop != 0 || sty.MarginBottom != 0 || sty.MarginTopAuto || sty.MarginBottomAuto {
		return false
	}

	return !parallelStackingContextFree(sty)
}

// parallelStackingContextFree rejects styles that create a stacking context or
// transform containing block; an isolated engine starts clean while the serial
// engine carries ancestor state.
func parallelStackingContextFree(sty *ResolvedStyle) bool {
	if sty.HasTransform || sty.Opacity < 1 || sty.Isolation == "isolate" {
		return false
	}

	if sty.MixBlendMode != blendNormal || sty.Filter != "" {
		return false
	}

	return !sty.ZIndexSet
}

// parallelSubtreeOK rejects any descendant whose layout depends on state
// outside its candidate: floats, out-of-flow positioning, transforms, sticky
// or fixed boxes, counters, list markers, and container types.
//
//nolint:cyclop,nestif // subtree disqualifiers are a flat checklist
func parallelSubtreeOK(node *html.Node, styles map[*html.Node]*ResolvedStyle) bool {
	if node == nil {
		return true
	}

	if node.Type == html.ElementNode {
		sty := styles[node]
		if sty == nil {
			return false
		}

		if sty.Display != cssDisplayNone {
			if sty.Float != cssDisplayNone ||
				sty.Position == positionAbsolute || sty.Position == positionFixed || sty.Position == positionSticky {
				return false
			}

			if !parallelStackingContextFree(sty) {
				return false
			}

			if sty.CounterReset != "" || sty.CounterIncrement != "" || sty.Display == displayListItem {
				return false
			}

			if sty.ContainerType != "" || contentNeedsEnv(sty.Content) {
				return false
			}
		}
	}

	for _, child := range node.Children {
		if !parallelSubtreeOK(child, styles) {
			return false
		}
	}

	return true
}

// parallelPart is one built section: its ops, flattened boxes, and root box.
type parallelPart struct {
	ops   []Op
	boxes []*box
	root  *box
}

// parallelBuild fans the certified sections out over a bounded worker pool and
// merges the parts in document order.
//
//nolint:cyclop,funlen // worker dispatch plus error collection
func parallelBuild(
	ctx context.Context, opts Options, faces *pdf.FaceSet, font *pdf.Font,
	styles map[*html.Node]*ResolvedStyle, containers map[*html.Node]sizeContainer,
	plan *parallelPlan, workers int,
) (*Result, error) {
	parts := make([]*parallelPart, len(plan.sections))
	errs := make([]error, len(plan.sections))

	workerCount := workers
	if workerCount < 1 {
		workerCount = 1
	}

	if workerCount > len(plan.sections) {
		workerCount = len(plan.sections)
	}

	if workerCount > runtime.GOMAXPROCS(0) {
		workerCount = runtime.GOMAXPROCS(0)
	}

	var next int

	var nextMu sync.Mutex

	var waitGroup sync.WaitGroup

	work := func() {
		defer waitGroup.Done()

		for {
			nextMu.Lock()
			idx := next
			next++
			nextMu.Unlock()

			if idx >= len(plan.sections) {
				return
			}

			if err := ctx.Err(); err != nil {
				errs[idx] = err

				continue
			}

			part, err := buildParallelPart(ctx, opts, faces, font, styles, containers, plan.sections[idx])
			errs[idx] = err
			parts[idx] = part
		}
	}

	for range workerCount {
		waitGroup.Add(1)

		go work()
	}

	waitGroup.Wait()

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: parallel build: %w", err)
	}

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return mergeParallelParts(opts, parts)
}

// buildParallelPart builds one certified section on a fresh engine that shares
// only the read-only resolution inputs.
func buildParallelPart(
	ctx context.Context, opts Options, faces *pdf.FaceSet, font *pdf.Font,
	styles map[*html.Node]*ResolvedStyle, containers map[*html.Node]sizeContainer,
	section *html.Node,
) (*parallelPart, error) {
	eng := newEngine(ctx, opts, faces, font, styles, containers, make([]Op, 0, estimateOpCapacity(section)))

	rootBox := eng.build(section, opts.Width, 0, 0)

	if eng.err != nil {
		return nil, eng.err
	}

	eng.finalizeChrome(rootBox)

	boxes := make([]*box, 0)
	flattenBoxes(rootBox, &boxes)

	return &parallelPart{ops: eng.ops, boxes: boxes, root: rootBox}, nil
}

// mergeParallelParts reconstructs the serial continuous canvas: ops in section
// order with a Y translation, rebased op IDs and box ranges, one synthetic
// root, and empty flow stores for PaintContext to rebuild.
func mergeParallelParts(opts Options, parts []*parallelPart) (*Result, error) {
	opCount, boxCount := 0, 1

	for _, part := range parts {
		if part == nil {
			//nolint:err113 // probe-only internal invariant
			return nil, errors.New("layout: missing parallel section part")
		}

		opCount += len(part.ops)
		boxCount += len(part.boxes)
	}

	root := &box{ //nolint:exhaustruct // synthetic root carries geometry only
		kind:     boxKindBlock,
		w:        opts.Width,
		children: make([]*box, 0, len(parts)),
	}
	ops := make([]Op, 0, opCount)
	boxes := make([]*box, 0, boxCount)
	boxes = append(boxes, root)

	offsetY := 0.0
	opBase := 0

	for _, part := range parts {
		for idx := range part.ops {
			op := part.ops[idx]
			op.ID = uint64(len(ops) + 1) //nolint:gosec // op count is bounded by memory
			op.Y += offsetY
			// Sticky is disqualified, so StickyID stays 0 for every op.
			ops = append(ops, op)
		}

		for _, boxNode := range part.boxes {
			boxNode.y += offsetY
			boxNode.opStart += opBase
			boxNode.opEnd += opBase
			boxes = append(boxes, boxNode)
		}

		if part.root != nil {
			root.children = append(root.children, part.root)
			offsetY += part.root.height
		}

		opBase += len(part.ops)
	}

	root.height = offsetY
	root.opStart, root.opEnd = 0, len(ops)-1

	return &Result{ //nolint:exhaustruct // flow stores start empty for PaintContext
		Ops:    ops,
		Width:  opts.Width,
		Height: offsetY,
		root:   root,
		boxes:  boxes,
	}, nil
}
