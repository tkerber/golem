// Package cmd implements vim-like stateful operation.
package cmd

import (
	"log"
	"time"

	"github.com/tkerber/golem/debug"
)

// timeout is the time waited in normal mode before an ambiguous binding is
// executed.
const timeout = time.Millisecond * 500

// The State of some window/program is... well, it's state (in regards to
// keypresses)
type State interface {
	// Processes a key.
	// Returns the new state and whether the key was swallowed or not.
	ProcessKeyPress(key RealKey) (State, bool)
	// Gets the StateIndependant.
	GetStateIndependant() *StateIndependant
}

// NewState creates a new state, in its original setting.
func NewState(bindings *BindingTree, setState func(State)) State {
	return &NormalMode{
		&StateIndependant{bindings, setState},
		make([]Key, 0),
		bindings,
		make(chan bool),
	}
}

// A StateIndependant encompasses all data indepentant of the state, avoiding
// copying it around every time the state is changed.
type StateIndependant struct {
	Bindings *BindingTree
	SetState func(s State)
}

// NormalMode is a mode which mostly deals with key sequence bindings.
//
// These sequences are user-defined and mapped to specific actions, which get
// executed if the key sequence is used.
type NormalMode struct {
	*StateIndependant
	CurrentKeys []Key
	CurrentTree *BindingTree
	// If a binding could be processed, but further bindings are available
	// starting with the same key sequence (e.g. fg pressed, both fg and fgh
	// mapped), the state waits for a timeout to occur. If another key is
	// pressed before this happens, writing true to this channel will cancel
	// it. False is written if a key is pressed, but it is invalid. This will
	// be taken as a request to immediately execute the command.
	//
	// This should be a non-buffered channel.
	cancelTimeout chan bool
}

// NewNormalMode creates a baseline NormalMode state from a base state.
func NewNormalMode(s State) *NormalMode {
	si := s.GetStateIndependant()
	return &NormalMode{
		si,
		make([]Key, 0),
		si.Bindings,
		make(chan bool),
	}
}

// executeAfterTimeout executed a binding after a timeout.
//
// It may be cancelled by writing true into the timeoutChan, or sped up by
// writing false. It further invoked the state-setting function after the
// binding is executed, to reset the state to a blank normal mode.
func executeAfterTimeout(
	timeoutChan <-chan bool,
	binding func(),
	s State,
	keys []Key) {

	select {
	case cancel := <-timeoutChan:
		if cancel {
			return
		}
		// Continue
	case <-time.After(timeout):
		// Continue
	}
	go binding()
	// Somewhat ugly. We have to tell the owner of the state to reset it.
	log.Printf(
		"Executing binding for %v after delay...",
		KeysString(keys))
	s.GetStateIndependant().SetState(NewNormalMode(s))
}

// ProcessKeyPress processes exactly one key press in normal mode.
//
// It returns the new state, and whether the key press was swallowed or not.
func (s *NormalMode) ProcessKeyPress(key RealKey) (State, bool) {
	subtree, ok := s.CurrentTree.Subtrees[key.Normalize()]
	// No match found
	if !ok {
		// If any bindings are waiting to run, run them now.
		if s.CurrentTree.Binding != nil {
			s.cancelTimeout <- false
		}
		// If we are already in an empty normal mode, stay that way.
		if len(s.CurrentKeys) == 0 {
			return s, false
		}

		// Otherwise reset normal mode, and don't swallow the key, UNLESS it is
		// escape.
		return NewNormalMode(s), key.Keyval == KeyEscape
	}
	// If any bindings are waiting to run, cancel them now.
	if s.CurrentTree.Binding != nil {
		s.cancelTimeout <- true
	}
	timeoutChan := make(chan bool)
	// We have a binding
	if subtree.Binding != nil {
		soleBinding := len(subtree.Subtrees) == 0
		if soleBinding {
			// We have a difinite match for a binding. Execute it and reset the
			// state.
			if debug.PrintBindings {
				log.Printf("Executing binding for %v...", KeysString(append(s.CurrentKeys, key)))
			}
			go subtree.Binding()
			return NewNormalMode(s), true
		}
		// Otherwise, we wait for another keypress.
		go executeAfterTimeout(
			timeoutChan,
			subtree.Binding,
			s,
			append(s.CurrentKeys, key))
		// The return is the same as if no binding exists. i.e. Fallthrough.
	}
	// We add the key to our list and wait for a new keypress.
	return &NormalMode{
		s.StateIndependant,
		append(s.CurrentKeys, key),
		subtree,
		timeoutChan,
	}, true
}

// GetStateIndependant gets the state independant associated with this state.
func (s *NormalMode) GetStateIndependant() *StateIndependant {
	return s.StateIndependant
}

// InsertMode is a mode which ignores any keypresses, with the exception of the
// escape key,
type InsertMode struct {
	*StateIndependant
}

// NewInsertMode basically just copies over the StateIndependant and returns
// a new InsertMode.
func NewInsertMode(s State) *InsertMode {
	return &InsertMode{s.GetStateIndependant()}
}

// ProcessKeyPress passes through any keys except escape, which it immediately
// swallows and switches to normal mode.
func (s *InsertMode) ProcessKeyPress(key RealKey) (State, bool) {
	if key.Keyval == KeyEscape {
		return NewNormalMode(s), true
	}
	return s, false
}

// GetStateIndependant gets the state independant associated with this state.
func (s *InsertMode) GetStateIndependant() *StateIndependant {
	return s.StateIndependant
}

// CommandLineMode a mode which allows the user to enter a single line of text.
//
// The invoker of CommandLineMode supplies a Finalizer function, which is used
// to act on the text after the user presses enter.
type CommandLineMode struct {
	*StateIndependant
	CurrentKeys []Key
	Finalizer   func(string)
}

// NewCommandLineMode initializes a command line mode, starting from some
// state s and a finalizer function.
//
// The finalizer function is run if a command line entry is accepted, with the
// command line entry as an argument.
func NewCommandLineMode(s State, f func(string)) *CommandLineMode {
	return &CommandLineMode{
		s.GetStateIndependant(),
		make([]Key, 0),
		f,
	}
}

// NewPartialCommandLineMode acts like NewCommandLineMode, except that it
// defaults to a provided string as the command line instead of an empty one.
//
// Note that the string is parsed into it's Key components; if this fails,
// it defaults back to an empty string.
func NewPartialCommandLineMode(
	s State, part string, f func(string)) *CommandLineMode {

	keys := ParseKeys(part)
	return &CommandLineMode{
		s.GetStateIndependant(),
		keys,
		f}
}

// ProcessKeyPress processes the press of a single Key in CommandLineMode.
//
// Typically the Key is added to the current command line, with a few
// exceptions.
//
// BackSpace deletes the last read key, or if none are left, returns to
// NormalMode.
//
// Enter accepts the CommandLine and runs the finalizer, returning to
// NormalMode afterwards.
//
// Escape returns to NormalMode.
func (s *CommandLineMode) ProcessKeyPress(key RealKey) (State, bool) {
	switch key.Keyval {
	// Execute command line
	case KeyReturn:
		s.Finalizer(KeysStringSelective(s.CurrentKeys, false))
		fallthrough
	// Cancel command line
	case KeyEscape:
		return NewNormalMode(s), true
	// Delete last key.
	case KeyBackSpace:
		// Remove the last key from the list.
		if len(s.CurrentKeys) > 0 {
			return &CommandLineMode{
				s.StateIndependant,
				s.CurrentKeys[0 : len(s.CurrentKeys)-1],
				s.Finalizer,
			}, true
		}
		return NewNormalMode(s), true
	// Add new key
	default:
		return &CommandLineMode{
			s.StateIndependant,
			append(s.CurrentKeys, key),
			s.Finalizer,
		}, true
	}
}

// GetStateIndependant gets the state independant associated with this state.
func (s *CommandLineMode) GetStateIndependant() *StateIndependant {
	return s.StateIndependant
}
