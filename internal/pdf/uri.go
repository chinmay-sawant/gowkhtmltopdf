package pdf

// URI action encoding.
//
// A /URI action value is a byte string whose bytes are expected to be ASCII
// (RFC 3986). PDF text encoders are the wrong tool for it: pdfString folds
// code points through WinAnsi/PDFDocEncoding, where byte 0x96 is U+0152 (OE),
// so an en dash in an href used to reach readers as "Œ" (31 annotations on a
// Wikipedia page). uriString instead percent-encodes the UTF-8 bytes of every
// code point above ASCII, matching what browsers write (%E2%80%93 for U+2013,
// %C3%B3 for ó).

const (
	uriLiteralDelims = 2    // '(' and ')' framing the literal
	uriASCIILimit    = 0x7F // last ASCII byte; bytes above are percent-encoded
)

// uriString renders uri as a PDF literal string for a /URI action. ASCII
// bytes pass through (escaped only where the literal string syntax needs it),
// and every byte above ASCII is percent-encoded from its UTF-8 form. Existing
// percent escapes are ASCII and pass through, so an href that is already
// percent-encoded is not double-encoded.
func uriString(uri string) string {
	out := make([]byte, 0, len(uri)+uriLiteralDelims)
	out = append(out, '(')

	for i := range len(uri) {
		octet := uri[i]
		if octet > uriASCIILimit {
			out = append(out, '%', hexUpperDigits[octet>>nibbleShift], hexUpperDigits[octet&nibbleMask])

			continue
		}

		out = appendPDFLiteralByte(out, octet)
	}

	return string(append(out, ')'))
}
