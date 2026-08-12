// Package gowkhtmltopdf converts HTML documents to PDF (and to raster
// images) from Go. It is a pure-Go reimplementation of the wkhtmltopdf
// command-line tools: no cgo, no browser process, and no Qt/WebKit. The
// runtime is the Go standard library plus two allowlisted pure-Go modules —
// github.com/go-text/typesetting (OpenType shaping) and
// github.com/tdewolff/canvas (SVG-as-image rasterization).
//
// # Quick start
//
//	import gowkhtmltopdf "gowkhtmltopdf"
//
//	c := gowkhtmltopdf.NewConverter()
//	c.Global().Set("size.pagesize", "A4")
//	c.Global().Set("orientation", "portrait")
//	c.Global().Set("enablelocalfileaccess", "true")
//
//	obj := gowkhtmltopdf.NewObjectSettings().SetPage("report.html")
//	obj.Set("load.blocklocalfileaccess", "false") // pair with the global flag
//	c.AddObject(obj)
//
//	if err := c.Convert(context.Background()); err != nil {
//		log.Fatal(err)
//	}
//	os.WriteFile("report.pdf", c.Output(), 0o644)
//
// # Settings
//
// Global and object settings are set and read by wkhtmltopdf-style dotted
// names, e.g. "size.pagesize", "margin.top", "orientation", "web.background",
// "header.left" or "load.jsdelay". Set returns an error for unknown names;
// Get returns (value, false) for names that have no scalar representation.
// The exact name list mirrors the CLI surface documented by
// gowkhtmltopdf --help.
//
// # Local file access
//
// The security ACL blocks local file reads by default. To convert a local
// file, enable access explicitly - both the global flag and the object-level
// block must be toggled, mirroring the CLI's --enable-local-file-access and
// --block-local-file-access pair:
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
