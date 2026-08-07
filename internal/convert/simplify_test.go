package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// chromeHTML is a wiki-like page with chrome left visible (no author
// display:none). Used to prove --simplify-dom is opt-in.
const chromeHTML = `<!DOCTYPE html>
<html><body>
  <a class="mw-jump-link" href="#content">Jump to content UNIQUEJUMP</a>
  <nav class="site-nav"><ul><li>Random article UNIQUENAV</li></ul></nav>
  <div id="mw-navigation">Wiki tools UNIQUEMWNAV</div>
  <aside role="complementary">Appearance UNIQUEASIDE</aside>
  <footer>Site footer UNIQUEFOOTER</footer>
  <main id="content">
    <h1>Article Title UNIQUE TITLE</h1>
    <p>Body paragraph UNIQUEBODY for the print path.</p>
  </main>
</body></html>`

func TestSimplifyDOMOffKeepsChrome(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, chromeHTML, filepath.Join(t.TempDir(), "off.pdf"))
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	needles := []string{
		"UNIQUENAV", "UNIQUEJUMP", "UNIQUEMWNAV", "UNIQUEASIDE", "UNIQUEFOOTER", "UNIQUEBODY",
	}

	for _, needle := range needles {
		if !bytes.Contains(data, []byte(needle)) {
			t.Errorf("flag off: PDF missing %q (chrome/body should remain)", needle)
		}
	}
}

func TestSimplifyDOMOnHidesChrome(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, chromeHTML, filepath.Join(t.TempDir(), "on.pdf"))
	cmd.Global.UseCompression = false
	cmd.Global.Web.SimplifyDOM = true
	cmd.Objects[0].Web.SimplifyDOM = true

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("UNIQUEBODY")) {
		t.Error("flag on: body text missing from PDF")
	}

	if !bytes.Contains(data, []byte("UNIQUE TITLE")) && !bytes.Contains(data, []byte("Article Title")) {
		t.Error("flag on: title missing from PDF")
	}
	// Landmarks-only default: nav/aside/footer gone; MediaWiki IDs stay visible.
	for _, needle := range []string{"UNIQUENAV", "UNIQUEASIDE", "UNIQUEFOOTER"} {
		if bytes.Contains(data, []byte(needle)) {
			t.Errorf("flag on: landmark chrome %q should be display:none", needle)
		}
	}

	for _, needle := range []string{"UNIQUEJUMP", "UNIQUEMWNAV"} {
		if !bytes.Contains(data, []byte(needle)) {
			t.Errorf("landmarks-only: MediaWiki chrome %q should remain without profile", needle)
		}
	}
}

func TestSimplifyDOMMediaWikiProfile(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, chromeHTML, filepath.Join(t.TempDir(), "mw.pdf"))
	cmd.Global.UseCompression = false
	cmd.Global.Web.SimplifyDOM = true
	cmd.Global.Web.SimplifyDOMProfile = profileMediaWiki
	cmd.Objects[0].Web.SimplifyDOM = true
	cmd.Objects[0].Web.SimplifyDOMProfile = profileMediaWiki

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("UNIQUEBODY")) {
		t.Error("body missing")
	}

	for _, needle := range []string{"UNIQUENAV", "UNIQUEJUMP", "UNIQUEMWNAV", "UNIQUEASIDE", "UNIQUEFOOTER"} {
		if bytes.Contains(data, []byte(needle)) {
			t.Errorf("mediawiki profile: chrome %q should be hidden", needle)
		}
	}
}

func TestSimplifyChromeCSSParsesAndMatches(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(SimplifyChromeCSS)
	if err != nil || sheet == nil {
		t.Fatalf("parse SimplifyChromeCSS: %v", err)
	}

	root, err := html.Parse(chromeHTML)
	if err != nil {
		t.Fatal(err)
	}

	font, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	off := layoutMust(t, root, nil, font)
	on := layoutMust(t, root, []*css.Stylesheet{sheet}, font)
	offText := layoutText(off)
	onText := layoutText(on)

	if !strings.Contains(offText, "UNIQUENAV") {
		t.Fatal("layout without simplify should include nav text")
	}

	if strings.Contains(onText, "UNIQUENAV") || strings.Contains(onText, "UNIQUEFOOTER") {
		t.Fatalf("layout with simplify should hide chrome; got %q", onText)
	}

	if !strings.Contains(onText, "UNIQUEBODY") {
		t.Fatalf("layout with simplify should keep body; got %q", onText)
	}
	// Landmarks-only sheet does not hide MediaWiki IDs.
	if !strings.Contains(onText, "UNIQUEMWNAV") {
		t.Fatalf("landmarks-only should keep #mw-navigation; got %q", onText)
	}
}

func TestAppendSimplifySheetNoopWhenOff(t *testing.T) {
	t.Parallel()

	got := AppendSimplifySheet(nil, false, "")
	if got != nil {
		t.Fatalf("want nil, got %d sheets", len(got))
	}

	got = AppendSimplifySheet(nil, true, "")
	if len(got) != 1 {
		t.Fatalf("want 1 sheet, got %d", len(got))
	}

	got = AppendSimplifySheet(nil, true, profileMediaWiki)
	if len(got) != 2 {
		t.Fatalf("want 2 sheets (landmarks+mw), got %d", len(got))
	}
}

func TestSimplifyDOMEnabled(t *testing.T) {
	t.Parallel()

	if SimplifyDOMEnabled(settings.Web{}, settings.Web{}) { //nolint:exhaustruct // intentional zero-value fields
		t.Fatal("default must be off")
	}

	if !SimplifyDOMEnabled(settings.Web{SimplifyDOM: true}, settings.Web{}) { //nolint:exhaustruct,lll // intentional zero-value fields
		t.Fatal("global on")
	}

	if !SimplifyDOMEnabled(settings.Web{}, settings.Web{ //nolint:exhaustruct // intentional zero-value fields
		SimplifyDOM: true,
	}) {
		t.Fatal("object on")
	}
}

func TestSimplifyDOMProfile(t *testing.T) {
	t.Parallel()

	if SimplifyDOMProfile(settings.Web{}, settings.Web{}) != "" { //nolint:exhaustruct // intentional zero-value fields
		t.Fatal("default profile empty")
	}

	if SimplifyDOMProfile(settings.Web{ //nolint:exhaustruct // intentional zero-value fields
		SimplifyDOMProfile: profileMediaWiki,
	}, settings.Web{}, //nolint:exhaustruct // intentional zero-value fields
	) != profileMediaWiki {
		t.Fatal("global mediawiki")
	}

	if SimplifyDOMProfile(settings.Web{}, settings.Web{ //nolint:exhaustruct // intentional zero-value fields
		SimplifyDOMProfile: "wiki",
	}) != profileMediaWiki {
		t.Fatal("object wiki alias")
	}
}

func layoutMust(t *testing.T, root *html.Node, sheets []*css.Stylesheet, font *pdf.Font) *layout.Result {
	t.Helper()

	res, err := layout.Layout(root, layout.Options{ //nolint:exhaustruct // intentional zero-value fields
		Width:  500,
		Height: 700,
		Font:   font,
		Sheets: sheets,
		Media:  mediaPrint,
	})
	if err != nil {
		t.Fatal(err)
	}

	return res
}

func layoutText(res *layout.Result) string {
	var buf strings.Builder

	for _, op := range res.Ops {
		if op.Kind == layout.OpText {
			buf.WriteString(op.Text)
			buf.WriteByte(' ')
		}
	}

	return buf.String()
}

// TestSubresourceFailureIsolation: missing CSS + broken image must not
// abort conversion; body text still appears (phase 21.5).
func TestSubresourceFailureIsolation(t *testing.T) {
	t.Parallel()

	htmlSrc := `<html><head>
<link rel="stylesheet" href="missing-no-such.css">
</head><body>
<p>ISOLATIONBODY</p>
<img src="missing-no-such.png" width="40" height="40">
</body></html>`
	cmd, _ := newCommand(t, htmlSrc, filepath.Join(t.TempDir(), "iso.pdf"))
	cmd.Global.UseCompression = false

	var log bytes.Buffer

	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF should succeed with missing subresources: %v", err)
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(data, []byte("ISOLATIONBODY")) {
		t.Error("body text missing after CSS/image failures")
	}

	if !strings.Contains(log.String(), "skipping <link") {
		t.Errorf("expected warning about missing stylesheet; log=%q", log.String())
	}
}
