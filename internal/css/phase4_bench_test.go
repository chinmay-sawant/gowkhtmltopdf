package css //nolint:testpackage // exercises unexported selector matching via Match

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// manyMediaSheet builds a stylesheet with count @media blocks, the shape that
// used to lowercase the whole remaining source per at-rule (quadratic).
func manyMediaSheet(count int) string {
	var b strings.Builder

	for range count {
		b.WriteString("@media print { .x { color: red; } }\n")
	}

	return b.String()
}

func BenchmarkParseManyAtRules(b *testing.B) {
	src := manyMediaSheet(300)

	b.ReportAllocs()
	b.SetBytes(int64(len(src)))

	for range b.N {
		sheet, err := Parse(src)
		if err != nil {
			b.Fatal(err)
		}

		if len(sheet.Rules) == 0 {
			b.Fatal("no rules parsed")
		}
	}
}

// attrCaseSelector parses one selector.
func attrCaseSelector(tb testing.TB, sel string) Selector {
	tb.Helper()

	s, ok := parseSelector(sel)
	if !ok {
		tb.Fatal("parse selector")
	}

	return s
}

func BenchmarkAttrIgnoreCaseMatch(b *testing.B) {
	sel := attrCaseSelector(b, `[title="helloworld" i]`)
	node := &html.Node{Type: html.ElementNode, Name: "a", Attrs: map[string]string{"title": "HELLOWORLD"}}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if !Match(sel, node) {
			b.Fatal("expected match")
		}
	}
}

// wideTableHTML builds a table with rowCount rows, each with one cell.
func wideTableHTML(rowCount int) string {
	var b strings.Builder

	b.WriteString("<html><body><table>")

	for range rowCount {
		b.WriteString("<tr><td>x</td></tr>")
	}

	b.WriteString("</table></body></html>")

	return b.String()
}

func wideTableRows(b *testing.B, rowCount int) []*html.Node {
	b.Helper()

	root, err := html.Parse(wideTableHTML(rowCount))
	if err != nil {
		b.Fatal(err)
	}

	var rows []*html.Node

	root.Walk(func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "tr" {
			rows = append(rows, n)
		}
	})

	return rows
}

// BenchmarkWideTableNthLastOfType matches :nth-last-of-type() on every row of
// a wide table: ofTypeLastIndex used to scan the shared parent twice per call.
func BenchmarkWideTableNthLastOfType(b *testing.B) {
	rows := wideTableRows(b, 200)
	sel := attrCaseSelector(b, "tr:nth-last-of-type(odd)")

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		for _, row := range rows {
			Match(sel, row)
		}
	}
}

// BenchmarkWideTableNthChild matches :nth-child() on every row of a wide
// table (elementIndex keeps its early-exit scan).
func BenchmarkWideTableNthChild(b *testing.B) {
	rows := wideTableRows(b, 200)
	sel := attrCaseSelector(b, "tr:nth-child(odd)")

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		for _, row := range rows {
			Match(sel, row)
		}
	}
}

func BenchmarkResolveVarsMany(b *testing.B) {
	value := strings.Repeat("var(--a) ", 200) + "solid red"
	lookup := func(name string) (string, bool) {
		switch name {
		case "--a":
			return "1px", true
		default:
			return "", false
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if got := ResolveVars(value, lookup); got == "" {
			b.Fatal("no output")
		}
	}
}
