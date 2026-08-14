package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

func xmlEscape(text string) string {
	var buf strings.Builder

	for _, rVal := range text {
		switch rVal {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(rVal)
		}
	}

	return buf.String()
}

//nolint:cyclop,funlen // XMP metadata template assembly with schema definitions
func (d *Document) buildXMPMetadata() []byte {
	now := d.creationTime
	if now.IsZero() {
		now = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	dateStr := now.UTC().Format("2006-01-02T15:04:05Z")

	var buf bytes.Buffer

	buf.WriteString("<?xpacket begin=\"\xef\xbb\xbf\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>\n")
	buf.WriteString("<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n")
	buf.WriteString(" <rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n")
	buf.WriteString("  <rdf:Description rdf:about=\"\"\n")
	buf.WriteString("    xmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	buf.WriteString("    xmlns:pdf=\"http://ns.adobe.com/pdf/1.3/\"\n")
	buf.WriteString("    xmlns:xmp=\"http://ns.adobe.com/xap/1.0/\"")

	if d.policy.IsPDFA3() {
		buf.WriteString("\n    xmlns:pdfaid=\"http://www.aiim.org/pdfa/ns/id/\"")
	}

	if d.policy.IsPDFUA1() {
		buf.WriteString("\n    xmlns:pdfuaid=\"http://www.aiim.org/pdfua/ns/id/\"")
	}

	if d.policy.IsPDFA3() && d.policy.IsPDFUA1() {
		buf.WriteString("\n    xmlns:pdfaExtension=\"http://www.aiim.org/pdfa/ns/extension/\"")
		buf.WriteString("\n    xmlns:pdfaSchema=\"http://www.aiim.org/pdfa/ns/schema#\"")
		buf.WriteString("\n    xmlns:pdfaProperty=\"http://www.aiim.org/pdfa/ns/property#\"")
	}

	buf.WriteString(">\n")
	buf.WriteString("   <dc:format>application/pdf</dc:format>\n")
	fmt.Fprintf(&buf, "   <pdf:Producer>%s</pdf:Producer>\n", xmlEscape(d.policy.ProducerVersion()))
	fmt.Fprintf(&buf, "   <xmp:CreateDate>%s</xmp:CreateDate>\n", dateStr)
	fmt.Fprintf(&buf, "   <xmp:ModifyDate>%s</xmp:ModifyDate>\n", dateStr)

	if title, ok := d.info["Title"]; ok && title != "" {
		buf.WriteString("   <dc:title>\n")
		buf.WriteString("    <rdf:Alt>\n")
		fmt.Fprintf(&buf, "     <rdf:li xml:lang=\"x-default\">%s</rdf:li>\n", xmlEscape(title))
		buf.WriteString("    </rdf:Alt>\n")
		buf.WriteString("   </dc:title>\n")
	}

	if d.policy.IsPDFA3() {
		buf.WriteString("   <pdfaid:part>3</pdfaid:part>\n")
		buf.WriteString("   <pdfaid:conformance>A</pdfaid:conformance>\n")
	}

	if d.policy.IsPDFUA1() {
		buf.WriteString("   <pdfuaid:part>1</pdfuaid:part>\n")
	}

	if d.policy.IsPDFA3() && d.policy.IsPDFUA1() {
		buf.WriteString("   <pdfaExtension:schemas>\n")
		buf.WriteString("    <rdf:Bag>\n")
		buf.WriteString("     <rdf:li rdf:parseType=\"Resource\">\n")
		buf.WriteString("      <pdfaSchema:schema>PDF/UA Identification Schema</pdfaSchema:schema>\n")
		buf.WriteString("      <pdfaSchema:namespaceURI>http://www.aiim.org/pdfua/ns/id/</pdfaSchema:namespaceURI>\n")
		buf.WriteString("      <pdfaSchema:prefix>pdfuaid</pdfaSchema:prefix>\n")
		buf.WriteString("      <pdfaSchema:property>\n")
		buf.WriteString("       <rdf:Seq>\n")
		buf.WriteString("        <rdf:li rdf:parseType=\"Resource\">\n")
		buf.WriteString("         <pdfaProperty:name>part</pdfaProperty:name>\n")
		buf.WriteString("         <pdfaProperty:valueType>Integer</pdfaProperty:valueType>\n")
		buf.WriteString("         <pdfaProperty:category>internal</pdfaProperty:category>\n")
		buf.WriteString("         <pdfaProperty:description>" +
			"Indicates, which part of ISO 14289 standard is followed" +
			"</pdfaProperty:description>\n")
		buf.WriteString("        </rdf:li>\n")
		buf.WriteString("       </rdf:Seq>\n")
		buf.WriteString("      </pdfaSchema:property>\n")
		buf.WriteString("     </rdf:li>\n")
		buf.WriteString("    </rdf:Bag>\n")
		buf.WriteString("   </pdfaExtension:schemas>\n")
	}

	buf.WriteString("  </rdf:Description>\n")
	buf.WriteString(" </rdf:RDF>\n")
	buf.WriteString("</x:xmpmeta>\n")
	buf.WriteString("<?xpacket end=\"w\"?>")

	return buf.Bytes()
}
