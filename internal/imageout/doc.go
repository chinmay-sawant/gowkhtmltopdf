// Package imageout converts HTML to a raster image (PNG or JPEG) for the
// gowkhtmltoimage command. It runs the shared pipeline — load, html.Parse,
// css.Parse, layout.Layout — then paints the display list with stdlib
// image/draw and encodes with image/png and image/jpeg.
//
// Text uses the same embedded TrueType faces as PDF mode (pure-Go outline
// raster with greyscale anti-aliasing) and the same OpenType/presentation
// shaping path (pdf.ShapeTextFont) so advances and complex-script forms
// match layout/PDF. Layout always attaches a face when DefaultFont is
// available; imageout falls back to DefaultFont if an op has a nil face.
package imageout
