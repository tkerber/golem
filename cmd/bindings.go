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

// ParseBinding parses a single binding string, as given to the bind command.
// e.g. fg<Return> ::builtin:foobar
// Currently only ::builtin: type bindings are supported, and a numeric count
// (e.g. d{n}d is not supported.)
// TODO
func ParseBinding(
	from string,
	to string,
	builtins map[string]func(),
) (*Binding, error) {

	keys, err := ParseKeys(from)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(to, "::builtin:") {
		return nil, fmt.Errorf("Only builtin mappings supported atm :(")
	}
	builtinName := to[len("::builtin:"):len(to)]
	builtin, ok := builtins[builtinName]
	if !ok {
		return nil, fmt.Errorf("Unknown builtin function: %v", builtinName)
	}
	return &Binding{keys, builtin}, nil
}
