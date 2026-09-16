package convert

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// learncpp's print CSS appends each link URL through
// `.cryout p a::after { content: " (" attr(href) ")" }` (10_style.css:6542).
// The home page opens six <p> tags and closes none, so the lesson-row anchors
// live in <div>s. Browsers implicitly close an open <p> at the next
// block-level start tag; without that rule every lesson-row anchor stays a
// <p> descendant and the URL suffix inflates the print output. An anchor in a
// real (closed) <p> must still get the suffix.
func TestPrintAfterURLSkipsAnchorsOutsideParagraph(t *testing.T) {
	t.Parallel()

	htmlDoc := `<html><head><style>
.cryout p a::after { content: " (" attr(href) ")" }
</style></head><body><div class="cryout">
<p>LearnCpp.com is a free website
<div class="lessontable-row-title"><a href="https://example.com/lesson-1/">Lesson 1</a></div>
<div class="lessontable-row-title"><a href="https://example.com/lesson-2/">Lesson 2</a></div>
<p>See <a href="https://example.com/inside/">this lesson</a> for details.</p>
</div></body></html>`

	cmd, _ := newCommand(t, htmlDoc, "")

	sem, err := pdf.ParseSemantic(runPDF(t, cmd))
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	text := sem.DocumentText()

	for _, url := range []string{"/lesson-1/", "/lesson-2/"} {
		if strings.Contains(text, url) {
			t.Errorf("lesson-row URL %q appended outside <p>; text = %q", url, text)
		}
	}

	if !strings.Contains(text, "/inside/") {
		t.Errorf("closed-<p> anchor URL missing; text = %q", text)
	}
}
