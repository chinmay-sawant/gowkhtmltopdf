package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// boxWithClass returns the first box whose element carries the class token.
func boxWithClass(t *testing.T, res *Result, class string) *box {
	t.Helper()

	var found *box

	var walk func(current *box)
	walk = func(current *box) {
		if found != nil {
			return
		}

		if current.node != nil && current.node.Type == html.ElementNode &&
			strings.Contains(" "+current.node.Attribute("class")+" ", " "+class+" ") {
			found = current

			return
		}

		for _, c := range current.children {
			walk(c)
		}
	}
	walk(res.root)

	if found == nil {
		t.Fatalf("no box with class %q in box tree", class)
	}

	return found
}

// layoutLessonRow lays out the learncpp .lessontable-row shape: a fixed 46px
// number badge plus an auto-width title item inside a 460px row, with the
// print-only generated URL suffix enabled by printURL.
func layoutLessonRow(t *testing.T, printURL bool) *Result {
	t.Helper()

	rule := ""
	if printURL {
		rule = `.lessontable-row-title a::after { content: " (" attr(href) ")"; font-size: 80%; word-wrap: break-word }`
	}

	cssSheet := sheet(t, `
	.lessontable-row { display:flex; align-items:center; width:460px; margin:0 }
	.lessontable-row-number { width:46px; min-width:46px; margin:3px 10px 3px 4px; text-align:center }
	.lessontable-row-title { font-size: 11pt }
	`+rule)

	root := mustParse(t, `<html><body>
	<div class="lessontable-row"><div class="lessontable-row-number">0.1</div>
	<div class="lessontable-row-title"><a href="https://www.learncpp.com/cpp-tutorial/`+
		`introduction-to-these-tutorials/">Introduction to these tutorials</a></div></div>
	</body></html>`)

	res, err := Layout(root, Options{
		Width: 500, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Background: true, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// TestFlexLessonRowTitleGetsRemainingWidth: with print CSS appending the link
// URL as ::after content, the title item must be measured with that generated
// run included. Before the fix the base width came from the plain text only
// (~141pt for this row), so the painted URL wrapped inside a ~130pt column and
// every TOC row broke into several lines.
func TestFlexLessonRowTitleGetsRemainingWidth(t *testing.T) {
	t.Parallel()

	withoutURL := layoutLessonRow(t, false)
	withURL := layoutLessonRow(t, true)

	titleWithout := boxWithClass(t, withoutURL, "lessontable-row-title")
	titleWith := boxWithClass(t, withURL, "lessontable-row-title")

	t.Logf("title width: text only=%.2fpt, with generated URL=%.2fpt", titleWithout.w, titleWith.w)

	// Row content width is 460px = 345pt; the 46px badge leaves ~310.5pt.
	const minTitle = 300.0
	if titleWith.w < minTitle {
		t.Fatalf("title width with generated URL = %.2fpt, want >= %.0fpt (badge-only shrink bug)", titleWith.w, minTitle)
	}

	if titleWith.w <= titleWithout.w+50 {
		t.Fatalf("title width grew only %.2f -> %.2f; generated URL not part of the intrinsic measure",
			titleWithout.w, titleWith.w)
	}
	// The generated URL must paint after the element's own inline text.
	texts := textOpsInRange(withURL, titleWith.opStart, titleWith.opEnd)

	titleIdx, urlIdx := -1, -1

	for idx, textOp := range texts {
		if strings.Contains(textOp.Text, "Introduction to these") {
			titleIdx = idx
		}

		if strings.Contains(textOp.Text, "https://www.learncpp.com") {
			urlIdx = idx
		}
	}

	if titleIdx < 0 || urlIdx < 0 {
		t.Fatalf("missing text ops: title=%d url=%d ops=%v", titleIdx, urlIdx, textLabels(texts))
	}

	if urlIdx < titleIdx {
		t.Fatalf("generated ::after URL painted before the anchor text: %v", textLabels(texts))
	}
}

func textOpsInRange(res *Result, start, end int) []Op {
	var out []Op

	for idx := start; idx <= end && idx < len(res.Ops); idx++ {
		if res.Ops[idx].Kind == OpText {
			out = append(out, res.Ops[idx])
		}
	}

	return out
}

func textLabels(ops []Op) []string {
	out := make([]string, 0, len(ops))
	for _, op := range ops {
		out = append(out, op.Text)
	}

	return out
}

// TestPrintAfterURLOrderOnUnquotedHref pins the real learncpp markup: the
// unquoted href ends in '/', so treating the tag as self-closing orphans the
// anchor text and the generated URL paints before the title. With the tag kept
// open, the ::after URL paints after the element's own inline text.
func TestPrintAfterURLOrderOnUnquotedHref(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `.cryout p a::after { content: " (" attr(href) ")" }`)

	root := mustParse(t, `<html><body><div class="cryout"><p><a href=https://example.com/x/>`+
		`Title</a></p></div></body></html>`)

	res, err := Layout(root, Options{
		Width: 400, Height: 300, Sheets: []*css.Stylesheet{cssSheet}, Background: true, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	titleIdx := firstTextOp(res.Ops, "Title")
	urlIdx := firstTextOp(res.Ops, "https://example.com/x/")

	if titleIdx < 0 || urlIdx < 0 {
		t.Fatalf("missing text ops: title=%d url=%d", titleIdx, urlIdx)
	}

	if urlIdx < titleIdx {
		t.Fatalf("generated URL painted at %d before title at %d", urlIdx, titleIdx)
	}
	// Adjacent runs usually merge into one op; the URL must still come after
	// the title text inside it.
	if titleIdx == urlIdx {
		text := res.Ops[titleIdx].Text
		if strings.Index(text, "https://example.com/x/") < strings.Index(text, "Title") {
			t.Fatalf("generated URL precedes the title text in %q", text)
		}
	}
}

// firstTextOp returns the index of the first text op whose run contains
// needle, or -1 when no text op matches.
func firstTextOp(ops []Op, needle string) int {
	for idx, paintOp := range ops {
		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, needle) {
			return idx
		}
	}

	return -1
}
