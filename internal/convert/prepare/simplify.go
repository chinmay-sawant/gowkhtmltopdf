package prepare

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const profileMediaWiki = "mediawiki"

const SimplifyChromeCSS = `
nav, footer, aside,
[role="navigation"], [role="contentinfo"], [role="complementary"],
nav.site-nav {
  display: none !important;
}
`

const SimplifyMediaWikiCSS = `
#mw-navigation,
.mw-jump-link {
  display: none !important;
}
`

func SimplifyDOMEnabled(global, object settings.Web) bool {
	return global.SimplifyDOM || object.SimplifyDOM
}

func normalizeSimplifyProfile(raw string) string {
	profile := strings.ToLower(strings.TrimSpace(raw))
	if profile == profileMediaWiki || profile == "wiki" || profile == "mw" {
		return profileMediaWiki
	}

	return ""
}

func SimplifyDOMProfile(global, object settings.Web) string {
	if profile := normalizeSimplifyProfile(object.SimplifyDOMProfile); profile != "" {
		return profile
	}

	return normalizeSimplifyProfile(global.SimplifyDOMProfile)
}

// BuildOptions constructs shared prepare Options from viewport/media and Web
// layers. Layers are ordered lowest→highest SimplifyDOMProfile priority
// (typically global, optional image web, then object). SimplifyDOM is true
// when any layer enables it.
func BuildOptions(viewportW, viewportH float64, media string, objectIndex int, webs ...settings.Web) Options {
	opts := Options{ //nolint:exhaustruct // Simplify fields filled below
		ViewportW:   viewportW,
		ViewportH:   viewportH,
		MediaType:   media,
		ObjectIndex: objectIndex,
	}

	for _, web := range webs {
		if web.SimplifyDOM {
			opts.SimplifyDOM = true
		}
	}

	for i := len(webs) - 1; i >= 0; i-- {
		if profile := normalizeSimplifyProfile(webs[i].SimplifyDOMProfile); profile != "" {
			opts.SimplifyProfile = profile

			break
		}
	}

	return opts
}

//nolint:wsl,nlreturn // stylesheet append flow
func AppendSimplifySheet(sheets []*css.Stylesheet, enabled bool, profile string) []*css.Stylesheet {
	if !enabled {
		return sheets
	}

	sheet, err := css.Parse(SimplifyChromeCSS)
	if err != nil || sheet == nil {
		return sheets
	}
	sheets = append(sheets, sheet)
	if strings.EqualFold(strings.TrimSpace(profile), profileMediaWiki) {
		mw, err := css.Parse(SimplifyMediaWikiCSS)
		if err == nil && mw != nil {
			sheets = append(sheets, mw)
		}
	}
	return sheets
}
