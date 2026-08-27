//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strconv"
	"strings"
	"testing"
)

func joinedPaintText(res *Result) string {
	var boxNode strings.Builder

	for _, op := range res.Ops {
		if op.Kind == OpText || op.Kind == OpBullet {
			boxNode.WriteString(op.Text)
		}
	}

	return strings.Join(strings.Fields(boxNode.String()), " ")
}

func TestCounterResetIncrementLayout(t *testing.T) { //nolint:funlen // parse, map, and two layout proofs
	t.Parallel()

	t.Run("parseAndMap", func(t *testing.T) {
		t.Parallel()

		ops := parseCounterList("section 1 item", counterResetDefault)
		if len(ops) != pairLen || ops[0].name != "section" || ops[0].value != 1 ||
			ops[1].name != "item" || ops[1].value != 0 {
			t.Fatalf("parse reset: %+v", ops)
		}

		incs := parseCounterList("section", counterIncrementDefault)
		if len(incs) != 1 || incs[0].name != "section" || incs[0].value != 1 {
			t.Fatalf("parse increment: %+v", incs)
		}

		if parseCounterList("none", counterResetDefault) != nil {
			t.Fatal("none should be empty")
		}

		cmap := newCounterMap()
		pushes := cmap.applyReset("section")

		if cmap.value("section") != 0 {
			t.Fatalf("reset want 0 got %d", cmap.value("section"))
		}

		cmap.applyIncrement("section")
		if cmap.value("section") != 1 {
			t.Fatalf("first increment want 1 got %d", cmap.value("section"))
		}

		cmap.applyIncrement("section")
		if cmap.value("section") != pairLen {
			t.Fatalf("second increment want 2 got %d", cmap.value("section"))
		}

		inner := cmap.applyReset("section")
		cmap.applyIncrement("section")

		got := joinInts(cmap.values("section"), ".")
		if got != "2.1" {
			t.Fatalf("nested values want 2.1 got %s", got)
		}

		cmap.pop(inner)
		if cmap.value("section") != pairLen {
			t.Fatalf("after pop want 2 got %d", cmap.value("section"))
		}

		cmap.pop(pushes)
	})

	t.Run("layout", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.reset { counter-reset: section; }
.inc { counter-increment: section; }
.inc::before { content: counter(section) ". "; }
`)
		res := layoutHTML(t, `<html><body>
<div class="reset">
<span class="inc">A</span>
<span class="inc">B</span>
<span class="inc">C</span>
</div>
</body></html>`, cssSheet)

		got := joinedPaintText(res)
		if got != "1. A 2. B 3. C" {
			t.Fatalf("counter-reset/increment layout: %q", got)
		}
	})

	t.Run("inlineStyleReset", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.inc { counter-increment: item; }
.inc::before { content: counter(item) " "; }
`)
		res := layoutHTML(t, `<html><body>
<div style="counter-reset: item">
<span class="inc">A</span> <span class="inc">B</span>
</div>
</body></html>`, cssSheet)

		got := joinedPaintText(res)
		if got != "1 A 2 B" {
			t.Fatalf("inline counter-reset: %q", got)
		}
	})
}

func TestCounterInBefore(t *testing.T) {
	t.Parallel()

	t.Run("countersFunction", func(t *testing.T) {
		t.Parallel()

		env := defaultContentEnv()
		env.counters.applyReset("section")
		env.counters.applyIncrement("section")
		_ = env.counters.applyReset("section")
		env.counters.applyIncrement("section")

		got := evalContentValue(`counters(section, ".") " "`, nil, env)
		if got != "1.1 " {
			t.Fatalf("counters() want %q got %q", "1.1 ", got)
		}

		got = evalContentValue(`counter(section)`, nil, env)
		if got != "1" {
			t.Fatalf("counter() want 1 got %q", got)
		}

		got = parseContentValue(`counter(missing) "." counters(missing, ".")`, nil)
		if got != "0.0" {
			t.Fatalf("unset counters want 0.0 got %q", got)
		}
	})

	t.Run("nestedLayout", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
ol { counter-reset: section; list-style: none; margin: 0; padding: 0; }
li { display: inline; counter-increment: section; }
li::before { content: counters(section, ".") " "; }
`)
		res := layoutHTML(t, `<html><body>
<ol>
<li>A
<ol>
<li>B</li>
<li>C</li>
</ol>
</li>
<li>D</li>
</ol>
</body></html>`, cssSheet)

		got := joinedPaintText(res)
		if !strings.Contains(got, "1 A") || !strings.Contains(got, "1.1 B") ||
			!strings.Contains(got, "1.2 C") || !strings.Contains(got, "2 D") {
			t.Fatalf("nested counters in ::before: %q", got)
		}
	})
}

func joinInts(vals []int, sep string) string {
	parts := make([]string, len(vals))
	for idx, val := range vals {
		parts[idx] = strconv.Itoa(val)
	}

	return strings.Join(parts, sep)
}
