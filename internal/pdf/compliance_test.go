//nolint:testpackage,exhaustruct,cyclop,funlen,varnamelen,wsl,gosec,lll,dupl // compliance tests
package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
	"time"
)

func TestCanonicalProfileAndHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input      string
		wantCanon  string
		wantPDFA3  bool
		wantPDFUA1 bool
		wantPDFA4  bool
		wantPDFUA2 bool
		wantCompl  bool
	}{
		{input: "", wantCanon: "", wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: false},
		{input: "   ", wantCanon: "", wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: false},
		// PDF/A-3a aliases
		{input: "PDF/A-3a", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/a-3a", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "a3a", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdfa-3a", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf-a-3a", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "a-3a", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/a-3", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "a3", wantCanon: ProfilePDFA3a, wantPDFA3: true, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		// PDF/UA-1 aliases
		{input: "PDF/UA-1", wantCanon: ProfilePDFUA1, wantPDFA3: false, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/ua-1", wantCanon: ProfilePDFUA1, wantPDFA3: false, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "ua1", wantCanon: ProfilePDFUA1, wantPDFA3: false, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdfua-1", wantCanon: ProfilePDFUA1, wantPDFA3: false, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "ua-1", wantCanon: ProfilePDFUA1, wantPDFA3: false, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/ua", wantCanon: ProfilePDFUA1, wantPDFA3: false, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		// 1.7 Dual profile aliases
		{input: "PDF/A-3a+PDF/UA-1", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/a-3a+pdf/ua-1", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "a3a-ua1", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "a3a+ua1", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/a-3a-pdf/ua-1", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdfa-3a+pdfua-1", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/ua-1+pdf/a-3a", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "ua1+a3a", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		{input: "ua1-a3a", wantCanon: ProfilePDFA3aPDFUA1, wantPDFA3: true, wantPDFUA1: true, wantPDFA4: false, wantPDFUA2: false, wantCompl: true},
		// PDF/A-4 aliases
		{input: "PDF/A-4", wantCanon: ProfilePDFA4, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: false, wantCompl: true},
		{input: "pdf/a-4", wantCanon: ProfilePDFA4, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: false, wantCompl: true},
		{input: "a4", wantCanon: ProfilePDFA4, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: false, wantCompl: true},
		{input: "pdfa-4", wantCanon: ProfilePDFA4, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: false, wantCompl: true},
		{input: "pdf-a-4", wantCanon: ProfilePDFA4, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: false, wantCompl: true},
		{input: "a-4", wantCanon: ProfilePDFA4, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: false, wantCompl: true},
		// PDF/UA-2 aliases
		{input: "PDF/UA-2", wantCanon: ProfilePDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: true, wantCompl: true},
		{input: "pdf/ua-2", wantCanon: ProfilePDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: true, wantCompl: true},
		{input: "ua2", wantCanon: ProfilePDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: true, wantCompl: true},
		{input: "pdfua-2", wantCanon: ProfilePDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: true, wantCompl: true},
		{input: "ua-2", wantCanon: ProfilePDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: true, wantCompl: true},
		// 2.0 Dual profile aliases
		{input: "PDF/A-4+PDF/UA-2", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "pdf/a-4+pdf/ua-2", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "a4-ua2", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "a4+ua2", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "pdf/a-4-pdf/ua-2", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "pdfa-4+pdfua-2", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "pdf/ua-2+pdf/a-4", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "ua2+a4", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		{input: "ua2-a4", wantCanon: ProfilePDFA4PDFUA2, wantPDFA3: false, wantPDFUA1: false, wantPDFA4: true, wantPDFUA2: true, wantCompl: true},
		// Unsupported or unknown
		{input: "PDF/A-1b", wantCanon: "", wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: false},
		{input: "custom-profile", wantCanon: "", wantPDFA3: false, wantPDFUA1: false, wantPDFA4: false, wantPDFUA2: false, wantCompl: false},
	}

	for _, tc := range tests {
		p := WriterPolicy{ConformanceProfile: tc.input}
		if got := p.CanonicalProfile(); got != tc.wantCanon {
			t.Errorf("CanonicalProfile(%q) = %q, want %q", tc.input, got, tc.wantCanon)
		}
		if got := p.IsPDFA3(); got != tc.wantPDFA3 {
			t.Errorf("IsPDFA3(%q) = %v, want %v", tc.input, got, tc.wantPDFA3)
		}
		if got := p.IsPDFUA1(); got != tc.wantPDFUA1 {
			t.Errorf("IsPDFUA1(%q) = %v, want %v", tc.input, got, tc.wantPDFUA1)
		}
		if got := p.IsPDFA4(); got != tc.wantPDFA4 {
			t.Errorf("IsPDFA4(%q) = %v, want %v", tc.input, got, tc.wantPDFA4)
		}
		if got := p.IsPDFUA2(); got != tc.wantPDFUA2 {
			t.Errorf("IsPDFUA2(%q) = %v, want %v", tc.input, got, tc.wantPDFUA2)
		}
		if got := p.IsCompliant(); got != tc.wantCompl {
			t.Errorf("IsCompliant(%q) = %v, want %v", tc.input, got, tc.wantCompl)
		}
	}
}

func TestProfileValidationMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  WriterPolicy
		wantErr error
	}{
		// Valid cases
		{
			name:    "PDF17 with empty profile is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ""},
			wantErr: nil,
		},
		{
			name:    "PDF14 with empty profile is valid",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ""},
			wantErr: nil,
		},
		{
			name:    "PDF20 with empty profile is valid",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: ""},
			wantErr: nil,
		},
		{
			name:    "PDF17 with PDF/A-3a is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFA3a},
			wantErr: nil,
		},
		{
			name:    "PDF17 with a3a alias is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "a3a"},
			wantErr: nil,
		},
		{
			name:    "PDF17 with PDF/UA-1 is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFUA1},
			wantErr: nil,
		},
		{
			name:    "PDF17 with ua1 alias is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "ua1"},
			wantErr: nil,
		},
		{
			name:    "PDF17 with Dual profile is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFA3aPDFUA1},
			wantErr: nil,
		},
		{
			name:    "PDF17 with a3a-ua1 alias is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "a3a-ua1"},
			wantErr: nil,
		},
		// PDF 2.0 valid profiles
		{
			name:    "PDF20 with PDF/A-4 is valid",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: ProfilePDFA4},
			wantErr: nil,
		},
		{
			name:    "PDF20 with a4 alias is valid",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: "a4"},
			wantErr: nil,
		},
		{
			name:    "PDF20 with PDF/UA-2 is valid",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: ProfilePDFUA2},
			wantErr: nil,
		},
		{
			name:    "PDF20 with ua2 alias is valid",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: "ua2"},
			wantErr: nil,
		},
		{
			name:    "PDF20 with Dual A4+UA2 is valid",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: ProfilePDFA4PDFUA2},
			wantErr: nil,
		},
		{
			name:    "PDF20 with a4-ua2 alias is valid",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: "a4-ua2"},
			wantErr: nil,
		},
		// Version mismatches (require PDF 1.7)
		{
			name:    "PDF14 with PDF/A-3a returns ErrConformanceRequiresPDF17",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ProfilePDFA3a},
			wantErr: ErrConformanceRequiresPDF17,
		},
		{
			name:    "PDF14 with PDF/UA-1 returns ErrConformanceRequiresPDF17",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ProfilePDFUA1},
			wantErr: ErrConformanceRequiresPDF17,
		},
		{
			name:    "PDF14 with dual profile returns ErrConformanceRequiresPDF17",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ProfilePDFA3aPDFUA1},
			wantErr: ErrConformanceRequiresPDF17,
		},
		{
			name:    "PDF20 with PDF/A-3a returns ErrConformanceRequiresPDF17",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: ProfilePDFA3a},
			wantErr: ErrConformanceRequiresPDF17,
		},
		// Version mismatches (require PDF 2.0)
		{
			name:    "PDF14 with PDF/A-4 returns ErrConformanceRequiresPDF20",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ProfilePDFA4},
			wantErr: ErrConformanceRequiresPDF20,
		},
		{
			name:    "PDF14 with PDF/UA-2 returns ErrConformanceRequiresPDF20",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ProfilePDFUA2},
			wantErr: ErrConformanceRequiresPDF20,
		},
		{
			name:    "PDF17 with PDF/A-4 returns ErrConformanceRequiresPDF20",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-4"},
			wantErr: ErrConformanceRequiresPDF20,
		},
		{
			name:    "PDF17 with a4 alias returns ErrConformanceRequiresPDF20",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "a4"},
			wantErr: ErrConformanceRequiresPDF20,
		},
		{
			name:    "PDF17 with PDF/UA-2 returns ErrConformanceRequiresPDF20",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/UA-2"},
			wantErr: ErrConformanceRequiresPDF20,
		},
		{
			name:    "PDF17 with ua2 alias returns ErrConformanceRequiresPDF20",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "ua2"},
			wantErr: ErrConformanceRequiresPDF20,
		},
		// Unsupported legacy PDF/A-1 profiles
		{
			name:    "PDF17 with PDF/A-1 returns ErrPDFA1Unsupported",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-1"},
			wantErr: ErrPDFA1Unsupported,
		},
		{
			name:    "PDF17 with PDF/A-1b returns ErrPDFA1Unsupported",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-1b"},
			wantErr: ErrPDFA1Unsupported,
		},
		{
			name:    "PDF17 with a1 alias returns ErrPDFA1Unsupported",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "a1"},
			wantErr: ErrPDFA1Unsupported,
		},
		{
			name:    "PDF17 with a1b alias returns ErrPDFA1Unsupported",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "a1b"},
			wantErr: ErrPDFA1Unsupported,
		},
		// Unknown profiles
		{
			name:    "PDF17 with unknown profile returns ErrUnknownConformanceProfile",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "unrecognized"},
			wantErr: ErrUnknownConformanceProfile,
		},
		{
			name:    "PDF20 with unknown profile returns ErrUnknownConformanceProfile",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: "unrecognized"},
			wantErr: ErrUnknownConformanceProfile,
		},
		// Feature gates with profile
		{
			name:    "PDF17 with encryption and profile fails closed",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFA3a, Encryption: true},
			wantErr: ErrEncryptionUnsupported,
		},
		{
			name:    "PDF20 with encryption and profile fails closed",
			policy:  WriterPolicy{Version: PDF20, ConformanceProfile: ProfilePDFA4, Encryption: true},
			wantErr: ErrEncryptionUnsupported,
		},
		{
			name:    "PDF17 with forms and profile fails closed",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFA3a, Forms: true},
			wantErr: ErrFormsUnsupported,
		},
		{
			name:    "PDF17 with signatures and profile fails closed",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFA3a, Signatures: true},
			wantErr: ErrSignaturesUnsupported,
		},
		{
			name:    "PDF17 with object streams and profile fails closed",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFA3a, ObjectStreams: true},
			wantErr: ErrObjectStreamsUnsupported,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.policy.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate() err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestSRGBICCProfileStructure(t *testing.T) {
	t.Parallel()

	icc := sRGBICCProfile()
	if len(icc) != sRGBICCProfileSize {
		t.Fatalf("sRGBICCProfile length = %d, want %d", len(icc), sRGBICCProfileSize)
	}

	size := binary.BigEndian.Uint32(icc[0:4])
	if size != sRGBICCProfileSize {
		t.Errorf("ICC profile header size = %d, want %d", size, sRGBICCProfileSize)
	}

	if string(icc[12:16]) != "mntr" {
		t.Errorf("ICC device class = %q, want 'mntr'", string(icc[12:16]))
	}

	if string(icc[16:20]) != "RGB " {
		t.Errorf("ICC color space = %q, want 'RGB '", string(icc[16:20]))
	}

	if string(icc[20:24]) != "XYZ " {
		t.Errorf("ICC PCS = %q, want 'XYZ '", string(icc[20:24]))
	}

	if string(icc[36:40]) != "acsp" {
		t.Errorf("ICC signature = %q, want 'acsp'", string(icc[36:40]))
	}

	// Verify tag count
	tagCount := binary.BigEndian.Uint32(icc[128:132])
	if tagCount != 9 {
		t.Errorf("ICC tag count = %d, want 9", tagCount)
	}
}

func TestPDFA3aSingleProfileEmission(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFA3a,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "PDF/A-3a Archival Test")
	doc.SetInfo("Author", "Test Author")

	page := doc.AddPage(300, 400)
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(20, 350)
	content.TextShow("Archival PDF/A-3a content")
	content.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// 1. Must start with %PDF-1.7
	if !strings.HasPrefix(outStr, "%PDF-1.7\n%\xe2\xe3\xcf\xd3\n") {
		t.Errorf("PDF header mismatch: %q", outStr[:min(len(outStr), 25)])
	}

	// 2. Catalog has /OutputIntents
	if !strings.Contains(outStr, "/OutputIntents [") {
		t.Errorf("Catalog missing /OutputIntents array, output:\n%s", outStr)
	}

	// 3. OutputIntent object exists with /S /GTS_PDFA1 and sRGB identifier
	if !strings.Contains(outStr, "/Type /OutputIntent") ||
		!strings.Contains(outStr, "/S /GTS_PDFA1") ||
		!strings.Contains(outStr, "/OutputConditionIdentifier (sRGB IEC61966-2.1)") ||
		!strings.Contains(outStr, "/DestOutputProfile ") {
		t.Errorf("Missing valid OutputIntent object in output:\n%s", outStr)
	}

	// 4. ICC stream object exists with /N 3 and /Filter /FlateDecode
	if !strings.Contains(outStr, "/N 3") {
		t.Errorf("Missing /N 3 on ICC profile stream object")
	}

	// 5. Metadata stream has pdfaid claims
	if !strings.Contains(outStr, "<pdfaid:part>3</pdfaid:part>") {
		t.Errorf("Metadata missing <pdfaid:part>3</pdfaid:part>")
	}
	if !strings.Contains(outStr, "<pdfaid:conformance>A</pdfaid:conformance>") {
		t.Errorf("Metadata missing <pdfaid:conformance>A</pdfaid:conformance>")
	}

	// 6. Single PDF/A-3a document must NOT contain pdfuaid or pdfaExtension
	if strings.Contains(outStr, "pdfuaid") {
		t.Errorf("Single PDF/A-3a document should not contain pdfuaid")
	}
	if strings.Contains(outStr, "pdfaExtension") {
		t.Errorf("Single PDF/A-3a document should not contain pdfaExtension")
	}

	// 7. Page resources include DefaultRGB mapped to ICCBased
	if !strings.Contains(outStr, "/ColorSpace << /DefaultRGB [/ICCBased ") {
		t.Errorf("Page resources missing /DefaultRGB [/ICCBased ...]")
	}

	// 8. Info dictionary ModDate/CreationDate match XMP dates
	if !strings.Contains(outStr, "<xmp:CreateDate>2026-08-14T16:00:00Z</xmp:CreateDate>") {
		t.Errorf("XMP missing create date")
	}
	if !strings.Contains(outStr, "/CreationDate (D:20260814160000Z)") {
		t.Errorf("Info missing matching CreationDate")
	}
}

func TestDualPDFA3aPDFUA1ProfileEmission(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFA3aPDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "Dual Profile PDF/A-3a + PDF/UA-1 Test")

	page := doc.AddPage(300, 400)
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(20, 350)
	content.TextShow("Dual compliant content")
	content.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// 1. OutputIntent + ICC present
	if !strings.Contains(outStr, "/OutputIntents [") {
		t.Errorf("Catalog missing /OutputIntents")
	}
	if !strings.Contains(outStr, "/Type /OutputIntent") {
		t.Errorf("Missing /Type /OutputIntent")
	}

	// 2. Both pdfaid and pdfuaid in XMP
	if !strings.Contains(outStr, "<pdfaid:part>3</pdfaid:part>") {
		t.Errorf("Missing <pdfaid:part>3</pdfaid:part>")
	}
	if !strings.Contains(outStr, "<pdfaid:conformance>A</pdfaid:conformance>") {
		t.Errorf("Missing <pdfaid:conformance>A</pdfaid:conformance>")
	}
	if !strings.Contains(outStr, "<pdfuaid:part>1</pdfuaid:part>") {
		t.Errorf("Missing <pdfuaid:part>1</pdfuaid:part>")
	}

	// 3. Extension schema for pdfuaid is present
	if !strings.Contains(outStr, "<pdfaExtension:schemas>") ||
		!strings.Contains(outStr, "<pdfaSchema:prefix>pdfuaid</pdfaSchema:prefix>") ||
		!strings.Contains(outStr, "<pdfaSchema:namespaceURI>http://www.aiim.org/pdfua/ns/id/</pdfaSchema:namespaceURI>") ||
		!strings.Contains(outStr, "<pdfaProperty:name>part</pdfaProperty:name>") {
		t.Errorf("Dual profile XMP missing required pdfaExtension declaration")
	}

	// 4. DefaultRGB is on page resources
	if !strings.Contains(outStr, "/ColorSpace << /DefaultRGB [/ICCBased ") {
		t.Errorf("Page resources missing DefaultRGB")
	}
}

func TestPDFUA1SingleProfileEmission(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetInfo("Title", "PDF/UA-1 Single Profile Test")
	page := doc.AddPage(300, 400)
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(20, 350)
	content.TextShow("PDF/UA-1 text")
	content.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// 1. pdfuaid present
	if !strings.Contains(outStr, "<pdfuaid:part>1</pdfuaid:part>") {
		t.Errorf("Metadata missing <pdfuaid:part>1</pdfuaid:part>")
	}

	// 2. pdfaid NOT present in single PDF/UA-1
	if strings.Contains(outStr, "pdfaid") {
		t.Errorf("Single PDF/UA-1 should not contain pdfaid")
	}

	// 3. OutputIntents and DefaultRGB NOT present in single PDF/UA-1
	if strings.Contains(outStr, "/OutputIntents") {
		t.Errorf("Single PDF/UA-1 should not emit /OutputIntents")
	}
	if strings.Contains(outStr, "/DefaultRGB") {
		t.Errorf("Single PDF/UA-1 should not emit /DefaultRGB")
	}
}

func TestUnclaimedPDF17HasNoClaims(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetInfo("Title", "Unclaimed PDF 1.7")
	doc.AddPage(300, 400)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	if strings.Contains(outStr, "pdfaid") {
		t.Errorf("Unclaimed 1.7 contains forbidden token pdfaid")
	}
	if strings.Contains(outStr, "pdfuaid") {
		t.Errorf("Unclaimed 1.7 contains forbidden token pdfuaid")
	}
	if strings.Contains(outStr, "pdfaExtension") {
		t.Errorf("Unclaimed 1.7 contains forbidden token pdfaExtension")
	}
	if strings.Contains(outStr, "/OutputIntents") {
		t.Errorf("Unclaimed 1.7 contains /OutputIntents")
	}
	if strings.Contains(outStr, "/DefaultRGB") {
		t.Errorf("Unclaimed 1.7 contains /DefaultRGB")
	}
}

func makeTestPNG(t *testing.T, alpha bool) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	for y := range 10 {
		for x := range 10 {
			a := uint8(255)
			if alpha && x < 5 {
				a = 128
			}

			img.Set(x, y, color.RGBA{R: uint8(x * 20), G: uint8(y * 20), B: 150, A: a})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode error: %v", err)
	}

	return buf.Bytes()
}

func makeTestJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	for y := range 10 {
		for x := range 10 {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("jpeg.Encode error: %v", err)
	}

	return buf.Bytes()
}

func TestImagesAndFontsUnderPDFA3(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFA3a,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetInfo("Title", "Images and Fonts under A-3")
	page := doc.AddPage(400, 400)
	content := page.Content()

	// Embed font
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 14)
	content.TextAt(20, 380)
	content.TextShow("Hello PDF/A-3a with images!")
	content.EndText()

	// Add PNG with alpha (soft mask)
	if err := content.AddPNGImage("Png1", 20, 200, 80, 80, makeTestPNG(t, true)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}

	// Add JPEG image
	if err := content.AddJPEGImage("Jpg1", 120, 200, 80, 80, makeTestJPEG(t)); err != nil {
		t.Fatalf("AddJPEGImage: %v", err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// Verify Soft-Mask for PNG alpha is present
	if !strings.Contains(outStr, "/SMask ") {
		t.Errorf("PNG with alpha should emit /SMask under PDF/A-3")
	}

	// Verify DefaultRGB is present in page resources
	if !strings.Contains(outStr, "/ColorSpace << /DefaultRGB [/ICCBased ") {
		t.Errorf("Page resources should specify DefaultRGB for ICC-managed color under PDF/A-3")
	}

	// Verify font has ToUnicode and FontDescriptor with FontFile2
	if !strings.Contains(outStr, "/ToUnicode ") {
		t.Errorf("Embedded font must have /ToUnicode")
	}

	if !strings.Contains(outStr, "/FontFile2 ") {
		t.Errorf("Embedded font must have /FontFile2")
	}
}

func TestUnembeddedFontFailsClosedUnderCompliance(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFA3a,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetInfo("Title", "Unembedded font test")
	page := doc.AddPage(200, 200)
	content := page.Content()

	// Mark font as used without setting fontFiles
	content.fontUses["FMissing"] = ""
	content.BeginText()
	content.TextShow("Missing font")
	content.EndText()

	var buf bytes.Buffer

	errWrite := doc.Write(&buf)
	if errWrite == nil {
		t.Fatalf("expected error when unembedded font is used in compliant profile, got nil")
	}

	if !strings.Contains(errWrite.Error(), "not embedded") {
		t.Errorf("error message should mention 'not embedded', got: %v", errWrite)
	}
}

func TestType0UnicodeFontUnderPDFA3(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFA3a,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetInfo("Title", "Type0 Unicode font test")
	page := doc.AddPage(200, 200)
	content := page.Content()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(20, 100)
	// Code point above 0xFF to force Type0
	content.TextShow("Unicode \u2014 em dash & euro \u20ac")
	content.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()
	if !strings.Contains(outStr, "/Subtype /Type0") {
		t.Errorf("Expected Type0 font in PDF output for Unicode runes")
	}

	if !strings.Contains(outStr, "/ToUnicode ") {
		t.Errorf("Expected /ToUnicode on Type0 font")
	}
}

func TestPDFUA1TitleRequired(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF17,
		ConformanceProfile: ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	// No title set
	doc.AddPage(200, 200)

	var buf bytes.Buffer

	errWrite := doc.Write(&buf)
	if !errors.Is(errWrite, ErrTitleRequired) {
		t.Fatalf("expected ErrTitleRequired when PDF/UA-1 has no title, got: %v", errWrite)
	}
}

func TestGrayICCProfileStructure(t *testing.T) {
	t.Parallel()

	icc := grayICCProfile()
	if len(icc) != grayICCProfileSize {
		t.Fatalf("grayICCProfile length = %d, want %d", len(icc), grayICCProfileSize)
	}

	size := binary.BigEndian.Uint32(icc[0:4])
	if size != grayICCProfileSize {
		t.Errorf("ICC profile header size = %d, want %d", size, grayICCProfileSize)
	}

	if string(icc[12:16]) != "mntr" {
		t.Errorf("ICC device class = %q, want 'mntr'", string(icc[12:16]))
	}

	if string(icc[16:20]) != "GRAY" {
		t.Errorf("ICC color space = %q, want 'GRAY'", string(icc[16:20]))
	}

	if string(icc[20:24]) != "XYZ " {
		t.Errorf("ICC PCS = %q, want 'XYZ '", string(icc[20:24]))
	}

	if string(icc[36:40]) != "acsp" {
		t.Errorf("ICC signature = %q, want 'acsp'", string(icc[36:40]))
	}

	// Verify tag count
	tagCount := binary.BigEndian.Uint32(icc[128:132])
	if tagCount != 4 {
		t.Errorf("ICC tag count = %d, want 4", tagCount)
	}
}

func TestPDFA4SingleProfileEmission(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF20,
		ConformanceProfile: ProfilePDFA4,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "PDF/A-4 Archival Test")
	doc.SetInfo("Author", "Test Author")

	page := doc.AddPage(300, 400)
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(20, 350)
	content.TextShow("Archival PDF/A-4 content")
	content.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// 1. Must start with %PDF-2.0
	if !strings.HasPrefix(outStr, "%PDF-2.0\n%\xe2\xe3\xcf\xd3\n") {
		t.Errorf("PDF header mismatch: %q", outStr[:min(len(outStr), 25)])
	}

	// 2. Catalog has /OutputIntents
	if !strings.Contains(outStr, "/OutputIntents [") {
		t.Errorf("Catalog missing /OutputIntents array, output:\n%s", outStr)
	}

	// 3. OutputIntent object exists with /S /GTS_PDFA1 and sRGB identifier
	if !strings.Contains(outStr, "/Type /OutputIntent") ||
		!strings.Contains(outStr, "/S /GTS_PDFA1") ||
		!strings.Contains(outStr, "/OutputConditionIdentifier (sRGB IEC61966-2.1)") ||
		!strings.Contains(outStr, "/DestOutputProfile ") {
		t.Errorf("Missing valid OutputIntent object in output:\n%s", outStr)
	}

	// 4. ICC stream objects exist for sRGB (/N 3) and Gray (/N 1)
	if !strings.Contains(outStr, "/N 3") {
		t.Errorf("Missing /N 3 on sRGB ICC profile stream object")
	}
	if !strings.Contains(outStr, "/N 1") {
		t.Errorf("Missing /N 1 on Gray ICC profile stream object")
	}

	// 5. Metadata stream has pdfaid claims for part 4 / rev 2020 (no conformance element)
	if !strings.Contains(outStr, "<pdfaid:part>4</pdfaid:part>") {
		t.Errorf("Metadata missing <pdfaid:part>4</pdfaid:part>")
	}
	if !strings.Contains(outStr, "<pdfaid:rev>2020</pdfaid:rev>") {
		t.Errorf("Metadata missing <pdfaid:rev>2020</pdfaid:rev>")
	}
	if strings.Contains(outStr, "<pdfaid:conformance>") {
		t.Errorf("PDF/A-4 should not contain <pdfaid:conformance>")
	}

	// 6. Single PDF/A-4 document must NOT contain pdfuaid or pdfaExtension
	if strings.Contains(outStr, "pdfuaid") {
		t.Errorf("Single PDF/A-4 document should not contain pdfuaid")
	}
	if strings.Contains(outStr, "pdfaExtension") {
		t.Errorf("Single PDF/A-4 document should not contain pdfaExtension")
	}

	// 7. Page resources include DefaultRGB and DefaultGray mapped to ICCBased
	if !strings.Contains(outStr, "/DefaultRGB [/ICCBased ") {
		t.Errorf("Page resources missing DefaultRGB")
	}
	if !strings.Contains(outStr, "/DefaultGray [/ICCBased ") {
		t.Errorf("Page resources missing DefaultGray")
	}

	// 8. Trailer Info dictionary is OMITTED under PDF/A-4
	trailerIdx := strings.LastIndex(outStr, "trailer\n")
	if trailerIdx >= 0 {
		trailerPart := outStr[trailerIdx:]
		if strings.Contains(trailerPart, "/Info ") {
			t.Errorf("PDF/A-4 trailer must omit /Info, found in: %s", trailerPart)
		}
	}
}

func TestPDFUA2SingleProfileEmission(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF20,
		ConformanceProfile: ProfilePDFUA2,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetInfo("Title", "PDF/UA-2 Single Profile Test")
	page := doc.AddPage(300, 400)
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(20, 350)
	content.TextShow("PDF/UA-2 text")
	content.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// 1. pdfuaid part 2 / rev 2024 present
	if !strings.Contains(outStr, "<pdfuaid:part>2</pdfuaid:part>") {
		t.Errorf("Metadata missing <pdfuaid:part>2</pdfuaid:part>")
	}
	if !strings.Contains(outStr, "<pdfuaid:rev>2024</pdfuaid:rev>") {
		t.Errorf("Metadata missing <pdfuaid:rev>2024</pdfuaid:rev>")
	}

	// 2. pdfaid NOT present in single PDF/UA-2
	if strings.Contains(outStr, "pdfaid") {
		t.Errorf("Single PDF/UA-2 should not contain pdfaid")
	}

	// 3. Namespace object (/Type /Namespace /NS (http://iso.org/pdf2/ssn)) present
	if !strings.Contains(outStr, "/Type /Namespace") || !strings.Contains(outStr, "/NS (http://iso.org/pdf2/ssn)") {
		t.Errorf("Missing PDF 2.0 /Namespace object in PDF/UA-2 output")
	}

	// 4. StructTreeRoot /Namespaces present
	if !strings.Contains(outStr, "/Namespaces [") {
		t.Errorf("StructTreeRoot missing /Namespaces array in PDF/UA-2 output")
	}

	// 5. Document StructElem has /NS
	if !strings.Contains(outStr, "/S /Document") || !strings.Contains(outStr, "/NS ") {
		t.Errorf("Document StructElem missing /NS reference in PDF/UA-2 output")
	}

	// 6. OutputIntents and DefaultRGB NOT present in single PDF/UA-2
	if strings.Contains(outStr, "/OutputIntents") {
		t.Errorf("Single PDF/UA-2 should not emit /OutputIntents")
	}
	if strings.Contains(outStr, "/DefaultRGB") {
		t.Errorf("Single PDF/UA-2 should not emit /DefaultRGB")
	}
}

func TestDualPDFA4PDFUA2ProfileEmission(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF20,
		ConformanceProfile: ProfilePDFA4PDFUA2,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "Dual Profile PDF/A-4 + PDF/UA-2 Test")

	page := doc.AddPage(300, 400)
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(20, 350)
	content.TextShow("Dual 2.0 compliant content")
	content.EndText()

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// 1. OutputIntent + ICC present
	if !strings.Contains(outStr, "/OutputIntents [") {
		t.Errorf("Catalog missing /OutputIntents")
	}
	if !strings.Contains(outStr, "/Type /OutputIntent") {
		t.Errorf("Missing /Type /OutputIntent")
	}

	// 2. Both pdfaid (part 4, rev 2020) and pdfuaid (part 2, rev 2024) in XMP
	if !strings.Contains(outStr, "<pdfaid:part>4</pdfaid:part>") {
		t.Errorf("Missing <pdfaid:part>4</pdfaid:part>")
	}
	if !strings.Contains(outStr, "<pdfaid:rev>2020</pdfaid:rev>") {
		t.Errorf("Missing <pdfaid:rev>2020</pdfaid:rev>")
	}
	if !strings.Contains(outStr, "<pdfuaid:part>2</pdfuaid:part>") {
		t.Errorf("Missing <pdfuaid:part>2</pdfuaid:part>")
	}
	if !strings.Contains(outStr, "<pdfuaid:rev>2024</pdfuaid:rev>") {
		t.Errorf("Missing <pdfuaid:rev>2024</pdfuaid:rev>")
	}

	// 3. Extension schema for pdfuaid is present with part and rev
	if !strings.Contains(outStr, "<pdfaExtension:schemas>") ||
		!strings.Contains(outStr, "<pdfaSchema:prefix>pdfuaid</pdfaSchema:prefix>") ||
		!strings.Contains(outStr, "<pdfaProperty:name>part</pdfaProperty:name>") ||
		!strings.Contains(outStr, "<pdfaProperty:name>rev</pdfaProperty:name>") {
		t.Errorf("Dual profile XMP missing required pdfaExtension declaration")
	}

	// 4. DefaultRGB and DefaultGray are on page resources
	if !strings.Contains(outStr, "/DefaultRGB [/ICCBased ") {
		t.Errorf("Page resources missing DefaultRGB")
	}
	if !strings.Contains(outStr, "/DefaultGray [/ICCBased ") {
		t.Errorf("Page resources missing DefaultGray")
	}

	// 5. Namespaces object and references
	if !strings.Contains(outStr, "/Type /Namespace") {
		t.Errorf("Missing /Type /Namespace")
	}
	if !strings.Contains(outStr, "/Namespaces [") {
		t.Errorf("Missing /Namespaces array on StructTreeRoot")
	}

	// 6. Trailer Info is omitted under A-4
	trailerIdx := strings.LastIndex(outStr, "trailer\n")
	if trailerIdx >= 0 {
		trailerPart := outStr[trailerIdx:]
		if strings.Contains(trailerPart, "/Info ") {
			t.Errorf("Dual A-4/UA-2 trailer must omit /Info, found in: %s", trailerPart)
		}
	}
}

func TestImagesAndFontsUnderPDFA4(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF20,
		ConformanceProfile: ProfilePDFA4,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	doc.SetInfo("Title", "Images and Fonts under A-4")
	page := doc.AddPage(400, 400)
	content := page.Content()

	// Embed font
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 14)
	content.TextAt(20, 380)
	content.TextShow("Hello PDF/A-4 with images!")
	content.EndText()

	// Add PNG with alpha (soft mask)
	if err := content.AddPNGImage("Png1", 20, 200, 80, 80, makeTestPNG(t, true)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}

	// Add JPEG image
	if err := content.AddJPEGImage("Jpg1", 120, 200, 80, 80, makeTestJPEG(t)); err != nil {
		t.Fatalf("AddJPEGImage: %v", err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write error: %v", err)
	}

	outStr := buf.String()

	// Verify Soft-Mask for PNG alpha is present
	if !strings.Contains(outStr, "/SMask ") {
		t.Errorf("PNG with alpha should emit /SMask under PDF/A-4")
	}

	// Verify DefaultRGB and DefaultGray are present in page resources
	if !strings.Contains(outStr, "/DefaultRGB [/ICCBased ") {
		t.Errorf("Page resources should specify DefaultRGB for ICC-managed color under PDF/A-4")
	}
	if !strings.Contains(outStr, "/DefaultGray [/ICCBased ") {
		t.Errorf("Page resources should specify DefaultGray for ICC-managed color under PDF/A-4")
	}

	// Verify font has ToUnicode and FontDescriptor with FontFile2
	if !strings.Contains(outStr, "/ToUnicode ") {
		t.Errorf("Embedded font must have /ToUnicode")
	}

	if !strings.Contains(outStr, "/FontFile2 ") {
		t.Errorf("Embedded font must have /FontFile2")
	}
}

func TestPDFUA2TitleRequired(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{
		Version:            PDF20,
		ConformanceProfile: ProfilePDFUA2,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy error: %v", err)
	}

	// No title set
	doc.AddPage(200, 200)

	var buf bytes.Buffer

	errWrite := doc.Write(&buf)
	if !errors.Is(errWrite, ErrTitleRequired) {
		t.Fatalf("expected ErrTitleRequired when PDF/UA-2 has no title, got: %v", errWrite)
	}
}
