package blinkengine

import (
	"fmt"

	blayout "github.com/chinmay-sawant/blinkless/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// opMapper copies blinkless display operations onto this module's operations.
// It is stateful for one display list: a face is parsed once per blinkless
// font pointer, and a blend group keeps one identity per blinkless group.
type opMapper struct {
	fonts  map[any]*pdf.Font
	groups map[any]*layout.BlendGroup
}

func newOpMapper() *opMapper {
	return &opMapper{
		fonts:  map[any]*pdf.Font{},
		groups: map[any]*layout.BlendGroup{},
	}
}

// ops maps source order to source order, dropping the kinds that paint
// nothing (deactivated operations and an unset kind).
func (m *opMapper) ops(src []blayout.DisplayOp) ([]layout.Op, error) {
	out := make([]layout.Op, 0, len(src))

	for idx := range src {
		mapped, keep, err := m.op(&src[idx])
		if err != nil {
			return nil, fmt.Errorf("op %d: %w", idx, err)
		}

		if keep {
			out = append(out, mapped)
		}
	}

	return out, nil
}

// op maps one operation. keep is false for the kinds that paint nothing.
func (m *opMapper) op(src *blayout.DisplayOp) (layout.Op, bool, error) {
	kind, keep := mapKind(src)
	if !keep {
		return layout.Op{}, false, nil //nolint:exhaustruct // dropped operation
	}

	dst := copyPlainOp(src, kind)

	if src.Font != nil {
		face, err := m.font(src.Font)
		if err != nil {
			return layout.Op{}, false, err //nolint:exhaustruct // failed operation
		}

		dst.Font = face
	}

	if kind == layout.OpGridRun && src.Grid != nil {
		dst.Grid = mapGridRun(src)
	}

	// A boundary operation carries the group and a begin/end mark and no
	// paint. Kind goes through unset (OpUnknown) exactly like a native mark.
	if mark := src.GroupBoundary(); mark != 0 {
		dst.SetGroupBoundary(m.group(src.Group()), src.IsGroupBegin())

		return dst, true, nil
	}

	m.payload(src, &dst)
	copyTextPayload(src, &dst)

	return dst, true, nil
}

// copyPlainOp copies the always-safe fields. Font, Grid, and XformSet are
// filled by the caller after this copy.
func copyPlainOp(src *blayout.DisplayOp, kind layout.OpKind) layout.Op {
	return layout.Op{ //nolint:exhaustruct // Font, Grid, and XformSet are set by the caller
		ID:                 src.ID,
		Kind:               kind,
		X:                  src.X,
		Y:                  src.Y,
		W:                  src.W,
		H:                  src.H,
		R:                  src.R,
		G:                  src.G,
		B:                  src.B,
		Alpha:              src.Alpha,
		Width:              src.Width,
		Size:               src.Size,
		LetterSpacing:      src.LetterSpacing,
		InkDescent:         src.InkDescent,
		Text:               src.Text,
		Radius:             src.Radius,
		RadiusTopLeft:      src.RadiusTopLeft,
		RadiusTopRight:     src.RadiusTopRight,
		RadiusBottomRight:  src.RadiusBottomRight,
		RadiusBottomLeft:   src.RadiusBottomLeft,
		RadiusY:            src.RadiusY,
		RadiusTopLeftY:     src.RadiusTopLeftY,
		RadiusTopRightY:    src.RadiusTopRightY,
		RadiusBottomRightY: src.RadiusBottomRightY,
		RadiusBottomLeftY:  src.RadiusBottomLeftY,
		StickyID:           src.StickyID,
		ZIndex:             src.ZIndex,
		RotateDeg:          src.RotateDeg,
		StrokeMask:         src.StrokeMask,
		LineInset:          src.LineInset,
		Bold:               src.Bold,
		FakeOblique:        src.FakeOblique,
		IsJPEG:             src.IsJPEG,
		IsBackground:       src.IsBackground,
		Fixed:              src.Fixed,
		Pinned:             src.Pinned,
		ZIndexSet:          src.ZIndexSet,
		Positioned:         src.Positioned,
	}
}

// payload copies the rare payload through the nil-safe accessors blinkless
// documents; reading the promoted fields directly would panic on an operation
// that never bound one.
func (m *opMapper) payload(src *blayout.DisplayOp, dst *layout.Op) {
	if uri := src.LinkURI(); uri != "" {
		dst.SetURI(uri)
	}

	if mode := src.BlendModeName(); mode != "" {
		dst.SetBlendMode(mode)
	}

	if data, width, height := src.ImageBytes(); len(data) > 0 {
		dst.SetImage(data, width, height, src.ImageAlt())
	}

	if src.XformSet {
		// Copy Blinkless's affine display transform; the PDF painter only
		// changes it to the PDF coordinate basis when writing the cm operator.
		mx := src.Transform()
		dst.SetXform(layout.Matrix2D{A: mx.A, B: mx.B, C: mx.C, D: mx.D, E: mx.E, F: mx.F})
	}

	if path, ok := src.PathValue(); ok {
		points := make([]layout.PathPoint, len(path.Points))
		for idx, point := range path.Points {
			// Path geometry is stored local to the op's canvas box so the
			// pagination passes that shift X/Y carry it (see PathGeometry).
			points[idx] = layout.PathPoint{X: point.X - src.X, Y: point.Y - src.Y}
		}

		dst.SetPath(layout.PathGeometry{Points: points, Closed: path.Closed})
	}

	if corner, ok := src.CornerPathValue(); ok {
		if points := cornerPathPoints(corner, src.X, src.Y); len(points) > 0 {
			dst.SetPath(layout.PathGeometry{Points: points, Closed: true})
		}
	}

	if basis, ok := src.GlyphBasisValue(); ok {
		dst.SetGlyphBasis(layout.Matrix2D{A: basis.A, B: basis.B, C: basis.C, D: basis.D})
	}

	if opacity := rawOpacity(src); opacity > 0 && opacity < 1 {
		dst.SetPaintOpacity(opacity)
	}

	if group := src.Group(); group != nil {
		dst.SetBlendGroup(m.group(group))
	}
}

// copyTextPayload copies the shaping and synthesis hints that text painting
// reads from an operation.
func copyTextPayload(src *blayout.DisplayOp, dst *layout.Op) {
	if transform := src.TextTransformValue(); transform != "" {
		dst.SetTextTransform(transform)
	}

	if language := src.TextLanguage(); language != "" {
		dst.SetTextLanguage(language)
	}

	if features := src.FontFeatures(); features != "" {
		dst.SetFontFeatures(features)
	}

	if autospace := src.TextAutospace(); autospace != "" {
		dst.SetTextAutospace(autospace)
	}

	if src.NoFakeBoldValue() {
		dst.SetNoFakeBold()
	}

	if src.Outline() {
		dst.SetOutline()
	}
}

// mapKind copies one blinkless kind onto this module's kind. An off-list kind
// stays inert rather than being mistaken for a fill, so a kind added later
// cannot change what an older consumer paints.
//
//nolint:cyclop // one arm per display-list kind, plus the inert default
func mapKind(src *blayout.DisplayOp) (layout.OpKind, bool) {
	switch src.Kind {
	case blayout.DisplayOpNoop:
		return 0, false
	case blayout.DisplayOpUnknown:
		// Layout emits the unset kind only as a blend-group boundary mark.
		return layout.OpUnknown, src.GroupBoundary() != 0
	case blayout.DisplayOpFillRect:
		return layout.OpFillRect, true
	case blayout.DisplayOpStrokeRect:
		return layout.OpStrokeRect, true
	case blayout.DisplayOpLine:
		return layout.OpLine, true
	case blayout.DisplayOpText:
		return layout.OpText, true
	case blayout.DisplayOpImage:
		return layout.OpImage, true
	case blayout.DisplayOpLinkURI:
		return layout.OpLinkURI, true
	case blayout.DisplayOpBullet:
		return layout.OpBullet, true
	case blayout.DisplayOpGridRun:
		return layout.OpGridRun, true
	case blayout.DisplayOpPath:
		return mapCornerPathKind(src)
	default:
		return 0, false
	}
}

// mapCornerPathKind maps a blinkless corner-shape path onto the fill or
// stroke kind whose painters read path geometry. The command list is checked
// here because a path with no commands paints nothing.
func mapCornerPathKind(src *blayout.DisplayOp) (layout.OpKind, bool) {
	corner, ok := src.CornerPathValue()
	if !ok || len(corner.Cmds) == 0 {
		return 0, false
	}

	if corner.Stroke {
		return layout.OpStrokeRect, true
	}

	return layout.OpFillRect, true
}

// cornerPathPoints flattens a blinkless corner-path command list into the
// straight-edged geometry this module's painters consume, rebased to the op's
// canvas box (originX, originY). The engine's corner painter emits moveTo,
// lineTo, and close; a curve verb contributes its end point so a future
// curve-bearing path still lands as a polygon instead of being dropped.
func cornerPathPoints(corner *blayout.CornerPath, originX, originY float64) []layout.PathPoint {
	points := make([]layout.PathPoint, 0, len(corner.Cmds))

	for idx := range corner.Cmds {
		command := &corner.Cmds[idx]

		switch command.Verb {
		case blayout.PathMoveTo, blayout.PathLineTo:
			points = append(points, layout.PathPoint{X: command.Pts[0].X - originX, Y: command.Pts[0].Y - originY})
		case blayout.PathQuadTo:
			points = append(points, layout.PathPoint{X: command.Pts[1].X - originX, Y: command.Pts[1].Y - originY})
		case blayout.PathCubicTo:
			points = append(points, layout.PathPoint{X: command.Pts[2].X - originX, Y: command.Pts[2].Y - originY})
		case blayout.PathClose:
			// Close has no point of its own; Closed carries it.
			continue
		}
	}

	return points
}

// mapGridRun copies one table row's collapsed border segments. The segment
// fields are the same on both sides of the bridge.
func mapGridRun(src *blayout.DisplayOp) *layout.GridRun {
	segs := make([]layout.GridSeg, len(src.Grid.Segs))

	for idx := range src.Grid.Segs {
		seg := &src.Grid.Segs[idx]
		segs[idx] = layout.GridSeg{
			X:         seg.X,
			Y:         seg.Y,
			W:         seg.W,
			H:         seg.H,
			Width:     seg.Width,
			R:         seg.R,
			G:         seg.G,
			B:         seg.B,
			LineInset: seg.LineInset,
		}
	}

	return &layout.GridRun{Segs: segs}
}

// rawOpacity reconstructs the raw element opacity (0 means "no override")
// from blinkless's folded Opacity() accessor. Opacity() multiplies Alpha in
// when Alpha is a real value, and Alpha is copied separately, so dividing it
// back out keeps the two from being applied twice.
func rawOpacity(src *blayout.DisplayOp) float64 {
	opacity := src.Opacity()
	if alpha := src.Alpha; alpha > 0 && alpha < 1 {
		return opacity / alpha
	}

	return opacity
}

// font resolves one blinkless face to this module's parsed font. Blinkless
// hands consumers the raw SFNT bytes (Font.Bytes) instead of a shared face
// type, so the first operation that uses a face parses it once and the face
// pointer keys the cache for the rest of the list. A face with no retained
// bytes draws with the engine default rather than dropping its text.
func (m *opMapper) font(src any) (*pdf.Font, error) {
	if face, ok := m.fonts[src]; ok {
		return face, nil
	}

	var data []byte
	if raw, ok := src.(interface{ Bytes() []byte }); ok {
		data = raw.Bytes()
	}

	if len(data) == 0 {
		face, err := pdf.DefaultFont()
		if err != nil {
			return nil, fmt.Errorf("default face: %w", err)
		}

		m.fonts[src] = face

		return face, nil
	}

	face, err := pdf.ParseFontBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse face bytes: %w", err)
	}

	m.fonts[src] = face

	return face, nil
}

// group maps one blend group, including its parent chain. Nesting keeps one
// identity per blinkless group so boundary marks and member operations point
// at the same copy.
func (m *opMapper) group(src *blayout.DisplayGroup) *layout.BlendGroup {
	if src == nil {
		return nil
	}

	if group, ok := m.groups[src]; ok {
		return group
	}

	group := &layout.BlendGroup{
		ID:      src.ID,
		Mode:    src.Mode,
		Isolate: src.Isolate,
		Parent:  m.group(src.Parent),
	}
	if src.Mask != nil {
		if raster := src.Mask.Raster; raster != nil {
			group.Mask = &layout.MaskDescriptor{
				PNGBytes:    append([]byte(nil), raster.ImageBytes...),
				PixelWidth:  raster.PixelWidth,
				PixelHeight: raster.PixelHeight,
			}
		}
	}
	m.groups[src] = group

	return group
}
