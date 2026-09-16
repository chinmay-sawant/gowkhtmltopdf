package layout

import (
	"testing"
)

// Browsers ship the UA rules q::before{content:open-quote} and
// q::after{content:close-quote}. The engine painted no quotes for <q>
// because selectContentDecl only saw author rules (ana-de-armas finding 3:
// six missing double quotes in citations).
func TestUAQuoteContentForQ(t *testing.T) {
	t.Parallel()

	t.Run("implicitDefaults", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html><body><p>x <q>hello</q> y</p></body></html>`,
			sheet(t, `body { margin: 0; font-size: 12pt }`))

		want := "x " + defaultQuoteOpen + "hello" + defaultQuoteClose + " y"
		if got := joinedPaintText(res); got != want {
			t.Fatalf("plain <q> = %q, want %q", got, want)
		}
	})

	t.Run("authorQuotes", func(t *testing.T) {
		t.Parallel()

		// The real Wikipedia citation rule: straight double quotes at depth 1,
		// straight single quotes at depth 2.
		res := layoutHTML(t,
			`<html><body><p class="mw-parser-output"><cite class="citation">x <q>hello</q> y</cite></p></body></html>`,
			sheet(t, `body { margin: 0; font-size: 12pt }
.mw-parser-output .citation q { quotes: "\"" "\"" "'" "'" }`))

		if got := joinedPaintText(res); got != `x "hello" y` {
			t.Fatalf("citation q = %q, want %q", got, `x "hello" y`)
		}
	})

	t.Run("nestedDepth", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html><body><q>a <q>b</q> c</q></body></html>`,
			sheet(t, `body { margin: 0; font-size: 12pt }`))

		want := defaultQuoteOpen + "a \u2018b\u2019 c" + defaultQuoteClose
		if got := joinedPaintText(res); got != want {
			t.Fatalf("nested default <q> = %q, want %q", got, want)
		}
	})

	t.Run("authorContentWins", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html><body><q>hello</q></body></html>`,
			sheet(t, `body { margin: 0; font-size: 12pt }
q::before { content: "[" }
q::after { content: "]" }`))

		if got := joinedPaintText(res); got != "[hello]" {
			t.Fatalf("author content = %q, want [hello]", got)
		}
	})
}
