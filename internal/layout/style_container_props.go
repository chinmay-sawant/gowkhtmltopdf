package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// applyContainerProps owns container-type, container-name, and the container
// shorthand. The three properties are not inherited, so an explicit CSS-wide
// keyword must resolve here: the cascade drops unhandled declarations silently
// and no generic inheritance pass covers non-inherited properties. inherit
// copies the parent's used value; initial, unset, and revert use the property
// initial ("normal" / none).
func applyContainerProps(
	style *ResolvedStyle, prop, value string, parent *ResolvedStyle, hasParent bool,
) bool {
	parentType, parentName := containerParentValues(parent)

	switch prop {
	case "container-type":
		if v, ok := containerWideValue(value, parentType, contentNormal, hasParent); ok {
			style.ContainerType = v
		} else if v, ok := normalizeContainerType(value); ok {
			style.ContainerType = v
		}
	case "container-name":
		if v, ok := containerWideValue(value, parentName, "", hasParent); ok {
			style.ContainerName = v
		} else {
			style.ContainerName = css.ParseContainerNameValue(value)
		}
	case containerKeyword:
		applyContainerShorthand(style, value, parentType, parentName, hasParent)
	default:
		return false
	}

	return true
}

// containerParentValues returns the parent's used container type and name,
// mapping the unmaterialized initial type (empty string) to "normal" so
// `inherit` copies a meaningful value.
func containerParentValues(parent *ResolvedStyle) (string, string) {
	if parent == nil {
		return contentNormal, ""
	}

	parentType := parent.ContainerType
	if parentType == "" {
		parentType = contentNormal
	}

	return parentType, parent.ContainerName
}

// normalizeContainerType validates a container-type value against the
// keywords the engine can use for size queries.
func normalizeContainerType(value string) (string, bool) {
	low := strings.ToLower(strings.TrimSpace(value))
	if low == contentNormal || low == containerSize || low == containerInlineSize {
		return low, true
	}

	return "", false
}

// applyContainerShorthand resolves container as one declaration because every
// CSS-wide keyword applies to both longhands at once.
func applyContainerShorthand(style *ResolvedStyle, value, parentType, parentName string, hasParent bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case inheritKeyword:
		if hasParent {
			style.ContainerType, style.ContainerName = parentType, parentName

			return
		}

		style.ContainerType, style.ContainerName = contentNormal, ""
	case cssKeywordInitial, cssKeywordUnset, cssKeywordRevert:
		style.ContainerType, style.ContainerName = contentNormal, ""
	default:
		name, ctype := css.ParseContainerShorthand(value)
		style.ContainerName = name

		if ctype != "" {
			style.ContainerType = ctype
		}
	}
}

// containerWideValue resolves one CSS-wide keyword for a non-inherited
// container longhand. inherit copies the parent's used value; initial, unset,
// and revert fall back to the property initial because these properties do
// not inherit. Ordinary values return ("", false) so the caller validates.
func containerWideValue(value, parentValue, initial string, hasParent bool) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case inheritKeyword:
		if hasParent {
			return parentValue, true
		}

		return initial, true
	case cssKeywordInitial, cssKeywordUnset, cssKeywordRevert:
		return initial, true
	default:
		return "", false
	}
}
