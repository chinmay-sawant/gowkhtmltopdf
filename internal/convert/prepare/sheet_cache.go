package prepare

import (
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// SheetCache reuses stylesheets parsed by an earlier collection of the same
// tree. Keys are tree nodes or a parsed sheet plus an import index, so a
// cache belongs to exactly one tree; a later collection over that tree can
// skip every parse and fetch while still re-evaluating viewport gating. A nil
// cache parses and fetches everything again.
type SheetCache struct {
	inline  map[*html.Node]*css.Stylesheet
	links   map[*html.Node]cachedSheet
	imports map[importKey]cachedSheet
}

// cachedSheet is one fetched stylesheet plus the base and resolved URL that
// produced it, so a cache hit repeats the bookkeeping of a real fetch.
type cachedSheet struct {
	sheet *css.Stylesheet
	base  string
	seen  string
}

// importKey identifies one @import rule by its owning sheet and rule index.
type importKey struct {
	sheet *css.Stylesheet
	index int
}

// NewSheetCache returns an empty cache for one parsed tree.
func NewSheetCache() *SheetCache {
	return &SheetCache{
		inline:  make(map[*html.Node]*css.Stylesheet),
		links:   make(map[*html.Node]cachedSheet),
		imports: make(map[importKey]cachedSheet),
	}
}
