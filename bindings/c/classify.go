package main

import (
	"context"
	"errors"

	"github.com/chinmay-sawant/gowkhtmltopdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
)

// classifyError maps an engine failure onto the ABI status table documented
// in include/gowkhtmltopdf.h. It stays free of C types so the PDF and image
// entry points share it in both the cgo and pure-Go stub builds. A done
// context always wins: when cancellation raced with another failure, callers
// observe TIMEOUT instead of a generic render error.
func classifyError(err error, ctx context.Context) int32 {
	if err == nil {
		return statusOK
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded), ctxDone(ctx):
		return statusTimeout
	case errors.Is(err, load.ErrAccessDenied),
		errors.Is(err, load.ErrNetworkPolicy),
		errors.Is(err, load.ErrInvalidProxy):
		return statusLoadDenied
	case errors.Is(err, convert.ErrInvalidCopies):
		return statusResourceLimit
	case invalidArgument(err):
		return statusInvalidArg
	default:
		return statusRenderError
	}
}

// ctxDone reports whether ctx already carries a deadline or cancellation.
func ctxDone(ctx context.Context) bool {
	return ctx != nil && ctx.Err() != nil
}

// invalidArgument reports whether err originates from caller-supplied
// document or option validation rather than the rendering pipeline.
func invalidArgument(err error) bool {
	for _, sentinel := range []error{
		gowkhtmltopdf.ErrNoPageObjects,
		gowkhtmltopdf.ErrEmptyHTML,
		gowkhtmltopdf.ErrInvalidContent,
		gowkhtmltopdf.ErrInvalidPageSize,
		gowkhtmltopdf.ErrInvalidOrientation,
		gowkhtmltopdf.ErrInvalidPDFVersion,
		gowkhtmltopdf.ErrInvalidPDFProfile,
		gowkhtmltopdf.ErrInvalidImageFormat,
		gowkhtmltopdf.ErrNilContext,
		gowkhtmltopdf.ErrMissingPDFOutput,
	} {
		if errors.Is(err, sentinel) {
			return true
		}
	}

	return false
}
