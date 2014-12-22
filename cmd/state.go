package cmd

import (
	"log"
	"time"
)

// timeout is the time waited in normal mode before an ambiguous binding is
// executed.
const timeout = time.Millisecond * 500

// The State of some window/program is... well, it's state (in regards to
// keypresses)
type State interface {
	// Processes a key.
	// Returns the new state and whether the key was swallowed or not.
	ProcessKeyPress(key Key) (State, bool)
	// Gets the StateIndependant.
	GetStateIndependant() *StateIndependant
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

// NewNormalMode creates a baseline NormalMode state from a StateIndependant
// and returns it.
func NewNormalMode(s *StateIndependant) *NormalMode {
	return &NormalMode{
		s,
		make([]Key, 0),
		s.Bindings,
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
	s *StateIndependant,
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
	s.SetState(NewNormalMode(s))
}

// ProcessKeyPress processes exactly one key press in normal mode.
//
// It returns the new state, and whether the key press was swallowed or not.
func (s *NormalMode) ProcessKeyPress(key Key) (State, bool) {
	subtree, ok := s.CurrentTree.Subtrees[key.Normalize()]
	// No match found
	if !ok {
		if s.CurrentTree.Binding != nil {
			s.cancelTimeout <- false
		}
		// Don't change empty state.
		if len(s.CurrentKeys) == 0 {
			return s, false
		}

		// Otherwise reset normal mode, and don't swallow the key, UNLESS it is
		// escape.
		return NewNormalMode(s.StateIndependant), key.Keyval == KeyEscape
	}
	if s.CurrentTree.Binding != nil {
		s.cancelTimeout <- true
	}
	// We need this in the timeout thingy.
	timeoutChan := make(chan bool)
	if subtree.Binding != nil {
		soleBinding := len(subtree.Subtrees) == 0
		if soleBinding {
			// We have a difinite match for a binding. Execute it and reset the
			// state.
			log.Printf("Executing binding for %v...", KeysString(append(s.CurrentKeys, key)))
			go subtree.Binding()
			return NewNormalMode(s.StateIndependant), true
		}
		// Otherwise, we wait for another keypress.
		go executeAfterTimeout(
			timeoutChan,
			subtree.Binding,
			s.StateIndependant,
			append(s.CurrentKeys, key))
		// The return is the same as if no binding exists. i.e. Fallthrough.
	}
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
func (s *InsertMode) ProcessKeyPress(key Key) (State, bool) {
	if key.Keyval == KeyEscape {
		return NewNormalMode(s.StateIndependant), true
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
	CurrentLineCommand string
	Finalizer          func(string)
}
