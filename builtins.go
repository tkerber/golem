package main

import (
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
)

func builtinsFor(w *window) cmd.Builtins {
	return cmd.Builtins{
		"commandMode":    w.builtinCommandMode,
		"editURI":        w.builtinEditURI,
		"goBack":         w.builtinGoBack,
		"goForward":      w.builtinGoForward,
		"goHome":         w.builtinGoHome,
		"insertMode":     w.builtinInsertMode,
		"nop":            w.builtinNop,
		"open":           w.builtinOpen,
		"reload":         w.builtinReload,
		"runCmd":         w.builtinRunCmd,
		"scrollDown":     w.builtinScrollDown,
		"scrollLeft":     w.builtinScrollLeft,
		"scrollRight":    w.builtinScrollRight,
		"scrollToBottom": w.builtinScrollToBottom,
		"scrollToTop":    w.builtinScrollToTop,
		"scrollUp":       w.builtinScrollUp,
	}
}

func (w *window) builtinCommandMode(_ ...interface{}) {
	w.setState(cmd.NewCommandLineMode(w.State, w.runCmd))
}

func (w *window) builtinEditURI(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		fmt.Sprintf("open %v", w.WebView.GetURI()),
		w.runCmd))
}

func (w *window) builtinGoBack(_ ...interface{}) {
	w.WebView.GoBack()
}

func (w *window) builtinGoForward(_ ...interface{}) {
	w.WebView.GoForward()
}

func (w *window) builtinGoHome(_ ...interface{}) {
	w.runCmd(fmt.Sprintf("open %v", w.parent.homePage))
}

func (w *window) builtinInsertMode(_ ...interface{}) {
	w.setState(cmd.NewInsertMode(w.State))
}

// builtinNop does nothing. It is occasionally useful as a binding.
func (w *window) builtinNop(_ ...interface{}) {}

func (w *window) builtinOpen(_ ...interface{}) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, "open ", w.runCmd))
}

func (w *window) builtinReload(_ ...interface{}) {
	w.WebView.Reload()
}

func (w *window) builtinRunCmd(args ...interface{}) {
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
}

func (w *window) builtinScrollDown(_ ...interface{}) {
	w.scrollDelta(w.parent.scrollDelta, true)
}

func (w *window) builtinScrollLeft(_ ...interface{}) {
	w.scrollDelta(-w.parent.scrollDelta, false)
}

func (w *window) builtinScrollRight(_ ...interface{}) {
	w.scrollDelta(w.parent.scrollDelta, false)
}

func (w *window) builtinScrollToBottom(_ ...interface{}) {
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

func (w *window) builtinScrollToTop(_ ...interface{}) {
	err := w.getWebView().setScrollTop(0)
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
}

func (w *window) builtinScrollUp(_ ...interface{}) {
	w.scrollDelta(-w.parent.scrollDelta, true)
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
