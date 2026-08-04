package html

import (
	stdhtml "html"
	"strings"
)

// UnescapeEntities decodes HTML character references in text or attribute
// values (&amp; → &, &#NN; / &#xHH; numeric refs). Uses the stdlib decoder.
func UnescapeEntities(s string) string {
	if s == "" || !strings.Contains(s, "&") {
		return s
	}
	return stdhtml.UnescapeString(s)
}
