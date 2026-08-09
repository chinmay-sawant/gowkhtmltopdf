package convert_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"gowkhtmltopdf/internal/imageout"
)

// BenchmarkImageAssets measures image-mode conversion with an inline image
// source. It is separate from BenchmarkWebFetchImage so network fetching and
// rasterization/encoding costs can be compared directly.
func BenchmarkImageAssets(b *testing.B) {
	tpl := loadBenchmarkTemplate(b, "image-grid.html.tmpl")
	imageURL := benchmarkDataURL(benchmarkPNG())
	sources := make(map[int][]byte, len(benchmarkPageSizes))

	for _, images := range benchmarkPageSizes {
		sources[images] = executeBenchmarkTemplate(b, tpl, benchmarkTemplateData{ //nolint:exhaustruct,lll // intentional zero-value fields
			Images: benchmarkImages(images, imageURL),
		})
	}

	for _, images := range benchmarkPageSizes {
		b.Run(fmt.Sprintf("%dImages", images), func(b *testing.B) {
			var output bytes.Buffer
			req := benchmarkImageRequest(sources[images], &output)
			b.ReportMetric(float64(images), "images")
			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := imageout.RunRequest(b.Context(), req, io.Discard); err != nil {
					b.Fatalf("run image benchmark: %v", err)
				}
			}

			b.StopTimer()
			b.SetBytes(int64(output.Len()))
		})
	}
}
