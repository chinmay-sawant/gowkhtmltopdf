package html_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func FuzzParseHTML(f *testing.F) {
	seeds := []string{
		"<html><body><h1>Hello World</h1></body></html>",
		"<p>Testing &amp; entities <a href=\"https://example.com\">link</a></p>",
		"<div><span>Nested <b>bold</b> <i>italic</i></span></div>",
		"<table><tr><th>Header</th></tr><tr><td>Data</td></tr></table>",
		"<img src=\"test.png\" alt=\"image\" />",
		"<!-- comment --><div>Text with <br/> break</div>",
		"<style>body { color: red; }</style><script>alert(1)</script>",
		"<!DOCTYPE html><html><head><title>Title</title></head><body>Content</body></html>",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, orig string) {
		if len(orig) > 64<<10 {
			return
		}
		// Must not panic on any string input.
		_, _ = html.Parse(orig)
	})
}
