package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// compareDisplays pins that Relayout is a placement-for-placement replacement
// for a full Apply at the same viewport and state. Both paths run the same
// cascade over the same rule text, so the floats are identical and no
// tolerance is needed. The 0.01 tolerance in nearBox covers hand-written
// vw/vh expectations, not this comparison.
func compareDisplays(t *testing.T, got, want *layout.Display) {
	t.Helper()

	if got.Width != want.Width || got.Height != want.Height {
		t.Fatalf("canvas %dx%d, want %dx%d", got.Width, got.Height, want.Width, want.Height)
	}

	if len(got.Boxes) != len(want.Boxes) {
		t.Fatalf("boxes %d, want %d", len(got.Boxes), len(want.Boxes))
	}

	for i := range want.Boxes {
		if got.Boxes[i] != want.Boxes[i] {
			t.Fatalf("box %d: %+v, want %+v", i, got.Boxes[i], want.Boxes[i])
		}
	}

	if len(got.Ops) != len(want.Ops) {
		t.Fatalf("ops %d, want %d", len(got.Ops), len(want.Ops))
	}

	for i := range want.Ops {
		compareOp(t, i, &got.Ops[i], &want.Ops[i])
	}
}

func compareOp(t *testing.T, index int, got, want *layout.DisplayOp) {
	t.Helper()

	if got.Kind != want.Kind || got.X != want.X || got.Y != want.Y ||
		got.W != want.W || got.H != want.H || got.Text != want.Text {
		t.Fatalf("op %d: kind=%v geom=(%g,%g,%g,%g) text=%q, want kind=%v geom=(%g,%g,%g,%g) text=%q",
			index, got.Kind, got.X, got.Y, got.W, got.H, got.Text,
			want.Kind, want.X, want.Y, want.W, want.H, want.Text)
	}
}
