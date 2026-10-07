package layout_test

import "testing"

const mediaVWSource = `
<style>
#bar { width: 50vw; height: 10vh; background-color: #123456; }
@media (min-width: 400px) { #bar { background-color: #abcdef; } }
</style>
<div id="bar"></div>
`

func TestRelayoutFlipsMediaAndViewportUnits(t *testing.T) {
	t.Parallel()

	narrow := displayAt(t, mediaVWSource, 320, 240, "", "", "")
	narrowBar := boxByID(t, narrow, "bar")

	nearBox(t, "narrow width", narrowBar.W, 160)
	nearBox(t, "narrow height", narrowBar.H, 24)

	if !hasFill(narrow, 0x12, 0x34, 0x56) {
		t.Fatal("narrow viewport did not paint the base fill")
	}

	if hasFill(narrow, 0xab, 0xcd, 0xef) {
		t.Fatal("narrow viewport matched min-width:400px")
	}

	styled := applyAt(t, mediaVWSource, 320, 240, "", "", "")
	wide := relayoutDisplay(t, styled, 640, 480, "", "", "")
	wideBar := boxByID(t, wide, "bar")

	nearBox(t, "wide width", wideBar.W, 320)
	nearBox(t, "wide height", wideBar.H, 48)

	if !hasFill(wide, 0xab, 0xcd, 0xef) {
		t.Fatal("wide viewport did not flip the media rule")
	}

	if hasFill(wide, 0x12, 0x34, 0x56) {
		t.Fatal("wide viewport kept the base fill")
	}
}
