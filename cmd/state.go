// Package cmd implements vim-like stateful operation.
package cmd

import (
	"log"
	"time"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"
)

// max returns the greater of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the lesser of two integers.
func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// ImmutableAppend functions like append for a slice of keys, except it is
// guaranteed to return a freshly allocated slice every time.
func ImmutableAppend(keys []Key, app ...Key) []Key {
	ret := make([]Key, len(keys)+len(app))
	copy(ret[:len(keys)], keys)
	copy(ret[len(keys):], app)
	return ret
}

var pasteKey = NewKeyFromString("C-v")
var primarySelectionPasteKey = NewKeyFromString("C-V")

// PrintBindings specifies whether or not to print bindings as and when they
// run.
var PrintBindings = false

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
	// Gets the code of the current substate.
	GetSubstate() Substate
}

// A ContainerState is a state which contains another state.
type ContainerState interface {
	State
	ChildState() State
	SwapChildState(newState State) ContainerState
}

// A Substate allows using the same state for several different purposes.
//
// Substates are not defined in this module, with the exception of the
// SubstateDefault, and left for the main program to define.
type Substate uint

// A CompletionFunction is a function called to complete a state.
//
// It takes the state to complete, a function to be called when the first
// completion is found (with true, or false if none is found), and a pointer to
// the slice of completed states which will be generated concurrently.
//
// It returns a function which can be called to cancel completion. Completion
// should always be cancelled eventually.
type CompleterFunction func(s State, firstFunc func(ok bool), states *[]State) (cancel func())

// SubstateDefault is a substate marking that the "default" substate is being
// used. It is not defined more specifically (that is up to the application
// to decide), except that NormalMode's SubstateDefault should be the single
// state which the application has as its default "resting" state.
const SubstateDefault Substate = 0

// NewState creates a new state, in its original setting.
func NewState(
	bindings map[Substate]*BindingTree,
	setState func(State),
	completer CompleterFunction) State {

	return &NormalMode{
		&StateIndependant{bindings, setState, completer},
		SubstateDefault,
		make([]Key, 0),
		bindings[SubstateDefault],
		make(chan bool),
		0,
		false,
		false,
	}
}

// A StateIndependant encompasses all data indepentant of the state, avoiding
// copying it around every time the state is changed.
type StateIndependant struct {
	Bindings  map[Substate]*BindingTree
	SetState  func(s State)
	Completer CompleterFunction
}

// NormalMode is a mode which mostly deals with key sequence bindings.
//
// These sequences are user-defined and mapped to specific actions, which get
// executed if the key sequence is used.
type NormalMode struct {
	*StateIndependant
	Substate
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

	// The single number associated with the key sequence.
	num int
	// Whether or not we are currently parsing a <num> virtual key.
	inNum bool
	// Whether or not we parsed a <num> virtual key.
	hadNum bool
}

// NewNormalMode creates a baseline NormalMode state from a base state.
func NewNormalMode(s State) *NormalMode {
	return NewNormalModeWithSubstate(s, SubstateDefault)
}

// NewNormalModeWithSubstate creates a new NormalMode with the specified
// substate.
func NewNormalModeWithSubstate(s State, st Substate) *NormalMode {
	si := s.GetStateIndependant()
	return &NormalMode{
		si,
		st,
		make([]Key, 0),
		si.Bindings[st],
		make(chan bool),
		0,
		false,
		false,
	}
}

// PredictState predicts the state if "fast forwarded" a slice of keys.
//
// A few peculiarities are of note:
//
//  - No bindings are executed, under any circumstances.
//  - The virtual <num> key is not handled.
//
// This method is primarily meant for creating states predicted through
// completion.
func (s *NormalMode) PredictState(keys []Key) *NormalMode {
	t := s.CurrentTree
	for _, k := range keys {
		t := t.Subtrees[k]
		if t == nil {
			return NewNormalMode(s)
		}
	}
	newKeys := make([]Key, len(s.CurrentKeys)+len(keys))
	copy(newKeys[:len(s.CurrentKeys)], s.CurrentKeys)
	copy(newKeys[len(s.CurrentKeys):], keys)
	return &NormalMode{
		s.StateIndependant,
		s.Substate,
		newKeys,
		t,
		make(chan bool, 1),
		s.num,
		false,
		s.hadNum || s.inNum,
	}
}

// executeAfterTimeout executed a binding after a timeout.
//
// It may be cancelled by writing true into the timeoutChan, or sped up by
// writing false. It further invoked the state-setting function after the
// binding is executed, to reset the state to a blank normal mode.
func executeAfterTimeout(
	timeoutChan <-chan bool,
	binding func([]Key, *int, Substate),
	nump *int,
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
	go binding(keys, nump, s.GetSubstate())
	// Somewhat ugly. We have to tell the owner of the state to reset it.
	if PrintBindings {
		log.Printf(
			"Executing binding for %v after delay...",
			KeysString(keys))
	}
	s.GetStateIndependant().SetState(NewNormalMode(s))
}

// ProcessKeyPress processes exactly one key press in normal mode.
//
// It returns the new state, and whether the key press was swallowed or not.
func (s *NormalMode) ProcessKeyPress(key RealKey) (State, bool) {
	if s.CurrentTree == nil {
		s.CurrentTree = s.Bindings[s.Substate]
		if s.CurrentTree == nil {
			return s, false
		}
	}
	subtree, ok := s.CurrentTree.Subtrees[key.Normalize()]
	num := s.num
	inNum := s.inNum
	hadNum := s.hadNum
	if ok && inNum {
		inNum = false
		hadNum = true
	}
	// Start completion.
	switch key.Keyval {
	case KeyTab:
		NewCompletion(s)
		return s, false
	case KeyReturn, KeyKPEnter:
		if s.CurrentTree.Binding != nil {
			if PrintBindings {
				log.Printf("Executing binding for %v...",
					KeysString(s.CurrentKeys))
			}
			var nump *int
			if hadNum || inNum {
				nump = &num
			} else {
				nump = nil
			}
			go subtree.Binding.To(
				s.CurrentKeys,
				nump,
				s.Substate)
		}
	}
	// If we are waiting for a virtual <num> key, and the key pressed was
	// a number, we use up the <num> key, and set the number.
	// If we just used up a <num> key, and the key pressed was a number,
	// we don't use up any keys, and amend the number.
	// We check if a <num> key was used simply by checking if the saved
	// num is zero.
	if !ok && key.IsNum() {
		if s.inNum {
			// We are currently in a <num> virtual key.
			digit, _ := key.NumVal()
			num = num*10 + digit
			ok = true
			subtree = s.CurrentTree
		} else {
			// We aren't in a new <num> virtual key. Check if we can start
			// a new one.
			subtree, ok = s.CurrentTree.Subtrees[VirtualKey("num")]
			if ok {
				// If we can, start the new num.
				num, _ = key.NumVal()
				inNum = true
				hadNum = false
			}
		}
	}
	// Key wasn't handled.
	if !ok {
		// If any bindings are waiting to run, run them now.
		if s.CurrentTree.Binding != nil {
			s.cancelTimeout <- false
		}
		// If we are already in an empty normal mode, stay that way.
		if len(s.CurrentKeys) == 0 && s.Substate == SubstateDefault {
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
		// We use a pointer to num to pass to the executers. That was, passing
		// nil indicates no number was passed.
		// As states are stateful, the number pointed to is guaranteed not
		// to change.
		var nump *int
		if hadNum || inNum {
			nump = &num
		} else {
			nump = nil
		}
		if soleBinding {
			// We have a difinite match for a binding. Execute it and reset the
			// state.
			if PrintBindings {
				log.Printf("Executing binding for %v...",
					KeysString(ImmutableAppend(s.CurrentKeys, key)))
			}
			go subtree.Binding.To(
				ImmutableAppend(s.CurrentKeys, key),
				nump,
				s.Substate)
			return NewNormalMode(s), true
		}
		// Otherwise, we wait for another keypress.
		go executeAfterTimeout(
			timeoutChan,
			subtree.Binding.To,
			nump,
			s,
			ImmutableAppend(s.CurrentKeys, key))
		// The return is the same as if no binding exists. i.e. Fallthrough.
	}
	// We add the key to our list and wait for a new keypress.
	return &NormalMode{
		s.StateIndependant,
		s.Substate,
		ImmutableAppend(s.CurrentKeys, key),
		subtree,
		timeoutChan,
		num,
		inNum,
		hadNum,
	}, true
}

// GetStateIndependant gets the state independant associated with this state.
func (s *NormalMode) GetStateIndependant() *StateIndependant {
	return s.StateIndependant
}

// GetSubstate gets the substate associated with this state.
func (s *NormalMode) GetSubstate() Substate {
	return s.Substate
}

// InsertMode is a mode which ignores any keypresses, with the exception of the
// escape key,
type InsertMode struct {
	*StateIndependant
	Substate
}

// NewInsertMode basically just copies over the StateIndependant and returns
// a new InsertMode.
func NewInsertMode(s State, st Substate) *InsertMode {
	return &InsertMode{s.GetStateIndependant(), st}
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

// GetSubstate gets the substate associated with this state.
func (s *InsertMode) GetSubstate() Substate {
	return s.Substate
}

// CommandLineMode a mode which allows the user to enter a single line of text.
//
// The invoker of CommandLineMode supplies a Finalizer function, which is used
// to act on the text after the user presses enter.
type CommandLineMode struct {
	*StateIndependant
	Substate
	CurrentKeys []Key
	CursorPos   int
	CursorHome  int
	CursorEnd   int
	Finalizer   func(string)
}

// NewCommandLineMode initializes a command line mode, starting from some
// state s and a finalizer function.
//
// The finalizer function is run if a command line entry is accepted, with the
// command line entry as an argument.
func NewCommandLineMode(
	s State,
	st Substate,
	f func(string)) *CommandLineMode {

	return &CommandLineMode{
		s.GetStateIndependant(),
		st,
		make([]Key, 0),
		0,
		0,
		0,
		f,
	}
}

// NewPartialCommandLineMode acts like NewCommandLineMode, except that it
// defaults to a provided string as the command line instead of an empty one.
//
// Note that the strings are parsed into their Key components.
func NewPartialCommandLineMode(
	s State,
	st Substate,
	beforeCursor,
	afterCursor string,
	f func(string)) *CommandLineMode {

	keysBC := ParseKeys(beforeCursor)
	keysAC := ParseKeys(afterCursor)
	keys := make([]Key, len(keysBC)+len(keysAC))
	copy(keys[:len(keysBC)], keysBC)
	copy(keys[len(keysBC):], keysAC)
	return &CommandLineMode{
		s.GetStateIndependant(),
		st,
		keys,
		len(keysBC),
		len(keysBC),
		len(keysAC),
		f}
}

// Paste pastes a string into the command line.
func (s *CommandLineMode) Paste(str string) State {
	insertKeys := ParseKeys(str)
	// Grow keys
	newKeys := make([]Key, len(s.CurrentKeys)+len(insertKeys))
	// Copy data over
	copy(newKeys[:s.CursorPos], s.CurrentKeys[:s.CursorPos])
	copy(newKeys[s.CursorPos:s.CursorPos+len(insertKeys)], insertKeys)
	copy(newKeys[s.CursorPos+len(insertKeys):], s.CurrentKeys[s.CursorPos:])
	// Return
	return &CommandLineMode{
		s.StateIndependant,
		s.Substate,
		newKeys,
		s.CursorPos + len(insertKeys),
		s.CursorHome,
		s.CursorEnd,
		s.Finalizer}
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
	key = key.Normalize()
	if key == pasteKey || key == primarySelectionPasteKey {
		var clip *gtk.Clipboard
		var err error
		if key == pasteKey {
			clip, err = gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		} else {
			clip, err = gtk.ClipboardGet(gdk.SELECTION_PRIMARY)
		}
		if err != nil {
			log.Printf("Failed to acquire clipboard: %v", err)
			return s, false
		}
		str, err := clip.WaitForText()
		if err != nil {
			return s, true
		}
		return s.Paste(str), true
	}
	switch key.Keyval {
	// Complete command
	case KeyTab:
		NewCompletion(s)
		return s, true
	// Move cursor to start
	case KeyKPHome:
		fallthrough
	case KeyHome:
		if s.CursorPos == 0 {
			return s, false
		}
		return &CommandLineMode{
			s.StateIndependant,
			s.Substate,
			s.CurrentKeys,
			s.CursorHome,
			s.CursorHome,
			s.CursorEnd,
			s.Finalizer}, true
	// Move cursor to end
	case KeyKPEnd:
		fallthrough
	case KeyEnd:
		if s.CursorPos == len(s.CurrentKeys) {
			return s, false
		}
		return &CommandLineMode{
			s.StateIndependant,
			s.Substate,
			s.CurrentKeys,
			len(s.CurrentKeys) - s.CursorEnd,
			s.CursorHome,
			s.CursorEnd,
			s.Finalizer}, true
	// Execute command line
	case KeyKPEnter:
		fallthrough
	case KeyReturn:
		s.Finalizer(KeysStringSelective(s.CurrentKeys, false))
		fallthrough
	// Cancel command line
	case KeyEscape:
		return NewNormalMode(s), true
	// Move cursor left
	case KeyKPLeft:
		fallthrough
	case KeyLeft:
		pos := max(s.CursorPos-1, 0)
		return &CommandLineMode{
			s.StateIndependant,
			s.Substate,
			s.CurrentKeys,
			pos,
			min(pos, s.CursorHome),
			s.CursorEnd,
			s.Finalizer,
		}, true
	// Move cursor right
	case KeyKPRight:
		fallthrough
	case KeyRight:
		pos := min(s.CursorPos+1, len(s.CurrentKeys))
		return &CommandLineMode{
			s.StateIndependant,
			s.Substate,
			s.CurrentKeys,
			pos,
			s.CursorHome,
			min(len(s.CurrentKeys)-pos, s.CursorEnd),
			s.Finalizer,
		}, true
	// Delete last key.
	case KeyDelete:
		fallthrough
	case KeyKPDelete:
		// Remove the next key from the list.
		if s.CursorPos < len(s.CurrentKeys) {
			newKeys := make([]Key, len(s.CurrentKeys)-1)
			// Copy keys before cursor
			copy(
				newKeys[:s.CursorPos],
				s.CurrentKeys[:s.CursorPos])
			// Copy all but one key after cursor
			copy(
				newKeys[s.CursorPos:],
				s.CurrentKeys[s.CursorPos+1:])
			return &CommandLineMode{
				s.StateIndependant,
				s.Substate,
				newKeys,
				s.CursorPos,
				s.CursorHome,
				min(len(newKeys)-s.CursorPos, s.CursorEnd),
				s.Finalizer,
			}, true
		} else if len(s.CurrentKeys) == 0 {
			return NewNormalMode(s), true
		}
		return s, false
	// Delete next key. Very similar to above.
	case KeyBackSpace:
		// Remove the last key from the list.
		if s.CursorPos > 0 {
			newKeys := make([]Key, len(s.CurrentKeys)-1)
			// Copy all but one key before cursor
			copy(
				newKeys[:s.CursorPos-1],
				s.CurrentKeys[:s.CursorPos-1])
			// Copy keys after cursor
			copy(
				newKeys[s.CursorPos-1:],
				s.CurrentKeys[s.CursorPos:])
			return &CommandLineMode{
				s.StateIndependant,
				s.Substate,
				newKeys,
				s.CursorPos - 1,
				min(s.CursorPos-1, s.CursorHome),
				s.CursorEnd,
				s.Finalizer,
			}, true
		} else if len(s.CurrentKeys) == 0 {
			return NewNormalMode(s), true
		}
		return s, false
	// Add new key
	default:
		newKeys := make([]Key, len(s.CurrentKeys)+1)
		// Copy keys before cursor
		copy(
			newKeys[:s.CursorPos],
			s.CurrentKeys[:s.CursorPos])
		// Copy keys after cursor
		copy(
			newKeys[s.CursorPos+1:],
			s.CurrentKeys[s.CursorPos:])
		newKeys[s.CursorPos] = key
		return &CommandLineMode{
			s.StateIndependant,
			s.Substate,
			newKeys,
			s.CursorPos + 1,
			s.CursorHome,
			s.CursorEnd,
			s.Finalizer,
		}, true
	}
}

// GetStateIndependant gets the state independant associated with this state.
func (s *CommandLineMode) GetStateIndependant() *StateIndependant {
	return s.StateIndependant
}

// GetSubstate gets the substate associated with this state.
func (s *CommandLineMode) GetSubstate() Substate {
	return s.Substate
}

// StatusMode is a mode which displays a single status line.
//
// It keeps the previous state as a part of it, and does nothing itself.
// All methods called are directed to the previous that; most notably: any
// key press will revert out of StatusMode, and the old state will handle the
// key press as normal.
type StatusMode struct {
	State
	Substate
	Status string
}

// NewStatusMode creates a new StatusMode with a given state and status string.
func NewStatusMode(s State, st Substate, status string) *StatusMode {
	// Only wrap the innermost state, avoid nested status modes (they are
	// useless anyway)
	if sm, ok := s.(*StatusMode); ok {
		return &StatusMode{sm.State, st, status}
	}
	return &StatusMode{s, st, status}
}

// GetSubstate gets the substate associated with this state.
func (s *StatusMode) GetSubstate() Substate {
	return s.Substate
}

// ChildState retrieves the status modes contained state.
func (s *StatusMode) ChildState() State {
	return s.State
}

// SwapChildState swaps out the child state, returning the resulting container.
func (s *StatusMode) SwapChildState(newState State) ContainerState {
	return &StatusMode{
		newState,
		s.Substate,
		s.Status,
	}
}

// ConfirmMode tries to confirm an action with the user.
//
// If a key in ConfirmKeys is pressed, the action is confirmed (callback called
// w/ true)
// If a key in CancelKeys is pressed, the action is cancelled (callback called
// w/ false)
// If Escape is pressed, the callback is not called at all.
// If Enter is pressed, and Default is not nil, the default is taken.
type ConfirmMode struct {
	State
	Substate
	Prompt      string
	ConfirmKeys []Key
	CancelKeys  []Key
	Default     *bool
	Callback    func(bool)
}

// NewYesNoConfirmMode creates a new ConfirmMode for a yes/no confirmation.
func NewYesNoConfirmMode(
	s State,
	st Substate,
	prompt string,
	def *bool,
	callback func(bool)) *ConfirmMode {

	if def == nil {
		prompt = prompt + " (y/n):"
	} else if *def {
		prompt = prompt + " (Y/n):"
	} else {
		prompt = prompt + " (y/N):"
	}
	return &ConfirmMode{
		s,
		st,
		prompt,
		[]Key{NewKeyFromRune('y'), NewKeyFromRune('Y')},
		[]Key{NewKeyFromRune('n'), NewKeyFromRune('N')},
		def,
		callback}
}

// ProcessKeyPress processes a key, and check if the confirmation was
// handled.
func (s *ConfirmMode) ProcessKeyPress(k RealKey) (State, bool) {
	k = k.Normalize()
	for _, k2 := range s.ConfirmKeys {
		if k == k2 {
			s.Callback(true)
			return s.State, true
		}
	}
	for _, k2 := range s.CancelKeys {
		if k == k2 {
			s.Callback(false)
			return s.State, true
		}
	}
	switch k.Keyval {
	case KeyReturn:
		fallthrough
	case KeyKPEnter:
		if s.Default != nil {
			s.Callback(*s.Default)
		}
		fallthrough
	case KeyEscape:
		return s.State, true
	default:
		return s, false
	}
}

// GetSubstate gets the substate of the current state.
func (s *ConfirmMode) GetSubstate() Substate {
	return s.Substate
}

// ChildState returns the contained state of the confirm mode.
func (s *ConfirmMode) ChildState() State {
	return s.State
}

// SwapChildState swaps out the child state, returning the resulting container.
func (s *ConfirmMode) SwapChildState(newState State) ContainerState {
	return &ConfirmMode{
		newState,
		s.Substate,
		s.Prompt,
		s.ConfirmKeys,
		s.CancelKeys,
		s.Default,
		s.Callback,
	}
}

// CompletionMode is a mode in which some other state is being completed.
type CompletionMode struct {
	State
	Substate
	CompletionStates  *[]State
	CurrentCompletion int
	CancelFunc        func()
}

// NewCompletion schedules a completion to start once calculations have found
// at least one completion.
//
// If no completions are found, nothing is done.
func NewCompletion(s State) {
	var completionStates []State
	var cancelFunc func()
	si := s.GetStateIndependant()
	firstFunc := func(exists bool) {
		if exists {
			si.SetState(&CompletionMode{
				s,
				SubstateDefault,
				&completionStates,
				0,
				cancelFunc,
			})
		}
	}
	cancelFunc = si.Completer(s, firstFunc, &completionStates)
}

// GetSubstate gets the substate of the current state.
func (s *CompletionMode) GetSubstate() Substate {
	return s.Substate
}

// ProcessKeyPress processes a single key press in completion mode.
//
// Tab or down chooses the next completion, Shift-Tab (ISO Left Tab) or up the
// previous one, escape cancels it and any other key is passed to the
// completion state (effectively accepting it)
func (s *CompletionMode) ProcessKeyPress(key RealKey) (State, bool) {
	switch key.Keyval {
	case KeyEscape:
		s.CancelFunc()
		return s.State, true
	case KeyTab, KeyDown, KeyKPDown:
		comp := s.CurrentCompletion + 1
		if comp >= len(*s.CompletionStates) {
			comp %= len(*s.CompletionStates)
		}
		return &CompletionMode{
			s.State,
			s.Substate,
			s.CompletionStates,
			comp,
			s.CancelFunc,
		}, true
	case KeyLeftTab, KeyUp, KeyKPUp:
		comp := s.CurrentCompletion - 1
		if comp < 0 {
			comp = len(*s.CompletionStates) - 1
		}
		return &CompletionMode{
			s.State,
			s.Substate,
			s.CompletionStates,
			comp,
			s.CancelFunc,
		}, true
	default:
		s.CancelFunc()
		return (*s.CompletionStates)[s.CurrentCompletion].ProcessKeyPress(key)
	}
}

// ChildState gets the original state being completed.
func (s *CompletionMode) ChildState() State {
	return s.State
}

// SwapChildState swaps out the child state, returning the resulting container.
func (s *CompletionMode) SwapChildState(newState State) ContainerState {
	return &CompletionMode{
		newState,
		s.Substate,
		s.CompletionStates,
		s.CurrentCompletion,
		s.CancelFunc,
	}
}
