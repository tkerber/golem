package cmd

import "log"
import "strings"

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

// An Instruction is a single atomic command for golem.
type Instruction interface {
	// This is a placeholder.

	// Get the instructions command (or nil if the instruction was internal)
	Command() string
}

type CommandInstruction struct {
	CommandStr string
}

func (i *CommandInstruction) Command() string {
	return i.CommandStr
}

type OpenInstruction struct {
	CommandInstruction
	Uri string
}

// Run runs the command handler, which listens for keypresses and converts
// them into instructions for golem.
func (c *Handler) Run() {
	cmdStr := ""
	for {
		keycode := <-c.KeyPressHandle
		//log.Printf("Keypress %x [%v] recieved!", keycode, KeyvalName(keycode))
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
				cmdStr += string(KeyvalToUnicode(keycode))
				continue
			}
		}
		c.KeyPressSwallowChan <- false
	}
}

func (c *Handler) runCmd(cmd string) {
	log.Printf("Command \"%v\" entered.", cmd)
	if strings.HasPrefix(cmd, "open ") {
		c.InstructionChan <- &OpenInstruction{CommandInstruction{cmd}, cmd[5:]}
	}
}
