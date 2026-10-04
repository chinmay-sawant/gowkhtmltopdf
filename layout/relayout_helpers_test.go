package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// applyAt parses source and applies it at one viewport and state.
func applyAt(t *testing.T, source string, width, height int, focus, hover, active string) *css.Document {
	t.Helper()

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return mustApply(t, tree, width, height, focus, hover, active)
}

// mustApply applies one already parsed tree at a viewport and state.
func mustApply(t *testing.T, tree *html.Document, width, height int, focus, hover, active string) *css.Document {
	t.Helper()

	styled, err := css.Apply(t.Context(), tree, css.Options{
		WidthPx: width, HeightPx: height, Media: "screen",
		Focus: focus, Hover: hover, Active: active,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	return styled
}

// layOf lays out one styled document.
func layOf(t *testing.T, styled *css.Document) *layout.Display {
	t.Helper()

	display, err := layout.DisplayList(t.Context(), styled)
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	return display
}

// displayAt applies source at one viewport and lays it out.
func displayAt(t *testing.T, source string, width, height int, focus, hover, active string) *layout.Display {
	t.Helper()

	return layOf(t, applyAt(t, source, width, height, focus, hover, active))
}
