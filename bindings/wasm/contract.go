package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf"
)

const (
	defaultMode       = "pdf"
	maxHTMLBytes      = 4 << 20
	maxOutputBytes    = 32 << 20
	maxImageDimension = 4096
)

var (
	errInvalidRequest  = errors.New("invalid request")
	errUnsupportedMode = errors.New("unsupported output mode")
	errInputTooLarge   = errors.New("HTML input exceeds browser limit")
	errOutputTooLarge  = errors.New("output exceeds browser limit")
	errImageTooLarge   = errors.New("image dimensions exceed browser limit")
)

// Request is the JSON boundary used by the browser adapter. It deliberately
// contains no native file or URL fields: browser input is inline HTML only.
type Request struct {
	HTML        string `json:"html"`
	Mode        string `json:"mode"`
	PageSize    string `json:"pageSize"`
	Orientation string `json:"orientation"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Quality     int    `json:"quality"`
}

// Result is the owned output returned by one browser conversion.
type Result struct {
	Mode   string
	MIME   string
	Bytes  []byte
	Width  int
	Height int
}

// ErrorResponse is the stable error shape exposed to JavaScript.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DecodeRequest validates JSON before any renderer or loader work starts.
func DecodeRequest(raw string) (Request, error) {
	decoder := json.NewDecoder(io.LimitReader(strings.NewReader(raw), maxHTMLBytes+1<<20))
	decoder.DisallowUnknownFields()

	var request Request
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("%w: %v", errInvalidRequest, err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Request{}, fmt.Errorf("%w: multiple JSON values", errInvalidRequest)
		}
		return Request{}, fmt.Errorf("%w: %v", errInvalidRequest, err)
	}

	if err := request.validate(); err != nil {
		return Request{}, err
	}

	return request, nil
}

func (r *Request) validate() error {
	if r == nil {
		return fmt.Errorf("%w: request is nil", errInvalidRequest)
	}

	if len([]byte(r.HTML)) == 0 {
		return fmt.Errorf("%w: HTML is empty", errInvalidRequest)
	}
	if len([]byte(r.HTML)) > maxHTMLBytes {
		return fmt.Errorf("%w: %w", errInvalidRequest, errInputTooLarge)
	}

	r.Mode = strings.ToLower(strings.TrimSpace(r.Mode))
	if r.Mode == "" {
		r.Mode = defaultMode
	}

	switch r.Mode {
	case "pdf":
		if r.Width != 0 || r.Height != 0 || r.Quality != 0 {
			return fmt.Errorf("%w: image options require png or jpeg mode", errInvalidRequest)
		}
	case "png", "jpeg":
		if r.PageSize != "" || r.Orientation != "" {
			return fmt.Errorf("%w: PDF options require pdf mode", errInvalidRequest)
		}
		if r.Width < 0 || r.Height < 0 || r.Width > maxImageDimension || r.Height > maxImageDimension {
			return fmt.Errorf("%w: %w", errInvalidRequest, errImageTooLarge)
		}
		if r.Quality < 0 || r.Quality > 100 {
			return fmt.Errorf("%w: image quality must be between 0 and 100", errInvalidRequest)
		}
	default:
		return fmt.Errorf("%w: %q", errUnsupportedMode, r.Mode)
	}

	return nil
}

// Convert runs the existing public engine APIs with browser-safe settings.
func Convert(ctx context.Context, request Request, onProgress func(string, int)) (Result, error) {
	if err := request.validate(); err != nil {
		return Result{}, err
	}
	if ctx == nil {
		return Result{}, gowkhtmltopdf.ErrNilContext
	}

	if request.Mode == "pdf" {
		document := browserPDFDocument(request, onProgress)
		output, err := document.PDF(ctx)
		if err != nil {
			return Result{}, err
		}
		if len(output) > maxOutputBytes {
			return Result{}, errOutputTooLarge
		}

		return Result{Mode: request.Mode, MIME: "application/pdf", Bytes: output}, nil
	}

	document := browserImageDocument(request, onProgress)
	output, err := document.Image(ctx)
	if err != nil {
		return Result{}, err
	}
	if len(output) > maxOutputBytes {
		return Result{}, errOutputTooLarge
	}

	width, height, err := imageDimensions(output)
	if err != nil {
		return Result{}, fmt.Errorf("decode image output: %w", err)
	}
	if width > maxImageDimension || height > maxImageDimension {
		return Result{}, errImageTooLarge
	}

	mime := "image/png"
	if request.Mode == "jpeg" {
		mime = "image/jpeg"
	}

	return Result{
		Mode:   request.Mode,
		MIME:   mime,
		Bytes:  output,
		Width:  width,
		Height: height,
	}, nil
}

func browserPDFDocument(request Request, onProgress func(string, int)) *gowkhtmltopdf.Document {
	onPhase, onValue := progressHooks(onProgress)
	document := gowkhtmltopdf.NewDocument(gowkhtmltopdf.Page{
		Source: gowkhtmltopdf.HTML([]byte(request.HTML)),
	})
	document.PageSize = request.PageSize
	document.Orientation = request.Orientation
	document.Network = &gowkhtmltopdf.NetworkPolicy{}
	document.OnPhase = onPhase
	document.OnProgress = onValue

	return document
}

func browserImageDocument(request Request, onProgress func(string, int)) *gowkhtmltopdf.ImageDocument {
	onPhase, onValue := progressHooks(onProgress)
	document := &gowkhtmltopdf.ImageDocument{
		Source:      gowkhtmltopdf.HTML([]byte(request.HTML)),
		Width:       request.Width,
		Height:      request.Height,
		Format:      request.Mode,
		Quality:     request.Quality,
		Transparent: true,
		Network:     &gowkhtmltopdf.NetworkPolicy{},
		OnPhase:     onPhase,
		OnProgress:  onValue,
	}

	return document
}

func progressHooks(onProgress func(string, int)) (func(string), func(int)) {
	if onProgress == nil {
		return nil, nil
	}

	phase := ""
	onPhase := func(next string) {
		phase = next
		onProgress(phase, 0)
	}
	onValue := func(percent int) { onProgress(phase, percent) }

	return onPhase, onValue
}

func imageDimensions(output []byte) (int, int, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(output))
	if err != nil {
		return 0, 0, err
	}

	return config.Width, config.Height, nil
}

func errorResponse(err error) ErrorResponse {
	if err == nil {
		return ErrorResponse{}
	}

	code := "render_error"
	switch {
	case errors.Is(err, errInputTooLarge), errors.Is(err, errOutputTooLarge),
		errors.Is(err, errImageTooLarge):
		code = "resource_limit"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		code = "timeout"
	case errors.Is(err, errInvalidRequest), errors.Is(err, errUnsupportedMode),
		errors.Is(err, gowkhtmltopdf.ErrInvalidContent),
		errors.Is(err, gowkhtmltopdf.ErrEmptyHTML):
		code = "invalid_request"
	}

	return ErrorResponse{Code: code, Message: err.Error()}
}
