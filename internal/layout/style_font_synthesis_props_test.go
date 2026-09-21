//nolint:all // targeted unit tests for font-synthesis
package layout

import (
	"strings"
	"testing"
)

func TestFontSynthesisWeightNoneDisablesFakeBold(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontSynthesisProps(&style, "font-synthesis-weight", "none", 12, nil, nil, false) {
		t.Fatal("font-synthesis-weight not owned")
	}

	if style.FontSynthesisWeight {
		t.Fatal("expected FontSynthesisWeight=false")
	}

	doc := parseTestHTML(t, `<html><body style="margin:0">`+
		`<p style="font-weight:700;font-synthesis-weight:none">nosynth</p>`+
		`<p style="font-weight:700">dosynth</p>`+
		`</body></html>`)

	res, err := Layout(doc, Options{Width: 500, Height: 400})
	if err != nil {
		t.Fatal(err)
	}

	var noSynth, doSynth *Op

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "nosynth") {
			noSynth = paintOp
		}

		if strings.Contains(paintOp.Text, "dosynth") {
			doSynth = paintOp
		}
	}

	if noSynth == nil || doSynth == nil {
		t.Fatalf("missing ops: no=%v do=%v", noSynth != nil, doSynth != nil)
	}

	if !noSynth.NoFakeBold {
		t.Fatal("font-synthesis-weight:none must set NoFakeBold")
	}

	if FakeBoldFor(noSynth) {
		t.Fatal("FakeBoldFor must be false when NoFakeBold is set")
	}

	// dosynth may or may not fake-bold depending on whether a real bold face
	// resolved; NoFakeBold must stay false so FakeBoldFor can still decide.
	if doSynth.NoFakeBold {
		t.Fatal("default synthesis must leave NoFakeBold false")
	}
}

func TestFontSynthesisShorthandNone(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontSynthesisProps(&style, "font-synthesis", "none", 12, nil, nil, false) {
		t.Fatal("font-synthesis not owned")
	}

	if style.FontSynthesisWeight || style.FontSynthesisStyle ||
		style.FontSynthesisSmallCaps || style.FontSynthesisPosition {
		t.Fatal("font-synthesis:none must clear all allow flags")
	}
}
