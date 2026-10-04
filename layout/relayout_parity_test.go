package layout_test

import "testing"

const paritySource = `
<style>
#bar { width: 50vw; padding: 4px; background-color: #123456; }
#bar:hover { background-color: #654321; }
@media (min-width: 400px) { #bar { width: 25vw; } }
</style>
<div id="bar"><span>parity probe</span></div>
`

// TestRelayoutMatchesFullApply is the placement parity gate: the relayout
// result must equal a full Apply at the same viewport and state, box for box
// and op for op.
func TestRelayoutMatchesFullApply(t *testing.T) {
	t.Parallel()

	styled := applyAt(t, paritySource, 320, 240, "", "", "")
	relaid := relayoutDisplay(t, styled, 640, 480, "", "bar", "")
	full := displayAt(t, paritySource, 640, 480, "", "bar", "")

	compareDisplays(t, relaid, full)
}

// TestRelayoutDoesNotLeakBetweenSizes runs two relayouts in a row with
// different sizes and states, then compares the second result with a fresh
// Apply. A sheet cache or state carried across the first relayout would show
// up here.
func TestRelayoutDoesNotLeakBetweenSizes(t *testing.T) {
	t.Parallel()

	styled := applyAt(t, paritySource, 320, 240, "", "bar", "")

	wide := relayout(t, styled, 640, 480, "", "bar", "")
	narrow := relayout(t, wide, 320, 240, "", "", "")

	compareDisplays(t, layOf(t, narrow), displayAt(t, paritySource, 320, 240, "", "", ""))
}
