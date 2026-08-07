// Package cli implements the wkhtmltopdf-compatible CLI parser, multi-object
// grammar, and help output.
//
// ponytail: multi-object grammar intentional; do not grow flags without
// engine consumer. Phase 1 Policy A: only flags that feed convert/load/imageout
// (or Command fields used by main) are registered. Inert stub options were
// deleted rather than accepted-no-ops.
package cli
