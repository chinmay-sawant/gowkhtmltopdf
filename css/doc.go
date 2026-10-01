// Package css parses stylesheets and applies them to an html.Document.
//
// Apply collects style elements from that document, plus any sheets passed
// in Options.Extra. The layout package places the resulting document.
// Linked style sheets and images are not fetched.
//
// This package does not write a PDF.
package css
