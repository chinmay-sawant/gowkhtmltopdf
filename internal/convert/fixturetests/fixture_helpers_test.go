package fixturetests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	convert "github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// runPDF executes the public conversion boundary used by the fixture suite.
func runPDF(t *testing.T, req *convert.Request) []byte {
	t.Helper()

	var buf bytes.Buffer
	req.Output = &buf

	if err := convert.Run(t.Context(), req, io.Discard, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	return buf.Bytes()
}

// goldenDir resolves the fixture corpus from the fixture-test package.
func goldenDir() string {
	return filepath.Join("..", "..", "..", "testdata", "golden")
}

// fixtureIDPrefix returns "fixture-NN" from a body fixture file name.
func fixtureIDPrefix(file string) string {
	base := strings.TrimSuffix(filepath.Base(file), ".html")

	parts := strings.SplitN(base, "-", 3)
	if len(parts) < 2 || parts[0] != "fixture" {
		return ""
	}

	for _, c := range parts[1] {
		if c < '0' || c > '9' {
			return ""
		}
	}

	return parts[0] + "-" + parts[1]
}

func isHFCompanionHTML(name string) bool {
	return strings.HasSuffix(name, "-header.html") || strings.HasSuffix(name, "-footer.html")
}

// attachHFCompanions keeps header and footer fixture resources in the same
// conversion contract as the main convert package tests.
func attachHFCompanions(req *convert.Request, dir, file string) {
	prefix := fixtureIDPrefix(file)
	if prefix == "" || isHFCompanionHTML(file) {
		return
	}

	header := filepath.Join(dir, prefix+"-header.html")
	if _, err := os.Stat(header); err == nil {
		req.Global.Header.HTMLURL = header
		req.Global.Margin.Top = -1
	}

	footer := filepath.Join(dir, prefix+"-footer.html")
	if _, err := os.Stat(footer); err == nil {
		req.Global.Footer.HTMLURL = footer
		req.Global.Margin.Bottom = -1
	}
}

// requestForFixture builds the same public Request used by make samples:
// A4, default margins, backgrounds, local file access, and explicit fixture
// font paths.
func requestForFixture(t *testing.T, file string) *convert.Request {
	t.Helper()
	dir := t.TempDir()

	if err := copyGoldenTree(goldenDir(), dir); err != nil {
		t.Fatalf("copy golden directory: %v", err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = filepath.Join(dir, file)
	obj.Load.BlockLocalFileAccess = false
	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true
	global.PageSize = "A4"
	global.Margin = settings.DefaultMargins()
	global.Background = true

	fontDirs := []string{}
	if _, err := os.Stat("/usr/share/fonts/truetype/droid"); err == nil {
		fontDirs = append(fontDirs, "/usr/share/fonts/truetype/droid")
	}

	testFonts := filepath.Join("..", "..", "..", "testdata", "fonts")
	if _, err := os.Stat(testFonts); err == nil {
		fontDirs = append(fontDirs, testFonts)
	}

	global.FontPaths = fontDirs
	req := convert.NewPDFRequest(global, []settings.PdfObject{obj}, nil, nil)
	attachHFCompanions(req, dir, file)

	return req
}

func copyGoldenTree(src, dst string) error {
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

			if err := copyGoldenTree(sourcePath, destinationPath); err != nil {
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
