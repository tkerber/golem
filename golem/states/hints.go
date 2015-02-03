package states

import (
	"errors"
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
)

// HintsCallback is an interface for golem.(*webView), implementing the
// methods needed by hints. (mostly web extension calls)
type HintsCallback interface {
	LinkHintsMode() (int64, error)
	FormVariableHintsMode() (int64, error)
	ClickHintsMode() (int64, error)
	EndHintsMode() error
	FilterHintsMode(string) (bool, error)
}

// HintsMode is a mode which displays key strings on items of intrest in a
// web view, and allows the selection of said items by typing these key
// strings.
type HintsMode struct {
	*cmd.StateIndependant
	cmd.Substate
	HintsCallback
	CurrentKeys      []cmd.Key
	ExecuterFunction func(string) bool
}

// NewHintsMode creates a new hints mode.
//
// It returns a hints mode, and asynchronously either an error or nil in the
// given channel.
//
// The special error nil is returned if no error occured, but hints mode
// should stop.
//
// The channel is closed when initialization finishes.
func NewHintsMode(
	s cmd.State,
	st cmd.Substate,
	cb HintsCallback,
	e func(string) bool) (*HintsMode, <-chan error) {

	hm := &HintsMode{
		s.GetStateIndependant(),
		st,
		cb,
		make([]cmd.Key, 0),
		e,
	}

	c := make(chan error, 1)

	go func() {
		// Start hints mode
		var err error
		var nHints int64
		switch hm.Substate {
		case HintsSubstateFollow:
			nHints, err = hm.HintsCallback.ClickHintsMode()
		case HintsSubstateBackground,
			HintsSubstateRapid,
			HintsSubstateTab,
			HintsSubstateWindow:

			nHints, err = hm.HintsCallback.LinkHintsMode()
		case HintsSubstateSearchEngine:
			nHints, err = hm.HintsCallback.FormVariableHintsMode()
		default:
			err = fmt.Errorf("Unknown hints type: %d", hm.Substate)
		}
		if err != nil {
			c <- err
		} else if nHints == 0 {
			c <- errors.New("No hints to follow")
		} else if nHints == -1 {
			c <- errors.New("Internal error displaying hints")
		} else if nHints == 1 {
			c <- nil
		}
		close(c)
	}()

	return hm, c
}

// ProcessKeyPress processes exactly one key press in hints mode.
//
// It returns the new state, and whether the key press was swallowed or not.
func (s *HintsMode) ProcessKeyPress(key cmd.RealKey) (cmd.State, bool) {
	var newKeys []cmd.Key
	switch key.Keyval {
	// TODO maybe do something on enter. For now, just end hints mode.
	// TODO maybe handle tab.
	case cmd.KeyReturn, cmd.KeyKPEnter, cmd.KeyEscape:
		return cmd.NewNormalMode(s), true
	case cmd.KeyBackSpace:
		if len(s.CurrentKeys) == 0 {
			return cmd.NewNormalMode(s), true
		}
		newKeys = s.CurrentKeys[:len(s.CurrentKeys)-1]
	default:
		newKeys = cmd.ImmutableAppend(s.CurrentKeys, key)
	}
	ret := &HintsMode{
		s.StateIndependant,
		s.Substate,
		s.HintsCallback,
		newKeys,
		s.ExecuterFunction,
	}
	go func() {
		hitAndEnd, err := s.HintsCallback.FilterHintsMode(cmd.KeysString(newKeys))
		if err != nil {
			log.Printf("Failed to filter hints: %v", err)
		} else if hitAndEnd {
			sNew := s.GetState()
			if sNew == s || sNew == ret {
				s.StateIndependant.SetState(cmd.NewNormalMode(s))
			}
		}
	}()
	return ret, true
}

// GetStateIndependant gets the state independant associated with this state.
func (s *HintsMode) GetStateIndependant() *cmd.StateIndependant {
	return s.StateIndependant
}

// GetSubstate gets the substate associated with this state.
func (s *HintsMode) GetSubstate() cmd.Substate {
	return s.Substate
}
