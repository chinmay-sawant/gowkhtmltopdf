package imageout

import "sync"

const (
	// maxPooledRasterBytes caps one retained supersample canvas; larger
	// buffers are dropped instead of pinned. 32 MiB is the retention bound the
	// old sync.Pool used.
	maxPooledRasterBytes = 32 << 20 // 32 MiB

	// maxRasterCacheBytes caps the total retained supersample canvas bytes.
	// Two max-size buffers cover the sequential reuse pattern of one Render
	// per canvas without pinning a large heap across conversions.
	maxRasterCacheBytes = 64 << 20 // 64 MiB
)

// pixBuffer is one reusable supersample canvas backing array.
type pixBuffer struct {
	b []byte
}

// pixBufferCache recycles supersample canvases across rasterizations. Unlike a
// sync.Pool it survives the harness GCs that emptied the old pool, so buffers
// of the current canvas size stay reusable for the next Render.
//
// get returns the smallest retained buffer that fits, which keeps the common
// exact-size case on one buffer. put retains a buffer only while it fits both
// the per-buffer cap and the total byte budget. A mutex guards the list
// because concurrent Renders may rasterize at once.
type pixBufferCache struct {
	mu      sync.Mutex
	buffers []*pixBuffer
	total   int
	perBuf  int
	budget  int
}

func newPixBufferCache(perBuf, budget int) *pixBufferCache {
	return &pixBufferCache{ //nolint:exhaustruct // mutex/list/total start at zero
		perBuf: perBuf,
		budget: budget,
	}
}

// get removes and returns the smallest retained buffer with cap >= needed, or
// nil when none fits.
func (c *pixBufferCache) get(needed int) *pixBuffer {
	c.mu.Lock()
	defer c.mu.Unlock()

	best := -1
	bestCap := 0

	for i, pBuf := range c.buffers {
		size := cap(pBuf.b)
		if size >= needed && (best < 0 || size < bestCap) {
			best = i
			bestCap = size
		}
	}

	if best < 0 {
		return nil
	}

	pBuf := c.buffers[best]
	last := len(c.buffers) - 1
	c.buffers[best] = c.buffers[last]
	c.buffers[last] = nil
	c.buffers = c.buffers[:last]
	c.total -= cap(pBuf.b)

	return pBuf
}

// put retains pBuf when it fits the per-buffer cap and the total byte budget,
// and drops it otherwise.
func (c *pixBufferCache) put(pBuf *pixBuffer) {
	if pBuf == nil {
		return
	}

	size := cap(pBuf.b)
	if size == 0 || size > c.perBuf {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.total+size > c.budget {
		return
	}

	c.buffers = append(c.buffers, pBuf)
	c.total += size
}

// stats reports the retained buffer count, total retained bytes, and the
// largest retained capacity. It takes the cache lock so callers can inspect
// retention while concurrent Renders may be running.
func (c *pixBufferCache) stats() (int, int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	total := 0
	largest := 0

	for _, pBuf := range c.buffers {
		size := cap(pBuf.b)
		total += size

		if size > largest {
			largest = size
		}
	}

	return len(c.buffers), total, largest
}

// supersamplePixCache recycles supersample canvases between rasterizations.
//
//nolint:gochecknoglobals // supersample canvas recycling
var supersamplePixCache = newPixBufferCache(maxPooledRasterBytes, maxRasterCacheBytes)
