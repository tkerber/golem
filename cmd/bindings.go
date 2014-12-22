package cmd

import (
	"fmt"
	"strings"
)

type bindingConflict Binding

func (e *bindingConflict) Error() string {
	return fmt.Sprintf(
		"Multiple bindings attempted to register for keysequence '%v'.",
		KeysString(e.From))
}

type Builtins map[string]func()

type RawBinding struct {
	From string
	To   string
}

// ParseBinding parses a single binding string, as given to the bind command.
// e.g. fg<Return> ::builtin:foobar
// Currently only ::builtin: type bindings are supported, and a numeric count
// (e.g. d{n}d is not supported.)
// TODO
func (b RawBinding) ParseBinding(builtins Builtins) (*Binding, error) {
	keys, err := ParseKeys(b.From)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(b.To, "::builtin:") {
		return nil, fmt.Errorf("Only builtin mappings supported atm :(")
	}
	builtinName := b.To[len("::builtin:"):len(b.To)]
	builtin, ok := builtins[builtinName]
	if !ok {
		return nil, fmt.Errorf("Unknown builtin function: %v", builtinName)
	}
	return &Binding{keys, builtin}, nil
}

func ParseRawBindings(
	bindings []RawBinding,
	builtins Builtins) ([]*Binding, error) {

	ret := make([]*Binding, len(bindings))
	for i, binding := range bindings {
		b, err := binding.ParseBinding(builtins)
		if err != nil {
			return nil, err
		}
		ret[i] = b
	}
	return ret, nil
}

type Binding struct {
	From []Key
	To   func()
}

type BindingTree struct {
	Binding  func()
	Subtrees map[Key]*BindingTree
}

func NewBindingTree(bindings []*Binding) (*BindingTree, error) {
	t := &BindingTree{nil, make(map[Key]*BindingTree)}
	for _, binding := range bindings {
		err := t.Append(binding)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func (t *BindingTree) Append(binding *Binding) error {
	for _, key := range binding.From {
		next, ok := t.Subtrees[key]
		if !ok {
			next = &BindingTree{nil, make(map[Key]*BindingTree)}
			t.Subtrees[key] = next
		}
		t = next
	}
	// Binding conflict
	if t.Binding != nil {
		return (*bindingConflict)(binding)
	}
	t.Binding = binding.To
	return nil
}
