package main

import (
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
)

// builtinsfor retrieves the builtin functions bound to a specific window.
func builtinsFor(w *window) cmd.Builtins {
	return cmd.Builtins{
		"commandMode":    w.builtinCommandMode,
		"editURI":        w.builtinEditURI,
		"goBack":         w.builtinGoBack,
		"goForward":      w.builtinGoForward,
		"insertMode":     w.builtinInsertMode,
		"nop":            w.builtinNop,
		"open":           w.builtinOpen,
		"panic":          w.builtinPanic,
		"reload":         w.builtinReload,
		"reloadNoCache":  w.builtinReloadNoCache,
		"scrollDown":     w.builtinScrollDown,
		"scrollLeft":     w.builtinScrollLeft,
		"scrollRight":    w.builtinScrollRight,
		"scrollToBottom": w.builtinScrollToBottom,
		"scrollToTop":    w.builtinScrollToTop,
		"scrollUp":       w.builtinScrollUp,
		"tabClose":       w.builtinTabClose,
		"tabEditURI":     w.builtinTabEditURI,
		"tabGo":          w.builtinTabGo,
		"tabNext":        w.builtinTabNext,
		"tabOpen":        w.builtinTabOpen,
		"tabPrev":        w.builtinTabPrev,
		"windowEditURI":  w.builtinWindowEditURI,
		"windowOpen":     w.builtinWindowOpen,
	}
}

// min returns the smallest of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the largest of two integers
func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// getWithDefault gets the integer stored in a pointer, or def if it is nil.
func getWithDefault(ptr *int, def, minv, maxv int) int {
	if ptr == nil {
		return def
	}
	return max(min(*ptr, maxv), minv)
}

// builtinCommandMode initiates command mode.
func (w *window) builtinCommandMode(_ *int) {
	w.setState(cmd.NewCommandLineMode(w.State, w.runCmd))
}

// builtinEditURI initiates command mode with the open command primed for
// the current URI.
func (w *window) builtinEditURI(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		fmt.Sprintf("open %v", w.WebView.GetURI()),
		w.runCmd))
}

// builtinGoBack goes one step back in browser history.
func (w *window) builtinGoBack(n *int) {
	for num := getWithDefault(n, 1, 0, 50); num > 0 && w.WebView.CanGoBack(); num-- {
		w.WebView.GoBack()
	}
}

// builtinGoForward goes one step forward in browser history.
func (w *window) builtinGoForward(n *int) {
	for num := getWithDefault(n, 1, 0, 50); num > 0 && w.WebView.CanGoForward(); num-- {
		w.WebView.GoForward()
	}
}

// builtinInsertMode initiates insert mode.
func (w *window) builtinInsertMode(_ *int) {
	w.setState(cmd.NewInsertMode(w.State))
}

// builtinNop does nothing. It is occasionally useful as a binding.
func (w *window) builtinNop(_ *int) {}

// builtinOpen initiates command mode, primed with an open command.
func (w *window) builtinOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, "open ", w.runCmd))
}

func (w *window) builtinPanic(_ *int) {
	panic("Builtin 'panic' called.")
}

// builtinReload reloads the current page.
func (w *window) builtinReload(_ *int) {
	w.WebView.Reload()
}

// builtinReloadNoCache reloads the current page, bypassing the cache.
func (w *window) builtinReloadNoCache(_ *int) {
	w.WebView.ReloadBypassCache()
}

// builtinScrollDown scrolls down.
func (w *window) builtinScrollDown(n *int) {
	w.scrollDelta(w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), true)
}

// builtinScrollLeft scrolls left.
func (w *window) builtinScrollLeft(n *int) {
	w.scrollDelta(-w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), false)
}

// builtinScrollRight scrolls right.
func (w *window) builtinScrollRight(n *int) {
	w.scrollDelta(w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), false)
}

// builtinScrollToBottom scrolls to the bottom of the page.
func (w *window) builtinScrollToBottom(_ *int) {
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

// builtinScrollTotop scrolls to the top of the page.
func (w *window) builtinScrollToTop(_ *int) {
	err := w.getWebView().setScrollTop(0)
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
}

// builtinScrollUp scrolls up.
func (w *window) builtinScrollUp(n *int) {
	w.scrollDelta(-w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), true)
}

// builtinTabClose closes the current tab.
func (w *window) builtinTabClose(n *int) {
	num := getWithDefault(n, 1, 0, len(w.webViews))
	for i := 0; i < num; i++ {
		w.tabClose()
	}
}

// builtinTabEditURI initiates command mode with a tabopen command primed for
// the current URI.
func (w *window) builtinTabEditURI(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		fmt.Sprintf("tabopen %v", w.GetURI()),
		w.runCmd))
}

// builtinTabGo goes to the specified tab.
func (w *window) builtinTabGo(n *int) {
	num := getWithDefault(n, 1, 1, len(w.webViews))
	w.tabGo(num - 1)
}

// builtinTabNext goes to the next tab.
func (w *window) builtinTabNext(n *int) {
	num := getWithDefault(n, 1, 0, 1<<20)
	size := len(w.webViews)
	newTab := (w.currentWebView + num) % size
	// Banish all ye negative modulo results.
	if newTab < 0 {
		newTab += size
	}
	w.tabGo(newTab)
}

// builtinTabOpen initiates command mode primed with a tabopen command.
func (w *window) builtinTabOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, "tabopen ", w.runCmd))
}

// builtinTabPrev goes to the previous tab.
func (w *window) builtinTabPrev(n *int) {
	num := getWithDefault(n, 1, 0, 1<<20)
	size := len(w.webViews)
	newTab := (w.currentWebView - num) % size
	// Banish all ye negative modulo results.
	if newTab < 0 {
		newTab += size
	}
	w.tabGo(newTab)
}

// builtinWindowEditURI initiates command mode with a winopen command primed
// for the current URI.
func (w *window) builtinWindowEditURI(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		fmt.Sprintf("winopen %v", w.GetURI()),
		w.runCmd))
}

// builtinWindowOpen initiates command mode primed with a winopen command.
func (w *window) builtinWindowOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, "winopen ", w.runCmd))
}

// scrollDelta scrolls a given amount of pixes either vertically or
// horizontally.
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
