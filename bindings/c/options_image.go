package main

import (
	"github.com/chinmay-sawant/gowkhtmltopdf"
)

const (
	// smartWidthUnset leaves SmartWidth at the engine default (enabled).
	smartWidthUnset = -1
	// cropUnset marks "no crop on this axis" per the header contract.
	cropUnset = -1
)

// imageOptions mirrors GwkImageOptions with Go-native types. Zero values
// select engine defaults exactly like the documented C defaults do; crop
// axes use cropUnset instead.
type imageOptions struct {
	format      string
	baseURL     string
	allow       []string
	width       int
	height      int
	quality     int
	smartWidth  int
	transparent bool
	cropLeft    int
	cropTop     int
	cropWidth   int
	cropHeight  int
	zoom        float64
	localFiles  bool
	restricted  bool
	timeoutMS   int64
}

// buildImageDocument maps opts onto a root ImageDocument holding one inline
// HTML source. Optional fields stay zero so the engine defaults apply,
// matching the header contract for a NULL options pointer.
//
//nolint:exhaustruct // optional ImageDocument fields stay zero to inherit engine defaults.
func buildImageDocument(html []byte, opts imageOptions) *gowkhtmltopdf.ImageDocument {
	doc := &gowkhtmltopdf.ImageDocument{Source: gowkhtmltopdf.Content{HTML: html}}

	if opts.format != "" {
		doc.Format = opts.format
	}
	if opts.width != 0 {
		doc.Width = opts.width
	}
	if opts.height != 0 {
		doc.Height = opts.height
	}
	if opts.quality != 0 {
		doc.Quality = opts.quality
	}
	if opts.smartWidth != smartWidthUnset {
		enabled := opts.smartWidth != 0
		doc.SmartWidth = &enabled
	}
	if opts.transparent {
		doc.Transparent = true
	}
	if anyCropSet(opts) {
		doc.Crop = &gowkhtmltopdf.Crop{
			Left:   opts.cropLeft,
			Top:    opts.cropTop,
			Width:  opts.cropWidth,
			Height: opts.cropHeight,
		}
	}
	if opts.zoom != 0 {
		doc.Zoom = opts.zoom
	}
	if opts.baseURL != "" {
		doc.Source.Base = opts.baseURL
	}
	if opts.localFiles {
		doc.AllowLocalFiles = true
	}
	if len(opts.allow) > 0 {
		doc.Allow = opts.allow
	}
	if opts.restricted {
		policy := gowkhtmltopdf.RestrictedNetworkPolicy()
		doc.Network = &policy
	}

	return doc
}

// anyCropSet reports whether at least one crop axis overrides the unset
// marker; untouched axes keep their marker value.
func anyCropSet(opts imageOptions) bool {
	return opts.cropLeft != cropUnset || opts.cropTop != cropUnset ||
		opts.cropWidth != cropUnset || opts.cropHeight != cropUnset
}
