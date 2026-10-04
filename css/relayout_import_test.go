package css_test

import (
	"encoding/base64"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
)

// TestRelayoutRegatesImportedSheets is the @import half of the collection
// gate: the import media prelude is re-evaluated at the new viewport, and the
// imported sheet comes back from the cache without a second fetch.
func TestRelayoutRegatesImportedSheets(t *testing.T) {
	t.Parallel()

	body := base64.StdEncoding.EncodeToString([]byte("#bar { color: #123456 }"))
	source := `<style>@import url("data:text/css;base64,` + body +
		`") (min-width:400px);</style><div id="bar">x</div>`

	styled, err := css.Apply(t.Context(), mustDoc(t, source),
		css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	if got := sheetCount(t, styled); got != 1 {
		t.Fatalf("sheets at 200px = %d, want 1 (import gated out)", got)
	}

	wide, err := css.Relayout(t.Context(), styled, 800, 600, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if got := sheetCount(t, wide); got != 2 {
		t.Fatalf("sheets at 800px = %d, want 2 (import before importer)", got)
	}

	narrow, err := css.Relayout(t.Context(), wide, 200, 100, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if got := sheetCount(t, narrow); got != 1 {
		t.Fatalf("sheets back at 200px = %d, want 1", got)
	}

	again, err := css.Relayout(t.Context(), narrow, 800, 600, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if first, second := firstSheet(t, wide), firstSheet(t, again); first != second {
		t.Fatal("the re-gated import was fetched and parsed again; pointer changed")
	}
}
