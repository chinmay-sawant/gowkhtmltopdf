// Package convert orchestrates the load → layout → paginate → print
// pipeline (mirrors PdfConverterPrivate): one pdf.Document for the whole
// job, one page object laid out and painted into it per input.
//
// Load HTTP failures use settings.HttpStatusError (404→exit 2, 401→exit 3).
package convert
