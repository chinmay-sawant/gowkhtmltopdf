//nolint:wsl // fixture contract checks keep source markers adjacent
package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//nolint:gochecknoglobals // immutable fixture metadata is shared by subtests
var chromeFixtureContracts13To20 = []struct {
	name    string
	fixture string
	markers []string
}{
	{
		name:    "legacy-flex-flow-auto-margins",
		fixture: "case-13-legacy-flex-flow-auto-margins.html",
		markers: []string{
			"flex-flow-auto-margins.html",
			".physical > div { margin: 13px auto 17px auto; }",
			"margin-inline-start: auto",
			"vertical-lr ltr row logical",
			"vertical-rl rtl column-reverse logical",
		},
	},
	{
		name:    "legacy-flex-align-baseline",
		fixture: "case-14-legacy-flex-align-baseline.html",
		markers: []string{
			"flex-align-baseline.html",
			"align-items: baseline",
			"margin-top:20px",
			"vertical-lr ltr row",
			"vertical-rl rtl column-reverse",
		},
	},
	{
		name:    "wpt-flex-basis-011",
		fixture: "case-17-wpt-flex-basis-011.html",
		markers: []string{
			"flex-basis-011.html",
			"flex: 1 0 100%",
			"class=\"flexbox column\"",
			"<div>AAA</div>",
			"<div>BBB</div>",
		},
	},
	{
		name:    "wpt-rtl-flow-reverse",
		fixture: "case-18-wpt-rtl-flow-reverse.html",
		markers: []string{
			"flexbox_rtl-flow-reverse.html",
			"flex-flow: column wrap-reverse",
			"direction: rtl",
			"class=\"span-one\">one</span>",
			"class=\"span-four\">four</span>",
		},
	},
	{
		name:    "wpt-min-size-auto-overflow-clip",
		fixture: "case-20-wpt-min-size-auto-overflow-clip.html",
		markers: []string{
			"min-size-auto-overflow-clip.html",
			"overflow:clip",
			"width: 100px",
			"width:150px;height:50px",
			"min-size-auto-overflow-clip-ref.html",
		},
	},
}

// TestChromeCases13To20FixtureContracts keeps the five investigated fixtures
// tied to their Chromium interaction. Their layout assertions live in the
// focused geometry tests.
func TestChromeCases13To20FixtureContracts(t *testing.T) {
	t.Parallel()

	for _, tc := range chromeFixtureContracts13To20 {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertChromeFixtureContract(t, tc.fixture, tc.markers)
		})
	}
}

func assertChromeFixtureContract(t *testing.T, fixture string, markers []string) {
	t.Helper()

	path := filepath.Join("..", "..", "test", "chrome", "cases", fixture)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	text := string(source)
	if !strings.HasPrefix(text, "<!doctype html>") {
		t.Fatal("fixture must start with a doctype")
	}
	if !strings.Contains(text, "Port status: completed") {
		t.Fatal("fixture must record a completed layout gate")
	}
	if strings.Contains(text, "<script") {
		t.Fatal("fixture must remain a static HTML input")
	}

	for _, marker := range markers {
		if !strings.Contains(text, marker) {
			t.Errorf("fixture is missing source marker %q", marker)
		}
	}
}
