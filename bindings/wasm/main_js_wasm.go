//go:build js && wasm

package main

import (
	"context"
	"syscall/js"
	"time"
)

const conversionTimeout = 60 * time.Second

func main() {
	convert := js.FuncOf(convertJS)
	js.Global().Set("gowkhtmltopdfWASM", convert)
	select {}
}

func convertJS(_ js.Value, args []js.Value) (response any) {
	defer func() {
		if recover() != nil {
			response = jsError(ErrorResponse{
				Code:    "internal_error",
				Message: "WASM conversion failed unexpectedly",
			})
		}
	}()

	if len(args) < 1 || len(args) > 2 || args[0].Type() != js.TypeString {
		return jsError(errorResponse(errInvalidRequest))
	}

	var progress js.Value
	hasProgress := false
	if len(args) == 2 {
		if args[1].Type() != js.TypeFunction {
			return jsError(errorResponse(errInvalidRequest))
		}
		progress = args[1]
		hasProgress = true
	}

	request, err := DecodeRequest(args[0].String())
	if err != nil {
		return jsError(errorResponse(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), conversionTimeout)
	defer cancel()
	result, err := Convert(ctx, request, func(phase string, percent int) {
		if hasProgress {
			progress.Invoke(phase, percent)
		}
	})
	if err != nil {
		return jsError(errorResponse(err))
	}

	return jsResult(result)
}

func jsResult(result Result) js.Value {
	response := js.Global().Get("Object").New()
	response.Set("ok", true)
	response.Set("mode", result.Mode)
	response.Set("mime", result.MIME)
	response.Set("version", wasmVersion)
	response.Set("width", result.Width)
	response.Set("height", result.Height)

	bytes := js.Global().Get("Uint8Array").New(len(result.Bytes))
	js.CopyBytesToJS(bytes, result.Bytes)
	response.Set("bytes", bytes)

	return response
}

func jsError(err ErrorResponse) js.Value {
	response := js.Global().Get("Object").New()
	response.Set("ok", false)
	errorValue := js.Global().Get("Object").New()
	errorValue.Set("code", err.Code)
	errorValue.Set("message", err.Message)
	response.Set("error", errorValue)

	return response
}
