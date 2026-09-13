package layout

import (
	"slices"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// styleResolutionMemo caches resolved element styles for repeated declaration
// shapes. The 500-page report resolves 66,007 elements into 15 distinct
// ResolvedStyle records, so without the memo every element pays a full cascade
// plus the intern fingerprint/equality scan only to discover that the record
// already exists.
//
// The key captures every input the element resolution reads:
//
//   - parent style pointer: inheritance and custom-property resolution read
//     the parent record;
//   - matched rule pointers plus specificity: the exact matched-declaration
//     identity, so :nth-child, sibling combinators, :has, media gates and
//     container gates are reflected in which rules are present;
//   - node name: the UA sheet dispatches on the tag;
//   - inline style attribute: parsed only on a miss;
//   - rem base: rem lengths depend on the root font size;
//   - the --print-link-underline outcome for anchors: applied after the
//     cascade, so it cannot be derived from the declarations.
//
// A hit skips cascadeRaw, the whole raw-to-used application, and the intern
// fingerprint/equality scan, and returns the pointer already interned by
// styleStore. A miss stores a copy of the matched-rule sequence and compares
// it field by field on later lookups, so different declaration shapes that
// land in one map cell cannot share a record.
//
// The memo never crosses a resolution pass: it lives in styleContext, and
// container re-cascade passes (ctx.containers non-nil) do not use it because
// their gate outcomes are measured, not static.
type styleResolutionMemo struct {
	entries map[styleMemoKey][]*styleMemoEntry
	// disabled is the differential-test switch; container passes also skip
	// the memo without setting it.
	disabled bool
}

// styleMemoKey is the comparable part of the memo key. Elements that share it
// still need their matched-rule sequences compared before sharing a record.
type styleMemoKey struct {
	parent        *ResolvedStyle
	nodeName      string
	inline        string
	remBase       float64
	linkUnderline bool
}

// styleMemoEntry is one cached resolution: the exact matched-rule sequence
// that produced style, kept so equality is decided by the rules, never by the
// map hash alone.
type styleMemoEntry struct {
	hits  []ruleHit
	style *ResolvedStyle
}

// usable reports whether this pass may consult the memo for an element.
func (m *styleResolutionMemo) usable(parent *ResolvedStyle) bool {
	return !m.disabled && parent != nil
}

// lookup returns the cached style for key when the stored matched-rule
// sequence equals hits.
func (m *styleResolutionMemo) lookup(key styleMemoKey, hits []ruleHit) *ResolvedStyle {
	for _, entry := range m.entries[key] {
		if ruleHitsEqual(entry.hits, hits) {
			return entry.style
		}
	}

	return nil
}

// insert stores style under key and a private copy of hits.
func (m *styleResolutionMemo) insert(key styleMemoKey, hits []ruleHit, style *ResolvedStyle) {
	if m.entries == nil {
		m.entries = make(map[styleMemoKey][]*styleMemoEntry)
	}

	m.entries[key] = append(m.entries[key], &styleMemoEntry{
		hits:  slices.Clone(hits),
		style: style,
	})
}

// ruleHitsEqual reports whether two matched-rule sequences are identical.
// ruleHit holds a rule pointer and three specificity ints, so field
// comparison is the exact identity check the memo key needs.
func ruleHitsEqual(left, right []ruleHit) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

// resolveElementStyleMemo resolves one element through the memo when the pass
// allows it and returns the interned style pointer.
//
// The root element is excluded: it has no parent, and its resolution is the
// only writer of ctx.remBase, so caching it would skip that side effect.
func resolveElementStyleMemo(
	node *html.Node, ctx *styleContext, parent *ResolvedStyle, store *styleStore,
) *ResolvedStyle {
	if ctx == nil || ctx.containers != nil || node.Name == "html" || !ctx.memo.usable(parent) {
		resolveElementStyle(node, ctx, parent, &store.candidate)

		return store.append(store.candidate)
	}

	// The matched-rule list is needed for the key, so it is computed before
	// the cascade instead of inside it.
	hits := ctx.matchedRules(node, "")
	key := styleMemoKey{
		parent:        parent,
		nodeName:      node.Name,
		inline:        node.Attribute("style"),
		remBase:       ctx.remBase,
		linkUnderline: linkUnderlinePolicy(ctx, node),
	}

	if cached := ctx.memo.lookup(key, hits); cached != nil {
		return cached
	}

	resolveElementStyleWithHits(node, ctx, parent, &store.candidate, hits)
	style := store.append(store.candidate)

	ctx.memo.insert(key, hits, style)

	return style
}

// linkUnderlinePolicy reports whether resolveElementStyle overrides
// text-decoration for this node under --print-link-underline. The policy
// runs after the cascade, so its outcome is part of the memo key.
func linkUnderlinePolicy(ctx *styleContext, node *html.Node) bool {
	return ctx.printLinkUnderline &&
		node.Name == "a" &&
		strings.TrimSpace(node.Attribute("href")) != ""
}
