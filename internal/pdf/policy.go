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
	// PDF20 is PDF version 2.0 (ISO 32000-2).
	PDF20
)

var (
	// ErrUnsupportedPDFVersion indicates an unknown or out-of-range PDF version.
	ErrUnsupportedPDFVersion = errors.New("pdf: unsupported PDF version")
	// ErrEncryptionUnsupported indicates PDF encryption is not supported.
	ErrEncryptionUnsupported = errors.New("pdf: encryption is unsupported")
	// ErrFormsUnsupported indicates AcroForms / interactive forms are not supported.
	ErrFormsUnsupported = errors.New("pdf: interactive forms (AcroForms) are unsupported")
	// ErrSignaturesUnsupported indicates digital signatures are not supported.
	ErrSignaturesUnsupported = errors.New("pdf: digital signatures are unsupported")
	// ErrObjectStreamsUnsupported indicates object streams and compressed xref are not supported.
	ErrObjectStreamsUnsupported = errors.New("pdf: object streams and compressed xref are unsupported")
	// ErrConformanceProfilesUnsupported indicates unsupported PDF conformance profiles.
	ErrConformanceProfilesUnsupported = errors.New(
		"pdf: conformance profiles (PDF/A, PDF/UA) are unsupported",
	)
	// ErrConformanceRequiresPDF17 indicates a 1.7 conformance profile was requested without PDF 1.7.
	ErrConformanceRequiresPDF17 = errors.New("pdf: conformance profile requires PDF 1.7")
	// ErrProfileRequiresPDF17 is an alias for ErrConformanceRequiresPDF17.
	ErrProfileRequiresPDF17 = ErrConformanceRequiresPDF17
	// ErrConformanceRequiresPDF20 indicates a 2.0 conformance profile was requested without PDF 2.0.
	ErrConformanceRequiresPDF20 = errors.New("pdf: conformance profile requires PDF 2.0")
	// ErrProfileRequiresPDF20 is an alias for ErrConformanceRequiresPDF20.
	ErrProfileRequiresPDF20 = ErrConformanceRequiresPDF20
	// ErrPDFA1Unsupported indicates PDF/A-1 is unsupported.
	ErrPDFA1Unsupported = errors.New("pdf: PDF/A-1 is unsupported")
	// ErrUnknownConformanceProfile indicates an unrecognized conformance profile string.
	ErrUnknownConformanceProfile = errors.New("pdf: unknown conformance profile")
	// ErrTitleRequired indicates that PDF/UA requires a non-empty document title.
	ErrTitleRequired = errors.New("pdf: PDF/UA requires a non-empty document title")
	// ErrPDFUAMissingAlt indicates that PDF/UA requires non-empty alt text for figures (images).
	ErrPDFUAMissingAlt = errors.New("pdf: PDF/UA requires non-empty alt text for figures (images)")
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
	// ProfilePDFA4 indicates PDF/A-4 archival conformance (ISO 19005-4).
	ProfilePDFA4 = "PDF/A-4"
	// ProfilePDFUA2 indicates PDF/UA-2 accessibility conformance (ISO 14289-2).
	ProfilePDFUA2 = "PDF/UA-2"
	// ProfilePDFA4PDFUA2 indicates combined PDF/A-4 and PDF/UA-2 conformance.
	ProfilePDFA4PDFUA2 = "PDF/A-4+PDF/UA-2"
	// ProfileDualA4UA2 is an alias for ProfilePDFA4PDFUA2.
	ProfileDualA4UA2 = ProfilePDFA4PDFUA2
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
//
//nolint:cyclop // comprehensive mapping of profile aliases across PDF versions
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
	case "pdf/a-4+pdf/ua-2", "pdf/a-4-pdf/ua-2", "pdf/a-4+ua-2", "pdf/a-4-ua-2",
		"pdfa-4+pdfua-2", "pdfa-4-pdfua-2", "pdfa4+pdfua2", "pdfa4-pdfua2",
		"a4+ua2", "a4-ua2", "a4,ua2", "a4+pdf/ua-2", "a4-pdf/ua-2",
		"pdf/ua-2+pdf/a-4", "pdf/ua-2-pdf/a-4", "pdfua-2+pdfa-4", "pdfua2+pdfa4",
		"ua2+a4", "ua2-a4", "ua2,a4", "ua2+pdf/a-4", "ua2-pdf/a-4",
		"a4+ua", "a4-ua", "ua2+a", "ua2-a",
		"pdf/a-4+pdf/ua", "pdf/a-4-pdf/ua", "pdf/ua+pdf/a-4", "pdf/ua-pdf/a-4":
		return ProfilePDFA4PDFUA2
	case "pdf/a-4", "pdf/a-4a", "pdf/a4", "pdfa-4", "pdfa-4a", "pdfa4", "pdfa4a",
		"pdf-a-4", "pdf-a-4a", "pdf-a4", "a-4", "a-4a", "a4", "a4a":
		return ProfilePDFA4
	case "pdf/ua-2", "pdf/ua2", "pdfua-2", "pdfua2", "pdf-ua-2", "pdf-ua2", "ua-2", "ua2":
		return ProfilePDFUA2
	}

	switch p.ConformanceProfile {
	case ProfilePDFA3a:
		return ProfilePDFA3a
	case ProfilePDFUA1:
		return ProfilePDFUA1
	case ProfilePDFA3aPDFUA1:
		return ProfilePDFA3aPDFUA1
	case ProfilePDFA4:
		return ProfilePDFA4
	case ProfilePDFUA2:
		return ProfilePDFUA2
	case ProfilePDFA4PDFUA2:
		return ProfilePDFA4PDFUA2
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

// IsPDFA4 reports whether the policy specifies PDF/A-4 archival conformance.
func (p WriterPolicy) IsPDFA4() bool {
	c := p.CanonicalProfile()

	return c == ProfilePDFA4 || c == ProfilePDFA4PDFUA2
}

// IsPDFUA2 reports whether the policy specifies PDF/UA-2 accessibility conformance.
func (p WriterPolicy) IsPDFUA2() bool {
	c := p.CanonicalProfile()

	return c == ProfilePDFUA2 || c == ProfilePDFA4PDFUA2
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

	canonical := p.CanonicalProfile()
	if canonical != "" {
		if isPDF17Profile(canonical) {
			if p.Version == PDF17 {
				return nil
			}

			return ErrConformanceRequiresPDF17
		}

		if isPDF20Profile(canonical) {
			if p.Version == PDF20 {
				return nil
			}

			return ErrConformanceRequiresPDF20
		}
	}

	cleaned := normalizeProfileToken(p.ConformanceProfile)

	if isPDFA1Profile(cleaned) {
		return ErrPDFA1Unsupported
	}

	return ErrUnknownConformanceProfile
}

// normalizeProfileToken lower-cases, trims, and strips spaces/underscores from
// a conformance profile string so matching can compare canonical tokens.
func normalizeProfileToken(profileStr string) string {
	raw := strings.TrimSpace(strings.ToLower(profileStr))
	cleaned := strings.ReplaceAll(raw, " ", "")

	return strings.ReplaceAll(cleaned, "_", "")
}

func isPDF17Profile(canonicalProfile string) bool {
	return canonicalProfile == ProfilePDFA3a ||
		canonicalProfile == ProfilePDFUA1 ||
		canonicalProfile == ProfilePDFA3aPDFUA1
}

func isPDF20Profile(canonicalProfile string) bool {
	return canonicalProfile == ProfilePDFA4 ||
		canonicalProfile == ProfilePDFUA2 ||
		canonicalProfile == ProfilePDFA4PDFUA2
}

func isPDFA1Profile(profileStr string) bool {
	switch profileStr {
	case "pdf/a-1", "pdf/a-1a", "pdf/a-1b", "pdfa-1", "pdfa-1a", "pdfa-1b",
		"pdfa1", "pdfa1a", "pdfa1b", "a1", "a1a", "a1b", "a-1", "a-1a", "a-1b",
		"pdf-a-1", "pdf-a-1a", "pdf-a-1b", "pdf-a1", "pdf-a1a", "pdf-a1b":
		return true
	}

	return strings.HasPrefix(profileStr, "pdf/a-1") ||
		strings.HasPrefix(profileStr, "pdfa-1") ||
		strings.HasPrefix(profileStr, "pdfa1") ||
		strings.HasPrefix(profileStr, "pdf-a-1") ||
		strings.HasPrefix(profileStr, "pdf-a1")
}

// HeaderVersion returns the version token for the PDF file header (e.g. "1.4", "1.7", or "2.0").
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

// ProducerVersion returns the producer string (e.g. "gowkhtmltopdf 1.4", "gowkhtmltopdf 1.7", or "gowkhtmltopdf 2.0").
func (p WriterPolicy) ProducerVersion() string {
	return "gowkhtmltopdf " + p.HeaderVersion()
}
