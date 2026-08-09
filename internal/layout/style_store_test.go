//nolint:testpackage // tests exercise unexported styleStore internals
package layout

import (
	"bytes"
	"testing"
	"text/template"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

type styleStoreBenchmarkRow struct {
	Number, SKU, Description, Quantity, Amount string
}

type styleStoreBenchmarkPage struct {
	First  bool
	Number int
	Rows   []styleStoreBenchmarkRow
}

func TestStyleStoreInterningPolicy(t *testing.T) {
	t.Parallel()

	store := styleStore{} //nolint:exhaustruct // intentional zero-value store
	base := initialStyle()
	first := store.intern(base)

	if got := store.intern(base); got != first {
		t.Fatal("equivalent style did not reuse its canonical pointer")
	}

	different := base
	different.Color = [3]float64{1, 0, 0}

	if got := store.intern(different); got == first {
		t.Fatal("semantically different style reused its canonical pointer")
	}

	differentMargin := base
	differentMargin.MarginTop = 4

	if got := store.intern(differentMargin); got == first {
		t.Fatal("difference outside the coarse key reused its canonical pointer")
	}

	withCustomProps := base
	withCustomProps.CustomProps = map[string]string{"--accent": "blue"}
	firstCustom := store.intern(withCustomProps)

	if got := store.intern(withCustomProps); got == firstCustom {
		t.Fatal("style with custom properties was interned")
	}
}

func TestStyleStorePointersRemainStableAcrossChunks(t *testing.T) {
	t.Parallel()

	store := styleStore{} //nolint:exhaustruct // intentional zero-value store
	base := initialStyle()
	first := store.intern(base)

	for i := range styleStoreChunkSize * 3 {
		candidate := base
		candidate.Width = float64(i)

		store.intern(candidate)
	}

	if first != store.intern(base) {
		t.Fatal("canonical pointer changed after chunk growth")
	}

	if first.Display != base.Display || first.Width != base.Width {
		t.Fatal("canonical style changed after chunk growth")
	}
}

func TestStyleStoreSharesEquivalentCascadeResults(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
<div id="first" class="same">first</div><div id="second" class="same">second</div>
</body></html>`)
	styles := resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{sheet(t, `.same { color:#123456; margin:2pt }`)},
	}, nil)
	nodes := styleStoreNodesByID(root)

	if styles[nodes["first"]] != styles[nodes["second"]] {
		t.Fatal("equivalent cascaded elements did not share their canonical style")
	}
}

func TestStyleStoreKeepsCascadeBoundariesDistinct(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
<span id="base">base</span><div class="large"><span id="large">large</span></div>
<table><tr>
  <td id="amount-one" class="amount">1</td><td id="amount-two" class="amount">2</td>
  <td id="plain">plain</td><td id="inline" style="text-align:center">inline</td>
</tr></table>
<a id="link" href="#target">link</a><a id="no-href">plain anchor</a>
<div style="--accent:#ff0000"><span id="custom-one" class="accent">one</span></div>
<div style="--accent:#ff0000"><span id="custom-two" class="accent">two</span></div>
</body></html>`)
	styles := resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{sheet(t, `
.large { font-size:20pt }
td.amount { text-align:right }
.accent { color:var(--accent) }
a { text-decoration:none }
`)},
		Media: "print", PrintLinkUnderline: true,
	}, nil)
	nodes := styleStoreNodesByID(root)

	if styles[nodes["amount-one"]] != styles[nodes["amount-two"]] {
		t.Fatal("repeated td.amount styles did not share their canonical style")
	}

	assertStyleStoreFontBoundary(t, styles, nodes)
	assertStyleStoreSelectorBoundaries(t, styles, nodes)
	assertStyleStorePolicyAndCustomBoundaries(t, styles, nodes)
}

func TestStyleStoreSelectorContextsRespectResolvedEquality(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
<div id="id-match">id</div><div id="attr-match" data-tone="warm">attribute</div><div id="plain">plain</div>
<ul><li id="nth-one">one</li><li id="nth-two">two</li></ul>
<ul><li id="nth-three">three</li><li id="nth-four">four</li></ul>
<div><p id="sibling-one">one</p><p id="sibling-two">two</p></div>
<div><p id="sibling-three">three</p><p id="sibling-four">four</p></div>
<article id="has-one"><span class="flag">one</span></article>
<article id="has-two"><span class="flag">two</span></article><article id="has-none">none</article>
</body></html>`)
	styles := resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{sheet(t, `
#id-match { color:red }
[data-tone="warm"] { color:red }
li:nth-child(2) { color:blue }
p + p { margin-left:5pt }
article:has(.flag) { color:green }
`)},
	}, nil)
	nodes := styleStoreNodesByID(root)

	assertStyleStoreShares(t, styles, nodes, "id-match", "attr-match")
	assertStyleStoreDiffers(t, styles, nodes, "id-match", "plain")
	assertStyleStoreShares(t, styles, nodes, "nth-two", "nth-four")
	assertStyleStoreDiffers(t, styles, nodes, "nth-one", "nth-two")
	assertStyleStoreShares(t, styles, nodes, "sibling-two", "sibling-four")
	assertStyleStoreDiffers(t, styles, nodes, "sibling-one", "sibling-two")
	assertStyleStoreShares(t, styles, nodes, "has-one", "has-two")
	assertStyleStoreDiffers(t, styles, nodes, "has-one", "has-none")
}

func TestStyleStoreMediaAndFontFamilyOrder(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
<span id="print-one" class="print">one</span><span id="print-two" class="print">two</span>
<span id="screen" class="screen">screen</span>
<span id="family-ab" class="family-ab">ab</span><span id="family-ba" class="family-ba">ba</span>
</body></html>`)
	styles := resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Media: "print", Sheets: []*css.Stylesheet{sheet(t, `
@media print { .print { color:red } }
@media screen { .screen { color:blue } }
.family-ab { font-family: Alpha, Beta, sans-serif }
.family-ba { font-family: Beta, Alpha, sans-serif }
`)},
	}, nil)
	nodes := styleStoreNodesByID(root)

	assertStyleStoreShares(t, styles, nodes, "print-one", "print-two")
	assertStyleStoreDiffers(t, styles, nodes, "print-one", "screen")
	assertStyleStoreDiffers(t, styles, nodes, "family-ab", "family-ba")

	if styles[nodes["family-ab"]].FontFamily[0] != "Alpha" || styles[nodes["family-ba"]].FontFamily[0] != "Beta" {
		t.Fatalf(
			"font-family order lost: ab=%v ba=%v",
			styles[nodes["family-ab"]].FontFamily,
			styles[nodes["family-ba"]].FontFamily,
		)
	}
}

func TestStyleStoreContainerMatchAndMiss(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.card { container:card / inline-size; font-size:12pt }
.wide { width:400px }.narrow { width:100px }
@container card (inline-size > 20em) { .title { color:red } }
`)
	root := mustParse(t, `<html><body>
<div class="card wide"><p id="wide-one" class="title">one</p></div>
<div class="card wide"><p id="wide-two" class="title">two</p></div>
<div class="card narrow"><p id="narrow" class="title">narrow</p></div>
</body></html>`)
	passOne := resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Media: "print", Sheets: []*css.Stylesheet{cssSheet},
	}, nil)
	containers := measureSizeContainers(root, passOne, testViewport)
	styles := resolveStylesWithContainers(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800, containers)
	nodes := styleStoreNodesByID(root)

	assertStyleStoreShares(t, styles, nodes, "wide-one", "wide-two")
	assertStyleStoreDiffers(t, styles, nodes, "wide-one", "narrow")

	if styles[nodes["wide-one"]].Color[0] < 0.9 || styles[nodes["narrow"]].Color[0] > 0.1 {
		t.Fatalf(
			"container match/miss colors: wide=%v narrow=%v",
			styles[nodes["wide-one"]].Color,
			styles[nodes["narrow"]].Color,
		)
	}
}

func TestStyleStoreSharesRepeatedBenchmarkTemplateStyles(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("../../testdata/golden/benchmarks/templates/report.html.tmpl")
	if err != nil {
		t.Fatalf("parse benchmark template: %v", err)
	}

	data := struct {
		Pages []styleStoreBenchmarkPage
	}{
		Pages: []styleStoreBenchmarkPage{
			{
				First:  true,
				Number: 1,
				Rows: []styleStoreBenchmarkRow{
					{Number: "1", SKU: "A", Description: "first", Quantity: "1", Amount: "$1"},
					{Number: "2", SKU: "B", Description: "second", Quantity: "2", Amount: "$2"},
				},
			},
			{
				First:  false,
				Number: 2,
				Rows: []styleStoreBenchmarkRow{
					{Number: "3", SKU: "C", Description: "third", Quantity: "3", Amount: "$3"},
					{Number: "4", SKU: "D", Description: "fourth", Quantity: "4", Amount: "$4"},
				},
			},
		},
	}

	var rendered bytes.Buffer

	if err := tpl.Execute(&rendered, data); err != nil {
		t.Fatalf("execute benchmark template: %v", err)
	}

	root := mustParse(t, rendered.String())

	styles := resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{benchmarkTemplateSheet(t, root)},
	}, nil)
	plainCells, amountCells, headers, rows := benchmarkTemplateStyleNodes(root)

	assertAllStylePointersMatch(t, styles, plainCells, "plain td")
	assertAllStylePointersMatch(t, styles, amountCells, "td.amount")
	assertAllStylePointersMatch(t, styles, headers, "th")
	assertAllStylePointersMatch(t, styles, rows, "tr")

	if styles[plainCells[0]] == styles[amountCells[0]] || styles[headers[0]] == styles[plainCells[0]] {
		t.Fatal("benchmark template selector-specific styles were incorrectly shared")
	}
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
