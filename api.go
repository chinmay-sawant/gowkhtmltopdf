// Package gowkhtmltopdf exposes the Go-native Document and ImageDocument
// models over the pure-Go conversion engines.
package gowkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/imageout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// LibraryVersion is the upstream wkhtmltopdf settings-surface identifier. It
// is distinct from the project release in VERSION.
const LibraryVersion = "0.12.7-dev"

// Version returns the compatibility identifier banner.
func Version() string {
	return LibraryVersion + " (gowkhtmltopdf pure-go)"
}

// NetworkPolicy controls HTTP(S) document and subresource loading.
type NetworkPolicy = load.NetworkPolicy

// CompatibleNetworkPolicy preserves historical permissive URL behavior.
func CompatibleNetworkPolicy() NetworkPolicy {
	return load.CompatibleNetworkPolicy()
}

// RestrictedNetworkPolicy blocks private destinations and cross-host
// redirects unless an explicit policy exception permits them.
func RestrictedNetworkPolicy() NetworkPolicy {
	return load.RestrictedNetworkPolicy()
}

// MaxDocumentCopies is the Document.Copies upper bound. It matches the
// convert engine ceiling (maxConversionCopies).
const MaxDocumentCopies = 1000

// Static errors are stable errors.Is targets for the native Document API.
var (
	ErrNoPageObjects            = errors.New("gowkhtmltopdf: no page objects added")
	ErrNoRenderablePDFObjects   = ErrNoPageObjects
	ErrEmptyHTML                = errors.New("gowkhtmltopdf: empty HTML")
	ErrInvalidPageSize          = errors.New("gowkhtmltopdf: invalid page size")
	ErrMissingPDFOutput         = convert.ErrMissingOutput
	ErrInvalidPDFCopies         = convert.ErrInvalidCopies
	ErrMissingPDFOutlineOutput  = convert.ErrMissingOutlineOutput
	ErrMissingImageOutput       = errs.ErrMissingImageOutput
	ErrNilContext               = errs.ErrNilContext
	ErrInvalidPDFVersion        = settings.ErrInvalidPDFVersion
	ErrInvalidPDFProfile        = settings.ErrInvalidPDFProfile
	ErrProfilePDF20Unsupported  = settings.ErrProfilePDF20Unsupported
	ErrProfilePDFA1Unsupported  = settings.ErrProfilePDFA1Unsupported
	ErrConformanceRequiresPDF17 = pdf.ErrConformanceRequiresPDF17
	ErrProfileRequiresPDF17     = pdf.ErrConformanceRequiresPDF17
	ErrConformanceRequiresPDF20 = pdf.ErrConformanceRequiresPDF20
	ErrTitleRequired            = pdf.ErrTitleRequired
	ErrPDFUAMissingAlt          = pdf.ErrPDFUAMissingAlt
	errNilLogWriter             = errors.New("gowkhtmltopdf: nil log writer")
)

// convertHooks translates engine log/progress streams into native Document
// callbacks without exposing internal request or settings types.
type convertHooks struct {
	OnInfo, OnWarn, OnError func(string)
	OnPhase                 func(string)
	OnProgress              func(int)
}

func (h convertHooks) lineLog() *lineLog {
	return &lineLog{
		buf:     bytes.Buffer{},
		onInfo:  h.OnInfo,
		onWarn:  h.OnWarn,
		onError: h.OnError,
	}
}

func (h convertHooks) progress() func(string, int) {
	if h.OnPhase == nil && h.OnProgress == nil {
		return nil
	}

	return func(phase string, percent int) {
		if h.OnPhase != nil {
			h.OnPhase(phase)
		}

		if h.OnProgress != nil {
			h.OnProgress(percent)
		}
	}
}

func (h convertHooks) executePDFTo(ctx context.Context, req *convert.Request) error {
	if ctx == nil {
		return reportPreflight(h.OnError, ErrNilContext)
	}

	if err := convert.Run(ctx, req, h.lineLog(), h.progress()); err != nil {
		return reportPreflight(h.OnError, fmt.Errorf("convert: %w", err))
	}

	return nil
}

func (h convertHooks) executeImageTo(ctx context.Context, req *imageout.Request) error {
	if ctx == nil {
		return reportPreflight(h.OnError, ErrNilContext)
	}

	if err := imageout.RunRequest(ctx, req, h.lineLog()); err != nil {
		return reportPreflight(h.OnError, fmt.Errorf("image convert: %w", err))
	}

	return nil
}

func reportPreflight(onError func(string), err error) error {
	if err != nil && onError != nil {
		onError(err.Error())
	}

	return err
}

type lineLog struct {
	buf     bytes.Buffer
	onInfo  func(string)
	onWarn  func(string)
	onError func(string)
}

// Write splits engine log lines and routes them to the corresponding public
// callback. The loop is deliberately kept here so partial writes are buffered
// across engine calls.
//
//nolint:cyclop,wsl // line classification has one branch per public severity.
func (w *lineLog) Write(payload []byte) (int, error) {
	if w == nil {
		return 0, errNilLogWriter
	}

	w.buf.Write(payload)

	for {
		raw := w.buf.Bytes()
		index := bytes.IndexByte(raw, '\n')

		if index < 0 {
			break
		}

		message := strings.TrimSpace(string(raw[:index]))
		w.buf.Next(index + 1)

		if message == "" {
			continue
		}

		switch line.SeverityOf(message) {
		case line.Warn:

			if w.onWarn != nil {
				w.onWarn(message)
			}
		case line.Error:

			if w.onError != nil {
				w.onError(message)
			}
		case line.Info:

			if w.onInfo != nil {
				w.onInfo(message)
			}
		}
	}

	return len(payload), nil
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}
