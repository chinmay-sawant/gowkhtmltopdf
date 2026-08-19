//nolint:testpackage,err113 // preflight re-layout uses shared convert helpers
package convert

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestFontPreflightRelayoutFallsBack(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	restore := layout.SetEmbedPreflightForTest(func(fnt *pdf.Font, used []rune) error {
		if fnt == faces.Serif || fnt == faces.SerifBold || fnt == faces.SerifItalic || fnt == faces.SerifBoldItalic {
			return errors.New("synthetic embed failure")
		}

		return pdf.PreflightEmbed(fnt, used)
	})
	t.Cleanup(restore)

	html := `<html><head><style>
body { font-family: serif, sans-serif; font-size: 18px; width: 140px; }
</style></head><body><p>MMMMMMMMMMMMMMMMMMMM wrap-sensitive preflight</p></body></html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("expected PDF; log=%q", log.String())
	}

	if !strings.Contains(log.String(), "embed preflight") {
		t.Fatalf("expected preflight re-layout warning; log=%q", log.String())
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Fatal("expected embedded fallback face")
	}
}

func TestPreflightPassesBundledFaces(t *testing.T) {
	t.Parallel()

	html := `<html><body style="font-family: sans-serif; font-size: 14pt;"><p>OK</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("expected PDF")
	}
}
