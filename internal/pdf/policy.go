package pdf

import "errors"

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
	// ErrConformanceProfilesUnsupported indicates standards conformance profiles (PDF/A, PDF/UA) are not supported.
	ErrConformanceProfilesUnsupported = errors.New(
		"pdf: conformance profiles (PDF/A, PDF/UA) are unsupported (deferred to issue #33)",
	)
)

const (
	versionToken14 = "1.4"
	versionToken17 = "1.7"
	versionToken20 = "2.0"
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

// Validate checks whether the policy specifies a supported PDF version and valid feature set.
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

	if p.ConformanceProfile != "" {
		return ErrConformanceProfilesUnsupported
	}

	return nil
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
