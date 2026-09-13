package prepare

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestLinkStylesheetMediaMatches is the white-box test for the stylesheet
// media predicate. It moved here from the convert package when the exported
// compatibility seam was deleted.
func TestLinkStylesheetMediaMatches(t *testing.T) {
	t.Parallel()

	mark := func(media string) *html.Node {
		return &html.Node{
			Type:  html.ElementNode,
			Name:  "link",
			Attrs: map[string]string{"rel": "stylesheet", "href": "x.css", "media": media},
		}
	}

	const (
		viewW, viewH = 538.0, 785.0
		mediaPrint   = "print"
	)

	if !linkStylesheet(mark(""), viewW, viewH, mediaPrint) {
		t.Error("empty media should load")
	}

	if !linkStylesheet(mark("print"), viewW, viewH, mediaPrint) {
		t.Error("print should load")
	}

	if !linkStylesheet(mark("all"), viewW, viewH, mediaPrint) {
		t.Error("all should load")
	}

	if linkStylesheet(mark("screen"), viewW, viewH, mediaPrint) {
		t.Error("screen-only must be excluded for print")
	}

	if !linkStylesheet(mark("(min-width: 500px)"), viewW, viewH, mediaPrint) {
		t.Error("min-width feature matching A4 content should load")
	}

	if linkStylesheet(mark("(min-width: 2000px)"), viewW, viewH, mediaPrint) {
		t.Error("unmatched min-width must not load")
	}

	if !linkStylesheet(mark("screen"), viewW, viewH, "screen") {
		t.Error("screen media type should accept screen stylesheets")
	}
}
