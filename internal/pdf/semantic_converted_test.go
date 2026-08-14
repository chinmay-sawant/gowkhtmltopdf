package pdf_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

//nolint:cyclop,funlen // table-driven oracle fixtures share assertion logic
func TestSemanticPDFOracleConvertedFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		file     string
		minPages int
		needles  []string
		uri      bool
		image    bool
		dest     bool
	}{
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-01-simple-invoice.html",
			minPages: 1,
			needles:  []string{"Invoice", "234.40"},
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-06-external-link.html",
			minPages: 1,
			needles:  []string{"Partner Handbook"},
			uri:      true,
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-07-image-logo.html",
			minPages: 1,
			needles:  []string{"Nordwind"},
			image:    true,
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-24-internal-anchors.html",
			minPages: 2,
			needles:  []string{"Internal link report", "Appendix"},
			dest:     true,
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-55-lantern-cooperative-report.html",
			minPages: 3,
			needles:  []string{"NORTHLINE"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.file, func(t *testing.T) {
			t.Parallel()

			data := convertGoldenFixture(t, testCase.file)

			doc, err := pdf.ParseSemantic(data)
			if err != nil {
				t.Fatalf("ParseSemantic: %v", err)
			}

			if doc.Version != "1.4" {
				t.Errorf("doc.Version = %q, want 1.4", doc.Version)
			}

			if doc.PageCount() < testCase.minPages {
				t.Errorf("pages = %d, want >= %d", doc.PageCount(), testCase.minPages)
			}

			text := doc.DocumentText()

			pos := 0
			for _, needle := range testCase.needles {
				idx := strings.Index(text[pos:], needle)
				if idx < 0 {
					t.Errorf("extracted text missing %q after offset %d; text=%q", needle, pos, text)

					continue
				}

				pos += idx + len(needle)
			}

			if testCase.uri && !doc.HasURI() {
				t.Error("expected a URI annotation")
			}

			if testCase.image && !doc.HasImageXObject() {
				t.Error("expected an image XObject")
			}

			if testCase.dest && !doc.HasInternalDest() {
				t.Error("expected an internal destination / GoTo annotation")
			}
		})
	}
}

//nolint:cyclop,funlen // table-driven oracle fixtures share assertion logic for PDF 1.7
func TestSemanticPDF17OracleConvertedFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		file     string
		minPages int
		needles  []string
		uri      bool
		image    bool
		dest     bool
	}{
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-01-simple-invoice.html",
			minPages: 1,
			needles:  []string{"Invoice", "234.40"},
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-06-external-link.html",
			minPages: 1,
			needles:  []string{"Partner Handbook"},
			uri:      true,
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-07-image-logo.html",
			minPages: 1,
			needles:  []string{"Nordwind"},
			image:    true,
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-24-internal-anchors.html",
			minPages: 2,
			needles:  []string{"Internal link report", "Appendix"},
			dest:     true,
		},
		{ //nolint:exhaustruct // zero-value fields not exercised by this case
			file:     "fixture-55-lantern-cooperative-report.html",
			minPages: 3,
			needles:  []string{"NORTHLINE"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.file, func(t *testing.T) {
			t.Parallel()

			data := convertGoldenFixtureWithVersion(t, testCase.file, "1.7")

			doc, err := pdf.ParseSemantic(data)
			if err != nil {
				t.Fatalf("ParseSemantic: %v", err)
			}

			if doc.Version != "1.7" {
				t.Errorf("doc.Version = %q, want 1.7", doc.Version)
			}

			if doc.PageCount() < testCase.minPages {
				t.Errorf("pages = %d, want >= %d", doc.PageCount(), testCase.minPages)
			}

			text := doc.DocumentText()

			pos := 0
			for _, needle := range testCase.needles {
				idx := strings.Index(text[pos:], needle)
				if idx < 0 {
					t.Errorf("extracted text missing %q after offset %d; text=%q", needle, pos, text)

					continue
				}

				pos += idx + len(needle)
			}

			if testCase.uri && !doc.HasURI() {
				t.Error("expected a URI annotation")
			}

			if testCase.image && !doc.HasImageXObject() {
				t.Error("expected an image XObject")
			}

			if testCase.dest && !doc.HasInternalDest() {
				t.Error("expected an internal destination / GoTo annotation")
			}
		})
	}
}

func convertGoldenFixture(t *testing.T, file string) []byte {
	t.Helper()

	return convertGoldenFixtureWithVersion(t, file, "")
}

func convertGoldenFixtureWithVersion(t *testing.T, file, version string) []byte {
	t.Helper()

	golden := filepath.Join("..", "..", "testdata", "golden")

	dir := t.TempDir()
	if err := copyTree(golden, dir); err != nil {
		t.Fatalf("copy golden tree: %v", err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = filepath.Join(dir, file)
	obj.Load.BlockLocalFileAccess = false

	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true
	global.PageSize = "A4"
	global.Margin = settings.DefaultMargins()
	global.Background = true

	if version != "" {
		global.PdfVersion = version
	}

	var out bytes.Buffer

	req := convert.NewPDFRequest(global, []settings.PdfObject{obj}, &out, nil)
	if err := convert.Run(t.Context(), req, discardWriter{}, nil); err != nil {
		t.Fatalf("convert.Run(%s): %v", file, err)
	}

	return out.Bytes()
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func copyTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read golden dir %s: %w", src, err)
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destinationPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(destinationPath, 0o700); err != nil {
				return fmt.Errorf("mkdir %s: %w", destinationPath, err)
			}

			if err := copyTree(sourcePath, destinationPath); err != nil {
				return err
			}

			continue
		}

		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", sourcePath, err)
		}

		if err := os.WriteFile(destinationPath, content, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", destinationPath, err)
		}
	}

	return nil
}
