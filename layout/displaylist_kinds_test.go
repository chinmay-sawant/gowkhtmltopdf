package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// publicKinds is every kind the layout package names for a consumer. A kind
// that reaches Display.Ops without a name here is one a caller cannot switch
// on, which is how a deactivated operation or a group marker would silently
// fall through to a zero value and paint as a fill.
func publicKinds() map[layout.DisplayKind]bool {
	return map[layout.DisplayKind]bool{
		layout.DisplayOpNoop:       true,
		layout.DisplayOpUnknown:    true,
		layout.DisplayOpFillRect:   true,
		layout.DisplayOpStrokeRect: true,
		layout.DisplayOpLine:       true,
		layout.DisplayOpText:       true,
		layout.DisplayOpImage:      true,
		layout.DisplayOpLinkURI:    true,
		layout.DisplayOpBullet:     true,
		layout.DisplayOpGridRun:    true,
	}
}

// TestEveryEmittedKindIsNamed walks documents chosen to make the engine emit
// each kind, and fails if any operation carries a kind the public list does not
// name. The noop case is the one that first escaped: overflow clipping writes
// it, so a document with overflow:hidden hands a consumer kind 255.
func TestEveryEmittedKindIsNamed(t *testing.T) {
	t.Parallel()

	sources := []string{
		`<div style="background:#eee;border:1px solid #333;border-radius:8px">box</div>`,
		`<p>text</p><ul><li>bullet</li></ul>`,
		`<a href="/x">link</a>`,
		`<table style="border-collapse:collapse"><tr><td style="border:1px solid #ccc">a</td></tr></table>`,
		`<div style="height:200px;overflow:hidden"><div style="height:900px"></div></div>`,
		`<div style="isolation:isolate;background:#eee">group</div>`,
		`<div style="mix-blend-mode:multiply;background:#ccc">blend</div>`,
	}

	named := publicKinds()

	for _, source := range sources {
		display := displayOf(t, source)

		for index, paintOp := range display.Ops {
			if !named[paintOp.Kind] {
				t.Errorf("source %q op %d carries unnamed kind %d", source, index, paintOp.Kind)
			}
		}
	}
}

// TestNoopKindIsNameable pins the value a caller has to skip. It lives outside
// the OpKind sequence so a switch over the named kinds cannot accidentally
// treat it as one.
func TestNoopKindIsNameable(t *testing.T) {
	t.Parallel()

	if layout.DisplayOpNoop != 255 {
		t.Errorf("DisplayOpNoop is %d, want 255", layout.DisplayOpNoop)
	}

	named := publicKinds()
	for _, kind := range []layout.DisplayKind{
		layout.DisplayOpFillRect,
		layout.DisplayOpStrokeRect,
		layout.DisplayOpLine,
		layout.DisplayOpText,
		layout.DisplayOpImage,
		layout.DisplayOpLinkURI,
		layout.DisplayOpBullet,
		layout.DisplayOpGridRun,
	} {
		if !named[kind] {
			t.Errorf("kind %d is not in the named set", kind)
		}
	}
}

// TestOverflowClippingEmitsNoop is the regression guard for the case that
// exposed the missing constant: an absolutely positioned child clipped by an
// overflow box on a positioned ancestor is deactivated rather than removed,
// because the box tree stores operation indices that must not shift. A
// consumer iterating a real page therefore meets kind 255 in the middle of it.
func TestOverflowClippingEmitsNoop(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<div style="height:40px;overflow:hidden;position:relative">`+
		`<div style="position:absolute;top:200px;width:30px;height:30px;background:#0f0"></div>`+
		`<p>visible</p></div>`)

	var noops, painted int

	for _, paintOp := range display.Ops {
		if paintOp.Kind == layout.DisplayOpNoop {
			noops++
		} else {
			painted++
		}
	}

	if noops == 0 {
		t.Fatal("clipped positioned child left no deactivated operation behind")
	}

	if painted == 0 {
		t.Fatal("the visible content was deactivated too")
	}
}
