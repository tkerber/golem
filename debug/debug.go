package debug

import "log"

const PrintKeys = false
const PrintCommands = true
const PrintBindings = false
const LogLineNumbers = true

func init() {
	if LogLineNumbers {
		log.SetFlags(log.Flags() | log.Lshortfile)
	}
}
