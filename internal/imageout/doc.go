// Package imageout converts HTML to a raster image (PNG or JPEG) for the
// gowkhtmltoimage command (Phase 07). It runs the shared pipeline - load,
// html.Parse, css.Parse, layout.Layout - then rasterizes the display list
// with stdlib image/draw and encodes with image/png and image/jpeg.
//
// Text rendering uses the embedded bitmap font in font.go: the stdlib has no
// text rasterizer, so layout's text ops are drawn with a fixed 5x8 pixel
// public-domain font scaled linearly from 12 pt. See font.go for details.
package imageout
