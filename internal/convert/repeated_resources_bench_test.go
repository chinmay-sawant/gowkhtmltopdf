package convert_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/settings"
)

const repeatedResourceReferencesPerPage = 8

type repeatedResourceServer struct {
	server  *httptest.Server
	fetches atomic.Int64
	bytes   atomic.Int64
	image   []byte
}

//nolint:wsl // handler setup intentionally groups route registration and server startup.
func newRepeatedResourceServer(tb testing.TB) *repeatedResourceServer {
	tb.Helper()

	resources := &repeatedResourceServer{ //nolint:exhaustruct // server and counters are initialized below
		image: benchmarkPNG(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/shared.png", func(writer http.ResponseWriter, _ *http.Request) {
		resources.fetches.Add(1)
		resources.bytes.Add(int64(len(resources.image)))
		writer.Header().Set("Content-Type", "image/png")
		_, _ = writer.Write(resources.image)
	})
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/shared.png" {
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(writer, "<html><body><h1>%s</h1>", request.URL.Path)
		for range repeatedResourceReferencesPerPage {
			_, _ = io.WriteString(writer, `<img src="/shared.png" width="48" height="48">`)
		}
		_, _ = io.WriteString(writer, "</body></html>")
	})
	resources.server = httptest.NewServer(mux)
	tb.Cleanup(resources.server.Close)

	return resources
}

//nolint:wsl // request construction keeps object defaults and per-page fields together.
func repeatedResourceRequest(serverURL string, pages int, output io.Writer) *convert.Request {
	global := settings.DefaultPdfGlobal()
	global.Quiet = true
	objects := make([]settings.PdfObject, pages)
	for index := range objects {
		objects[index] = settings.DefaultPdfObject()
		objects[index].Page = fmt.Sprintf("%s/document-%d.html", serverURL, index)
	}

	return convert.NewPDFRequest(global, objects, output, nil)
}

//nolint:wsl // benchmark timing boundaries intentionally surround the loop.
func BenchmarkRepeatedResourcePDF(b *testing.B) {
	for _, pages := range []int{1, 10, 500} {
		b.Run(fmt.Sprintf("%dPages", pages), func(b *testing.B) {
			resources := newRepeatedResourceServer(b)
			var output bytes.Buffer
			req := repeatedResourceRequest(resources.server.URL, pages, &output)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				resources.fetches.Store(0)
				resources.bytes.Store(0)
				output.Reset()
				if err := convert.Run(b.Context(), req, io.Discard, nil); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			peakRSS := processPeakRSSBytes()
			b.ReportMetric(float64(resources.fetches.Load()), "fetches/op")
			b.ReportMetric(float64(resources.bytes.Load()), "fetched_bytes/op")
			b.ReportMetric(float64(output.Len()), "pdf_bytes/op")
			b.SetBytes(int64(output.Len()))
			if peakRSS > 0 {
				b.ReportMetric(float64(peakRSS), "peak_rss_bytes")
			}
		})
	}
}

// processPeakRSSBytes reads Linux's process high-water RSS without conflating
// resident memory with Go's cumulative B/op benchmark metric. Other platforms
// leave the metric absent because they do not expose /proc/self/status.
func processPeakRSSBytes() int64 {
	if runtime.GOOS != "linux" {
		return 0
	}

	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "VmHWM:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}

		kib, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}

		return kib * 1024
	}

	return 0
}

//nolint:wsl // evidence assertions intentionally follow the conversion call.
func TestRepeatedResourceCacheEvidence(t *testing.T) {
	t.Parallel()
	for _, pages := range []int{1, 10, 500} {
		t.Run(fmt.Sprintf("%dPages", pages), func(t *testing.T) {
			t.Parallel()

			resources := newRepeatedResourceServer(t)
			var output bytes.Buffer
			req := repeatedResourceRequest(resources.server.URL, pages, &output)
			if err := convert.Run(t.Context(), req, io.Discard, nil); err != nil {
				t.Fatalf("convert.Run: %v", err)
			}

			wantFetches := int64(pages)
			if got := resources.fetches.Load(); got != wantFetches {
				t.Fatalf("image fetches = %d, want %d (one per document, not one per reference)", got, wantFetches)
			}
			wantReferences := int64(pages * repeatedResourceReferencesPerPage)
			if got := wantReferences / resources.fetches.Load(); got != repeatedResourceReferencesPerPage {
				t.Fatalf("references/fetch = %d, want %d", got, repeatedResourceReferencesPerPage)
			}
			if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
				t.Fatal("repeated-resource output is not a PDF")
			}
		})
	}
}
