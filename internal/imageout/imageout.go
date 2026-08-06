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
	"gowkhtmltopdf/internal/line"
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
// then box-filters down to the final CSS-pixel size. Glyph bitmaps for this
// run live on a per-rasterize atlas (P5-05) so concurrent Renders do not share
// mutable cache state.
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
	atlas := newGlyphAtlas()
	for i := range res.Ops {
		paint(img, &res.Ops[i], pxPerPt, atlas)
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
//
// Paint semantics (fake-bold gate, stroke min-width) come from layout.StyleOf /
// layout.FakeBoldFor so PDF and raster stay on one table (P5-01). Fill alpha
// deliberately diverges: StyleOf pre-composites translucent fills against white
// (PDF paper); raster keeps raw op.R/G/B/Alpha and draw.Over onto NRGBA so a
// transparent canvas (Transparent) can show through. FakeBold + stroke width
// still follow the shared table.
//
// Page assembly: prologue already shares convert.CollectSheets +
// MergeFontFaces; multi-page PDF assembly remains convert-specific (P5-02).
func paint(img *image.NRGBA, op *layout.Op, pxPerPt float64, atlas *glyphAtlas) {
	ps := layout.StyleOf(op)
	switch op.Kind {
	case layout.OpFillRect:
		// Raw alpha for Over compositing — see paint comment (PDF vs raster).
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
		lw := strokeWidthScale(ps.StrokeWidth, pxPerPt)
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
		lw := strokeWidthScale(ps.StrokeWidth, pxPerPt)
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
		ttfDrawString(img, bx, by, op.Text, op.Size, face, c, pxPerPt, atlas)
		// Latin-only fake-bold (CJK gate lives in layout.FakeBoldFor).
		if layout.FakeBoldFor(op) {
			ttfDrawString(img, bx+float64(rasterSS), by, op.Text, op.Size, face, c, pxPerPt, atlas)
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

// strokeWidthScale returns the stroke thickness in canvas pixels from a
// StyleOf-resolved width (already min-clamped to 1 when non-positive).
func strokeWidthScale(strokeWidth float64, pxPerPt float64) int {
	w := strokeWidth
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

// Run is the CLI-facing adapter (P1-1): opens the output sink, builds a
// convert.Request, and delegates to RunRequest. Existing cmd/tests keep this
// signature; the engine no longer depends on *cli.Command internals beyond
// OpenOutput and output-path format sniffing.
func Run(ctx context.Context, cmd *cli.Command, log io.Writer) error {
	if cmd == nil {
		return errors.New("imageout: nil command")
	}
	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return err
	}
	img := cmd.Image
	// Resolve --format / extension before the engine so Request only needs
	// Image.Format (library callers set Format explicitly or get PNG).
	format, err := resolveFormat(img.Format, cmd.Output)
	if err != nil {
		closeOut()
		return err
	}
	img.Format = format
	req := &convert.Request{
		Global:  cmd.Global,
		Image:   &img,
		Objects: cmd.Objects,
		Output:  out,
	}
	runErr := RunRequest(ctx, req, log)
	if closeErr := closeOut(); closeErr != nil && runErr == nil {
		return closeErr
	}
	return runErr
}

// RunRequest drives image conversion from a CLI-independent convert.Request
// (P1-1). req.Image must be non-nil; req.Output receives encoded PNG/JPEG
// bytes (nil → os.Stdout is not auto-selected; callers must supply a writer).
func RunRequest(ctx context.Context, req *convert.Request, log io.Writer) error {
	if req == nil {
		return errors.New("imageout: nil request")
	}
	if req.Image == nil {
		return errors.New("imageout: nil Image settings")
	}
	if log == nil {
		log = io.Discard
	}
	// Policy A: one quiet bit — CLI --quiet sets Global.Quiet (not Image.Quiet).
	if req.Global.Quiet {
		log = io.Discard
	}
	obj, err := firstObject(req.Objects, log)
	if err != nil {
		return err
	}

	// P2-07: full load policy at construction. Image.Load holds image-mode
	// Proxy; CLI/library ACL (--allow / --enable-local-file-access) lives on
	// Global.Load. Merge so NewLoader applies everything; no post-construction
	// field pokes on Loader.
	loader := load.NewLoader(imageLoadGlobal(req.Global, *req.Image))
	loader.Log = log

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}
	var registry *pdf.Registry
	if dirs := req.Global.FontPaths; len(dirs) > 0 || req.Global.UseSystemFonts {
		scan := append([]string{}, dirs...)
		if req.Global.UseSystemFonts {
			scan = append(scan, pdf.DefaultSystemFontDirs()...)
		}
		registry = pdf.ScanFontDirs(scan)
		if log != io.Discard && len(scan) > 0 {
			line.Emit(log, line.Info, "scanned %d font path(s)", len(scan))
		}
	}

	res, err := loader.Load(ctx, obj.Page, obj.Load)
	if err != nil {
		return fmt.Errorf("load %q: %w", obj.Page, err)
	}
	if res.Skip {
		return fmt.Errorf("load %q: load-error policy is skip; nothing to render", obj.Page)
	}

	root, err := html.ParseDocument(res.Body)
	if err != nil {
		return fmt.Errorf("parse html: %w", err)
	}

	imgSet := req.Image
	media := mediaFor(req.Global, *imgSet, obj)
	// P2-01/P2-14: shared gatherer; default ~1024×768 CSS-px viewport in pt
	// (768×576) for <link media> feature queries. Media matches layout.
	sheets := convert.CollectSheets(ctx, loader, root, res.Base, obj.Load, convert.SheetOptions{
		ViewportW: 768.0,
		ViewportH: 576.0,
		MediaType: media,
		// ObjectIndex 0: image mode has one page; omit "object N:" prefix.
	}, log)
	enabled := convert.SimplifyDOMEnabled(imgSet.Web, obj.Web) || req.Global.Web.SimplifyDOM
	profile := convert.SimplifyDOMProfile(imgSet.Web, obj.Web)
	if profile == "" {
		profile = convert.SimplifyDOMProfile(req.Global.Web, settings.Web{})
	}
	sheets = convert.AppendSimplifySheet(sheets, enabled, profile)
	registry = convert.MergeFontFaces(ctx, loader, registry, sheets, res.Base, obj.Load, 1, log)

	cache := map[string][]byte{}
	imagesFn := func(src string) ([]byte, error) {
		if !imgSet.Web.Images {
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
		Width:              imgSet.Width,
		Height:             imgSet.Height,
		Font:               font,
		Registry:           registry,
		Sheets:             sheets,
		Media:              media,
		Images:             imagesFn,
		Background:         req.Global.Background,
		Transparent:        imgSet.Transparent,
		Crop:               cropRect(imgSet.Crop),
		SmartWidth:         imgSet.SmartWidth,
		PrintLinkUnderline: imgSet.Web.PrintLinkUnderline || req.Global.Web.PrintLinkUnderline || obj.Web.PrintLinkUnderline,
	})
	if err != nil {
		return err
	}

	format, err := resolveFormat(imgSet.Format, "")
	if err != nil {
		return err
	}
	if format == "jpg" && imgSet.Transparent {
		line.Emit(log, line.Warn, "--transparent ignored for JPEG output (white background used)")
		img = onWhite(img)
	}
	data, err := encode(img, format, imgSet.Quality)
	if err != nil {
		return fmt.Errorf("encode %s: %w", format, err)
	}

	out := req.Output
	if out == nil {
		return errors.New("imageout: nil Output writer")
	}
	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// imageLoadGlobal builds the LoadGlobal for image mode: Proxy from Image.Load
// plus ACL (Allow / EnableLocalFileAccess) from Global.Load, where CLI and
// ImageConverter.Global set them. NewLoader applies the full policy.
func imageLoadGlobal(global settings.PdfGlobal, image settings.ImageGlobal) settings.LoadGlobal {
	lg := image.Load
	// Global.Load is the ACL home (enablelocalfileaccess / allow keys).
	// Prefer Global when set; keep any Image.Load ACL already present (OR).
	if len(global.Load.Allow) > 0 {
		lg.Allow = append(append([]string(nil), lg.Allow...), global.Load.Allow...)
	}
	lg.EnableLocalFileAccess = lg.EnableLocalFileAccess || global.Load.EnableLocalFileAccess
	return lg
}

// imageLoadGlobalCmd is a test helper: ACL merge from a parsed Command.
func imageLoadGlobalCmd(cmd *cli.Command) settings.LoadGlobal {
	return imageLoadGlobal(cmd.Global, cmd.Image)
}

// firstObject returns the first page-like object. Image mode renders a
// single page; extra page objects and TOC objects are ignored with warnings.
// An empty Page is accepted when Load.InlineHTML is set (P2-04 in-memory
// source kind via ObjectSettings.SetBody).
func firstObject(objects []settings.PdfObject, log io.Writer) (*settings.PdfObject, error) {
	var first *settings.PdfObject
	for i := range objects {
		o := &objects[i]
		if o.IsTableOfContent {
			continue
		}
		if o.Page == "" && len(o.Load.InlineHTML) == 0 {
			continue
		}
		if first == nil {
			first = o
			continue
		}
		line.Emit(log, line.Warn, "image mode renders the first page only; ignoring object %d", i+1)
	}
	if first == nil {
		return nil, errors.New("no input to convert")
	}
	return first, nil
}

// mediaFor resolves the layout media via settings.ResolveMedia (P1-4).
// Image mode defaults to "screen". CLI --print-media-type lands on
// Global.Web; object overrides live on LoadPage and are mapped into a
// temporary Web view. Image.Web is merged so library Set paths still apply.
func mediaFor(global settings.PdfGlobal, image settings.ImageGlobal, obj *settings.PdfObject) string {
	web := image.Web
	if global.Web.PrintMediaType {
		web.PrintMediaType = true
	}
	if web.MediaType == settings.MediaIgnore {
		web.MediaType = global.Web.MediaType
	}
	var objWeb *settings.Web
	if obj != nil {
		w := settings.Web{
			PrintMediaType: obj.Load.PrintMediaType || obj.Web.PrintMediaType,
			MediaType:      obj.Load.MediaType,
		}
		if obj.Web.MediaType != settings.MediaIgnore {
			w.MediaType = obj.Web.MediaType
		}
		objWeb = &w
	}
	return settings.ResolveMedia("screen", web, objWeb)
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
