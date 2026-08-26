//go:build !cgo

// Stub entry points compiled when cgo is disabled. They keep the package
// buildable and non-empty under CGO_ENABLED=0, where the shared library
// surface cannot exist; every entry point reports statusInternal with the
// documented CGO-required diagnostic.
package main

import (
	"context"
)

// runPDFWithContext mirrors the cgo build mode entry point signature.
func runPDFWithContext(_ context.Context, _ []byte, _ pdfOptions) (int32, []byte, string) {
	return statusInternal, nil, stubRequiredCGOError
}

// runImageWithContext mirrors the cgo build mode entry point signature.
func runImageWithContext(_ context.Context, _ []byte, _ imageOptions) (int32, []byte, string) {
	return statusInternal, nil, stubRequiredCGOError
}
