package css

import "strings"

// parseLayerRule consumes a top-level @layer: a statement `@layer a, b;`
// that fixes layer order, or a block `@layer name { ... }` / `@layer { ... }`
// whose rules carry the layer's cascade rank.
func parseLayerRule(src string, str *Stylesheet, order *int) (string, error) {
	rest, rules, err := parseLayerBlockRules(src, str, order, "all", nil, 0)
	if err != nil {
		return "", err
	}

	str.Rules = append(str.Rules, rules...)

	return rest, nil
}

// parseLayerBlockRules parses one @layer statement or block and returns the
// rules stamped with the layer rank. media/contQ/depth describe the enclosing
// block when the @layer is nested inside @media/@container.
func parseLayerBlockRules(
	src string, str *Stylesheet, order *int, media string, contQ *ContainerQuery, depth int,
) (string, []Rule, error) {
	open := strings.IndexByte(src, '{')
	semi := strings.IndexByte(src, ';')

	if semi >= 0 && (open < 0 || semi < open) {
		for _, name := range strings.Split(src[len("@layer"):semi], ",") {
			if name = strings.TrimSpace(name); name != "" {
				str.layerRank(name)
			}
		}

		return src[semi+1:], nil, nil
	}

	if open < 0 {
		rest, err := skipAtRule(src)

		return rest, nil, err
	}

	block, rest, err := takeBlock(src, open)
	if err != nil {
		return "", nil, err
	}

	rank := str.layerRank(strings.TrimSpace(src[len("@layer"):open]))

	rules, err := parseRuleList(str, media, contQ, block, order, depth)
	if err != nil {
		return "", nil, err
	}

	setRuleLayer(rules, rank)

	return rest, rules, nil
}

// layerRank returns the 1-based cascade rank for a layer name, assigning the
// next rank on first use. Empty name is an anonymous layer. Rank 0 is
// reserved for unlayered rules, which win over every layer.
func (s *Stylesheet) layerRank(name string) int {
	if s.layerRanks == nil {
		s.layerRanks = make(map[string]int)
	}

	if name == "" {
		s.layerCount++

		return s.layerCount
	}

	if rank, ok := s.layerRanks[name]; ok {
		return rank
	}

	s.layerCount++
	s.layerRanks[name] = s.layerCount
	s.Layers = append(s.Layers, name)

	return s.layerCount
}

// setRuleLayer stamps rank on every rule of a layer block.
func setRuleLayer(rules []Rule, rank int) {
	for i := range rules {
		rules[i].Layer = rank
	}
}
