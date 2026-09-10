package profiling

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// The benchmarks below time golden-template generation only: every fixture is
// converted through the same public Document/ImageDocument entry points the
// harness profiles. They never run in the normal suite (Go only runs them
// with -bench).
//
// Whole-corpus time per feature:
//
//	go test ./internal/profiling -run '^$' \
//	  -bench '^BenchmarkGolden(PDF|ImagePNG|ImageJPEG)$' -benchmem -benchtime=1x
//
// Per-template time:
//
//	go test ./internal/profiling -run '^$' \
//	  -bench 'Golden.*ByFixture$' -benchmem -benchtime=100ms

// BenchmarkGoldenPDF converts all golden templates to PDF in one iteration.
func BenchmarkGoldenPDF(b *testing.B) { benchmarkCorpus(b, modePDF) }

// BenchmarkGoldenImagePNG converts all golden templates to PNG in one iteration.
func BenchmarkGoldenImagePNG(b *testing.B) { benchmarkCorpus(b, modePNG) }

// BenchmarkGoldenImageJPEG converts all golden templates to JPEG in one iteration.
func BenchmarkGoldenImageJPEG(b *testing.B) { benchmarkCorpus(b, modeJPEG) }

// BenchmarkGoldenPDFByFixture converts each golden template to PDF separately.
func BenchmarkGoldenPDFByFixture(b *testing.B) { benchmarkFixtures(b, modePDF) }

// BenchmarkGoldenImagePNGByFixture converts each golden template to PNG separately.
func BenchmarkGoldenImagePNGByFixture(b *testing.B) { benchmarkFixtures(b, modePNG) }

// BenchmarkGoldenImageJPEGByFixture converts each golden template to JPEG separately.
func BenchmarkGoldenImageJPEGByFixture(b *testing.B) { benchmarkFixtures(b, modeJPEG) }

func benchmarkCorpus(b *testing.B, mode string) {
	b.Helper()

	fixtureDir := resolveFixtureDir(b)
	fixtures := discoverFixtures(b, fixtureDir)
	ctx := context.Background()
	supported := make([]string, 0, len(fixtures))

	for _, path := range fixtures {
		if err := runFixture(ctx, mode, path, fixtureDir); err != nil {
			b.Logf("skip %s: %v", filepath.Base(path), err)

			continue
		}

		supported = append(supported, path)
	}

	b.Logf("benchmarking %d of %d fixtures", len(supported), len(fixtures))
	b.ReportAllocs()

	for b.Loop() {
		for _, path := range supported {
			if err := runFixture(ctx, mode, path, fixtureDir); err != nil {
				b.Fatalf("%s: %v", filepath.Base(path), err)
			}
		}
	}
}

func benchmarkFixtures(b *testing.B, mode string) {
	b.Helper()

	fixtureDir := resolveFixtureDir(b)
	fixtures := discoverFixtures(b, fixtureDir)
	ctx := context.Background()

	for _, path := range fixtures {
		name := strings.TrimSuffix(filepath.Base(path), ".html")

		if err := runFixture(ctx, mode, path, fixtureDir); err != nil {
			runErr := err
			b.Run(name, func(b *testing.B) {
				b.Skipf("unsupported: %v", runErr)
			})

			continue
		}

		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				if err := runFixture(ctx, mode, path, fixtureDir); err != nil {
					b.Fatalf("%s: %v", name, err)
				}
			}
		})
	}
}
