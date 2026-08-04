package convert

import (
	"testing"

	"gowkhtmltopdf/internal/layout"
)

func TestResolveRelativeLinkURIs(t *testing.T) {
	ops := []layout.Op{
		{Kind: layout.OpLinkURI, URI: "docs/a.html"},
		{Kind: layout.OpLinkURI, URI: "#frag"},
		{Kind: layout.OpLinkURI, URI: "https://example.com/x"},
		{Kind: layout.OpLinkURI, URI: "mailto:a@b.c"},
	}
	resolveRelativeLinkURIs(ops, "https://example.com/base/page.html")
	if ops[0].URI != "https://example.com/base/docs/a.html" {
		t.Errorf("relative = %q", ops[0].URI)
	}
	if ops[1].URI != "#frag" {
		t.Errorf("fragment mutated: %q", ops[1].URI)
	}
	if ops[2].URI != "https://example.com/x" {
		t.Errorf("absolute mutated: %q", ops[2].URI)
	}
	if ops[3].URI != "mailto:a@b.c" {
		t.Errorf("mailto mutated: %q", ops[3].URI)
	}
}
