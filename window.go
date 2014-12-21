package main

import (
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/ui"
	"github.com/tkerber/golem/webkit"
)

type window struct {
	*ui.Window
	cmd.State
}

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
	win.State = cmd.NewNormalMode(&cmd.StateIndependant{
		defaultBindings,
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

	uiWin.Window.Connect("destroy", func() {
		g.closeChan <- win
	})

	win.Show()
	return nil
}
