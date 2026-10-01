package screen_test

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/screen"
)

const pngSignature = "\x89PNG\r\n\x1a\n"

func TestRenderInnerBoxIsHitTarget(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><head><style>
body { margin: 0; }
#outer { width: 200px; height: 80px; background: #dddddd; }
#inner { width: 100px; height: 40px; background: #2266cc; }
</style></head><body><div id="outer"><div id="inner" data-action="login">Go</div></div></body></html>`

	frame := mustRender(t, page, 320, 200)
	requirePNG(t, frame, 320, 200)

	inner := findBox(t, frame.Boxes, "inner")
	requireInner(t, inner)
	requireInnerHit(t, frame.Boxes, inner)
}

func TestRenderInputValueDoesNotChangePNG(t *testing.T) {
	t.Parallel()

	const head = `<!DOCTYPE html><html><head><style>body{margin:0}</style></head><body>`

	valueHi := mustRender(t, head+`<input id="email" value="hi"></body></html>`, 200, 80)
	valueYo := mustRender(t, head+`<input id="email" value="yo"></body></html>`, 200, 80)

	if !bytes.Equal(valueHi.PNG, valueYo.PNG) {
		t.Fatalf("png changed with input value (%d vs %d bytes)", len(valueHi.PNG), len(valueYo.PNG))
	}

	email := findBox(t, valueHi.Boxes, "email")
	if email.Text != "" {
		t.Fatalf("email text = %q", email.Text)
	}
}

func TestRenderRejectsBadInput(t *testing.T) {
	t.Parallel()

	// nil is the case under test: Render must reject a missing context.
	_, err := screen.Render(nil, []byte("<p>Hi</p>"), 50, 40) //nolint:staticcheck // nil context is intentional
	if !errors.Is(err, screen.ErrNilContext) {
		t.Fatalf("nil context: %v", err)
	}

	_, err = screen.Render(t.Context(), []byte("  \n"), 50, 40)
	if !errors.Is(err, screen.ErrEmptyHTML) {
		t.Fatalf("empty: %v", err)
	}

	_, err = screen.Render(t.Context(), []byte("<p>Hi</p>"), 0, 40)
	if !errors.Is(err, screen.ErrBadSize) {
		t.Fatalf("width: %v", err)
	}

	_, err = screen.Render(t.Context(), []byte("<p>Hi</p>"), 50, -1)
	if !errors.Is(err, screen.ErrBadSize) {
		t.Fatalf("height: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = screen.Render(ctx, []byte("<p>Hi</p>"), 50, 40)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
}

func mustRender(t *testing.T, page string, width, height int) *screen.Frame {
	t.Helper()

	frame, err := screen.Render(t.Context(), []byte(page), width, height)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	return frame
}

func requirePNG(t *testing.T, frame *screen.Frame, width, minHeight int) {
	t.Helper()

	if !bytes.HasPrefix(frame.PNG, []byte(pngSignature)) {
		t.Fatal("png signature missing")
	}

	if frame.Width != width || frame.Height < minHeight {
		t.Fatalf("frame size %dx%d", frame.Width, frame.Height)
	}
}

func requireInner(t *testing.T, inner screen.Box) {
	t.Helper()

	if math.Abs(inner.W-100) > 2 || math.Abs(inner.H-40) > 2 {
		t.Fatalf("inner size %.2fx%.2f", inner.W, inner.H)
	}

	if inner.Action != "login" || !strings.Contains(inner.Text, "Go") {
		t.Fatalf("inner action %q text %q", inner.Action, inner.Text)
	}
}

func requireInnerHit(t *testing.T, boxes []screen.Box, inner screen.Box) {
	t.Helper()

	hit := hitBox(boxes, inner.X+inner.W/2, inner.Y+inner.H/2)
	if hit == nil || hit.ID != "inner" {
		t.Fatalf("center hit %#v", hit)
	}

	outside := hitBox(boxes, inner.X+inner.W+10, inner.Y+4)
	if outside != nil && outside.ID == "inner" {
		t.Fatal("point past inner still hit inner")
	}
}

func findBox(t *testing.T, boxes []screen.Box, boxID string) screen.Box {
	t.Helper()

	for _, box := range boxes {
		if box.ID == boxID {
			return box
		}
	}

	t.Fatalf("box %q not found", boxID)

	return screen.Box{}
}

func hitBox(boxes []screen.Box, x, y float64) *screen.Box {
	var found *screen.Box

	for i := range boxes {
		box := &boxes[i]
		if x >= box.X && x < box.X+box.W && y >= box.Y && y < box.Y+box.H {
			found = box
		}
	}

	return found
}
