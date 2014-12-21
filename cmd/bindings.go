package cmd

import (
	"fmt"
	"strings"
)

type bindingConflict struct {
	b1 Binding
	b2 Binding
}

func (e *bindingConflict) String() string {
	return fmt.Sprintf(
		"Multiple bindings attempted to register for keysequence '%v'.",
		KeysString(b1.From))
}

type Binding struct {
	From []Key
	To   func()
}

type BindingTree struct {
	Binding  *func()
	Subtrees map[Key]*BindingTree
}

func NewBindingTree(bindings []*Binding) (*BindingTree, error) {
	t := &BindingTree{nil, make(map[Key]*BindingTree)}
	for binding := range bindings {
		err := t.Append(binding)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func (t *BindingTree) Append(binding *Binding) error {
	for key := range binding.From {
		next, ok := b.Subtrees[key]
		if !ok {
			next = BindingTree{nil, make(map[Key]*BindingTree)}
			t.Subtrees[key] = next
		}
		t = next
	}
	// Binding conflict
	if t.Binding != nil {
		return bindingConflict{t.Binding, binding}
	}
	t.Binding = binding
	return nil
}

// ParseBinding parses a single binding string, as given to the bind command.
// e.g. fg<Return> ::builtin:foobar
// Currently only ::builtin: type bindings are supported, and a numeric count
// (e.g. d{n}d is not supported.)
// TODO
func ParseBinding(str string, builtins map[string]func()) (*Binding, error) {
	arr = strings.Split(str, " ")
	if len(arr) != 2 {
		return nil, fmt.Errorf("Failed to parse binding: %v", str)
	}
	keys, err := ParseKeys(arr[0])
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(arr[1], "::builtin:") {
		return nil, fmt.Errorf("Only builtin mappings supported atm :(")
	}
	builtinName := arr[1][len("::builtin:"):len(arr[1])]
	builtin, ok := builtins[builtinName]
	if !ok {
		return nil, fmt.Errorf("Unknown builtin function: %v", builtinName)
	}
	return &Binding{keys, builtin}, nil
}
