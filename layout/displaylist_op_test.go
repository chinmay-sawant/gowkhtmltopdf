package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// The zero DisplayOp has no embedded opExtra. Reading a promoted field
// directly on it dereferences nil and panics, so every rare payload an
// external replay engine needs must be reachable through a nil-safe accessor.
// These tests are the guard for that: they touch the zero value on purpose,
// because the panic a zero value causes is invisible until a consumer
// constructs one.

// TestZeroOpStringAccessorsAreNilSafe covers the accessors that return a string.
func TestZeroOpStringAccessorsAreNilSafe(t *testing.T) {
	t.Parallel()

	var zero layout.DisplayOp

	if got := zero.LinkURI(); got != "" {
		t.Errorf("LinkURI %q", got)
	}

	if got := zero.ImageAlt(); got != "" {
		t.Errorf("ImageAlt %q", got)
	}

	if got := zero.BlendModeName(); got != "" {
		t.Errorf("BlendModeName %q", got)
	}

	if got := zero.FontFeatures(); got != "" {
		t.Errorf("FontFeatures %q", got)
	}

	if got := zero.TextLanguage(); got != "" {
		t.Errorf("TextLanguage %q", got)
	}

	if got := zero.TextAutospace(); got != "" {
		t.Errorf("TextAutospace %q", got)
	}

	if got := zero.TextTransformValue(); got != "" {
		t.Errorf("TextTransformValue %q", got)
	}
}

// TestZeroOpFlagAccessorsAreNilSafe covers the accessors that return a bool.
func TestZeroOpFlagAccessorsAreNilSafe(t *testing.T) {
	t.Parallel()

	var zero layout.DisplayOp

	if zero.Outline() {
		t.Error("Outline true on zero op")
	}

	if zero.NoFakeBoldValue() {
		t.Error("NoFakeBoldValue true on zero op")
	}
}

// TestZeroOpNumericAccessorsAreNilSafe covers opacity, which must report the
// CSS initial value rather than the engine's 0-means-unset sentinel.
func TestZeroOpNumericAccessorsAreNilSafe(t *testing.T) {
	t.Parallel()

	var zero layout.DisplayOp

	if got := zero.Opacity(); got != 1 {
		t.Errorf("Opacity %v, want CSS initial 1", got)
	}
}

// TestZeroOpObjectAccessorsAreNilSafe covers the accessors that return a
// reference type, where nil is the correct "nothing here" answer.
func TestZeroOpObjectAccessorsAreNilSafe(t *testing.T) {
	t.Parallel()

	var zero layout.DisplayOp

	if got := zero.Group(); got != nil {
		t.Errorf("Group %v", got)
	}

	if got := zero.GroupBoundary(); got != 0 {
		t.Errorf("GroupBoundary %v", got)
	}

	if !zero.Transform().IsIdentity() {
		t.Error("Transform should be identity on zero op")
	}

	data, width, height := zero.ImageBytes()
	if data != nil || width != 0 || height != 0 {
		t.Errorf("ImageBytes %v %d %d", data, width, height)
	}
}
