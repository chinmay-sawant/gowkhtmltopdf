package layout

import (
	"testing"
	"unsafe"
)

func TestOpSizePacked(t *testing.T) {
	t.Parallel()

	sz := unsafe.Sizeof(Op{})
	t.Log(sz)

	if sz > 256 {
		t.Fatalf("Op size = %d, want <= 256", sz)
	}
}
