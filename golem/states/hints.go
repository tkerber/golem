package states

import "github.com/tkerber/golem/cmd"

// HintsCallback is an interface for golem.(*webView), implementing the
// methods needed by hints. (mostly web extension calls)
type HintsCallback interface {
	LinkHintsMode()
	EndHintsMode()
	FilterHintsMode(string)
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

// ProcessKeyPress processes exactly one key press in hints mode.
//
// It returns the new state, and whether the key press was swallowed or not.
func (s *HintsMode) ProcessKeyPress(key cmd.RealKey) (cmd.State, bool) {
	switch key.Keyval {
	// TODO maybe do something on enter. For now, just end hints mode.
	// TODO maybe handle tab.
	case cmd.KeyReturn, cmd.KeyKPEnter, cmd.KeyEscape:
		s.HintsCallback.EndHintsMode()
		return cmd.NewNormalMode(s), true
	default:
		newKeys := cmd.ImmutableAppend(s.CurrentKeys, key)
		s.HintsCallback.FilterHintsMode(cmd.KeysString(newKeys))
		return &HintsMode{
			s.StateIndependant,
			s.Substate,
			s.HintsCallback,
			newKeys,
			s.ExecuterFunction,
		}, true
	}
}

// GetStateIndependant gets the state independant associated with this state.
func (s *HintsMode) GetStateIndependant() *cmd.StateIndependant {
	return s.StateIndependant
}

// GetSubstate gets the substate associated with this state.
func (s *HintsMode) GetSubstate() cmd.Substate {
	return s.Substate
}
