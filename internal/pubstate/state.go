// Package pubstate lets the public html, css, and layout packages share
// one parsed tree without putting internal types in their exported signatures.
// Each public package registers a reader from init. The other packages call
// the reader. There is no map, so a document can be garbage-collected.
package pubstate

import (
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Styled is the cascade result the layout package reads.
type Styled struct {
	Root     *html.Node
	Sheets   []*css.Stylesheet
	Registry *pdf.Registry
	Media    string
	WidthPx  int
	HeightPx int
}

// The readers are package state because html, css, and layout cannot share
// unexported fields without an import cycle, and an exported field would
// leak the internal node type. Parse and Apply replace them with the same
// functions. After that they are only read.
//
//nolint:gochecknoglobals // registration point for the public engine packages
var (
	readRoot   func(any) *html.Node
	readStyled func(any) (Styled, bool)
)

// RegisterRoot stores the reader for a public HTML document.
// The html package calls it from init.
func RegisterRoot(fn func(any) *html.Node) {
	readRoot = fn
}

// Root returns the parsed tree for a public HTML document.
// A nil document or a missing reader returns nil.
func Root(doc any) *html.Node {
	if doc == nil || readRoot == nil {
		return nil
	}

	return readRoot(doc)
}

// RegisterStyled stores the reader for a public CSS document.
// The css package calls it from init.
func RegisterStyled(fn func(any) (Styled, bool)) {
	readStyled = fn
}

// StyledOf returns the cascade result for a public CSS document.
func StyledOf(doc any) (Styled, bool) {
	if doc == nil || readStyled == nil {
		return Styled{}, false //nolint:exhaustruct // no styled document is registered
	}

	return readStyled(doc)
}
