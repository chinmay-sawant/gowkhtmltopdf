package settings

// PdfGlobalOptions is the typed library-side builder for PDF global settings.
// The string-based Set method remains available for CLI compatibility; this
// builder keeps common library configuration compile-time discoverable.
type PdfGlobalOptions struct {
	global PdfGlobal
}

// NewPdfGlobalOptions returns a builder initialized with engine defaults.
func NewPdfGlobalOptions() PdfGlobalOptions {
	return PdfGlobalOptions{global: DefaultPdfGlobal()}
}

func (o PdfGlobalOptions) WithPageSize(pageSize string) PdfGlobalOptions {
	o.global.PageSize = pageSize

	return o
}

func (o PdfGlobalOptions) WithMargins(top, right, bottom, left float64) PdfGlobalOptions {
	o.global.Margin = Margin{Top: top, Right: right, Bottom: bottom, Left: left}

	return o
}

func (o PdfGlobalOptions) WithTitle(title string) PdfGlobalOptions {
	o.global.Title = title

	return o
}

func (o PdfGlobalOptions) WithCopies(copies int, collate bool) PdfGlobalOptions {
	o.global.Copies = copies
	o.global.Collate = collate

	return o
}

func (o PdfGlobalOptions) WithOutline(enabled bool, depth int) PdfGlobalOptions {
	o.global.Outline = enabled
	o.global.OutlineDepth = depth

	return o
}

func (o PdfGlobalOptions) WithSmartShrinking(enabled bool) PdfGlobalOptions {
	o.global.SmartShrinking = enabled

	return o
}

func (o PdfGlobalOptions) WithBackground(enabled bool) PdfGlobalOptions {
	o.global.Background = enabled

	return o
}

func (o PdfGlobalOptions) WithCompression(enabled bool) PdfGlobalOptions {
	o.global.UseCompression = enabled

	return o
}

func (o PdfGlobalOptions) WithResolveRelativeLinks(enabled bool) PdfGlobalOptions {
	o.global.ResolveRelativeLinks = enabled

	return o
}

func (o PdfGlobalOptions) WithPDFVersion(version string) PdfGlobalOptions {
	if parsed, err := ParsePDFVersion(version); err == nil {
		o.global.PdfVersion = parsed
	} else {
		o.global.PdfVersion = version
	}

	return o
}

func (o PdfGlobalOptions) WithPDFProfile(profile string) PdfGlobalOptions {
	if parsed, err := ParsePDFProfile(profile); err == nil {
		o.global.PdfProfile = parsed
	} else {
		o.global.PdfProfile = profile
	}

	return o
}

// WithSetting applies any supported dotted setting while retaining the value
// builder's independent-snapshot semantics. Common settings should use the
// typed With* methods; this escape hatch keeps the full wkhtmltopdf-compatible
// key surface available without requiring a second builder type.
func (o PdfGlobalOptions) WithSetting(name, value string) (PdfGlobalOptions, error) {
	if err := o.global.Set(name, value); err != nil {
		return o, err
	}

	return o, nil
}

// Build returns an independent settings snapshot. Slices and maps on the
// builder (Allow, Network*, Header/Footer.Replace, FontPaths, Ignored, …)
// are cloned so later WithSetting / field mutations do not change the result.
func (o PdfGlobalOptions) Build() PdfGlobal {
	return ClonePdfGlobal(o.global)
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source)) //nolint:wsl

	for key, value := range source {
		clone[key] = value
	}

	return clone
}
