package layout

import (
	"testing"
)

// Phase-6 reuse contract: the page-index builders reset retained storage
// instead of reallocating, res.Pages owns its buckets, and a later rebuild
// cannot mutate them. TestPaginateOpsDoesNotBuildDiscardedPageMap documents
// the sequential-run rule for AllocsPerRun.

// TestPageIndexRebuildReusesStorage proves an invalidate + ensureFlowIndex
// rebuild of the live index allocates nothing once the stores are warm.
//
//nolint:paralleltest // AllocsPerRun reads process-wide counters
func TestPageIndexRebuildReusesStorage(t *testing.T) {
	const contentH = 700.0

	res := layoutHTML(t, `<html><body><p>alpha</p><p>beta</p><p>gamma</p></body></html>`)

	ensureFlowIndex(res, contentH)

	if len(res.flowPageOf) != len(res.Ops) {
		t.Fatalf("first build flowPageOf=%d ops=%d", len(res.flowPageOf), len(res.Ops))
	}

	allocs := testing.AllocsPerRun(20, func() {
		invalidateFlowIndex(res)
		ensureFlowIndex(res, contentH)
	})
	if allocs != 0 {
		t.Fatalf("live index rebuild allocated %.1f allocs/run, want 0", allocs)
	}

	if len(res.flowPageOf) != len(res.Ops) {
		t.Fatalf("rebuild flowPageOf=%d ops=%d", len(res.flowPageOf), len(res.Ops))
	}
}

// TestPageIndexScratchReusesStorage proves repeated forced fresh builds into
// the scratch store allocate nothing after the first.
//
//nolint:paralleltest // AllocsPerRun reads process-wide counters
func TestPageIndexScratchReusesStorage(t *testing.T) {
	const contentH = 700.0

	res := layoutHTML(t, `<html><body><p>alpha</p><p>beta</p><p>gamma</p></body></html>`)

	if pages := pageIndexedOps(res, contentH); len(pages) == 0 {
		t.Fatal("first scratch build produced no pages")
	}

	allocs := testing.AllocsPerRun(20, func() {
		pageIndexedOps(res, contentH)
	})
	if allocs != 0 {
		t.Fatalf("scratch index rebuild allocated %.1f allocs/run, want 0", allocs)
	}
}

// TestPagesDoNotAliasPageIndex proves res.Pages keeps its paint-time buckets
// when the scratch store or the live index is rebuilt afterwards.
func TestPagesDoNotAliasPageIndex(t *testing.T) {
	t.Parallel()

	const contentH = 700.0

	res := layoutHTML(t, `<html><body>
		<p>alpha</p><p>beta</p><p>gamma</p><p>delta</p>
	</body></html>`)

	buildPagesAfterSplits(res, contentH, nil)

	want := make([][]int, len(res.Pages))
	for page := range res.Pages {
		want[page] = append([]int(nil), res.Pages[page]...)
	}

	// A later forced fresh build and a later live rebuild must not reset the
	// buckets res.Pages points at.
	pageIndexedOps(res, contentH)
	invalidateFlowIndex(res)
	ensureFlowIndex(res, contentH)

	if len(res.Pages) != len(want) {
		t.Fatalf("res.Pages len=%d want %d", len(res.Pages), len(want))
	}

	for page := range want {
		if len(res.Pages[page]) != len(want[page]) {
			t.Fatalf("res.Pages[%d] len=%d want %d after rebuild", page, len(res.Pages[page]), len(want[page]))
		}

		for i := range want[page] {
			if res.Pages[page][i] != want[page][i] {
				t.Fatalf("res.Pages[%d][%d]=%d want %d after rebuild", page, i, res.Pages[page][i], want[page][i])
			}
		}
	}
}
