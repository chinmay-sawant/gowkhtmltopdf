package profiling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"testing"
	"time"

	gowk "github.com/chinmay-sawant/gowkhtmltopdf"
)

const (
	modePDF     = "pdf"
	modePNG     = "image-png"
	modeJPEG    = "image-jpeg"
	modeAll     = "all"
	dirPerm     = 0o755
	filePerm    = 0o644
	minFixtures = 20
	topFixtures = 15
)

var errUnknownMode = errors.New("unknown mode")

// TestProfileGoldenCorpus converts every golden template through the public
// PDF and image entry points and writes heap, allocs, and goroutine profiles
// plus a per-fixture memory summary. It only runs when GOWK_PROFILE_DIR is
// set.
func TestProfileGoldenCorpus(t *testing.T) { //nolint:paralleltest // single-process memory harness
	outDir := strings.TrimSpace(os.Getenv("GOWK_PROFILE_DIR"))
	if outDir == "" {
		t.Skip("set GOWK_PROFILE_DIR to run the profiling harness")
	}

	if err := os.MkdirAll(outDir, dirPerm); err != nil {
		t.Fatalf("create profile dir: %v", err)
	}

	fixtureDir := resolveFixtureDir(t)

	fixtures := discoverFixtures(t, fixtureDir)
	if len(fixtures) < minFixtures {
		t.Fatalf("found %d fixtures in %s, want at least %d", len(fixtures), fixtureDir, minFixtures)
	}

	modes, err := selectedModes()
	if err != nil {
		t.Fatalf("select modes: %v", err)
	}

	for _, mode := range modes {
		runProfile(t, mode, outDir, fixtureDir, fixtures)
	}
}

type fixtureStat struct {
	Fixture           string  `json:"fixture"`
	DurationMS        float64 `json:"durationMs"`
	TotalAllocBytes   uint64  `json:"totalAllocBytes"`
	Mallocs           uint64  `json:"mallocs"`
	Frees             uint64  `json:"frees"`
	RetainedBytes     int64   `json:"retainedBytes"`
	HeapAllocAfterRun uint64  `json:"heapAllocAfterRunBytes"`
	HeapAllocAfterGC  uint64  `json:"heapAllocAfterGcBytes"`
	StackInuseAfter   uint64  `json:"stackInuseAfterBytes"`
	NumGC             uint32  `json:"numGc"`
	Skipped           bool    `json:"skipped,omitempty"`
	Error             string  `json:"error,omitempty"`
}

type runSummary struct {
	Mode             string        `json:"mode"`
	GoVersion        string        `json:"goVersion"`
	NumCPU           int           `json:"numCpu"`
	GOMAXPROCS       int           `json:"gomaxprocs"`
	MemProfileRate   int           `json:"memProfileRate"`
	FixtureCount     int           `json:"fixtureCount"`
	Failures         int           `json:"failures"`
	Skipped          int           `json:"skipped"`
	TotalAllocBytes  uint64        `json:"totalAllocBytes"`
	TotalMallocs     uint64        `json:"totalMallocs"`
	TotalFrees       uint64        `json:"totalFrees"`
	PeakHeapAlloc    uint64        `json:"peakHeapAllocBytes"`
	FinalHeapAlloc   uint64        `json:"finalHeapAllocBytes"`
	FinalHeapInuse   uint64        `json:"finalHeapInuseBytes"`
	FinalStackInuse  uint64        `json:"finalStackInuseBytes"`
	FinalStackSys    uint64        `json:"finalStackSysBytes"`
	FinalHeapObjects uint64        `json:"finalHeapObjects"`
	NumGC            uint32        `json:"numGc"`
	GoroutinesBefore int           `json:"goroutinesBefore"`
	GoroutinesAfter  int           `json:"goroutinesAfter"`
	WallMillis       float64       `json:"wallMillis"`
	Fixtures         []fixtureStat `json:"fixtures"`
}

func runProfile(t *testing.T, mode, outDir, fixtureDir string, fixtures []string) {
	t.Helper()

	var summary runSummary

	summary.Mode = mode
	summary.GoVersion = runtime.Version()
	summary.NumCPU = runtime.NumCPU()
	summary.GOMAXPROCS = runtime.GOMAXPROCS(0)
	summary.MemProfileRate = runtime.MemProfileRate

	ctx := t.Context()
	wallStart := time.Now()

	runtime.GC()
	summary.GoroutinesBefore = runtime.NumGoroutine()

	for _, path := range fixtures {
		summary.Fixtures = append(summary.Fixtures, profileFixture(ctx, t, mode, path, fixtureDir))
	}

	runtime.GC()
	runtime.GC()

	var final runtime.MemStats

	runtime.ReadMemStats(&final)
	summary.finish(final, time.Since(wallStart))

	writeProfiles(t, outDir, mode)
	writeSummary(t, outDir, &summary)
	logSummary(t, &summary)
}

func profileFixture(ctx context.Context, t *testing.T, mode, path, fixtureDir string) fixtureStat {
	t.Helper()

	var before, afterRun, afterGC runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	started := time.Now()
	runErr := runFixture(ctx, mode, path, fixtureDir)
	elapsed := time.Since(started)

	runtime.ReadMemStats(&afterRun)
	runtime.GC()
	runtime.ReadMemStats(&afterGC)

	var stat fixtureStat

	stat.Fixture = filepath.Base(path)
	stat.DurationMS = float64(elapsed.Nanoseconds()) / 1e6
	stat.TotalAllocBytes = afterGC.TotalAlloc - before.TotalAlloc
	stat.Mallocs = afterGC.Mallocs - before.Mallocs
	stat.Frees = afterGC.Frees - before.Frees
	stat.RetainedBytes = int64(afterGC.HeapAlloc) - int64(before.HeapAlloc) //nolint:gosec // memstats fit int64
	stat.HeapAllocAfterRun = afterRun.HeapAlloc
	stat.HeapAllocAfterGC = afterGC.HeapAlloc
	stat.StackInuseAfter = afterGC.StackInuse
	stat.NumGC = afterGC.NumGC - before.NumGC

	if runErr != nil {
		stat.Error = runErr.Error()

		switch {
		case isResourceSkip(mode, runErr):
			stat.Skipped = true

			t.Logf("%s: skipped (%v)", stat.Fixture, runErr)
		default:
			t.Errorf("%s: %v", stat.Fixture, runErr)
		}
	}

	return stat
}

// isResourceSkip reports image conversions that the raster dimension budget
// rejects. Very tall golden templates cannot be rasterized at 1x and are
// recorded as skipped rather than counted as harness failures.
func isResourceSkip(mode string, err error) bool {
	if mode == modePDF || err == nil {
		return false
	}

	return strings.Contains(err.Error(), "raster exceeds resource budget")
}

func (s *runSummary) finish(final runtime.MemStats, wall time.Duration) {
	s.FixtureCount = len(s.Fixtures)

	for _, stat := range s.Fixtures {
		s.TotalAllocBytes += stat.TotalAllocBytes
		s.TotalMallocs += stat.Mallocs
		s.TotalFrees += stat.Frees

		if stat.Skipped {
			s.Skipped++

			continue
		}

		if stat.Error != "" {
			s.Failures++

			continue
		}

		if stat.HeapAllocAfterRun > s.PeakHeapAlloc {
			s.PeakHeapAlloc = stat.HeapAllocAfterRun
		}
	}

	s.FinalHeapAlloc = final.HeapAlloc
	s.FinalHeapInuse = final.HeapInuse
	s.FinalStackInuse = final.StackInuse
	s.FinalStackSys = final.StackSys
	s.FinalHeapObjects = final.HeapObjects
	s.NumGC = final.NumGC
	s.GoroutinesAfter = runtime.NumGoroutine()
	s.WallMillis = float64(wall.Nanoseconds()) / 1e6
}

func writeProfiles(t *testing.T, outDir, mode string) {
	t.Helper()

	writePprof(t, filepath.Join(outDir, "heap-"+mode+".pprof"), "heap", 0)
	writePprof(t, filepath.Join(outDir, "allocs-"+mode+".pprof"), "allocs", 0)
	writePprof(t, filepath.Join(outDir, "goroutine-"+mode+".pprof"), "goroutine", 0)
	writePprof(t, filepath.Join(outDir, "stacks-"+mode+".txt"), "goroutine", 1)
}

func writePprof(t *testing.T, path, name string, debug int) {
	t.Helper()

	profile := pprof.Lookup(name)
	if profile == nil {
		t.Fatalf("pprof profile %q not registered", name)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}

	defer func() { _ = file.Close() }()

	if err := profile.WriteTo(file, debug); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeSummary(t *testing.T, outDir string, summary *runSummary) {
	t.Helper()

	blob, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}

	jsonPath := filepath.Join(outDir, "summary-"+summary.Mode+".json")
	if err := os.WriteFile(jsonPath, blob, filePerm); err != nil {
		t.Fatalf("write %s: %v", jsonPath, err)
	}

	mdPath := filepath.Join(outDir, "summary-"+summary.Mode+".md")
	if err := os.WriteFile(mdPath, []byte(summaryMarkdown(summary)), filePerm); err != nil {
		t.Fatalf("write %s: %v", mdPath, err)
	}
}

func summaryMarkdown(summary *runSummary) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "# Golden corpus memory profile: %s\n\n", summary.Mode)
	fmt.Fprintf(&builder, "- Go: %s\n", summary.GoVersion)
	fmt.Fprintf(&builder, "- Fixtures: %d (skipped: %d, failures: %d)\n",
		summary.FixtureCount, summary.Skipped, summary.Failures)
	fmt.Fprintf(&builder, "- Wall: %.1f ms\n", summary.WallMillis)
	fmt.Fprintf(&builder, "- Total allocations: %s across %d mallocs\n",
		humanBytes(summary.TotalAllocBytes), summary.TotalMallocs)
	fmt.Fprintf(&builder, "- Peak heap after a single conversion: %s\n", humanBytes(summary.PeakHeapAlloc))
	fmt.Fprintf(&builder, "- Final heap after two GCs: %s in %d objects\n",
		humanBytes(summary.FinalHeapAlloc), summary.FinalHeapObjects)
	fmt.Fprintf(&builder, "- Final stack in use: %s of %s system\n",
		humanBytes(summary.FinalStackInuse), humanBytes(summary.FinalStackSys))
	fmt.Fprintf(&builder, "- Goroutines: %d before, %d after\n\n", summary.GoroutinesBefore, summary.GoroutinesAfter)

	top := append([]fixtureStat(nil), summary.Fixtures...)

	sort.SliceStable(top, func(i, j int) bool {
		return top[i].TotalAllocBytes > top[j].TotalAllocBytes
	})

	fmt.Fprintf(&builder, "| Fixture | ms | Total alloc | Retained | Mallocs |\n")
	fmt.Fprintf(&builder, "|---------|---:|------------:|---------:|--------:|\n")

	for idx, stat := range top {
		if idx >= topFixtures {
			break
		}

		if stat.RetainedBytes < 0 {
			stat.RetainedBytes = 0
		}

		fmt.Fprintf(&builder, "| %s | %.1f | %s | %s | %d |\n",
			stat.Fixture, stat.DurationMS, humanBytes(stat.TotalAllocBytes),
			humanBytes(uint64(stat.RetainedBytes)), stat.Mallocs)
	}

	return builder.String()
}

func logSummary(t *testing.T, summary *runSummary) {
	t.Helper()
	t.Logf("mode=%s fixtures=%d skipped=%d failures=%d wall=%.1fms "+
		"total_alloc=%s peak_heap=%s stack_in_use=%s goroutines=%d->%d",
		summary.Mode, summary.FixtureCount, summary.Skipped, summary.Failures, summary.WallMillis,
		humanBytes(summary.TotalAllocBytes), humanBytes(summary.PeakHeapAlloc),
		humanBytes(summary.FinalStackInuse), summary.GoroutinesBefore, summary.GoroutinesAfter)
}

func humanBytes(value uint64) string {
	const mib = 1 << 20

	if value >= mib {
		return fmt.Sprintf("%.1f MiB", float64(value)/mib)
	}

	return fmt.Sprintf("%.1f KiB", float64(value)/1024)
}

func selectedModes() ([]string, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GOWK_PROFILE_MODE")))

	switch mode {
	case "", modeAll:
		return []string{modePDF, modePNG, modeJPEG}, nil
	case modePDF, modePNG, modeJPEG:
		return []string{mode}, nil
	default:
		return nil, fmt.Errorf("%w: GOWK_PROFILE_MODE=%q (want pdf, image-png, image-jpeg, or all)", errUnknownMode, mode)
	}
}

func runFixture(ctx context.Context, mode, path, fixtureDir string) error {
	switch mode {
	case modePDF:
		if err := buildPDFDocument(path, fixtureDir).WritePDF(ctx, io.Discard); err != nil {
			return fmt.Errorf("pdf: %w", err)
		}

		return nil
	case modePNG, modeJPEG:
		format := "png"
		if mode == modeJPEG {
			format = "jpeg"
		}

		if err := buildImageDocument(path, fixtureDir, format).WriteImage(ctx, io.Discard); err != nil {
			return fmt.Errorf("image: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("%w: %q", errUnknownMode, mode)
	}
}

func resolveFixtureDir(tb testing.TB) string {
	tb.Helper()

	if dir := strings.TrimSpace(os.Getenv("GOWK_PROFILE_FIXTURES")); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			tb.Fatalf("resolve fixture dir: %v", err)
		}

		return abs
	}

	candidates := []string{
		filepath.Join("..", "..", "testdata", "golden"),
		filepath.Join("testdata", "golden"),
	}

	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}

		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs
		}
	}

	tb.Fatalf("golden fixture dir not found; set GOWK_PROFILE_FIXTURES")

	return ""
}

func discoverFixtures(tb testing.TB, dir string) []string {
	tb.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatalf("read fixtures: %v", err)
	}

	fixtures := make([]string, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".html") || isCompanionHTML(name) {
			continue
		}

		fixtures = append(fixtures, filepath.Join(dir, name))
	}

	sort.Strings(fixtures)

	return fixtures
}

func isCompanionHTML(name string) bool {
	return strings.HasSuffix(name, "-header.html") || strings.HasSuffix(name, "-footer.html")
}

func fontPaths(fixtureDir string) []string {
	var paths []string

	if info, err := os.Stat("/usr/share/fonts/truetype/droid"); err == nil && info.IsDir() {
		paths = append(paths, "/usr/share/fonts/truetype/droid")
	}

	testFonts := filepath.Join(filepath.Dir(fixtureDir), "fonts")
	if info, err := os.Stat(testFonts); err == nil && info.IsDir() {
		paths = append(paths, testFonts)
	}

	return paths
}

func buildPDFDocument(path, fixtureDir string) *gowk.Document {
	doc := gowk.NewDocument(gowk.Page{Source: gowk.File(path)})
	doc.PageSize = "A4"
	doc.AllowLocalFiles = true
	doc.Allow = []string{fixtureDir}
	doc.FontPaths = fontPaths(fixtureDir)

	background := true
	doc.Background = &background

	attachCompanions(doc, path)

	return doc
}

func buildImageDocument(path, fixtureDir, format string) *gowk.ImageDocument {
	doc := &gowk.ImageDocument{
		Source:          gowk.File(path),
		Format:          format,
		AllowLocalFiles: true,
		Allow:           []string{fixtureDir},
		FontPaths:       fontPaths(fixtureDir),
	}

	background := true
	doc.Background = &background

	return doc
}

// attachCompanions mirrors the golden corpus setup: a body fixture with a
// fixture-NN-header.html or fixture-NN-footer.html companion gets a global
// header/footer with an auto margin.
func attachCompanions(doc *gowk.Document, path string) {
	base := strings.TrimSuffix(path, ".html")

	header := base + "-header.html"
	if _, err := os.Stat(header); err == nil {
		doc.Header = &gowk.HeaderFooter{HTMLURL: header}
		doc.Margin.Top = -1
	}

	footer := base + "-footer.html"
	if _, err := os.Stat(footer); err == nil {
		doc.Footer = &gowk.HeaderFooter{HTMLURL: footer}
		doc.Margin.Bottom = -1
	}
}
