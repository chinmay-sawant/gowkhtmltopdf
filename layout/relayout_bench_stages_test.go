package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/html"
)

// BenchmarkRelayoutStages measures collection alone, with no layout pass, so
// the style-parse saving shows without the shared cascade cost on top.
func BenchmarkRelayoutStages(b *testing.B) {
	tree, err := html.Parse([]byte(relayoutBenchPage()))
	if err != nil {
		b.Fatal(err)
	}

	base, err := css.Apply(b.Context(), tree, css.Options{WidthPx: 320, HeightPx: 640, Media: "screen"})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("apply-only", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			if _, err := css.Apply(b.Context(), tree, css.Options{
				WidthPx: 320 + i%2, HeightPx: 640, Media: "screen",
			}); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("relayout-only", func(b *testing.B) {
		doc := base

		for i := 0; b.Loop(); i++ {
			next, err := css.Relayout(b.Context(), doc, 320+i%2, 640, "", "", "")
			if err != nil {
				b.Fatal(err)
			}

			doc = next
		}
	})
}
