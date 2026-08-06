// Package settings implements the wkhtmltopdf-compatible settings model and
// dotted-name Set surface used by the CLI and library API.
//
// # Policy A (settings honesty)
//
// Only options with an engine consumer (convert, load, imageout, or layout)
// get typed fields and dedicated setters. Inert wkhtml keys (dpi, javascript,
// plugins, log-level, js-delay, user-style-sheet, produce-forms, …) may be
// accepted into Ignored map[string]string for script compatibility, but must
// not reappear as typed stubs without a convert/load consumer.
//
// Dual storage is collapsed where possible: Grayscale is the sole color bit
// convert reads; page geometry is PageSize name + Size width/height (mm);
// DumpOutline / DumpDefaultTOCXSL live on PdfGlobal.
//
// See plans/reviews/ponytail/ponytail-ultra-2026-08-06.md Phase 1.
package settings
