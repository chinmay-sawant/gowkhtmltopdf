package layout

import (
	"encoding/binary"
	"errors"
	"hash"
	"hash/fnv"
	"io"
	"unsafe"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// ErrSectionClone is returned when a section cannot be cloned (text count
// mismatch, missing source, or a shape convert should full-layout instead).
var ErrSectionClone = errors.New("layout: section chrome clone not applicable")

// SectionChromeHash fingerprints a section's element tree with text nodes as
// holes and resolved-style pointer identity per element. Sections that share
// a hash are a clone group.
func SectionChromeHash(node *html.Node, styles map[*html.Node]*ResolvedStyle) uint64 {
	h := fnv.New64a()
	hashSectionChrome(h, node, styles)

	return h.Sum64()
}

func hashSectionChrome(sum hash.Hash64, node *html.Node, styles map[*html.Node]*ResolvedStyle) {
	if node == nil {
		return
	}

	if node.Type == html.TextNode {
		_, _ = sum.Write([]byte{0})

		return
	}

	if node.Type != html.ElementNode {
		for _, child := range node.Children {
			hashSectionChrome(sum, child, styles)
		}

		return
	}

	_, _ = io.WriteString(sum, node.Name)

	if sty := styles[node]; sty != nil {
		var buf [8]byte

		binary.LittleEndian.PutUint64(buf[:], uint64(uintptr(unsafe.Pointer(sty))))
		_, _ = sum.Write(buf[:])
	} else {
		_, _ = sum.Write([]byte{1})
	}

	for _, child := range node.Children {
		hashSectionChrome(sum, child, styles)
	}
}

// CloneSectionChrome copies src's ops and boxes, adds yOffset to Y, replaces
// OpText strings in document order from uniqueTexts, and assigns new Op.IDs.
// uniqueTexts must match the number of OpText entries or be empty (keep text).
// Convert should full-layout the section when this returns an error.
//
//nolint:cyclop // count, clone, translate, and re-id is one transform
func CloneSectionChrome(src *Result, yOffset float64, uniqueTexts ...string) (*Result, error) {
	if src == nil {
		return nil, ErrSectionClone
	}

	textIdx := 0

	for i := range src.Ops {
		if src.Ops[i].Kind == OpText {
			textIdx++
		}
	}

	if len(uniqueTexts) != 0 && len(uniqueTexts) != textIdx {
		return nil, ErrSectionClone
	}

	clone := CloneResult(src)
	nextID := uint64(1)
	textIdx = 0

	for i := range clone.Ops {
		clonedOp := &clone.Ops[i]
		clonedOp.ID = nextID
		nextID++

		if clonedOp.Kind == OpGridRun && clonedOp.Grid != nil {
			clonedOp.Grid = &GridRun{Segs: append([]GridSeg(nil), clonedOp.Grid.Segs...)}
		}

		shiftOpY(clonedOp, yOffset)

		if clonedOp.Kind == OpText && len(uniqueTexts) > 0 {
			clonedOp.Text = uniqueTexts[textIdx]
			textIdx++
		}
	}

	shiftClonedBoxesY(clone.root, yOffset)

	for i := range clone.Locations {
		clone.Locations[i].Y += yOffset
	}

	clone.Height += yOffset

	return clone, nil
}

func shiftClonedBoxesY(boxNode *box, yOffset float64) {
	if boxNode == nil || yOffset == 0 {
		return
	}

	boxNode.y += yOffset

	for _, child := range boxNode.children {
		shiftClonedBoxesY(child, yOffset)
	}

	for _, row := range boxNode.rows {
		for _, cell := range row {
			shiftClonedBoxesY(cell, yOffset)
		}
	}

	if boxNode.stickyPort != nil {
		shiftClonedBoxesY(boxNode.stickyPort, yOffset)
	}
}
