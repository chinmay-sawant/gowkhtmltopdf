package layout_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// relayoutBenchPage builds a 140-row list with a stylesheet to collect.
func relayoutBenchPage() string {
	var page strings.Builder

	page.WriteString(`<style>
		body { margin: 0; font-size: 12pt }
		.grid { padding: 8px }
		.row { border-bottom: 1px solid #ddd; padding: 4px 2px }
		.row:hover { background-color: #f0f0f0 }
		.name { color: #223344; width: 60vw }
		.amount { color: #445566; text-align: right }
		@media (min-width: 500px) { .name { width: 40vw } }
	</style><div class="grid">`)

	for i := range 140 {
		fmt.Fprintf(&page,
			`<div class="row"><span class="name">Item %d</span><span class="amount">$%d.00</span></div>`,
			i, i)
	}

	page.WriteString(`</div>`)

	return page.String()
}

// BenchmarkRelayoutMedium compares full Apply plus DisplayList with Relayout
// plus DisplayList at alternating widths.
func BenchmarkRelayoutMedium(b *testing.B) {
	tree, err := html.Parse([]byte(relayoutBenchPage()))
	if err != nil {
		b.Fatal(err)
	}

	base, err := css.Apply(b.Context(), tree, css.Options{WidthPx: 320, HeightPx: 640, Media: "screen"})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("full-apply", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			styled, err := css.Apply(b.Context(), tree, css.Options{
				WidthPx: 320 + i%2, HeightPx: 640, Media: "screen",
			})
			if err != nil {
				b.Fatal(err)
			}

			if _, err := layout.DisplayList(b.Context(), styled); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("relayout", func(b *testing.B) {
		doc := base

		for i := 0; b.Loop(); i++ {
			next, err := css.Relayout(b.Context(), doc, 320+i%2, 640, "", "", "")
			if err != nil {
				b.Fatal(err)
			}

			if _, err := layout.DisplayList(b.Context(), next); err != nil {
				b.Fatal(err)
			}

			doc = next
		}
	})
}
