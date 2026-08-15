package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

func xmlEscape(text string) string {
	if !strings.ContainsAny(text, "&<>\"'") {
		return text
	}

	const xmlSlack = 8

	var buf strings.Builder

	buf.Grow(len(text) + xmlSlack)

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

	const xmpEstSize = 1024

	var buf bytes.Buffer

	buf.Grow(xmpEstSize)

	buf.WriteString("<?xpacket begin=\"\xef\xbb\xbf\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>\n")
	buf.WriteString("<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n")
	buf.WriteString(" <rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n")
	buf.WriteString("  <rdf:Description rdf:about=\"\"\n")
	buf.WriteString("    xmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	buf.WriteString("    xmlns:pdf=\"http://ns.adobe.com/pdf/1.3/\"\n")
	buf.WriteString("    xmlns:xmp=\"http://ns.adobe.com/xap/1.0/\"")

	hasPDFA := d.isPDFA3 || d.isPDFA4
	hasPDFUA := d.isUA1 || d.isUA2

	if hasPDFA {
		buf.WriteString("\n    xmlns:pdfaid=\"http://www.aiim.org/pdfa/ns/id/\"")
	}

	if hasPDFUA {
		buf.WriteString("\n    xmlns:pdfuaid=\"http://www.aiim.org/pdfua/ns/id/\"")
	}

	if hasPDFA && hasPDFUA {
		buf.WriteString("\n    xmlns:pdfaExtension=\"http://www.aiim.org/pdfa/ns/extension/\"")
		buf.WriteString("\n    xmlns:pdfaSchema=\"http://www.aiim.org/pdfa/ns/schema#\"")
		buf.WriteString("\n    xmlns:pdfaProperty=\"http://www.aiim.org/pdfa/ns/property#\"")
	}

	buf.WriteString(">\n")
	buf.WriteString("   <dc:format>application/pdf</dc:format>\n")
	fmt.Fprintf(&buf, "   <pdf:Producer>%s</pdf:Producer>\n", xmlEscape(d.policy.ProducerVersion()))
	fmt.Fprintf(&buf, "   <xmp:CreateDate>%s</xmp:CreateDate>\n", dateStr)
	fmt.Fprintf(&buf, "   <xmp:ModifyDate>%s</xmp:ModifyDate>\n", dateStr)
	fmt.Fprintf(&buf, "   <xmp:MetadataDate>%s</xmp:MetadataDate>\n", dateStr)

	if creator, ok := d.info["Creator"]; ok && creator != "" {
		fmt.Fprintf(&buf, "   <xmp:CreatorTool>%s</xmp:CreatorTool>\n", xmlEscape(creator))
	}

	if title, ok := d.info["Title"]; ok && title != "" {
		buf.WriteString("   <dc:title>\n")
		buf.WriteString("    <rdf:Alt>\n")
		fmt.Fprintf(&buf, "     <rdf:li xml:lang=\"x-default\">%s</rdf:li>\n", xmlEscape(title))
		buf.WriteString("    </rdf:Alt>\n")
		buf.WriteString("   </dc:title>\n")
	}

	if author, ok := d.info["Author"]; ok && author != "" {
		buf.WriteString("   <dc:creator>\n")
		buf.WriteString("    <rdf:Seq>\n")
		fmt.Fprintf(&buf, "     <rdf:li>%s</rdf:li>\n", xmlEscape(author))
		buf.WriteString("    </rdf:Seq>\n")
		buf.WriteString("   </dc:creator>\n")
	}

	if subject, ok := d.info["Subject"]; ok && subject != "" {
		buf.WriteString("   <dc:description>\n")
		buf.WriteString("    <rdf:Alt>\n")
		fmt.Fprintf(&buf, "     <rdf:li xml:lang=\"x-default\">%s</rdf:li>\n", xmlEscape(subject))
		buf.WriteString("    </rdf:Alt>\n")
		buf.WriteString("   </dc:description>\n")
	}

	if keywords, ok := d.info["Keywords"]; ok && keywords != "" {
		fmt.Fprintf(&buf, "   <pdf:Keywords>%s</pdf:Keywords>\n", xmlEscape(keywords))
	}

	if d.isPDFA3 {
		buf.WriteString("   <pdfaid:part>3</pdfaid:part>\n")
		buf.WriteString("   <pdfaid:conformance>A</pdfaid:conformance>\n")
	} else if d.isPDFA4 {
		buf.WriteString("   <pdfaid:part>4</pdfaid:part>\n")
		buf.WriteString("   <pdfaid:rev>2020</pdfaid:rev>\n")
	}

	if d.isUA1 {
		buf.WriteString("   <pdfuaid:part>1</pdfuaid:part>\n")
	} else if d.isUA2 {
		buf.WriteString("   <pdfuaid:part>2</pdfuaid:part>\n")
		buf.WriteString("   <pdfuaid:rev>2024</pdfuaid:rev>\n")
	}

	if hasPDFA && hasPDFUA {
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

		if d.isUA2 {
			buf.WriteString("        <rdf:li rdf:parseType=\"Resource\">\n")
			buf.WriteString("         <pdfaProperty:name>rev</pdfaProperty:name>\n")
			buf.WriteString("         <pdfaProperty:valueType>Integer</pdfaProperty:valueType>\n")
			buf.WriteString("         <pdfaProperty:category>internal</pdfaProperty:category>\n")
			buf.WriteString("         <pdfaProperty:description>" +
				"Indicates, which revision of ISO 14289 standard is followed" +
				"</pdfaProperty:description>\n")
			buf.WriteString("        </rdf:li>\n")
		}

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
