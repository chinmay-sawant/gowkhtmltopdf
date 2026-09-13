package layout

import (
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// groupFrame is one open element group while a page fragment is painted.
type groupFrame struct {
	group   *BlendGroup
	content *pdf.Content
}

// groupStack tracks CSS element groups across the paint of one page or header
// band. Operations carry their group pointer, so page splits and paint-order
// reordering cannot detach a fragment from its group: each page fragment is
// buffered and composited on its own (the page-fragment subset), and no
// operation is ever dropped. Frames nest by the group parent chain, not by
// open order, so a sibling operation that paint order sorts between a group's
// members still paints into the parent stream. Groups nest: an inner group's
// Form XObject is emitted into the enclosing group's buffer.
type groupStack struct {
	page      *pdf.Page
	bbox      [4]float64
	remaining map[*BlendGroup]int
	frames    []groupFrame
}

func newGroupStack(page *pdf.Page) groupStack {
	stack := groupStack{ //nolint:exhaustruct // remaining and frames fill during painting
		page: page,
	}
	if page != nil {
		// The form content paints in page coordinates, so the page box is a
		// conservative bounding box that never clips a group fragment.
		stack.bbox = [4]float64{0, 0, page.Width(), page.Height()}
	}

	return stack
}

// hasBlendGroups reports whether any op belongs to a CSS element group or
// marks one. Paint skips all group bookkeeping when this is false, keeping the
// ungrouped path a no-op.
func hasBlendGroups(ops []Op) bool {
	for i := range ops {
		if ops[i].Group() != nil || ops[i].GroupBoundary() != 0 {
			return true
		}
	}

	return false
}

// groupPaints reports whether the operation writes pixels a group must own.
func groupPaints(op *Op) bool {
	if op == nil || op.GroupBoundary() != 0 {
		return false
	}

	return op.Kind != opKindNoop && op.Kind != OpLinkURI
}

// prepare counts, per page fragment, how many paint operations reference each
// group. A group closes after its last counted operation has been painted.
func (s *groupStack) prepare(ops []Op, orders ...[]int) {
	for _, order := range orders {
		for _, idx := range order {
			if idx < 0 || idx >= len(ops) {
				continue
			}

			if !groupPaints(&ops[idx]) {
				continue
			}

			for group := ops[idx].Group(); group != nil; group = group.Parent {
				if s.remaining == nil {
					s.remaining = map[*BlendGroup]int{}
				}

				s.remaining[group]++
			}
		}
	}
}

// target returns the content stream that accepts op's paint: the buffer owned
// by the operation's own group chain, or fallback when the operation belongs
// to no group. Routing by the operation's group, not by the innermost open
// frame, is what keeps a plain sibling that paint order sorts between a
// group's operations from being captured by that group's buffer.
func (s *groupStack) target(op *Op, fallback *pdf.Content) *pdf.Content {
	if group := op.Group(); group != nil {
		if content := s.frameContent(group); content != nil {
			return content
		}
	}

	return fallback
}

// frameContent returns the open frame's buffer for group, or nil.
func (s *groupStack) frameContent(group *BlendGroup) *pdf.Content {
	for idx := len(s.frames) - 1; idx >= 0; idx-- {
		if s.frames[idx].group == group {
			return s.frames[idx].content
		}
	}

	return nil
}

// enter opens every group buffer in the operation's chain that is not open
// yet, outermost first. Each new frame nests inside its parent group's buffer
// (or fallback at the page root), so two sibling groups become two sibling
// forms no matter how their operations interleave in paint order.
func (s *groupStack) enter(op *Op, fallback *pdf.Content) {
	group := op.Group()
	if group == nil || s.page == nil || s.page.Doc() == nil {
		return
	}

	for _, node := range groupChain(group) {
		if s.frameContent(node) != nil {
			continue
		}

		parent := fallback
		if content := s.frameContent(node.Parent); content != nil {
			parent = content
		}

		s.frames = append(s.frames, groupFrame{group: node, content: parent.BeginTransparencyGroup()})
	}
}

// leave decrements the operation's group counts and closes every group whose
// last fragment operation has now been painted. A frame closes into its parent
// group's buffer (or the fallback stream at the root) once no open descendant
// still needs it, so closing order never reparents an open sibling.
func (s *groupStack) leave(op *Op, fallback *pdf.Content) error {
	group := op.Group()
	if group == nil {
		return nil
	}

	for node := group; node != nil; node = node.Parent {
		s.remaining[node]--
	}

	return s.closeReady(fallback)
}

// closeReady composes every frame whose counted operations have all painted,
// deepest first so a child form lands in its parent's buffer before the parent
// closes. Counts guarantee remaining[parent] >= remaining[child], so a parent
// only becomes ready after every descendant frame it owns.
func (s *groupStack) closeReady(fallback *pdf.Content) error {
	for {
		ready, readyDepth := -1, -1

		for idx := range s.frames {
			if s.remaining[s.frames[idx].group] > 0 {
				continue
			}

			if depth := groupDepth(s.frames[idx].group); depth > readyDepth {
				ready, readyDepth = idx, depth
			}
		}

		if ready < 0 {
			return nil
		}

		frame := s.frames[ready]
		s.frames = append(s.frames[:ready], s.frames[ready+1:]...)

		parent := fallback
		if content := s.frameContent(frame.group.Parent); content != nil {
			parent = content
		}

		if err := parent.EndTransparencyGroup(frame.content, frame.group.Mode, s.bbox); err != nil {
			return fmt.Errorf("layout: transparency group: %w", err)
		}
	}
}

// groupChain returns group's ancestors, root first.
func groupChain(group *BlendGroup) []*BlendGroup {
	depth := groupDepth(group)
	chain := make([]*BlendGroup, depth)

	for node := group; node != nil; node = node.Parent {
		depth--
		chain[depth] = node
	}

	return chain
}

// groupDepth returns the number of groups in group's parent chain.
func groupDepth(group *BlendGroup) int {
	depth := 0

	for node := group; node != nil; node = node.Parent {
		depth++
	}

	return depth
}
