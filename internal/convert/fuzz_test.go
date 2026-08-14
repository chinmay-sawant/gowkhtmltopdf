package convert_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/settings"
)

func FuzzConvertHTML(f *testing.F) {
	seeds := []string{
		"<html><body><h1>Title</h1><p>Body</p></body></html>",
		"<html><body><div style=\"width:100px;height:50px;background:#f00;\"></div></body></html>",
		"<html><head><style>table, td { border: 1px solid black; }</style></head>" +
			"<body><table><tr><td>cell</td></tr></table></body></html>",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, htmlSrc string) {
		if len(htmlSrc) > 8<<10 {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()

		global := settings.DefaultPdfGlobal()
		obj := settings.PdfObject{} //nolint:exhaustruct // intentional zero-value fields
		obj.Load.InlineHTML = []byte(htmlSrc)
		objects := []settings.PdfObject{obj}

		var out bytes.Buffer
		req := &convert.PDFRequest{
			Global:        global,
			Objects:       objects,
			Now:           nil,
			Output:        &out,
			OutlineOutput: nil,
		}

		_ = convert.RunTypedPDF(ctx, req, io.Discard, nil)
	})
}
