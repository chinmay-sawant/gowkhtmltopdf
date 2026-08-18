package convert_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	cliCompareEnv        = "GOWKHTMLTOPDF_CLI_COMPARE"
	cliCompareRuns       = 3
	cliCompareTimeFormat = "%e %M"
)

type cliTiming struct {
	elapsed time.Duration
	rssKiB  int
	pdfSize int64
	pages   int
}

type cliCompareRow struct {
	pages     int
	gowk      cliTiming
	wkhtml    cliTiming
	gowkPath  string
	wkhtmlBin string
}

// TestCompareWithWkhtmltopdfBinary is a process-level CLI comparison of
// bin/gowkhtmltopdf against the installed wkhtmltopdf binary. The default
// path is a 2-page smoke when both binaries are present. Set
// GOWKHTMLTOPDF_CLI_COMPARE=1 for the full page matrix (median of three
// timed runs, one warmup discarded).
//
//nolint:funlen,forbidigo,paralleltest // live comparison table prints to stdout
func TestCompareWithWkhtmltopdfBinary(t *testing.T) {
	wkPath, err := exec.LookPath("wkhtmltopdf")
	if err != nil {
		t.Skip("wkhtmltopdf binary not found in PATH")
	}

	gowkPath := filepath.Join("..", "..", "bin", "gowkhtmltopdf")
	if _, statErr := os.Stat(gowkPath); statErr != nil {
		t.Skip("bin/gowkhtmltopdf not found; run make build")
	}

	sizes := []int{2}
	runs := 1
	full := os.Getenv(cliCompareEnv) == "1"

	if full {
		sizes = append([]int(nil), benchmarkPageSizes...)
		runs = cliCompareRuns
	}

	tmpDir := t.TempDir()
	tpl := loadBenchmarkTemplate(t, "report.html.tmpl")
	wkVersion := wkhtmltopdfVersion(t, wkPath)

	fmt.Println(strings.Repeat("=", 96))
	fmt.Println("PROCESS CLI COMPARISON: gowkhtmltopdf vs wkhtmltopdf")
	fmt.Printf("gowk:    %s\n", gowkPath)
	fmt.Printf("wkhtml:  %s (%s)\n", wkPath, wkVersion)
	fmt.Printf("flags:   --quiet --enable-local-file-access\n")
	fmt.Printf("runs:    warmup + %d timed (median)\n", runs)
	fmt.Println(strings.Repeat("=", 96))
	fmt.Printf(
		"%-8s | %-12s | %-12s | %-8s | %-14s | %-14s | %-8s\n",
		"Pages",
		"gowk time",
		"wkhtml time",
		"Speedup",
		"gowk RSS",
		"wkhtml RSS",
		"gowk PDF",
	)
	fmt.Println(strings.Repeat("-", 96))

	rows := make([]cliCompareRow, 0, len(sizes))

	for _, pages := range sizes {
		htmlBytes := executeBenchmarkTemplate(t, tpl, benchmarkTemplateData{ //nolint:exhaustruct // partial template data
			Pages: benchmarkPages(pages),
		})
		htmlFile := filepath.Join(tmpDir, fmt.Sprintf("doc_%d.html", pages))

		if writeErr := os.WriteFile(htmlFile, htmlBytes, 0o600); writeErr != nil {
			t.Fatalf("write html: %v", writeErr)
		}

		gowkOut := filepath.Join(tmpDir, fmt.Sprintf("gowk_%d.pdf", pages))
		wkOut := filepath.Join(tmpDir, fmt.Sprintf("wk_%d.pdf", pages))
		gowk := timeCLIMedian(t, gowkPath, htmlFile, gowkOut, pages, runs)
		wkhtml := timeCLIMedian(t, wkPath, htmlFile, wkOut, pages, runs)
		speedup := float64(wkhtml.elapsed) / float64(gowk.elapsed)

		fmt.Printf(
			"%-8d | %-12s | %-12s | %-8s | %-14s | %-14s | %-8d\n",
			pages,
			formatMs(gowk.elapsed),
			formatMs(wkhtml.elapsed),
			fmt.Sprintf("%.2fx", speedup),
			formatKiB(gowk.rssKiB),
			formatKiB(wkhtml.rssKiB),
			gowk.pdfSize,
		)

		rows = append(rows, cliCompareRow{
			pages:     pages,
			gowk:      gowk,
			wkhtml:    wkhtml,
			gowkPath:  gowkPath,
			wkhtmlBin: wkPath,
		})
	}

	fmt.Println(strings.Repeat("=", 96))

	if !full {
		return
	}

	writeCLICompareArtifacts(t, rows, wkVersion)
}

func timeCLIMedian(
	tb testing.TB,
	bin string,
	htmlFile string,
	outPDF string,
	wantPages int,
	runs int,
) cliTiming {
	tb.Helper()

	warmup := timeCLIOnce(tb, bin, htmlFile, outPDF)
	if warmup.pages != wantPages {
		tb.Fatalf("%s warmup pages = %d, want %d", bin, warmup.pages, wantPages)
	}

	samples := make([]cliTiming, 0, runs)
	for range runs {
		samples = append(samples, timeCLIOnce(tb, bin, htmlFile, outPDF))
	}

	elapsed := make([]time.Duration, len(samples))
	rss := make([]int, len(samples))

	for i, sample := range samples {
		elapsed[i] = sample.elapsed
		rss[i] = sample.rssKiB

		if sample.pages != wantPages {
			tb.Fatalf("%s pages = %d, want %d", bin, sample.pages, wantPages)
		}
	}

	slices.Sort(elapsed)
	slices.Sort(rss)

	last := samples[len(samples)-1]

	return cliTiming{
		elapsed: elapsed[len(elapsed)/2],
		rssKiB:  rss[len(rss)/2],
		pdfSize: last.pdfSize,
		pages:   last.pages,
	}
}

func timeCLIOnce(tb testing.TB, bin string, htmlFile string, outPDF string) cliTiming {
	tb.Helper()

	timeFile := outPDF + ".time"
	args := []string{
		"-f",
		cliCompareTimeFormat,
		"-o",
		timeFile,
		"--",
		bin,
		"--quiet",
		"--enable-local-file-access",
		htmlFile,
		outPDF,
	}
	cmd := exec.CommandContext(tb.Context(), "/usr/bin/time", args...)
	cmd.Stderr = os.Stderr
	start := time.Now()

	if err := cmd.Run(); err != nil {
		tb.Fatalf("run %s: %v", bin, err)
	}

	elapsed := time.Since(start)
	rssKiB := parseTimeRSS(tb, timeFile)

	pdfData, err := os.ReadFile(outPDF)
	if err != nil {
		tb.Fatalf("read pdf %s: %v", outPDF, err)
	}

	return cliTiming{
		elapsed: elapsed,
		rssKiB:  rssKiB,
		pdfSize: int64(len(pdfData)),
		pages:   countPDFPages(pdfData),
	}
}

func parseTimeRSS(tb testing.TB, path string) int {
	tb.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read time file: %v", err)
	}

	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		tb.Fatalf("unexpected time output %q", raw)
	}

	rss, err := strconv.Atoi(fields[1])
	if err != nil {
		tb.Fatalf("parse rss %q: %v", fields[1], err)
	}

	return rss
}

func countPDFPages(data []byte) int {
	markers := [][]byte{
		[]byte("/Type /Page\n"),
		[]byte("/Type /Page\r"),
		[]byte("/Type /Page/"),
		[]byte("/Type/Page\n"),
		[]byte("/Type/Page/"),
	}

	var n int
	for _, marker := range markers {
		n += bytes.Count(data, marker)
	}

	return n
}

func wkhtmltopdfVersion(tb testing.TB, bin string) string {
	tb.Helper()

	out, err := exec.CommandContext(tb.Context(), bin, "--version").CombinedOutput()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(out))
}

func formatMs(d time.Duration) string {
	ms := d.Round(time.Millisecond).Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%d ms", ms)
	}

	return fmt.Sprintf("%.3f s", float64(ms)/1000)
}

func formatKiB(kib int) string {
	return formatInt(kib) + " KiB"
}

func formatInt(n int) string {
	digits := strconv.Itoa(n)
	if len(digits) <= 3 {
		return digits
	}

	lead := len(digits) % 3
	if lead == 0 {
		lead = 3
	}

	var builder strings.Builder

	builder.WriteString(digits[:lead])

	for i := lead; i < len(digits); i += 3 {
		builder.WriteByte(',')
		builder.WriteString(digits[i : i+3])
	}

	return builder.String()
}

func writeCLICompareArtifacts(tb testing.TB, rows []cliCompareRow, wkVersion string) {
	tb.Helper()

	dir := filepath.Join("..", "..", "testdata", "golden", "benchmarks")
	csvPath := filepath.Join(dir, "cli-compare-results.csv")
	mdPath := filepath.Join(dir, "cli-compare.md")
	csv := renderCLICompareCSV(rows)
	markdown := renderCLICompareMarkdown(rows, wkVersion)

	if err := os.WriteFile(csvPath, csv, 0o600); err != nil {
		tb.Fatalf("write %s: %v", csvPath, err)
	}

	if err := os.WriteFile(mdPath, markdown, 0o600); err != nil {
		tb.Fatalf("write %s: %v", mdPath, err)
	}

	tb.Logf("wrote %s and %s", csvPath, mdPath)
}

func renderCLICompareCSV(rows []cliCompareRow) []byte {
	var csv bytes.Buffer

	csv.WriteString("pages,gowk_ms,wkhtml_ms,speedup,gowk_rss_kib,wkhtml_rss_kib,gowk_pdf_bytes,wkhtml_pdf_bytes\n")

	for _, row := range rows {
		gowkMS := row.gowk.elapsed.Round(time.Millisecond).Milliseconds()
		wkMS := row.wkhtml.elapsed.Round(time.Millisecond).Milliseconds()
		speedup := float64(row.wkhtml.elapsed) / float64(row.gowk.elapsed)
		fmt.Fprintf(
			&csv,
			"%d,%d,%d,%.3f,%d,%d,%d,%d\n",
			row.pages,
			gowkMS,
			wkMS,
			speedup,
			row.gowk.rssKiB,
			row.wkhtml.rssKiB,
			row.gowk.pdfSize,
			row.wkhtml.pdfSize,
		)
	}

	return csv.Bytes()
}

func renderCLICompareMarkdown(rows []cliCompareRow, wkVersion string) []byte {
	var markdown bytes.Buffer

	markdown.WriteString("# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf\n\n")
	markdown.WriteString("Process-level measurement. Each cell is the median of three timed runs after one warmup.\n")
	markdown.WriteString("Wall time is Go `time.Since` around `/usr/bin/time`; ")
	markdown.WriteString("RSS is peak resident set from `%M` (KiB).\n")
	markdown.WriteString("Both binaries used `--quiet --enable-local-file-access` ")
	markdown.WriteString("on the same generated report fixture.\n\n")
	fmt.Fprintf(&markdown, "- gowkhtmltopdf: `%s` (generic CLI)\n", rows[0].gowkPath)
	fmt.Fprintf(&markdown, "- wkhtmltopdf: `%s` (%s)\n", rows[0].wkhtmlBin, wkVersion)
	markdown.WriteString("- Reproduce: `make bench-cli-compare`\n\n")
	markdown.WriteString("| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS |")
	markdown.WriteString(" wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |\n")
	markdown.WriteString("|---:|---:|---:|---:|---:|---:|---:|---:|\n")

	for _, row := range rows {
		speedup := float64(row.wkhtml.elapsed) / float64(row.gowk.elapsed)
		fmt.Fprintf(
			&markdown,
			"| %d | %s | %s | %.2fx | %s | %s | %s | %s |\n",
			row.pages,
			formatMs(row.gowk.elapsed),
			formatMs(row.wkhtml.elapsed),
			speedup,
			formatKiB(row.gowk.rssKiB),
			formatKiB(row.wkhtml.rssKiB),
			formatInt(int(row.gowk.pdfSize)),
			formatInt(int(row.wkhtml.pdfSize)),
		)
	}

	return markdown.Bytes()
}
