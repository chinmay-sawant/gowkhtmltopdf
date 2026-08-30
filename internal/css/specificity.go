package css

// Specificity returns (a, b, c): ID count, class/attribute/pseudo count, type count.
// :has() / :not() / :is() contribute the specificity of their most specific
// argument (Selectors 4); :where() contributes 0. Parsed selectors return
// their cached triple; selectors built by hand (or by wrappers that recombine
// parts) compute it on the fly.
func Specificity(s Selector) (int, int, int) {
	if s.specValid {
		return s.spec[0], s.spec[1], s.spec[2]
	}

	return computeSpecificity(s)
}

// computeSpecificity walks the selector parts; see Specificity.
func computeSpecificity(s Selector) (int, int, int) {
	idCount, classCount, typeCount := 0, 0, 0

	for _, page := range s.Parts {
		if page.ID != "" {
			idCount++
		}

		classCount += len(page.Classes) + len(page.Attrs)

		if page.Tag != "*" {
			typeCount++
		}

		if page.PseudoElement != "" {
			typeCount++ // pseudo-elements count like type selectors
		}

		for _, pageSize := range page.Pseudos {
			aDelta, bDelta, cDelta := pseudoSpecificityDelta(pageSize)
			idCount += aDelta
			classCount += bDelta
			typeCount += cDelta
		}
	}

	return idCount, classCount, typeCount
}
