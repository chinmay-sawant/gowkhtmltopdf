// Package layout places a css.Document and paints that placement to an image.
//
// Lay uses the HTML tree and the stylesheets from css.Apply. Boxes are CSS
// pixels, y down, origin at the top left of the image. The image is the
// painted layout. Pagination and PDF writing are not part of this package.
package layout
