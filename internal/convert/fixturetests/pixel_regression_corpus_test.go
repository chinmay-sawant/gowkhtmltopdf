package fixturetests

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	errUnexpectedReferenceEntry  = errors.New("unexpected entry in approved reference directory")
	errReferenceInventoryDiffers = errors.New("approved reference inventory differs")
	errInvalidVisualFixtureName  = errors.New("invalid numbered fixture name")
	errInvalidVisualFixtureID    = errors.New("invalid fixture ID in name")
	errVisualFixtureOutOfRange   = errors.New("fixture is outside the expected range 01-64")
	errVisualFixturePrefix       = errors.New("fixture must use a two-digit prefix")
	errVisualFixtureCount        = errors.New("unexpected number of body HTML files for fixture")
	errVisualFixturePDFCount     = errors.New("unexpected number of numbered fixture PDFs")
)

const (
	visualFixtureLastID = 64
	visualFixturePDFs   = 65
)

// TestVisualReferenceInventory keeps the approved PDF set aligned with the
// numbered HTML fixtures. It runs even when Ghostscript comparison is off.
func TestVisualReferenceInventory(t *testing.T) {
	t.Parallel()

	if err := checkVisualReferenceInventory(); err != nil {
		t.Fatal(err)
	}
}

func checkVisualReferenceInventory() error {
	fixtures, err := visualGoldenFixtures()
	if err != nil {
		return err
	}

	want := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		want = append(want, strings.TrimSuffix(fixture, ".html")+".pdf")
	}

	referenceDir := filepath.Join("..", "..", "..", "output", "validated")

	entries, err := os.ReadDir(referenceDir)
	if err != nil {
		return fmt.Errorf("read approved reference directory %q: %w", referenceDir, err)
	}

	got := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".pdf" {
			return fmt.Errorf("%w: %q", errUnexpectedReferenceEntry, entry.Name())
		}

		got = append(got, entry.Name())
	}

	if !slices.Equal(got, want) {
		return fmt.Errorf("%w:\n got: %v\nwant: %v", errReferenceInventoryDiffers, got, want)
	}

	return nil
}

// TestVisualGoldenFixtureCorpus compares every numbered fixture to its
// manually approved PDF. It runs serially to avoid shared renderer state.
//
//nolint:paralleltest // Shared font and subset state requires serial fixture rendering.
func TestVisualGoldenFixtureCorpus(t *testing.T) {
	executable := os.Getenv("GOWKHTMLTOPDF_VISUAL_GS")
	if executable == "" {
		t.Skip("set GOWKHTMLTOPDF_VISUAL_GS to run the pinned Ghostscript fixture corpus comparison")
	}

	if err := checkVisualReferenceInventory(); err != nil {
		t.Fatal(err)
	}

	fixtures, err := visualGoldenFixtures()
	if err != nil {
		t.Fatal(err)
	}

	//nolint:paralleltest // Keep fixture subtests serial to avoid shared renderer state.
	for _, fixture := range fixtures {
		t.Run(strings.TrimSuffix(fixture, ".html"), func(t *testing.T) {
			referencePath, err := validatedReferencePath(fixture)
			if err != nil {
				t.Fatalf("validatedReferencePath: %v", err)
			}

			freshPDF := runPDF(t, requestForFixture(t, fixture))

			err = compareFixturePDF(t.Context(), visualComparison{
				executable:    executable,
				fixture:       strings.TrimSuffix(fixture, ".html"),
				referencePath: referencePath,
				freshPDF:      freshPDF,
			})
			if err != nil {
				t.Fatalf("visual fixture comparison: %v", err)
			}
		})
	}
}

func visualGoldenFixtures() ([]string, error) {
	entries, err := os.ReadDir(goldenDir())
	if err != nil {
		return nil, fmt.Errorf("read golden fixture directory: %w", err)
	}

	counts := make(map[int]int, visualFixtureLastID)
	fixtures := make([]string, 0, visualFixturePDFs)

	for _, entry := range entries {
		if !isVisualBodyFixture(entry) {
			continue
		}

		name := entry.Name()

		fixtureNumber, err := visualFixtureNumber(name)
		if err != nil {
			return nil, err
		}

		counts[fixtureNumber]++

		fixtures = append(fixtures, name)
	}

	if err := validateVisualFixtureCounts(counts, len(fixtures)); err != nil {
		return nil, err
	}

	sort.Strings(fixtures)

	return fixtures, nil
}

func isVisualBodyFixture(entry os.DirEntry) bool {
	name := entry.Name()

	return !entry.IsDir() &&
		filepath.Ext(name) == ".html" &&
		strings.HasPrefix(name, "fixture-") &&
		!isHFCompanionHTML(name)
}

func visualFixtureNumber(name string) (int, error) {
	prefix := fixtureIDPrefix(name)
	if prefix == "" {
		return 0, fmt.Errorf("%w: %q", errInvalidVisualFixtureName, name)
	}

	fixtureNumber, err := strconv.Atoi(strings.TrimPrefix(prefix, "fixture-"))
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", errInvalidVisualFixtureID, name, err)
	}

	if fixtureNumber < 1 || fixtureNumber > visualFixtureLastID {
		return 0, fmt.Errorf("%w: %q", errVisualFixtureOutOfRange, name)
	}

	wantPrefix := fmt.Sprintf("fixture-%02d-", fixtureNumber)
	if !strings.HasPrefix(name, wantPrefix) {
		return 0, fmt.Errorf("%w: %q, want %q", errVisualFixturePrefix, name, wantPrefix)
	}

	return fixtureNumber, nil
}

func validateVisualFixtureCounts(counts map[int]int, fixtureCount int) error {
	for fixtureNumber := 1; fixtureNumber <= visualFixtureLastID; fixtureNumber++ {
		want := 1
		if fixtureNumber == 29 {
			want = 2
		}

		if got := counts[fixtureNumber]; got != want {
			return fmt.Errorf("%w: %02d has %d, want %d", errVisualFixtureCount, fixtureNumber, got, want)
		}
	}

	if fixtureCount != visualFixturePDFs {
		return fmt.Errorf("%w: got %d, want %d", errVisualFixturePDFCount, fixtureCount, visualFixturePDFs)
	}

	return nil
}
