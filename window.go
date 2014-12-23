package main

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/debug"
	"github.com/tkerber/golem/ui"
)

type signalHandle struct {
	obj    *glib.Object
	handle glib.SignalHandle
}

func (h signalHandle) disconnect() {
	h.obj.HandlerDisconnect(h.handle)
}

type window struct {
	*ui.Window
	cmd.State
	webViews            []*webView
	parent              *golem
	builtins            cmd.Builtins
	bindings            *cmd.BindingTree
	activeSignalHandles []signalHandle
}

const keyTimeout = time.Millisecond * 10

// nop does nothing. It is occasionally useful as a binding.
func (w *window) nop() {}

func (w *window) setState(state cmd.State) {
	w.State = state
	w.UpdateState(w.State)
}

func (g *golem) newWindow() error {
	wv, err := g.newWebView()
	if err != nil {
		log.Printf("Error: Failed to open new window: %v\n", err)
		return err
	}

	uiWin, err := ui.NewWindow(wv.WebView)
	if err != nil {
		log.Printf("Error: Failed to open new window: %v\n", err)
		return err
	}

	win := &window{
		uiWin,
		nil,
		[]*webView{
			wv,
		},
		g,
		nil,
		new(cmd.BindingTree),
		make([]signalHandle, 0),
	}

	win.builtins = builtinsFor(win)

	win.setState(cmd.NewNormalMode(&cmd.StateIndependant{
		win.bindings,
		win.setState,
	}))

	win.rebuildBindings()

	g.wMutex.Lock()
	g.windows = append(g.windows, win)
	g.wMutex.Unlock()

	win.reconnectWebViewSignals()

	// Due to a bug with keypresses registering multiple times, we ignore
	// keypresses within 10ms of each other.
	// After each keypress, true gets sent to this channel 10ms after.
	timeoutChan := make(chan bool, 1)
	timeoutChan <- true

	uiWin.Window.Connect("key-press-event", func(w *gtk.Window, e *gdk.Event) bool {
		select {
		case <-timeoutChan:
			// Make sure that the timeout is properly applied.
			defer func() {
				go func() {
					<-time.After(keyTimeout)
					timeoutChan <- true
				}()
			}()
			// This conversion *shouldn't* be unsafe, BUT we really don't want
			// crashes here. TODO
			ek := gdk.EventKey{e}
			key := cmd.NewKeyFromEventKey(ek)
			if debug.PrintKeys {
				log.Printf("%v", key)
			}
			// We ignore modifier keys.
			if key.IsModifier {
				return false
			}

			oldState := win.State
			newState, ret := win.State.ProcessKeyPress(key)
			// If this is not the case, a state change command was issued. This
			// takes precedence.
			if oldState == win.State {
				win.setState(newState)
			}
			return ret
		default:
			return false
		}
	})
	uiWin.Window.Connect("destroy", func() {
		for _, wv := range win.webViews {
			wv.close()
		}
		g.closeWindow(win)
	})

	// Load the start page
	win.builtins["goHome"]()

	win.Show()
	return nil
}

func (w *window) rebuildBindings() {
	bindings, err := cmd.ParseRawBindings(w.parent.rawBindings, w.builtins)
	if err != nil {
		log.Printf("Error: Failed to parse key bindings: %v\n", err)
	}
	bindingTree, err := cmd.NewBindingTree(bindings)
	if err != nil {
		log.Printf("Error: Failed to parse key bindings: %v\n", err)
	}
	*(w.bindings) = *bindingTree
}

func (w *window) getWebView() *webView {
	return w.parent.webViews[w.WebView.GetPageID()]
}

func (w *window) reconnectWebViewSignals() {
	for _, handle := range w.activeSignalHandles {
		handle.disconnect()
	}
	w.activeSignalHandles = make([]signalHandle, 3)
	handle, err := w.WebView.Connect("notify::title", func() {
		title := w.GetTitle()
		if title != "" {
			w.SetTitle(fmt.Sprintf("%s - Golem", title))
		} else {
			w.SetTitle("Golem")
		}
	})
	if err != nil {
		panic("Failed to connect to window event.")
	}
	w.activeSignalHandles[0] = signalHandle{w.WebView.Object, handle}
	handle, err = w.WebView.Connect("notify::uri", w.UpdateLocation)
	if err != nil {
		panic("Failed to connect to window event.")
	}
	w.activeSignalHandles[1] = signalHandle{w.WebView.Object, handle}
	bfl := w.WebView.GetBackForwardList()
	handle, err = bfl.Connect("changed", w.UpdateLocation)
	w.activeSignalHandles[2] = signalHandle{bfl.Object, handle}
}

func (w *window) runCmd(cmd string) {
	runCmd(w, w.parent, cmd)
}

func runCmd(w *window, g *golem, cmd string) {
	// Space followed optionally by a line comment (starting with ")
	blankRegex := regexp.MustCompile(`^\s*(".*|)$`)
	if blankRegex.MatchString(cmd) {
		return
	}

	splitRegex := regexp.MustCompile(`\s+`)
	parts := splitRegex.Split(cmd, -1)
	if len(parts[0]) == 0 {
		parts = parts[1:len(parts)]
	}
	f, ok := commands[parts[0]]
	if ok {
		if debug.PrintCommands {
			log.Printf("Running command '%v'.", cmd)
		}
		f(w, g, parts)
	} else {
		log.Printf("Failed to run command '%v': No such command.", cmd)
	}
}
