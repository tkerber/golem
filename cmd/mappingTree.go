package cmd

// A mappingTree maps a single rune to any further mapping trees which may
// follow. If a rune is mapped to nil, it isn't mapped to anything
type mappingTree struct {
	mappedCommand *string
	subtrees      map[rune]*mappingTree
}

func (t *mappingTree) subtree(r rune) (*mappingTree, bool) {
	st, ok := t.subtrees[r]
	return st, ok
}

func (t *mappingTree) command() (string, bool) {
	if t.mappedCommand == nil {
		return "", false
	} else {
		return *t.mappedCommand, true
	}
}

func compileMappingTree(m map[string]string) *mappingTree {
	t := &mappingTree{nil, make(map[rune]*mappingTree)}
	for key, val := range m {
		currT := t
		for _, runeVal := range key {
			newT, ok := currT.subtrees[runeVal]
			if !ok {
				newT = &mappingTree{nil, make(map[rune]*mappingTree)}
				currT.subtrees[runeVal] = newT
			}
			currT = newT
		}
		// So that not everything points to the same value.
		val := val
		currT.mappedCommand = &val
	}
	return t
}
