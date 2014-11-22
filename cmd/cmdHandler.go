package cmd

import "regexp"
import "strings"
import "log"
import "github.com/tkerber/golem/webkit"
import "github.com/tkerber/golem/ui"
import "github.com/tkerber/golem/ipc"

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
	//InstructionChan chan Instruction
	// Channel through which to pass updates to the status bar.
	//StatusChan chan string

	// The user interface associated with this command handler
	UI *ui.UI
	// The currect command string
	cmdStr string
	// The currect state
	state cmdState
	// The current subtree which is being mapped
	mappingTree *mappingTree
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
	"i":       "mode insert",
	"o":       "_start_command open ",
	"r":       "reload",
	"k":       "scroll down",
	"l":       "scroll up",
	"command": "_enter_command_mode",
})

func NewHandler(ui *ui.UI) *Handler {
	return &Handler{
		make(chan uint),
		make(chan bool),
		ui,
		"",
		normalMode,
		rootMappingTree,
	}
}

// Run runs the command handler, which listens for keypresses and converts
// them into instructions for golem.
func (c *Handler) Run() {
	for {
		c.UI.SetCmdStatus(c.cmdStr)
		keycode := <-c.KeyPressHandle
		//log.Printf("%v [%v]", keycode, KeyvalName(keycode))
		switch c.state {
		case normalMode:
			r := KeyvalToUnicode(keycode)
			if r != 0 {
				subtree, ok := rootMappingTree.subtree(r)
				if ok {
					cmd, ok := subtree.command()
					if ok {
						c.RunCmd(cmd)
					} else {
						c.mappingTree = subtree
						c.state = partialMappingMode
						c.cmdStr = string(r)
					}
					c.KeyPressSwallowChan <- true
					continue
				}
			}
			c.KeyPressSwallowChan <- false
		case commandMode:
			c.KeyPressSwallowChan <- true
			switch keycode {
			case returnKey:
				cmd := c.cmdStr[1:]
				// We don't fallthrough to do that, as the command must be
				// run *after* the mode set.
				c.cmdStr = ""
				c.state = normalMode
				c.RunCmd(cmd)
			case escapeKey:
				c.cmdStr = ""
				c.state = normalMode
			case backSpaceKey:
				c.cmdStr = c.cmdStr[:len(c.cmdStr)-1]
				if len(c.cmdStr) == 0 {
					c.state = normalMode
				}
			default:
				r := KeyvalToUnicode(keycode)
				if r != 0 {
					c.cmdStr += string(r)
				}
			}
		case partialMappingMode:
			// TODO duplication from normalMode, not handling commands which
			// start with other commands properly (eg. k, kk)
			r := KeyvalToUnicode(keycode)
			if r != 0 {
				subtree, ok := c.mappingTree.subtree(r)
				if ok {
					cmd, ok := subtree.command()
					if ok {
						c.cmdStr = ""
						c.state = normalMode
						c.RunCmd(cmd)
					} else {
						c.mappingTree = subtree
						c.cmdStr += string(r)
					}
					c.KeyPressSwallowChan <- true
					continue
				}
			}
			// If no matching mappings are available, don't swallow the
			// character and silently break off partialMappingMode.
			c.KeyPressSwallowChan <- false
			c.cmdStr = ""
			c.state = normalMode
			// TODO
		case insertMode:
			// Swallow *no* characters. Break off insert mode for *only*
			// the escape character.
			c.KeyPressSwallowChan <- false
			if keycode == escapeKey {
				c.state = normalMode
			}
		}
	}
}

// RunCmd runs a command.
func (c *Handler) RunCmd(cmd string) {
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
		c.UI.WebView.LoadURI(uri)
	case "reload":
		c.UI.WebView.Reload()
	case "mode":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		switch splitCmd[1] {
		case "normal":
			c.state = normalMode
		case "insert":
			c.state = insertMode
		default:
			log.Printf("Attempted to access invalid mode: \"%v\"", splitCmd[1])
			return
		}
		c.cmdStr = ""
	case "_enter_command_mode":
		c.cmdStr = ":"
		c.state = commandMode
	case "_start_command":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		c.cmdStr = ":" + splitCmd[1]
		c.state = commandMode
	case "scroll":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		vDelta := 0
		hDelta := 0
		switch splitCmd[1] {
		case "up":
			vDelta = -40
		case "down":
			vDelta = 40
		default:
			log.Printf("Unknown scroll direction: \"%v\"", splitCmd[1])
			return
		}
		if vDelta != 0.0 {
			err := ipc.ScrollDown(vDelta)
			if err != nil {
				log.Printf("Failed to initiate IPC for scrolling: \"%v\"", err)
			}
		}
		if hDelta != 0.0 {
			err := ipc.ScrollRight(hDelta)
			if err != nil {
				log.Printf("Failed to initiate IPC for scrolling: \"%v\"", err)
			}
		}
	default:
		log.Printf("Unknown command: \"%v\"", cmd)
	}
}
