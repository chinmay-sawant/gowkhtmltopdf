package convert

import "testing"

// TestOutputFixture40TransformBadge checks the transformed badge, its white
// label, and the sibling that remains in normal flow.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture40TransformBadge(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-40-transform-badge.pdf")
	fresh := freshOps(t, "fixture-40-transform-badge.html")

	assertPageOpsMatch(t, committed, fresh)

	black := [3]float64{0, 0, 0}
	assertOpsTextRun(t, committed, 1, "Static transform badge",
		52.346, 774.397, 16, "LiberationSans-Bold", black)
	assertOpsTextRun(t, committed, 1, "NEW",
		61.149, 742.204, 9, "LiberationSans-Bold", [3]float64{1, 1, 1})
	assertOpsTextRun(t, committed, 1, "Sibling stays in flow",
		100.400, 744.930, 11, "LiberationSans-Regular", black)
	assertOpsFillRect(t, committed, 1,
		50.929, 735.377, 39.677, 24.126,
		[3]float64{198.0 / 255, 40.0 / 255, 40.0 / 255})

	assertFixtureOpsShape(t, committed, 1, 6, 152, 5, 0, [4]float64{0, 0, 595.28, 841.89})
}
