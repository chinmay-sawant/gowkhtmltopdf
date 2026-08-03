package convert

import (
	"fmt"
	"io"

	"gowkhtmltopdf/internal/cli"
)

// ErrEngineNotReady is the explicit Phase 1 convert stub error; removed when
// the pipeline lands in Phase 5.
var ErrEngineNotReady = fmt.Errorf("conversion engine not implemented yet")

// RunPDF executes the full pdf conversion. Phase 1: parses settings, then
// fails with ErrEngineNotReady.
func RunPDF(cmd *cli.Command, log io.Writer) error {
	return ErrEngineNotReady
}

// DefaultTOCXSL returns the default TOC stylesheet. In pure Go the default
// TOC look is a built-in Go template; this returns a description of it for
// --dump-default-toc-xsl compatibility.
func DefaultTOCXSL() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!-- gowkhtmltopdf default TOC stylesheet.
     Upstream ships an XSLT here; the pure-Go implementation uses an
     equivalent built-in template (see internal/outline/toc.go). -->
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" indent="yes"/>
  <xsl:template match="/">
    <h1>Table of Contents</h1>
    <ul id="toc"/>
  </xsl:template>
</xsl:stylesheet>
`
}
