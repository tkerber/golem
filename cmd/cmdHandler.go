package cmd

import "regexp"
import "strings"
import "log"

//import "fmt"
import "github.com/tkerber/golem/webkit"

//import "github.com/conformal/gotk3/gtk"

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
	// Channel through which to pass updates to the status bar.
	StatusChan chan string
}

// An Instruction is a function to be called by the main event loop.
type Instruction func(*webkit.WebView) error

type cmdState int

const (
	normalMode cmdState = iota
	commandMode
	partialMappingMode
	insertMode
)

var rootMappingTree = compileMappingTree(map[string]string{
	":":       "_enter_command_mode",
	"i":       "mode insert_mode",
	"o":       "_start_command open ",
	"r":       "reload",
	"k":       "scroll down",
	"l":       "scroll up",
	"command": "_enter_command_mode",
})

// Run runs the command handler, which listens for keypresses and converts
// them into instructions for golem.
func (c *Handler) Run() {
	cmdStr := ""
	state := normalMode
	mapTreePos := rootMappingTree
	for {
		c.StatusChan <- cmdStr
		keycode := <-c.KeyPressHandle
		//log.Printf("%v [%v]", keycode, KeyvalName(keycode))
		switch state {
		case normalMode:
			r := KeyvalToUnicode(keycode)
			if r != 0 {
				subtree, ok := rootMappingTree.subtree(r)
				if ok {
					cmd, ok := subtree.command()
					if ok {
						c.runCmd(cmd, &cmdStr, &state)
					} else {
						mapTreePos = subtree
						state = partialMappingMode
						cmdStr = string(r)
					}
					c.KeyPressSwallowChan <- true
					continue
				}
			}
			//if keycode == colonKey {
			//	c.KeyPressSwallowChan <- true
			//	cmdStr += ":"
			//	state = commandMode
			//	continue
			//}
			c.KeyPressSwallowChan <- false
		case commandMode:
			c.KeyPressSwallowChan <- true
			switch keycode {
			case returnKey:
				cmd := cmdStr[1:]
				// We don't fallthrough to do that, as the command must be
				// run *after* the mode set.
				cmdStr = ""
				state = normalMode
				c.runCmd(cmd, &cmdStr, &state)
			case escapeKey:
				cmdStr = ""
				state = normalMode
			case backSpaceKey:
				cmdStr = cmdStr[:len(cmdStr)-1]
				if len(cmdStr) == 0 {
					state = normalMode
				}
			default:
				r := KeyvalToUnicode(keycode)
				if r != 0 {
					cmdStr += string(r)
				}
			}
		case partialMappingMode:
			// TODO duplication from normalMode, not handling commands which
			// start with other commands properly (eg. k, kk)
			r := KeyvalToUnicode(keycode)
			if r != 0 {
				subtree, ok := mapTreePos.subtree(r)
				if ok {
					cmd, ok := subtree.command()
					if ok {
						cmdStr = ""
						state = normalMode
						c.runCmd(cmd, &cmdStr, &state)
					} else {
						mapTreePos = subtree
						cmdStr += string(r)
					}
					c.KeyPressSwallowChan <- true
					continue
				}
			}
			// If no matching mappings are available, don't swallow the
			// character and silently break off partialMappingMode.
			c.KeyPressSwallowChan <- false
			cmdStr = ""
			state = normalMode
			// TODO
		case insertMode:
			// Swallow *no* characters. Break off insert mode for *only*
			// the escape character.
			c.KeyPressSwallowChan <- false
			if keycode == escapeKey {
				state = normalMode
			}
		}
	}
}

// runCmd runs a command.
func (c *Handler) runCmd(cmd string, cmdStr *string, state *cmdState) {
	splitCmd := regexp.MustCompile("\\s+").Split(cmd, 2)
	switch splitCmd[0] {
	case "open":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		uri := splitCmd[1]
		if !(strings.HasPrefix(uri, "http://") ||
			strings.HasPrefix(uri, "https://")) {
			uri = "http://" + uri
		}
		c.InstructionChan <- func(w *webkit.WebView) error {
			w.LoadURI(uri)
			return nil
		}
	case "reload":
		c.InstructionChan <- func(w *webkit.WebView) error {
			w.Reload()
			return nil
		}
	case "mode":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		switch splitCmd[1] {
		case "normal":
			*state = normalMode
		case "insert":
			*state = insertMode
		default:
			log.Printf("Attempted to access invalid mode: \"%v\"", splitCmd[1])
		}
	case "_enter_command_mode":
		*cmdStr = ":"
		*state = commandMode
	case "_start_command":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		*cmdStr = ":" + splitCmd[1]
		*state = commandMode
	case "scroll":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		vDelta := 0.0
		//hDelta := 0.0
		switch splitCmd[1] {
		case "up":
			vDelta = 10
		case "down":
			vDelta = -10
		default:
			log.Printf("Unknown scroll direction: \"%v\"", splitCmd[1])
			return
		}
		if vDelta != 0.0 {
			c.InstructionChan <- func(w *webkit.WebView) error {
				// TODO This will take some doing.
				// See http://stackoverflow.com/questions/21781868/scrolling-a-webkit2-webkit-window-in-gtk3
				// Problem is getting an instance of the DOM; but we *do* need
				// this. Create C WebExtension & figure out how to access it
				// from go (callbacks?)

				//log.Printf("%v", w.GetFocusChild())
				//var c *gtk.Container = w
				//var adj gtk.Adjustment
				//for {
				//	// This is ugly. fix it TODO
				//	adj = c.GetFocusVAdjustment()
				//	if adj == nil {
				//		c = c.GetFocusChild()
				//		if c == nil {
				//			return fmt.Errorf("No scrollable field found.")
				//		}
				//	}
				//}
				//log.Printf("%v", adj)
				//adj.Set("value", adj.GetDouble("value")+vDelta)
				//return nil
				return nil
			}
		}
	default:
		log.Printf("Unknown command: \"%v\"", cmd)
	}
}
