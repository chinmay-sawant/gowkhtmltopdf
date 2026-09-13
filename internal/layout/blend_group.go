package layout

// GroupMark values flag display-list operations that only delimit a blend
// group. A position:sticky or pagination pass may deactivate a marker's Kind,
// so painters must test the mark, not the op kind.
const (
	// groupMarkBegin precedes the first operation of a grouped element.
	groupMarkBegin uint8 = 1
	// groupMarkEnd follows the last operation of a grouped element.
	groupMarkEnd uint8 = 2
)

// BlendGroup describes one element compositing group created by
// mix-blend-mode or isolation: isolate. Every operation painted inside the
// group carries a pointer to it. PDF and PNG painters buffer those operations
// and composite the group once, which is the element-group semantics the CSS
// compositing model requires.
//
// Parent chains nested groups: an inner group's operations are buffered
// inside the enclosing group's buffer, so siblings in different groups never
// blend with each other's backdrop.
type BlendGroup struct {
	// ID is the stable creation-order identity of the group. Nested groups
	// have increasing IDs; tests use it to check nesting.
	ID int
	// Mode is the CSS mix-blend-mode, "" or "normal" when the element only
	// declared isolation: isolate.
	Mode string
	// Isolate marks a group whose initial backdrop is transparent. HTML
	// stacking contexts are always isolated groups (CSS Compositing 3.2), so
	// engine-created groups set this true.
	Isolate bool
	// Parent is the enclosing group, nil at the page root.
	Parent *BlendGroup
}

// IsGroupBegin reports whether the operation opens a blend group.
func (op *Op) IsGroupBegin() bool {
	return op.GroupBoundary() == groupMarkBegin
}

// IsGroupEnd reports whether the operation closes a blend group.
func (op *Op) IsGroupEnd() bool {
	return op.GroupBoundary() == groupMarkEnd
}

// Group returns the operation's owning blend group, nil when the operation
// carries no payload or belongs to no group. Nil-safe: hand-built display
// lists never bound the shared empty extra.
func (op *Op) Group() *BlendGroup {
	if op == nil || op.opExtra == nil {
		return nil
	}

	return op.opExtra.BlendGroup
}

// GroupBoundary returns groupMarkBegin or groupMarkEnd for a group marker and
// 0 for paint operations. Nil-safe like Group.
func (op *Op) GroupBoundary() uint8 {
	if op == nil || op.opExtra == nil {
		return 0
	}

	return op.opExtra.GroupMark
}

// SetBlendGroup points the operation at the element group that owns it.
// Operations outside any group keep a nil group and paint directly.
func (op *Op) SetBlendGroup(group *BlendGroup) {
	if group == nil {
		return
	}

	op.setBlendGroup(group)
}

func (op *Op) setBlendGroup(group *BlendGroup) {
	if group == nil {
		return
	}

	if op.opExtra != nil && op.opExtra != emptyExtra && op.opExtra.BlendGroup == group {
		return
	}

	op.detachExtra().BlendGroup = group
}

func (op *Op) setGroupMark(group *BlendGroup, mark uint8) {
	extra := op.detachExtra()
	extra.BlendGroup = group
	extra.GroupMark = mark
}
