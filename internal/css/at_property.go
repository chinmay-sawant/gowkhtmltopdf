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

	prop := PropertyRule{
		Name:        name,
		Syntax:      "",
		Initial:     "",
		Inherits:    false,
		InheritsSet: false,
	}

	for _, declaration := range parseDeclarations(block) {
		applyPropertyDeclaration(&prop, declaration)
	}

	str.Properties = append(str.Properties, prop)

	return rest, nil
}

// applyPropertyDeclaration copies one declaration of an @property block into
// its registration, ignoring descriptors the engine does not track.
func applyPropertyDeclaration(prop *PropertyRule, decl Declaration) {
	switch decl.Prop {
	case "syntax":
		prop.Syntax = strings.Trim(decl.Value, `"'`)
	case "initial-value":
		prop.Initial = decl.Value
	case "inherits":
		switch strings.ToLower(strings.TrimSpace(decl.Value)) {
		case "true":
			prop.Inherits, prop.InheritsSet = true, true
		case "false":
			prop.Inherits, prop.InheritsSet = false, true
		}
	}
}
