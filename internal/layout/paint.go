package layout

import (
	"gowkhtmltopdf/internal/pdf"
)

// PaintOptions describes the destination page geometry, in points.
type PaintOptions struct {
	PageWidth    float64
	PageHeight   float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
}

// Paint paginates the display list across pages and paints it into doc.
//
// Phase-04 pagination: operations are placed on the page containing their
// top edge; operations that would cross a page boundary are moved wholly to
// the next page (no fragment continuation yet — that is phase 05).
func Paint(doc *pdf.Document, res *Result, opts PaintOptions) error {
	if doc == nil || res == nil {
		return nil
	}
	contentH := opts.PageHeight - opts.MarginTop - opts.MarginBottom
	if contentH <= 0 {
		contentH = opts.PageHeight
	}
	if len(res.Ops) == 0 {
		doc.AddPage(opts.PageWidth, opts.PageHeight)
		return nil
	}

	// group ops by page
	pages := [][]Op{}
	for _, op := range res.Ops {
		page := int(op.Y / contentH)
		if page < 0 {
			page = 0
		}
		opH := op.H
		if op.Kind == OpText || op.Kind == OpBullet {
			opH = op.Size * 1.2
		}
		// ops crossing a page boundary move wholly to the next page
		if op.Y+opH > float64(page+1)*contentH {
			page++
		}
		for page >= len(pages) {
			pages = append(pages, nil)
		}
		pages[page] = append(pages[page], op)
	}

	for pageIdx, ops := range pages {
		p := doc.AddPage(opts.PageWidth, opts.PageHeight)
		c := p.Content()
		fontUsed := false
		for _, op := range ops {
			switch op.Kind {
			case OpFillRect:
				drawFill(c, &op, pageIdx, contentH, opts, p.Height())
			case OpStrokeRect:
				drawStroke(c, &op, pageIdx, contentH, opts, p.Height())
			case OpLine:
				drawLine(c, &op, pageIdx, contentH, opts, p.Height())
			case OpText, OpBullet:
				if !fontUsed {
					c.UseEmbeddedFont("F0", op.Font)
					fontUsed = true
				}
				drawText(c, &op, pageIdx, contentH, opts, p.Height())
			case OpImage:
				drawImage(p, c, &op, pageIdx, contentH, opts)
			case OpLinkURI:
				drawLink(p, &op, pageIdx, contentH, opts)
			}
		}
	}
	return nil
}

// canvasToPDF converts a canvas point (y down, origin at content top-left of
// page 0) to PDF coordinates on the given page.
func canvasToPDF(opX, opY float64, pageIdx int, contentH float64, opts PaintOptions, pageH float64) (x, y float64) {
	x = opts.MarginLeft + opX
	y = pageH - opts.MarginTop - opY + float64(pageIdx)*contentH
	return x, y
}

func drawFill(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	if op.Alpha < 1 {
		c.SetOpacity(op.Alpha)
	}
	c.SetFillColor(op.R, op.G, op.B)
	c.Rect(x, y, op.W, op.H)
	c.Fill()
}

func drawStroke(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	c.SetStrokeColor(op.R, op.G, op.B)
	c.SetLineWidth(1)
	c.Rect(x, y, op.W, op.H)
	c.Stroke()
}

func drawLine(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x1, y1 := canvasToPDF(op.X, op.Y, pageIdx, contentH, opts, pageH)
	x2, y2 := canvasToPDF(op.X+op.W, op.Y+op.H, pageIdx, contentH, opts, pageH)
	w := op.Width
	if w <= 0 {
		w = 1
	}
	c.SetStrokeColor(op.R, op.G, op.B)
	c.SetLineWidth(w)
	c.MoveTo(x1, y1)
	c.LineTo(x2, y2)
	c.Stroke()
}

func drawText(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y, pageIdx, contentH, opts, pageH)
	c.SetFillColor(op.R, op.G, op.B)
	c.SetFont("F0", op.Size)
	c.BeginText()
	c.TextAt(x, y)
	if op.Bold {
		c.SetLineWidth(op.Size * 0.06)
		c.TextRenderMode(2) // fill + stroke: fake bold
	}
	c.TextShow(op.Text)
	if op.Bold {
		c.TextRenderMode(0)
	}
	c.EndText()
}

func drawImage(p *pdf.Page, c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, p.Height())
	if op.IsJPEG {
		_ = c.AddJPEGImage("I0", x, y, op.W, op.H, op.Image)
	} else {
		_ = c.AddPNGImage("I0", x, y, op.W, op.H, op.Image)
	}
}

func drawLink(p *pdf.Page, op *Op, pageIdx int, contentH float64, opts PaintOptions) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, p.Height())
	p.AddLinkURI([4]float64{x, y, x + op.W, y + op.H}, op.URI)
}
