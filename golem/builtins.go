package golem

import (
	"fmt"
	"log"

	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
)

// builtinsfor retrieves the builtin functions bound to a specific window.
func builtinsFor(w *Window) cmd.Builtins {
	return cmd.Builtins{
		"commandMode":      w.builtinCommandMode,
		"editURI":          w.builtinEditURI,
		"goBack":           w.builtinGoBack,
		"goForward":        w.builtinGoForward,
		"insertMode":       w.builtinInsertMode,
		"nop":              w.builtinNop,
		"open":             w.builtinOpen,
		"panic":            w.builtinPanic,
		"quickmarks":       w.builtinQuickmarks,
		"quickmarksTab":    w.builtinQuickmarksTab,
		"quickmarksWindow": w.builtinQuickmarksWindow,
		"quickmarksRapid":  w.builtinQuickmarksRapid,
		"reload":           w.builtinReload,
		"reloadNoCache":    w.builtinReloadNoCache,
		"scrollDown":       w.builtinScrollDown,
		"scrollLeft":       w.builtinScrollLeft,
		"scrollRight":      w.builtinScrollRight,
		"scrollPageDown":   w.builtinScrollPageDown,
		"scrollPageUp":     w.builtinScrollPageUp,
		"scrollToBottom":   w.builtinScrollToBottom,
		"scrollToTop":      w.builtinScrollToTop,
		"scrollUp":         w.builtinScrollUp,
		"tabClose":         w.builtinTabClose,
		"tabEditURI":       w.builtinTabEditURI,
		"tabGo":            w.builtinTabGo,
		"tabNext":          w.builtinTabNext,
		"tabOpen":          w.builtinTabOpen,
		"tabPrev":          w.builtinTabPrev,
		"toggleQuickmark":  w.builtinToggleQuickmark,
		"windowEditURI":    w.builtinWindowEditURI,
		"windowOpen":       w.builtinWindowOpen,
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
func (w *Window) builtinCommandMode(_ *int) {
	w.setState(cmd.NewCommandLineMode(w.State, states.CommandLineSubstateCommand, w.runCmd))
}

// builtinEditURI initiates command mode with the open command primed for
// the current URI.
func (w *Window) builtinEditURI(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		fmt.Sprintf("open %v", w.getWebView().GetURI()),
		"",
		w.runCmd))
}

// builtinGoBack goes one step back in browser history.
func (w *Window) builtinGoBack(n *int) {
	for num := getWithDefault(n, 1, 0, 50); num > 0 && w.getWebView().CanGoBack(); num-- {
		w.getWebView().GoBack()
	}
}

// builtinGoForward goes one step forward in browser history.
func (w *Window) builtinGoForward(n *int) {
	for num := getWithDefault(n, 1, 0, 50); num > 0 && w.getWebView().CanGoForward(); num-- {
		w.getWebView().GoForward()
	}
}

// builtinInsertMode initiates insert mode.
func (w *Window) builtinInsertMode(_ *int) {
	w.setState(cmd.NewInsertMode(w.State, cmd.SubstateDefault))
}

// builtinNop does nothing. It is occasionally useful as a binding.
func (w *Window) builtinNop(_ *int) {}

// builtinOpen initiates command mode, primed with an open command.
func (w *Window) builtinOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, states.CommandLineSubstateCommand, "open ", "", w.runCmd))
}

// builtinPanic causes a panic. You probably don't want to use this.
func (w *Window) builtinPanic(_ *int) {
	panic("Builtin 'panic' called.")
}

// builtinQuickmarks enters quickmark mode (i.e. a binding mode for launching
// quickmarks)
func (w *Window) builtinQuickmarks(_ *int) {
	w.setState(cmd.NewNormalModeWithSubstate(w.State, states.NormalSubstateQuickmark))
}

// builtinQuickmarksTab enters quickmark mode (i.e. a binding mode for
// launching quickmarks), opening in a new tab.
func (w *Window) builtinQuickmarksTab(_ *int) {
	w.setState(cmd.NewNormalModeWithSubstate(w.State, states.NormalSubstateQuickmarkTab))
}

// builtinQuickmarksWindow enters quickmark mode (i.e. a binding mode for
// launching quickmarks), opening in a new window.
func (w *Window) builtinQuickmarksWindow(_ *int) {
	w.setState(cmd.NewNormalModeWithSubstate(w.State, states.NormalSubstateQuickmarkWindow))
}

// builtinQuickmarksRapid enters quickmark mode (i.e. a binding mode for
// launching quickmarks), opening in a new tab, and remaining in quickmarks
// rapid mode.
func (w *Window) builtinQuickmarksRapid(_ *int) {
	w.setState(cmd.NewNormalModeWithSubstate(w.State, states.NormalSubstateQuickmarksRapid))
}

// builtinReload reloads the current page.
func (w *Window) builtinReload(_ *int) {
	w.getWebView().Reload()
}

// builtinReloadNoCache reloads the current page, bypassing the cache.
func (w *Window) builtinReloadNoCache(_ *int) {
	w.getWebView().ReloadBypassCache()
}

// builtinScrollDown scrolls down.
func (w *Window) builtinScrollDown(n *int) {
	w.scrollDelta(w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), true)
}

// builtinScrollLeft scrolls left.
func (w *Window) builtinScrollLeft(n *int) {
	w.scrollDelta(-w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), false)
}

// builtinScrollRight scrolls right.
func (w *Window) builtinScrollRight(n *int) {
	w.scrollDelta(w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), false)
}

// builtinScrollPageDown scrolls down 80% of the page.
func (w *Window) builtinScrollPageDown(n *int) {
	w.scrollDelta(
		int(float64(w.Window.WebView.GetWebView().GetAllocatedHeight())*
			0.8*
			float64(getWithDefault(n, 1, 0, 1<<20))),
		true)
}

// builtinScrollPageUp scrolls up 80% of the page.
func (w *Window) builtinScrollPageUp(n *int) {
	w.scrollDelta(
		int(-float64(w.Window.WebView.GetWebView().GetAllocatedHeight())*
			0.8*
			float64(getWithDefault(n, 1, 0, 1<<20))),
		true)
}

// builtinScrollToBottom scrolls to the bottom of the page.
func (w *Window) builtinScrollToBottom(_ *int) {
	ext := w.getWebView()
	height, err := ext.getScrollHeight()
	if err != nil {
		w.setState(cmd.NewStatusMode(w.State,
			states.StatusSubstateError,
			fmt.Sprintf("Error scrolling: %v", err)))
		log.Printf("Error scrolling: %v", err)
	}
	err = ext.setScrollTop(height)
	if err != nil {
		log.Printf("Error scrolling: %v", err)
	}
}

// builtinScrollTotop scrolls to the top of the page.
func (w *Window) builtinScrollToTop(_ *int) {
	err := w.getWebView().setScrollTop(0)
	if err != nil {
		w.setState(cmd.NewStatusMode(w.State, states.StatusSubstateError, fmt.Sprintf("Error scrolling: %v", err)))
		log.Printf("Error scrolling: %v", err)
	}
}

// builtinScrollUp scrolls up.
func (w *Window) builtinScrollUp(n *int) {
	w.scrollDelta(-w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), true)
}

// builtinTabClose closes the current tab.
func (w *Window) builtinTabClose(n *int) {
	num := getWithDefault(n, 1, 0, len(w.webViews))
	i := w.currentWebView
	j := i + num
	if j > len(w.webViews) {
		diff := j - len(w.webViews)
		i -= diff
		j -= diff
	}
	w.tabsClose(i, j)
}

// builtinTabEditURI initiates command mode with a tabopen command primed for
// the current URI.
func (w *Window) builtinTabEditURI(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		fmt.Sprintf("tabopen %v", w.getWebView().GetURI()),
		"",
		w.runCmd))
}

// builtinTabGo goes to the specified tab.
func (w *Window) builtinTabGo(n *int) {
	num := getWithDefault(n, 1, 1, len(w.webViews))
	w.tabGo(num - 1)
}

// builtinTabNext goes to the next tab.
func (w *Window) builtinTabNext(n *int) {
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
func (w *Window) builtinTabOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, states.CommandLineSubstateCommand, "tabopen ", "", w.runCmd))
}

// builtinTabPrev goes to the previous tab.
func (w *Window) builtinTabPrev(n *int) {
	num := getWithDefault(n, 1, 0, 1<<20)
	size := len(w.webViews)
	newTab := (w.currentWebView - num) % size
	// Banish all ye negative modulo results.
	if newTab < 0 {
		newTab += size
	}
	w.tabGo(newTab)
}

// builtinToggleQuickmark toggles the quickmark state of the current site.
func (w *Window) builtinToggleQuickmark(_ *int) {
	// TODO confirm on delete.
	uri := w.getWebView().GetURI()
	if _, ok := w.parent.hasQuickmark[uri]; ok {
		b := false
		w.setState(cmd.NewYesNoConfirmMode(
			w.State,
			cmd.SubstateDefault,
			"Are you sure you want to remove the quickmark for this page?",
			&b,
			func(b bool) {
				if b {
					cmdRemoveQuickmark(w, w.parent, []string{"", uri})
				}
			}))
	} else {
		w.setState(cmd.NewPartialCommandLineMode(
			w.State,
			states.CommandLineSubstateCommand,
			"addquickmark ",
			" "+uri,
			w.runCmd))
	}
}

// builtinWindowEditURI initiates command mode with a winopen command primed
// for the current URI.
func (w *Window) builtinWindowEditURI(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		fmt.Sprintf("winopen %v", w.getWebView().GetURI()),
		"",
		w.runCmd))
}

// builtinWindowOpen initiates command mode primed with a winopen command.
func (w *Window) builtinWindowOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(w.State, states.CommandLineSubstateCommand, "winopen ", "", w.runCmd))
}

// scrollDelta scrolls a given amount of pixes either vertically or
// horizontally.
func (w *Window) scrollDelta(delta int, vertical bool) {
	var curr int64
	var err error
	wv := w.getWebView()
	if vertical {
		curr, err = wv.getScrollTop()
	} else {
		curr, err = wv.getScrollLeft()
	}
	if err != nil {
		w.setState(cmd.NewStatusMode(w.State, states.StatusSubstateError, fmt.Sprintf("Error scrolling: %v", err)))
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
		w.setState(cmd.NewStatusMode(w.State, states.StatusSubstateError, fmt.Sprintf("Error scrolling: %v", err)))
		log.Printf("Error scrolling: %v", err)
		return
	}
}
