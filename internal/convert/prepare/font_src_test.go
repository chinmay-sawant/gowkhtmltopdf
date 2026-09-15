package prepare

import (
	"strings"
	"testing"
)

func TestFontURIPathIgnoresQueryAndFragment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		uri  string
		want string
	}{
		{"Custom.woff2", "custom.woff2"},
		{"Custom.woff2?v=4.7.0", "custom.woff2"},
		{"../fonts/dashicons.eot#iefix", "../fonts/dashicons.eot"},
		{"Custom.TTF?x=1#y", "custom.ttf"},
		{"https://example.test/fonts/A.WOFF2?v=1", "/fonts/a.woff2"},
	}

	for _, test := range tests {
		if got := fontURIPath(test.uri); got != test.want {
			t.Errorf("fontURIPath(%q) = %q, want %q", test.uri, got, test.want)
		}
	}
}

func TestDataURIMetaStopsAtPayload(t *testing.T) {
	t.Parallel()

	const meta = "data:font/woff;base64"

	uri := meta + "," + strings.Repeat("A", 5000)
	if got := dataURIMeta(uri); got != meta {
		t.Errorf("dataURIMeta = %q, want %q", got, meta)
	}

	long := "data:" + strings.Repeat("a", 100) + ",payload"
	got := dataURIMeta(long)

	if len(got) != 64+3 || !strings.HasSuffix(got, "...") {
		t.Errorf("dataURIMeta long header = %q (len %d), want a 64-char cap plus ellipsis", got, len(got))
	}
}
