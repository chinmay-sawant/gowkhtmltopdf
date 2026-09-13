package imageout

import "testing"

func testPixBuffer(size int) *pixBuffer {
	return &pixBuffer{b: make([]byte, 0, size)}
}

// assertPixBufferCap fails unless get(size) returns a retained buffer with the
// wanted capacity.
func assertPixBufferCap(t *testing.T, cache *pixBufferCache, size, wantCap int) {
	t.Helper()

	got := cache.get(size)
	if got == nil {
		t.Fatalf("get(%d) = nil, want the cap-%d buffer", size, wantCap)
	}

	if cap(got.b) != wantCap {
		t.Fatalf("get(%d) cap = %d, want the cap-%d buffer", size, cap(got.b), wantCap)
	}
}

// TestPixBufferCacheSmallestFit checks that get picks the tightest retained
// buffer and skips ones that do not fit.
func TestPixBufferCacheSmallestFit(t *testing.T) {
	t.Parallel()

	cache := newPixBufferCache(1024, 4096)

	for _, size := range []int{16, 64, 32, 8} {
		cache.put(testPixBuffer(size))
	}

	assertPixBufferCap(t, cache, 20, 32)
	assertPixBufferCap(t, cache, 20, 64)

	if got := cache.get(1000); got != nil {
		t.Fatalf("get(1000) = %v, want nil (no buffer fits)", got)
	}

	assertPixBufferCap(t, cache, 4, 8)

	cache.put(nil)
	cache.put(testPixBuffer(0))

	assertPixBufferCap(t, cache, 0, 16)
}

// TestPixBufferCacheBudgetEviction checks both retention limits: the
// per-buffer cap and the total byte budget bound what the cache keeps.
func TestPixBufferCacheBudgetEviction(t *testing.T) {
	t.Parallel()

	cache := newPixBufferCache(100, 150)

	cache.put(testPixBuffer(60))
	cache.put(testPixBuffer(80))

	if cache.total != 140 {
		t.Fatalf("total after two fitting puts = %d, want 140", cache.total)
	}

	cache.put(testPixBuffer(20))  // would push the total over 150
	cache.put(testPixBuffer(101)) // over the per-buffer cap

	if cache.total != 140 {
		t.Fatalf("total after rejected puts = %d, want 140", cache.total)
	}

	if len(cache.buffers) != 2 {
		t.Fatalf("retained buffers = %d, want 2", len(cache.buffers))
	}

	got := cache.get(20)
	if got == nil || cap(got.b) != 60 {
		t.Fatalf("get(20) = %v, want the cap-60 buffer", got)
	}

	if cache.total != 80 {
		t.Fatalf("total after get = %d, want 80", cache.total)
	}

	got = cache.get(60)
	if got == nil || cap(got.b) != 80 {
		t.Fatalf("get(60) = %v, want the cap-80 buffer", got)
	}

	if got = cache.get(1); got != nil {
		t.Fatalf("get(1) on an empty cache = %v, want nil", got)
	}
}
