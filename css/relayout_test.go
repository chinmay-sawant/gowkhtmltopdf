package css_test

import (
	"errors"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pubstate"
)

func TestRelayoutRejectsBadInput(t *testing.T) {
	t.Parallel()

	styled, err := css.Apply(t.Context(), mustDoc(t, `<p>Hi</p>`),
		css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	//nolint:staticcheck // nil context is intentional
	if _, err := css.Relayout(nil, styled, 300, 200, "", "", ""); !errors.Is(err, css.ErrNilContext) {
		t.Fatalf("nil context: %v", err)
	}

	if _, err := css.Relayout(t.Context(), nil, 300, 200, "", "", ""); !errors.Is(err, css.ErrNilDocument) {
		t.Fatalf("nil document: %v", err)
	}

	if _, err := css.Relayout(t.Context(), styled, 0, 200, "", "", ""); !errors.Is(err, css.ErrBadSize) {
		t.Fatalf("size: %v", err)
	}
}

func TestRelayoutStoresViewportAndState(t *testing.T) {
	t.Parallel()

	styled, err := css.Apply(t.Context(), mustDoc(t, `<p>Hi</p>`),
		css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	next, err := css.Relayout(t.Context(), styled, 320, 240, "f", "h", "a")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := pubstate.StyledOf(next)
	if !ok {
		t.Fatal("relayout result is not a styled document")
	}

	if got.WidthPx != 320 || got.HeightPx != 240 {
		t.Fatalf("viewport %dx%d, want 320x240", got.WidthPx, got.HeightPx)
	}

	if got.State.Focus != "f" || got.State.Hover != "h" || got.State.Active != "a" {
		t.Fatalf("state %+v, want focus f, hover h, active a", got.State)
	}

	if got.Root == nil {
		t.Fatal("relayout result has no tree")
	}
}
