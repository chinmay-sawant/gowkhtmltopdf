package fixturetests

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	visualGhostscriptVersion = "10.08.0"
	visualRasterDPI          = 150
	visualTextAlphaBits      = 4
	visualGraphicsAlphaBits  = 4
)

var (
	errInvalidGoldenFixture       = errors.New("invalid golden fixture name")
	errApprovedReferenceMissing   = errors.New("approved reference PDF is missing")
	errGhostscriptVersionMismatch = errors.New("ghostscript version does not match pinned version")
	errNoRasterizedPages          = errors.New("ghostscript produced no page images")
	errUnexpectedRasterFormat     = errors.New("rasterized page has unexpected format")
	errRasterPageCountMismatch    = errors.New("raster page count mismatch")
	errEmptyRasterPages           = errors.New("no rendered pages")
	errNilRasterPage              = errors.New("raster page is nil")
	errRasterDimensionsMismatch   = errors.New("raster dimensions differ")
	errRasterPixelsMismatch       = errors.New("raster pixels differ")
)

// validatedReferencePath maps a golden HTML fixture to its approved PDF.
func validatedReferencePath(goldenHTML string) (string, error) {
	name := filepath.Base(goldenHTML)
	if filepath.Ext(name) != ".html" || fixtureIDPrefix(name) == "" {
		return "", fmt.Errorf("%w: %q", errInvalidGoldenFixture, goldenHTML)
	}

	pdfName := strings.TrimSuffix(name, ".html") + ".pdf"

	return filepath.Join("..", "..", "..", "output", "validated", pdfName), nil
}

// readApprovedPDF reads a reference without creating or changing it.
func readApprovedPDF(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %q", errApprovedReferenceMissing, path)
	}

	if err != nil {
		return nil, fmt.Errorf("read approved reference PDF %q: %w", path, err)
	}

	return data, nil
}

// requirePinnedGhostscript rejects renderer versions that could change pixels.
func requirePinnedGhostscript(ctx context.Context, executable string) error {
	output, err := exec.CommandContext(ctx, executable, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("read Ghostscript version: %w: %s", err, strings.TrimSpace(string(output)))
	}

	got := strings.TrimSpace(string(output))
	if got != visualGhostscriptVersion {
		return fmt.Errorf("%w: got %q, want %q", errGhostscriptVersionMismatch, got, visualGhostscriptVersion)
	}

	return nil
}

// rasterizePDF renders every PDF page to an RGB PNG at the fixed comparison settings.
func rasterizePDF(ctx context.Context, executable, pdfPath string) ([]image.Image, error) {
	outputDir, err := os.MkdirTemp("", "gowkhtmltopdf-raster-")
	if err != nil {
		return nil, fmt.Errorf("create raster directory: %w", err)
	}
	defer os.RemoveAll(outputDir)

	outputPattern := filepath.Join(outputDir, "page-%04d.png")
	args := []string{
		"-dSAFER",
		"-dBATCH",
		"-dNOPAUSE",
		"-sDEVICE=png16m",
		fmt.Sprintf("-r%d", visualRasterDPI),
		fmt.Sprintf("-dTextAlphaBits=%d", visualTextAlphaBits),
		fmt.Sprintf("-dGraphicsAlphaBits=%d", visualGraphicsAlphaBits),
		"-sOutputFile=" + outputPattern,
		pdfPath,
	}

	output, err := exec.CommandContext(ctx, executable, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("rasterize PDF %q with Ghostscript: %w: %s", pdfPath, err, strings.TrimSpace(string(output)))
	}

	paths, err := filepath.Glob(filepath.Join(outputDir, "page-*.png"))
	if err != nil {
		return nil, fmt.Errorf("list rasterized pages for %q: %w", pdfPath, err)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("%w for %q", errNoRasterizedPages, pdfPath)
	}

	pages := make([]image.Image, 0, len(paths))

	for _, path := range paths {
		page, err := decodePNG(path)
		if err != nil {
			return nil, err
		}

		pages = append(pages, page)
	}

	return pages, nil
}

func decodePNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open rasterized page %q: %w", path, err)
	}

	page, format, decodeErr := image.Decode(file)
	if decodeErr != nil {
		_ = file.Close()

		return nil, fmt.Errorf("decode rasterized page %q: %w", path, decodeErr)
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close rasterized page %q: %w", path, err)
	}

	if format != "png" {
		return nil, fmt.Errorf("%w: %q has format %q", errUnexpectedRasterFormat, path, format)
	}

	return page, nil
}

type visualComparison struct {
	executable    string
	fixture       string
	referencePath string
	freshPDF      []byte
	diffDir       string
}

// compareFixturePDF compares fresh PDF bytes with a read-only approved reference.
func compareFixturePDF(ctx context.Context, comparison visualComparison) error {
	referencePDF, err := readApprovedPDF(comparison.referencePath)
	if err != nil {
		return fmt.Errorf("%s: %w", comparison.fixture, err)
	}

	workDir, err := os.MkdirTemp("", "gowkhtmltopdf-visual-")
	if err != nil {
		return fmt.Errorf("create visual comparison directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	referenceCopy := filepath.Join(workDir, "reference.pdf")
	if err := os.WriteFile(referenceCopy, referencePDF, 0o600); err != nil {
		return fmt.Errorf("copy approved reference PDF to temporary storage: %w", err)
	}

	freshPath := filepath.Join(workDir, "fresh.pdf")
	if err := os.WriteFile(freshPath, comparison.freshPDF, 0o600); err != nil {
		return fmt.Errorf("write fresh PDF to temporary storage: %w", err)
	}

	if err := requirePinnedGhostscript(ctx, comparison.executable); err != nil {
		return fmt.Errorf("%s: %w", comparison.fixture, err)
	}

	referencePages, err := rasterizePDF(ctx, comparison.executable, referenceCopy)
	if err != nil {
		return fmt.Errorf("%s reference: %w", comparison.fixture, err)
	}

	freshPages, err := rasterizePDF(ctx, comparison.executable, freshPath)
	if err != nil {
		return fmt.Errorf("%s fresh output: %w", comparison.fixture, err)
	}

	return compareRasterPages(comparison.fixture, referencePages, freshPages, comparison.diffDir)
}

// compareRasterPages requires equal page counts, dimensions, and RGB pixels.
func compareRasterPages(fixture string, reference, fresh []image.Image, diffDir string) error {
	if len(reference) != len(fresh) {
		page := min(len(reference), len(fresh)) + 1

		return fmt.Errorf("%w for %s at page %d: reference has %d, fresh has %d",
			errRasterPageCountMismatch, fixture, page, len(reference), len(fresh))
	}

	if len(reference) == 0 {
		return fmt.Errorf("%w for %s", errEmptyRasterPages, fixture)
	}

	actualDiffDir := diffDir

	dimensionErrors := comparePageDimensions(fixture, reference, fresh, &actualDiffDir)
	if len(dimensionErrors) > 0 {
		return errors.Join(dimensionErrors...)
	}

	return comparePagePixels(fixture, reference, fresh, &actualDiffDir)
}

func comparePageDimensions(fixture string, reference, fresh []image.Image, diffDir *string) []error {
	pageErrors := make([]error, 0, len(reference))

	for pageIndex, referencePage := range reference {
		freshPage := fresh[pageIndex]
		pageNumber := pageIndex + 1

		if referencePage == nil || freshPage == nil {
			pageErrors = append(pageErrors, fmt.Errorf("%s page %d: %w", fixture, pageNumber, errNilRasterPage))

			continue
		}

		if referencePage.Bounds().Size() == freshPage.Bounds().Size() {
			continue
		}

		diff := rasterDiff(referencePage, freshPage)

		diffPath, err := saveVisualDiff(diffDir, fixture, pageNumber, diff)
		if err != nil {
			pageErrors = append(pageErrors, fmt.Errorf("%s page %d dimensions differ and diff could not be saved: %w",
				fixture, pageNumber, err))

			continue
		}

		pageErrors = append(pageErrors, fmt.Errorf(
			"%w: %s page %d, reference is %dx%d, fresh is %dx%d; diff image: %s",
			errRasterDimensionsMismatch,
			fixture,
			pageNumber,
			referencePage.Bounds().Dx(),
			referencePage.Bounds().Dy(),
			freshPage.Bounds().Dx(),
			freshPage.Bounds().Dy(),
			diffPath,
		))
	}

	return pageErrors
}

func comparePagePixels(fixture string, reference, fresh []image.Image, diffDir *string) error {
	pageErrors := make([]error, 0, len(reference))

	for pageIndex, referencePage := range reference {
		changedPixels, maxChannelDelta, diff := compareRasterPage(referencePage, fresh[pageIndex])
		if changedPixels > 0 {
			pageNumber := pageIndex + 1

			diffPath, err := saveVisualDiff(diffDir, fixture, pageNumber, diff)
			if err != nil {
				pageErrors = append(pageErrors, fmt.Errorf(
					"%s page %d differs at %d RGB pixel(s), max channel delta %d, diff image could not be saved: %w",
					fixture,
					pageNumber,
					changedPixels,
					maxChannelDelta,
					err,
				))
			} else {
				pageErrors = append(pageErrors, fmt.Errorf(
					"%w: %s page %d, %d RGB pixel(s), max channel delta %d; diff image: %s",
					errRasterPixelsMismatch, fixture, pageNumber, changedPixels, maxChannelDelta, diffPath))
			}
		}
	}

	return errors.Join(pageErrors...)
}

func compareRasterPage(reference, fresh image.Image) (int, uint32, *image.NRGBA) {
	refBounds := reference.Bounds()
	freshBounds := fresh.Bounds()
	width := max(refBounds.Dx(), freshBounds.Dx())
	height := max(refBounds.Dy(), freshBounds.Dy())
	changedPixels := 0
	maxChannelDelta := uint32(0)
	diff := image.NewNRGBA(image.Rect(0, 0, width, height))

	for row := range height {
		for column := range width {
			inReference := column < refBounds.Dx() && row < refBounds.Dy()
			inFresh := column < freshBounds.Dx() && row < freshBounds.Dy()

			if !inReference || !inFresh {
				diff.SetNRGBA(column, row, color.NRGBA{R: 255, G: 191, A: 255})

				changedPixels++
				maxChannelDelta = 255
			} else {
				want := rgbChannels(reference.At(refBounds.Min.X+column, refBounds.Min.Y+row))
				got := rgbChannels(fresh.At(freshBounds.Min.X+column, freshBounds.Min.Y+row))

				pixelDelta := max(channelDelta(want[0], got[0]), channelDelta(want[1], got[1]), channelDelta(want[2], got[2]))
				if pixelDelta > 0 {
					diff.SetNRGBA(column, row, color.NRGBA{R: 255, B: 255, A: 255})

					changedPixels++
					maxChannelDelta = max(maxChannelDelta, pixelDelta)
				}
			}
		}
	}

	return changedPixels, maxChannelDelta, diff
}

func rgbChannels(pixel color.Color) [3]uint32 {
	red, green, blue, _ := pixel.RGBA()

	return [3]uint32{red, green, blue}
}

func rasterDiff(reference, fresh image.Image) *image.NRGBA {
	_, _, diff := compareRasterPage(reference, fresh)

	return diff
}

func channelDelta(want, got uint32) uint32 {
	if want > got {
		return (want - got) / 257
	}

	return (got - want) / 257
}

func saveVisualDiff(dir *string, fixture string, page int, diff *image.NRGBA) (string, error) {
	if *dir == "" {
		created, err := os.MkdirTemp("", "gowkhtmltopdf-visual-diff-")
		if err != nil {
			return "", fmt.Errorf("create diff directory: %w", err)
		}

		*dir = created
	}

	if err := os.MkdirAll(*dir, 0o700); err != nil {
		return "", fmt.Errorf("create diff directory %q: %w", *dir, err)
	}

	name := strings.TrimSuffix(filepath.Base(fixture), filepath.Ext(fixture))
	path := filepath.Join(*dir, fmt.Sprintf("%s-page-%02d-diff.png", name, page))

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("create diff image %q: %w", path, err)
	}

	if err := png.Encode(file, diff); err != nil {
		_ = file.Close()

		return "", fmt.Errorf("write diff image %q: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close diff image %q: %w", path, err)
	}

	return path, nil
}
