package gowkhtmltopdf

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

var (
	// ErrNilDocument reports a method call on a nil Document receiver.
	ErrNilDocument = errors.New("gowkhtmltopdf: nil document")
	// ErrNilImageDocument reports a method call on a nil ImageDocument receiver.
	ErrNilImageDocument = errors.New("gowkhtmltopdf: nil image document")
	// ErrInvalidContent reports a Content value with an invalid source shape.
	ErrInvalidContent = errors.New("gowkhtmltopdf: invalid content")
	// ErrInvalidImageFormat reports an unsupported ImageDocument format.
	ErrInvalidImageFormat = errors.New("gowkhtmltopdf: invalid image format")
	// ErrEmptyContent is a content-oriented alias for the legacy HTML
	// sentinel. Empty HTML remains matchable through errors.Is.
	ErrEmptyContent = ErrEmptyHTML
	// ErrInvalidOrientation identifies an unsupported Document orientation.
	ErrInvalidOrientation = errors.New("gowkhtmltopdf: invalid orientation")
	// ErrInvalidImageQuality reports an image quality outside 0 to 100.
	ErrInvalidImageQuality = errors.New("gowkhtmltopdf: image quality must be between 0 and 100")
	// ErrInvalidCrop reports negative crop dimensions or offsets.
	ErrInvalidCrop = errors.New("gowkhtmltopdf: crop dimensions and offsets must be non-negative")
)

// Validate checks that Content identifies one valid source and that Base is
// used only with in-memory HTML.
func (c Content) Validate() error {
	return c.validate()
}

// Validate checks the document tree without opening files, resolving URLs, or
// starting the renderer.
func (d *Document) Validate() error {
	if d == nil {
		return ErrNilDocument
	}

	if err := validatePDFOptions(d); err != nil {
		return err
	}

	if d.Cover != nil {
		if err := validatePage("cover", *d.Cover); err != nil {
			return err
		}
	}

	for index, page := range d.Pages {
		if err := validatePage(fmt.Sprintf("pages[%d]", index), page); err != nil {
			return err
		}
	}

	if d.Cover == nil && len(d.Pages) == 0 {
		return ErrNoRenderablePDFObjects
	}

	return nil
}

func (d *ImageDocument) Validate() error {
	if d == nil {
		return ErrNilImageDocument
	}

	if err := d.Source.validate(); err != nil {
		return fmt.Errorf("source: %w", err)
	}

	switch format := strings.ToLower(strings.TrimSpace(d.Format)); format {
	case "", "png", "jpg", "jpeg":
	default:
		return fmt.Errorf("%w: %q", ErrInvalidImageFormat, d.Format)
	}

	if d.Quality < 0 || d.Quality > 100 {
		return fmt.Errorf("%w: got %d", ErrInvalidImageQuality, d.Quality)
	}

	return validateImageCrop(d.Crop)
}

func validateImageCrop(crop *Crop) error {
	if crop == nil {
		return nil
	}

	if crop.Width < 0 || crop.Height < 0 || crop.Left < 0 || crop.Top < 0 {
		return ErrInvalidCrop
	}

	return nil
}

func validatePDFOptions(document *Document) error {
	if pageSize := strings.TrimSpace(document.PageSize); pageSize != "" {
		if _, _, err := settings.ParsePageSize(pageSize); err != nil {
			return fmt.Errorf("%w: %q", ErrInvalidPageSize, document.PageSize)
		}
	}

	if orientation := strings.TrimSpace(document.Orientation); orientation != "" {
		if _, err := settings.ParseOrientation(orientation); err != nil {
			return fmt.Errorf("%w: %q", ErrInvalidOrientation, document.Orientation)
		}
	}

	if document.PDFVersion != "" {
		if _, err := settings.ParsePDFVersion(document.PDFVersion); err != nil {
			return fmt.Errorf("pdf version: %w", err)
		}
	}

	if document.PDFProfile != "" {
		if _, err := settings.ParsePDFProfile(document.PDFProfile); err != nil {
			return fmt.Errorf("pdf profile: %w", err)
		}
	}

	return validateDocumentCopies(document.Copies)
}

func validateDocumentCopies(copies int) error {
	// Zero means "use the engine default"; positive values must fit the
	// convert maxConversionCopies ceiling.
	if copies < 0 || copies > MaxDocumentCopies {
		return fmt.Errorf("%w: got %d", ErrInvalidPDFCopies, copies)
	}

	return nil
}

func validatePage(name string, page Page) error {
	if err := page.Source.validate(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	return nil
}

//nolint:cyclop,wsl // exact-one-source validation has one branch per source kind.
func (c Content) validate() error {
	sources := 0
	if c.HTML != nil {
		sources++
	}
	if strings.TrimSpace(c.File) != "" {
		sources++
	}
	if strings.TrimSpace(c.URL) != "" {
		sources++
	}

	switch {
	case sources == 0:
		return fmt.Errorf("%w: %w", ErrInvalidContent, ErrEmptyHTML)
	case sources > 1:
		return fmt.Errorf("%w: exactly one of HTML, File, or URL is required", ErrInvalidContent)
	}

	if c.HTML != nil {
		if len(c.HTML) == 0 {
			return fmt.Errorf("%w: %w", ErrInvalidContent, ErrEmptyHTML)
		}

		return nil
	}

	if c.Base != "" {
		return fmt.Errorf("%w: Base is only valid with HTML", ErrInvalidContent)
	}
	if c.URL == "" {
		return nil
	}

	parsed, err := url.Parse(c.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%w: URL must be an absolute HTTP(S) URL", ErrInvalidContent)
	}

	return nil
}
