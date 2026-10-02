package css

import "strings"

// PropertyRule is one @property registration. Name is the custom property
// including the leading --; Syntax is the raw syntax descriptor without
// quotes; Initial is the raw initial-value ("" when absent); Inherits is the
// parsed inherits flag and InheritsSet records whether it was present.
type PropertyRule struct {
	Name        string
	Syntax      string
	Initial     string
	Inherits    bool
	InheritsSet bool
}

// parsePropertyRule consumes one @property block. A registration whose name
// is not a custom property ident is skipped.
func parsePropertyRule(src string, str *Stylesheet) (string, error) {
	open := strings.IndexByte(src, '{')
	if open < 0 {
		return skipAtRule(src)
	}

	name := strings.TrimSpace(src[len("@property"):open])
	if !strings.HasPrefix(name, "--") || !IsIdentToken(name) {
		return skipAtRule(src)
	}

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", err
	}

	prop := PropertyRule{Name: name}

	for _, d := range parseDeclarations(block) {
		switch d.Prop {
		case "syntax":
			prop.Syntax = strings.Trim(d.Value, `"'`)
		case "initial-value":
			prop.Initial = d.Value
		case "inherits":
			switch strings.ToLower(strings.TrimSpace(d.Value)) {
			case "true":
				prop.Inherits, prop.InheritsSet = true, true
			case "false":
				prop.Inherits, prop.InheritsSet = false, true
			}
		}
	}

	str.Properties = append(str.Properties, prop)

	return rest, nil
}
