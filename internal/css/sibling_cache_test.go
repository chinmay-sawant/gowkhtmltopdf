package css

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestSiblingCacheIsBounded proves the process-global sibling index cache
// evicts instead of retaining every parent it has seen. Without the bound, a
// long-lived library or server process keeps node-keyed indexes from every
// document it ever matched.
func TestSiblingCacheIsBounded(t *testing.T) { //nolint:paralleltest // asserts on the process-global cache
	cache := newSiblingCache()
	parents := make([]*html.Node, sibCacheCap+1)

	for i := range parents {
		parents[i] = cacheTestParent()
		cache.putIfAbsent(parents[i], buildParentCache(parents[i]))
	}

	if cache.get(parents[0]) != nil {
		t.Error("oldest entry must be evicted once the cap is exceeded")
	}

	if cache.get(parents[sibCacheCap]) == nil {
		t.Error("newest entry must stay cached")
	}

	if got := cache.size(); got != sibCacheCap {
		t.Fatalf("cache size = %d, want %d", got, sibCacheCap)
	}

	for range sibCacheCap + 32 {
		getParentCache(cacheTestParent())
	}

	if got := sibCache.size(); got > sibCacheCap {
		t.Fatalf("global sibling cache holds %d entries, want at most %d", got, sibCacheCap)
	}
}

// cacheTestParent builds a distinct parent with one element child so
// buildParentCache has something to index.
func cacheTestParent() *html.Node {
	child := &html.Node{Type: html.ElementNode, Name: "span"}
	parent := &html.Node{Type: html.ElementNode, Name: "div", Children: []*html.Node{child}}
	child.Parent = parent

	return parent
}
