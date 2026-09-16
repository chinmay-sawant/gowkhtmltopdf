package layout

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Phase-13 probe: the mobile menu glyph \ue9bd with no icon face available.
//
// The literal "()" measured in the 19-page learncpp no-images render was not a
// glyph fallback. It was the print rule
// ".cryout p a::after { content: " (" attr(href) ")" }"
// matching the href-less #nav-toggle and menu-search anchors after the
// pre-Phase-11 parser nested the fixed header inside an unclosed <p>: an empty
// attr(href) paints "()" at 80% font size. With the HTML5 implicit </p>, those
// anchors sit outside <p> and the rule no longer matches. The missing icon
// glyph itself keeps its rune in the display list (the PDF writer folds it to
// "?"); the engine never substitutes "()" for it.

const missingIconCSS = `
@media print {
	.cryout p a::after { content: " (" attr(href) ")"; font-size: 80% }
}
.icon-menu::before { content: "\e9bd"; font-family: "IconMetaNoSuchFace", sans-serif }
`

// TestLearnCppMissingIconFontPaintsNoEmptyParens pins that a PUA glyph with no
// face that covers it is not replaced by a literal "()" placeholder, and that
// the print link-suffix rule does not fire for href-less header anchors.
func TestLearnCppMissingIconFontPaintsNoEmptyParens(t *testing.T) {
	t.Parallel()

	const contentH = 841.9 - 2*28.35

	root := mustParse(t, `<html><body><div class=cryout>
		<p>lead text<div id=branding><a id=nav-toggle><i class=icon-menu></i></a></div>
	</div></body></html>`)

	s := sheet(t, missingIconCSS)

	res, err := Layout(root, Options{
		Width: 538.5, Height: contentH, Sheets: []*css.Stylesheet{s},
		Background: true, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	iconFound := scanMissingIconOps(t, res)
	if !iconFound {
		t.Fatal("icon content missing from the display list")
	}

	doc := pdf.NewDocument()

	if err := Paint(doc, res, PaintOptions{
		PageWidth: 595.4, PageHeight: 841.9,
		MarginTop: 28.35, MarginBottom: 28.35, MarginLeft: 28.35, MarginRight: 28.35,
	}); err != nil {
		t.Fatalf("Paint: %v", err)
	}

	var out bytes.Buffer

	if err := doc.Write(&out); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if bytes.Contains(out.Bytes(), []byte(`(\(\))`)) {
		t.Fatal(`PDF contains a literal "()" text run`)
	}
}

// scanMissingIconOps fails on literal empty-paren runs and reports whether the
// PUA icon rune survives in the display list.
func scanMissingIconOps(t *testing.T, res *Result) bool {
	t.Helper()

	iconFound := false

	for _, paintedOp := range res.Ops {
		if paintedOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintedOp.Text, "()") || paintedOp.Text == "(" || paintedOp.Text == ")" {
			t.Fatalf("literal empty-paren run painted: %q", paintedOp.Text)
		}

		if strings.ContainsRune(paintedOp.Text, '\ue9bd') {
			iconFound = true
			// The fixture declares no icon face, so the run must be carried
			// by a face that lacks the glyph; the writer folds it to "?".
			if paintedOp.Font != nil && paintedOp.Font.GlyphID('\ue9bd') != 0 {
				t.Fatalf("icon run unexpectedly resolved to a face with the glyph: %s", paintedOp.Font.PostScriptName)
			}
		}
	}

	return iconFound
}
