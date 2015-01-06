// Package states defines constants of the substates used in golem's state
// machine.
package states

import "github.com/tkerber/golem/cmd"

const (
	// NormalSubstateNormal indicates "normal mode"
	NormalSubstateNormal cmd.Substate = iota
	// NormalSubstateQuickmark indicates quickmark bindings mode.
	NormalSubstateQuickmark
	// NormalSubstateQuickmarkTab indicates quickmark bindings mode opening in
	// a new tab.
	NormalSubstateQuickmarkTab
	// NormalSubstateQuickmarkWindow indicates quickmark bindings mode opening
	// in a new window.
	NormalSubstateQuickmarkWindow
	// NormalSubstateQuickmarksRapid indicates quickmark bindings mode opening
	// several quickmarks in background tabs.
	NormalSubstateQuickmarksRapid
)

const (
	// CommandLineSubstateCommand indicates a command being entered.
	CommandLineSubstateCommand cmd.Substate = iota
	// CommandLineSubstateSearch indicates a search being entered.
	CommandLineSubstateSearch
	// CommandLineSubstateBackSearch indicates a backwards search being entered.
	CommandLineSubstateBackSearch
)

const (
	// StatusSubstateMinor indicates a minor, inconsequential status.
	StatusSubstateMinor cmd.Substate = iota
	// StatusSubstateMajor indicates a more important status.
	StatusSubstateMajor
	// StatusSubstateError indicates an error status.
	StatusSubstateError
)
