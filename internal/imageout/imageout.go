package imageout

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"strings"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// ptToPx maps layout canvas points to output pixels. The layout engine works
// in points with CSS pixels at 96 dpi (1 px = 0.75 pt, see
// layout/style.go pxToPt), so rasterizing at 1 CSS px = 1 output px means
// multiplying every point by 96/72.
const ptToPx = 96.0 / 72.0

// rasterSS is the supersample factor for the paint canvas. Ops are painted
// at rasterSS times the final resolution, then box-filtered down. This
// stabilises small-text baselines and edges (stdlib has no FreeType hinting).
const rasterSS = 2

// screenWidthDefault is the wkhtmltoimage default viewport width in pixels
// (settings.ImageGlobal.Width default is already 1024; this guards against
// 0-width RenderOptions).
const screenWidthDefault = 1024

// maxSmartViewport caps the smart-width viewport growth in pixels.
const maxSmartViewport = 4096

// RenderOptions controls one Render call. Width/Height and Crop are in
// output pixels; the layout viewport is Width CSS pixels at 96 dpi.
type RenderOptions struct {
	Width       int // viewport width in pixels; <= 0 means 1024
	Height      int // minimum canvas height in pixels; 0 = content height
	Font        *pdf.Font
	Registry    *pdf.Registry // optional --font-path / system faces (CJK)
	Sheets      []*css.Stylesheet
	Media       string // "screen" (default), "print" or ""
	Images      func(src string) ([]byte, error)
	Background  bool // paint background colors
	Transparent bool // PNG background: alpha 0 instead of white
	Crop        image.Rectangle
	SmartWidth  bool // grow the viewport until content fits (default on)
	// PrintLinkUnderline mirrors --print-link-underline (opt-in).
	PrintLinkUnderline bool
}

// Render lays out root and rasterizes the result. The canvas is the viewport
// (or, with SmartWidth, the smallest grown viewport that fits the content)
// wide and max(content height, Height) tall. A non-empty Crop is applied to
// the rasterized canvas.
func Render(root *html.Node, opts RenderOptions) (image.Image, error) {
	if root == nil {
		return nil, errors.New("imageout: nil root")
	}
	font := opts.Font
	if font == nil {
		var err error
		font, err = pdf.DefaultFont()
		if err != nil {
			return nil, fmt.Errorf("imageout: default font: %w", err)
		}
	}

	var res *layout.Result
	var err error
	if opts.SmartWidth {
		res, err = layoutSmartWidth(root, opts, font)
	} else {
		viewport := float64(opts.Width)
		if viewport <= 0 {
			viewport = screenWidthDefault
		}
		res, err = layout.Layout(root, layoutOptions(opts, font, viewport))
	}
	if err != nil {
		return nil, fmt.Errorf("imageout: layout: %w", err)
	}

	img := rasterize(res, maxHeight(res, opts), opts.Transparent)
	var out image.Image = img
	if crop := opts.Crop; !crop.Empty() {
		inter := crop.Intersect(img.Bounds())
		if inter.Empty() {
			return nil, errors.New("imageout: crop rectangle does not intersect the canvas")
		}
		// re-origin the crop to (0,0): SubImage keeps the canvas
		// coordinate system, which is awkward for library callers
		out = reOrigin(img.SubImage(inter))
	}
	return out, nil
}

// reOrigin copies src into a fresh image whose bounds start at (0,0).
func reOrigin(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// layoutOptions builds layout.Options for a viewport of viewportPx pixels.
func layoutOptions(opts RenderOptions, font *pdf.Font, viewportPx float64) layout.Options {
	heightPt := float64(opts.Height) * 0.75
	if heightPt <= 0 {
		heightPt = viewportPx * 0.75
	}
	return layout.Options{
		Width:              viewportPx * 0.75,
		Height:             heightPt,
		Font:               font,
		Registry:           opts.Registry,
		Sheets:             opts.Sheets,
		Media:              opts.Media,
		Images:             opts.Images,
		Background:         opts.Background,
		PrintLinkUnderline: opts.PrintLinkUnderline,
	}
}

// layoutSmartWidth lays out repeatedly, growing the viewport by 1.5x while
// painted content overflows the right edge (the layout engine always fills
// the full viewport width, so overflow is measured from the display list:
// max op.X+op.W). Growth is capped at maxSmartViewport pixels.
func layoutSmartWidth(root *html.Node, opts RenderOptions, font *pdf.Font) (*layout.Result, error) {
	viewport := float64(opts.Width)
	if viewport <= 0 {
		viewport = screenWidthDefault
	}
	var res *layout.Result
	for i := 0; i < 8; i++ {
		var err error
		res, err = layout.Layout(root, layoutOptions(opts, font, viewport))
		if err != nil {
			return nil, err
		}
		if viewport >= maxSmartViewport || contentWidthPx(res) <= viewport+0.5 {
			return res, nil
		}
		viewport *= 1.5
		if viewport > maxSmartViewport {
			viewport = maxSmartViewport
		}
	}
	return res, nil
}

// contentWidthPx returns the rightmost painted edge of the display list,
// converted to pixels. Link annotations do not paint and are ignored.
func contentWidthPx(res *layout.Result) float64 {
	maxW := 0.0
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == layout.OpLinkURI {
			continue
		}
		if e := op.X + op.W; e > maxW {
			maxW = e
		}
	}
	return maxW * ptToPx
}

// maxHeight resolves the canvas height: the larger of the laid-out content
// height and the requested minimum (--height). layout.Result.Height reports
// the content height only, so the minimum must be applied here.
func maxHeight(res *layout.Result, opts RenderOptions) float64 {
	h := res.Height
	if hp := float64(opts.Height) * 0.75; hp > h {
		h = hp
	}
	return h
}

// rasterize paints the display list into an NRGBA canvas. The canvas is
// white unless transparent is set, in which case it starts fully transparent
// and only painted ops become visible. Painting uses rasterSS supersampling
// then box-filters down to the final CSS-pixel size.
func rasterize(res *layout.Result, height float64, transparent bool) *image.NRGBA {
	pxPerPt := ptToPx * float64(rasterSS)
	w := int(math.Round(res.Width * pxPerPt))
	h := int(math.Round(height * pxPerPt))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if !transparent {
		draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	}
	for i := range res.Ops {
		paint(img, &res.Ops[i], pxPerPt)
	}
	if rasterSS <= 1 {
		return img
	}
	return downscaleBox(img, rasterSS)
}

// downscaleBox averages factor×factor blocks of src into one output pixel.
func downscaleBox(src *image.NRGBA, factor int) *image.NRGBA {
	if factor <= 1 {
		return src
	}
	sb := src.Bounds()
	dw := sb.Dx() / factor
	dh := sb.Dy() / factor
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	n := uint32(factor * factor)
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var r, g, b, a uint32
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					c := src.NRGBAAt(sb.Min.X+x*factor+dx, sb.Min.Y+y*factor+dy)
					r += uint32(c.R)
					g += uint32(c.G)
					b += uint32(c.B)
					a += uint32(c.A)
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n),
			})
		}
	}
	return dst
}

// paint draws one display-list operation onto the canvas. pxPerPt converts
// layout points into the (possibly supersampled) canvas pixel space.
func paint(img *image.NRGBA, op *layout.Op, pxPerPt float64) {
	switch op.Kind {
	case layout.OpFillRect:
		c := color.NRGBA{
			R: uint8(op.R * 255), G: uint8(op.G * 255), B: uint8(op.B * 255),
			A: uint8(op.Alpha * 255),
		}
		r := ptRectScale(op.X, op.Y, op.W, op.H, pxPerPt).Intersect(img.Bounds())
		if !r.Empty() {
			draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Over)
		}

	case layout.OpStrokeRect:
		c := color.NRGBA{
			R: uint8(op.R * 255), G: uint8(op.G * 255), B: uint8(op.B * 255), A: 255,
		}
		lw := strokeWidthScale(op, pxPerPt)
		r := ptRectScale(op.X, op.Y, op.W, op.H, pxPerPt)
		rects := [4]image.Rectangle{
			image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+lw),
			image.Rect(r.Min.X, r.Max.Y-lw, r.Max.X, r.Max.Y),
			image.Rect(r.Min.X, r.Min.Y, r.Min.X+lw, r.Max.Y),
			image.Rect(r.Max.X-lw, r.Min.Y, r.Max.X, r.Max.Y),
		}
		for _, rr := range rects {
			if rr = rr.Intersect(img.Bounds()); !rr.Empty() {
				draw.Draw(img, rr, image.NewUniform(c), image.Point{}, draw.Over)
			}
		}

	case layout.OpLine:
		c := color.NRGBA{
			R: uint8(op.R * 255), G: uint8(op.G * 255), B: uint8(op.B * 255), A: 255,
		}
		lw := strokeWidthScale(op, pxPerPt)
		// centre the stroke on the line: half its width, in points
		half := float64(lw) / 2 / pxPerPt
		var r image.Rectangle
		if op.H <= 0 { // horizontal line
			r = ptRectScale(op.X, op.Y-half, op.W, 2*half, pxPerPt)
		} else { // vertical line
			r = ptRectScale(op.X-half, op.Y, 2*half, op.H, pxPerPt)
		}
		if r = r.Intersect(img.Bounds()); !r.Empty() {
			draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Over)
		}

	case layout.OpText, layout.OpBullet:
		c := color.NRGBA{
			R: uint8(op.R * 255), G: uint8(op.G * 255), B: uint8(op.B * 255), A: 255,
		}
		// Keep fractional baselines so glyphs share one stable baseline
		// instead of independently rounded Y positions (bobbing text).
		bx := op.X * pxPerPt
		by := op.Y * pxPerPt
		face := op.Font
		if face == nil {
			// Layout always attaches a face when DefaultFont is available;
			// this is defensive only (no 5×7 bitmap dual path).
			var err error
			face, err = pdf.DefaultFont()
			if err != nil || face == nil {
				return
			}
		}
		ttfDrawString(img, bx, by, op.Text, op.Size, face, c, pxPerPt)
		// ponytail: fake-bold double-draw when CSS weight wants bold but the
		// face is regular; upgrade when synthetic bold outlines land in pdf.
		if op.Bold && !face.Bold() {
			ttfDrawString(img, bx+float64(rasterSS), by, op.Text, op.Size, face, c, pxPerPt)
		}

	case layout.OpImage:
		var src image.Image
		var err error
		if op.IsJPEG {
			src, err = jpeg.Decode(bytes.NewReader(op.Image))
		} else {
			src, err = png.Decode(bytes.NewReader(op.Image))
		}
		if err != nil || op.W <= 0 || op.H <= 0 {
			return // layout already validated the bytes; skip on failure
		}
		r := ptRectScale(op.X, op.Y, op.W, op.H, pxPerPt).Intersect(img.Bounds())
		if !r.Empty() {
			sb := src.Bounds()
			if r.Dx() == sb.Dx() && r.Dy() == sb.Dy() {
				draw.Draw(img, r, src, sb.Min, draw.Over)
			} else {
				// Go 1.26 removed image/draw's scalers; nearest
				// neighbour keeps it stdlib-only.
				draw.Draw(img, r, scaleNearest(src, r.Dx(), r.Dy()), image.Point{}, draw.Over)
			}
		}

	case layout.OpLinkURI:
		// annotations do not paint
	}
}

// strokeWidthScale returns the stroke thickness in canvas pixels.
func strokeWidthScale(op *layout.Op, pxPerPt float64) int {
	w := op.Width
	if w <= 0 {
		w = 1
	}
	if lw := int(math.Round(w * pxPerPt)); lw >= 1 {
		return lw
	}
	return 1
}

// ptRectScale converts a point-space rectangle into canvas pixels.
func ptRectScale(x, y, w, h, pxPerPt float64) image.Rectangle {
	return image.Rect(
		int(math.Round(x*pxPerPt)), int(math.Round(y*pxPerPt)),
		int(math.Round((x+w)*pxPerPt)), int(math.Round((y+h)*pxPerPt)),
	)
}

// scaleNearest resizes src to w×h with nearest-neighbour sampling. Go 1.26
// removed image/draw's BiLinear/NearestNeighbor scalers, so a tiny scaler
// lives here; natural-size images take the draw.Draw fast path in paint.
func scaleNearest(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	if sb.Dx() == 0 || sb.Dy() == 0 {
		return dst
	}
	fx := float64(sb.Dx()) / float64(w)
	fy := float64(sb.Dy()) / float64(h)
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + int((float64(y)+0.5)*fy)
		if sy > sb.Max.Y-1 {
			sy = sb.Max.Y - 1
		}
		for x := 0; x < w; x++ {
			sx := sb.Min.X + int((float64(x)+0.5)*fx)
			if sx > sb.Max.X-1 {
				sx = sb.Max.X - 1
			}
			dst.SetNRGBA(x, y, color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA))
		}
	}
	return dst
}

// Run drives the gowkhtmltoimage command: load the first page object, render
// it with Render, encode as PNG or JPEG (--format / output extension, quality
// from --quality) and write to the output path or stdout ("-" or "").
func Run(ctx context.Context, cmd *cli.Command, log io.Writer) error {
	if log == nil {
		log = io.Discard
	}
	// Policy A: one quiet bit — CLI --quiet sets Global.Quiet (not Image.Quiet).
	if cmd.Global.Quiet {
		log = io.Discard
	}
	obj, err := firstObject(cmd, log)
	if err != nil {
		return err
	}

	loader := load.NewLoader(cmd.Image.Load)
	loader.Log = log
	loader.Allow = cmd.Global.Allow
	loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}
	var registry *pdf.Registry
	if dirs := cmd.Global.FontPaths; len(dirs) > 0 || cmd.Global.UseSystemFonts {
		scan := append([]string{}, dirs...)
		if cmd.Global.UseSystemFonts {
			scan = append(scan, pdf.DefaultSystemFontDirs()...)
		}
		registry = pdf.ScanFontDirs(scan)
		if log != io.Discard && len(scan) > 0 {
			fmt.Fprintf(log, "info: scanned %d font path(s)\n", len(scan))
		}
	}

	res, err := loader.Load(ctx, obj.Page, obj.Load)
	if err != nil {
		return fmt.Errorf("load %q: %w", obj.Page, err)
	}
	if res.Skip {
		return fmt.Errorf("load %q: load-error policy is skip; nothing to render", obj.Page)
	}

	root, err := html.Parse(string(res.Body))
	if err != nil {
		return fmt.Errorf("parse html: %w", err)
	}

	sheets := collectSheets(ctx, loader, root, res.Base, obj.Load, log)
	enabled := convert.SimplifyDOMEnabled(cmd.Image.Web, obj.Web) || cmd.Global.Web.SimplifyDOM
	profile := convert.SimplifyDOMProfile(cmd.Image.Web, obj.Web)
	if profile == "" {
		profile = convert.SimplifyDOMProfile(cmd.Global.Web, settings.Web{})
	}
	sheets = convert.AppendSimplifySheet(sheets, enabled, profile)
	registry = convert.MergeFontFaces(ctx, loader, registry, sheets, res.Base, obj.Load, 1, log)

	cache := map[string][]byte{}
	imagesFn := func(src string) ([]byte, error) {
		if !cmd.Image.Web.Images {
			return nil, fmt.Errorf("images disabled")
		}
		if b, ok := cache[src]; ok {
			return b, nil
		}
		r, err := loader.FetchSub(ctx, res.Base, src, obj.Load)
		if err != nil {
			return nil, err
		}
		cache[src] = r.Body
		return r.Body, nil
	}

	// Policy A: Quiet is Global.Quiet; body paint background is Global.Background
	// only (single field for PDF + image; CLI --background / library Set).
	img, err := Render(root, RenderOptions{
		Width:              cmd.Image.Width,
		Height:             cmd.Image.Height,
		Font:               font,
		Registry:           registry,
		Sheets:             sheets,
		Media:              mediaFor(cmd, obj),
		Images:             imagesFn,
		Background:         cmd.Global.Background,
		Transparent:        cmd.Image.Transparent,
		Crop:               cropRect(cmd.Image.Crop),
		SmartWidth:         cmd.Image.SmartWidth,
		PrintLinkUnderline: cmd.Image.Web.PrintLinkUnderline || cmd.Global.Web.PrintLinkUnderline || obj.Web.PrintLinkUnderline,
	})
	if err != nil {
		return err
	}

	format, err := resolveFormat(cmd.Image.Format, cmd.Output)
	if err != nil {
		return err
	}
	if format == "jpg" && cmd.Image.Transparent {
		fmt.Fprintln(log, "warning: --transparent ignored for JPEG output (white background used)")
		img = onWhite(img)
	}
	data, err := encode(img, format, cmd.Image.Quality)
	if err != nil {
		return fmt.Errorf("encode %s: %w", format, err)
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return err
	}
	if _, err := out.Write(data); err != nil {
		closeOut()
		if cmd.OutputWriter != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return fmt.Errorf("write %q: %w", cmd.Output, err)
	}
	return closeOut()
}

// firstObject returns the first page-like object. Image mode renders a
// single page; extra page objects and TOC objects are ignored with warnings.
func firstObject(cmd *cli.Command, log io.Writer) (*settings.PdfObject, error) {
	var first *settings.PdfObject
	for i := range cmd.Objects {
		o := &cmd.Objects[i]
		if o.IsTableOfContent || o.Page == "" {
			continue
		}
		if first == nil {
			first = o
			continue
		}
		fmt.Fprintf(log, "warning: image mode renders the first page only; ignoring object %d\n", i+1)
	}
	if first == nil {
		return nil, errors.New("no input to convert")
	}
	return first, nil
}

// mediaFor resolves the layout media: screen (browser default) unless
// --print-media-type was given.
func mediaFor(cmd *cli.Command, obj *settings.PdfObject) string {
	media := "screen"
	if cmd.Image.Web.PrintMediaType || obj.Load.PrintMediaType {
		media = "print"
	}
	switch obj.Load.MediaType {
	case settings.MediaPrint:
		media = "print"
	case settings.MediaScreen:
		media = "screen"
	case settings.MediaIgnore:
		media = ""
	}
	return media
}

// cropRect converts image settings into a pixel rectangle; returns the zero
// rectangle (no crop) when any value is unset (defaults are -1).
func cropRect(c settings.CropSettings) image.Rectangle {
	if c.Left < 0 || c.Top < 0 || c.Width <= 0 || c.Height <= 0 {
		return image.Rectangle{}
	}
	return image.Rect(c.Left, c.Top, c.Left+c.Width, c.Top+c.Height)
}

// resolveFormat picks the output format: the --format flag wins, otherwise
// the output file extension (.jpg/.jpeg), otherwise PNG.
func resolveFormat(flag, output string) (string, error) {
	f := strings.ToLower(strings.TrimSpace(flag))
	if f == "" {
		switch strings.ToLower(filepath.Ext(output)) {
		case ".jpg", ".jpeg":
			f = "jpg"
		default:
			f = "png"
		}
	}
	switch f {
	case "png":
		return "png", nil
	case "jpg", "jpeg":
		return "jpg", nil
	}
	return "", fmt.Errorf("unsupported format %q (supported: png, jpg)", flag)
}

// encode serializes img as PNG or JPEG. quality applies to JPEG only
// (1..100); PNG is lossless and ignores it.
func encode(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case "jpg":
		if quality < 1 {
			quality = 1
		}
		if quality > 100 {
			quality = 100
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
	return buf.Bytes(), nil
}

// onWhite composites img onto a white background (JPEG has no alpha).
func onWhite(img image.Image) image.Image {
	dst := image.NewNRGBA(img.Bounds())
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Over)
	return dst
}

// collectSheets gathers <style> blocks and <link rel="stylesheet"> resources
// from the DOM. A failed stylesheet only logs a warning; layout proceeds
// without it.
func collectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, lp settings.LoadPage, log io.Writer) []*css.Stylesheet {
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
				fmt.Fprintf(log, "warning: skipping <style>: %v\n", err)
			} else if sheet != nil {
				sheets = append(sheets, sheet)
			}
			return // raw-text element; no element children
		case "link":
			if linkStylesheet(n) {
				href := n.Attribute("href")
				r, err := loader.FetchSub(ctx, base, href, lp)
				if err != nil {
					fmt.Fprintf(log, "warning: skipping <link href=%q>: %v\n", href, err)
					return
				}
				sheet, err := css.Parse(string(r.Body))
				if err != nil {
					fmt.Fprintf(log, "warning: skipping <link href=%q>: %v\n", href, err)
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
// attribute matches the screen/image pipeline (empty, all, screen, or
// feature queries MediaMatches accepts for "screen"). Viewport uses a
// generous default so min-width feature links still load for typical widths.
func linkStylesheet(n *html.Node) bool {
	if n.Name != "link" || !strings.Contains(strings.ToLower(n.Attribute("rel")), "stylesheet") {
		return false
	}
	if n.Attribute("href") == "" {
		return false
	}
	media := n.Attribute("media")
	if media == "" {
		return true
	}
	// Default ~1024 CSS px wide viewport for image mode link filtering.
	const vw, vh = 768.0, 576.0 // 1024px×768px in pt
	return css.MediaMatches(media, "screen", vw, vh)
}
