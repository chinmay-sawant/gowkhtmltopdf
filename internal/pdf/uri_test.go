package pdf

import (
	"strings"
	"testing"
)

// TestURIStringPercentEncoding locks the /URI action encoding: non-ASCII
// targets are percent-encoded from their UTF-8 bytes (RFC 3986 says URI bytes
// are ASCII) and already-encoded hrefs pass through unchanged. The en dash
// case is the w3schools/Wikipedia regression: pdfString folded U+2013 to
// PDFDocEncoding byte 0x96, which readers decode as U+0152 (OE).
func TestURIStringPercentEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "en dash rune",
			uri: "https://en.wikipedia.org/wiki/Golden_Globe_Award_for_Best_Actress_in_a_Motion_Picture_" +
				"\u2013_Musical_or_Comedy",
			want: "(https://en.wikipedia.org/wiki/Golden_Globe_Award_for_Best_Actress_in_a_Motion_Picture_" +
				"%E2%80%93_Musical_or_Comedy)",
		},
		{
			name: "accented rune",
			uri:  "https://en.wikipedia.org/wiki/Province_of_Le\u00f3n",
			want: "(https://en.wikipedia.org/wiki/Province_of_Le%C3%B3n)",
		},
		{
			name: "already percent-encoded en dash",
			uri:  "https://en.wikipedia.org/wiki/Satellite_Award_for_Best_Cast_%E2%80%93_Motion_Picture",
			want: "(https://en.wikipedia.org/wiki/Satellite_Award_for_Best_Cast_%E2%80%93_Motion_Picture)",
		},
		{
			name: "already percent-encoded accents and literal parens",
			uri:  "https://en.wikipedia.org/wiki/%C3%81lex_Gonz%C3%A1lez_(actor)",
			want: "(https://en.wikipedia.org/wiki/%C3%81lex_Gonz%C3%A1lez_\\(actor\\))",
		},
		{
			name: "plain ASCII",
			uri:  "https://example.com/report",
			want: "(https://example.com/report)",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := uriString(testCase.uri); got != testCase.want {
				t.Errorf("uriString(%q) = %q, want %q", testCase.uri, got, testCase.want)
			}
		})
	}
}

// TestLinkURIAnnotPercentEncoded proves the annotation writer uses the URI
// encoder: the en dash must reach the PDF as %E2%80%93, never as the
// PDFDocEncoding byte 0x96 (written \226) that readers show as U+0152.
func TestLinkURIAnnotPercentEncoded(t *testing.T) {
	t.Parallel()

	const uri = "https://en.wikipedia.org/wiki/Satellite_Award_for_Best_Cast_\u2013_Motion_Picture"

	doc := fixedDoc(t)
	doc.SetCompression(false)

	page := doc.AddPage(200, 200)
	page.AddLinkURI([4]float64{10, 10, 120, 30}, uri)

	out := string(writePDF(t, doc))

	const wantURI = "/URI (https://en.wikipedia.org/wiki/Satellite_Award_for_Best_Cast_%E2%80%93_Motion_Picture)"
	if !strings.Contains(out, wantURI) {
		t.Errorf("URI action is not percent-encoded, want %q:\n%s", wantURI, out)
	}

	if strings.Contains(out, `\226`) {
		t.Error("URI action carries the PDFDocEncoding en dash byte readers decode as U+0152")
	}
}
