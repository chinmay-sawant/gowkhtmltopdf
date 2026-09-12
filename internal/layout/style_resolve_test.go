package layout

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// repeatedTableStyles parses a table of identical td.plain / td.amount rows
// and returns the tree, the options, and the resolved styles.
func repeatedTableStyles(
	t *testing.T, rows int,
) (*html.Node, Options, map[*html.Node]*ResolvedStyle) {
	t.Helper()

	var markup strings.Builder

	markup.WriteString(`<html><body><table>`)

	for range rows {
		markup.WriteString(`<tr><td class="plain">x</td><td class="amount">1</td></tr>`)
	}

	markup.WriteString(`</table></body></html>`)

	root := mustParse(t, markup.String())
	opts := Options{
		Sheets: []*css.Stylesheet{sheet(t, `
			td { border: 1px solid #000; padding: 1px }
			td.amount { text-align: right }
		`)},
		Media:  "print",
		Width:  testViewport,
		Height: 800,
	}

	return root, opts, resolveStylesWith(root, opts, nil)
}

// TestStyleStoreSharesRepeatedElements pins the sharing contract at the
// styleStore interning boundary: repeated declaration shapes resolve to one
// pointer, and a changed shape keeps its own record.
func TestStyleStoreSharesRepeatedElements(t *testing.T) {
	t.Parallel()

	const rows = 64

	root, _, styles := repeatedTableStyles(t, rows)

	plain := styleRecordsByClass(t, root, styles, "plain")
	amount := styleRecordsByClass(t, root, styles, "amount")

	if len(plain) != rows || len(amount) != rows {
		t.Fatalf("cell counts plain=%d amount=%d, want %d each", len(plain), len(amount), rows)
	}

	requireSharedRecords(t, "interned td.plain", plain)
	requireSharedRecords(t, "interned td.amount", amount)
	requireDistinctRecords(t, "td.plain vs td.amount", plain[0], amount[0])
}

// TestNthChildRowsResolvePerPosition pins that nth-child declarations keep
// per-position values and that rows with different declaration shapes do not
// collapse onto one interned record.
func TestNthChildRowsResolvePerPosition(t *testing.T) {
	t.Parallel()

	var markup strings.Builder

	markup.WriteString(`<html><body><ul>`)

	for i := range 6 {
		fmt.Fprintf(&markup, `<li class="row">%d</li>`, i)
	}

	markup.WriteString(`</ul></body></html>`)

	root := mustParse(t, markup.String())
	opts := Options{
		Sheets: []*css.Stylesheet{sheet(t, `
			.row { color: #111 }
			.row:nth-child(2n) { color: #222 }
			.row:nth-child(3) { font-weight: 700 }
		`)},
		Media:  "print",
		Width:  testViewport,
		Height: 800,
	}
	styles := resolveStylesWith(root, opts, nil)

	rows := styleRecordsByClass(t, root, styles, "row")

	// Rows are 1-based children: row[1] and row[3] are even, row[2] is the
	// third child (even plus bold), and the odd rows share the base record.
	if rows[1].Color == rows[0].Color {
		t.Errorf("even row color = %v, want the nth-child(2n) override", rows[1].Color)
	}

	if rows[2].FontWeight < fontWeightBoldValue {
		t.Errorf("third row font-weight = %d, want bold", rows[2].FontWeight)
	}

	if rows[1] != rows[3] {
		t.Errorf("row[1] and row[3] = %p/%p, want one interned even record", rows[1], rows[3])
	}

	if rows[0] != rows[4] {
		t.Errorf("row[0] and row[4] = %p/%p, want one interned base record", rows[0], rows[4])
	}

	if rows[1] == rows[2] {
		t.Errorf("row[1] and row[2] share record %p despite different nth-child declarations", rows[1])
	}
}

// TestInheritablePropBitsComplete pins the declared-property mask table: every
// inheritable name maps to its own entry's bit, and the table fits one word.
func TestInheritablePropBitsComplete(t *testing.T) {
	t.Parallel()

	if len(inheritableProps) > 64 {
		t.Fatalf("inheritableProps has %d entries; the uint64 declared mask fits 64", len(inheritableProps))
	}

	// Names can be shared between entries (list-style), so the expected bit
	// set for a name is the OR of every entry that lists it.
	wantBits := make(map[string]uint64, len(inheritablePropBits))

	for i, entry := range inheritableProps {
		for _, name := range entry.names {
			wantBits[name] |= uint64(1) << i
		}
	}

	for name, want := range wantBits {
		got, ok := inheritablePropBits[name]
		if !ok {
			t.Errorf("inheritable property %q missing from inheritablePropBits", name)

			continue
		}

		if got != want {
			t.Errorf("inheritable property %q bits = %#x, want %#x", name, got, want)
		}
	}

	for name := range inheritablePropBits {
		if _, ok := wantBits[name]; !ok {
			t.Errorf("inheritablePropBits has stale name %q", name)
		}
	}
}

// TestDeclaredInheritableMask pins the fold itself: a multi-name entry sets
// one bit for either spelling, unknown properties contribute nothing, and an
// empty map yields no bits.
func TestDeclaredInheritableMask(t *testing.T) {
	t.Parallel()

	if got := declaredInheritableMask(nil); got != 0 {
		t.Fatalf("nil raw mask = %#x, want 0", got)
	}

	wordWrap, ok := inheritablePropBits["word-wrap"]
	if !ok {
		t.Fatal("word-wrap must share the overflow-wrap entry")
	}

	overflowWrap := inheritablePropBits["overflow-wrap"]

	if got := declaredInheritableMask(map[string]string{"word-wrap": "break-word"}); got != wordWrap {
		t.Fatalf("word-wrap mask = %#x, want %#x", got, wordWrap)
	}

	if got := declaredInheritableMask(map[string]string{"overflow-wrap": "normal"}); got != overflowWrap {
		t.Fatalf("overflow-wrap mask = %#x, want %#x", got, overflowWrap)
	}

	if wordWrap != overflowWrap {
		t.Fatalf("word-wrap bit %#x != overflow-wrap bit %#x", wordWrap, overflowWrap)
	}

	if got := declaredInheritableMask(map[string]string{"margin-top": "0", "display": "block"}); got != 0 {
		t.Fatalf("non-inheritable declarations mask = %#x, want 0", got)
	}

	color := inheritablePropBits["color"]

	if got := declaredInheritableMask(map[string]string{"color": "#fff"}); got != color {
		t.Fatalf("color mask = %#x, want %#x", got, color)
	}
}

// TestSortRestLonghandPropsPreservesSortOrder pins the replacement sort
// against sort.Strings: the longhand pass must keep the exact byte order the
// old implementation produced, because overlapping longhands depend on it.
func TestSortRestLonghandPropsPreservesSortOrder(t *testing.T) {
	t.Parallel()

	cases := [][]string{
		nil,
		{},
		{"a"},
		{"overflow-x", "overflow", "overflow-y"},
		{"z-index", "color", "width", "margin-top", "border-top-width", "background", "background-color"},
		{"b", "a", "d", "c", "f", "e"},
	}

	for _, input := range cases {
		want := append([]string(nil), input...)
		sortStringsReference(want)

		got := append([]string(nil), input...)
		got = sortRestLonghandProps(got)

		if !reflect.DeepEqual(got, want) {
			t.Errorf("sortRestLonghandProps(%v) = %v, want %v", input, got, want)
		}
	}
}

// sortStringsReference is the byte order the longhand pass has always used.
func sortStringsReference(props []string) {
	for i := 1; i < len(props); i++ {
		prop := props[i]
		prev := i - 1

		for prev >= 0 && props[prev] > prop {
			props[prev+1] = props[prev]
			prev--
		}

		props[prev+1] = prop
	}
}
