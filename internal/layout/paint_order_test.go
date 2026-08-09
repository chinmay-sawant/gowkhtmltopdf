package layout

import "testing"

func TestPaintOrderSharedPolicyKeepsStableMetadataOrder(t *testing.T) {
	ops := []Op{
		{Kind: OpText, ZIndexSet: true},      //nolint:exhaustruct // ordering-only operation
		{Kind: OpFillRect, ZIndexSet: true},  //nolint:exhaustruct // ordering-only operation
		{Kind: OpText, ZIndexSet: true},       //nolint:exhaustruct // ordering-only operation
		{Kind: OpFillRect, ZIndexSet: true},   //nolint:exhaustruct // ordering-only operation
		{Kind: OpLinkURI, ZIndexSet: true},   //nolint:exhaustruct // metadata-only operation
	}
	// The test intentionally sets the actual z-index values below after the
	// compact literals keep the operation kinds easy to scan.
	ops[0].ZIndex = 0
	ops[1].ZIndex = 0
	ops[2].ZIndex = 2
	ops[3].ZIndex = -1
	ops[4].ZIndex = 0

	if got, want := PaintOrder(ops), []int{3, 1, 0, 4, 2}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want %v", got, want)
	}

	subset := []int{4, 0, 1, 3}
	if got, want := paintOrderSubset(ops, subset), []int{3, 1, 0, 4}; !sameIndices(got, want) {
		t.Fatalf("subset paint order = %v, want %v", got, want)
	}
}

func sameIndices(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
