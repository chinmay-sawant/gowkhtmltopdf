// Package gowkhtmltopdf converts authored HTML documents to PDF and raster
// images through a pure-Go, no-cgo engine. It does not start a browser,
// WebKit, Qt, or a native converter process.
//
// # 0.2.5 library
//
// The 0.2.5 library models a PDF as a Document and an image render as an
// ImageDocument. Content identifies exactly one source kind: in-memory HTML,
// a local file, or an HTTP(S) URL. Document.WritePDF and Document.PDF validate
// and render a PDF; ImageDocument.WriteImage and ImageDocument.Image do the
// corresponding image work.
//
// The 0.2.4 migration is a hard break: the old root symbols are removed. See
// documentation/MIGRATION-0.2.4.md when moving an embedder from 0.2.3.
//
// # Quick start (0.2.5)
//
//	import (
//		"context"
//		"os"
//		gowkhtmltopdf "github.com/chinmay-sawant/gowkhtmltopdf"
//	)
//
//	func main() {
//		out, err := os.Create("report.pdf")
//		if err != nil {
//			panic(err)
//		}
//		defer out.Close()
//
//		doc := gowkhtmltopdf.Document{
//			Pages: []gowkhtmltopdf.Page{{
//				Source: gowkhtmltopdf.Content{
//					HTML: []byte("<h1>Report</h1>"),
//				},
//			}},
//			PageSize: "A4",
//		}
//		if err := doc.WritePDF(context.Background(), out); err != nil {
//			panic(err)
//		}
//	}
//
// # Content and security
//
// A Content value must have exactly one of HTML, File, or URL set. Base is
// valid with HTML and resolves relative subresources. Local file reads are
// disabled by default; Document.AllowLocalFiles enables broad local reads and
// Document.Allow adds ACL path prefixes. NetworkPolicy controls remote fetches.
//
// # Validation and ownership
//
// Validate, WritePDF, PDF, WriteImage, and Image return errors for invalid
// user configuration before the engine runs. The write boundary owns a copy
// of caller-provided HTML and option slices. Use errors.Is with package
// sentinels. A Document with a TOC but no cover or body page is invalid.
//
// # Versioning
//
// LibraryVersion is the wkhtmltopdf compatibility identifier and is distinct
// from the project release in VERSION. PDFVersion selects a file version; it
// does not by itself claim PDF/A or PDF/UA conformance.
package gowkhtmltopdf
