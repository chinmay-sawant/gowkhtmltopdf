package imageout //nolint:testpackage // benchmark exercises the unexported overlay helper.

import (
	"fmt"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// borderHeavyOverlayOps builds a display list shaped like a border-heavy report
// page: rounded cards contain fill/text operations, a four-sided mixed border,
// and a horizontal separator. The top side is a matching overlay; vertical
// sides return early, while bottom and separator lines are non-matching
// horizontal candidates that made the old prefix scan expensive.
func borderHeavyOverlayOps(cardCount int) []layout.Op {
	const (
		cardsPerRow = 8
		cardXGap    = 168.0
		cardYGap    = 72.0
		cardW       = 156.0
		cardH       = 54.0
		radius      = 8.0
	)

	ops := make([]layout.Op, 0, cardCount*9)

	for card := range cardCount {
		cardX := float64(card%cardsPerRow)*cardXGap + 12
		cardY := float64(card/cardsPerRow)*cardYGap + 12

		ops = append(ops,
			layout.Op{ //nolint:exhaustruct // synthetic card background
				Kind: layout.OpFillRect, X: cardX, Y: cardY, W: cardW, H: cardH,
			},
			layout.Op{ //nolint:exhaustruct // rounded base border
				Kind: layout.OpStrokeRect, X: cardX, Y: cardY, W: cardW, H: cardH,
				R: 0.35, G: 0.4, B: 0.45, Width: 1.5, Radius: radius,
			},
			layout.Op{ //nolint:exhaustruct // matching rounded top overlay
				Kind: layout.OpLine, X: cardX + radius, Y: cardY, W: cardW - 2*radius,
				Width: 4, R: 0.1, G: 0.65, B: 0.4,
			},
			layout.Op{ //nolint:exhaustruct // right side, rejected by orientation
				Kind: layout.OpLine, X: cardX + cardW, Y: cardY + radius, H: cardH - 2*radius,
				Width: 2, R: 0.35, G: 0.4, B: 0.45,
			},
			layout.Op{ //nolint:exhaustruct // bottom side, scans backward without a match
				Kind: layout.OpLine, X: cardX + radius, Y: cardY + cardH, W: cardW - 2*radius,
				Width: 2, R: 0.35, G: 0.4, B: 0.45,
			},
			layout.Op{ //nolint:exhaustruct // left side, rejected by orientation
				Kind: layout.OpLine, X: cardX, Y: cardY + radius, H: cardH - 2*radius,
				Width: 2, R: 0.35, G: 0.4, B: 0.45,
			},
			layout.Op{ //nolint:exhaustruct // horizontal card separator
				Kind: layout.OpLine, X: cardX + 12, Y: cardY + 34, W: cardW - 24,
				Width: 1, R: 0.75, G: 0.78, B: 0.8,
			},
			layout.Op{ //nolint:exhaustruct // text placeholder from a report card
				Kind: layout.OpText, X: cardX + 12, Y: cardY + 22, W: cardW - 24, H: 12,
			},
		)
	}

	return ops
}

// BenchmarkRoundedBorderLineOverlay measures the display-list lookup performed
// by imageout's raster loop for border-heavy reports. It intentionally excludes
// layout and pixel painting: each subbenchmark reuses realistic Op data and
// times only the helper call for every display-list operation. The card sizes
// show lookup cost as the number of preceding border operations grows.
func BenchmarkRoundedBorderLineOverlay(b *testing.B) {
	for _, cards := range []int{32, 128, 512} {
		b.Run(fmt.Sprintf("%dCards", cards), func(b *testing.B) {
			ops := borderHeavyOverlayOps(cards)

			b.ReportAllocs()

			b.ResetTimer()

			matches := 0

			for range b.N {
				for index := range ops {
					if _, ok := roundedBorderLineOverlay(ops, index); ok {
						matches++
					}
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(len(ops)), "display_ops")

			if matches == 0 {
				b.Fatal("benchmark fixture produced no rounded-border overlays")
			}

			b.ReportMetric(float64(matches)/float64(b.N), "matches/iteration")
		})
	}
}
