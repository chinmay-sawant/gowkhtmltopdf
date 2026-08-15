//nolint:cyclop,wsl,lll,varnamelen,nlreturn // comprehensive table-driven profile test suites
package pdfprofile_test

import (
	"errors"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdfprofile"
)

func TestParsePDFProfileValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"", pdfprofile.ProfileNone},
		{"   ", pdfprofile.ProfileNone},
		{"a3a-ua1", pdfprofile.ProfilePDFA3aPDFUA1},
		{"a3a+ua1", pdfprofile.ProfilePDFA3aPDFUA1},
		{"a3a,ua1", pdfprofile.ProfilePDFA3aPDFUA1},
		{"PDF/A-3a+PDF/UA-1", pdfprofile.ProfilePDFA3aPDFUA1},
		{"pdfa3a+pdfua1", pdfprofile.ProfilePDFA3aPDFUA1},
		{"ua1-a3a", pdfprofile.ProfilePDFA3aPDFUA1},
		{"a3a", pdfprofile.ProfilePDFA3a},
		{"PDF/A-3a", pdfprofile.ProfilePDFA3a},
		{"pdfa-3a", pdfprofile.ProfilePDFA3a},
		{"ua1", pdfprofile.ProfilePDFUA1},
		{"PDF/UA-1", pdfprofile.ProfilePDFUA1},
		{"pdfua-1", pdfprofile.ProfilePDFUA1},
		{"a4-ua2", pdfprofile.ProfilePDFA4PDFUA2},
		{"a4+ua2", pdfprofile.ProfilePDFA4PDFUA2},
		{"a4,ua2", pdfprofile.ProfilePDFA4PDFUA2},
		{"PDF/A-4+PDF/UA-2", pdfprofile.ProfilePDFA4PDFUA2},
		{"pdfa4+pdfua2", pdfprofile.ProfilePDFA4PDFUA2},
		{"ua2-a4", pdfprofile.ProfilePDFA4PDFUA2},
		{"a4", pdfprofile.ProfilePDFA4},
		{"PDF/A-4", pdfprofile.ProfilePDFA4},
		{"pdfa-4", pdfprofile.ProfilePDFA4},
		{"ua2", pdfprofile.ProfilePDFUA2},
		{"PDF/UA-2", pdfprofile.ProfilePDFUA2},
		{"pdfua-2", pdfprofile.ProfilePDFUA2},
	}

	for _, tc := range tests {
		got, err := pdfprofile.Parse(tc.input)
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Parse(%q) = %q, want %q", tc.input, got, tc.want)
		}
		if canonical := pdfprofile.Canonical(tc.input); canonical != tc.want {
			t.Errorf("Canonical(%q) = %q, want %q", tc.input, canonical, tc.want)
		}
	}
}

func TestParsePDFProfileRejected(t *testing.T) {
	t.Parallel()

	rejected := []string{
		"a3", "ua", "ua2+a", "a4+ua", "a3+ua1", "unknown", "pdf/a-2",
	}

	for _, input := range rejected {
		got, err := pdfprofile.Parse(input)
		if err == nil {
			t.Errorf("Parse(%q) expected error, got %q", input, got)
		}
		if !errors.Is(err, pdfprofile.ErrInvalidPDFProfile) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalidPDFProfile", input, err)
		}
		if canonical := pdfprofile.Canonical(input); canonical != "" {
			t.Errorf("Canonical(%q) = %q, want empty string", input, canonical)
		}
	}
}

func TestParsePDFProfileA1Unsupported(t *testing.T) {
	t.Parallel()

	a1Profiles := []string{
		"pdf/a-1", "pdf/a-1a", "pdf/a-1b", "pdfa-1", "a-1", "a1",
	}

	for _, input := range a1Profiles {
		got, err := pdfprofile.Parse(input)
		if err == nil {
			t.Errorf("Parse(%q) expected error, got %q", input, got)
		}
		if !errors.Is(err, pdfprofile.ErrProfilePDFA1Unsupported) {
			t.Errorf("Parse(%q) error = %v, want ErrProfilePDFA1Unsupported", input, err)
		}
	}
}

func TestProfilePredicates(t *testing.T) {
	t.Parallel()

	if !pdfprofile.IsPDFA3(pdfprofile.ProfilePDFA3a) || !pdfprofile.IsPDFA3(pdfprofile.ProfilePDFA3aPDFUA1) {
		t.Error("IsPDFA3 failed on A3 profiles")
	}
	if pdfprofile.IsPDFA3(pdfprofile.ProfilePDFA4) || pdfprofile.IsPDFA3(pdfprofile.ProfilePDFUA1) {
		t.Error("IsPDFA3 returned true for non-A3")
	}

	if !pdfprofile.IsPDFA4(pdfprofile.ProfilePDFA4) || !pdfprofile.IsPDFA4(pdfprofile.ProfilePDFA4PDFUA2) {
		t.Error("IsPDFA4 failed on A4 profiles")
	}
	if pdfprofile.IsPDFA4(pdfprofile.ProfilePDFA3a) || pdfprofile.IsPDFA4(pdfprofile.ProfilePDFUA2) {
		t.Error("IsPDFA4 returned true for non-A4")
	}

	if !pdfprofile.IsPDFUA1(pdfprofile.ProfilePDFUA1) || !pdfprofile.IsPDFUA1(pdfprofile.ProfilePDFA3aPDFUA1) {
		t.Error("IsPDFUA1 failed on UA1 profiles")
	}
	if pdfprofile.IsPDFUA1(pdfprofile.ProfilePDFUA2) {
		t.Error("IsPDFUA1 returned true for UA2")
	}

	if !pdfprofile.IsPDFUA2(pdfprofile.ProfilePDFUA2) || !pdfprofile.IsPDFUA2(pdfprofile.ProfilePDFA4PDFUA2) {
		t.Error("IsPDFUA2 failed on UA2 profiles")
	}
	if pdfprofile.IsPDFUA2(pdfprofile.ProfilePDFUA1) {
		t.Error("IsPDFUA2 returned true for UA1")
	}

	if !pdfprofile.IsPDFUA(pdfprofile.ProfilePDFUA1) || !pdfprofile.IsPDFUA(pdfprofile.ProfilePDFUA2) ||
		!pdfprofile.IsPDFUA(pdfprofile.ProfilePDFA3aPDFUA1) || !pdfprofile.IsPDFUA(pdfprofile.ProfilePDFA4PDFUA2) {
		t.Error("IsPDFUA failed on UA profiles")
	}
	if pdfprofile.IsPDFUA(pdfprofile.ProfilePDFA3a) || pdfprofile.IsPDFUA(pdfprofile.ProfilePDFA4) || pdfprofile.IsPDFUA("") {
		t.Error("IsPDFUA returned true for non-UA profiles")
	}
}
