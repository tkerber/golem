// Package debug contains constants used to control to which degree debugging
// data is printed.
package debug

import "log"

// Whether or not to print each keypress.
const PrintKeys = false

// Whether or not to print each command when it is executed.
const PrintCommands = true

// Whether or not to print each binding when it is executed.
const PrintBindings = false

// Whether or not to log line numbers.
const LogLineNumbers = true

func init() {
	if LogLineNumbers {
		log.SetFlags(log.Flags() | log.Lshortfile)
	}
}
