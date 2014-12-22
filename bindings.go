package main

import (
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
)

type binding struct {
	from string
	to   string
}

func builtinsFor(w *window) cmd.Builtins {
	return cmd.Builtins{
		"nop":        w.nop,
		"insertMode": func() { w.setState(cmd.NewInsertMode(w.State)) },
		"commandMode": func() {
			w.setState(cmd.NewCommandLineMode(w.State, w.runCmd))
		},
		"open": func() {
			w.setState(cmd.NewPartialCommandLineMode(w.State, "open ", w.runCmd))
		},
		"goHome": func() {
			w.runCmd(fmt.Sprintf("open %v", w.parent.homePage))
		},
		"scrollToBottom": w.scrollToBottom,
		"scrollToTop":    w.scrollToTop,
		"scrollUp":       func() { w.scrollDelta(-w.parent.scrollDelta, true) },
		"scrollDown":     func() { w.scrollDelta(w.parent.scrollDelta, true) },
		"scrollLeft":     func() { w.scrollDelta(-w.parent.scrollDelta, false) },
		"scrollRight":    func() { w.scrollDelta(w.parent.scrollDelta, false) },
		"goBack": func() {
			w.WebView.GoBack()
		},
		"goForward": func() {
			w.WebView.GoForward()
		},
		"reload": func() { w.WebView.Reload() },
		"editURI": func() {
			w.setState(cmd.NewPartialCommandLineMode(
				w.State,
				fmt.Sprintf("open %v", w.WebView.GetURI()),
				w.runCmd))
		},
	}
}

func (w *window) scrollToBottom() {
	ext := w.getWebView()
	height, err := ext.getScrollHeight()
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
	err = ext.setScrollTop(height)
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
}

func (w *window) scrollToTop() {
	err := w.getWebView().setScrollTop(0)
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
}

func (w *window) scrollDelta(delta int, vertical bool) {
	var curr int64
	var err error
	wv := w.getWebView()
	if vertical {
		curr, err = wv.getScrollTop()
	} else {
		curr, err = wv.getScrollLeft()
	}
	if err != nil {
		log.Printf("Error scrolling: %v", err)
		return
	}
	curr += int64(delta)
	if vertical {
		err = wv.setScrollTop(curr)
	} else {
		err = wv.setScrollLeft(curr)
	}
	if err != nil {
		log.Printf("Error scrolling: %v", err)
		return
	}
}
