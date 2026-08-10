package pdf

// Shared numeric constants for PDF content streams, TrueType/OpenType parsing,
// and image codecs. Named to satisfy mnd without scattering unexplained literals.
const (
	// Byte / codepoint bounds.
	maxLatin1Code = 0xFF
	maxBMPCode    = 0xFFFF
	maxUint8      = 0xFF
	maxUint16Val  = 0xFFFF

	// PDF / font metrics.
	pdfUnitsPerEm      = 1000
	fontWeightBoldMin  = 700
	defaultItalicAngle = -12
	fixed14Divisor     = 64 // TrueType F2DOT14 / fixed-point style scales often /64 for degrees-ish
	f2dot14Scale       = 16384.0

	// Rec.601 luma coefficients (grayscale conversion).
	lumaR = 0.299
	lumaG = 0.587
	lumaB = 0.114

	// TrueType / sfnt directory.
	sfntOffsetTableSize = 12
	sfntTableRecordSize = 16
	sfntSearchRangeMul  = 16
	sfntTableAlign      = 4
	sfntHeadCheckAdj    = 0xB1B0AFBA
	sfntNameRecordSize  = 12
	sfntNameHeaderSize  = 6

	// glyf / loca / hmtx.
	glyfHeaderSize            = 10
	bytesPerHMetric           = 4
	bytesPerLongHorMetricSide = 2
	uint16Bytes               = 2
	int16Bytes                = 2
	uint32Bytes               = 4

	// Simple-glyph point flag bits.
	glyfOnCurve          = 0x01
	glyfXShortVector     = 0x02
	glyfXSameOrPos       = 0x10
	glyfYShortVector     = 0x04
	glyfYSameOrPos       = 0x20
	glyfRepeatFlag       = 0x08
	glyfRepeatMask       = 0x80 // count of repeated flag bytes
	glyfArgWords         = 0x0001
	glyfArgsXYValues     = 0x0002
	glyfMoreComponents   = 0x0020
	glyfHaveScale        = 0x0008
	glyfHaveXYScale      = 0x0040
	glyfHaveTwoByTwo     = 0x0080
	glyfHaveInstructions = 0x0100

	// cmap formats / sizes.
	cmapFormat4          = 4
	cmapFormat6          = 6
	cmapFormat12         = 12
	cmapFormat0Size      = 262
	cmapFormat4Header    = 14
	cmapFormat6Header    = 10
	cmapFormat12Header   = 16
	cmapFormat4LenBase   = 16 // format-4 length field base (14-byte header + reservedPad)
	cmapFormat4SegStride = 8  // four uint16 arrays per segment
	cmapPlatformUnicode  = 0
	cmapPlatformMac      = 1
	cmapPlatformWin      = 3
	cmapWinUnicodeBMP    = 1

	// hhea / maxp / head minimum sizes.
	hheaMinSize = 36
	maxpMinSize = 6
	headMinSize = 52

	// Content stream / operators.
	pdfFloatPrec     = 3
	pdfNumBase       = 10
	rgbComponents    = 3
	pointComponents  = 2
	rectComponents   = 4
	curveComponents  = 6
	matrixComponents = 6
	numArgsMin3      = 3
	numArgsMin4      = 4
	numArgsMin5      = 5

	// JPEG markers / scan.
	jpegMarkerPrefix = 0xFF
	jpegTEM          = 0x01
	jpegSOI          = 0xD8
	jpegEOI          = 0xD9
	bitsPerByte      = 8
	rgbChannels      = 3

	// WOFF / OTTO.
	ottoFlavor = 0x4F54544F // 'OTTO'
	tagSize    = 4
	padMask3   = 3 // (n+3)&^3 four-byte pad

	// Composite glyph component scale sizes (bytes): F2DOT14 values.
	scaleBytes        = 2 // one F2DOT14 value
	xyScaleBytes      = 4 // two F2DOT14 values
	twoByTwoBytes     = 8 // four F2DOT14 values
	twoByTwoSecondOff = 6 // offset of the vertical F2DOT14 value

	// Arabic / shaping presentation form tiers.
	arabicFormsB  = 3
	arabicFormsA  = 2
	arabicTatweel = 0x0640

	// CID / Type0.
	cidBytesPerEntry  = 2
	cidGlyphHighShift = 8
	toUnicodeTwoByte  = 2

	// Font PDF encoding chunking.
	cidToGIDChunk = 100
	codeBytesTwo  = 2

	// Shape feature parsing.
	featureTagLen    = 4
	featureEndTagIdx = 5
	decimalBase      = 10

	// Bezier / midpoint.
	midpointDiv   = 2
	minCurveSteps = 2

	// Registry scan depth.
	fontScanMaxDepth = 2
)
