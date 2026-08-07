package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
)

// Narrow figcaptions must wrap at spaces, not mid-word, when each word fits
// the caption box (emergency mid-word only if a single token exceeds width).
func TestCaptionPrefersWordBoundaries(t *testing.T) {
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	cssSheet := sheet(t, `
body { margin: 0; font-size: 9pt; }
figure { float: right; width: 120pt; margin: 0; }
img { display: block; width: 120pt; height: 80pt; }
figcaption { font-size: 8pt; width: 120pt; }
`)
	// Words that each fit ~120pt at 8–9pt but together need multiple lines.
	cap := "De Armas at the San Sebastian International Film Festival in 2022"
	root := mustParse(t, `<html><body>
<figure>
<img width="120" height="80" src="t.png">
<figcaption>`+cap+`</figcaption>
</figure>
<p>Body text beside the float.</p>
</body></html>`)

	out, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 400, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		Images: func(string) ([]byte, error) { return png, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	var texts []string

	for _, op := range out.Ops {
		if op.Kind == OpText && !strings.Contains(op.Text, "Body") {
			// Caption-ish ops (exclude body).
			if op.X > 200 { // float:right caption on the right half
				texts = append(texts, op.Text)
			}
		}
	}

	if len(texts) == 0 {
		// Fallback: collect any non-body text.
		for _, op := range out.Ops {
			if op.Kind == OpText && !strings.Contains(op.Text, "Body") {
				texts = append(texts, op.Text)
			}
		}
	}

	joined := strings.Join(texts, "")
	if !strings.Contains(joined, "International") && !strings.Contains(joined, "International") {
		// Allow soft line break that keeps the word intact across ops only if
		// whole "International" appears concatenated.
		t.Fatalf("caption missing International: %v", texts)
	}
	// No mid-word fragment of International / Sebastian / Festival alone as a
	// 1–3 letter orphan that is not a whole word.
	banned := []string{"Int", "ernational", "Sebas", "tián", "Festi", "val"}
	// "Int" alone as an op is the classic bad mid-break; whole words OK.
	for _, op := range texts {
		tok := strings.TrimSpace(op)
		for _, b := range banned {
			if tok == b {
				t.Fatalf("mid-word caption break %q in ops %v", tok, texts)
			}
		}
	}
	// Positive: at least one op should contain a full long word.
	okWord := false

	for _, op := range texts {
		if strings.Contains(op, "International") || strings.Contains(op, "Sebastian") ||
			strings.Contains(op, "Festival") {
			okWord = true

			break
		}
	}

	if !okWord {
		t.Fatalf("expected intact long words in caption ops, got %v", texts)
	}
}

// overflow-wrap:break-word must not mid-break a word that fits the full line
// width just because remaining space on the current line is tight.
func TestBreakWordDoesNotMidBreakFittingWord(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
p { margin: 0; width: 140pt; overflow-wrap: break-word; }
`)
	// "International" fits 140pt; after "The " remainW is tight enough that a
	// greedy mid-break would split it — we must wrap the whole word.
	res := layoutHTML(t, `<html><body><p style="overflow-wrap:break-word">The International festival</p></body></html>`, cssSheet)

	var texts []string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts = append(texts, op.Text)
		}
	}

	joined := strings.Join(texts, "")
	if !strings.Contains(joined, "International") {
		// Word may be a single op.
		found := false

		for _, tx := range texts {
			if strings.Contains(tx, "International") {
				found = true
			}
		}

		if !found {
			t.Fatalf("missing International in %v", texts)
		}
	}

	for _, tx := range texts {
		tok := strings.TrimSpace(tx)
		if tok == "Int" || tok == "ernational" || tok == "Internationa" {
			t.Fatalf("mid-word break %q with break-word when word fits line: %v", tok, texts)
		}
	}
}
