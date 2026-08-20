package pdf

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tdewolff/font"
)

var (
	errWOFF2Invalid    = errors.New("woff2: invalid or unsupported WOFF2 data")
	errWOFF2Collection = errors.New("woff2: font collections are unsupported")
)

// DecodeWOFF2 decompresses a WOFF2 file into SFNT (TrueType/OpenType) bytes.
// Decoding uses tdewolff/font.ParseWOFF2 (MIT; already in the canvas module
// graph), which applies Brotli and reconstructs transformed glyf/loca/hmtx.
// Collections and unsupported transforms are rejected by the decoder.
//
// Callers must still run ParseTTF on the result (CFF/OTTO and fvar gates).
func DecodeWOFF2(data []byte) ([]byte, error) {
	if len(data) < 4 || string(data[0:4]) != woff2Signature {
		return nil, errWOFFBadSignature
	}

	sfnt, err := font.ParseWOFF2(data)
	if err != nil {
		return nil, mapWOFF2Err(err)
	}

	if uint64(len(sfnt)) > woffMaxSFNTSize {
		return nil, errWOFFSFNTTooLarge
	}

	return sfnt, nil
}

func mapWOFF2Err(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "collection"):
		return fmt.Errorf("%w", errWOFF2Collection)
	case strings.Contains(msg, "bad signature"):
		return fmt.Errorf("%w", errWOFFBadSignature)
	default:
		return fmt.Errorf("%w", errWOFF2Invalid)
	}
}
