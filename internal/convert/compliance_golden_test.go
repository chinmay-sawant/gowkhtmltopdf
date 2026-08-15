package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const complianceFooterPage = "Page [page]"

//nolint:cyclop,funlen // comprehensive golden needle assertions for PDF/A-3a + PDF/UA-1 dual profile
func TestConvertDualProfileGoldenNeedles(t *testing.T) {
	t.Parallel()

	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Compliance Document — 2026</title>
<style>
  table { border-collapse: collapse; width: 100%; }
  th, td { border: 1px solid #ccc; padding: 4px; }
</style>
</head>
<body>
<h1>Chapter 1: Compliance Overview</h1>
<p>This document demonstrates PDF 1.7 archival and accessibility compliance.</p>
<table>
  <tr><th>Item</th><th>Status</th></tr>
  <tr><td>PDF/A-3a</td><td>Pass</td></tr>
  <tr><td>PDF/UA-1</td><td>Pass</td></tr>
</table>
<p><a href="https://example.com/spec">Specification Link</a></p>
</body>
</html>`

	cmd, _ := newCommand(t, htmlContent, "")
	cmd.Global.PdfProfile = settings.ProfilePDFA3aPDFUA1
	cmd.Global.Title = "Compliance Document — 2026"
	cmd.Global.Header.Left = "Header [page]"
	cmd.Global.Footer.Right = complianceFooterPage
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	str := string(data)

	// 1. Header is %PDF-1.7
	if !bytes.HasPrefix(data, []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")) {
		t.Errorf("expected %%PDF-1.7 header, got %q", data[:min(25, len(data))])
	}

	// 2. Trailer /ID is present
	idRe := regexp.MustCompile(`/ID\s*\[\s*<([0-9A-Fa-f]{32})>\s*<([0-9A-Fa-f]{32})>\s*\]`)
	if !idRe.MatchString(str) {
		t.Errorf("dual profile output missing trailer /ID: %s", str)
	}

	// 3. Claiming XMP Metadata stream contains pdfaid and pdfuaid
	if !strings.Contains(str, "<pdfaid:part>3</pdfaid:part>") {
		t.Errorf("XMP metadata missing <pdfaid:part>3</pdfaid:part>")
	}

	if !strings.Contains(str, "<pdfaid:conformance>A</pdfaid:conformance>") {
		t.Errorf("XMP metadata missing <pdfaid:conformance>A</pdfaid:conformance>")
	}

	if !strings.Contains(str, "<pdfuaid:part>1</pdfuaid:part>") {
		t.Errorf("XMP metadata missing <pdfuaid:part>1</pdfuaid:part>")
	}

	if !strings.Contains(str, "pdfaExtension:schemas") {
		t.Errorf("XMP metadata missing pdfaExtension:schemas for pdfuaid")
	}

	// 4. OutputIntent and ICC stream present
	if !strings.Contains(str, "/Type /OutputIntent") {
		t.Errorf("catalog missing /OutputIntent")
	}

	if !strings.Contains(str, "/S /GTS_PDFA1") {
		t.Errorf("OutputIntent missing /S /GTS_PDFA1")
	}

	if !strings.Contains(str, "/OutputConditionIdentifier (sRGB IEC61966-2.1)") {
		t.Errorf("OutputIntent missing sRGB condition identifier")
	}

	// 5. Structure tree keys present
	if !strings.Contains(str, "/MarkInfo << /Marked true >>") {
		t.Errorf("catalog missing /MarkInfo << /Marked true >>")
	}

	if !strings.Contains(str, "/ViewerPreferences << /DisplayDocTitle true >>") {
		t.Errorf("catalog missing /ViewerPreferences << /DisplayDocTitle true >>")
	}

	if !strings.Contains(str, "/Lang (en)") {
		t.Errorf("catalog missing /Lang (en)")
	}

	if !strings.Contains(str, "/Type /StructTreeRoot") {
		t.Errorf("catalog missing /StructTreeRoot")
	}

	// 6. Standard Structure Elements present
	for _, elem := range []string{"/S /H1", "/S /P", "/S /Table", "/S /TR", "/S /TH", "/S /TD", "/S /Link"} {
		if !strings.Contains(str, elem) {
			t.Errorf("missing expected structure element %s in tagged tree", elem)
		}
	}

	// 7. Marked content BDC / EMC in content streams
	if !strings.Contains(str, "/MCID") || !strings.Contains(str, "BDC") || !strings.Contains(str, "EMC") {
		t.Errorf("content stream missing marked content operators BDC/EMC")
	}

	// 8. Header/Footer marked as Artifact
	if !strings.Contains(str, "/Artifact << /Type /Pagination >> BDC") {
		t.Errorf("header/footer missing /Artifact pagination marking")
	}

	// 9. Negative assertions: no PDF 2.0 / A-4 / UA-2 claims
	if strings.Contains(str, "<pdfaid:part>4</pdfaid:part>") {
		t.Errorf("forbidden claim: PDF/A-4 part 4 found")
	}

	if strings.Contains(str, "<pdfuaid:part>2</pdfuaid:part>") {
		t.Errorf("forbidden claim: PDF/UA-2 part 2 found")
	}

	// 10. Semantic Parse
	sem, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if sem.Version != pdfVersion17 {
		t.Errorf("sem.Version = %q, want 1.7", sem.Version)
	}
}

//nolint:cyclop,funlen // comprehensive golden needle assertions for PDF/A-4 + PDF/UA-2 dual profile
func TestConvertDualPDFA4PDFUA2GoldenNeedles(t *testing.T) {
	t.Parallel()

	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>PDF 2.0 Dual Compliance Document</title>
<style>
  table { border-collapse: collapse; width: 100%; }
  th, td { border: 1px solid #ccc; padding: 4px; }
</style>
</head>
<body>
<h1>Chapter 1: PDF 2.0 Compliance Overview</h1>
<p>This document demonstrates PDF/A-4 and PDF/UA-2 compliance.</p>
<table>
  <tr><th>Profile</th><th>Status</th></tr>
  <tr><td>PDF/A-4</td><td>Pass</td></tr>
  <tr><td>PDF/UA-2</td><td>Pass</td></tr>
</table>
<p><a href="https://example.com/spec2">PDF 2.0 Specification Link</a></p>
</body>
</html>`

	cmd, _ := newCommand(t, htmlContent, "")
	cmd.Global.PdfProfile = settings.ProfilePDFA4PDFUA2
	cmd.Global.Title = "PDF 2.0 Dual Compliance Document"
	cmd.Global.Header.Left = "Header [page]"
	cmd.Global.Footer.Right = complianceFooterPage
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	str := string(data)

	// 1. Header is %PDF-2.0
	if !bytes.HasPrefix(data, []byte("%PDF-2.0\n%\xe2\xe3\xcf\xd3\n")) {
		t.Errorf("expected %%PDF-2.0 header, got %q", data[:min(25, len(data))])
	}

	// 2. Trailer /ID is present and /Info is omitted
	idRe := regexp.MustCompile(`/ID\s*\[\s*<([0-9A-Fa-f]{32})>\s*<([0-9A-Fa-f]{32})>\s*\]`)
	if !idRe.MatchString(str) {
		t.Errorf("dual 2.0 profile output missing trailer /ID: %s", str)
	}

	trailerIdx := strings.LastIndex(str, "trailer\n")
	if trailerIdx >= 0 {
		trailerPart := str[trailerIdx:]
		if strings.Contains(trailerPart, "/Info ") {
			t.Errorf("PDF/A-4 trailer must omit /Info, found in: %s", trailerPart)
		}
	}

	// 3. Claiming XMP Metadata stream contains pdfaid and pdfuaid
	if !strings.Contains(str, "<pdfaid:part>4</pdfaid:part>") {
		t.Errorf("XMP metadata missing <pdfaid:part>4</pdfaid:part>")
	}

	if !strings.Contains(str, "<pdfaid:rev>2020</pdfaid:rev>") {
		t.Errorf("XMP metadata missing <pdfaid:rev>2020</pdfaid:rev>")
	}

	if strings.Contains(str, "<pdfaid:conformance>") {
		t.Errorf("PDF/A-4 should not contain <pdfaid:conformance>")
	}

	if !strings.Contains(str, "<pdfuaid:part>2</pdfuaid:part>") {
		t.Errorf("XMP metadata missing <pdfuaid:part>2</pdfuaid:part>")
	}

	if !strings.Contains(str, "<pdfuaid:rev>2024</pdfuaid:rev>") {
		t.Errorf("XMP metadata missing <pdfuaid:rev>2024</pdfuaid:rev>")
	}

	if !strings.Contains(str, "pdfaExtension:schemas") {
		t.Errorf("XMP metadata missing pdfaExtension:schemas for pdfuaid")
	}

	// 4. OutputIntent and ICC stream present (sRGB and Gray)
	if !strings.Contains(str, "/Type /OutputIntent") {
		t.Errorf("catalog missing /OutputIntent")
	}

	if !strings.Contains(str, "/S /GTS_PDFA1") {
		t.Errorf("OutputIntent missing /S /GTS_PDFA1")
	}

	if !strings.Contains(str, "/OutputConditionIdentifier (sRGB IEC61966-2.1)") {
		t.Errorf("OutputIntent missing sRGB condition identifier")
	}

	if !strings.Contains(str, "/DefaultRGB [/ICCBased ") {
		t.Errorf("Page resources missing DefaultRGB")
	}

	if !strings.Contains(str, "/DefaultGray [/ICCBased ") {
		t.Errorf("Page resources missing DefaultGray")
	}

	// 5. Structure tree keys and namespace object present
	if !strings.Contains(str, "/MarkInfo << /Marked true >>") {
		t.Errorf("catalog missing /MarkInfo << /Marked true >>")
	}

	if !strings.Contains(str, "/ViewerPreferences << /DisplayDocTitle true >>") {
		t.Errorf("catalog missing /ViewerPreferences << /DisplayDocTitle true >>")
	}

	if !strings.Contains(str, "/Lang (en)") {
		t.Errorf("catalog missing /Lang (en)")
	}

	if !strings.Contains(str, "/Type /Namespace") || !strings.Contains(str, "/NS (http://iso.org/pdf2/ssn)") {
		t.Errorf("missing PDF 2.0 /Namespace object in output")
	}

	if !strings.Contains(str, "/Namespaces [") {
		t.Errorf("StructTreeRoot missing /Namespaces array")
	}

	// 6. Standard Structure Elements present
	for _, elem := range []string{"/S /H1", "/S /P", "/S /Table", "/S /TR", "/S /TH", "/S /TD", "/S /Link"} {
		if !strings.Contains(str, elem) {
			t.Errorf("missing expected structure element %s in tagged tree", elem)
		}
	}

	// 7. Semantic Parse
	sem, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if sem.Version != pdfVersion20 {
		t.Errorf("sem.Version = %q, want 2.0", sem.Version)
	}
}

//nolint:lll // positive and negative figure alt assertions
func TestComplianceFigureAltAndMissingAlt(t *testing.T) {
	t.Parallel()

	// 1. Positive: image with alt text succeeds and produces /Figure + /Alt
	dataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	htmlWithAlt := `<!DOCTYPE html>
<html lang="en">
<head><title>Figure Test</title></head>
<body>
<h1>Figure Section</h1>
<p>Below is a diagram:</p>
<img src="` + dataURI + `" alt="Sample Diagram">
</body>
</html>`

	cmdPass, _ := newCommand(t, htmlWithAlt, "")
	cmdPass.Global.PdfProfile = settings.ProfilePDFA3aPDFUA1
	cmdPass.Global.Title = "Figure Test"
	cmdPass.Global.UseCompression = false

	dataPass := runPDF(t, cmdPass)
	strPass := string(dataPass)

	if !strings.Contains(strPass, "/S /Figure") {
		t.Error("expected /S /Figure structure element for <img> with alt")
	}

	if !strings.Contains(strPass, "/Alt (Sample Diagram)") {
		t.Error("expected /Alt (Sample Diagram) in Figure structure element")
	}

	// 2. Negative: image without alt text fails closed with ErrPDFUAMissingAlt
	htmlNoAlt := `<!DOCTYPE html>
<html lang="en">
<head><title>Figure Missing Alt Test</title></head>
<body>
<h1>Figure Section</h1>
<img src="` + dataURI + `">
</body>
</html>`

	cmdFail, _ := newCommand(t, htmlNoAlt, "")
	cmdFail.Global.PdfProfile = settings.ProfilePDFA3aPDFUA1
	cmdFail.Global.Title = "Figure Missing Alt Test"

	var buf bytes.Buffer
	cmdFail.Output = &buf

	err := Run(t.Context(), cmdFail, &buf, nil)
	if err == nil {
		t.Fatal("expected error for <img> without alt in PDF/UA-1 mode, got nil")
	}

	if !errors.Is(err, pdf.ErrPDFUAMissingAlt) && !strings.Contains(err.Error(), "alt") {
		t.Errorf("expected ErrPDFUAMissingAlt, got %v", err)
	}
}

func TestComplianceUnclaimedIsolation(t *testing.T) {
	t.Parallel()

	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Unclaimed 1.7</title></head>
<body><h1>Unclaimed</h1><p>Test paragraph.</p></body>
</html>`

	// Unclaimed PDF 1.7
	cmd17, _ := newCommand(t, htmlContent, "")
	cmd17.Global.PdfVersion = pdfVersion17
	cmd17.Global.Title = "Unclaimed 1.7"
	cmd17.Global.UseCompression = false

	data17 := runPDF(t, cmd17)
	str17 := string(data17)

	for _, token := range []string{"pdfaid", "pdfuaid", "pdfaExtension", "/StructTreeRoot", "/MarkInfo"} {
		if strings.Contains(str17, token) {
			t.Errorf("unclaimed 1.7 output contains compliance token %q", token)
		}
	}
}

func resolveVeraPDFBinary(t *testing.T) string {
	t.Helper()

	if bin := os.Getenv("VERAPDF_BIN"); bin != "" {
		return bin
	}

	if bin, err := exec.LookPath("verapdf"); err == nil {
		return bin
	}

	const treeBin = "../../verapdf/verapdf"
	if _, err := os.Stat(treeBin); err == nil {
		return treeBin
	}

	t.Skip("optional validator verapdf not installed (set VERAPDF_BIN or PATH to enable)")

	return ""
}

func runVeraPDFFlavour(t *testing.T, verapdfBin, flavour, pdfPath string) {
	t.Helper()

	out, err := exec.CommandContext(
		t.Context(), verapdfBin, "-f", flavour, "--format", "text", pdfPath,
	).CombinedOutput()
	if err != nil {
		t.Errorf("verapdf -f %s validation failed: %v\nOutput: %s", flavour, err, string(out))

		return
	}

	t.Logf("verapdf -f %s outcome: %s", flavour, strings.TrimSpace(string(out)))
}

func TestVeraPDFOptionalValidation(t *testing.T) {
	t.Parallel()

	verapdfBin := resolveVeraPDFBinary(t)
	if verOut, err := exec.CommandContext(t.Context(), verapdfBin, "--version").CombinedOutput(); err == nil {
		t.Logf("running with verapdf version: %s", strings.TrimSpace(string(verOut)))
	}

	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>veraPDF Test Document</title>
</head>
<body>
<h1>veraPDF Compliance Test</h1>
<p>Testing PDF/A and PDF/UA profile compliance.</p>
</body>
</html>`

	// 1. Validate PDF 1.7 (PDF/A-3a and PDF/UA-1)
	cmd17, _ := newCommand(t, htmlContent, "")
	cmd17.Global.PdfProfile = settings.ProfilePDFA3aPDFUA1
	cmd17.Global.Title = "veraPDF Test Document"
	data17 := runPDF(t, cmd17)

	pdfFile17 := filepath.Join(t.TempDir(), "verapdf_test_17.pdf")
	if err := os.WriteFile(pdfFile17, data17, 0o600); err != nil {
		t.Fatalf("write PDF 1.7 file: %v", err)
	}

	runVeraPDFFlavour(t, verapdfBin, "3a", pdfFile17)
	runVeraPDFFlavour(t, verapdfBin, "ua1", pdfFile17)

	// 2. Validate PDF 2.0 (PDF/A-4 and PDF/UA-2)
	cmd20, _ := newCommand(t, htmlContent, "")
	cmd20.Global.PdfProfile = settings.ProfilePDFA4PDFUA2
	cmd20.Global.Title = "veraPDF Test Document"
	data20 := runPDF(t, cmd20)

	pdfFile20 := filepath.Join(t.TempDir(), "verapdf_test_20.pdf")
	if err := os.WriteFile(pdfFile20, data20, 0o600); err != nil {
		t.Fatalf("write PDF 2.0 file: %v", err)
	}

	runVeraPDFFlavour(t, verapdfBin, "4", pdfFile20)
	runVeraPDFFlavour(t, verapdfBin, "ua2", pdfFile20)
}

// TestPDFUA1ContentMarkedCompleteness validates that in PDF/UA-1 mode, 100% of all visual
// marking operations in all content streams are properly enclosed in marked content sequences
// or artifact sequences per ISO 14289-1 and Matterhorn Protocol 01-005.
//
//nolint:cyclop,gocyclo,funlen,gocognit,wsl,lll,nlreturn
func TestPDFUA1ContentMarkedCompleteness(t *testing.T) {
	t.Parallel()

	dataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	htmlDoc := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Completeness Test</title>
<style>
  body { font-family: sans-serif; }
  h1 { background-color: #f0f0f0; border-bottom: 2px solid #333; }
  p { margin: 8px 0; }
  table { border-collapse: collapse; width: 100%; background-color: #fafafa; }
  th, td { border: 1px solid #aaa; padding: 6px; }
  hr { border: 0; height: 1px; background: #666; }
  ul { padding-left: 20px; }
</style>
</head>
<body>
<h1>Heading 1 with Background & Border</h1>
<p>Paragraph with an <a href="https://example.com">inline link</a>.</p>
<hr>
<table>
  <tr><th>Header Col 1</th><th>Header Col 2</th></tr>
  <tr><td>Cell 1</td><td>Cell 2</td></tr>
</table>
<ul>
  <li>List Item 1</li>
  <li>List Item 2</li>
</ul>
<img src="` + dataURI + `" alt="Test image diagram">
</body>
</html>`

	cmd, _ := newCommand(t, htmlDoc, "")
	cmd.Global.PdfProfile = settings.ProfilePDFUA1
	cmd.Global.Title = "Completeness Test"
	cmd.Global.Header.Left = "Header Text"
	cmd.Global.Header.Line = true
	cmd.Global.Footer.Right = complianceFooterPage
	cmd.Global.Footer.Line = true
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	str := string(data)

	// Verify required structure tags and artifacts exist
	for _, expected := range []string{
		"/Artifact << /Type /Pagination >> BDC",
		"/Artifact << /Type /Background >> BDC",
		"/Artifact << /Type /Layout >> BDC",
		"/MCID",
		"/S /H1",
		"/S /P",
		"/S /Table",
		"/S /TR",
		"/S /TH",
		"/S /TD",
		"/S /L",
		"/S /LI",
		"/S /Figure",
		"/S /Link",
	} {
		if !strings.Contains(str, expected) {
			t.Errorf("missing expected structure or artifact token %q", expected)
		}
	}

	// Verify that all content stream painting operations are enclosed in BDC...EMC
	streamRe := regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	matches := streamRe.FindAllStringSubmatch(str, -1)
	if len(matches) == 0 {
		t.Fatal("no streams found in uncompressed PDF")
	}

	checkedStreams := 0
	for _, m := range matches {
		streamContent := m[1]
		// Skip non-content streams (ICC profiles, XMP metadata, embedded fonts, image streams)
		if strings.Contains(streamContent, "<x:xmpmeta") ||
			strings.Contains(streamContent, "FontFile") ||
			strings.Contains(streamContent, "ICCBased") ||
			len(streamContent) == 0 {
			continue
		}

		lines := strings.Split(streamContent, "\n")
		markedDepth := 0
		hasDrawingOp := false

		for lineIdx, rawLine := range lines {
			trimmed := strings.TrimSpace(rawLine)
			if trimmed == "" {
				continue
			}

			if strings.HasSuffix(trimmed, "BDC") || strings.HasSuffix(trimmed, "BMC") {
				markedDepth++
				continue
			}
			if trimmed == "EMC" {
				if markedDepth <= 0 {
					t.Errorf("unmatched EMC at line %d in stream: %q", lineIdx, trimmed)
				} else {
					markedDepth--
				}
				continue
			}

			// Check visual drawing operators
			isVisualOp := trimmed == "f" || trimmed == "f*" || trimmed == "F" ||
				trimmed == "S" || trimmed == "s" ||
				trimmed == "B" || trimmed == "B*" || trimmed == "b" || trimmed == "b*" ||
				trimmed == "BT" || strings.HasSuffix(trimmed, " Do") || trimmed == "sh"

			if isVisualOp {
				hasDrawingOp = true
				if markedDepth == 0 {
					t.Errorf("Matterhorn 01-005 violation: visual op %q at line %d outside marked content (depth=0)", trimmed, lineIdx)
				}
			}
		}

		if hasDrawingOp {
			checkedStreams++
			if markedDepth != 0 {
				t.Errorf("stream ended with unclosed marked content depth %d", markedDepth)
			}
		}
	}

	if checkedStreams == 0 {
		t.Errorf("expected at least one checked page content stream, got %d", checkedStreams)
	}
}
