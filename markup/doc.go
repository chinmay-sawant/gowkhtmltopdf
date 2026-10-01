// Package markup exposes the HTML tokenizer as an owned node tree.
//
// The parser underneath is the same one the PDF engine uses. This package
// returns a copy, so a caller cannot see or change the internal tree.
// Script text is kept as text. Nothing in this package runs it.
package markup
