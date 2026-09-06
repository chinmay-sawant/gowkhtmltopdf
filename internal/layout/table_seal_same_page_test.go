package layout

import "testing"

func TestVerticalClusterStartsSoon(t *testing.T) {
	t.Parallel()

	starts := map[int]borderCluster{
		762: {y: 761.9, minX: 34, maxX: 561, bw: 1, n: 5},
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
}
