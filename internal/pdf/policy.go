package pdf

import (
	"errors"
	"strings"
)

// PDFVersion identifies a supported or reserved PDF version.
//
//nolint:revive // PDFVersion naming is intentional for explicit clarity across packages
type PDFVersion int

const (
	// PDF14 is PDF version 1.4 (default).
	PDF14 PDFVersion = iota
	// PDF17 is PDF version 1.7 (ISO 32000-1).
	PDF17
	// PDF20 is PDF version 2.0 (ISO 32000-2, reserved for issue #32).
	PDF20
)

var (
	// ErrUnsupportedPDFVersion indicates an unknown or out-of-range PDF version.
	ErrUnsupportedPDFVersion = errors.New("pdf: unsupported PDF version")
	// ErrReservedPDF20 indicates PDF 2.0 is not yet implemented.
	ErrReservedPDF20 = errors.New("pdf: PDF 2.0 support is planned for issue #32")
	// ErrEncryptionUnsupported indicates PDF encryption is not supported.
	ErrEncryptionUnsupported = errors.New("pdf: encryption is unsupported")
	// ErrFormsUnsupported indicates AcroForms / interactive forms are not supported.
	ErrFormsUnsupported = errors.New("pdf: interactive forms (AcroForms) are unsupported")
	// ErrSignaturesUnsupported indicates digital signatures are not supported.
	ErrSignaturesUnsupported = errors.New("pdf: digital signatures are unsupported")
	// ErrObjectStreamsUnsupported indicates object streams and compressed xref are not supported.
	ErrObjectStreamsUnsupported = errors.New("pdf: object streams and compressed xref are unsupported")
	// ErrConformanceProfilesUnsupported indicates PDF 2.0 conformance profiles (PDF/A-4, PDF/UA-2) are not supported.
	ErrConformanceProfilesUnsupported = errors.New(
		"pdf: conformance profiles (PDF/A, PDF/UA) are unsupported (deferred to issue #33)",
	)
	// ErrConformanceRequiresPDF17 indicates a conformance profile was requested without PDF 1.7.
	ErrConformanceRequiresPDF17 = errors.New("pdf: conformance profile requires PDF 1.7")
	// ErrProfileRequiresPDF17 is an alias for ErrConformanceRequiresPDF17.
	ErrProfileRequiresPDF17 = ErrConformanceRequiresPDF17
	// ErrPDFA1Unsupported indicates PDF/A-1 is unsupported.
	ErrPDFA1Unsupported = errors.New("pdf: PDF/A-1 is unsupported")
	// ErrUnknownConformanceProfile indicates an unrecognized conformance profile string.
	ErrUnknownConformanceProfile = errors.New("pdf: unknown conformance profile")
	// ErrTitleRequired indicates that PDF/UA-1 requires a non-empty document title.
	ErrTitleRequired = errors.New("pdf: PDF/UA-1 requires a non-empty document title")
	// ErrPDFUAMissingAlt indicates that PDF/UA-1 requires non-empty alt text for figures (images).
	ErrPDFUAMissingAlt = errors.New("pdf: PDF/UA-1 requires non-empty alt text for figures (images)")
	// ErrMissingImageAlt is an alias for ErrPDFUAMissingAlt.
	ErrMissingImageAlt = ErrPDFUAMissingAlt
	errNilFont         = errors.New("pdf: cannot embed nil font")
	errFontNotEmbedded = errors.New("pdf: font is not embedded")
)

const (
	versionToken14 = "1.4"
	versionToken17 = "1.7"
	versionToken20 = "2.0"
)

const (
	// ProfileNone indicates no conformance profile (standard unconstrained PDF).
	ProfileNone = ""
	// ProfilePDFA3a indicates PDF/A-3a archival conformance (ISO 19005-3 Level A).
	ProfilePDFA3a = "PDF/A-3a"
	// ProfilePDFUA1 indicates PDF/UA-1 accessibility conformance (ISO 14289-1).
	ProfilePDFUA1 = "PDF/UA-1"
	// ProfilePDFA3aPDFUA1 indicates combined PDF/A-3a and PDF/UA-1 conformance.
	ProfilePDFA3aPDFUA1 = "PDF/A-3a+PDF/UA-1"
	// ProfileDualA3aUA1 is an alias for ProfilePDFA3aPDFUA1.
	ProfileDualA3aUA1 = ProfilePDFA3aPDFUA1
)

// WriterPolicy configures document-level serialization behavior.
type WriterPolicy struct {
	Version            PDFVersion
	Encryption         bool
	Forms              bool
	Signatures         bool
	ObjectStreams      bool
	ConformanceProfile string
}

// CanonicalProfile normalizes profile strings (and common aliases) to canonical constants.
// Returns an empty string if the profile is not recognized or is empty.
func (p WriterPolicy) CanonicalProfile() string {
	raw := strings.TrimSpace(strings.ToLower(p.ConformanceProfile))
	if raw == "" {
		return ""
	}

	cleaned := strings.ReplaceAll(raw, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "_", "")

	switch cleaned {
	case "pdf/a-3a+pdf/ua-1", "pdf/a-3a-pdf/ua-1", "pdf/a-3a+ua-1", "pdf/a-3a-ua-1",
		"pdfa-3a+pdfua-1", "pdfa-3a-pdfua-1", "pdfa3a+pdfua1", "pdfa3a-pdfua1",
		"a3a+ua1", "a3a-ua1", "a3a,ua1", "a3a+pdf/ua-1", "a3a-pdf/ua-1",
		"pdf/ua-1+pdf/a-3a", "pdf/ua-1-pdf/a-3a", "pdfua-1+pdfa-3a", "pdfua1+pdfa3a",
		"ua1+a3a", "ua1-a3a", "ua1,a3a", "ua1+pdf/a-3a", "ua1-pdf/a-3a",
		"a3+ua1", "a3-ua1", "ua1+a3", "ua1-a3", "pdf/a-3+pdf/ua-1", "pdf/a-3-pdf/ua-1":
		return ProfilePDFA3aPDFUA1
	case "pdf/a-3a", "pdf/a-3", "pdfa-3a", "pdfa-3", "pdf-a-3a", "pdf-a-3",
		"a-3a", "a-3", "a3a", "a3", "pdfa3a", "pdfa3":
		return ProfilePDFA3a
	case "pdf/ua-1", "pdf/ua", "pdfua-1", "pdfua", "pdf-ua-1", "pdf-ua",
		"ua-1", "ua1", "ua", "pdfua1":
		return ProfilePDFUA1
	}

	switch p.ConformanceProfile {
	case ProfilePDFA3a:
		return ProfilePDFA3a
	case ProfilePDFUA1:
		return ProfilePDFUA1
	case ProfilePDFA3aPDFUA1:
		return ProfilePDFA3aPDFUA1
	}

	return ""
}

// IsPDFA3 reports whether the policy specifies PDF/A-3 archival conformance.
func (p WriterPolicy) IsPDFA3() bool {
	c := p.CanonicalProfile()

	return c == ProfilePDFA3a || c == ProfilePDFA3aPDFUA1
}

// IsPDFUA1 reports whether the policy specifies PDF/UA-1 accessibility conformance.
func (p WriterPolicy) IsPDFUA1() bool {
	c := p.CanonicalProfile()

	return c == ProfilePDFUA1 || c == ProfilePDFA3aPDFUA1
}

// IsCompliant reports whether any valid conformance profile is active.
func (p WriterPolicy) IsCompliant() bool {
	return p.CanonicalProfile() != ""
}

// HasConformanceProfile reports whether a conformance profile is requested.
func (p WriterPolicy) HasConformanceProfile() bool {
	return p.ConformanceProfile != ""
}

// Validate checks whether the policy specifies a supported PDF version and valid feature set.
//
//nolint:cyclop // comprehensive matrix validation across versions and profiles
func (p WriterPolicy) Validate() error {
	if p.Version == PDF20 {
		return ErrReservedPDF20
	}

	if p.Version < PDF14 || p.Version > PDF20 {
		return ErrUnsupportedPDFVersion
	}

	if p.Encryption {
		return ErrEncryptionUnsupported
	}

	if p.Forms {
		return ErrFormsUnsupported
	}

	if p.Signatures {
		return ErrSignaturesUnsupported
	}

	if p.ObjectStreams {
		return ErrObjectStreamsUnsupported
	}

	if p.ConformanceProfile == "" {
		return nil
	}

	if p.Version != PDF17 {
		return ErrConformanceRequiresPDF17
	}

	if canonical := p.CanonicalProfile(); canonical != "" {
		return nil
	}

	raw := strings.TrimSpace(strings.ToLower(p.ConformanceProfile))
	cleaned := strings.ReplaceAll(raw, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "_", "")

	if isUnsupportedPDF20Profile(cleaned) {
		return ErrConformanceProfilesUnsupported
	}

	if isPDFA1Profile(cleaned) {
		return ErrPDFA1Unsupported
	}

	return ErrUnknownConformanceProfile
}

func isUnsupportedPDF20Profile(profileStr string) bool {
	switch profileStr {
	case "pdf/a-4", "pdf/a-4e", "pdf/a-4f", "pdfa-4", "pdfa4", "a4", "a-4",
		"pdf/ua-2", "pdfua-2", "pdfua2", "ua2", "ua-2",
		"pdf/a-4+pdf/ua-2", "a4-ua2", "a4+ua2", "ua2+a4":
		return true
	}

	return strings.Contains(profileStr, "a-4") || strings.Contains(profileStr, "a4") ||
		strings.Contains(profileStr, "ua-2") || strings.Contains(profileStr, "ua2")
}

func isPDFA1Profile(profileStr string) bool {
	switch profileStr {
	case "pdf/a-1", "pdf/a-1a", "pdf/a-1b", "pdfa-1", "pdfa-1a", "pdfa-1b",
		"pdfa1", "pdfa1a", "pdfa1b", "a1", "a1a", "a1b", "a-1", "a-1a", "a-1b":
		return true
	}

	return strings.Contains(profileStr, "a-1") || strings.Contains(profileStr, "a1")
}

// HeaderVersion returns the version token for the PDF file header (e.g. "1.4" or "1.7").
func (p WriterPolicy) HeaderVersion() string {
	switch p.Version {
	case PDF14:
		return versionToken14
	case PDF17:
		return versionToken17
	case PDF20:
		return versionToken20
	default:
		return versionToken14
	}
}

// ProducerVersion returns the producer string (e.g. "gowkhtmltopdf 1.4" or "gowkhtmltopdf 1.7").
func (p WriterPolicy) ProducerVersion() string {
	return "gowkhtmltopdf " + p.HeaderVersion()
}
