package css

import (
	"strings"
	"testing"
	"time"
)

// TestParseNestedFunctionalPseudoDepth proves recursive selector parsing is
// depth bounded. Before the guard, 200k nested :not( rescanned to the matching
// paren at every level (quadratic in the input size) and never finished.
func TestParseNestedFunctionalPseudoDepth(t *testing.T) { //nolint:paralleltest // timeout probe is intentionally serial
	const probeDepth = 200000

	done := make(chan struct{})

	var parsedOK bool

	go func() {
		_, parsedOK = ParseSelectors(nestedNotSelector(probeDepth))

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("parsing 200k nested :not() did not finish in 10s; parse depth cap missing")
	}

	if parsedOK {
		t.Fatalf("selector with %d nested :not() must be rejected", probeDepth)
	}

	if _, ok := ParseSelectors(nestedNotSelector(maxParseDepth)); !ok {
		t.Fatalf("selector with %d nested :not() must still parse", maxParseDepth)
	}

	if _, ok := ParseSelectors(nestedNotSelector(maxParseDepth + 1)); ok {
		t.Fatalf("selector with %d nested :not() must be rejected", maxParseDepth+1)
	}
}

// TestParseDepthGuardCoversContainerAndAtRules proves the one maxParseDepth
// constant also bounds the container-condition and nested-at-rule parsers.
func TestParseDepthGuardCoversContainerAndAtRules(t *testing.T) {
	t.Parallel()

	shallowCond := strings.Repeat("not ", maxParseDepth) + "(min-width: 1px)"
	if _, ok := parseContainerCond(shallowCond); !ok {
		t.Fatalf("container condition with %d nots must still parse", maxParseDepth)
	}

	deepCond := strings.Repeat("not ", maxParseDepth+1) + "(min-width: 1px)"
	if _, ok := parseContainerCond(deepCond); ok {
		t.Fatalf("container condition with %d nots must be rejected", maxParseDepth+1)
	}

	sheet, err := Parse(nestedContainerStylesheet(maxParseDepth))
	if err != nil {
		t.Fatalf("Parse at the nesting cap: %v", err)
	}

	if len(sheet.Rules) != 1 {
		t.Fatalf("nested @container at the cap: got %d rules, want 1", len(sheet.Rules))
	}

	sheet, err = Parse(nestedContainerStylesheet(maxParseDepth + 1))
	if err != nil {
		t.Fatalf("Parse past the nesting cap must not error: %v", err)
	}

	if len(sheet.Rules) != 0 {
		t.Fatalf("nested @container past the cap: got %d rules, want 0", len(sheet.Rules))
	}
}

// nestedNotSelector returns "a:not(:not(...(b)...))" with depth :not() levels.
func nestedNotSelector(depth int) string {
	return "a" + strings.Repeat(":not(", depth) + "b" + strings.Repeat(")", depth)
}

// nestedContainerStylesheet wraps one rule in depth @container blocks.
func nestedContainerStylesheet(depth int) string {
	return strings.Repeat("@container (min-width: 1px) {", depth) + "a{color:red}" + strings.Repeat("}", depth)
}
