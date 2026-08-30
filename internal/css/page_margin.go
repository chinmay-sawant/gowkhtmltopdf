package css

import "strings"

// PageMarginBoxes is the lite @page margin-box content map. Only quoted
// content strings are stored. counter(), running(), and other functions drop.
type PageMarginBoxes struct {
	TopLeft, TopCenter, TopRight          string
	BottomLeft, BottomCenter, BottomRight string
}

// parsePageBlock removes nested at-rules while collecting unnamed margin-box
// content. Named pages and page pseudos pass capture=false because convert's
// lite header/footer consumer repeats only the unnamed page chrome.
func parsePageBlock(block string, capture bool) (PageMarginBoxes, string) {
	var boxes PageMarginBoxes

	var declarations strings.Builder

	declarations.Grow(len(block))

	for idx := 0; idx < len(block); {
		char := block[idx]
		if char == '"' || char == '\'' {
			relEnd := strings.IndexByte(block[idx+1:], char)
			if relEnd < 0 {
				declarations.WriteString(block[idx:])

				break
			}

			end := idx + minQuotedLen + relEnd
			declarations.WriteString(block[idx:end])
			idx = end

			continue
		}

		if char != '@' {
			declarations.WriteByte(char)

			idx++

			continue
		}

		next, name, inner, ok := takeNestedAtRule(block, idx)
		if !ok {
			declarations.WriteString(block[idx:])

			break
		}

		if capture {
			content, contentOK := quotedPageBoxContent(inner)
			if contentOK {
				setPageMarginBox(&boxes, name, content)
			}
		}

		idx = next
	}

	return boxes, declarations.String()
}

func takeNestedAtRule(block string, atIndex int) (int, string, string, bool) {
	if atIndex >= len(block) || block[atIndex] != '@' {
		return atIndex, "", "", false
	}

	ident := pageIdent(block[atIndex+1:])
	if ident == "" {
		rest, err := skipAtRule(block[atIndex:])
		if err != nil {
			return atIndex, "", "", false
		}

		return atIndex + len(block[atIndex:]) - len(rest), "", "", true
	}

	openRel := strings.IndexByte(block[atIndex+1+len(ident):], '{')
	if openRel < 0 {
		rest, err := skipAtRule(block[atIndex:])
		if err != nil {
			return atIndex, "", "", false
		}

		return atIndex + len(block[atIndex:]) - len(rest), "", "", true
	}

	open := atIndex + 1 + len(ident) + openRel

	inner, rest, err := takeBlock(block, open)
	if err != nil {
		return atIndex, "", "", false
	}

	return len(block) - len(rest), strings.ToLower(ident), inner, true
}

func setPageMarginBox(boxes *PageMarginBoxes, name, content string) {
	switch name {
	case "top-left":
		boxes.TopLeft = content
	case "top-center":
		boxes.TopCenter = content
	case "top-right":
		boxes.TopRight = content
	case "bottom-left":
		boxes.BottomLeft = content
	case "bottom-center":
		boxes.BottomCenter = content
	case "bottom-right":
		boxes.BottomRight = content
	}
}

// quotedPageBoxContent returns a concatenated quoted-string content list.
// Functions (counter, running, string) and keywords (none, normal) drop.
func quotedPageBoxContent(block string) (string, bool) {
	var text string

	found := false

	for _, decl := range parseDeclarations(block) {
		if strings.ToLower(decl.Prop) != "content" {
			continue
		}

		part, ok := quotedContentList(decl.Value)
		if !ok {
			return "", false
		}

		text = part
		found = true
	}

	return text, found && text != ""
}

//nolint:cyclop // quoted content validation follows the small CSS token grammar
func quotedContentList(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	low := strings.ToLower(value)
	if low == "none" || low == "normal" {
		return "", false
	}

	var out strings.Builder

	idx := 0
	saw := false

	for idx < len(value) {
		for idx < len(value) && (value[idx] == ' ' || value[idx] == '\t' || value[idx] == '\n' || value[idx] == '\r') {
			idx++
		}

		if idx >= len(value) {
			break
		}

		quote := value[idx]
		if quote != '"' && quote != '\'' {
			return "", false
		}

		end := idx + 1
		for end < len(value) && value[end] != quote {
			if value[end] == '\\' && end+1 < len(value) {
				end += 2

				continue
			}

			end++
		}

		if end >= len(value) {
			return "", false
		}

		out.WriteString(decodePageBoxString(value[idx+1 : end]))

		saw = true
		idx = end + 1
	}

	if !saw {
		return "", false
	}

	return out.String(), true
}

func decodePageBoxString(src string) string {
	if !strings.ContainsRune(src, '\\') {
		return src
	}

	var out strings.Builder

	out.Grow(len(src))

	for idx := 0; idx < len(src); idx++ {
		if src[idx] != '\\' || idx+1 >= len(src) {
			out.WriteByte(src[idx])

			continue
		}

		idx++
		out.WriteByte(src[idx])
	}

	return out.String()
}
