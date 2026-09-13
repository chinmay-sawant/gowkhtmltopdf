package gowkhtmltopdf_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	benchmarkHTMLDirEnv   = "GOWKHTMLTOPDF_BENCH_HTML_DIR"
	benchmarkHTMLSizesEnv = "GOWKHTMLTOPDF_BENCH_HTML_SIZES"
)

var errBenchmarkHTMLSize = errors.New("invalid benchmark HTML page size")

// TestWriteBenchmarkHTML materializes the rendered report fixture the public
// benchmark and the process-level CLI comparisons share, so shell benchmark
// drivers can point bin/gowkhtmltopdf at byte-identical HTML without a second
// template renderer. It is opt-in and writes only inside the directory named
// by GOWKHTMLTOPDF_BENCH_HTML_DIR; sizes default to the public benchmark
// matrix when GOWKHTMLTOPDF_BENCH_HTML_SIZES is empty:
//
//	GOWKHTMLTOPDF_BENCH_HTML_DIR=/tmp/bench-html \
//	  GOWKHTMLTOPDF_BENCH_HTML_SIZES=2,100,500 \
//	  go test . -run '^TestWriteBenchmarkHTML$' -count=1
func TestWriteBenchmarkHTML(t *testing.T) {
	t.Parallel()

	dir := os.Getenv(benchmarkHTMLDirEnv)
	if dir == "" {
		t.Skipf("set %s to write rendered benchmark HTML", benchmarkHTMLDirEnv)
	}

	sizes, err := parseBenchmarkHTMLSizes(os.Getenv(benchmarkHTMLSizesEnv))
	if err != nil {
		t.Fatalf("parse %s: %v", benchmarkHTMLSizesEnv, err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create benchmark HTML directory: %v", err)
	}

	for _, pages := range sizes {
		path := filepath.Join(dir, fmt.Sprintf("doc_%d.html", pages))
		if err := os.WriteFile(path, libraryBenchmarkReportHTML(pages), 0o600); err != nil {
			t.Fatalf("write benchmark HTML %s: %v", path, err)
		}
	}
}

func parseBenchmarkHTMLSizes(value string) ([]int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return slices.Clone(libraryBenchmarkPageSizes), nil
	}

	fields := strings.Split(value, ",")
	sizes := make([]int, 0, len(fields))

	for _, field := range fields {
		pages, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil || pages <= 0 {
			return nil, fmt.Errorf("%w: %q", errBenchmarkHTMLSize, field)
		}

		sizes = append(sizes, pages)
	}

	return sizes, nil
}
