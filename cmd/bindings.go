package cmd

import (
	"fmt"
	"strings"
)

// A bindingConflict indicates that a binding conflicts with an existing one.
type bindingConflict Binding

// Error prints the error message associated with this bindingConflict.
func (e *bindingConflict) Error() string {
	return fmt.Sprintf(
		"Multiple bindings attempted to register for keysequence '%v'.",
		KeysString(e.From))
}

// Builtins are a collection of functions, which are accessible by their name.
type Builtins map[string]func(*int)

// A RawBinding map one string (representing the keysequence to be pressed) to
// another (representing what the binding should do).
type RawBinding struct {
	From string
	To   string
}

// hasPrefixes checks if str has of of prefixes as its prefix.
func hasPrefixes(str string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}

// stripPrefixes strips at most one of prefixes from the front of str.
//
// If str doesn't start with any of the prefixes, it is returned unchanged.
func stripPrefixes(str string, prefixes ...string) string {
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return str[len(prefix):len(str)]
		}
	}
	return str
}

// ParseBinding parses a single binding string, as given to the bind command.
//
// e.g. fg<Enter> builtin:foobar
//
// The To part of the binding can be interpreted in several ways:
//
// If it starts with 'builtin:' or 'b:' it maps directly to a builtin
// function.
//
// If it starts with 'command:', 'cmd:' or 'c:' it maps to executing the
// following command. (This is implemented by running the builtin command
// 'runCmd' with the command as the single string argument)
//
// No other prefixes are currently supported.
func (b RawBinding) ParseBinding(
	builtins Builtins, runCmd func(string)) (*Binding, error) {

	keys := ParseKeys(b.From)
	if hasPrefixes(b.To, "builtin:", "b:") {
		builtinName := stripPrefixes(b.To, "builtin:", "b:")
		builtin, ok := builtins[builtinName]
		if !ok {
			return nil, fmt.Errorf("Unknown builtin function: %v", builtinName)
		}
		return &Binding{
			keys,
			func(_ []Key, i *int, _ Substate) { builtin(i) },
		}, nil
	} else if hasPrefixes(b.To, "command:", "cmd:", "c:") {
		cmd := stripPrefixes(b.To, "command:", "cmd:", "c:")
		return &Binding{keys, func(_ []Key, _ *int, _ Substate) {
			runCmd(cmd)
		}}, nil
	}
	// TODO maybe add other mapping types.
	return nil, fmt.Errorf("Unkown mapping: %v", b.To)
}

// ParseRawBindings applies RawBinding.ParseBinding to a slice of raw
// bindings.
//
// Parse errors are not ignored, but parsing will continue regardless. All
// parse errors are returned, along with all bindings which did not fail to
// parse.
func ParseRawBindings(
	bindings []RawBinding,
	builtins Builtins,
	runCmd func(string)) ([]*Binding, []error) {

	ret := make([]*Binding, 0, len(bindings))
	var errs []error
	for _, binding := range bindings {
		b, err := binding.ParseBinding(builtins, runCmd)
		// We do return errors, but we continue parsing regardless. This way,
		// a single parse error still yields a useable program.
		if err != nil {
			errs = append(errs, err)
		} else {
			ret = append(ret, b)
		}
	}
	return ret, errs
}

// A Binding maps a sequence of keys to a function to be executed when they
// are pressed. The executer function will be passed the exact key sequence
// pressed, as well as a pointer to the (potentially) extracted <num> virtual
// key.
type Binding struct {
	From []Key
	To   func([]Key, *int, Substate)
}

// A BindingTree is a tree structure for a set of bindings. Each key sequence
// corresponds to a node in the tree, (the empty sequence being the root),
// with a binding function optionally attached to any node.
type BindingTree struct {
	Binding  func([]Key, *int, Substate)
	Subtrees map[Key]*BindingTree
}

// NewBindingTree converts a slice of Bindings into a tree format. Errors
// occur only if bindings conflict.
//
// Building the binding tree continues if errors occur; only those elements
// which cause the conflict (i.e. the latter conflicting ones) will be ignored.
func NewBindingTree(bindings []*Binding) (*BindingTree, []error) {
	t := &BindingTree{nil, make(map[Key]*BindingTree)}
	var errs []error
	for _, binding := range bindings {
		err := t.Append(binding)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return t, errs
}

// Append adds a new binding to a BindingTree.
//
// Fails if the binding conflicts with an existing one.
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
