package pdf

// Link annotation merging.
//
// Paint emits one link operation per text line, so a link whose text wraps
// across N lines reaches the writer as N AddLinkURI calls that share one URI
// and use vertically adjacent rectangles. Serializing each call emits N
// annotation objects for a single target; on a large index page those objects
// dominate the file. AddLinkURI and AddLinkDest merge a new call into the
// trailing annotation when both calls target the same destination and the
// rectangles are geometrically mergeable, so the merged annotation covers no
// more than the union of the clicks it replaces.
//
// Tagged PDF is excluded on purpose: PDF/UA wires one structure element and
// one /StructParent key to each annotation object, so collapsing annotations
// would drop structure mappings.

// linkMergeGapPt is the largest vertical gap, in points, between two
// rectangles of the same link that still merges into one annotation. Wrapped
// line boxes usually touch or overlap; the gap only absorbs rounding.
const linkMergeGapPt = 2.0

// linkTargetsMatch reports whether two annotations resolve to the same
// destination: the same URI, or the same page and position for internal
// links. A URI annotation never matches a destination annotation.
func linkTargetsMatch(left, right *annotation) bool {
	if left.hasDest != right.hasDest {
		return false
	}

	if left.hasDest {
		return left.destPage == right.destPage && left.destX == right.destX && left.destY == right.destY
	}

	return left.uri != "" && left.uri == right.uri
}

// linkRectsMergeable reports whether two rectangles for the same destination
// can collapse into their union without covering area the individual
// rectangles did not already cover: they overlap horizontally and are
// vertically overlapping or separated by at most linkMergeGapPt.
func linkRectsMergeable(left, right [4]float64) bool {
	if left[2] < right[0] || right[2] < left[0] {
		return false
	}

	if left[3] < right[1] {
		return right[1]-left[3] <= linkMergeGapPt
	}

	if right[3] < left[1] {
		return left[1]-right[3] <= linkMergeGapPt
	}

	return true
}

// unionLinkRect returns the bounding rectangle of a and b.
func unionLinkRect(a, b [4]float64) [4]float64 {
	return [4]float64{
		min(a[0], b[0]),
		min(a[1], b[1]),
		max(a[2], b[2]),
		max(a[3], b[3]),
	}
}
