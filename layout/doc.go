// Package layout places a css.Document and hands that placement back two
// ways. Lay paints it to an image. DisplayList returns it as a display list of
// vector operations and no image.
//
// Both entries share one placement, so boxes and geometry agree between them.
// Canvas size agrees except that Height can differ by one pixel: Lay reads its
// size back off the painted picture, which rounds, while DisplayList converts
// the placement height straight from points. Lay uses the HTML tree and the
// stylesheets from css.Apply. Boxes are CSS pixels, y down, origin at the top
// left of the image. The image is the painted layout. Pagination and PDF
// writing are not part of this package.
package layout
