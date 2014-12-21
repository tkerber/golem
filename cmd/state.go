package cmd

import "time"

const timeout = time.Millisecond * 500

// The State of some window/program is... well, it's state (in regards to
// keypresses)
type State interface {
	// Processes a key.
	// Returns the new state and whether the key was swallowed or not.
	ProcessKeyPress(key Key) (State, bool)
}

// A StateIndependant encompasses all data indepentant of the state, avoiding
// copying it around every time the state is changed.
type StateIndependant struct {
	Bindings *BindingTree
	SetState func(s State)
}

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

func NewNormalMode(s *StateIndependant) *NormalMode {
	return &NormalMode{
		s,
		make([]Key, 0),
		s.Bindings,
		make(chan bool),
	}
}

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

		// Otherwise reset normal mode, and don't swallow the key, UNLESS
		// it is escape.
		return NewNormalMode(s.StateIndependant), key.Keyval == KeyEscape
	}
	if s.CurrentTree.Binding != nil {
		s.cancelTimeout <- true
	}
	if subtree.Binding != nil {
		// TODO if the bindings itself sets the state, it just gets
		// immediately overwritten.
		// possible fixes:
		// - check for such overwrites and ignore this methods return in such
		//   cases.
		// - have a specific type of binding which returns a new state.
		//   executing this type of binding would overwite the return value.
		// - delay state-setting bindings slightly
		soleBinding := len(subtree.Subtrees) == 0
		if soleBinding {
			// We have a difinite match for a binding. Execute it and
			// reset the state.
			go subtree.Binding()
			return NewNormalMode(s.StateIndependant), true
		}
		// Otherwise, we wait for another keypress.
		go func() {
			select {
			case cancel := <-s.cancelTimeout:
				if cancel {
					return
				}
				// Continue
			case <-time.After(timeout):
				// Continue
			}
			go subtree.Binding()
			// Somewhat ugly. We have to tell the owner of the state to
			// reset it.
			s.SetState(NewNormalMode(s.StateIndependant))
		}()
		// The return is the same as if no binding exists. i.e. Fallthrough.
	}
	return &NormalMode{
		s.StateIndependant,
		append(s.CurrentKeys, key),
		subtree,
		make(chan bool),
	}, true
}

type InsertMode struct {
	*StateIndependant
}

func (s *InsertMode) ProcessKeyPress(key Key) (State, bool) {
	if key.Keyval == KeyEscape {
		return NewNormalMode(s.StateIndependant), true
	}
	return s, false
}

type CommandMode struct {
	*StateIndependant
	CurrentCommand string
}
