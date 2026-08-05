// Package svg provides a minimal SVG-as-image rasterizer for <img src="*.svg">.
// Scope: viewBox/width/height, rect, circle, ellipse, line, polyline, polygon,
// path (M/L/H/V/Z/C/c/Q/q subset), solid fill/stroke. No CSS-in-SVG, text, or
// external references.
package svg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"strconv"
	"strings"
)

// Rasterize decodes SVG XML into a PNG image. maxSide caps the longer edge
// in pixels (default 512).
func Rasterize(data []byte, maxSide int) (pngBytes []byte, w, h int, err error) {
	if maxSide <= 0 {
		maxSide = 512
	}
	if !looksLikeSVG(data) {
		return nil, 0, 0, fmt.Errorf("svg: not SVG")
	}
	root, err := parseSVG(data)
	if err != nil {
		return nil, 0, 0, err
	}
	vw, vh := root.vbW, root.vbH
	if vw <= 0 || vh <= 0 {
		vw, vh = root.width, root.height
	}
	if vw <= 0 {
		vw = 100
	}
	if vh <= 0 {
		vh = 100
	}
	scale := 1.0
	if vw > float64(maxSide) || vh > float64(maxSide) {
		scale = float64(maxSide) / math.Max(vw, vh)
	}
	pw := int(math.Ceil(vw * scale))
	ph := int(math.Ceil(vh * scale))
	if pw < 1 {
		pw = 1
	}
	if ph < 1 {
		ph = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, pw, ph))
	// Transparent background.
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	c := &canvas{img: img, sx: scale, sy: scale, ox: -root.vbX * scale, oy: -root.vbY * scale}
	for _, sh := range root.shapes {
		c.draw(sh)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), pw, ph, nil
}

func looksLikeSVG(data []byte) bool {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "\xef\xbb\xbf") {
		s = strings.TrimSpace(s[3:])
	}
	low := strings.ToLower(s)
	return strings.Contains(low, "<svg") || strings.HasPrefix(low, "<?xml")
}

type svgRoot struct {
	vbX, vbY, vbW, vbH float64
	width, height      float64
	shapes             []shape
}

type shape struct {
	kind                  string // rect, circle, ellipse, line, polyline, polygon, path
	x, y, w, h, rx, ry, r float64
	x1, y1, x2, y2        float64
	points                []float64
	d                     string
	fill                  color.RGBA
	stroke                color.RGBA
	strokeW               float64
	fillSet, strokeSet    bool
}

func parseSVG(data []byte) (*svgRoot, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	root := &svgRoot{}
	var fillDef color.RGBA
	var strokeDef color.RGBA
	fillDefSet, strokeDefSet := false, false
	strokeWDef := 1.0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(se.Name.Local)
		attrs := attrMap(se.Attr)
		switch name {
		case "svg":
			root.width = parseLen(attrs["width"], 0)
			root.height = parseLen(attrs["height"], 0)
			if vb := attrs["viewbox"]; vb != "" {
				parts := splitNums(vb)
				if len(parts) >= 4 {
					root.vbX, root.vbY, root.vbW, root.vbH = parts[0], parts[1], parts[2], parts[3]
				}
			}
			if root.vbW == 0 && root.width > 0 {
				root.vbW = root.width
			}
			if root.vbH == 0 && root.height > 0 {
				root.vbH = root.height
			}
			if c, ok := parsePaint(attrs["fill"]); ok {
				fillDef, fillDefSet = c, true
			}
			if c, ok := parsePaint(attrs["stroke"]); ok {
				strokeDef, strokeDefSet = c, true
			}
			if attrs["stroke-width"] != "" {
				strokeWDef = parseLen(attrs["stroke-width"], 1)
			}
		case "rect", "circle", "ellipse", "line", "polyline", "polygon", "path":
			sh := shape{kind: name, strokeW: strokeWDef, fill: fillDef, stroke: strokeDef, fillSet: fillDefSet, strokeSet: strokeDefSet}
			applyPaint(&sh, attrs)
			switch name {
			case "rect":
				sh.x = parseLen(attrs["x"], 0)
				sh.y = parseLen(attrs["y"], 0)
				sh.w = parseLen(attrs["width"], 0)
				sh.h = parseLen(attrs["height"], 0)
				sh.rx = parseLen(attrs["rx"], 0)
				sh.ry = parseLen(attrs["ry"], sh.rx)
			case "circle":
				sh.x = parseLen(attrs["cx"], 0)
				sh.y = parseLen(attrs["cy"], 0)
				sh.r = parseLen(attrs["r"], 0)
			case "ellipse":
				sh.x = parseLen(attrs["cx"], 0)
				sh.y = parseLen(attrs["cy"], 0)
				sh.rx = parseLen(attrs["rx"], 0)
				sh.ry = parseLen(attrs["ry"], 0)
			case "line":
				sh.x1 = parseLen(attrs["x1"], 0)
				sh.y1 = parseLen(attrs["y1"], 0)
				sh.x2 = parseLen(attrs["x2"], 0)
				sh.y2 = parseLen(attrs["y2"], 0)
			case "polyline", "polygon":
				sh.points = splitNums(attrs["points"])
			case "path":
				sh.d = attrs["d"]
			}
			root.shapes = append(root.shapes, sh)
			_ = dec.Skip()
		default:
			// skip unknown elements' content when empty; nested graphics ignored for v1
		}
	}
	return root, nil
}

func attrMap(attrs []xml.Attr) map[string]string {
	m := map[string]string{}
	for _, a := range attrs {
		m[strings.ToLower(a.Name.Local)] = a.Value
	}
	return m
}

func applyPaint(sh *shape, attrs map[string]string) {
	if c, ok := parsePaint(attrs["fill"]); ok {
		sh.fill, sh.fillSet = c, true
		if attrs["fill"] == "none" {
			sh.fillSet = false
		}
	}
	if c, ok := parsePaint(attrs["stroke"]); ok {
		sh.stroke, sh.strokeSet = c, true
		if attrs["stroke"] == "none" {
			sh.strokeSet = false
		}
	}
	if attrs["stroke-width"] != "" {
		sh.strokeW = parseLen(attrs["stroke-width"], 1)
	}
	if !sh.fillSet && !sh.strokeSet {
		// SVG default fill black
		sh.fill = color.RGBA{0, 0, 0, 255}
		sh.fillSet = true
	}
}

func parsePaint(s string) (color.RGBA, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return color.RGBA{}, false
	}
	if strings.EqualFold(s, "none") {
		return color.RGBA{}, true
	}
	r, g, b, a, ok := parseCSSColor(s)
	if !ok {
		return color.RGBA{}, false
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a * 255)}, true
}

func parseCSSColor(v string) (r, g, b int, alpha float64, ok bool) {
	// Minimal local copy to avoid layout→css→svg cycles: #rgb/#rrggbb and names.
	v = strings.TrimSpace(v)
	alpha = 1
	if strings.HasPrefix(v, "#") {
		hex := v[1:]
		switch len(hex) {
		case 3:
			return nibble(hex[0]) * 17, nibble(hex[1]) * 17, nibble(hex[2]) * 17, 1, true
		case 6:
			return hexByte(hex[0:2]), hexByte(hex[2:4]), hexByte(hex[4:6]), 1, true
		}
	}
	switch strings.ToLower(v) {
	case "black":
		return 0, 0, 0, 1, true
	case "white":
		return 255, 255, 255, 1, true
	case "red":
		return 255, 0, 0, 1, true
	case "green":
		return 0, 128, 0, 1, true
	case "blue":
		return 0, 0, 255, 1, true
	}
	return 0, 0, 0, 0, false
}

func nibble(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

func hexByte(s string) int { return nibble(s[0])*16 + nibble(s[1]) }

func parseLen(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return f
}

func splitNums(s string) []float64 {
	s = strings.ReplaceAll(s, ",", " ")
	var out []float64
	for _, p := range strings.Fields(s) {
		f, err := strconv.ParseFloat(p, 64)
		if err == nil {
			out = append(out, f)
		}
	}
	return out
}

type canvas struct {
	img    *image.RGBA
	sx, sy float64
	ox, oy float64
}

func (c *canvas) tx(x, y float64) (int, int) {
	return int(math.Round(x*c.sx + c.ox)), int(math.Round(y*c.sy + c.oy))
}

func (c *canvas) draw(sh shape) {
	switch sh.kind {
	case "rect":
		x0, y0 := c.tx(sh.x, sh.y)
		x1, y1 := c.tx(sh.x+sh.w, sh.y+sh.h)
		if sh.fillSet {
			fillRect(c.img, x0, y0, x1, y1, sh.fill)
		}
		if sh.strokeSet {
			strokeRect(c.img, x0, y0, x1, y1, sh.stroke, int(math.Max(1, sh.strokeW*c.sx)))
		}
	case "circle":
		cx, cy := c.tx(sh.x, sh.y)
		r := int(math.Round(sh.r * c.sx))
		if sh.fillSet {
			fillCircle(c.img, cx, cy, r, sh.fill)
		}
		if sh.strokeSet {
			strokeCircle(c.img, cx, cy, r, sh.stroke, int(math.Max(1, sh.strokeW*c.sx)))
		}
	case "ellipse":
		cx, cy := c.tx(sh.x, sh.y)
		rx := int(math.Round(sh.rx * c.sx))
		ry := int(math.Round(sh.ry * c.sy))
		if sh.fillSet {
			fillEllipse(c.img, cx, cy, rx, ry, sh.fill)
		}
	case "line":
		if sh.strokeSet {
			x1, y1 := c.tx(sh.x1, sh.y1)
			x2, y2 := c.tx(sh.x2, sh.y2)
			strokeLine(c.img, x1, y1, x2, y2, sh.stroke, int(math.Max(1, sh.strokeW*c.sx)))
		}
	case "polyline", "polygon":
		pts := make([][2]int, 0, len(sh.points)/2)
		for i := 0; i+1 < len(sh.points); i += 2 {
			x, y := c.tx(sh.points[i], sh.points[i+1])
			pts = append(pts, [2]int{x, y})
		}
		if sh.kind == "polygon" && sh.fillSet && len(pts) >= 3 {
			fillPolygon(c.img, pts, sh.fill)
		}
		if sh.strokeSet {
			for i := 0; i+1 < len(pts); i++ {
				strokeLine(c.img, pts[i][0], pts[i][1], pts[i+1][0], pts[i+1][1], sh.stroke, int(math.Max(1, sh.strokeW*c.sx)))
			}
			if sh.kind == "polygon" && len(pts) >= 2 {
				strokeLine(c.img, pts[len(pts)-1][0], pts[len(pts)-1][1], pts[0][0], pts[0][1], sh.stroke, int(math.Max(1, sh.strokeW*c.sx)))
			}
		}
	case "path":
		pts := pathToPolyline(sh.d)
		if len(pts) >= 2 {
			ip := make([][2]int, len(pts))
			for i, p := range pts {
				ip[i][0], ip[i][1] = c.tx(p[0], p[1])
			}
			if sh.fillSet && len(ip) >= 3 {
				fillPolygon(c.img, ip, sh.fill)
			}
			if sh.strokeSet {
				w := int(math.Max(1, sh.strokeW*c.sx))
				for i := 0; i+1 < len(ip); i++ {
					strokeLine(c.img, ip[i][0], ip[i][1], ip[i+1][0], ip[i+1][1], sh.stroke, w)
				}
			}
		}
	}
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			set(img, x, y, col)
		}
	}
}

func strokeRect(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA, w int) {
	strokeLine(img, x0, y0, x1, y0, col, w)
	strokeLine(img, x1, y0, x1, y1, col, w)
	strokeLine(img, x1, y1, x0, y1, col, w)
	strokeLine(img, x0, y1, x0, y0, col, w)
}

func fillCircle(img *image.RGBA, cx, cy, r int, col color.RGBA) {
	fillEllipse(img, cx, cy, r, r, col)
}

func strokeCircle(img *image.RGBA, cx, cy, r int, col color.RGBA, w int) {
	for a := 0; a < 360; a++ {
		rad := float64(a) * math.Pi / 180
		x := cx + int(math.Round(float64(r)*math.Cos(rad)))
		y := cy + int(math.Round(float64(r)*math.Sin(rad)))
		for dy := -w / 2; dy <= w/2; dy++ {
			for dx := -w / 2; dx <= w/2; dx++ {
				set(img, x+dx, y+dy, col)
			}
		}
	}
}

func fillEllipse(img *image.RGBA, cx, cy, rx, ry int, col color.RGBA) {
	if rx <= 0 || ry <= 0 {
		return
	}
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if dx*dx+dy*dy <= 1 {
				set(img, x, y, col)
			}
		}
	}
}

func strokeLine(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA, w int) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	for {
		for oy := -w / 2; oy <= w/2; oy++ {
			for ox := -w / 2; ox <= w/2; ox++ {
				set(img, x0+ox, y0+oy, col)
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func fillPolygon(img *image.RGBA, pts [][2]int, col color.RGBA) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0][1], pts[0][1]
	for _, p := range pts {
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		j := len(pts) - 1
		for i := 0; i < len(pts); i++ {
			yi, yj := pts[i][1], pts[j][1]
			xi, xj := pts[i][0], pts[j][0]
			if (yi > y) != (yj > y) {
				x := xi + (y-yi)*(xj-xi)/(yj-yi)
				xs = append(xs, x)
			}
			j = i
		}
		for i := 0; i+1 < len(xs); i += 2 {
			a, b := xs[i], xs[i+1]
			if a > b {
				a, b = b, a
			}
			for x := a; x <= b; x++ {
				set(img, x, y, col)
			}
		}
	}
}

func set(img *image.RGBA, x, y int, col color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	i := img.PixOffset(x, y)
	// Source-over
	sr := float64(col.R) / 255
	sg := float64(col.G) / 255
	sb := float64(col.B) / 255
	sa := float64(col.A) / 255
	dr := float64(img.Pix[i]) / 255
	dg := float64(img.Pix[i+1]) / 255
	db := float64(img.Pix[i+2]) / 255
	da := float64(img.Pix[i+3]) / 255
	outA := sa + da*(1-sa)
	if outA <= 0 {
		return
	}
	img.Pix[i] = uint8(((sr*sa + dr*da*(1-sa)) / outA) * 255)
	img.Pix[i+1] = uint8(((sg*sa + dg*da*(1-sa)) / outA) * 255)
	img.Pix[i+2] = uint8(((sb*sa + db*da*(1-sa)) / outA) * 255)
	img.Pix[i+3] = uint8(outA * 255)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// pathToPolyline approximates a path into line segments (M/L/H/V/Z/C/c/Q/q).
func pathToPolyline(d string) [][2]float64 {
	toks := tokenizePath(d)
	var pts [][2]float64
	var cx, cy, sx, sy float64
	i := 0
	cmd := byte(0)
	for i < len(toks) {
		t := toks[i]
		if len(t) == 1 && strings.ContainsAny(t, "MmLlHhVvZzCcQq") {
			cmd = t[0]
			i++
			if cmd == 'Z' || cmd == 'z' {
				cx, cy = sx, sy
				pts = append(pts, [2]float64{cx, cy})
				continue
			}
		}
		switch cmd {
		case 'M', 'm':
			if i+1 >= len(toks) {
				return pts
			}
			x, y := num(toks[i]), num(toks[i+1])
			i += 2
			if cmd == 'm' && len(pts) > 0 {
				x += cx
				y += cy
			}
			cx, cy, sx, sy = x, y, x, y
			pts = append(pts, [2]float64{x, y})
			if cmd == 'M' {
				cmd = 'L'
			} else {
				cmd = 'l'
			}
		case 'L', 'l':
			if i+1 >= len(toks) {
				return pts
			}
			x, y := num(toks[i]), num(toks[i+1])
			i += 2
			if cmd == 'l' {
				x += cx
				y += cy
			}
			cx, cy = x, y
			pts = append(pts, [2]float64{x, y})
		case 'H', 'h':
			if i >= len(toks) {
				return pts
			}
			x := num(toks[i])
			i++
			if cmd == 'h' {
				x += cx
			}
			cx = x
			pts = append(pts, [2]float64{cx, cy})
		case 'V', 'v':
			if i >= len(toks) {
				return pts
			}
			y := num(toks[i])
			i++
			if cmd == 'v' {
				y += cy
			}
			cy = y
			pts = append(pts, [2]float64{cx, cy})
		case 'C', 'c':
			if i+5 >= len(toks) {
				return pts
			}
			x1, y1 := num(toks[i]), num(toks[i+1])
			x2, y2 := num(toks[i+2]), num(toks[i+3])
			x, y := num(toks[i+4]), num(toks[i+5])
			i += 6
			if cmd == 'c' {
				x1 += cx
				y1 += cy
				x2 += cx
				y2 += cy
				x += cx
				y += cy
			}
			for step := 1; step <= 8; step++ {
				tt := float64(step) / 8
				bx, by := cubic(cx, cy, x1, y1, x2, y2, x, y, tt)
				pts = append(pts, [2]float64{bx, by})
			}
			cx, cy = x, y
		case 'Q', 'q':
			if i+3 >= len(toks) {
				return pts
			}
			x1, y1 := num(toks[i]), num(toks[i+1])
			x, y := num(toks[i+2]), num(toks[i+3])
			i += 4
			if cmd == 'q' {
				x1 += cx
				y1 += cy
				x += cx
				y += cy
			}
			for step := 1; step <= 8; step++ {
				tt := float64(step) / 8
				bx := (1-tt)*(1-tt)*cx + 2*(1-tt)*tt*x1 + tt*tt*x
				by := (1-tt)*(1-tt)*cy + 2*(1-tt)*tt*y1 + tt*tt*y
				pts = append(pts, [2]float64{bx, by})
			}
			cx, cy = x, y
		default:
			i++
		}
	}
	return pts
}

func cubic(x0, y0, x1, y1, x2, y2, x3, y3, t float64) (float64, float64) {
	u := 1 - t
	x := u*u*u*x0 + 3*u*u*t*x1 + 3*u*t*t*x2 + t*t*t*x3
	y := u*u*u*y0 + 3*u*u*t*y1 + 3*u*t*t*y2 + t*t*t*y3
	return x, y
}

func tokenizePath(d string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(d); i++ {
		c := d[i]
		if c == ',' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			flush()
			continue
		}
		if (c >= 'A' && c <= 'Z' && c != 'E' && c != 'e') || (c >= 'a' && c <= 'z' && c != 'e') {
			flush()
			out = append(out, string(c))
			continue
		}
		cur.WriteByte(c)
	}
	flush()
	return out
}

func num(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
