package main

import (
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
)

func builtinsFor(w *window) cmd.Builtins {
	return cmd.Builtins{
		"nop":        w.nop,
		"insertMode": func(args ...interface{}) { w.setState(cmd.NewInsertMode(w.State)) },
		"commandMode": func(args ...interface{}) {
			w.setState(cmd.NewCommandLineMode(w.State, w.runCmd))
		},
		"open": func(args ...interface{}) {
			w.setState(cmd.NewPartialCommandLineMode(w.State, "open ", w.runCmd))
		},
		"goHome": func(args ...interface{}) {
			w.runCmd(fmt.Sprintf("open %v", w.parent.homePage))
		},
		"scrollToBottom": w.scrollToBottom,
		"scrollToTop":    w.scrollToTop,
		"scrollUp":       func(args ...interface{}) { w.scrollDelta(-w.parent.scrollDelta, true) },
		"scrollDown":     func(args ...interface{}) { w.scrollDelta(w.parent.scrollDelta, true) },
		"scrollLeft":     func(args ...interface{}) { w.scrollDelta(-w.parent.scrollDelta, false) },
		"scrollRight":    func(args ...interface{}) { w.scrollDelta(w.parent.scrollDelta, false) },
		"goBack": func(args ...interface{}) {
			w.WebView.GoBack()
		},
		"goForward": func(args ...interface{}) {
			w.WebView.GoForward()
		},
		"reload": func(args ...interface{}) { w.WebView.Reload() },
		"editURI": func(args ...interface{}) {
			w.setState(cmd.NewPartialCommandLineMode(
				w.State,
				fmt.Sprintf("open %v", w.WebView.GetURI()),
				w.runCmd))
		},
		"runCmd": func(args ...interface{}) {
			if len(args) < 1 {
				log.Printf("Failed to execute builtin 'runCmd': Not enough arguments")
				return
			}
			cmd, ok := args[0].(string)
			if !ok {
				log.Printf(
					"Invalid type for argument for builtin 'runCmd': %T",
					args[0])
			}
			w.runCmd(cmd)
		},
	}
}

func (w *window) scrollToBottom(args ...interface{}) {
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

func (w *window) scrollToTop(args ...interface{}) {
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
