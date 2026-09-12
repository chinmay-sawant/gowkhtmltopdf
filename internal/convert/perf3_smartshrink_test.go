package convert

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestSmartShrinkNoRelayoutWhenWithinTenthPoint(t *testing.T) {
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

	result, _, err := layoutBody(
		t.Context(),
		state,
		render,
		io.Discard,
		func(layout.Options) (*layout.Result, error) {
			calls++

			return &layout.Result{Width: 100, MaxContentX: 100.05}, nil
		},
	)

	if err != nil {
		t.Fatalf("layoutBody: %v", err)
	}

	if result == nil || calls != 1 {
		t.Fatalf("layout calls = %d, want 1 (content fits within 0.1pt)", calls)
	}
}

func TestSmartShrinkRelayoutsWhenMaxContentXOverflows(t *testing.T) {
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

	_, _, err := layoutBody(
		t.Context(),
		state,
		render,
		io.Discard,
		func(layout.Options) (*layout.Result, error) {
			calls++
			width := 100.0
			maxX := 120.0

			if calls > 1 {
				maxX = 100
			}

			return &layout.Result{Width: width, MaxContentX: maxX}, nil
		},
	)

	if err != nil {
		t.Fatalf("layoutBody: %v", err)
	}

	if calls != 2 {
		t.Fatalf("layout calls = %d, want 2 (overflow forces shrink)", calls)
	}
}

func TestMeasuredWidthFastPrefersMaxContentX(t *testing.T) {
	t.Parallel()

	if got := measuredWidth(&layout.Result{Width: 50, MaxContentX: 80}); got != 80 {
		t.Fatalf("measuredWidth = %v, want 80", got)
	}

	if got := measuredWidth(&layout.Result{Width: 90, MaxContentX: 80}); got != 90 {
		t.Fatalf("measuredWidth = %v, want Width 90", got)
	}

	ops := []layout.Op{{Kind: layout.OpFillRect, X: 0, W: 140}}
	if got := measuredWidth(&layout.Result{Width: 100, Ops: ops}); got != 140 {
		t.Fatalf("scan fallback = %v, want 140", got)
	}
}

func TestSmartShrinkFitsOnNarrowPage(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html><body><p>fits</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
}
