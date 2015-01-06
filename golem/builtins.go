package golem

import (
	"fmt"
	"strings"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
	ggtk "github.com/tkerber/golem/gtk"
)

// builtinsfor retrieves the builtin functions bound to a specific window.
func builtinsFor(w *Window) cmd.Builtins {
	return cmd.Builtins{
		"backgroundEditURI":    w.builtinBackgroundEditURI,
		"backgroundOpen":       w.builtinBackgroundOpen,
		"commandMode":          w.builtinCommandMode,
		"cutClipboard":         w.builtinCutClipboard,
		"cutPrimary":           w.builtinCutPrimary,
		"editURI":              w.builtinEditURI,
		"goBack":               w.builtinGoBack,
		"goForward":            w.builtinGoForward,
		"insertMode":           w.builtinInsertMode,
		"nop":                  w.builtinNop,
		"open":                 w.builtinOpen,
		"panic":                w.builtinPanic,
		"pasteClipboard":       w.builtinPasteClipboard,
		"pastePrimary":         w.builtinPastePrimary,
		"quickmarks":           w.builtinQuickmarks,
		"quickmarksTab":        w.builtinQuickmarksTab,
		"quickmarksWindow":     w.builtinQuickmarksWindow,
		"quickmarksRapid":      w.builtinQuickmarksRapid,
		"reload":               w.builtinReload,
		"reloadNoCache":        w.builtinReloadNoCache,
		"scrollDown":           w.builtinScrollDown,
		"scrollLeft":           w.builtinScrollLeft,
		"scrollRight":          w.builtinScrollRight,
		"scrollPageDown":       w.builtinScrollPageDown,
		"scrollPageUp":         w.builtinScrollPageUp,
		"scrollToBottom":       w.builtinScrollToBottom,
		"scrollToTop":          w.builtinScrollToTop,
		"scrollUp":             w.builtinScrollUp,
		"tabClose":             w.builtinTabClose,
		"tabEditURI":           w.builtinTabEditURI,
		"tabGo":                w.builtinTabGo,
		"tabNext":              w.builtinTabNext,
		"tabOpen":              w.builtinTabOpen,
		"tabPasteClipboard":    w.builtinTabPasteClipboard,
		"tabPastePrimary":      w.builtinTabPastePrimary,
		"tabPrev":              w.builtinTabPrev,
		"toggleQuickmark":      w.builtinToggleQuickmark,
		"windowEditURI":        w.builtinWindowEditURI,
		"windowOpen":           w.builtinWindowOpen,
		"windowPasteClipboard": w.builtinWindowPasteClipboard,
		"windowPastePrimary":   w.builtinWindowPastePrimary,
		"yankClipboard":        w.builtinYankClipboard,
		"yankPrimary":          w.builtinYankPrimary,
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

// builtinBackgroundEditURI initiates command mode with a bgopen command
// primed for the current URI.
func (w *Window) builtinBackgroundEditURI(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		fmt.Sprintf("bgopen %v", w.getWebView().GetURI()),
		"",
		w.runCmd))
}

// builtinBackgroundOpen initiates command mode primed with a bgopen command.
func (w *Window) builtinBackgroundOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		"bgopen ",
		"",
		w.runCmd))
}

// builtinCommandMode initiates command mode.
func (w *Window) builtinCommandMode(_ *int) {
	w.setState(cmd.NewCommandLineMode(w.State, states.CommandLineSubstateCommand, w.runCmd))
}

// builtinCutClipboard cuts n tabs after and including the current, adding the
// uris to the clipboard and keeping the tabs in temporary storage for 1
// minute.
func (w *Window) builtinCutClipboard(n *int) {
	w.builtinYankClipboard(n)
	i, j := w.numTabsToIndicies(getWithDefault(n, 1, 0, len(w.webViews)))
	w.tabsClose(i, j, true)
}

// builtinCutPrimary cuts n tabs after and including the current, adding the
// uris to the primary selection and keeping the tabs in temporary storage
// for 1 minute.
func (w *Window) builtinCutPrimary(n *int) {
	w.builtinYankPrimary(n)
	i, j := w.numTabsToIndicies(getWithDefault(n, 1, 0, len(w.webViews)))
	w.tabsClose(i, j, true)
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
	wv := w.getWebView()
	item, ok := wv.GetBackForwardList().GetNthItemWeak(-getWithDefault(n, 1, 0, 50))
	if ok {
		wv.GoToBackForwardListItem(item)
	}
}

// builtinGoForward goes one step forward in browser history.
func (w *Window) builtinGoForward(n *int) {
	wv := w.getWebView()
	item, ok := wv.GetBackForwardList().GetNthItemWeak(getWithDefault(n, 1, 0, 50))
	if ok {
		wv.GoToBackForwardListItem(item)
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

// builtinPasteClipboard pastes uris stored in the clipboard into the current
// tab (any more than one into new tabs).
//
// Pastes the tab cache if it isn't empty instead.
// If the tab cache is pasted behaves the same of builtinTabPasteClipboard.
func (w *Window) builtinPasteClipboard(_ *int) {
	if len(w.parent.webViewCache) != 0 {
		wvs := w.parent.pasteWebViews()
		_, err := w.newTabsWithWebViews(wvs...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
		return
	}
	uris, err := w.urisFromClipboard(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		return
	}
	w.getWebView().LoadURI(uris[0])
	if len(uris) > 1 {
		_, err := w.NewTabs(uris[1:]...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
	}
}

// builtinPastePrimary pastes uris stored in the primary selection into the
// current tab (any more than one into new tabs).
//
// Pastes the tab cache if it isn't empty instead.
func (w *Window) builtinPastePrimary(_ *int) {
	if len(w.parent.webViewCache) != 0 {
		wvs := w.parent.pasteWebViews()
		_, err := w.newTabsWithWebViews(wvs...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
		return
	}
	uris, err := w.urisFromClipboard(gdk.SELECTION_PRIMARY)
	if err != nil {
		return
	}
	w.getWebView().LoadURI(uris[0])
	if len(uris) > 1 {
		_, err := w.NewTabs(uris[1:]...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
	}
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
		w.logErrorf("Error scrolling: %v", err)
	}
	err = ext.setScrollTop(height)
	if err != nil {
		w.logErrorf("Error scrolling: %v", err)
	}
}

// builtinScrollTotop scrolls to the top of the page.
func (w *Window) builtinScrollToTop(_ *int) {
	err := w.getWebView().setScrollTop(0)
	if err != nil {
		w.logErrorf("Error scrolling %v", err)
	}
}

// builtinScrollUp scrolls up.
func (w *Window) builtinScrollUp(n *int) {
	w.scrollDelta(-w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), true)
}

// builtinTabClose closes the current tab.
func (w *Window) builtinTabClose(n *int) {
	i, j := w.numTabsToIndicies(getWithDefault(n, 1, 0, len(w.webViews)))
	w.tabsClose(i, j, false)
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

// builtinTabPasteClipboard pastes uris stored in the clipboard into new tabs.
//
// Pastes the tab cache if it isn't empty instead.
func (w *Window) builtinTabPasteClipboard(_ *int) {
	if len(w.parent.webViewCache) != 0 {
		wvs := w.parent.pasteWebViews()
		_, err := w.newTabsWithWebViews(wvs...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
		return
	}
	uris, err := w.urisFromClipboard(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		return
	}
	_, err = w.NewTabs(uris...)
	if err != nil {
		w.logErrorf("Failed to paste in web views: %v", err)
	}
}

// builtinTabPastePrimary pastes uris stored in the primary selection into new
// tabs.
//
// Pastes the tab cache if it isn't empty instead.
func (w *Window) builtinTabPastePrimary(_ *int) {
	if len(w.parent.webViewCache) != 0 {
		wvs := w.parent.pasteWebViews()
		_, err := w.newTabsWithWebViews(wvs...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
		return
	}
	uris, err := w.urisFromClipboard(gdk.SELECTION_PRIMARY)
	if err != nil {
		return
	}
	_, err = w.NewTabs(uris...)
	if err != nil {
		w.logErrorf("Failed to paste in web views: %v", err)
	}
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

// builtinWindowPasteClipboard pastes uris stored in the clipboard into a new
// window.
//
// Pastes the tab cache if it isn't empty instead.
func (w *Window) builtinWindowPasteClipboard(_ *int) {
	var win *Window
	var err error
	if len(w.parent.webViewCache) != 0 {
		wvs := w.parent.pasteWebViews()
		win, err = w.parent.newWindowWithWebView(wvs[0])
		if err != nil {
			w.logErrorf("Failed to open new window: %v", err)
			return
		}
		if len(wvs) > 1 {
			_, err := win.newTabsWithWebViews(wvs[1:]...)
			if err != nil {
				w.logErrorf("Failed to paste in web views: %v", err)
			}
		}
		return
	}
	uris, err := w.urisFromClipboard(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		return
	}
	win, err = w.parent.NewWindow(uris[0])
	if err != nil {
		w.logErrorf("Failed to open new window: %v", err)
		return
	}
	if len(uris) > 1 {
		_, err := w.NewTabs(uris[1:]...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
	}
}

// builtinWindowPastePrimary pastes uris stored in the primary selection into
// a new window.
//
// Pastes the tab cache if it isn't empty instead.
func (w *Window) builtinWindowPastePrimary(_ *int) {
	var win *Window
	var err error
	if len(w.parent.webViewCache) != 0 {
		wvs := w.parent.pasteWebViews()
		win, err = w.parent.newWindowWithWebView(wvs[0])
		if err != nil {
			w.logErrorf("Failed to open new window: %v", err)
			return
		}
		if len(wvs) > 1 {
			_, err := win.newTabsWithWebViews(wvs[1:]...)
			if err != nil {
				w.logErrorf("Failed to paste in web views: %v", err)
			}
		}
		return
	}
	uris, err := w.urisFromClipboard(gdk.SELECTION_PRIMARY)
	if err != nil {
		return
	}
	win, err = w.parent.NewWindow(uris[0])
	if err != nil {
		w.logErrorf("Failed to open new window: %v", err)
		return
	}
	if len(uris) > 1 {
		_, err := w.NewTabs(uris[1:]...)
		if err != nil {
			w.logErrorf("Failed to paste in web views: %v", err)
		}
	}
}

// builtinYankClipboard yanks the next n tabs (including the current) uris
// to the clipboard.
func (w *Window) builtinYankClipboard(n *int) {
	i, j := w.numTabsToIndicies(getWithDefault(n, 1, 0, len(w.webViews)))
	str := yankTabs(w.webViews[i:j])
	go ggtk.GlibMainContextInvoke(func() {
		clip, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		if err != nil {
			w.logErrorf("Failed to yank to clipboard: %v", err)
			return
		}
		clip.SetText(str)
	})
}

// builtinYankPrimary yanks the next n tabs (including the current) uris
// to the primary selection.
func (w *Window) builtinYankPrimary(n *int) {
	i, j := w.numTabsToIndicies(getWithDefault(n, 1, 0, len(w.webViews)))
	str := yankTabs(w.webViews[i:j])
	go ggtk.GlibMainContextInvoke(func() {
		clip, err := gtk.ClipboardGet(gdk.SELECTION_PRIMARY)
		if err != nil {
			w.logErrorf("Failed to yank to clipboard: %v", err)
			return
		}
		clip.SetText(str)
	})
}

// urisFromClipboard gets a slice of uris from the specified clipboard.
func (w *Window) urisFromClipboard(selection gdk.Atom) ([]string, error) {
	args := ggtk.GlibMainContextInvoke(func() (string, error) {
		clip, err := gtk.ClipboardGet(selection)
		if err != nil {
			w.logErrorf("Failed to access clipboard: %v", err)
			return "", err
		}
		return clip.WaitForText()
	})
	if args[1] != nil {
		return nil, args[1].(error)
	}
	return strings.Split(args[0].(string), "\n"), nil
}

// yankTabs extracts a uri string from given webviews to yank.
func yankTabs(wvs []*webView) string {
	uris := make([]string, len(wvs))
	for i, wv := range wvs {
		uris[i] = wv.GetURI()
	}
	return strings.Join(uris, "\n")
}

// numTabsToIndicies gets a slice indexes to the webViews slice from a number
// of tabs to select.
//
// n must be <= the existing number or tabs, and tabs will be taken from
// the current tab onward if possible. If not, tabs before the current tab
// will be added until it fits.
func (w *Window) numTabsToIndicies(n int) (i, j int) {
	i = w.currentWebView
	j = i + n
	if j > len(w.webViews) {
		diff := j - len(w.webViews)
		i -= diff
		j -= diff
	}
	return i, j
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
		w.logErrorf("Error scrolling: %v", err)
		return
	}
	curr += int64(delta)
	if vertical {
		err = wv.setScrollTop(curr)
	} else {
		err = wv.setScrollLeft(curr)
	}
	if err != nil {
		w.logErrorf("Error scrolling: %v", err)
		return
	}
}
