package settings

// ClonePdfGlobal returns an independent snapshot of src. Slices and maps
// (Allow, Network*, FontPaths, ExcludeFromOutline, Header/Footer.Replace,
// Ignored) are copied so later mutations of src cannot affect the result.
func ClonePdfGlobal(src PdfGlobal) PdfGlobal {
	dst := src
	dst.Header = CloneHeaderFooter(src.Header)
	dst.Footer = CloneHeaderFooter(src.Footer)
	dst.Load.Allow = cloneStrings(src.Load.Allow)
	dst.Load.NetworkAllowedSchemes = cloneStrings(src.Load.NetworkAllowedSchemes)
	dst.Load.NetworkAllowedHosts = cloneStrings(src.Load.NetworkAllowedHosts)
	dst.ExcludeFromOutline = cloneStrings(src.ExcludeFromOutline)
	dst.FontPaths = cloneStrings(src.FontPaths)
	dst.Ignored = cloneStringMap(src.Ignored)

	return dst
}

// ClonePdfObject returns an independent snapshot of src, including Header/
// Footer.Replace, load maps, POST items, InlineHTML, and Ignored.
func ClonePdfObject(src PdfObject) PdfObject {
	dst := src
	dst.Header = CloneHeaderFooter(src.Header)
	dst.Footer = CloneHeaderFooter(src.Footer)
	dst.Load.CustomHeaders = cloneStringMap(src.Load.CustomHeaders)
	dst.Load.Cookies = cloneStringMap(src.Load.Cookies)
	dst.Load.Post = clonePostItems(src.Load.Post)
	dst.Load.InlineHTML = cloneBytes(src.Load.InlineHTML)
	dst.Ignored = cloneStringMap(src.Ignored)

	return dst
}

// CloneImageGlobal returns an independent snapshot of src, including Allow,
// Network* slices, and Ignored.
func CloneImageGlobal(src ImageGlobal) ImageGlobal {
	dst := src
	dst.Load.Allow = cloneStrings(src.Load.Allow)
	dst.Load.NetworkAllowedSchemes = cloneStrings(src.Load.NetworkAllowedSchemes)
	dst.Load.NetworkAllowedHosts = cloneStrings(src.Load.NetworkAllowedHosts)
	dst.Ignored = cloneStringMap(src.Ignored)

	return dst
}

// CloneHeaderFooter returns an independent snapshot of src, including Replace.
func CloneHeaderFooter(src HeaderFooter) HeaderFooter {
	dst := src
	dst.Replace = cloneStringMap(src.Replace)

	return dst
}

func clonePostItems(src []PostItem) []PostItem {
	if src == nil {
		return nil
	}

	dst := make([]PostItem, len(src))
	copy(dst, src)

	return dst
}

func cloneStrings(src []string) []string {
	if src == nil {
		return nil
	}

	dst := make([]string, len(src))
	copy(dst, src)

	return dst
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}
