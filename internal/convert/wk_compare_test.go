package convert_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/settings"
)

//nolint:funlen,forbidigo,paralleltest // benchmark comparison table prints live stats to stdout
func TestCompareWithWkhtmltopdfBinary(t *testing.T) {
	wkPath, err := exec.LookPath("wkhtmltopdf")
	if err != nil {
		t.Skip("wkhtmltopdf binary not found in PATH")
	}

	sizes := []int{2}
	tmpDir := t.TempDir()

	tpl := loadBenchmarkTemplate(t, "report.html.tmpl")

	fmt.Println("==========================================================================================")
	fmt.Println("LIVE BENCHMARK COMPARISON: gowkhtmltopdf vs wkhtmltopdf (0.12.6.1)")
	fmt.Printf("Binary Path: %s\n", wkPath)
	fmt.Println("==========================================================================================")
	fmt.Printf("%-10s | %-18s | %-22s | %-15s\n", "Pages", "gowkhtmltopdf", "wkhtmltopdf (0.12.6.1)", "Speedup Factor")
	fmt.Println("------------------------------------------------------------------------------------------")

	for _, pages := range sizes {
		htmlBytes := executeBenchmarkTemplate(t, tpl, benchmarkTemplateData{ //nolint:exhaustruct // partial template data
			Pages: benchmarkPages(pages),
		})

		htmlFile := filepath.Join(tmpDir, fmt.Sprintf("doc_%d.html", pages))
		if err := os.WriteFile(htmlFile, htmlBytes, 0600); err != nil {
			t.Fatalf("write html: %v", err)
		}

		// 1. gowkhtmltopdf (Go engine)
		var goBuf bytes.Buffer

		global := settings.DefaultPdfGlobal()
		global.Quiet = true
		global.Load.EnableLocalFileAccess = true
		global.Load.Allow = []string{"/tmp", "/private/tmp", os.TempDir()}
		object := settings.DefaultPdfObject()
		object.Page = htmlFile
		req := convert.NewPDFRequest(global, []settings.PdfObject{object}, &goBuf, io.Discard)

		// Warm-up run
		_ = convert.Run(t.Context(), req, io.Discard, nil)

		goBuf.Reset()

		startGo := time.Now()

		if err := convert.Run(t.Context(), req, io.Discard, nil); err != nil {
			t.Fatalf("gowkhtmltopdf failed: %v", err)
		}

		durGo := time.Since(startGo)

		// 2. wkhtmltopdf binary
		outPdf := filepath.Join(tmpDir, fmt.Sprintf("out_wk_%d.pdf", pages))
		// Warm-up run
		_ = exec.Command(wkPath, "--quiet", htmlFile, outPdf).Run()

		startWk := time.Now()

		cmd := exec.Command(wkPath, "--quiet", htmlFile, outPdf)
		if err := cmd.Run(); err != nil {
			t.Fatalf("wkhtmltopdf binary failed: %v", err)
		}

		durWk := time.Since(startWk)

		speedup := float64(durWk) / float64(durGo)
		fmt.Printf("%-10d | %-18v | %-22v | %.2fx faster\n",
			pages, durGo.Round(time.Microsecond), durWk.Round(time.Microsecond), speedup)
	}

	fmt.Println("==========================================================================================")
}
