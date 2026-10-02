package layout

import (
	"strings"
	"unicode/utf8"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// controlBulletRune masks each password rune. Chrome paints U+2022 BULLET.
const controlBulletRune = '\u2022'

// isTextInput reports whether node is an input whose value paints as one line
// of text. A missing type falls back to text, matching the HTML parser.
// Checkbox, radio, range, button, and the other native widgets keep their own
// faces; unknown types paint nothing.
func isTextInput(node *html.Node) bool {
	if node == nil || node.Name != htmlInput {
		return false
	}

	switch strings.ToLower(node.Attribute("type")) {
	case "", "text", "password", "search", "email", "tel", "url", "number":
		return true
	}

	return false
}

// controlTextValue returns the text a text-like input paints: its value, or
// the placeholder when the value is empty. A password masks its value with one
// bullet per rune but keeps the placeholder readable.
func controlTextValue(node *html.Node) string {
	value := node.Attribute("value")
	if value != "" && strings.EqualFold(node.Attribute("type"), "password") {
		value = strings.Repeat(string(controlBulletRune), utf8.RuneCountInString(value))
	}

	if value != "" {
		return value
	}

	return node.Attribute("placeholder")
}

// paintTextInputWidget paints a text-like input's value inside its border box
// at the UA form-control font size. The text starts at the content-box left
// edge, centers vertically in the content box, and truncates so it cannot
// spill past the content-box right edge.
func (e *engine) paintTextInputWidget(
	node *html.Node, style ResolvedStyle, leftX, topY, width, height float64,
) {
	text := controlTextValue(node)
	if text == "" {
		return
	}

	face := e.faceFor(&style)
	if face == nil {
		return
	}

	size := e.scalePt(formControlFontSizePt)
	if size <= 0 {
		return
	}

	contentX, contentW := e.contentBox(leftX, width, &style)
	contentY := topY + e.scalePt(style.BorderTop.Width) + e.scalePt(style.PaddingTop)
	contentH := height - e.scalePt(style.BorderTop.Width+style.BorderBottom.Width) -
		e.scalePt(style.PaddingTop+style.PaddingBottom)

	if contentH < 0 {
		// An auto-height input has no content height; keep the value centered
		// on the content-box line instead of dropping it.
		contentH = 0
	}

	if contentW <= 0 {
		return
	}

	textStyle := style
	textStyle.FontSize = formControlFontSizePt

	text = e.truncateForEllipsis(text, contentW, &textStyle)
	if text == "" {
		return
	}

	textW := e.measureTextFace(text, &textStyle)
	ascent := e.fontAscentFace(face, size)
	descent := e.fontDescentFace(face, size)

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpText, X: contentX,
		Y: contentY + (contentH+ascent-descent)/two, W: textW, H: ascent + descent,
		Text: text, Font: face, Size: size, InkDescent: descent,
		R: style.Color[0], G: style.Color[1], B: style.Color[2],
		Bold: style.FontWeight >= fontWeightBoldValue,
	})
}
