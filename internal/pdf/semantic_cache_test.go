package pdf

import (
	"strconv"
	"testing"
)

func TestCompileSemanticRECachesPattern(t *testing.T) {
	t.Parallel()

	const pattern = `/MediaBox\s+(\d+)`

	first, err := compileSemanticRE(pattern)
	if err != nil {
		t.Fatalf("compileSemanticRE(%q): %v", pattern, err)
	}

	second, err := compileSemanticRE(pattern)
	if err != nil {
		t.Fatalf("compileSemanticRE(%q): %v", pattern, err)
	}

	if first != second {
		t.Error("same pattern compiled to a different *regexp.Regexp (cache miss)")
	}
}

func TestCompileSemanticREInvalidPatternReturnsError(t *testing.T) {
	t.Parallel()

	if _, err := compileSemanticRE(`(`); err == nil {
		t.Fatal("compileSemanticRE accepted an invalid pattern")
	}
}

func TestSemanticRegexCacheIsBounded(t *testing.T) {
	t.Parallel()

	for i := range semanticRegexCacheCap + 10 {
		pattern := `key` + strconv.Itoa(i) + `\s+(\d+)`

		if _, err := compileSemanticRE(pattern); err != nil {
			t.Fatalf("compileSemanticRE(%q): %v", pattern, err)
		}
	}

	semanticRegexCache.mu.Lock()
	entries := len(semanticRegexCache.re)
	orderLen := len(semanticRegexCache.order)
	semanticRegexCache.mu.Unlock()

	if entries > semanticRegexCacheCap {
		t.Fatalf("cache holds %d patterns, cap is %d", entries, semanticRegexCacheCap)
	}

	if orderLen != entries {
		t.Fatalf("recency list length %d != map length %d", orderLen, entries)
	}
}
