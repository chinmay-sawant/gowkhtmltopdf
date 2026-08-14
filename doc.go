// Package gowkhtmltopdf converts HTML documents to PDF (and to raster
// images) from Go. It is a pure-Go reimplementation of the wkhtmltopdf
// command-line tools: no cgo, no browser process, and no Qt/WebKit. The
// runtime is the Go standard library plus two allowlisted pure-Go modules —
// github.com/go-text/typesetting (OpenType shaping) and
// github.com/tdewolff/canvas (SVG-as-image rasterization).
//
// # Quick start (Preferred typed API)
//
//	import (
//		"context"
//		"os"
//		gowkhtmltopdf "gowkhtmltopdf"
//	)
//
//	out, err := os.Create("report.pdf")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer out.Close()
//
//	req := &gowkhtmltopdf.PDFRequest{
//		Global: gowkhtmltopdf.NewPdfGlobalOptions().
//			WithPageSize("A4").
//			Build(),
//		Objects: []*gowkhtmltopdf.ObjectSettings{
//			gowkhtmltopdf.NewObjectSettings().SetPage("report.html"),
//		},
//		Output: out,
//	}
//	req.EnableLocalFileAccess() // allow reading local report.html and subresources
//
//	if err := gowkhtmltopdf.RunPDF(context.Background(), req); err != nil {
//		log.Fatal(err)
//	}
//
// # Settings
//
// Global and object settings can be configured via typed builders (such as
// PdfGlobalOptions) or wkhtmltopdf-style dotted names (e.g. "size.pagesize",
// "margin.top", "orientation", "web.background", "header.left").
// The exact dotted name list mirrors the CLI surface documented by
// gowkhtmltopdf --help.
//
// # Local file access
//
// The security ACL blocks local file reads by default. To convert a local
// file, call EnableLocalFileAccess() on the request, converter, or settings:
//
//	req.EnableLocalFileAccess()
//
// Or configure the dotted keys explicitly:
//
//	c.Global().Set("enablelocalfileaccess", "true")
//	obj.Set("load.blocklocalfileaccess", "false")
//
// # In-memory HTML
//
// In-memory documents are an explicit source kind: SetBody (and the
// ConvertHTML / AddHTML helpers) mark the object as inline HTML, so no URL
// guessing is applied and an optional base URL resolves relative
// subresources:
//
//	c.AddHTML([]byte("<html>…</html>"), "https://example.com/templates/")
//
// Image mode uses the same source kind on the page object:
//
//	img := gowkhtmltopdf.NewImageConverter()
//	img.Object().SetBody([]byte("<html>…</html>"), "")
//
// An empty base leaves relative subresources unresolvable.
//
// # Thread safety
//
// A Converter (and ImageConverter) is not safe for concurrent Convert calls:
// use one converter per conversion, or guard access with a mutex. Settings
// may be freely read while idle. Each Convert run is fully context-aware;
// cancel the context to abort an in-flight conversion.
package gowkhtmltopdf
