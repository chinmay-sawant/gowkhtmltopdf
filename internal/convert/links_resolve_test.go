package convert //nolint:testpackage // white-box tests need unexported access

import (
	"testing"

	"gowkhtmltopdf/internal/layout"
)

func TestResolveRelativeLinkURIs(t *testing.T) {
	t.Parallel()

	ops := []layout.Op{
		{Kind: layout.OpLinkURI, URI: "docs/a.html"},           //nolint:exhaustruct // intentional zero-value fields
		{Kind: layout.OpLinkURI, URI: "#frag"},                 //nolint:exhaustruct // intentional zero-value fields
		{Kind: layout.OpLinkURI, URI: "https://example.com/x"}, //nolint:exhaustruct // intentional zero-value fields
		{Kind: layout.OpLinkURI, URI: "mailto:a@b.c"},          //nolint:exhaustruct // intentional zero-value fields
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
