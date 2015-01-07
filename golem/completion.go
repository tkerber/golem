package golem

import (
	"fmt"

	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
)

var completionsShown = 5

type completion struct {
	state cmd.State
	str   string
}

// complete retrieves the possible completions for a state and started them
// in a slice at the passed pointer.
//
// Complete is intended to be run with a go statement:
//	go complete(s, cancelCompletion, ptr)
//
// Sending to the cancel channel terminates execution of the function (at
// pre-set intervals). It is recommended to buffer the cancel channel and
// limit to sending one item, as it isn't guaranteed to be read.
//
// Passing nil for ptr is a fatal error.
func (g *Golem) complete(s cmd.State, cancel <-chan bool, ptr *[]completion) {
	switch s := s.(type) {
	case *cmd.NormalMode:
		g.completeNormalMode(s, cancel, ptr)
	case *cmd.CommandLineMode:
		g.completeCommandLineMode(s, cancel, ptr)
	default:
		return
	}
}

// completeCommandLineMode completes a command line mode state.
func (g *Golem) completeCommandLineMode(
	s *cmd.CommandLineMode,
	cancel <-chan bool,
	ptr *[]completion) {

	// TODO
}

// completeNormalMode completes a normal mode state.
func (g *Golem) completeNormalMode(
	s *cmd.NormalMode,
	cancel <-chan bool,
	ptr *[]completion) {

	for b := range s.CurrentTree.IterLeaves() {
		select {
		case <-cancel:
			return
		default:
		}
		// We can't complete virtual keys.
		if _, ok := b.From[len(b.From)-1].(cmd.VirtualKey); ok {
			continue
		}
		// Get the new tree
		t := s.CurrentTree
		for _, k := range b.From {
			t = t.Subtrees[k]
		}
		var str string
		keysStr := cmd.KeysString(b.From)
		switch s.Substate {
		case states.NormalSubstateNormal:
			// TODO attach a short descriptive text/name/both to bindings.
			str = fmt.Sprintf("%s\t?????", keysStr)
		case states.NormalSubstateQuickmark,
			states.NormalSubstateQuickmarkTab,
			states.NormalSubstateQuickmarkWindow,
			states.NormalSubstateQuickmarksRapid:

			str = fmt.Sprintf("%s\t%s", keysStr, g.quickmarks[keysStr])
		}
		*ptr = append(*ptr, completion{
			s.PredictState(b.From),
			str,
		})
	}
}
