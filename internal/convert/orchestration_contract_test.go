package convert

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

//nolint:wsl // focused white-box seam fixture.
func TestLayoutBodyKeepsSmartShrinkAtAReplaceableSeam(t *testing.T) {
	t.Parallel()

	state := &objectState{
		idx:  0,
		obj:  &settings.PdfObject{Page: "inline"},
		geom: hfGeom{contentW: 100},
	}
	render := objectRenderContext{
		global: settings.PdfGlobal{SmartShrinking: true},
		obj:    state.obj,
		zoom:   1,
	}
	var calls int
	var zooms []float64
	result, _, err := layoutBody(
		t.Context(),
		state,
		render,
		io.Discard,
		func(opts layout.Options) (*layout.Result, error) {
			calls++
			zooms = append(zooms, opts.Zoom)
			width := 200.0
			if calls > 1 {
				width = 100
			}

			return &layout.Result{Width: width}, nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("layoutBody: %v", err)
	}
	if result == nil || calls != 2 {
		t.Fatalf("layout calls = %d, result = %v, want two calls", calls, result != nil)
	}
	if len(zooms) != 2 || zooms[1] >= zooms[0] {
		t.Fatalf("layout zooms = %v, want a reduced second-pass zoom", zooms)
	}
}

//nolint:wsl // focused white-box seam fixture.
func TestLayoutBodyStopsBeforeLayoutWhenCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	state := &objectState{obj: &settings.PdfObject{Page: "inline"}}
	calls := 0
	_, _, err := layoutBody(ctx, state, objectRenderContext{}, io.Discard,
		func(layout.Options) (*layout.Result, error) {
			calls++

			return &layout.Result{}, nil
		}, nil)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("layoutBody error = %v, want context.Canceled", err)
	}
	if calls != 0 {
		t.Fatalf("layout calls = %d, want no callback invocation", calls)
	}
}
