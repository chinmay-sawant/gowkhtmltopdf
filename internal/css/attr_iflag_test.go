package css //nolint:testpackage // exercises unexported parseAttrSelector and Match

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestAttrIFlag(t *testing.T) {
	t.Parallel()

	t.Run("parse", func(t *testing.T) {
		t.Parallel()

		checkAttrIFlagParse(t)
	})

	t.Run("match", func(t *testing.T) {
		t.Parallel()

		checkAttrIFlagMatch(t)
	})
}

func checkAttrIFlagParse(t *testing.T) {
	t.Helper()

	cases := []struct {
		sel        string
		op         string
		value      string
		ignoreCase bool
		wantOK     bool
	}{
		{sel: "[type=foo i]", op: "=", value: "foo", ignoreCase: true, wantOK: true},
		{sel: "[type=foo I]", op: "=", value: "foo", ignoreCase: true, wantOK: true},
		{sel: `[class="Bar" i]`, op: "=", value: "Bar", ignoreCase: true, wantOK: true},
		{sel: `[class='Bar' I]`, op: "=", value: "Bar", ignoreCase: true, wantOK: true},
		{sel: `[type="foo"i]`, op: "=", value: "foo", ignoreCase: true, wantOK: true},
		{sel: "[type=foo  i]", op: "=", value: "foo", ignoreCase: true, wantOK: true},
		{sel: `[href$=".PDF" i]`, op: "$=", value: ".PDF", ignoreCase: true, wantOK: true},
		{sel: `[class~="Bar" i]`, op: "~=", value: "Bar", ignoreCase: true, wantOK: true},
		{sel: `[class*="Ar" i]`, op: "*=", value: "Ar", ignoreCase: true, wantOK: true},
		{sel: `[class^="Ba" i]`, op: "^=", value: "Ba", ignoreCase: true, wantOK: true},
		{sel: `[lang|="en" i]`, op: "|=", value: "en", ignoreCase: true, wantOK: true},
		{sel: "[type=foo]", op: "=", value: "foo", ignoreCase: false, wantOK: true},
		{sel: "[id]", op: "", value: "", ignoreCase: false, wantOK: true},
		// The Selectors 4 s flag requests the default exact comparison.
		{sel: "[type=foo s]", op: "=", value: "foo", ignoreCase: false, wantOK: true},
		{sel: `[type="foo" s]`, op: "=", value: "foo", ignoreCase: false, wantOK: true},
		{sel: "[attr i]", op: "", value: "", ignoreCase: false, wantOK: false},
	}

	for _, testCase := range cases {
		got, ok := parseAttrSelector(testCase.sel)
		if ok != testCase.wantOK {
			t.Errorf("parseAttrSelector(%q) ok=%v, want %v", testCase.sel, ok, testCase.wantOK)

			continue
		}

		if !testCase.wantOK {
			continue
		}

		if got.Op != testCase.op || got.Value != testCase.value || got.IgnoreCase != testCase.ignoreCase {
			t.Errorf("parseAttrSelector(%q) = {Op:%q Value:%q IgnoreCase:%v}, want {Op:%q Value:%q IgnoreCase:%v}",
				testCase.sel, got.Op, got.Value, got.IgnoreCase, testCase.op, testCase.value, testCase.ignoreCase)
		}
	}
}

func checkAttrIFlagMatch(t *testing.T) {
	t.Helper()

	root := treeFor(t, `<html><body>
		<input id="up" type="FOO">
		<input id="low" type="foo">
		<a id="pdf" href="/files/report.PDF">PDF</a>
		<a id="png" href="/files/report.png">PNG</a>
		<span id="cls" class="Bar baz"></span>
		<span id="en" lang="EN-us"></span>
		<span id="fr" lang="fr"></span>
	</body></html>`)

	ids := []string{"up", "low", "pdf", "png", "cls", "en", "fr"}
	nodes := make(map[string]*html.Node, len(ids))

	for _, attrID := range ids {
		node := byID(root, attrID)
		if node == nil {
			t.Fatalf("missing #%s", attrID)
		}

		nodes[attrID] = node
	}

	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{`[type=foo i]`, "up", true},
		{`[type=foo i]`, "low", true},
		{`[type=FOO i]`, "low", true},
		{`[type=foo]`, "up", false},
		{`[type=foo]`, "low", true},
		{`[type=FOO]`, "low", false},
		{`a[href$=".pdf" i]`, "pdf", true},
		{`a[href$=".pdf" i]`, "png", false},
		{`a[href$=".pdf"]`, "pdf", false},
		{`[class~="BAR" i]`, "cls", true},
		{`[class~="BAR"]`, "cls", false},
		{`[class*="AR" i]`, "cls", true},
		{`[class^="bar" i]`, "cls", true},
		{`[class$="AZ" i]`, "cls", true},
		{`[lang|="en" i]`, "en", true},
		{`[lang|="en" i]`, "fr", false},
		{`[lang|="en"]`, "en", false},
		{`[type=foo s]`, "low", true},
	}

	checkSelectorMatchIDs(t, nodes, cases)
}
