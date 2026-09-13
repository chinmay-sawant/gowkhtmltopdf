package settings

// StampCover marks obj as a cover page: excluded from the outline and with no
// inherited document headers/footers unless the caller sets HF afterward.
func StampCover(obj *PdfObject) {
	if obj == nil {
		return
	}

	obj.IsCover = true
	obj.IncludeInOutline = false
	obj.HeaderSet, obj.FooterSet = true, true

	var empty HeaderFooter
	obj.Header, obj.Footer = empty, empty
}

// StampTOC marks obj as a table-of-contents placeholder object.
func StampTOC(obj *PdfObject) {
	if obj == nil {
		return
	}

	obj.Page = ""
	obj.IsTableOfContent = true
	obj.UseOutline = false
	obj.IncludeInOutline = false
}
