package layout

import (
	"strings"
	"testing"
)

// learn-cpp-org-5: Font Awesome renders every icon as an empty host element
// (often <i class="fas fa-play">) whose ::before carries the glyph, and .fas
// makes the host display:inline-block. The inline-block build must still lay
// out the generated content; before the fix the block path only injected
// ::before/::after around an existing inline run, so empty hosts painted
// nothing.

func TestEmptyInlineBlockPseudoContentPaints(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 16px }
.fas { display: inline-block }
.fa-play:before { content: "\f04b" }
`)

	res := layoutHTML(t,
		`<html><body><p>before<i class="fas fa-play"></i>after</p></body></html>`,
		cssSheet)

	found := false

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "\uf04b") {
			found = true

			break
		}
	}

	if !found {
		t.Fatal("no text op carries U+F04B; the empty inline-block host's " +
			"::before glyph was dropped")
	}
}

// Control: the same glyph on a display:inline host keeps painting.
func TestEmptyInlinePseudoContentPaints(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 16px }
.fa-play:before { content: "\f04b" }
`)

	res := layoutHTML(t,
		`<html><body><p>before<span class="fa-play"></span>after</p></body></html>`,
		cssSheet)

	found := false

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "\uf04b") {
			found = true

			break
		}
	}

	if !found {
		t.Fatal("no text op carries U+F04B for the inline host")
	}
}
