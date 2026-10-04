package css_test

import (
	"encoding/base64"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
)

// TestRelayoutRegatesLinkedSheets pins the collection-time viewport gate: a
// linked sheet under a min-width media query appears only at the wider
// viewport, disappears again when the viewport shrinks, and comes back from
// the cache without a second parse.
func TestRelayoutRegatesLinkedSheets(t *testing.T) {
	t.Parallel()

	body := base64.StdEncoding.EncodeToString([]byte("#bar { background-color: #123456 }"))
	source := `<link rel="stylesheet" media="(min-width:400px)" href="data:text/css;base64,` +
		body + `"><div id="bar">x</div>`

	styled, err := css.Apply(t.Context(), mustDoc(t, source),
		css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	if got := sheetCount(t, styled); got != 0 {
		t.Fatalf("sheets at 200px = %d, want 0 under the media gate", got)
	}

	wide, err := css.Relayout(t.Context(), styled, 800, 600, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if got := sheetCount(t, wide); got != 1 {
		t.Fatalf("sheets at 800px = %d, want 1", got)
	}

	narrow, err := css.Relayout(t.Context(), wide, 200, 100, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if got := sheetCount(t, narrow); got != 0 {
		t.Fatalf("sheets back at 200px = %d, want 0", got)
	}

	again, err := css.Relayout(t.Context(), narrow, 800, 600, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if got := sheetCount(t, again); got != 1 {
		t.Fatalf("sheets at 800px again = %d, want 1", got)
	}

	if first, second := firstSheet(t, wide), firstSheet(t, again); first != second {
		t.Fatal("the re-gated link was fetched and parsed again; pointer changed")
	}
}
