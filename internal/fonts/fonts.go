// Package fonts decodes offline @font-face payloads. Preparation fetches the
// bytes through the load seam, calls Decode, then hands SFNT bytes to
// internal/pdf. TTF, OTF, and WOFF1 pass through unchanged because internal/pdf
// already parses them; WOFF2 is reconstructed here with the Brotli module
// already in the build graph.
package fonts

const (
	// Caps mirror internal/pdf's WOFF1 caps: untrusted @font-face payloads
	// must not allocate unbounded memory.
	maxTables   = 1024
	maxTableLen = 16 << 20 // 16 MiB per table
	maxSFNTSize = 32 << 20 // 32 MiB reconstructed SFNT
)

// Decode returns SFNT bytes for a font payload. WOFF2 input is reconstructed
// into SFNT; every other container passes through for internal/pdf to parse.
func Decode(data []byte) ([]byte, error) {
	if len(data) >= 4 && string(data[0:4]) == woff2Signature {
		return DecodeWOFF2(data)
	}

	return data, nil
}
