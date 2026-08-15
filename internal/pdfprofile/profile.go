package pdfprofile

import (
	"errors"
	"fmt"
	"strings"
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

var (
	// ErrInvalidPDFProfile reports an invalid or unsupported PDF conformance profile.
	ErrInvalidPDFProfile = errors.New(
		"settings: invalid pdf profile (allowed: a3a-ua1, a3a, ua1, a4-ua2, a4, ua2, " +
			"PDF/A-3a+PDF/UA-1, PDF/A-3a, PDF/UA-1, PDF/A-4+PDF/UA-2, PDF/A-4, PDF/UA-2)",
	)

	// ErrProfilePDF20Unsupported indicates PDF 2.0 conformance profiles are unsupported
	// (historical sentinel; never returned).
	ErrProfilePDF20Unsupported = errors.New(
		"settings: PDF 2.0 conformance profiles (PDF/A-4, PDF/UA-2) are unsupported",
	)

	// ErrProfilePDFA1Unsupported indicates PDF/A-1 is unsupported.
	ErrProfilePDFA1Unsupported = errors.New("settings: PDF/A-1 is unsupported")
)

// Canonical normalizes profile strings and aliases to canonical constants.
// Returns an empty string if the profile is unrecognized or empty.
func Canonical(value string) string {
	res, err := Parse(value)
	if err != nil {
		return ProfileNone
	}

	return res
}

// Parse validates and normalizes a PDF conformance profile string.
// Accepted values map to canonical constants: ProfilePDFA3aPDFUA1,
// ProfilePDFA3a, ProfilePDFUA1, ProfilePDFA4PDFUA2, ProfilePDFA4,
// ProfilePDFUA2, or ProfileNone ("").
// Invalid or unsupported values return an error wrapping the respective sentinel.
func Parse(value string) (string, error) {
	raw := strings.TrimSpace(strings.ToLower(value))
	if raw == "" {
		return ProfileNone, nil
	}

	cleaned := strings.ReplaceAll(raw, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "_", "")

	switch cleaned {
	case "pdf/a-3a+pdf/ua-1", "pdf/a-3a-pdf/ua-1", "pdf/a-3a+ua-1", "pdf/a-3a-ua-1",
		"pdfa-3a+pdfua-1", "pdfa-3a-pdfua-1", "pdfa3a+pdfua1", "pdfa3a-pdfua1",
		"a3a+ua1", "a3a-ua1", "a3a,ua1", "a3a+pdf/ua-1", "a3a-pdf/ua-1",
		"pdf/ua-1+pdf/a-3a", "pdf/ua-1-pdf/a-3a", "pdfua-1+pdfa-3a", "pdfua1+pdfa3a",
		"ua1+a3a", "ua1-a3a", "ua1,a3a", "ua1+pdf/a-3a", "ua1-pdf/a-3a":
		return ProfilePDFA3aPDFUA1, nil
	case "pdf/a-3a", "pdfa-3a", "pdf-a-3a",
		"a-3a", "a3a", "pdfa3a":
		return ProfilePDFA3a, nil
	case "pdf/ua-1", "pdfua-1", "pdf-ua-1",
		"ua-1", "ua1", "pdfua1":
		return ProfilePDFUA1, nil
	case "pdf/a-4+pdf/ua-2", "pdf/a-4-pdf/ua-2", "pdf/a-4+ua-2", "pdf/a-4-ua-2",
		"pdfa-4+pdfua-2", "pdfa-4-pdfua-2", "pdfa4+pdfua2", "pdfa4-pdfua2",
		"a4+ua2", "a4-ua2", "a4,ua2", "a4+pdf/ua-2", "a4-pdf/ua-2",
		"pdf/ua-2+pdf/a-4", "pdf/ua-2-pdf/a-4", "pdfua-2+pdfa-4", "pdfua2+pdfa4",
		"ua2+a4", "ua2-a4", "ua2,a4", "ua2+pdf/a-4", "ua2-pdf/a-4":
		return ProfilePDFA4PDFUA2, nil
	case "pdf/a-4", "pdf/a-4a", "pdf/a4", "pdfa-4", "pdfa-4a", "pdfa4", "pdfa4a",
		"pdf-a-4", "pdf-a-4a", "pdf-a4", "a-4", "a4":
		return ProfilePDFA4, nil
	case "pdf/ua-2", "pdf/ua2", "pdfua-2", "pdfua2", "pdf-ua-2", "pdf-ua2", "ua-2", "ua2":
		return ProfilePDFUA2, nil
	}

	if isUnsupportedProfilePDFA1(cleaned) {
		return "", ErrProfilePDFA1Unsupported
	}

	return "", fmt.Errorf("%w: %q", ErrInvalidPDFProfile, value)
}

func isUnsupportedProfilePDFA1(profileStr string) bool {
	switch profileStr {
	case "pdf/a-1", "pdf/a-1a", "pdf/a-1b", "pdfa-1", "pdfa-1a", "pdfa-1b",
		"pdf-a-1", "pdf-a-1a", "pdf-a-1b", "a-1", "a-1a", "a-1b", "a1", "a1a", "a1b",
		"pdfa1", "pdfa1a", "pdfa1b":
		return true
	default:
		return false
	}
}

// IsPDFA3 reports whether canonicalProfile specifies PDF/A-3 archival conformance.
func IsPDFA3(canonicalProfile string) bool {
	return canonicalProfile == ProfilePDFA3a || canonicalProfile == ProfilePDFA3aPDFUA1
}

// IsPDFA4 reports whether canonicalProfile specifies PDF/A-4 archival conformance.
func IsPDFA4(canonicalProfile string) bool {
	return canonicalProfile == ProfilePDFA4 || canonicalProfile == ProfilePDFA4PDFUA2
}

// IsPDFUA1 reports whether canonicalProfile specifies PDF/UA-1 accessibility conformance.
func IsPDFUA1(canonicalProfile string) bool {
	return canonicalProfile == ProfilePDFUA1 || canonicalProfile == ProfilePDFA3aPDFUA1
}

// IsPDFUA2 reports whether canonicalProfile specifies PDF/UA-2 accessibility conformance.
func IsPDFUA2(canonicalProfile string) bool {
	return canonicalProfile == ProfilePDFUA2 || canonicalProfile == ProfilePDFA4PDFUA2
}

// IsPDFUA reports whether canonicalProfile specifies either PDF/UA-1 or PDF/UA-2 conformance.
func IsPDFUA(canonicalProfile string) bool {
	return IsPDFUA1(canonicalProfile) || IsPDFUA2(canonicalProfile)
}
