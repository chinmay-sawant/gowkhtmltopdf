package layout

// boxKind is the internal layout role of a box. uint8 avoids a per-box
// string header (16 bytes) and keeps the hot box struct small.
type boxKind uint8

const (
	boxKindBlock    boxKind = iota // "block"
	boxKindTable                   // "table"
	boxKindCell                    // "cell"
	boxKindReplaced                // "replaced"
)

func (k boxKind) String() string {
	switch k {
	case boxKindBlock:
		return displayBlock
	case boxKindTable:
		return displayTable
	case boxKindCell:
		return tableCellKind
	case boxKindReplaced:
		return "replaced"
	default:
		return "unknown"
	}
}
