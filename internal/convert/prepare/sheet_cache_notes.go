package prepare

import (
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// inlineSheet returns the sheet parsed for one <style> node, or nil.
func (c *SheetCache) inlineSheet(node *html.Node) *css.Stylesheet {
	if c == nil {
		return nil
	}

	return c.inline[node]
}

// noteInline stores one parsed <style> sheet.
func (c *SheetCache) noteInline(node *html.Node, sheet *css.Stylesheet) {
	if c == nil || sheet == nil {
		return
	}

	c.inline[node] = sheet
}

// linkSheet returns the sheet fetched for one <link> node, or the zero value.
func (c *SheetCache) linkSheet(node *html.Node) cachedSheet {
	if c == nil {
		return cachedSheet{} //nolint:exhaustruct // no cached sheet
	}

	return c.links[node]
}

// noteLink stores one fetched <link> sheet.
func (c *SheetCache) noteLink(node *html.Node, sheet cachedSheet) {
	if c == nil {
		return
	}

	c.links[node] = sheet
}

// importSheet returns the sheet fetched for one @import rule, or the zero value.
func (c *SheetCache) importSheet(key importKey) cachedSheet {
	if c == nil {
		return cachedSheet{} //nolint:exhaustruct // no cached sheet
	}

	return c.imports[key]
}

// noteImport stores one fetched @import sheet.
func (c *SheetCache) noteImport(key importKey, sheet cachedSheet) {
	if c == nil {
		return
	}

	c.imports[key] = sheet
}
