package convert

import (
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/settings"
)

// SimplifyChromeCSS is the opt-in chrome-strip stylesheet injected when
// --simplify-dom / web.simplifydom is on. Approach: append display:none
// rules so chrome nodes stay in the DOM tree but are excluded from layout
// (not removed). No extra origins are fetched — the sheet is synthetic.
//
// Rules (each documented for phase 21.4):
//  1. nav, footer, aside — common landmark chrome tags
//  2. [role=navigation|contentinfo|complementary] — ARIA landmark roles
//  3. #mw-navigation — MediaWiki Vector/Minerva nav container
//  4. .mw-jump-link — MediaWiki skip-to-content links
//  5. nav.site-nav — common site-nav class (also matches rule 1; kept explicit)
//
// !important beats author styles that would keep chrome visible in print.
const SimplifyChromeCSS = `
nav, footer, aside,
[role="navigation"], [role="contentinfo"], [role="complementary"],
#mw-navigation,
.mw-jump-link,
nav.site-nav {
  display: none !important;
}
`

// SimplifyDOMEnabled reports whether chrome-strip heuristics apply for this
// page (global or object web.simplifydom). Default is off.
func SimplifyDOMEnabled(global, obj settings.Web) bool {
	return global.SimplifyDOM || obj.SimplifyDOM
}

// AppendSimplifySheet appends the chrome-strip stylesheet when enabled.
// A parse failure is ignored (the constant is fixed; this is defensive).
func AppendSimplifySheet(sheets []*css.Stylesheet, enabled bool) []*css.Stylesheet {
	if !enabled {
		return sheets
	}
	sheet, err := css.Parse(SimplifyChromeCSS)
	if err != nil || sheet == nil {
		return sheets
	}
	return append(sheets, sheet)
}
