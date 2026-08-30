//nolint:testpackage,wsl,varnamelen,paralleltest,prealloc,exhaustruct // overflow clip probes
package layout

import (
	"testing"
)

func TestOverflowClip(t *testing.T) {
	t.Parallel()

	t.Run("hidden", func(t *testing.T) { t.Parallel(); assertOverflowChildFillClipped(t, overflowHidden) })
	t.Run("clip", func(t *testing.T) { t.Parallel(); assertOverflowChildFillClipped(t, overflowClip) })
	t.Run("auto", func(t *testing.T) { t.Parallel(); assertOverflowChildFillClipped(t, overflowAuto) })
	t.Run("scroll", func(t *testing.T) { t.Parallel(); assertOverflowChildFillClipped(t, overflowScroll) })
	t.Run("visible", testOverflowVisibleUnclipped)
	t.Run("rect-intersect", testOverflowClipRectIntersect)
}

func assertOverflowChildFillClipped(t *testing.T, overflow string) {
	t.Helper()

	parent, fills := overflowChildFills(t, overflow)
	if parent.height > 50+1 {
		t.Fatalf("parent height = %.3f, want ~50", parent.height)
	}

	if len(fills) == 0 {
		t.Fatal("missing child fill")
	}

	pad := paddingBoxOfTest(parent)
	for _, fill := range fills {
		if fill.Kind != OpFillRect {
			continue
		}

		if fillOutsideClip(fill, pad) {
			t.Fatalf("overflow:%s child fill %+v paints outside parent padding box %+v",
				overflow, fill, pad)
		}
	}
}

func testOverflowVisibleUnclipped(t *testing.T) {
	t.Parallel()

	_, fills := overflowChildFills(t, visibleKeyword)
	var tall bool

	for _, fill := range fills {
		if fill.H >= 199 {
			tall = true

			break
		}
	}

	if !tall {
		t.Fatalf("overflow:visible should keep the 200pt child fill, got %+v", fills)
	}
}

func testOverflowClipRectIntersect(t *testing.T) {
	t.Parallel()

	op := Op{Kind: OpFillRect, X: 0, Y: 0, W: 80, H: 200} //nolint:exhaustruct // clip geometry
	clipPaintOp(&op, clipRect{x: 0, y: 0, w: 80, h: 50})

	if op.Kind != OpFillRect {
		t.Fatalf("clipped fill kind = %v, want OpFillRect", op.Kind)
	}

	if !near(op.H, 50) || !near(op.W, 80) || op.Y != 0 {
		t.Fatalf("clipped fill = %+v, want 80x50 at y=0", op)
	}

	outside := Op{Kind: OpFillRect, X: 0, Y: 80, W: 10, H: 10} //nolint:exhaustruct // fully outside
	clipPaintOp(&outside, clipRect{x: 0, y: 0, w: 50, h: 50})
	if outside.Kind != opKindNoop {
		t.Fatalf("outside fill kind = %v, want noop", outside.Kind)
	}
}

func overflowChildFills(t *testing.T, overflow string) (*box, []Op) {
	t.Helper()

	cssSheet := sheet(t, `
body { margin: 0 }
.parent { overflow: `+overflow+`; width: 80pt; height: 50pt; max-height: 50pt }
.child { height: 200pt; width: 80pt; background: #cc0000 }
`)
	res := layoutHTML(t, `<html><body>
<div class="parent"><div class="child">x</div></div>
</body></html>`, cssSheet)
	parent := findBoxByClass(t, res, "parent")
	child := findBoxByClass(t, res, "child")

	var fills []Op

	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.R < 0.7 || op.G > 0.2 {
			continue
		}

		fills = append(fills, op)
	}

	if child.height < 199 {
		t.Fatalf("child height = %.3f, want ~200", child.height)
	}

	return parent, fills
}

func paddingBoxOfTest(boxNode *box) clipRect {
	if boxNode == nil {
		return clipRect{}
	}

	if boxNode.style == nil {
		return clipRect{x: boxNode.x, y: boxNode.y, w: boxNode.w, h: boxNode.height}
	}

	left := boxNode.style.BorderLeft.Width
	top := boxNode.style.BorderTop.Width
	right := boxNode.style.BorderRight.Width
	bottom := boxNode.style.BorderBottom.Width

	return clipRect{
		x: boxNode.x + left,
		y: boxNode.y + top,
		w: boxNode.w - left - right,
		h: boxNode.height - top - bottom,
	}
}

func fillOutsideClip(op Op, clip clipRect) bool {
	if op.W <= 0 || op.H <= 0 {
		return false
	}

	return op.X < clip.x-0.01 ||
		op.Y < clip.y-0.01 ||
		op.X+op.W > clip.x+clip.w+0.01 ||
		op.Y+op.H > clip.y+clip.h+0.01
}
