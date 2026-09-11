package layout

import (
	"context"
	"testing"
)

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

// paginateOpsForTest runs paginateOps and returns the settled pre-split
// op-to-page assignment. Production PaintContext does not keep this map:
// buildPagesAfterSplits rebuilds page buckets after rect splitting and sticky
// shifts, so the pre-split slice would be stale as well as discarded. Tests
// that assert the settled assignment use this helper.
func paginateOpsForTest(ctx context.Context, res *Result, contentH float64) ([]int, error) {
	if err := paginateOps(ctx, res, contentH); err != nil {
		return nil, err
	}

	opPage := make([]int, len(res.Ops))

	for opIdx := range res.Ops {
		page, ok := checkedFlowPageOfY(res.Ops[opIdx].Y, contentH)
		if !ok {
			opPage[opIdx] = -1
		} else {
			opPage[opIdx] = page
		}
	}

	return opPage, nil
}

// TestPaginateOpsDoesNotBuildDiscardedPageMap pins the PDF-04 removal. The
// production path must not allocate the pre-split op-to-page slice that Paint
// discarded; paginateOpsForTest builds that slice, so its steady-state
// allocation count must exceed the production call by at least one.
//
// This test is intentionally not parallel: AllocsPerRun reads process-wide
// Mallocs, and the runner starts parallel tests only after sequential tests
// finish, so this measurement sees a quiet process.
//
//nolint:paralleltest // AllocsPerRun reads process-wide counters; parallelism would pollute it.
func TestPaginateOpsDoesNotBuildDiscardedPageMap(t *testing.T) {
	const contentH = 700.0

	fixture := `<html><body><p>alpha</p><p>beta</p></body></html>`
	prodRes := layoutHTML(t, fixture)
	testRes := layoutHTML(t, fixture)

	ctx := t.Context()

	prodAllocs := testing.AllocsPerRun(50, func() {
		if err := paginateOps(ctx, prodRes, contentH); err != nil {
			t.Fatalf("paginateOps: %v", err)
		}
	})
	withMapAllocs := testing.AllocsPerRun(50, func() {
		if _, err := paginateOpsForTest(ctx, testRes, contentH); err != nil {
			t.Fatalf("paginateOpsForTest: %v", err)
		}
	})

	t.Logf(
		"paginateOps allocs/run = %.1f, paginateOpsForTest allocs/run = %.1f, ops = %d; "+
			"the test helper builds one []int of len(ops)",
		prodAllocs, withMapAllocs, len(prodRes.Ops),
	)

	if diff := withMapAllocs - prodAllocs; diff < 1 {
		t.Fatalf(
			"paginateOps allocs = %.1f, paginateOpsForTest allocs = %.1f; production must not build the discarded op-page map",
			prodAllocs, withMapAllocs,
		)
	}
}
