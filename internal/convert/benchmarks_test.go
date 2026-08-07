package convert_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/imageout"
	"gowkhtmltopdf/internal/settings"
)

var benchmarkPageSizes = []int{2, 5, 10, 20, 50, 100, 200, 250, 500} //nolint:gochecknoglobals // fixed benchmark matrix

const (
	benchmarkRowsPerPage = 20
	liveBenchmarkEnv     = "GOWKHTMLTOPDF_LIVE_BENCHMARK"
	tvMazeShowsURL       = "https://api.tvmaze.com/shows?page=0"
)

// errTVMazeBadStatus reports a non-200 TVmaze API response.
var errTVMazeBadStatus = errors.New("TVmaze API returned non-200 status")

// errTVMazeNoShows reports an empty TVmaze show list.
var errTVMazeNoShows = errors.New("TVmaze API returned no shows")

// Baseline snapshot from benchmark-results.txt, recorded with
// go1.26.4 linux/amd64 on WSL2 (24 CPUs), -benchtime=1x -count=1. The
// report template uses 20 realistic rows per physical page:
// PDF pages:     2=10.31ms, 5=22.87ms, 10=36.91ms, 20=77.74ms,
//                50=226.25ms, 100=528.33ms, 200=1.51s, 250=3.47s,
//                500=14.14s.
// Template+PDF:  2=9.56ms, 5=25.55ms, 10=46.39ms, 20=100.62ms,
//                50=250.62ms, 100=621.44ms, 200=2.02s, 250=3.62s,
//                500=13.67s.
// Web images:    2=257.33ms, 5=258.05ms, 10=281.10ms, 20=310.47ms,
//                50=356.66ms, 100=413.68ms, 200=506.42ms, 250=564.00ms,
//                500=970.72ms.
// Inline images: 2=209.50ms, 5=220.61ms, 10=255.35ms, 20=282.33ms,
//                50=303.54ms, 100=340.31ms, 200=439.46ms, 250=491.22ms,
//                500=788.43ms.

type benchmarkPage struct {
	Number int
	First  bool
	Rows   []benchmarkRow
}

type benchmarkRow struct {
	Number      int
	SKU         string
	Description string
	Quantity    int
	Amount      string
}

type benchmarkImage struct {
	Src   string
	Label string
}

type benchmarkTemplateData struct {
	Pages    []benchmarkPage
	Images   []benchmarkImage
	ImageSrc string
}

type tvMazeImage struct {
	Medium   string `json:"medium"`
	Original string `json:"original"`
}

type tvMazeRating struct {
	Average float64 `json:"average"`
}

type tvMazeShow struct {
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Language  string       `json:"language"`
	Genres    []string     `json:"genres"`
	Status    string       `json:"status"`
	Premiered string       `json:"premiered"`
	Rating    tvMazeRating `json:"rating"`
	Image     *tvMazeImage `json:"image"`
}

type movieListingItem struct {
	Name      string
	Type      string
	Language  string
	Genres    string
	Status    string
	Premiered string
	Rating    string
	ImageURL  string
}

type movieListingData struct {
	Items []movieListingItem
}

func benchmarkTemplatePath(name string) string {
	return filepath.Join(
		"..",
		"..",
		"testdata",
		"golden",
		"benchmarks",
		"templates",
		name,
	)
}

func loadBenchmarkTemplate(tb testing.TB, name string) *template.Template {
	tb.Helper()

	path := benchmarkTemplatePath(name)

	source, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read benchmark template %s: %v", name, err)
	}

	tpl, err := template.New(name).Parse(string(source))
	if err != nil {
		tb.Fatalf("parse benchmark template %s: %v", name, err)
	}

	return tpl
}

func benchmarkPages(pageCount int) []benchmarkPage {
	pages := make([]benchmarkPage, pageCount)
	for page := range pages {
		rows := make([]benchmarkRow, benchmarkRowsPerPage)
		for row := range rows {
			line := row + 1
			rows[row] = benchmarkRow{
				Number:      line,
				SKU:         fmt.Sprintf("SKU-%03d-%03d", page+1, line),
				Description: fmt.Sprintf("Platform operations and support service %d", line),
				Quantity:    (line+page)%7 + 1,
				Amount:      fmt.Sprintf("%d.%02d", (page+1)*line, (page+line)%100),
			}
		}

		pages[page] = benchmarkPage{
			Number: page + 1,
			First:  page == 0,
			Rows:   rows,
		}
	}

	return pages
}

func benchmarkImages(count int, source string) []benchmarkImage {
	images := make([]benchmarkImage, count)
	for i := range images {
		images[i] = benchmarkImage{
			Src:   source,
			Label: fmt.Sprintf("asset-%03d", i+1),
		}
	}

	return images
}

func fetchTVMazeShows(ctx context.Context, client *http.Client) ([]tvMazeShow, int, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second} //nolint:exhaustruct // intentional zero-value fields
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tvMazeShowsURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create TVmaze request: %w", err)
	}

	req.Header.Set("User-Agent", "gowkhtmltopdf-benchmark/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch TVmaze shows: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("%w: HTTP %d", errTVMazeBadStatus, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("read TVmaze response: %w", err)
	}

	var shows []tvMazeShow
	if err := json.Unmarshal(body, &shows); err != nil {
		return nil, 0, fmt.Errorf("decode TVmaze shows: %w", err)
	}

	if len(shows) == 0 {
		return nil, len(body), errTVMazeNoShows
	}

	return shows, len(body), nil
}

func movieListingItems(shows []tvMazeShow, count int) []movieListingItem {
	items := make([]movieListingItem, count)
	for idx := range items {
		show := shows[idx%len(shows)]
		imageURL := ""

		if show.Image != nil {
			imageURL = show.Image.Medium
			if imageURL == "" {
				imageURL = show.Image.Original
			}
		}

		rating := "unrated"
		if show.Rating.Average > 0 {
			rating = fmt.Sprintf("%.1f/10", show.Rating.Average)
		}

		items[idx] = movieListingItem{
			Name:      html.EscapeString(show.Name),
			Type:      html.EscapeString(show.Type),
			Language:  html.EscapeString(show.Language),
			Genres:    html.EscapeString(strings.Join(show.Genres, " · ")),
			Status:    html.EscapeString(show.Status),
			Premiered: html.EscapeString(show.Premiered),
			Rating:    html.EscapeString(rating),
			ImageURL:  html.EscapeString(imageURL),
		}
	}

	return items
}

func liveBenchmarkShows(tb testing.TB) []tvMazeShow {
	tb.Helper()

	ctx, cancel := context.WithTimeout(tb.Context(), 30*time.Second)
	defer cancel()

	shows, _, err := fetchTVMazeShows(ctx, &http.Client{ //nolint:exhaustruct // intentional zero-value fields
		Timeout: 30 * time.Second,
	})
	if err != nil {
		tb.Fatalf("fetch TVmaze shows: %v", err)
	}

	return shows
}

func executeBenchmarkTemplate(tb testing.TB, tpl *template.Template, data any) []byte {
	tb.Helper()

	var rendered bytes.Buffer
	if err := tpl.Execute(&rendered, data); err != nil {
		tb.Fatalf("execute benchmark template: %v", err)
	}

	return append([]byte(nil), rendered.Bytes()...)
}

func benchmarkPDFRequest(html []byte, output io.Writer) *convert.Request {
	global := settings.DefaultPdfGlobal()
	global.Quiet = true
	object := settings.DefaultPdfObject()
	object.Page = ""
	object.Load.InlineHTML = html

	return convert.NewPDFRequest(global, []settings.PdfObject{object}, output, nil)
}

func benchmarkImageRequest(html []byte, output io.Writer) *convert.Request {
	global := settings.DefaultPdfGlobal()
	global.Quiet = true
	imageSettings := settings.DefaultImageGlobal()
	object := settings.DefaultPdfObject()
	object.Page = ""
	object.Load.InlineHTML = html

	return convert.NewImageRequest(global, imageSettings, []settings.PdfObject{object}, output)
}

func benchmarkPDFPageCount(data []byte) int {
	return bytes.Count(data, []byte("/Type /Page\n"))
}

func benchmarkPNG() []byte {
	const size = 48
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	for y := range size {
		for x := range size {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(30 + x*3), //nolint:gosec // bounded by image size
				G: uint8(80 + y*2), //nolint:gosec // bounded by image size
				B: 180,
				A: 255,
			})
		}
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		panic(fmt.Sprintf("encode benchmark PNG: %v", err))
	}

	return encoded.Bytes()
}

func benchmarkDataURL(data []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

// BenchmarkPDFPages measures the full inline-HTML to PDF pipeline for the
// fixed page-size matrix. Template execution is intentionally outside this
// benchmark so it isolates conversion cost for already-materialized HTML.
func BenchmarkPDFPages(b *testing.B) {
	tpl := loadBenchmarkTemplate(b, "report.html.tmpl")
	sources := make(map[int][]byte, len(benchmarkPageSizes))

	for _, pages := range benchmarkPageSizes {
		sources[pages] = executeBenchmarkTemplate(b, tpl, benchmarkTemplateData{ //nolint:exhaustruct,lll // intentional zero-value fields
			Pages: benchmarkPages(pages),
		})
	}

	for _, pages := range benchmarkPageSizes {
		b.Run(fmt.Sprintf("%dPages", pages), func(b *testing.B) {
			var output bytes.Buffer
			req := benchmarkPDFRequest(sources[pages], &output)
			b.ReportMetric(float64(pages), "pages")
			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := convert.Run(b.Context(), req, io.Discard, nil); err != nil {
					b.Fatalf("run PDF benchmark: %v", err)
				}
			}

			b.StopTimer()

			if got := benchmarkPDFPageCount(output.Bytes()); got != pages {
				b.Fatalf("rendered pages = %d, want %d", got, pages)
			}

			b.SetBytes(int64(output.Len()))
		})
	}
}

// BenchmarkTemplatePages measures template execution plus the full PDF
// pipeline. It uses the same page-size matrix as BenchmarkPDFPages.
func BenchmarkTemplatePages(b *testing.B) {
	tpl := loadBenchmarkTemplate(b, "report.html.tmpl")

	for _, pages := range benchmarkPageSizes {
		data := benchmarkTemplateData{Pages: benchmarkPages(pages)} //nolint:exhaustruct // intentional zero-value fields
		b.Run(fmt.Sprintf("%dPages", pages), func(b *testing.B) {
			var output bytes.Buffer
			req := benchmarkPDFRequest(nil, &output)

			b.ReportMetric(float64(pages), "pages")
			b.ResetTimer()

			for range b.N {
				var rendered bytes.Buffer
				if err := tpl.Execute(&rendered, data); err != nil {
					b.Fatalf("execute report template: %v", err)
				}

				req.Objects[0].Load.InlineHTML = rendered.Bytes()

				output.Reset()

				if err := convert.Run(b.Context(), req, io.Discard, nil); err != nil {
					b.Fatalf("run templated PDF benchmark: %v", err)
				}
			}

			b.StopTimer()

			if got := benchmarkPDFPageCount(output.Bytes()); got != pages {
				b.Fatalf("rendered pages = %d, want %d", got, pages)
			}

			b.SetBytes(int64(output.Len()))
		})
	}
}

func benchmarkImageServer(tb testing.TB, sources map[int][]byte, imageData []byte) *httptest.Server {
	tb.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/benchmark-image.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageData)
	})
	mux.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		if !strings.HasPrefix(req.URL.Path, "/document-") {
			http.NotFound(writer, req)

			return
		}

		name := strings.TrimPrefix(req.URL.Path, "/document-")
		name = strings.TrimSuffix(name, ".html")

		pages, err := strconv.Atoi(name)
		if err != nil {
			http.NotFound(writer, req)

			return
		}

		source, ok := sources[pages]
		if !ok {
			http.NotFound(writer, req)

			return
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write(source)
	})

	return httptest.NewServer(mux)
}

func benchmarkOutputDir() string {
	return filepath.Join(
		"..",
		"..",
		"testdata",
		"golden",
		"benchmarks",
		"output",
	)
}

func writeBenchmarkOutput(t *testing.T, name string, data []byte) {
	t.Helper()

	dir := benchmarkOutputDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create benchmark output directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatalf("write benchmark output %s: %v", name, err)
	}
}

// TestGenerateBenchmarkOutputs materializes viewable artifacts for the same
// workloads measured by the benchmarks. It is intentionally explicit so
// normal go test runs do not write files or add filesystem work to timings.
// The output directory is ignored by Git; run this test after benchmarking:
//
//	GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 go test ./internal/convert -run '^TestGenerateBenchmarkOutputs$' -count=1
func TestGenerateBenchmarkOutputs(t *testing.T) { //nolint:cyclop,funlen // materializes several artifact kinds
	t.Parallel()

	if os.Getenv("GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS") != "1" {
		t.Skip("set GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 to write benchmark artifacts")
	}

	reportTemplate := loadBenchmarkTemplate(t, "report.html.tmpl")
	for _, pages := range benchmarkPageSizes {
		source := executeBenchmarkTemplate(t, reportTemplate, benchmarkTemplateData{ //nolint:exhaustruct,lll // intentional zero-value fields
			Pages: benchmarkPages(pages),
		})

		var output bytes.Buffer

		req := benchmarkPDFRequest(source, &output)
		if err := convert.Run(t.Context(), req, io.Discard, nil); err != nil {
			t.Fatalf("generate PDF output for %d pages: %v", pages, err)
		}

		if got := benchmarkPDFPageCount(output.Bytes()); got != pages {
			t.Fatalf("generated PDF pages = %d, want %d", got, pages)
		}

		writeBenchmarkOutput(t, fmt.Sprintf("pdf-pages-%03d.pdf", pages), output.Bytes())

		output.Reset()
		templateRequest := benchmarkPDFRequest(nil, &output)

		if err := reportTemplate.Execute(&output, benchmarkTemplateData{ //nolint:exhaustruct,lll // intentional zero-value fields
			Pages: benchmarkPages(pages),
		}); err != nil {
			t.Fatalf("render report template for %d pages: %v", pages, err)
		}

		templateRequest.Objects[0].Load.InlineHTML = append([]byte(nil), output.Bytes()...)
		output.Reset()

		if err := convert.Run(t.Context(), templateRequest, io.Discard, nil); err != nil {
			t.Fatalf("generate template PDF output for %d pages: %v", pages, err)
		}

		if got := benchmarkPDFPageCount(output.Bytes()); got != pages {
			t.Fatalf("generated template PDF pages = %d, want %d", got, pages)
		}

		writeBenchmarkOutput(t, fmt.Sprintf("template-pages-%03d.pdf", pages), output.Bytes())
	}

	pngData := benchmarkPNG()
	webTemplate := loadBenchmarkTemplate(t, "web-fetch-image.html.tmpl")
	webSources := make(map[int][]byte, len(benchmarkPageSizes))

	for _, images := range benchmarkPageSizes {
		webSources[images] = executeBenchmarkTemplate(t, webTemplate, benchmarkTemplateData{ //nolint:exhaustruct,lll // intentional zero-value fields
			Images:   benchmarkImages(images, "/benchmark-image.png"),
			ImageSrc: "/benchmark-image.png",
		})
	}

	server := benchmarkImageServer(t, webSources, pngData)
	defer server.Close()

	for _, images := range benchmarkPageSizes {
		var output bytes.Buffer

		global := settings.DefaultPdfGlobal()
		global.Quiet = true
		object := settings.DefaultPdfObject()
		object.Page = fmt.Sprintf("%s/document-%d.html", server.URL, images)

		request := convert.NewImageRequest(
			global,
			settings.DefaultImageGlobal(),
			[]settings.PdfObject{object},
			&output,
		)
		if err := imageout.RunRequest(t.Context(), request, io.Discard); err != nil {
			t.Fatalf("generate web image output for %d images: %v", images, err)
		}

		writeBenchmarkOutput(t, fmt.Sprintf("web-fetch-images-%03d.png", images), output.Bytes())
	}

	imageTemplate := loadBenchmarkTemplate(t, "image-grid.html.tmpl")
	imageURL := benchmarkDataURL(pngData)

	for _, images := range benchmarkPageSizes {
		source := executeBenchmarkTemplate(t, imageTemplate, benchmarkTemplateData{ //nolint:exhaustruct,lll // intentional zero-value fields
			Images: benchmarkImages(images, imageURL),
		})

		var output bytes.Buffer

		request := benchmarkImageRequest(source, &output)
		if err := imageout.RunRequest(t.Context(), request, io.Discard); err != nil {
			t.Fatalf("generate inline image output for %d images: %v", images, err)
		}

		writeBenchmarkOutput(t, fmt.Sprintf("inline-images-%03d.png", images), output.Bytes())
	}
}

func liveBenchmarkEnabled(tb testing.TB) {
	tb.Helper()

	if os.Getenv(liveBenchmarkEnv) == "1" {
		return
	}

	tb.Skipf("set %s=1 to use the live TVmaze benchmark", liveBenchmarkEnv)
}

// TestGenerateLiveMovieOutput fetches current TVmaze show data and writes one
// viewable movie-listing PDF and PNG into the ignored benchmark output folder.
// It requires both opt-in environment variables because it uses the public
// internet and writes files:
//
//	GOWKHTMLTOPDF_LIVE_BENCHMARK=1 GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 go test
//	./internal/convert -run '^TestGenerateLiveMovieOutput$' -count=1
func TestGenerateLiveMovieOutput(t *testing.T) {
	t.Parallel()
	liveBenchmarkEnabled(t)

	if os.Getenv("GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS") != "1" {
		t.Skip("set GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 to write benchmark artifacts")
	}

	shows := liveBenchmarkShows(t)
	tpl := loadBenchmarkTemplate(t, "movie-listing.html.tmpl")
	source := executeBenchmarkTemplate(t, tpl, movieListingData{
		Items: movieListingItems(shows, 10),
	})

	var imageOutput bytes.Buffer
	if err := imageout.RunRequest(t.Context(), benchmarkImageRequest(source, &imageOutput), io.Discard); err != nil {
		t.Fatalf("generate live movie PNG: %v", err)
	}

	writeBenchmarkOutput(t, "live-movie-listing-010.png", imageOutput.Bytes())

	var pdfOutput bytes.Buffer
	if err := convert.Run(
		t.Context(),
		benchmarkPDFRequest(source, &pdfOutput),
		io.Discard,
		nil,
	); err != nil {
		t.Fatalf("generate live movie PDF: %v", err)
	}

	writeBenchmarkOutput(t, "live-movie-listing-010.pdf", pdfOutput.Bytes())
}

// BenchmarkLiveMovieData measures a real TVmaze API request. It is opt-in and
// should normally be run with -benchtime=1x to avoid repeated public API calls.
func BenchmarkLiveMovieData(b *testing.B) {
	liveBenchmarkEnabled(b)

	client := &http.Client{Timeout: 30 * time.Second} //nolint:exhaustruct // intentional zero-value fields

	var bodyBytes int

	var showCount int

	b.ResetTimer()

	for range b.N {
		shows, size, err := fetchTVMazeShows(b.Context(), client)
		if err != nil {
			b.Fatalf("fetch live TVmaze data: %v", err)
		}

		bodyBytes = size
		showCount = len(shows)
	}

	b.StopTimer()
	b.SetBytes(int64(bodyBytes))
	b.ReportMetric(float64(showCount), "shows")
}

// BenchmarkLiveMovieListing measures image-mode conversion using a listing
// built from real TVmaze show metadata. The API request is setup work; the
// timed conversion fetches the real poster URLs from TVmaze's image CDN.
func BenchmarkLiveMovieListing(b *testing.B) {
	liveBenchmarkEnabled(b)
	shows := liveBenchmarkShows(b)
	tpl := loadBenchmarkTemplate(b, "movie-listing.html.tmpl")

	for _, images := range benchmarkPageSizes {
		source := executeBenchmarkTemplate(b, tpl, movieListingData{
			Items: movieListingItems(shows, images),
		})

		b.Run(fmt.Sprintf("%dImages", images), func(b *testing.B) {
			var output bytes.Buffer
			req := benchmarkImageRequest(source, &output)

			b.ReportMetric(float64(images), "images")
			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := imageout.RunRequest(b.Context(), req, io.Discard); err != nil {
					b.Fatalf("render live movie listing: %v", err)
				}
			}

			b.StopTimer()
			b.SetBytes(int64(output.Len()))
		})
	}
}

// BenchmarkWebFetchImage measures image-mode conversion of HTML fetched from
// an HTTP server, including the HTTP image fetches. Image mode produces one
// raster canvas, so the matrix is image-tile counts rather than PDF pages.
func BenchmarkWebFetchImage(b *testing.B) {
	tpl := loadBenchmarkTemplate(b, "web-fetch-image.html.tmpl")
	pngData := benchmarkPNG()

	sources := make(map[int][]byte, len(benchmarkPageSizes))
	for _, images := range benchmarkPageSizes {
		sources[images] = executeBenchmarkTemplate(b, tpl, benchmarkTemplateData{ //nolint:exhaustruct,lll // intentional zero-value fields
			Images:   benchmarkImages(images, "/benchmark-image.png"),
			ImageSrc: "/benchmark-image.png",
		})
	}

	server := benchmarkImageServer(b, sources, pngData)
	b.Cleanup(server.Close)

	for _, images := range benchmarkPageSizes {
		b.Run(fmt.Sprintf("%dImages", images), func(b *testing.B) {
			var output bytes.Buffer

			global := settings.DefaultPdfGlobal()
			global.Quiet = true
			imageSettings := settings.DefaultImageGlobal()
			object := settings.DefaultPdfObject()
			object.Page = fmt.Sprintf("%s/document-%d.html", server.URL, images)
			req := convert.NewImageRequest(
				global,
				imageSettings,
				[]settings.PdfObject{object},
				&output,
			)

			b.ReportMetric(float64(images), "images")
			b.ResetTimer()

			for range b.N {
				output.Reset()

				if err := imageout.RunRequest(b.Context(), req, io.Discard); err != nil {
					b.Fatalf("run web-fetch image benchmark: %v", err)
				}
			}

			b.StopTimer()
			b.SetBytes(int64(output.Len()))
		})
	}
}

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
