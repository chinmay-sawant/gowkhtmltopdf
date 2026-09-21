package pdf

import (
	"bytes"
	"runtime"
	"testing"
)

// TestFlateStateRetainedAcrossGC pins the fix for the 2026-09-21 Chrome
// corpus capture. The old serial sync.Pool lost its state within two GC
// cycles, so the next flateBytes call reallocated a fresh deflate state of
// roughly 817 KiB. The retained state must survive; only small buffer copies
// are allowed in the measured call.
//
//nolint:paralleltest // the MemStats delta must be measured without sibling parallel tests
func TestFlateStateRetainedAcrossGC(t *testing.T) {
	raw := bytes.Repeat([]byte("retained flate state probe "), 4096)

	_ = flateBytes(raw)

	runtime.GC()
	runtime.GC()

	var before, after runtime.MemStats

	runtime.ReadMemStats(&before)

	_ = flateBytes(raw)

	runtime.ReadMemStats(&after)

	const maxReuseAllocation = 256 << 10

	if delta := after.TotalAlloc - before.TotalAlloc; delta > maxReuseAllocation {
		t.Fatalf("flateBytes after two GC cycles allocated %d bytes, want at most %d (a fresh state is about 817 KiB)",
			delta, maxReuseAllocation)
	}
}
