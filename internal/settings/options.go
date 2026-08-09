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
	o.global.Size.PageSize = pageSize

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

// Build returns an independent settings snapshot.
func (o PdfGlobalOptions) Build() PdfGlobal {
	result := o.global
	result.ExcludeFromOutline = append([]string(nil), o.global.ExcludeFromOutline...)
	result.FontPaths = append([]string(nil), o.global.FontPaths...)
	result.Ignored = cloneStringMap(o.global.Ignored)

	return result
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
