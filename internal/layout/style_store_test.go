//nolint:all
//nolint:testpackage // tests exercise unexported styleStore internals
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

type styleStoreBenchmarkRow struct {
	Number, SKU, Description, Quantity, Amount string
}

type styleStoreBenchmarkPage struct {
	First  bool
	Number int
	Rows   []styleStoreBenchmarkRow
}

func benchmarkTemplateSheet(t *testing.T, root *html.Node) *css.Stylesheet {
	t.Helper()

	var source string

	root.Walk(func(node *html.Node) {
		if source == "" && node.Type == html.ElementNode && node.Name == styleElement {
			source = node.TextContent()
		}
	})

	if source == "" {
		t.Fatal("benchmark template style block missing")
	}

	return sheet(t, source)
}

func benchmarkTemplateStyleNodes(root *html.Node) ([]*html.Node, []*html.Node, []*html.Node, []*html.Node) {
	plainCells := make([]*html.Node, 0)
	amountCells := make([]*html.Node, 0)
	headers := make([]*html.Node, 0)
	rows := make([]*html.Node, 0)

	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}

		switch node.Name {
		case "td":
			if node.Attribute("class") == "amount" {
				amountCells = append(amountCells, node)
			} else {
				plainCells = append(plainCells, node)
			}
		case "th":
			headers = append(headers, node)
		case "tr":
			rows = append(rows, node)
		}
	})

	return plainCells, amountCells, headers, rows
}

func assertAllStylePointersMatch(
	t *testing.T, styles map[*html.Node]*ResolvedStyle, nodes []*html.Node, kind string,
) {
	t.Helper()

	if len(nodes) < 2 {
		t.Fatalf("%s nodes=%d, want at least two", kind, len(nodes))
	}

	first := styles[nodes[0]]
	for _, node := range nodes[1:] {
		if styles[node] != first {
			t.Fatalf("repeated %s did not share a canonical style", kind)
		}
	}
}

func assertStyleStoreFontBoundary(
	t *testing.T, styles map[*html.Node]*ResolvedStyle, nodes map[string]*html.Node,
) {
	t.Helper()

	if styles[nodes["base"]] == styles[nodes["large"]] {
		t.Fatal("inherited font-size change reused the base style")
	}

	if styles[nodes["large"]].FontSize <= styles[nodes["base"]].FontSize {
		t.Fatalf(
			"large font size=%.1f, base=%.1f",
			styles[nodes["large"]].FontSize,
			styles[nodes["base"]].FontSize,
		)
	}
}

func assertStyleStoreSelectorBoundaries(
	t *testing.T, styles map[*html.Node]*ResolvedStyle, nodes map[string]*html.Node,
) {
	t.Helper()

	if styles[nodes["amount-one"]] == styles[nodes["plain"]] || styles[nodes["amount-one"]].TextAlign != "right" {
		t.Fatal("td.amount selector-specific alignment was not isolated")
	}

	if styles[nodes["inline"]] == styles[nodes["amount-one"]] || styles[nodes["inline"]].TextAlign != "center" {
		t.Fatal("inline declaration was not isolated from selector style")
	}
}

func assertStyleStorePolicyAndCustomBoundaries(
	t *testing.T, styles map[*html.Node]*ResolvedStyle, nodes map[string]*html.Node,
) {
	t.Helper()

	linkStyle := styles[nodes["link"]]
	plainAnchorStyle := styles[nodes["no-href"]]

	if linkStyle == plainAnchorStyle || linkStyle.TextDecoration != cssTextDecorationUnderline {
		t.Fatal("print-link policy did not isolate href anchor style")
	}

	if styles[nodes["custom-one"]] == styles[nodes["custom-two"]] ||
		styles[nodes["custom-one"]].CustomProps == nil || styles[nodes["custom-two"]].CustomProps == nil {
		t.Fatal("custom-property boundary was interned or lost")
	}
}

func assertStyleStoreShares(
	t *testing.T, styles map[*html.Node]*ResolvedStyle, nodes map[string]*html.Node, left, right string,
) {
	t.Helper()

	if styles[nodes[left]] != styles[nodes[right]] {
		t.Fatalf("styles for %q and %q should share a canonical pointer", left, right)
	}
}

func assertStyleStoreDiffers(
	t *testing.T, styles map[*html.Node]*ResolvedStyle, nodes map[string]*html.Node, left, right string,
) {
	t.Helper()

	if styles[nodes[left]] == styles[nodes[right]] {
		t.Fatalf("styles for %q and %q unexpectedly share a canonical pointer", left, right)
	}
}

func styleStoreNodesByID(root *html.Node) map[string]*html.Node {
	nodes := map[string]*html.Node{}

	root.Walk(func(node *html.Node) {
		if id := node.Attribute("id"); id != "" {
			nodes[id] = node
		}
	})

	return nodes
}
