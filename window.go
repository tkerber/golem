package main

import (
	"fmt"
	"log"
	"time"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/ui"
	"github.com/tkerber/golem/webkit"
)

type window struct {
	*ui.Window
	cmd.State
}

const keyTimeout = time.Millisecond * 10

// nop does nothing. It is occasionally useful as a binding.
func (w *window) nop() {}

func (w *window) setState(state cmd.State) {
	w.State = state
}

func (g *golem) newWindow() error {
	webView, err := webkit.NewWebViewWithUserContentManager(g.userContentManager)
	if err != nil {
		log.Printf("Error: Failed to open new window: %v\n", err)
		return err
	}

	uiWin, err := ui.NewWindow(webView)
	if err != nil {
		log.Printf("Error: Failed to open new window: %v\n", err)
		return err
	}

	win := &window{uiWin, nil}

	builtins := builtinsFor(win)
	bindings, err := cmd.ParseRawBindings(defaultBindings, builtins)
	if err != nil {
		log.Printf("Error: Failed to parse key bindings: %v\n", err)
		return err
	}
	bindingTree, err := cmd.NewBindingTree(bindings)
	if err != nil {
		log.Printf("Error: Failed to parse key bindings: %v\n", err)
		return err
	}

	win.State = cmd.NewNormalMode(&cmd.StateIndependant{
		bindingTree,
		win.setState,
	})

	g.openChan <- win

	uiWin.WebView.Connect("notify::title", func() {
		title := win.GetTitle()
		if title != "" {
			win.SetTitle(fmt.Sprintf("%s - Golem", title))
		} else {
			win.SetTitle("Golem")
		}
	})
	uiWin.WebView.Connect("notify::uri", win.UpdateLocation)
	bfl := webView.GetBackForwardList()
	bfl.Connect("changed", win.UpdateLocation)

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
			log.Printf("%v", key)
			// We ignore modifier keys.
			if key.IsModifier {
				return false
			}

			oldState := win.State
			newState, ret := win.State.ProcessKeyPress(key)
			// If this is not the case, a state change command was issued. This
			// takes precedence.
			if oldState == win.State {
				win.State = newState
			}
			return ret
		default:
			return false
		}
	})
	uiWin.Window.Connect("destroy", func() {
		g.closeChan <- win
	})

	win.Show()
	return nil
}
