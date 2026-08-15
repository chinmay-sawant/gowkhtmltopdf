package pdf

import (
	"errors"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdfprofile"
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
	ProfileNone = pdfprofile.ProfileNone
	// ProfilePDFA3a indicates PDF/A-3a archival conformance (ISO 19005-3 Level A).
	ProfilePDFA3a = pdfprofile.ProfilePDFA3a
	// ProfilePDFUA1 indicates PDF/UA-1 accessibility conformance (ISO 14289-1).
	ProfilePDFUA1 = pdfprofile.ProfilePDFUA1
	// ProfilePDFA3aPDFUA1 indicates combined PDF/A-3a and PDF/UA-1 conformance.
	ProfilePDFA3aPDFUA1 = pdfprofile.ProfilePDFA3aPDFUA1
	// ProfileDualA3aUA1 is an alias for ProfilePDFA3aPDFUA1.
	ProfileDualA3aUA1 = pdfprofile.ProfileDualA3aUA1
	// ProfilePDFA4 indicates PDF/A-4 archival conformance (ISO 19005-4).
	ProfilePDFA4 = pdfprofile.ProfilePDFA4
	// ProfilePDFUA2 indicates PDF/UA-2 accessibility conformance (ISO 14289-2).
	ProfilePDFUA2 = pdfprofile.ProfilePDFUA2
	// ProfilePDFA4PDFUA2 indicates combined PDF/A-4 and PDF/UA-2 conformance.
	ProfilePDFA4PDFUA2 = pdfprofile.ProfilePDFA4PDFUA2
	// ProfileDualA4UA2 is an alias for ProfilePDFA4PDFUA2.
	ProfileDualA4UA2 = pdfprofile.ProfileDualA4UA2
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
	return pdfprofile.Canonical(p.ConformanceProfile)
}

// IsPDFA3 reports whether the policy specifies PDF/A-3 archival conformance.
func (p WriterPolicy) IsPDFA3() bool {
	return pdfprofile.IsPDFA3(p.CanonicalProfile())
}

// IsPDFUA1 reports whether the policy specifies PDF/UA-1 accessibility conformance.
func (p WriterPolicy) IsPDFUA1() bool {
	return pdfprofile.IsPDFUA1(p.CanonicalProfile())
}

// IsPDFA4 reports whether the policy specifies PDF/A-4 archival conformance.
func (p WriterPolicy) IsPDFA4() bool {
	return pdfprofile.IsPDFA4(p.CanonicalProfile())
}

// IsPDFUA2 reports whether the policy specifies PDF/UA-2 accessibility conformance.
func (p WriterPolicy) IsPDFUA2() bool {
	return pdfprofile.IsPDFUA2(p.CanonicalProfile())
}

// IsPDFUA reports whether the policy specifies either PDF/UA-1 or PDF/UA-2 accessibility conformance.
func (p WriterPolicy) IsPDFUA() bool {
	return pdfprofile.IsPDFUA(p.CanonicalProfile())
}

// IsCompliant reports whether any valid conformance profile is active.
func (p WriterPolicy) IsCompliant() bool {
	return p.CanonicalProfile() != ""
}

// HasConformanceProfile reports whether a conformance profile is requested.
func (p WriterPolicy) HasConformanceProfile() bool {
	return p.CanonicalProfile() != ""
}

// Validate checks whether the policy specifies a supported PDF version and valid feature set.
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

	canonical, err := pdfprofile.Parse(p.ConformanceProfile)
	if err != nil {
		if errors.Is(err, pdfprofile.ErrProfilePDFA1Unsupported) {
			return ErrPDFA1Unsupported
		}

		return ErrUnknownConformanceProfile
	}

	return validateCanonicalProfileVersion(canonical, p.Version)
}

// validateCanonicalProfileVersion ensures a known profile is paired with the
// required base PDF version. Empty canonical means "not a known alias".
func validateCanonicalProfileVersion(canonical string, version PDFVersion) error {
	if canonical == "" {
		return nil
	}

	if isPDF17Profile(canonical) {
		if version == PDF17 {
			return nil
		}

		return ErrConformanceRequiresPDF17
	}

	if isPDF20Profile(canonical) {
		if version == PDF20 {
			return nil
		}

		return ErrConformanceRequiresPDF20
	}

	// Known tokens only; unknown aliases fall through to ErrUnknown… above.
	return nil
}

func isPDF17Profile(canonicalProfile string) bool {
	return pdfprofile.IsPDFA3(canonicalProfile) || pdfprofile.IsPDFUA1(canonicalProfile)
}

func isPDF20Profile(canonicalProfile string) bool {
	return pdfprofile.IsPDFA4(canonicalProfile) || pdfprofile.IsPDFUA2(canonicalProfile)
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
