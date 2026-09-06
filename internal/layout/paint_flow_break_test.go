//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"testing"
)

func TestForcedBreakTargetYLandsAtNextPageTop(t *testing.T) {
	t.Parallel()

	const contentH = 785.19

	t.Run("same page including bottom sliver", func(t *testing.T) {
		t.Parallel()

		target, skip := forcedBreakTargetY(contentH-22, contentH-40, contentH)
		if skip {
			t.Fatal("bottom sliver must not count as a fresh page")
		}

		if math.Abs(target-contentH) > 0.01 {
			t.Fatalf("target = %.4f, want %.4f", target, contentH)
		}
	})

	t.Run("float remainder below a page multiple", func(t *testing.T) {
		t.Parallel()

		boxY := 4*contentH - 4.5e-13

		target, skip := forcedBreakTargetY(boxY, contentH, contentH)
		if !skip {
			t.Fatalf("epsilon below page top must already be fresh, target=%.12f", target)
		}
	})

	t.Run("later page landing band snaps up", func(t *testing.T) {
		t.Parallel()

		target, skip := forcedBreakTargetY(2*contentH+9, contentH-10, contentH)
		if skip {
			t.Fatal("landing band should snap to that page top")
		}

		if math.Abs(target-2*contentH) > 0.01 {
			t.Fatalf("target = %.4f, want %.4f", target, 2*contentH)
		}
	})

	t.Run("preceding ink already on this page pushes next", func(t *testing.T) {
		t.Parallel()

		target, skip := forcedBreakTargetY(2*contentH-22, 2*contentH-40, contentH)
		if skip {
			t.Fatal("sliver after ink on the same page must move")
		}

		if math.Abs(target-2*contentH) > 0.01 {
			t.Fatalf("target = %.4f, want %.4f", target, 2*contentH)
		}
	})
}

func TestShiftSamePageFromYLeavesLaterPages(t *testing.T) {
	t.Parallel()

	const contentH = 100.0

	res := &Result{ //nolint:exhaustruct // index-only fixture
		Ops: []Op{
			{Y: 12},  //nolint:exhaustruct // index-only fixture
			{Y: 110}, //nolint:exhaustruct // index-only fixture
		},
		root: &box{ //nolint:exhaustruct // index-only fixture
			y: 10,
			children: []*box{
				{y: 12},
				{y: 110},
			},
		},
	}

	shiftSamePageFromY(res, 10, 0, contentH, -10)

	if got := res.Ops[0].Y; math.Abs(got-2) > 0.01 {
		t.Fatalf("same-page op Y = %.2f, want 2", got)
	}

	if got := res.Ops[1].Y; math.Abs(got-110) > 0.01 {
		t.Fatalf("later-page op Y = %.2f, want 110", got)
	}

	if got := res.root.children[1].y; math.Abs(got-110) > 0.01 {
		t.Fatalf("later-page box Y = %.2f, want 110", got)
	}
}
