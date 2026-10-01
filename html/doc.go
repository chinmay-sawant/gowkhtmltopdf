// Package html parses one HTML document for the CSS and layout packages.
//
// Parse keeps the engine's own tree. css.Apply reads that tree, and
// layout.Lay places it. markup.Parse returns a detached copy for inspection
// and is not the document those packages accept.
//
// This package does not write a PDF.
package html
