package layout

import (
	"strings"
	"testing"
)

// Programiz search field (real-sites evidence 2026-09-16, programiz-cpp row 10):
// <input class="search-input__control" placeholder="Search tutorials &
// examples"> never reached the text layer. An empty text input has no child
// nodes, so no inline run was ever collected for its value/placeholder.
func TestInputPlaceholderPaintsTextOnce(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><input placeholder="hello"></body></html>`)

	count := 0

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "hello") {
			count++
		}
	}

	if count != 1 {
		t.Fatalf("placeholder text ops = %d, want exactly 1", count)
	}
}

// Value text wins over the placeholder: a text input shows one string, and the
// value is the authored one when both attributes exist.
func TestInputValueBeatsPlaceholder(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><input placeholder="hello" value="world"></body></html>`)

	var sawValue, sawPlaceholder bool

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "world") {
			sawValue = true
		}

		if strings.Contains(paintOp.Text, "hello") {
			sawPlaceholder = true
		}
	}

	if !sawValue {
		t.Fatal("input value text was not painted")
	}

	if sawPlaceholder {
		t.Fatal("placeholder painted alongside a value, want value-only")
	}
}

// Non-text controls keep their own painting path: hidden inputs must not leak
// the value into the text layer, and the checkbox/radio path must not paint
// the attribute text over the box.
func TestInputNonTextTypesSkipPlaceholder(t *testing.T) {
	t.Parallel()

	for _, src := range []string{
		`<html><body><input type="hidden" value="hidden-value"></body></html>`,
		`<html><body><input type="checkbox" placeholder="check-me"></body></html>`,
		`<html><body><input type="radio" placeholder="radio-me"></body></html>`,
	} {
		res := layoutHTML(t, src)

		for _, paintOp := range res.Ops {
			if paintOp.Kind != OpText {
				continue
			}

			if strings.Contains(paintOp.Text, "hidden-value") ||
				strings.Contains(paintOp.Text, "check-me") ||
				strings.Contains(paintOp.Text, "radio-me") {
				t.Fatalf("non-text input %s painted %q", src, paintOp.Text)
			}
		}
	}
}

// The painted placeholder must stay inside the input's content box when the
// author sizes the field, and it must not start past the content-box right
// edge.
func TestInputPlaceholderInsideContentBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><input style="width:80pt" placeholder="hello"></body></html>`)

	var text *Op

	for i := range res.Ops {
		if res.Ops[i].Kind == OpText && strings.Contains(res.Ops[i].Text, "hello") {
			text = &res.Ops[i]

			break
		}
	}

	if text == nil {
		t.Fatal("placeholder text op missing")
	}

	input := findBox(t, res, htmlInput)

	if text.X < input.x-0.01 {
		t.Fatalf("placeholder x=%.2f starts before the input box x=%.2f", text.X, input.x)
	}

	if text.X+text.W > input.x+input.w+0.01 {
		t.Fatalf("placeholder right edge %.2f past the input box right edge %.2f",
			text.X+text.W, input.x+input.w)
	}
}
