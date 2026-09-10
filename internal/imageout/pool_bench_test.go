package imageout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

func BenchmarkSupersamplePool(b *testing.B) {
	for _, size := range []struct {
		name   string
		width  float64
		height float64
	}{
		{name: "small", width: 96, height: 96},
		{name: "large", width: 512, height: 512},
	} {
		b.Run(size.name, func(b *testing.B) {
			result := &layout.Result{
				Width:  size.width,
				Height: size.height,
			}

			b.ReportAllocs()

			b.ResetTimer()

			for range b.N {
				if _, err := rasterizeContext(b.Context(), result, size.height, false, 0, 0); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
