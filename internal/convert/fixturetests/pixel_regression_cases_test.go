package fixturetests

import (
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatedReferencePath(t *testing.T) {
	t.Parallel()

	path, err := validatedReferencePath("testdata/golden/fixture-01-simple-invoice.html")
	if err != nil {
		t.Fatalf("validatedReferencePath: %v", err)
	}

	want := filepath.Join("..", "..", "..", "output", "validated", "fixture-01-simple-invoice.pdf")
	if path != want {
		t.Fatalf("validatedReferencePath = %q, want %q", path, want)
	}
}

func TestCompareRasterPagesIdentical(t *testing.T) {
	t.Parallel()

	page := solidRaster(2, 2, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	if err := compareRasterPages("fixture-01-simple-invoice", []image.Image{page}, []image.Image{page}, ""); err != nil {
		t.Fatalf("compareRasterPages: %v", err)
	}
}

func TestCompareRasterPagesChangedPixel(t *testing.T) {
	t.Parallel()

	reference := solidRaster(2, 3, color.NRGBA{A: 255})
	fresh := solidRaster(2, 3, color.NRGBA{A: 255})
	fresh.SetNRGBA(0, 0, color.NRGBA{R: 20, G: 30, B: 40, A: 255})

	diffDir := t.TempDir()

	err := compareRasterPages("fixture-01-simple-invoice", []image.Image{reference}, []image.Image{fresh}, diffDir)
	requireComparisonError(t, err, errRasterPixelsMismatch, "fixture-01-simple-invoice page 1, 1 RGB pixel(s)")
}

func TestCompareRasterPagesChangedDimensions(t *testing.T) {
	t.Parallel()

	reference := solidRaster(2, 2, color.NRGBA{A: 255})
	fresh := solidRaster(3, 2, color.NRGBA{A: 255})
	diffDir := t.TempDir()

	err := compareRasterPages("fixture-01-simple-invoice", []image.Image{reference}, []image.Image{fresh}, diffDir)
	requireComparisonError(t, err, errRasterDimensionsMismatch, "fixture-01-simple-invoice page 1")

	diffPath := filepath.Join(diffDir, "fixture-01-simple-invoice-page-01-diff.png")
	if _, err := os.Stat(diffPath); err != nil {
		t.Fatalf("stat generated dimension diff %q: %v", diffPath, err)
	}
}

func TestCompareRasterPagesChangedPageCount(t *testing.T) {
	t.Parallel()

	reference := []image.Image{solidRaster(2, 2, color.NRGBA{A: 255})}

	err := compareRasterPages("fixture-01-simple-invoice", reference, []image.Image{}, "")
	requireComparisonError(t, err, errRasterPageCountMismatch, "fixture-01-simple-invoice at page 1")
}

func TestCompareFixturePDFFailsForMissingReference(t *testing.T) {
	t.Parallel()

	err := compareFixturePDF(t.Context(), visualComparison{
		fixture:       "fixture-01-simple-invoice",
		referencePath: filepath.Join(t.TempDir(), "missing.pdf"),
	})
	requireComparisonError(t, err, errApprovedReferenceMissing, "fixture-01-simple-invoice")
}

func TestGhostscriptHelperCommands(t *testing.T) {
	t.Parallel()

	validVersion := writeTestExecutable(t, "printf '10.08.0\\n'\n")
	if err := requirePinnedGhostscript(t.Context(), validVersion); err != nil {
		t.Fatalf("requirePinnedGhostscript for pinned version: %v", err)
	}

	otherVersion := writeTestExecutable(t, "printf '9.55.0\\n'\n")

	err := requirePinnedGhostscript(t.Context(), otherVersion)
	requireComparisonError(t, err, errGhostscriptVersionMismatch, "9.55.0")

	failingRenderer := writeTestExecutable(t, "printf 'raster failure\\n' >&2\nexit 1\n")

	_, err = rasterizePDF(t.Context(), failingRenderer, "fixture.pdf")
	requireComparisonError(t, err, nil, "raster failure")
}

func TestVisualGoldenFixture01SimpleInvoice(t *testing.T) {
	t.Parallel()

	executable := os.Getenv("GOWKHTMLTOPDF_VISUAL_GS")
	if executable == "" {
		t.Skip("set GOWKHTMLTOPDF_VISUAL_GS to run the pinned Ghostscript fixture comparison")
	}

	freshPDF := fixture01CandidatePDF(t)

	referencePath, err := validatedReferencePath("fixture-01-simple-invoice.html")
	if err != nil {
		t.Fatalf("validatedReferencePath: %v", err)
	}

	err = compareFixturePDF(t.Context(), visualComparison{
		executable:    executable,
		fixture:       "fixture-01-simple-invoice",
		referencePath: referencePath,
		freshPDF:      freshPDF,
	})
	if err != nil {
		t.Fatalf("visual fixture comparison: %v", err)
	}
}

func TestVisualGoldenFixture01MutationWritesDiff(t *testing.T) {
	t.Parallel()

	executable := os.Getenv("GOWKHTMLTOPDF_VISUAL_GS")
	if executable == "" {
		t.Skip("set GOWKHTMLTOPDF_VISUAL_GS to run the local visual mutation check")
	}

	request := requestForFixture(t, "fixture-01-simple-invoice.html")
	fixturePath := request.Objects[0].Page

	html, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read temporary fixture copy: %v", err)
	}

	mutatedHTML := strings.Replace(string(html), "color: #1a3d6d", "color: #96320e", 1)
	if mutatedHTML == string(html) {
		t.Fatal("fixture color to mutate was not found")
	}

	if err := os.WriteFile(fixturePath, []byte(mutatedHTML), 0o600); err != nil {
		t.Fatalf("write local visual mutation: %v", err)
	}

	var diffDir string
	if diffDir = os.Getenv("GOWKHTMLTOPDF_VISUAL_DIFF_DIR"); diffDir == "" {
		diffDir = t.TempDir()
	}

	referencePath, err := validatedReferencePath("fixture-01-simple-invoice.html")
	if err != nil {
		t.Fatalf("validatedReferencePath: %v", err)
	}

	freshPDF := runPDF(t, request)
	err = compareFixturePDF(t.Context(), visualComparison{
		executable:    executable,
		fixture:       "fixture-01-simple-invoice",
		referencePath: referencePath,
		freshPDF:      freshPDF,
		diffDir:       diffDir,
	})
	requireComparisonError(t, err, errRasterPixelsMismatch, "fixture-01-simple-invoice page 1")

	diffPath := filepath.Join(diffDir, "fixture-01-simple-invoice-page-01-diff.png")
	if _, err := os.Stat(diffPath); err != nil {
		t.Fatalf("stat generated mutation diff %q: %v", diffPath, err)
	}

	t.Logf("local mutation failed as expected; diff image: %s", diffPath)
}

func requireComparisonError(t *testing.T, err, target error, wantText string) {
	t.Helper()

	if target != nil && !errors.Is(err, target) {
		t.Fatalf("error = %v, want %v", err, target)
	}

	if err == nil || !strings.Contains(err.Error(), wantText) {
		t.Fatalf("error = %v, want text %q", err, wantText)
	}
}

func fixture01CandidatePDF(t *testing.T) []byte {
	t.Helper()

	if candidatePath := os.Getenv("GOWKHTMLTOPDF_VISUAL_CANDIDATE"); candidatePath != "" {
		if !filepath.IsAbs(candidatePath) {
			candidatePath = filepath.Join("..", "..", "..", candidatePath)
		}

		freshPDF, err := os.ReadFile(candidatePath)
		if err != nil {
			t.Fatalf("read visual candidate %q: %v", candidatePath, err)
		}

		return freshPDF
	}

	request := requestForFixture(t, "fixture-01-simple-invoice.html")

	return runPDF(t, request)
}

func solidRaster(width, height int, fill color.NRGBA) *image.NRGBA {
	page := image.NewNRGBA(image.Rect(0, 0, width, height))

	for row := range height {
		for column := range width {
			page.SetNRGBA(column, row, fill)
		}
	}

	return page
}

func writeTestExecutable(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "ghostscript")
	content := []byte("#!/bin/sh\n" + script)

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write test executable: %v", err)
	}

	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("make test executable runnable: %v", err)
	}

	return path
}
