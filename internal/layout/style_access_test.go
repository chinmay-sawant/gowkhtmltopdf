package layout

import "testing"

var resolvedStyleAccessSink float64 //nolint:gochecknoglobals // benchmark sink

// BenchmarkResolvedStyleAccess measures the cost of copying the full resolved
// style record at a stage boundary. The result controls whether a narrower
// stage view is worth introducing.
func BenchmarkResolvedStyleAccess(b *testing.B) {
	style := initialStyle()

	b.ReportAllocs()

	b.Run("pointer", func(b *testing.B) {
		for range b.N {
			resolvedStyleAccessSink += style.Width + style.FontSize
		}
	})

	b.Run("value-copy", func(b *testing.B) {
		for range b.N {
			snapshot := style
			resolvedStyleAccessSink += snapshot.Width + snapshot.FontSize
		}
	})
}

// BenchmarkBoxModelStyleProjection checks that the narrow sizing view stays a
// value projection with no heap cost at the boundary.
func BenchmarkBoxModelStyleProjection(b *testing.B) {
	style := initialStyle()

	b.ReportAllocs()

	for range b.N {
		view := boxModelStyleOf(&style)
		resolvedStyleAccessSink += view.width + view.height + view.paddingLeft
	}
}
