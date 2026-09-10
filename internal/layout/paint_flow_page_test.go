package layout

import "testing"

// TestPageBoundaryBucketersAgree pins the op-to-page mapping at page
// boundaries. A rect fragment split at a page top lands exactly on
// k*contentH, and float division can round a hair below k; pageBuckets
// already biased its Y by layoutEpsilon so such an op stays on the page it
// starts, but buildFlowOpIndex and pageIndexedOps did not. The three
// bucketers must agree on which page owns a boundary-aligned op.
func TestPageBoundaryBucketersAgree(t *testing.T) {
	t.Parallel()

	const contentH = 785.197

	base := 21 * contentH
	cases := []struct {
		name string
		y    float64
	}{
		{"boundary-minus-nano", base - 1e-9},
		{"boundary-exact", base},
		{"boundary-minus-micro", base - 1e-6},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			checkPageBoundaryBucket(t, testCase.y, contentH)
		})
	}

	// The shared owner keeps checkedFlowPageOfY's maxFlowPageIndex guard.
	if _, ok := checkedFlowPageOfY(contentH*float64(maxFlowPageIndex)+contentH, contentH); ok {
		t.Error("checkedFlowPageOfY admitted a page at the flow index bound")
	}
}

// checkPageBoundaryBucket proves the three bucketers agree on the page that
// owns a boundary-aligned op at boundaryY.
func checkPageBoundaryBucket(t *testing.T, boundaryY, contentH float64) {
	t.Helper()

	ops := []Op{{Kind: OpLine, Y: boundaryY}}

	want, accepted := checkedFlowPageOfY(boundaryY+layoutEpsilon, contentH)
	if !accepted {
		t.Fatalf("checkedFlowPageOfY(y+epsilon) rejected y=%g", boundaryY)
	}

	pageOf, _ := pageBuckets(ops, contentH)
	if len(pageOf) != 1 {
		t.Fatalf("pageBuckets returned %d entries, want 1", len(pageOf))
	}

	if pageOf[0] != want {
		t.Errorf("pageBuckets page = %d, want %d", pageOf[0], want)
	}

	_, flowPageOf, _, ok := buildFlowOpIndex(ops, contentH)
	if !ok {
		t.Fatal("buildFlowOpIndex rejected the probe op")
	}

	if flowPageOf[0] != want {
		t.Errorf("buildFlowOpIndex page = %d, want %d", flowPageOf[0], want)
	}

	if got := pageIndexOfOp(&Result{Ops: ops}, contentH); got != want {
		t.Errorf("pageIndexedOps page = %d, want %d", got, want)
	}
}

// pageIndexOfOp returns the page pageIndexedOps assigned to ops[0], or -1
// when no page claims it.
func pageIndexOfOp(result *Result, contentH float64) int {
	got := -1

	for page, idxs := range pageIndexedOps(result, contentH) {
		for _, opIdx := range idxs {
			if opIdx == 0 {
				got = page
			}
		}
	}

	return got
}
