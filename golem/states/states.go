// Package states defines constants of the substates used in golem's state
// machine.
package states

import "github.com/tkerber/golem/cmd"

const (
	// StatusSubstateMinor indicates a minor, inconsequential status.
	StatusSubstateMinor cmd.Substate = 1 + iota
	// StatusSubstateMajor indicates a more important status.
	StatusSubstateMajor
	// StatusSubstateError indicates an error status.
	StatusSubstateError
)
