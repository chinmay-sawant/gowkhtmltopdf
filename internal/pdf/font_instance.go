package pdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	gtfont "github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
)

// Variation is one design-space axis setting, matching a CSS
// font-variation-settings `"tag" number` pair. Tag must be four ASCII
// letters or digits.
type Variation struct {
	Tag   string
	Value float32
}

var droppedVariationTables = map[string]bool{ //nolint:gochecknoglobals // immutable tag set
	"fvar": true,
	"gvar": true,
	"avar": true,
	"HVAR": true,
	"VVAR": true,
	"MVAR": true,
	"cvar": true,
}

// Instance returns a static face with wght/opsz/wdth (and any other requested
// axes) applied to glyf outlines and hmtx advances. The receiver is never
// mutated. Static faces and default-instance coordinates return the receiver.
func (f *Font) Instance(vars []Variation) *Font {
	if f == nil || len(vars) == 0 {
		return f
	}

	f.ensureParsed()

	if !f.HasVariationAxes() {
		return f
	}

	key := variationCacheKey(vars)

	f.instMu.Lock()
	defer f.instMu.Unlock()

	if inst, ok := f.instances[key]; ok {
		return inst
	}

	inst, err := instanceFont(f, vars)
	if err != nil || inst == nil {
		return f
	}

	if f.instances == nil {
		f.instances = map[string]*Font{}
	}

	f.instances[key] = inst

	return inst
}

func variationCacheKey(vars []Variation) string {
	parts := make([]string, len(vars))
	for i, v := range vars {
		parts[i] = v.Tag + "=" + strconv.FormatFloat(float64(v.Value), 'g', -1, 32)
	}

	sort.Strings(parts)

	return strings.Join(parts, ",")
}

func instanceFont(src *Font, vars []Variation) (*Font, error) {
	face, err := gtfont.ParseTTF(bytes.NewReader(src.data))
	if err != nil || face == nil {
		return nil, err
	}

	gtVars := make([]gtfont.Variation, 0, len(vars))
	for _, v := range vars {
		if len(v.Tag) != fontAxisTagLength {
			continue
		}

		gtVars = append(gtVars, gtfont.Variation{
			Tag:   ot.MustNewTag(v.Tag),
			Value: v.Value,
		})
	}

	if len(gtVars) == 0 {
		return src, nil
	}

	face.SetVariations(gtVars)
	if variationCoordsAreDefault(face) {
		return src, nil
	}

	outlines := make([][]byte, src.numGlyphs)
	advances := make([]int32, src.numGlyphs)
	lsbs := make([]int16, src.numGlyphs)

	for gid := range src.numGlyphs {
		glyph := gtfont.GID(gid) //nolint:gosec // gid < numGlyphs, TrueType ids are uint16
		advances[gid] = clampAdvance(face.HorizontalAdvance(glyph))
		lsbs[gid] = instancedLSB(face, glyph, src, gid)
		outlines[gid] = encodeInstancedOutline(face, glyph)
	}

	data, err := buildInstancedTTF(src, outlines, advances, lsbs)
	if err != nil {
		return nil, err
	}

	inst, err := ParseTTF(data)
	if err != nil {
		return nil, err
	}

	inst.PostScriptName = src.PostScriptName
	inst.isInstance = true

	return inst, nil
}

func variationCoordsAreDefault(face *gtfont.Face) bool {
	for _, coord := range face.Coords() {
		if coord != 0 {
			return false
		}
	}

	return true
}

func clampAdvance(v float32) int32 {
	rounded := math.Round(float64(v))
	if rounded < 0 {
		return 0
	}

	if rounded > float64(maxUint16Val) {
		return int32(maxUint16Val)
	}

	return int32(rounded)
}

func instancedLSB(face *gtfont.Face, glyph gtfont.GID, src *Font, gid int) int16 {
	if ext, ok := face.GlyphExtents(glyph); ok {
		return roundToInt16(ext.XBearing)
	}

	if gid < len(src.lsb) {
		return src.lsb[gid]
	}

	return 0
}

func encodeInstancedOutline(face *gtfont.Face, glyph gtfont.GID) []byte {
	data := face.GlyphData(glyph)
	outline, ok := data.(gtfont.GlyphOutline)
	if !ok || len(outline.Segments) == 0 {
		return nil
	}

	return encodeSimpleGlyf(outlineContours(outline))
}

type glyfPt struct {
	x, y int16
	on   bool
}

func outlineContours(outline gtfont.GlyphOutline) [][]glyfPt {
	contours := make([][]glyfPt, 0, 1)
	cur := make([]glyfPt, 0, 8)

	flush := func() {
		if len(cur) == 0 {
			return
		}

		contours = append(contours, cur)
		cur = make([]glyfPt, 0, 8)
	}

	for _, seg := range outline.Segments {
		switch seg.Op {
		case ot.SegmentOpMoveTo:
			flush()
			cur = append(cur, segmentPoint(seg.Args[0], true))
		case ot.SegmentOpLineTo:
			cur = append(cur, segmentPoint(seg.Args[0], true))
		case ot.SegmentOpQuadTo:
			cur = append(cur, segmentPoint(seg.Args[0], false), segmentPoint(seg.Args[1], true))
		case ot.SegmentOpCubeTo:
			cur = append(cur, segmentPoint(seg.Args[2], true))
		}
	}

	flush()

	return contours
}

func segmentPoint(p ot.SegmentPoint, on bool) glyfPt {
	return glyfPt{x: roundToInt16(p.X), y: roundToInt16(p.Y), on: on}
}

func roundToInt16(v float32) int16 {
	rounded := math.Round(float64(v))
	if rounded > math.MaxInt16 {
		return math.MaxInt16
	}

	if rounded < math.MinInt16 {
		return math.MinInt16
	}

	return int16(rounded)
}

func encodeSimpleGlyf(contours [][]glyfPt) []byte {
	pts, endPts, xMin, yMin, xMax, yMax, ok := collectGlyfPoints(contours)
	if !ok {
		return nil
	}

	flags, xs, ys := encodeGlyfCoords(pts)

	size := glyfHeaderSize + uint16Bytes*len(endPts) + uint16Bytes + len(flags) + len(xs) + len(ys)
	buf := make([]byte, 0, size)
	buf = appendInt16(buf, int16(len(endPts))) //nolint:gosec // contour count fits int16
	buf = appendInt16(buf, xMin)
	buf = appendInt16(buf, yMin)
	buf = appendInt16(buf, xMax)
	buf = appendInt16(buf, yMax)

	for _, end := range endPts {
		buf = binary.BigEndian.AppendUint16(buf, end)
	}

	buf = binary.BigEndian.AppendUint16(buf, 0) // instructionLength
	buf = append(buf, flags...)
	buf = append(buf, xs...)
	buf = append(buf, ys...)

	return buf
}

func collectGlyfPoints(contours [][]glyfPt) ([]glyfPt, []uint16, int16, int16, int16, int16, bool) {
	pts := make([]glyfPt, 0, 16)
	endPts := make([]uint16, 0, len(contours))

	var xMin, yMin, xMax, yMax int16

	first := true

	for _, contour := range contours {
		contour = stripClosingDuplicate(contour)
		if len(contour) == 0 {
			continue
		}

		for _, p := range contour {
			if first {
				xMin, yMin, xMax, yMax = p.x, p.y, p.x, p.y
				first = false
			} else {
				xMin, yMin, xMax, yMax = glyfBounds(xMin, yMin, xMax, yMax, p)
			}

			pts = append(pts, p)
		}

		endPts = append(endPts, uint16(len(pts)-1)) //nolint:gosec // glyf point count fits uint16
	}

	if len(pts) == 0 {
		return nil, nil, 0, 0, 0, 0, false
	}

	return pts, endPts, xMin, yMin, xMax, yMax, true
}

func stripClosingDuplicate(contour []glyfPt) []glyfPt {
	if len(contour) > 1 && contour[0] == contour[len(contour)-1] {
		return contour[:len(contour)-1]
	}

	return contour
}

func glyfBounds(xMin, yMin, xMax, yMax int16, p glyfPt) (int16, int16, int16, int16) {
	if p.x < xMin {
		xMin = p.x
	}

	if p.y < yMin {
		yMin = p.y
	}

	if p.x > xMax {
		xMax = p.x
	}

	if p.y > yMax {
		yMax = p.y
	}

	return xMin, yMin, xMax, yMax
}

func encodeGlyfCoords(pts []glyfPt) (flags, xs, ys []byte) {
	flags = make([]byte, 0, len(pts))
	xs = make([]byte, 0, len(pts)*2)
	ys = make([]byte, 0, len(pts)*2)

	var prevX, prevY int16

	for _, p := range pts {
		flag, xBytes, yBytes := encodeGlyfDelta(p, prevX, prevY)
		flags = append(flags, flag)
		xs = append(xs, xBytes...)
		ys = append(ys, yBytes...)
		prevX, prevY = p.x, p.y
	}

	return flags, xs, ys
}

func encodeGlyfDelta(p glyfPt, prevX, prevY int16) (byte, []byte, []byte) {
	var flag byte
	if p.on {
		flag = glyfOnCurve
	}

	xFlag, xBytes := encodeAxisDelta(int32(p.x)-int32(prevX), glyfXShortVector, glyfXSameOrPos)
	yFlag, yBytes := encodeAxisDelta(int32(p.y)-int32(prevY), glyfYShortVector, glyfYSameOrPos)

	return flag | xFlag | yFlag, xBytes, yBytes
}

func encodeAxisDelta(delta int32, shortFlag, sameOrPosFlag byte) (byte, []byte) {
	if delta == 0 {
		return sameOrPosFlag, nil
	}

	if delta > -256 && delta < 256 {
		if delta > 0 {
			return shortFlag | sameOrPosFlag, []byte{byte(delta)}
		}

		return shortFlag, []byte{byte(-delta)}
	}

	clamped := delta
	if clamped > math.MaxInt16 {
		clamped = math.MaxInt16
	}

	if clamped < math.MinInt16 {
		clamped = math.MinInt16
	}

	return 0, binary.BigEndian.AppendUint16(nil, uint16(int16(clamped))) //nolint:gosec // clamped to int16
}

func buildInstancedTTF(src *Font, outlines [][]byte, advances []int32, lsbs []int16) ([]byte, error) {
	loca := make([]uint32, len(outlines)+1)
	padded := padOutlines(outlines, loca)
	glyf := new(bytes.Buffer)

	for _, outline := range padded {
		glyf.Write(outline)
	}

	head := bytes.Clone(src.tables["head"])
	if len(head) < 54 {
		return nil, errFontMissingHead
	}

	binary.BigEndian.PutUint16(head[50:52], 1) // long loca

	hhea := bytes.Clone(src.tables["hhea"])
	if len(hhea) < 36 {
		return nil, errFontMissingHhea
	}

	binary.BigEndian.PutUint16(hhea[34:36], uint16(len(advances))) //nolint:gosec // numGlyphs fits uint16

	tables := make([]struct {
		tag  string
		data []byte
	}, 0, len(src.tables))

	for tag, data := range src.tables {
		if droppedVariationTables[tag] {
			continue
		}

		switch tag {
		case "glyf":
			data = glyf.Bytes()
		case "loca":
			data = encodeUint32Slice(loca)
		case "hmtx":
			data = encodeHmtx(advances, lsbs)
		case "head":
			data = head
		case "hhea":
			data = hhea
		default:
			data = bytes.Clone(data)
		}

		tables = append(tables, struct {
			tag  string
			data []byte
		}{tag: tag, data: data})
	}

	return buildFontFile(tables)
}

func encodeHmtx(advances []int32, lsbs []int16) []byte {
	out := make([]byte, len(advances)*bytesPerHMetric)

	for i, adv := range advances {
		binary.BigEndian.PutUint16(out[i*bytesPerHMetric:], uint16(adv)) //nolint:gosec // clamped earlier
		lsb := int16(0)
		if i < len(lsbs) {
			lsb = lsbs[i]
		}

		binary.BigEndian.PutUint16(out[i*bytesPerHMetric+uint16Bytes:], uint16(lsb)) //nolint:gosec // lsb is int16
	}

	return out
}

func appendInt16(buf []byte, v int16) []byte {
	return binary.BigEndian.AppendUint16(buf, uint16(v)) //nolint:gosec // two's complement bit pattern
}

const fontAxisTagLength = 4

// VariationAxis is one fvar axis on a variable face.
type VariationAxis struct {
	Tag     string
	Minimum float32
	Default float32
	Maximum float32
}

// VariationAxes returns the face's fvar axes. Empty for static faces.
func (f *Font) VariationAxes() []VariationAxis {
	if f == nil {
		return nil
	}

	f.ensureParsed()

	raw, ok := f.tables["fvar"]
	if !ok || len(raw) < 16 {
		return nil
	}

	axisOffset := int(binary.BigEndian.Uint16(raw[4:6]))
	axisCount := int(binary.BigEndian.Uint16(raw[8:10]))
	axisSize := int(binary.BigEndian.Uint16(raw[10:12]))
	if axisSize < 16 || axisCount <= 0 {
		return nil
	}

	out := make([]VariationAxis, 0, axisCount)

	for i := range axisCount {
		off := axisOffset + i*axisSize
		if off+16 > len(raw) {
			break
		}

		out = append(out, VariationAxis{
			Tag:     string(raw[off : off+4]),
			Minimum: fixed1616(raw[off+4 : off+8]),
			Default: fixed1616(raw[off+8 : off+12]),
			Maximum: fixed1616(raw[off+12 : off+16]),
		})
	}

	return out
}

func fixed1616(b []byte) float32 {
	v := int32(binary.BigEndian.Uint32(b)) //nolint:gosec // Fixed 16.16 bit pattern

	return float32(v) / 65536
}

func (a VariationAxis) String() string {
	return fmt.Sprintf("%s[%g,%g,%g]", a.Tag, a.Minimum, a.Default, a.Maximum)
}
