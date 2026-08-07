package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// Print CSS: a.external.text::after { content: ' (' attr(href) ')'; }
// Must emit the real href, never the literal tokens "attr(href)".
func TestExternalLinkAttrHrefContent(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
.mw-parser-output a.external.text::after {
  content: ' (' attr(href) ')';
}
.mw-parser-output a.external.text[href^="//"]::after {
  content: ' (https:' attr(href) ')';
}
a { text-decoration: none; color: inherit; }
`)
	src := `<html><body><div class="mw-parser-output">
<p>See <a class="external text" href="https://example.com/foo">Example</a> and
<a class="external text" href="//upload.example/bar">Proto</a>.</p>
</div></body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 500, Height: 400, Sheets: []*css.Stylesheet{s}, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var got string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			got += op.Text
		}
	}

	if strings.Contains(got, "attr(href)") || strings.Contains(got, "attr(") {
		t.Fatalf("literal attr() leaked into paint: %q", got)
	}

	if !strings.Contains(got, "https://example.com/foo") {
		t.Fatalf("missing resolved https href in %q", got)
	}

	if !strings.Contains(got, "https://upload.example/bar") {
		t.Fatalf("missing protocol-relative → https: rewrite in %q", got)
	}

	if !strings.Contains(got, "Example") || !strings.Contains(got, "Proto") {
		t.Fatalf("missing link labels in %q", got)
	}
}

func TestParseContentValueAttr(t *testing.T) {
	n := &html.Node{Type: html.ElementNode, Name: "a", Attrs: map[string]string{"href": "https://ex.test/x"}}

	got := parseContentValue(`' (' attr(href) ')'`, n)
	if got != " (https://ex.test/x)" {
		t.Fatalf("got %q", got)
	}

	got2 := parseContentValue(`' (https:' attr(href) ')'`, n)
	// attr returns full href including https, so this doubles scheme — matches CSS
	// when href is protocol-relative; for absolute hrefs the [href^='//'] rule
	// does not apply. Still must not emit literal attr(.
	if strings.Contains(got2, "attr") {
		t.Fatalf("literal attr in %q", got2)
	}
}
