package layout_test

import (
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// boxByID returns the border box for one element id.
//
//nolint:unparam // one lookup helper shared by the relayout fixtures
func boxByID(t *testing.T, display *layout.Display, elementID string) layout.Box {
	t.Helper()

	for _, box := range display.Boxes {
		if box.ID == elementID {
			return box
		}
	}

	t.Fatalf("no box for #%s", elementID)

	return layout.Box{}
}

// nearBox compares one dimension with a 0.01 CSS pixel tolerance. Layout
// stores points and Display converts back to pixels, so a vw/vh value carries
// a float round-trip of about 1e-13 that no exact comparison should trip on.
func nearBox(t *testing.T, label string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.01 {
		t.Fatalf("%s = %.4f, want %.4f", label, got, want)
	}
}

// hasFill reports whether a fill op carries the exact RGB. Op channels are
// 0..1 floats, so the uint8 arguments are divided once by 255.
func hasFill(display *layout.Display, r, g, b uint8) bool {
	wantR, wantG, wantB := float64(r)/255, float64(g)/255, float64(b)/255

	for _, op := range display.Ops {
		if op.Kind == layout.DisplayOpFillRect &&
			op.R == wantR && op.G == wantG && op.B == wantB {
			return true
		}
	}

	return false
}

// textColor returns the RGB of the first text op containing needle.
func textColor(display *layout.Display, needle string) (uint8, uint8, uint8, bool) {
	for _, op := range display.Ops {
		if op.Kind == layout.DisplayOpText && strings.Contains(op.Text, needle) {
			return channel(op.R), channel(op.G), channel(op.B), true
		}
	}

	return 0, 0, 0, false
}

func channel(v float64) uint8 {
	return uint8(math.Round(v * 255))
}
