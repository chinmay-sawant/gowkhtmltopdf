//nolint:testpackage,wsl // generated-quote layout proofs
package layout

import (
	"testing"
)

func TestQuotes(t *testing.T) { //nolint:funlen // default, pair, nested depth, and layout proofs
	t.Parallel()

	t.Run("defaultCurly", func(t *testing.T) {
		t.Parallel()

		got := parseContentValue("open-quote close-quote", nil)
		if got != defaultQuoteOpen+defaultQuoteClose {
			t.Fatalf("default quotes want %q got %q", defaultQuoteOpen+defaultQuoteClose, got)
		}
	})

	t.Run("layoutPair", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
q { quotes: "«" "»"; }
q::before { content: open-quote; }
q::after { content: close-quote; }
`)
		res := layoutHTML(t, `<html><body><q>hello</q></body></html>`, cssSheet)

		got := joinedPaintText(res)
		if got != "«hello»" {
			t.Fatalf("quotes pair: %q", got)
		}
	})

	t.Run("nestedDepth", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
q { quotes: "«" "»" "‹" "›"; }
q::before { content: open-quote; }
q::after { content: close-quote; }
`)
		res := layoutHTML(t, `<html><body><q>outer <q>inner</q></q></body></html>`, cssSheet)

		got := joinedPaintText(res)
		if got != "«outer ‹inner›»" {
			t.Fatalf("nested quotes: %q", got)
		}
	})

	t.Run("defaultLayout", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
q::before { content: open-quote; }
q::after { content: close-quote; }
`)
		res := layoutHTML(t, `<html><body><q>x</q></body></html>`, cssSheet)

		got := joinedPaintText(res)
		want := defaultQuoteOpen + "x" + defaultQuoteClose
		if got != want {
			t.Fatalf("default layout quotes want %q got %q", want, got)
		}
	})
}
