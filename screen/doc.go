// Package screen renders one HTML document to a PNG and the element
// rectangles that match that PNG.
//
// A host uses the rectangles to decide which element a click landed on.
// Rectangle coordinates are CSS pixels, y down, origin at the top left of
// the PNG, at zoom 1 and with no padding. screen does not open a window,
// does not read the keyboard, and does not call user functions.
//
// Document and ImageDocument stay the PDF and image writers. They live in
// the root package. This package is the interactive-screen entry.
package screen
