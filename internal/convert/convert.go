package convert

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// mmToPt converts millimetres to PostScript points.
const mmToPt = 72.0 / 25.4

// RunPDF executes the full pdf conversion: load every object, lay it out,
// paint all objects into one document, and write the result.
func RunPDF(cmd *cli.Command, log io.Writer) error {
	ctx := context.Background()

	loader := load.NewLoader(cmd.Global.Load)
	loader.Log = log
	loader.Allow = cmd.Global.Allow
	loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}

	doc := pdf.NewDocument()
	for i := range cmd.Objects {
		if err := renderObject(ctx, loader, font, doc, cmd, &cmd.Objects[i], i, log); err != nil {
			return err
		}
	}

	if cmd.Global.Title != "" {
		doc.SetInfo("Title", cmd.Global.Title)
	}
	doc.SetInfo("Producer", "gowkhtmltopdf")
	doc.SetCompression(cmd.Global.UseCompression)
	doc.SetGrayscale(cmd.Global.Grayscale)
	doc.SetCreationTime(time.Now())

	var out io.Writer = os.Stdout
	closeOut := func() error { return nil }
	if cmd.Output != "" && cmd.Output != "-" {
		f, err := os.Create(cmd.Output)
		if err != nil {
			return fmt.Errorf("output %q: %w", cmd.Output, err)
		}
		out = f
		closeOut = f.Close
	}
	if err := doc.Write(out); err != nil {
		closeOut()
		return fmt.Errorf("write %q: %w", cmd.Output, err)
	}
	return closeOut()
}

// renderObject loads, lays out and paints one page object into doc.
func renderObject(ctx context.Context, loader *load.Loader, font *pdf.Font, doc *pdf.Document, cmd *cli.Command, obj *settings.PdfObject, idx int, log io.Writer) error {
	if obj.IsTableOfContent {
		fmt.Fprintf(log, "warning: object %d: table-of-contents objects are not supported yet, skipping\n", idx+1)
		return nil
	}

	res, err := loader.Load(ctx, obj.Page, obj.Load)
	if err != nil {
		return fmt.Errorf("object %d (%s): load: %w", idx+1, obj.Page, err)
	}
	if res.Skip {
		fmt.Fprintf(log, "warning: object %d (%s): load error policy is skip, omitting\n", idx+1, obj.Page)
		return nil
	}

	root, err := html.Parse(string(res.Body))
	if err != nil {
		return fmt.Errorf("object %d (%s): parse html: %w", idx+1, obj.Page, err)
	}

	sheets := collectSheets(ctx, loader, root, res.Base, obj.Load, idx+1, log)

	pageW, pageH, err := pageGeometry(cmd.Global)
	if err != nil {
		return fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}
	mT := cmd.Global.Margin.Top * mmToPt
	mB := cmd.Global.Margin.Bottom * mmToPt
	mL := cmd.Global.Margin.Left * mmToPt
	mR := cmd.Global.Margin.Right * mmToPt

	imagesFn := func(src string) ([]byte, error) {
		r, err := loader.FetchSub(ctx, res.Base, src, obj.Load)
		if err != nil {
			return nil, err
		}
		return r.Body, nil
	}

	lres, err := layout.Layout(root, layout.Options{
		Width:      pageW - mL - mR,
		Height:     pageH - mT - mB,
		Font:       font,
		Sheets:     sheets,
		Media:      "print",
		Images:     imagesFn,
		Background: cmd.Global.Background,
	})
	if err != nil {
		return fmt.Errorf("object %d (%s): layout: %w", idx+1, obj.Page, err)
	}

	if err := layout.Paint(doc, lres, layout.PaintOptions{
		PageWidth:    pageW,
		PageHeight:   pageH,
		MarginTop:    mT,
		MarginBottom: mB,
		MarginLeft:   mL,
		MarginRight:  mR,
	}); err != nil {
		return fmt.Errorf("object %d (%s): paint: %w", idx+1, obj.Page, err)
	}
	return nil
}

// pageGeometry resolves the page size in points. Explicit size.width/height
// (mm) win over a named size; landscape swaps the pair.
func pageGeometry(g settings.PdfGlobal) (w, h float64, err error) {
	name := g.PageSize
	if name == "" {
		name = g.Size.PageSize
	}
	if g.Size.Width > 0 && g.Size.Height > 0 {
		w, h = g.Size.Width*mmToPt, g.Size.Height*mmToPt
	} else {
		w, h, err = settings.ParsePageSize(name)
		if err != nil {
			return 0, 0, err
		}
	}
	if g.Orientation == settings.OrientationLandscape {
		w, h = h, w
	}
	return w, h, nil
}

// collectSheets gathers <style> blocks and <link rel="stylesheet"> resources
// from the DOM. A failed stylesheet only logs a warning; the layout proceeds
// without it.
func collectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, idx int, log io.Writer) []*css.Stylesheet {
	var sheets []*css.Stylesheet
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		switch n.Name {
		case "style":
			sheet, err := css.Parse(styleText(n))
			if err != nil {
				fmt.Fprintf(log, "warning: object %d: skipping <style>: %v\n", idx, err)
			} else if sheet != nil {
				sheets = append(sheets, sheet)
			}
			return // raw-text element; no element children
		case "link":
			if linkStylesheet(n) {
				href := n.Attribute("href")
				r, err := loader.FetchSub(ctx, base, href, lp)
				if err != nil {
					fmt.Fprintf(log, "warning: object %d: skipping <link href=%q>: %v\n", idx, href, err)
					return
				}
				sheet, err := css.Parse(string(r.Body))
				if err != nil {
					fmt.Fprintf(log, "warning: object %d: skipping <link href=%q>: %v\n", idx, href, err)
					return
				}
				sheets = append(sheets, sheet)
			}
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return sheets
}

// styleText concatenates the raw text of a <style> element.
func styleText(n *html.Node) string {
	var sb strings.Builder
	for _, c := range n.Children {
		if c.Type == html.TextNode {
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

// linkStylesheet reports whether n is a stylesheet <link> whose media
// attribute allows print output: empty, or containing "print" or "all".
func linkStylesheet(n *html.Node) bool {
	if n.Name != "link" || !strings.Contains(strings.ToLower(n.Attribute("rel")), "stylesheet") {
		return false
	}
	if n.Attribute("href") == "" {
		return false
	}
	media := strings.ToLower(n.Attribute("media"))
	return media == "" || strings.Contains(media, "print") || strings.Contains(media, "all")
}

// DefaultTOCXSL returns the default TOC stylesheet. In pure Go the default
// TOC look is a built-in Go template; this returns a description of it for
// --dump-default-toc-xsl compatibility.
func DefaultTOCXSL() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!-- gowkhtmltopdf default TOC stylesheet.
     Upstream ships an XSLT here; the pure-Go implementation uses an
     equivalent built-in template (see internal/outline/toc.go). -->
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" indent="yes"/>
  <xsl:template match="/">
    <h1>Table of Contents</h1>
    <ul id="toc"/>
  </xsl:template>
</xsl:stylesheet>
`
}
