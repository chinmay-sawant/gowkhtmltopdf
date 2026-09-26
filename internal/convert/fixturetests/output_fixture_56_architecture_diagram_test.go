package fixturetests

import "testing"

// TestOutputFixture56ArchitectureDiagram checks the first-page pipeline,
// a middle-page dependency heading, and the page-21 security heading.
// Measurements came from scripts/inspect_pdf_ops.py.
func TestOutputFixture56ArchitectureDiagram(t *testing.T) {
	committed := readCommittedOps(t, "fixture-56-architecture-diagram.pdf")
	fresh := freshOps(t, "fixture-56-architecture-diagram.html")

	assertPageOpsMatch(t, committed, fresh)

	ink := [3]float64{20.0 / 255, 32.0 / 255, 43.0 / 255}
	assertOpsTextRun(t, committed, 1, "Conversion pipeline ",
		34.016, 562.051, 13, "LiberationSerif-Bold", ink)
	assertOpsTextRun(t, committed, 19, "Dependency DAG ",
		50.766, 783.251, 14, "LiberationSerif-Bold", ink)
	assertOpsTextRun(t, committed, 21, "Security architecture ",
		50.766, 783.251, 14, "LiberationSerif-Bold", ink)

	assertFixtureOpsShape(t, committed, 21, 1920, 2206, 10834, 9, [4]float64{0, 0, 595.28, 841.89})
}
