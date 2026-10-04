package css_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pubstate"
)

func TestRelayoutReusesParsedSheets(t *testing.T) {
	t.Parallel()

	styled, err := css.Apply(t.Context(), mustDoc(t, `<style>p { color: #123456 }</style><p>Hi</p>`),
		css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	next, err := css.Relayout(t.Context(), styled, 320, 240, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	before, _ := pubstate.StyledOf(styled)
	after, _ := pubstate.StyledOf(next)

	if len(before.Sheets) != 1 || len(after.Sheets) != 1 {
		t.Fatalf("sheets before=%d after=%d, want 1 each", len(before.Sheets), len(after.Sheets))
	}

	if before.Sheets[0] != after.Sheets[0] {
		t.Fatal("relayout reparsed the inline sheet; pointer changed")
	}
}
