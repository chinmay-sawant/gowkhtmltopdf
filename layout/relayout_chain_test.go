package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// relayout restyles styled for a new viewport and state.
func relayout(t *testing.T, styled *css.Document, width, height int, focus, hover, active string) *css.Document {
	t.Helper()

	next, err := css.Relayout(t.Context(), styled, width, height, focus, hover, active)
	if err != nil {
		t.Fatalf("relayout: %v", err)
	}

	return next
}

// relayoutDisplay relayouts styled and lays the result out.
func relayoutDisplay(t *testing.T, styled *css.Document, width, height int, focus, hover, active string) *layout.Display {
	t.Helper()

	return layOf(t, relayout(t, styled, width, height, focus, hover, active))
}
