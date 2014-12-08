package cmd

import (
	"log"
	"regexp"

	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"

	"github.com/tkerber/golem/cfg"
	"github.com/tkerber/golem/ui"
	"github.com/tkerber/golem/webkit"
)

// A Handler is a collection of cannels golem requires to communicate with
// the Cmd routine.
type Handler struct {
	// Channel to pass new key codes through when they are pressed.
	KeyPressHandle chan uint
	// Channel through which the CmdHandler returns whether to swallow the
	// keypress or not
	KeyPressSwallowChan chan bool
	// The user interface associated with this command handler
	UI *ui.UI
	// The DBus object connected to the active webview.
	dbus *dbus.Object
	// The currect command string
	cmdStr string
	// The currect state
	state cmdState
	// The current subtree which is being mapped
	mappingTree *mappingTree
	// The global settings
	settings *cfg.Settings
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
	":":  "_enter_command_mode",
	"i":  "mode insert",
	"o":  "_start_command open ",
	"r":  "reload",
	"gg": "scroll_to top",
	"G":  "scroll_to bottom",
	".j": "scroll left",
	"k":  "scroll down",
	"l":  "scroll up",
	".;": "scroll right",
	",j": "back",
	",;": "forward",
})

func NewHandler(ui *ui.UI, cfg *cfg.Settings, dbus *dbus.Object) *Handler {
	return &Handler{
		make(chan uint),
		make(chan bool),
		ui,
		dbus,
		"",
		normalMode,
		rootMappingTree,
		cfg,
	}
}

func (c *Handler) setState(s cmdState) {
	c.state = s
	c.updateStatusBar()
}

func (c *Handler) updateStatusBar() {
	var status string
	switch c.state {
	case normalMode:
		status = ""
	case commandMode:
		status = ":" + c.cmdStr
	case partialMappingMode:
		status = "[" + c.cmdStr + "]"
	case insertMode:
		status = "--INSERT--"
	}
	c.UI.SetCmdStatus(status)
}

// Run runs the command handler, which listens for keypresses and converts
// them into instructions for golem.
func (c *Handler) Run() {
	for {
		c.updateStatusBar()
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
						c.setState(partialMappingMode)
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
				cmd := c.cmdStr
				// We don't fallthrough to do that, as the command must be
				// run *after* the mode set.
				c.cmdStr = ""
				c.setState(normalMode)
				c.RunCmd(cmd)
			case escapeKey:
				c.cmdStr = ""
				c.setState(normalMode)
			case backSpaceKey:
				c.cmdStr = c.cmdStr[:len(c.cmdStr)-1]
				if len(c.cmdStr) == 0 {
					c.setState(normalMode)
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
						c.setState(normalMode)
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
			c.setState(normalMode)
			// TODO
		case insertMode:
			// Swallow *no* characters. Break off insert mode for *only*
			// the escape character.
			c.KeyPressSwallowChan <- false
			if keycode == escapeKey {
				c.setState(normalMode)
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
		if regexp.MustCompile("\\w+:.*").MatchString(uri) {
			// We have a (hopefully) sensable protocol already. keep it.
		} else if regexp.MustCompile("\\S+\\.\\S+").MatchString(uri) {
			// What we have looks like a uri, but is missing the protocol.
			// We add http to it.

			// TODO any good way to have this sensibly default to https where
			// possible?
			uri = "http://" + uri
		} else {
			searchEngine := c.settings.DefaultSearchEngine
			splitSearch := regexp.MustCompile("\\s+").Split(uri, 2)
			s, ok := c.settings.SearchEngines[splitSearch[0]]
			var searchTerm string
			if len(splitSearch) > 1 && ok {
				searchEngine = s
				searchTerm = splitSearch[1]
			} else {
				searchTerm = splitSearch[0]
			}
			uri = searchEngine.SearchURI(searchTerm)
		}
		//log.Printf(uri)
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
			c.setState(normalMode)
		case "insert":
			c.setState(insertMode)
		default:
			log.Printf("Attempted to access invalid mode: \"%v\"", splitCmd[1])
			return
		}
		c.cmdStr = ""
	case "_enter_command_mode":
		c.cmdStr = ""
		c.setState(commandMode)
	case "_start_command":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		c.cmdStr = splitCmd[1]
		c.setState(commandMode)
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
		case "left":
			hDelta = -40
		case "right":
			hDelta = 40
		default:
			log.Printf("Unknown scroll direction: \"%v\"", splitCmd[1])
			return
		}
		if vDelta != 0 {
			c.dbus.Go(
				"com.github.tkerber.golem.WebExtension.ScrollDelta",
				dbus.FlagNoReplyExpected,
				nil,
				int64(vDelta),
				true)
		}
		if hDelta != 0 {
			c.dbus.Go(
				"com.github.tkerber.golem.WebExtension.ScrollDelta",
				dbus.FlagNoReplyExpected,
				nil,
				int64(hDelta),
				false)
		}
	case "scroll_to":
		if len(splitCmd) < 2 {
			log.Printf("Not enough arguments for command: \"%v\"", cmd)
			return
		}
		switch splitCmd[1] {
		case "top":
			c.dbus.Go(
				"com.github.tkerber.golem.WebExtension.ScrollToTop",
				dbus.FlagNoReplyExpected,
				nil)
		case "bottom":
			c.dbus.Go(
				"com.github.tkerber.golem.WebExtension.ScrollToBottom",
				dbus.FlagNoReplyExpected,
				nil)
		default:
			log.Printf("Unknown scroll direction: \"%v\"", splitCmd[1])
			return
		}
	case "back":
		c.UI.WebView.GoBack()
	case "forward":
		c.UI.WebView.GoForward()
	case "quit":
		gtk.MainQuit()
	default:
		log.Printf("Unknown command: \"%v\"", cmd)
	}
}
