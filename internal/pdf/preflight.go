package pdf

// PreflightEmbed verifies that fnt can be subset for used runes with the same
// scope rules ensureFont will apply. Callers (convert) must run this before
// committing paint so a failed face can be marked unavailable and the object
// re-laid-out with the next CSS/bundled fallback.
func PreflightEmbed(fnt *Font, used []rune) error {
	if fnt == nil {
		return errNilFont
	}

	if len(used) == 0 {
		used = []rune{' '}
	}

	scope := subsetSimple
	if needsType0(used) {
		scope = subsetUnicode
	}

	_, err := subsetFont(fnt, used, scope)

	return err
}
