package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

const stateSource = `
<style>
#b { color: #111111; }
#b:hover { color: #ff0000; }
#b:focus { background-color: #00ff00; }
#b:active { background-color: #0000ff; }
</style>
<div id="b">state probe</div>
`

func TestRelayoutAppliesStatePseudoClasses(t *testing.T) {
	t.Parallel()

	styled := applyAt(t, stateSource, 320, 240, "", "", "")

	plain := layOf(t, styled)
	assertTextColor(t, plain, 0x11, 0x11, 0x11)

	if hasFill(plain, 0, 0, 0xff) || hasFill(plain, 0, 0xff, 0) {
		t.Fatal("plain layout carries a state background")
	}

	hovered := relayoutDisplay(t, styled, 320, 240, "", "b", "")
	assertTextColor(t, hovered, 0xff, 0, 0)

	if hasFill(hovered, 0, 0, 0xff) {
		t.Fatal("hover alone painted the active background")
	}

	pressed := relayoutDisplay(t, styled, 320, 240, "b", "b", "b")
	assertTextColor(t, pressed, 0xff, 0, 0)

	if !hasFill(pressed, 0, 0, 0xff) {
		t.Fatal("active id did not paint the active background")
	}

	if hasFill(pressed, 0, 0xff, 0) {
		t.Fatal("active should win over focus by source order")
	}

	cleared := relayoutDisplay(t, styled, 320, 240, "", "", "")
	assertTextColor(t, cleared, 0x11, 0x11, 0x11)

	if hasFill(cleared, 0, 0, 0xff) || hasFill(cleared, 0, 0xff, 0) {
		t.Fatal("state background leaked into a state-free relayout")
	}
}

func assertTextColor(t *testing.T, display *layout.Display, wantR, wantG, wantB uint8) {
	t.Helper()

	const needle = "state probe"

	gotR, gotG, gotB, ok := textColor(display, needle)
	if !ok || gotR != wantR || gotG != wantG || gotB != wantB {
		t.Fatalf("text %q color %02x%02x%02x ok=%v, want %02x%02x%02x",
			needle, gotR, gotG, gotB, ok, wantR, wantG, wantB)
	}
}
