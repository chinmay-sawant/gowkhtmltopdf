package layout

import "sort"

// PaintOrder returns operation indices in the canonical display-list paint
// order. PDF and raster adapters consume this same policy; backend drawing
// remains responsible for interpreting each operation.
func PaintOrder(ops []Op) []int {
	idx := make([]int, len(ops))
	for i := range ops {
		idx[i] = i
	}

	return paintOrderSubset(ops, idx)
}

// paintOrderSubset returns an ordered copy of an operation subset. All
// adapters use this policy so a band cannot accidentally fall back to source
// order while the paginated body uses paint order.
func paintOrderSubset(ops []Op, idxs []int) []int {
	ordered := append([]int(nil), idxs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return paintOrderBefore(ops, ordered[i], ordered[j])
	})

	return ordered
}

// sortPaintIndices sorts an existing operation subset without changing which
// operations it contains. Pagination tests use this compatibility helper for
// fixed and per-page lists; the comparison itself lives in paintOrderSubset.
func sortPaintIndices(ops []Op, idxs []int) {
	copy(idxs, paintOrderSubset(ops, idxs))
}

func paintOrderBefore(ops []Op, left, right int) bool {
	leftOp, rightOp := &ops[left], &ops[right]

	leftZ, rightZ := 0, 0
	if leftOp.ZIndexSet {
		leftZ = leftOp.ZIndex
	}
	if rightOp.ZIndexSet {
		rightZ = rightOp.ZIndex
	}

	if leftZ != rightZ {
		return leftZ < rightZ
	}

	if leftOp.Positioned != rightOp.Positioned {
		return !leftOp.Positioned
	}

	leftLayer, rightLayer := paintLayer(leftOp.Kind), paintLayer(rightOp.Kind)
	if leftLayer != rightLayer {
		return leftLayer < rightLayer
	}

	return left < right
}

// paintLayer orders ops within a z-index band: chrome under content.
func paintLayer(k OpKind) int {
	switch k {
	case OpFillRect, OpStrokeRect, OpLine:
		return 0
	case OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
		return 1
	}

	return 1
}
