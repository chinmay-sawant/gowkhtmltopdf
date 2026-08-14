//nolint:mnd,funlen // binary layout constants for ICC.1:2001-04 sRGB profile
package pdf

import (
	"encoding/binary"
)

const sRGBICCProfileSize = 472

// sRGBICCProfile returns a minimal valid standard sRGB v2.1 ICC matrix/TRC profile.
// It is 472 bytes uncompressed and adheres strictly to ICC.1:2001-04 (v2.1.0).
//
//nolint:mnd // ICC profile binary header constants and tag table layout
func sRGBICCProfile() []byte {
	buf := make([]byte, sRGBICCProfileSize)

	// 1. Header (128 bytes)
	binary.BigEndian.PutUint32(buf[0:4], sRGBICCProfileSize) // Profile size
	copy(buf[4:8], "none")                                   // Preferred CMM type
	buf[8] = 2                                               // Major version 2
	buf[9] = 0x10                                            // Minor version 1.0 (v2.1.0)
	copy(buf[12:16], "mntr")                                 // Display device profile
	copy(buf[16:20], "RGB ")                                 // Color space of data
	copy(buf[20:24], "XYZ ")                                 // PCS (Profile Connection Space)

	// Creation date: 2026-08-14 00:00:00 UTC
	binary.BigEndian.PutUint16(buf[24:26], 2026)
	binary.BigEndian.PutUint16(buf[26:28], 8)
	binary.BigEndian.PutUint16(buf[28:30], 14)
	binary.BigEndian.PutUint16(buf[30:32], 0)
	binary.BigEndian.PutUint16(buf[32:34], 0)
	binary.BigEndian.PutUint16(buf[34:36], 0)

	copy(buf[36:40], "acsp") // Profile file signature

	// Illuminant: D50 (X=0.9642, Y=1.0, Z=0.8249 in s15Fixed16)
	binary.BigEndian.PutUint32(buf[68:72], 0x0000F6D6)
	binary.BigEndian.PutUint32(buf[72:76], 0x00010000)
	binary.BigEndian.PutUint32(buf[76:80], 0x0000D32D)

	// 2. Tag Table
	binary.BigEndian.PutUint32(buf[128:132], 9) // Tag count: 9 tags

	type tagEntry struct {
		tag    string
		offset uint32
		size   uint32
	}

	tags := []tagEntry{
		{"desc", 240, 108},
		{"cprt", 348, 28},
		{"wtpt", 376, 20},
		{"rXYZ", 396, 20},
		{"gXYZ", 416, 20},
		{"bXYZ", 436, 20},
		{"rTRC", 456, 16},
		{"gTRC", 456, 16},
		{"bTRC", 456, 16},
	}

	pos := 132
	for _, t := range tags {
		copy(buf[pos:pos+4], t.tag)
		binary.BigEndian.PutUint32(buf[pos+4:pos+8], t.offset)
		binary.BigEndian.PutUint32(buf[pos+8:pos+12], t.size)
		pos += 12
	}

	// 3. Tag Data

	// 3.1 'desc' tag at offset 240 (108 bytes)
	// Type 'desc'
	copy(buf[240:244], "desc")
	// ASCII length = 18 ("sRGB IEC61966-2.1\0")
	binary.BigEndian.PutUint32(buf[248:252], 18)
	copy(buf[252:270], "sRGB IEC61966-2.1\x00")
	// Remaining bytes of desc tag (Unicode & ScriptCode) are 0

	// 3.2 'cprt' tag at offset 348 (28 bytes)
	// Type 'text'
	copy(buf[348:352], "text")
	copy(buf[356:376], "Copyright (c) 2026\x00\x00")

	// 3.3 'wtpt' tag at offset 376 (20 bytes)
	// Type 'XYZ '
	copy(buf[376:380], "XYZ ")
	binary.BigEndian.PutUint32(buf[384:388], 0x0000F6D6) // D50 X
	binary.BigEndian.PutUint32(buf[388:392], 0x00010000) // D50 Y
	binary.BigEndian.PutUint32(buf[392:396], 0x0000D32D) // D50 Z

	// 3.4 'rXYZ' tag at offset 396 (20 bytes)
	copy(buf[396:400], "XYZ ")
	binary.BigEndian.PutUint32(buf[404:408], 0x00006FA2)
	binary.BigEndian.PutUint32(buf[408:412], 0x000038F5)
	binary.BigEndian.PutUint32(buf[412:416], 0x00000390)

	// 3.5 'gXYZ' tag at offset 416 (20 bytes)
	copy(buf[416:420], "XYZ ")
	binary.BigEndian.PutUint32(buf[424:428], 0x00006299)
	binary.BigEndian.PutUint32(buf[428:432], 0x0000B785)
	binary.BigEndian.PutUint32(buf[432:436], 0x000018DA)

	// 3.6 'bXYZ' tag at offset 436 (20 bytes)
	copy(buf[436:440], "XYZ ")
	binary.BigEndian.PutUint32(buf[444:448], 0x000024A0)
	binary.BigEndian.PutUint32(buf[448:452], 0x00000F84)
	binary.BigEndian.PutUint32(buf[452:456], 0x0000B6CF)

	// 3.7 'rTRC' / 'gTRC' / 'bTRC' shared tag at offset 456 (16 bytes)
	// Type 'curv'
	copy(buf[456:460], "curv")
	binary.BigEndian.PutUint32(buf[464:468], 1)      // 1 entry
	binary.BigEndian.PutUint16(buf[468:470], 0x0233) // Gamma 2.2 in u8Fixed8
	// 470..472 is 0x0000 padding

	return buf
}
