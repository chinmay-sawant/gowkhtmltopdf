package layout

import (
	"strings"
	"testing"
)

// inputBoxByType returns the laid-out box of the first input element whose
// type attribute matches typ.
func inputBoxByType(t *testing.T, res *Result, typ string) *box {
	t.Helper()

	var found *box

	var walk func(*box)

	walk = func(boxNode *box) {
		if boxNode.node != nil && boxNode.node.Name == htmlInput &&
			strings.EqualFold(boxNode.node.Attribute("type"), typ) {
			found = boxNode
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}

	walk(res.root)

	if found == nil {
		t.Fatalf("fixture has no input[type=%s] box", typ)
	}

	return found
}

// case33ThumbOp returns the range thumb fill: a fully rounded rect carrying the
// Chrome range accent fill.
func case33ThumbOp(res *Result) *Op {
	for i := range res.Ops {
		operation := &res.Ops[i]
		if operation.Kind == OpFillRect && near(operation.R, 0) &&
			near(operation.G, 117.0/255.0) && near(operation.B, 1) &&
			near(operation.W, operation.H) && near(operation.Radius, operation.W/2) {
			return operation
		}
	}

	return nil
}

// case33LabelOp returns the button label text op.
func case33LabelOp(res *Result) *Op {
	for i := range res.Ops {
		operation := &res.Ops[i]
		if operation.Kind == OpText && operation.Text == "XXXXXXX" {
			return operation
		}
	}

	return nil
}

func assertCentered(t *testing.T, label string, centerX, centerY, wantX, wantY float64) {
	t.Helper()

	if !near(centerX, wantX) || !near(centerY, wantY) {
		t.Fatalf("%s center (%.2f, %.2f), want (%.2f, %.2f)", label, centerX, centerY, wantX, wantY)
	}
}

func assertWithin(t *testing.T, label string, lo, hi, wantLo, wantHi float64) {
	t.Helper()

	if lo < wantLo || hi > wantHi {
		t.Fatalf("%s spans %.2f..%.2f outside %.2f..%.2f", label, lo, hi, wantLo, wantHi)
	}
}

// TestChromeFixtureCase33InputFaces pins the two native faces Chromium paints
// for the case-33 inputs: range keeps a fully rounded thumb centered in its
// border box, and button paints the value attribute centered in its border box
// at the UA form-control font size (13.333px = 10pt).
func TestChromeFixtureCase33InputFaces(t *testing.T) {
	t.Parallel()

	res := fixture21To40(t, "case-33-wpt-flex-item-compressible.html")
	rangeBox := inputBoxByType(t, res, "range")
	buttonBox := inputBoxByType(t, res, "button")

	thumb := case33ThumbOp(res)
	if thumb == nil {
		t.Fatal("range thumb fill not painted")
	}

	const wantThumb = 11.25 // scalePt(15) at 0.75 pt per CSS px

	if !near(thumb.W, wantThumb) || !near(thumb.H, wantThumb) || !near(thumb.Radius, wantThumb/2) {
		t.Fatalf("range thumb %.2fx%.2f radius %.2f, want %.2fx%.2f radius %.2f",
			thumb.W, thumb.H, thumb.Radius, wantThumb, wantThumb, wantThumb/2)
	}

	assertCentered(t, "range thumb", thumb.X+thumb.W/2, thumb.Y+thumb.H/2,
		rangeBox.x+rangeBox.w/2, rangeBox.y+rangeBox.height/2)

	label := case33LabelOp(res)
	if label == nil {
		t.Fatal("button label text not painted")
	}

	if !near(label.Size, 10) {
		t.Fatalf("button label size %.2fpt, want 10pt", label.Size)
	}

	assertCentered(t, "button label", label.X+label.W/2,
		label.Y-label.H/2+label.InkDescent, buttonBox.x+buttonBox.w/2,
		buttonBox.y+buttonBox.height/2)
	assertWithin(t, "button label", label.X, label.X+label.W,
		buttonBox.x, buttonBox.x+buttonBox.w)
}
