//nolint:testpackage // probes unexported vertical-cluster seal helpers
package layout

import "testing"

func TestVerticalClusterStartsSoon(t *testing.T) {
	t.Parallel()

	starts := map[int]borderCluster{
		762: {y: 761.9, minX: 34, maxX: 561, bw: 1, n: 5, r: 0, g: 0, b: 0},
	}

	if !verticalClusterStartsSoon(starts, 755.8, 34, 561, 800) {
		t.Fatal("expected same-page continuation just below endY")
	}

	if verticalClusterStartsSoon(starts, 755.8, 34, 561, 760) {
		t.Fatal("cluster past pageBot must not count as same-page continuation")
	}

	if verticalClusterStartsSoon(starts, 700, 34, 561, 800) {
		t.Fatal("cluster far below soonBand must not match")
	}

	// Repeated thead clones start exactly at pageBot; that must not suppress
	// sealing the previous page's open last-row bottom.
	atBoundary := map[int]borderCluster{
		1548: {y: 1547.72, minX: 0, maxX: 527, bw: 1, n: 5, r: 0, g: 0, b: 0},
	}

	if verticalClusterStartsSoon(atBoundary, 1541.60, 0, 527, 1547.72) {
		t.Fatal("cluster at pageBot (next-page thead) must not count as same-page continuation")
	}
}

func TestVerticalClusterStartsBefore(t *testing.T) {
	t.Parallel()

	// Last-on-page detection: a continuation 90pt below still counts (the old
	// 80pt bottomSealBand missed fixture-61 props 31/82).
	starts := map[int]borderCluster{
		845: {y: 845, minX: 34, maxX: 561, bw: 1, n: 5, r: 0, g: 0, b: 0},
	}

	if !verticalClusterStartsBefore(starts, 755, 34, 561, 900) {
		t.Fatal("expected same-page continuation within pageBot band")
	}

	if verticalClusterStartsBefore(starts, 755, 34, 561, 840) {
		t.Fatal("cluster at/past pageBot must not count")
	}

	if verticalClusterStartsBefore(starts, 850, 34, 561, 900) {
		t.Fatal("cluster above endY must not count")
	}
}
