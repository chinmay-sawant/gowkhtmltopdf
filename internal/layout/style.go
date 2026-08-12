package layout

import (
	"math"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// CSS keyword constants shared by the cascade (goconst).
const (
	inheritKeyword     = "inherit"
	solidKeyword       = "solid"
	clearKeyword       = "clear"
	visibleKeyword     = "visible"
	pageKeyword        = "page"
	avoidKeyword       = "avoid"
	avoidPageValue     = "avoid-page"
	remUnit            = "rem"
	divElementName     = "div"
	styleElement       = "style"
	borderWidthKeyword = "border-width"
	borderStyleKeyword = "border-style"
	borderColorKeyword = "border-color"
	gapKeyword         = "gap"
	containerKeyword   = "container"
	flexKeyword        = "flex"
	flexStartKeyword   = "flex-start"
)

// hashFontFamily fingerprints a CSS font-family list without allocating.
func hashFontFamily(fams []string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	hash := uint64(offset64)

	for _, fam := range fams {
		for i := range len(fam) {
			hash ^= uint64(fam[i])
			hash *= prime64
		}

		hash ^= 0xff // token separator
		hash *= prime64
	}

	return hash
}

// fontWeightStep is the bolder/lighter adjustment applied to the current
// weight (CSS Fonts 3 §3.3; clamped by the 100..900 numeric range).
const fontWeightStep = 100

// asciiFoldBit is the single-bit mask that lowercases an ASCII letter
// (s[i]|asciiFoldBit maps 'A'-'Z' to 'a'-'z').
const asciiFoldBit = 0x20

// ResolvedStyle is the used style of one element: values the layout engine
// consumes, in points (or unitless where noted). Only the phase-04 subset is
// modeled; everything else keeps its initial value.
//
//nolint:lll // the resolved-style table keeps field comments beside each property
type ResolvedStyle struct {
	Display                                                                                    string
	Position                                                                                   string  // "static" | "relative" | "absolute" | "fixed" | "sticky"
	Float                                                                                      string  // cssDisplayNone | floatLeft | floatRight
	Clear                                                                                      string  // cssDisplayNone | floatLeft | floatRight | "both"
	BoxSizing                                                                                  string  // "content-box" | "border-box"
	Top                                                                                        float64 // position offsets (pt); 0 = unset for absolute uses Auto flags
	Right                                                                                      float64
	Bottom                                                                                     float64
	Left                                                                                       float64
	TopAuto                                                                                    bool
	RightAuto                                                                                  bool
	BottomAuto                                                                                 bool
	LeftAuto                                                                                   bool
	FlexDirection                                                                              string  // "row" | fxCol | "row-reverse" | "column-reverse"
	FlexWrap                                                                                   string  // "nowrap" | "wrap" | "wrap-reverse"
	JustifyContent                                                                             string  // flex-start | flex-end | center | space-between | space-around | space-evenly
	AlignItems                                                                                 string  // stretch | flex-start | center | flex-end
	AlignContent                                                                               string  // flex-start | flex-end | center | space-between | space-around | space-evenly | stretch
	AlignSelf                                                                                  string  // auto | stretch | flex-start | flex-end | center | start | end
	JustifyItems                                                                               string  // grid: stretch | start | end | center
	JustifySelf                                                                                string  // grid item: auto | stretch | start | end | center
	Gap                                                                                        float64 // flex/grid gap shorthand (pt); kept for backward compat
	RowGap                                                                                     float64 // pt; 0 with ColumnGap 0 → layout falls back to Gap
	ColumnGap                                                                                  float64
	ColumnGapNormal                                                                            bool    // true when column-gap is normal/initial (multicol → 1em; flex/grid → 0)
	ColumnCount                                                                                int     // 0 = auto; ≥1 = used count hint
	ColumnWidth                                                                                float64 // -1 = auto; else length in pt
	ColumnSpan                                                                                 string  // cssDisplayNone | "all" (multicol spanner)
	ColumnFill                                                                                 string  // "balance" | overflowAuto
	FlexGrow                                                                                   float64
	FlexShrink                                                                                 float64 // default 1; 0 disables shrink
	FlexBasis                                                                                  float64 // -1 = auto
	FlexBasisPercent                                                                           float64 // >=0 means % of flex container content main size (width/height)
	FlexOrder                                                                                  int
	ZIndex                                                                                     int
	ZIndexSet                                                                                  bool
	WritingMode                                                                                string // "" | "horizontal-tb" | "vertical-rl" | "vertical-lr"
	GridTemplateColumns                                                                        string // raw grid-template-columns value
	GridTemplateRows                                                                           string
	GridTemplateAreas                                                                          string  // raw grid-template-areas value
	GridArea                                                                                   string  // named area (custom-ident); empty = line-based placement
	GridAutoFlow                                                                               string  // "row" | fxCol | "dense" | "row dense" | "column dense"
	GridColumnSpan                                                                             int     // from grid-column: span N (default 1)
	GridColumnStart                                                                            int     // 1-based; 0 = auto
	GridRowSpan                                                                                int     // from grid-row: span N (default 1)
	GridRowStart                                                                               int     // 1-based; 0 = auto
	Width                                                                                      float64 // -1 = auto; absolute length in pt when WidthPercent < 0
	WidthPercent                                                                               float64 // >=0 means width is that % of the containing block at layout time
	Height                                                                                     float64 // -1 = auto; absolute length in pt when HeightPercent < 0
	HeightPercent                                                                              float64 // >=0 means height is that % of the CB; indefinite CB → auto (cyclic honesty)
	MinWidth                                                                                   float64 // absolute pt when MinWidthPercent < 0; 0 = auto (content min for flex)
	MinWidthPercent                                                                            float64 // >=0 means % of containing block (deferred like WidthPercent)
	MinWidthSet                                                                                bool    // true when min-width was explicitly declared, including 0
	MaxWidth                                                                                   float64
	MaxWidthPercent                                                                            float64 // >=0 means % of containing block / img clamp context
	MinHeight                                                                                  float64
	MinHeightPercent                                                                           float64 // >=0 means % of CB height; indefinite → ignore
	MaxHeight                                                                                  float64
	Overflow                                                                                   string // "visible" | "hidden" | "scroll" | "auto" | "clip" (non-visible = sticky scrollport)
	MarginTop                                                                                  float64
	MarginRight                                                                                float64
	MarginBottom                                                                               float64
	MarginLeft                                                                                 float64
	MarginTopAuto                                                                              bool // margin-top: auto in a flex column
	MarginBottomAuto                                                                           bool // margin-bottom: auto in a flex column
	MarginLeftAuto                                                                             bool // margin-left: auto (horizontal centering with right auto)
	MarginRightAuto                                                                            bool // margin-right: auto
	PaddingTop                                                                                 float64
	PaddingRight                                                                               float64
	PaddingBottom                                                                              float64
	PaddingLeft                                                                                float64
	BorderTop                                                                                  border
	BorderRight                                                                                border
	BorderBottom                                                                               border
	BorderLeft                                                                                 border
	BorderRadius                                                                               float64
	BorderRadiusPercent                                                                        float64
	BorderRadiusTopLeft, BorderRadiusTopRight, BorderRadiusBottomRight, BorderRadiusBottomLeft float64
	Color                                                                                      [3]float64
	BGColor                                                                                    [4]float64 // rgba, 0..1
	FontFamily                                                                                 []string
	// famHash is the FNV-1a fingerprint of FontFamily, computed once during
	// style resolution. Text measurement reuses
	// it instead of re-hashing the family list per run.
	famHash            uint64
	FontSize           float64 // pts
	FontWeight         int
	FontItalic         bool
	LineHeight         float64 // pts; 0 = "normal"
	LineHeightUnitless float64 // multiplier when line-height was unitless; 0 otherwise
	TextAlign          string  // floatLeft | floatRight | "center" | "justify"
	TextTransform      string  // "none" | "uppercase" | "lowercase" | "capitalize"
	VerticalAlign      string  // "baseline" | "top" | "middle" | cssVerticalAlignBottom
	WhiteSpace         string  // "normal" | "nowrap" | "pre"
	// OverflowWrap is CSS overflow-wrap / word-wrap: "normal" | "break-word" | "anywhere".
	OverflowWrap string
	// WordBreak is CSS word-break: "normal" | "break-all" | "keep-all".
	WordBreak       string
	TextDecoration  string // cssDisplayNone | "underline" | "line-through"
	LetterSpacing   float64
	TextIndent      float64
	ListStyleType   string // "disc" | "circle" | "square" | "decimal" | cssDisplayNone | …
	BorderCollapse  string // "separate" | "collapse"
	BorderSpacing   float64
	TableLayout     string // overflowAuto | "fixed"
	IsReplaced      bool   // img, hr
	PageBreakBefore string // "" | "always" | "avoid"
	PageBreakAfter  string // "" | "always" | "avoid"
	PageBreakInside string // "" | "always" | "avoid"
	Orphans         int    // CSS orphans; inherited; initial 2; integer ≥ 1
	Widows          int    // CSS widows; inherited; initial 2; integer ≥ 1
	ContainerType   string // "" | "normal" | "inline-size" | "size"
	ContainerName   string // space-separated lower-case names; empty = none
	// Static 2D CSS transforms (paint-time CTM; sibling flow unchanged).
	Transform       Matrix2D
	HasTransform    bool
	TransformOrigin transformOriginSpec
	Opacity         float64 // 0..1; initial 1; also from filter:opacity()
	// CustomProps holds resolved CSS custom properties (--*) for this element
	// (inherited). Shared with the parent map when the element declares none.
	CustomProps map[string]string
}

type border struct {
	Width      float64 // layout width in CSS points, retained for pagination geometry
	PaintWidth float64 // device paint width; zero means use Width
	Style      string  // cssDisplayNone | "solid" | "dashed" | "dotted"
	Color      [3]float64
}

// initialStyle returns the CSS initial values.
func initialStyle() ResolvedStyle { //nolint:funlen // complete CSS initial-value record
	return ResolvedStyle{ //nolint:exhaustruct // intentional zero fields
		Display:          "inline",
		Position:         "static",
		Float:            cssDisplayNone,
		FlexGrow:         0,
		FlexShrink:       1,
		FlexBasis:        -1,
		FlexBasisPercent: -1,
		Clear:            cssDisplayNone,
		BoxSizing:        "content-box",
		TopAuto:          true,
		RightAuto:        true,
		BottomAuto:       true,
		LeftAuto:         true,
		FlexDirection:    "row",
		FlexWrap:         "nowrap",
		JustifyContent:   "flex-start",
		AlignItems:       "stretch",
		AlignContent:     "stretch",
		AlignSelf:        overflowAuto,
		JustifyItems:     "stretch",
		JustifySelf:      overflowAuto,
		ColumnGapNormal:  true,
		ColumnWidth:      -1,
		ColumnSpan:       cssDisplayNone,
		ColumnFill:       "balance",
		Width:            -1,
		WidthPercent:     -1,
		Height:           -1,
		HeightPercent:    -1,
		MinWidth:         0,
		MinWidthPercent:  -1,
		MaxWidth:         -1,
		MaxWidthPercent:  -1,
		MinHeight:        0,
		MinHeightPercent: -1,
		MaxHeight:        -1,
		Overflow:         "visible",
		Color:            [3]float64{0, 0, 0},
		BGColor:          [4]float64{0, 0, 0, 0},
		FontFamily:       nil,
		// Empty family hashes to the FNV-1a offset, matching what
		// resolveElementStyle records for elements without font-family.
		famHash:         hashFontFamily(nil),
		FontSize:        defaultFontSizePt, // 16px at 96dpi
		FontWeight:      fontWeightNormal,
		TextTransform:   textTransformNone,
		VerticalAlign:   "baseline",
		WhiteSpace:      "normal",
		OverflowWrap:    "normal",
		WordBreak:       "normal",
		TextDecoration:  cssDisplayNone,
		ListStyleType:   "disc",
		BorderCollapse:  "separate",
		BorderSpacing:   0,
		TableLayout:     overflowAuto,
		GridColumnSpan:  1,
		GridRowSpan:     1,
		WritingMode:     "horizontal-tb",
		Orphans:         two,
		Widows:          two,
		Transform:       IdentityMatrix(),
		TransformOrigin: defaultTransformOrigin(),
		Opacity:         1,
	}
}

// defaultTextStyle covers a text node without an element parent. Ordinary
// text nodes reuse their parent's style because text has no declarations of
// its own and the inline layout path only consumes inherited text properties.
var defaultTextStyle = initialStyle() //nolint:gochecknoglobals // immutable text default

// styleContext carries per-element resolution inputs.
type styleContext struct {
	sheets    []*css.Stylesheet
	media     string
	viewportW float64 // containing-block width for % of margins/padding/width
	viewportH float64 // for % of height
	// remBase is the used font-size of the root element for rem units (pt).
	// 0 means the CSS initial medium size (16px → 12pt).
	remBase float64
	// printLinkUnderline is the opt-in --print-link-underline operator policy.
	printLinkUnderline bool
	// containers maps size-query containers (inline-size|size) to their used
	// content-box inline size. nil means first pass: skip @container rules.
	containers map[*html.Node]sizeContainer
	// ruleHits is reused between sequential cascade lookups. A lookup consumes
	// the returned slice before the next element is resolved.
	ruleHits []ruleHit
	// Cascade maps are reused between sequential element lookups. Their values
	// are consumed before the next element is resolved.
	cascadeWins  map[string]cascadeWin
	cascadeProps map[string]string
}

// sizeContainer is one element that establishes a size query container.
type sizeContainer struct {
	inlineSize float64 // content-box inline size in pt
	fontSize   float64 // used font-size (em base for query lengths)
	names      string  // space-separated container-name values
}

// sameSizeContainerState reports whether a second container measurement is
// equivalent to the previous one. Container queries can use em lengths, so a
// changed used font-size is just as significant as a changed inline size or
// name. Keeping this comparison here gives the convergence loop one policy
// for deciding whether a second style pass is required.
func sameSizeContainerState(a, b sizeContainer) bool {
	return nearlyEqual(a.inlineSize, b.inlineSize) &&
		nearlyEqual(a.fontSize, b.fontSize) &&
		a.names == b.names
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= styleEpsilon
}

// resolveStylesWith is the single cascade entry: Options + optional size
// containers for @container rules (nil = first pass, skip container queries).
// Values are heap-backed once; the engine reuses these pointers (no second copy).
func resolveStylesWith(
	root *html.Node, opts Options, containers map[*html.Node]sizeContainer,
) map[*html.Node]*ResolvedStyle {
	return resolveStylesCtx(root, &styleContext{ //nolint:exhaustruct // intentional zero fields
		sheets:             opts.Sheets,
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		containers:         containers,
	})
}

// resolveStyles walks the tree top-down (test helper; no operator policies).
// @container rules are ignored on this first pass (no used sizes yet).
func resolveStyles(
	root *html.Node, sheets []*css.Stylesheet, media string, viewportW, viewportH float64,
) map[*html.Node]*ResolvedStyle {
	return resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Sheets: sheets, Media: media, Width: viewportW, Height: viewportH,
	}, nil)
}

// resolveStylesWithContainers is the second style pass: @container rules are
// applied when their query matches the nearest eligible ancestor in containers.
// Test helper; media/viewport always come from the caller's fixture.
func resolveStylesWithContainers(
	root *html.Node,
	sheets []*css.Stylesheet,
	media string, //nolint:unparam // test helper: media fixed per call site
	viewportW, viewportH float64, //nolint:unparam // test helper: viewport fixed per call site
	containers map[*html.Node]sizeContainer,
) map[*html.Node]*ResolvedStyle {
	return resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Sheets: sheets, Media: media, Width: viewportW, Height: viewportH,
	}, containers)
}

func resolveStylesCtx(root *html.Node, ctx *styleContext) map[*html.Node]*ResolvedStyle {
	nodeCount := countStyleNodes(root)
	out := make(map[*html.Node]*ResolvedStyle, nodeCount)
	store := styleStore{} //nolint:exhaustruct // intentional zero-value store

	var walk func(n *html.Node, parent *ResolvedStyle)
	walk = func(node *html.Node, parent *ResolvedStyle) {
		var sty *ResolvedStyle

		switch node.Type {
		case html.ElementNode:
			resolveElementStyle(node, ctx, parent, &store.candidate)
			sty = store.intern(store.candidate)
		case html.TextNode:
			sty = parent
			if sty == nil {
				sty = &defaultTextStyle
			}
		case html.CommentNode, html.DoctypeNode:
			// No style resolution; store a shared zero so map lookups stay non-nil.
			sty = &zeroResolvedStyle
		}

		out[node] = sty

		// Text nodes have no declarations of their own. Pointing them at the
		// already-resolved parent avoids allocating another full style copy per
		// text-bearing element while preserving inherited text properties.
		for _, child := range node.Children {
			if child.Type == html.TextNode {
				out[child] = sty

				continue
			}

			walk(child, sty)
		}
	}
	walk(root, nil)

	return out
}

// countStyleNodes counts the nodes needed for the result map. ResolvedStyle
// values are allocated lazily by styleStore, so an element count is no longer
// needed to reserve one full style record per element.
func countStyleNodes(root *html.Node) int {
	nodeCount := 0

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		nodeCount++

		for _, child := range node.Children {
			if child.Type == html.TextNode {
				nodeCount++

				continue
			}

			walk(child)
		}
	}

	walk(root)

	return nodeCount
}

// styleStoreChunkSize keeps canonical styles in small stable backing arrays.
// A chunk's capacity is never exceeded, so pointers returned from intern stay
// valid even when later chunks are appended.
const styleStoreChunkSize = 64

// styleStore owns canonical styles for one resolution pass. It deliberately
// does not cross Layout calls or @container re-cascade passes.
type styleStore struct {
	candidate ResolvedStyle
	chunks    [][]ResolvedStyle
	interned  map[styleStoreKey][]*ResolvedStyle
}

// styleStoreKey is a comparable coarse discriminator for style candidates.
// It reduces exact comparisons without deciding semantic equivalence itself.
type styleStoreKey struct {
	display, position, float, clear, boxSizing                                                 string
	fontHash                                                                                   uint64
	fontSize, lineHeight, lineHeightUnitless                                                   float64
	fontWeight                                                                                 int
	fontItalic                                                                                 bool
	color                                                                                      [3]float64
	bgColor                                                                                    [4]float64
	width, widthPercent                                                                        float64
	height, heightPercent                                                                      float64
	borderRadius, borderRadiusPercent                                                          float64
	borderRadiusTopLeft, borderRadiusTopRight, borderRadiusBottomRight, borderRadiusBottomLeft float64
	transform                                                                                  Matrix2D
	hasTransform                                                                               bool
}

func styleStoreKeyFor(style ResolvedStyle) styleStoreKey {
	return styleStoreKey{
		display: style.Display, position: style.Position, float: style.Float, clear: style.Clear,
		boxSizing: style.BoxSizing, fontHash: hashFontFamily(style.FontFamily), fontSize: style.FontSize,
		lineHeight: style.LineHeight, lineHeightUnitless: style.LineHeightUnitless,
		fontWeight: style.FontWeight, fontItalic: style.FontItalic,
		color: style.Color, bgColor: style.BGColor, width: style.Width, widthPercent: style.WidthPercent,
		height: style.Height, heightPercent: style.HeightPercent, transform: style.Transform,
		borderRadius: style.BorderRadius, borderRadiusPercent: style.BorderRadiusPercent,
		borderRadiusTopLeft: style.BorderRadiusTopLeft, borderRadiusTopRight: style.BorderRadiusTopRight,
		borderRadiusBottomRight: style.BorderRadiusBottomRight, borderRadiusBottomLeft: style.BorderRadiusBottomLeft,
		hasTransform: style.HasTransform,
	}
}

// intern returns a stable canonical pointer for styles without custom
// properties. Custom-property maps are intentionally left unique until they
// have a value-semantic representation; sharing them would expose mutability.
func (s *styleStore) intern(candidate ResolvedStyle) *ResolvedStyle {
	if candidate.CustomProps != nil {
		return s.append(candidate)
	}

	key := styleStoreKeyFor(candidate)
	for _, canonical := range s.interned[key] {
		if resolvedStylesEqual(*canonical, candidate) {
			return canonical
		}
	}

	canonical := s.append(candidate)

	if s.interned == nil {
		s.interned = make(map[styleStoreKey][]*ResolvedStyle)
	}

	s.interned[key] = append(s.interned[key], canonical)

	return canonical
}

func (s *styleStore) append(style ResolvedStyle) *ResolvedStyle {
	if len(s.chunks) == 0 || len(s.chunks[len(s.chunks)-1]) == styleStoreChunkSize {
		s.chunks = append(s.chunks, make([]ResolvedStyle, 0, styleStoreChunkSize))
	}

	chunk := len(s.chunks) - 1
	s.chunks[chunk] = append(s.chunks[chunk], style)

	return &s.chunks[chunk][len(s.chunks[chunk])-1]
}

// comparableResolvedStyle contains every ResolvedStyle field except the two
// non-comparable reference fields. Keeping the projection exhaustive makes the
// exact interning comparison allocation-free while preserving used-style
// identity.
type comparableResolvedStyle struct {
	Display, Position, Float, Clear, BoxSizing                                                 string
	Top, Right, Bottom, Left                                                                   float64
	TopAuto, RightAuto, BottomAuto, LeftAuto                                                   bool
	FlexDirection, FlexWrap, JustifyContent, AlignItems, AlignContent, AlignSelf               string
	JustifyItems, JustifySelf                                                                  string
	Gap, RowGap, ColumnGap                                                                     float64
	ColumnGapNormal                                                                            bool
	ColumnCount                                                                                int
	ColumnWidth                                                                                float64
	ColumnSpan, ColumnFill                                                                     string
	FlexGrow, FlexShrink, FlexBasis, FlexBasisPercent                                          float64
	FlexOrder, ZIndex                                                                          int
	ZIndexSet                                                                                  bool
	WritingMode                                                                                string
	GridTemplateColumns, GridTemplateRows, GridTemplateAreas, GridArea                         string
	GridAutoFlow                                                                               string
	GridColumnSpan, GridColumnStart, GridRowSpan, GridRowStart                                 int
	Width, WidthPercent, Height, HeightPercent                                                 float64
	MinWidth, MinWidthPercent, MaxWidth, MaxWidthPercent                                       float64
	MinWidthSet                                                                                bool
	MinHeight, MinHeightPercent, MaxHeight                                                     float64
	Overflow                                                                                   string
	MarginTop, MarginRight, MarginBottom, MarginLeft                                           float64
	MarginTopAuto, MarginBottomAuto, MarginLeftAuto, MarginRightAuto                           bool
	PaddingTop, PaddingRight, PaddingBottom, PaddingLeft                                       float64
	BorderTop, BorderRight, BorderBottom, BorderLeft                                           border
	BorderRadius, BorderRadiusPercent                                                          float64
	BorderRadiusTopLeft, BorderRadiusTopRight, BorderRadiusBottomRight, BorderRadiusBottomLeft float64
	Color                                                                                      [3]float64
	BGColor                                                                                    [4]float64
	famHash                                                                                    uint64
	FontSize                                                                                   float64
	FontWeight                                                                                 int
	FontItalic                                                                                 bool
	LineHeight, LineHeightUnitless                                                             float64
	TextAlign, TextTransform, VerticalAlign, WhiteSpace, OverflowWrap, WordBreak               string
	TextDecoration                                                                             string
	LetterSpacing, TextIndent                                                                  float64
	ListStyleType, BorderCollapse                                                              string
	BorderSpacing                                                                              float64
	TableLayout                                                                                string
	IsReplaced                                                                                 bool
	PageBreakBefore, PageBreakAfter, PageBreakInside                                           string
	Orphans, Widows                                                                            int
	ContainerType, ContainerName                                                               string
	Transform                                                                                  Matrix2D
	HasTransform                                                                               bool
	TransformOrigin                                                                            transformOriginSpec
	Opacity                                                                                    float64
}

func comparableResolvedStyleFor(style ResolvedStyle) comparableResolvedStyle {
	return comparableResolvedStyle{
		Display: style.Display, Position: style.Position, Float: style.Float, Clear: style.Clear,
		BoxSizing: style.BoxSizing, Top: style.Top, Right: style.Right, Bottom: style.Bottom, Left: style.Left,
		TopAuto: style.TopAuto, RightAuto: style.RightAuto, BottomAuto: style.BottomAuto, LeftAuto: style.LeftAuto,
		FlexDirection: style.FlexDirection, FlexWrap: style.FlexWrap, JustifyContent: style.JustifyContent,
		AlignItems: style.AlignItems, AlignContent: style.AlignContent, AlignSelf: style.AlignSelf,
		JustifyItems: style.JustifyItems, JustifySelf: style.JustifySelf, Gap: style.Gap, RowGap: style.RowGap,
		ColumnGap: style.ColumnGap, ColumnGapNormal: style.ColumnGapNormal, ColumnCount: style.ColumnCount,
		ColumnWidth: style.ColumnWidth, ColumnSpan: style.ColumnSpan, ColumnFill: style.ColumnFill,
		FlexGrow: style.FlexGrow, FlexShrink: style.FlexShrink, FlexBasis: style.FlexBasis,
		FlexBasisPercent: style.FlexBasisPercent, FlexOrder: style.FlexOrder, ZIndex: style.ZIndex,
		ZIndexSet: style.ZIndexSet, WritingMode: style.WritingMode, GridTemplateColumns: style.GridTemplateColumns,
		GridTemplateRows: style.GridTemplateRows, GridTemplateAreas: style.GridTemplateAreas, GridArea: style.GridArea,
		GridAutoFlow: style.GridAutoFlow, GridColumnSpan: style.GridColumnSpan, GridColumnStart: style.GridColumnStart,
		GridRowSpan: style.GridRowSpan, GridRowStart: style.GridRowStart, Width: style.Width,
		WidthPercent: style.WidthPercent, Height: style.Height, HeightPercent: style.HeightPercent,
		MinWidth: style.MinWidth, MinWidthPercent: style.MinWidthPercent, MaxWidth: style.MaxWidth,
		MaxWidthPercent: style.MaxWidthPercent, MinHeight: style.MinHeight,
		MinWidthSet:      style.MinWidthSet,
		MinHeightPercent: style.MinHeightPercent, MaxHeight: style.MaxHeight, Overflow: style.Overflow,
		MarginTop: style.MarginTop, MarginRight: style.MarginRight, MarginBottom: style.MarginBottom,
		MarginLeft: style.MarginLeft, MarginTopAuto: style.MarginTopAuto, MarginBottomAuto: style.MarginBottomAuto,
		MarginLeftAuto: style.MarginLeftAuto, MarginRightAuto: style.MarginRightAuto,
		PaddingTop: style.PaddingTop, PaddingRight: style.PaddingRight, PaddingBottom: style.PaddingBottom,
		PaddingLeft: style.PaddingLeft, BorderTop: style.BorderTop, BorderRight: style.BorderRight,
		BorderBottom: style.BorderBottom, BorderLeft: style.BorderLeft,
		BorderRadius: style.BorderRadius, BorderRadiusPercent: style.BorderRadiusPercent,
		BorderRadiusTopLeft: style.BorderRadiusTopLeft, BorderRadiusTopRight: style.BorderRadiusTopRight,
		BorderRadiusBottomRight: style.BorderRadiusBottomRight, BorderRadiusBottomLeft: style.BorderRadiusBottomLeft,
		Color: style.Color, BGColor: style.BGColor,
		famHash: style.famHash, FontSize: style.FontSize, FontWeight: style.FontWeight, FontItalic: style.FontItalic,
		LineHeight: style.LineHeight, LineHeightUnitless: style.LineHeightUnitless,
		TextAlign: style.TextAlign, TextTransform: style.TextTransform,
		VerticalAlign: style.VerticalAlign,
		WhiteSpace:    style.WhiteSpace, OverflowWrap: style.OverflowWrap, WordBreak: style.WordBreak,
		TextDecoration: style.TextDecoration, LetterSpacing: style.LetterSpacing, TextIndent: style.TextIndent,
		ListStyleType: style.ListStyleType, BorderCollapse: style.BorderCollapse, BorderSpacing: style.BorderSpacing,
		TableLayout: style.TableLayout, IsReplaced: style.IsReplaced, PageBreakBefore: style.PageBreakBefore,
		PageBreakAfter: style.PageBreakAfter, PageBreakInside: style.PageBreakInside, Orphans: style.Orphans,
		Widows: style.Widows, ContainerType: style.ContainerType, ContainerName: style.ContainerName,
		Transform: style.Transform, HasTransform: style.HasTransform, TransformOrigin: style.TransformOrigin,
		Opacity: style.Opacity,
	}
}

// resolvedStylesEqual is deliberately exact. The coarse key only selects a
// candidate bucket, so a hash collision cannot cause two used styles to share.
func resolvedStylesEqual(left, right ResolvedStyle) bool {
	if left.CustomProps != nil || right.CustomProps != nil ||
		(left.FontFamily == nil) != (right.FontFamily == nil) ||
		len(left.FontFamily) != len(right.FontFamily) {
		return false
	}

	for idx := range left.FontFamily {
		if left.FontFamily[idx] != right.FontFamily[idx] {
			return false
		}
	}

	return comparableResolvedStyleFor(left) == comparableResolvedStyleFor(right)
}

// zeroResolvedStyle is the empty style for comment/doctype nodes (shared).
var zeroResolvedStyle ResolvedStyle //nolint:gochecknoglobals // immutable zero sentinel

// resolveElementStyle cascades one element: inheritance, custom properties,
// fonts, the remaining properties, and the operator/blockify policies.
func resolveElementStyle(
	node *html.Node, ctx *styleContext, parent, sty *ResolvedStyle,
) {
	raw := cascadeRaw(ctx, node)
	*sty = initialStyle()

	var parentProps map[string]string

	if parent != nil {
		inheritProps(sty, parent, raw)
		parentProps = parent.CustomProps
	}

	sty.CustomProps = mergeCustomProps(parentProps, raw)
	raw = resolveRawVars(raw, sty.CustomProps)

	parentSize := sty.FontSize
	if parent != nil {
		parentSize = parent.FontSize
	}

	applyFontProps(sty, raw, parentSize, ctx)

	if node.Name == "html" && sty.FontSize > 0 {
		ctx.remBase = sty.FontSize
	}

	applyRestProps(sty, raw, ctx, parent)
	inheritUnitlessLineHeight(sty, parent, raw)
	// Opt-in operator policy (--print-link-underline): underline
	// anchors with href after the cascade. Default off — author CSS
	// (including text-decoration: inherit → parent) wins otherwise.
	if ctx != nil && ctx.printLinkUnderline && node.Name == "a" && strings.TrimSpace(node.Attribute("href")) != "" {
		sty.TextDecoration = cssTextDecorationUnderline
	}
	// CSS2.1 §9.7: float ≠ none blockifies table-internal / inline
	// displays before layout (table/flex/grid stay). Floated <table>
	// keeps display:table so fixture-29 wrapper packing still works.
	if sty.Float != cssDisplayNone {
		sty.Display = blockifyDisplayForFloat(sty.Display)
	}
	// FontFamily is final here (inherited, or parsed by applyFontProps /
	// parseFontShorthand); fingerprint it once so inline text measurement
	// does not re-hash the family list per run.
	sty.famHash = hashFontFamily(sty.FontFamily)
}

// hasExplicitLineHeight reports whether a declaration sets line-height either
// directly or through a font shorthand containing a slash value.
func hasExplicitLineHeight(raw map[string]string) bool {
	if _, ok := raw["line-height"]; ok {
		return true
	}

	font, ok := raw["font"]

	return ok && strings.Contains(font, "/")
}

func inheritUnitlessLineHeight(sty, parent *ResolvedStyle, raw map[string]string) {
	if parent == nil || hasExplicitLineHeight(raw) || parent.LineHeightUnitless <= 0 {
		return
	}

	sty.LineHeightUnitless = parent.LineHeightUnitless
	sty.LineHeight = sty.FontSize * sty.LineHeightUnitless
}
