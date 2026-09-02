//nolint:testpackage,wsl // white-box compositing tests use private style and layout seams
package layout

import "testing"

func TestBlendModeParsing(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyAdvancedProps(&style, "mix-blend-mode", " Multiply ", 12) {
		t.Fatal("mix-blend-mode was not accepted")
	}
	if style.MixBlendMode != blendMultiply {
		t.Fatalf("MixBlendMode = %q, want %q", style.MixBlendMode, blendMultiply)
	}
	if !applyAdvancedProps(&style, "background-blend-mode", "screen, multiply", 12) {
		t.Fatal("background-blend-mode was not accepted")
	}
	if style.BackgroundBlendMode != "screen, multiply" {
		t.Fatalf("BackgroundBlendMode = %q", style.BackgroundBlendMode)
	}
	if !applyAdvancedProps(&style, "isolation", "isolate", 12) {
		t.Fatal("isolation was not accepted")
	}
	if style.Isolation != "isolate" {
		t.Fatalf("Isolation = %q, want isolate", style.Isolation)
	}
	if applyAdvancedProps(&style, "mix-blend-mode", "unsupported-mode", 12) {
		t.Fatal("unsupported blend mode was accepted")
	}
	if applyAdvancedProps(&style, "mix-blend-mode", "plus-lighter", 12) {
		t.Fatal("plus-lighter was accepted before additive compositing support")
	}
}

func TestBackgroundBlendModeForLayerRepeatsLastMode(t *testing.T) {
	t.Parallel()

	if got := backgroundBlendModeForLayer("screen, multiply", 0); got != blendScreen {
		t.Fatalf("layer 0 mode = %q, want %q", got, blendScreen)
	}
	if got := backgroundBlendModeForLayer("screen, multiply", 4); got != blendMultiply {
		t.Fatalf("layer 4 mode = %q, want %q", got, blendMultiply)
	}
	if got := backgroundBlendModeForLayer("", 0); got != blendNormal {
		t.Fatalf("empty mode = %q, want %q", got, blendNormal)
	}
}

func TestBlendColorMultiply(t *testing.T) {
	t.Parallel()

	got := BlendColor(blendMultiply, [3]float64{0.5, 0.5, 0.5}, [3]float64{1, 0, 0})
	want := [3]float64{0.5, 0, 0}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("channel %d = %v, want %v", index, got[index], want[index])
		}
	}
}

func TestMixBlendModeReachesDisplayList(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="mix-blend-mode:multiply;background:#fff">blended</div></body></html>`)

	textBlended := false
	backgroundBlended := false
	for _, op := range res.Ops {
		if op.Kind == OpText {
			textBlended = op.BlendMode == blendMultiply
		}
		if op.Kind == OpFillRect {
			backgroundBlended = op.BlendMode == blendMultiply
		}
	}

	if !textBlended {
		t.Fatal("display list text has no multiply blend mode")
	}
	if !backgroundBlended {
		t.Fatal("display list background has no multiply blend mode")
	}
}

func TestIsolationClearsInheritedOperationBlendScope(t *testing.T) {
	t.Parallel()

	source := `<html><body><div style="mix-blend-mode:multiply">` +
		`<span style="isolation:isolate">isolated</span></div></body></html>`
	res := layoutHTML(t, source)

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "isolated" {
			if op.BlendMode != "" {
				t.Fatalf("isolated text blend mode = %q, want empty", op.BlendMode)
			}

			return
		}
	}

	t.Fatal("display list has no isolated text")
}
