package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/html"
)

// TestLayoutResolvesStylesPerDocument proves the tree carries no memoized
// styles: two Apply calls over one parsed tree at different viewports produce
// different placements, and Relayout reproduces the fresh-Apply placement at
// the same viewport. If layout stored resolved styles on the tree, the second
// call would serve the first viewport's styles.
func TestLayoutResolvesStylesPerDocument(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(mediaVWSource))
	if err != nil {
		t.Fatal(err)
	}

	narrow := mustApply(t, tree, 320, 240, "", "", "")
	wide := mustApply(t, tree, 640, 480, "", "", "")

	narrowBar := boxByID(t, layOf(t, narrow), "bar")
	wideDisplay := layOf(t, wide)
	wideBar := boxByID(t, wideDisplay, "bar")

	if narrowBar.W == wideBar.W {
		t.Fatalf("one tree at two viewports laid out at the same width %.2f; styles were memoized", narrowBar.W)
	}

	compareDisplays(t, layOf(t, relayout(t, narrow, 640, 480, "", "", "")), wideDisplay)
}
