package main

import (
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf"
)

// pdfOptions mirrors GwkPdfOptions with Go-native types. Zero values select
// engine defaults exactly like the documented C defaults do.
type pdfOptions struct {
	pageSize     string
	orientation  string
	title        string
	pdfVersion   string
	pdfProfile   string
	baseURL      string
	allow        []string
	widthMM      float64
	heightMM     float64
	marginTop    float64
	marginRight  float64
	marginBottom float64
	marginLeft   float64
	copies       int
	grayscale    bool
	localFiles   bool
	restricted   bool
	timeoutMS    int64
}

// validatePDFRange enforces the option ranges the header classifies as
// INVALID_ARG before any engine work starts. Copies 0 selects the engine
// default and 1 through MaxDocumentCopies is accepted. The string result is
// the diagnostic message; false means the request was rejected.
func validatePDFRange(opts pdfOptions) (string, bool) {
	if opts.copies == 0 || (opts.copies >= 1 && opts.copies <= gowkhtmltopdf.MaxDocumentCopies) {
		return "", true
	}

	return fmt.Sprintf("copies %d outside supported range 1 through %d",
		opts.copies, gowkhtmltopdf.MaxDocumentCopies), false
}

// buildPDFDocument maps opts onto a root Document holding one inline HTML
// page. Optional fields stay zero so the engine defaults apply, matching the
// header contract for a NULL options pointer.
//
//nolint:exhaustruct // optional Document fields stay zero to inherit engine defaults.
func buildPDFDocument(html []byte, opts pdfOptions) *gowkhtmltopdf.Document {
	doc := &gowkhtmltopdf.Document{
		Pages: []gowkhtmltopdf.Page{{Source: gowkhtmltopdf.Content{HTML: html}}},
	}

	if opts.pageSize != "" {
		doc.PageSize = opts.pageSize
	}
	if opts.widthMM != 0 || opts.heightMM != 0 {
		doc.WidthMM = opts.widthMM
		doc.HeightMM = opts.heightMM
	}
	if opts.orientation != "" {
		doc.Orientation = opts.orientation
	}
	if anyMarginSet(opts) {
		doc.Margin = gowkhtmltopdf.Margin{
			Top:    opts.marginTop,
			Right:  opts.marginRight,
			Bottom: opts.marginBottom,
			Left:   opts.marginLeft,
		}
	}
	if opts.title != "" {
		doc.Title = opts.title
	}
	if opts.pdfVersion != "" {
		doc.PDFVersion = opts.pdfVersion
	}
	if opts.pdfProfile != "" {
		doc.PDFProfile = opts.pdfProfile
	}
	if opts.copies != 0 {
		doc.Copies = opts.copies
	}
	if opts.grayscale {
		doc.Grayscale = true
	}
	if opts.localFiles {
		doc.AllowLocalFiles = true
	}
	if len(opts.allow) > 0 {
		doc.Allow = opts.allow
	}
	if opts.baseURL != "" {
		doc.Pages[0].Source.Base = opts.baseURL
	}
	if opts.restricted {
		policy := gowkhtmltopdf.RestrictedNetworkPolicy()
		doc.Network = &policy
	}

	return doc
}

// anyMarginSet reports whether at least one margin field overrides zero.
func anyMarginSet(opts pdfOptions) bool {
	return opts.marginTop != 0 || opts.marginRight != 0 ||
		opts.marginBottom != 0 || opts.marginLeft != 0
}
