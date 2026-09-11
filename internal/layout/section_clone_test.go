package layout

import (
	"testing"
)

//nolint:cyclop // table of op identity, Y, and text assertions
func TestCloneSectionChromeYAndText(t *testing.T) {
	t.Parallel()

	src := layoutHTML(t, `<html><body>
		<h1>Header</h1>
		<p>SKU-001</p>
	</body></html>`)

	var texts []string

	for _, op := range src.Ops {
		if op.Kind == OpText {
			texts = append(texts, op.Text)
		}
	}

	if len(texts) < 2 {
		t.Fatalf("text ops = %v", texts)
	}

	replaced := append([]string(nil), texts...)
	replaced[len(replaced)-1] = "SKU-002"

	const yOffset = 700.0

	clone, err := CloneSectionChrome(src, yOffset, replaced...)
	if err != nil {
		t.Fatal(err)
	}

	if len(clone.Ops) != len(src.Ops) {
		t.Fatalf("op count %d vs %d", len(clone.Ops), len(src.Ops))
	}

	textIdx := 0

	for idx := range clone.Ops {
		wantID := uint64(idx) + 1 //nolint:gosec // op index is a small test count
		if clone.Ops[idx].ID != wantID {
			t.Fatalf("op %d ID = %d, want %d", idx, clone.Ops[idx].ID, wantID)
		}

		if src.Ops[idx].Y+yOffset != clone.Ops[idx].Y {
			t.Fatalf("op %d Y = %v, want %v", idx, clone.Ops[idx].Y, src.Ops[idx].Y+yOffset)
		}

		if clone.Ops[idx].Kind != OpText {
			continue
		}

		if clone.Ops[idx].Text != replaced[textIdx] {
			t.Fatalf("text[%d] = %q, want %q", textIdx, clone.Ops[idx].Text, replaced[textIdx])
		}

		textIdx++
	}

	if src.root != nil && clone.root != nil && clone.root.y != src.root.y+yOffset {
		t.Fatalf("root box Y = %v, want %v", clone.root.y, src.root.y+yOffset)
	}
}

func TestCloneSectionChromeTextCountMismatch(t *testing.T) {
	t.Parallel()

	src := layoutHTML(t, `<html><body><p>only</p></body></html>`)

	if _, err := CloneSectionChrome(src, 10, "a", "b"); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestCloneSectionChromeNil(t *testing.T) {
	t.Parallel()

	if _, err := CloneSectionChrome(nil, 0); err == nil {
		t.Fatal("expected error")
	}
}
