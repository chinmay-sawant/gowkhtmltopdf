package layout

import (
	"testing"
	"unsafe"
)

// TestPerf3OpSizePin is the PERF3-02 ceiling on the display-list record.
// 432 bytes is the size on this tree after PERFT-11 packing. Later
// compaction may shrink Op; growth above 432 fails the pin.
func TestPerf3OpSizePin(t *testing.T) {
	t.Parallel()

	sz := unsafe.Sizeof(Op{})
	t.Log(sz)

	if sz > 432 {
		t.Fatalf("Op grew: %d", sz)
	}
}
