package cmd

import "strings"
import "github.com/tkerber/golem/webkit"

// A Handler is a collection of cannels golem requires to communicate with
// the Cmd routine.
type Handler struct {
	// Channel to pass new key codes through when they are pressed.
	KeyPressHandle chan uint
	// Channel through which the CmdHandler returns whether to swallow the
	// keypress or not
	KeyPressSwallowChan chan bool
	// Channel through which the CmdHandler passes final Instruction objects
	// to the main thread.
	InstructionChan chan Instruction
}

type Instruction func(*webkit.WebView) error

// Run runs the command handler, which listens for keypresses and converts
// them into instructions for golem.
func (c *Handler) Run() {
	cmdStr := ""
	for {
		keycode := <-c.KeyPressHandle
		if len(cmdStr) == 0 {
			if keycode == colonKey {
				c.KeyPressSwallowChan <- true
				cmdStr += ":"
				continue
			}
		} else {
			if keycode == escapeKey {
				c.KeyPressSwallowChan <- true
				cmdStr = ""
				continue
			} else if keycode == returnKey {
				c.KeyPressSwallowChan <- true
				go c.runCmd(cmdStr[1:])
				cmdStr = ""
				continue
			} else {
				c.KeyPressSwallowChan <- true
				r := KeyvalToUnicode(keycode)
				if r != 0 {
					cmdStr += string(r)
				}
				continue
			}
		}
		c.KeyPressSwallowChan <- false
	}
}

func (c *Handler) runCmd(cmd string) {
	if strings.HasPrefix(cmd, "open ") {
		c.InstructionChan <- func(w *webkit.WebView) error {
			w.LoadURI(cmd[5:])
			return nil
		}
	}
}
