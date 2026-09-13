package imageout

import (
	"context"
	"fmt"
	"image"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// hasElementGroups reports whether any op belongs to a CSS element group.
// Callers keep their exact pre-group op loop when this is false, so the
// ungrouped hot path gains no work and no allocations.
func hasElementGroups(ops []layout.Op) bool {
	for i := range ops {
		if ops[i].Group() != nil || ops[i].GroupBoundary() != 0 {
			return true
		}
	}

	return false
}

// rasterFrame is one element group open during grouped rasterization.
type rasterFrame struct {
	group   *layout.BlendGroup
	scratch *image.NRGBA
}

// paintWithElementGroups paints one paint-ordered op list, buffering every
// CSS element group into a scratch image and compositing that buffer once
// with the group's blend mode. Group contents composite on a transparent
// initial backdrop, which is the isolated-group semantics the HTML
// compositing model specifies. offsetPt applies canvas padding to a copy of
// each op (0 when the caller already offset them).
//
// Every op routes to the scratch owned by its own group chain (or dst when it
// belongs to no group), so a sibling that paint order sorts between a group's
// members still lands on the parent canvas instead of being absorbed by the
// group buffer.
//
// The scratch spans the union of the group's op bounds clipped to dst, so a
// group never drops content and never allocates the full canvas.
//
//nolint:cyclop // open, paint, and close phases share one op-ordered walk
func paintWithElementGroups(
	ctx context.Context, dst *image.NRGBA, ops []layout.Op, offsetPt, pxPerPt float64,
	atlas *glyphAtlas, imageCache *rasterImageCache,
) error {
	order := rasterPaintOrder(ops)

	bounds, remaining := groupPaintCensus(ops, order, offsetPt, pxPerPt, dst.Bounds())

	frames := make([]rasterFrame, 0, initialGroupFrameCap)

	for _, opIndex := range order {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("imageout: context: %w", err)
		}

		paintOp := ops[opIndex]
		if offsetPt != 0 {
			offsetPaintOp(&paintOp, offsetPt)
		}

		if paintOp.GroupBoundary() != 0 || paintOp.Kind == layout.OpLinkURI {
			continue
		}

		if paintOpBounds(&paintOp, pxPerPt).Intersect(dst.Bounds()).Empty() {
			continue
		}

		group := paintOp.Group()
		if group != nil {
			openGroupFrames(&frames, bounds, group)
		}

		target := dst
		if idx := frameIndex(frames, group); group != nil && idx >= 0 {
			target = frames[idx].scratch
		}

		if target != nil {
			paint(target, &paintOp, pxPerPt, atlas, imageCache)
		}

		if group != nil {
			closeFinishedFrames(&frames, remaining, dst, group)
		}
	}

	return nil
}

// initialGroupFrameCap is the small allocation every grouped rasterization
// starts with; deeper nesting grows the slice.
const initialGroupFrameCap = 4

// openGroupFrames opens every missing frame in group's ancestor chain,
// outermost first. Frames nest by the group parent chain, not by open order,
// so two sibling groups stay siblings even when their ops interleave.
func openGroupFrames(frames *[]rasterFrame, bounds map[*layout.BlendGroup]image.Rectangle, group *layout.BlendGroup) {
	for _, node := range elementGroupChain(group) {
		if frameIndex(*frames, node) >= 0 {
			continue
		}

		*frames = append(*frames, rasterFrame{group: node, scratch: newGroupScratch(bounds[node])})
	}
}

// frameIndex returns the open frame index for group, or -1.
func frameIndex(frames []rasterFrame, group *layout.BlendGroup) int {
	for idx := range frames {
		if frames[idx].group == group {
			return idx
		}
	}

	return -1
}

// closeFinishedFrames decrements group's member counts and composites every
// group whose last counted member has been painted, deepest first. Counts
// guarantee remaining[parent] >= remaining[child], so a parent only closes
// after the child forms it owns have landed.
//
//nolint:cyclop // deepest-ready scan plus parent-frame lookup is one close walk
func closeFinishedFrames(
	frames *[]rasterFrame, remaining map[*layout.BlendGroup]int, dst *image.NRGBA, group *layout.BlendGroup,
) {
	for node := group; node != nil; node = node.Parent {
		remaining[node]--
	}

	for {
		ready, readyDepth := -1, -1

		for idx := range *frames {
			if remaining[(*frames)[idx].group] > 0 {
				continue
			}

			if depth := elementGroupDepth((*frames)[idx].group); depth > readyDepth {
				ready, readyDepth = idx, depth
			}
		}

		if ready < 0 {
			return
		}

		top := (*frames)[ready]
		*frames = append((*frames)[:ready], (*frames)[ready+1:]...)

		parent := dst
		if idx := frameIndex(*frames, top.group.Parent); idx >= 0 && (*frames)[idx].scratch != nil {
			parent = (*frames)[idx].scratch
		}

		if top.scratch != nil && parent != nil {
			compositeBlend(parent, top.scratch, top.group.Mode)
		}
	}
}

// newGroupScratch allocates the transparent initial backdrop for one group
// fragment; an empty clause skips the allocation entirely.
func newGroupScratch(rect image.Rectangle) *image.NRGBA {
	if rect.Empty() {
		return nil
	}

	return image.NewNRGBA(rect)
}

// groupPaintCensus returns the union of each group's member op bounds clipped
// to clip and the number of counted members. The paint loop uses the same
// bounds predicate, so every counted member is painted exactly once.
func groupPaintCensus(
	ops []layout.Op, order []int, offsetPt, pxPerPt float64, clip image.Rectangle,
) (map[*layout.BlendGroup]image.Rectangle, map[*layout.BlendGroup]int) {
	bounds := map[*layout.BlendGroup]image.Rectangle{}
	remaining := map[*layout.BlendGroup]int{}

	for _, opIndex := range order {
		paintOp := ops[opIndex]
		if offsetPt != 0 {
			offsetPaintOp(&paintOp, offsetPt)
		}

		if paintOp.GroupBoundary() != 0 || paintOp.Group() == nil || paintOp.Kind == layout.OpLinkURI {
			continue
		}

		rect := paintOpBounds(&paintOp, pxPerPt).Intersect(clip)
		if rect.Empty() {
			continue
		}

		for group := paintOp.Group(); group != nil; group = group.Parent {
			if existing, ok := bounds[group]; ok {
				bounds[group] = existing.Union(rect)
			} else {
				bounds[group] = rect
			}

			remaining[group]++
		}
	}

	return bounds, remaining
}

// elementGroupChain returns group's ancestors, root first.
func elementGroupChain(group *layout.BlendGroup) []*layout.BlendGroup {
	depth := elementGroupDepth(group)
	chain := make([]*layout.BlendGroup, depth)

	for node := group; node != nil; node = node.Parent {
		depth--
		chain[depth] = node
	}

	return chain
}

// elementGroupDepth returns the number of groups in group's parent chain.
func elementGroupDepth(group *layout.BlendGroup) int {
	depth := 0

	for node := group; node != nil; node = node.Parent {
		depth++
	}

	return depth
}
