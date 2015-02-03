package golem

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"
	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
	ggtk "github.com/tkerber/golem/gtk"
	"github.com/tkerber/golem/webkit"
)

var builtinNames []string

func init() {
	dummyBuiltins := builtinsFor(nil)
	builtinNames = make([]string, 0, len(dummyBuiltins))
	for b := range dummyBuiltins {
		builtinNames = append(builtinNames, b)
	}
}

type builtinSpec struct {
	function func(*int)
	desc     string
}

// builtinsfor retrieves the builtin functions bound to a specific window.
func builtinsFor(w *Window) cmd.Builtins {
	specs := map[string]builtinSpec{
		"addSearchEngine":      {w.builtinAddSearchEngine, "Finds and adds a new search engine on the page"},
		"backgroundEditURI":    {w.builtinBackgroundEditURI, "Edits URI and opens in a background tab"},
		"backgroundOpen":       {w.builtinBackgroundOpen, "Opens URI in a background tab"},
		"commandMode":          {w.builtinCommandMode, "Enters command mode"},
		"cutClipboard":         {w.builtinCutClipboard, "Cuts tabs to clipboard selection"},
		"cutPrimary":           {w.builtinCutPrimary, "Cuts tabs to primary selection"},
		"editURI":              {w.builtinEditURI, "Edits the current URI"},
		"goBack":               {w.builtinGoBack, "Goes back in browser history"},
		"goForward":            {w.builtinGoForward, "Goes forward in browser history"},
		"hintsBackground":      {w.builtinHintsBackground, "Follows a link in a background tab"},
		"hintsFollow":          {w.builtinHintsFollow, "Clicks something"},
		"hintsRapid":           {w.builtinHintsRapid, "Follows several links in background tabs"},
		"hintsTab":             {w.builtinHintsTab, "Follows a link in a new tab"},
		"hintsWindow":          {w.builtinHintsWindow, "Follows a link in a new window"},
		"insertMode":           {w.builtinInsertMode, "Enters intert mode"},
		"noh":                  {w.builtinNoh, "Removes all highlighting"},
		"nop":                  {w.builtinNop, "Does nothing"},
		"open":                 {w.builtinOpen, "Opens a new page"},
		"panic":                {w.builtinPanic, "Crashes golem"},
		"pasteClipboard":       {w.builtinPasteClipboard, "Pastes from clipboard selection into current tab"},
		"pastePrimary":         {w.builtinPastePrimary, "Pastes from primary selection into current tab"},
		"quickmarks":           {w.builtinQuickmarks, "Opens a quickmark"},
		"quickmarksTab":        {w.builtinQuickmarksTab, "Opens a quickmark in a new tab"},
		"quickmarksWindow":     {w.builtinQuickmarksWindow, "Opens a quickmark in a new window"},
		"quickmarksRapid":      {w.builtinQuickmarksRapid, "Opens several quickmarks in background tabs"},
		"reload":               {w.builtinReload, "Reloads the page"},
		"reloadNoCache":        {w.builtinReloadNoCache, "Reloads the page, ignoring the cache"},
		"scrollDown":           {w.builtinScrollDown, "Scrolls down"},
		"scrollLeft":           {w.builtinScrollLeft, "Scrolls up"},
		"scrollRight":          {w.builtinScrollRight, "Scrolls right"},
		"scrollPageDown":       {w.builtinScrollPageDown, "Scrolls down a page"},
		"scrollPageUp":         {w.builtinScrollPageUp, "Scrolls up a page"},
		"scrollToBottom":       {w.builtinScrollToBottom, "Scrolls to the bottom of a page"},
		"scrollToTop":          {w.builtinScrollToTop, "Scrolls to the top of a page"},
		"scrollUp":             {w.builtinScrollUp, "Scrolls up"},
		"searchMode":           {w.builtinSearchMode, "Enters search mode"},
		"searchModeBackwards":  {w.builtinSearchModeBackwards, "Enters backwards search mode"},
		"searchNext":           {w.builtinSearchNext, "Searches for the next match"},
		"searchPrevious":       {w.builtinSearchPrevious, "Searches for the previous match"},
		"tabClose":             {w.builtinTabClose, "Closes tabs"},
		"tabEditURI":           {w.builtinTabEditURI, "Edits URI and opens in a new tab"},
		"tabGo":                {w.builtinTabGo, "Goes to a particular tab"},
		"tabNext":              {w.builtinTabNext, "Goes to the next tab"},
		"tabOpen":              {w.builtinTabOpen, "Opens a new tab"},
		"tabPasteClipboard":    {w.builtinTabPasteClipboard, "Pastes tabs from clipboard selection"},
		"tabPastePrimary":      {w.builtinTabPastePrimary, "Pastes tabs from primary selection"},
		"tabPrev":              {w.builtinTabPrev, "Goes to the previous tab"},
		"toggleQuickmark":      {w.builtinToggleQuickmark, "Toggles quickmark status of the current page"},
		"windowEditURI":        {w.builtinWindowEditURI, "Edits URI and opens in a new window"},
		"windowOpen":           {w.builtinWindowOpen, "Opens a new window"},
		"windowPasteClipboard": {w.builtinWindowPasteClipboard, "Pastes tabs from clipboard selection into a new window"},
		"windowPastePrimary":   {w.builtinWindowPastePrimary, "Pastes tabs from primary selection into a new window"},
		"yankClipboard":        {w.builtinYankClipboard, "Yanks tab URIs to clipboard selection"},
		"yankPrimary":          {w.builtinYankPrimary, "Yanks tab URIs to primary selection"},
	}
	ret := make(map[string]*cmd.Builtin, len(specs))
	for name, spec := range specs {
		ret[name] = &cmd.Builtin{spec.function, name, spec.desc}
	}
	return ret
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

// searchEngineReplacer convert the search engine uri passed into a format
// string.
var searchEngineReplacer = strings.NewReplacer(
	"%", "%%",
	"__golem_form_variable", "%v")

// builtinAddSearchEngine hints possible search engine fields on screens and
// allows selecting them for a primed 'ase' command.
func (w *Window) builtinAddSearchEngine(_ *int) {
	wv := w.getWebView()
	w.setState(states.NewHintsMode(
		w.State,
		states.HintsSubstateSearchEngine,
		wv,
		func(uri string) bool {
			w.setState(cmd.NewPartialCommandLineMode(
				w.State,
				states.CommandLineSubstateCommand,
				"ase ",
				fmt.Sprintf(" %s %s",
					strconv.Quote(wv.GetTitle()),
					strconv.Quote(searchEngineReplacer.Replace(uri))),
				w.runCmd))
			return false
		}))
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
	w.setState(cmd.NewCommandLineMode(
		w.State, states.CommandLineSubstateCommand, w.runCmd))
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
	item, ok := wv.GetBackForwardList().GetNthItemWeak(
		-getWithDefault(n, 1, 0, 50))
	if ok {
		wv.GoToBackForwardListItem(item)
	}
}

// builtinGoForward goes one step forward in browser history.
func (w *Window) builtinGoForward(n *int) {
	wv := w.getWebView()
	item, ok := wv.GetBackForwardList().GetNthItemWeak(
		getWithDefault(n, 1, 0, 50))
	if ok {
		wv.GoToBackForwardListItem(item)
	}
}

// builtinHintsBackground enters hints mode to follow a link in a new
// background tab.
func (w *Window) builtinHintsBackground(_ *int) {
	w.setState(states.NewHintsMode(
		w.State,
		states.HintsSubstateBackground,
		w.getWebView(),
		func(uri string) bool {
			_, err := w.NewTabs(uri)
			if err != nil {
				w.logErrorf("Failed to open new tab: %v", err)
			}
			return false
		}))
}

// builtinHintsFollow enters hints mode click something.
func (w *Window) builtinHintsFollow(_ *int) {
	w.setState(states.NewHintsMode(
		w.State,
		states.HintsSubstateFollow,
		w.getWebView(),
		func(uri string) bool {
			w.logErrorf("Hints callback on callbackless hint type.")
			return false
		}))
}

// builtinHintsRapid enters hints mode to follow several links in background
// tabs.
func (w *Window) builtinHintsRapid(_ *int) {
	w.setState(states.NewHintsMode(
		w.State,
		states.HintsSubstateRapid,
		w.getWebView(),
		func(uri string) bool {
			_, err := w.NewTabs(uri)
			if err != nil {
				w.logErrorf("Failed to open new tab: %v", err)
			}
			return true
		}))
}

// builtinHintsTab enters hints mode to follow a link in a new tab.
func (w *Window) builtinHintsTab(_ *int) {
	w.setState(states.NewHintsMode(
		w.State,
		states.HintsSubstateTab,
		w.getWebView(),
		func(uri string) bool {
			_, err := w.NewTabs(uri)
			if err != nil {
				w.logErrorf("Failed to open new tab: %v", err)
			}
			w.TabNext()
			return false
		}))
}

// builtinHintsWindow enters hints mode to follow a link in a new window.
func (w *Window) builtinHintsWindow(_ *int) {
	w.setState(states.NewHintsMode(
		w.State,
		states.HintsSubstateWindow,
		w.getWebView(),
		func(uri string) bool {
			w.parent.NewWindow(uri)
			return false
		}))
}

// builtinInsertMode initiates insert mode.
func (w *Window) builtinInsertMode(_ *int) {
	w.setState(cmd.NewInsertMode(w.State, cmd.SubstateDefault))
}

// builtinNoh removes all active highlighting from the page.
func (w *Window) builtinNoh(_ *int) {
	cmdNoHLSearch(w, w.parent, nil)
}

// builtinNop does nothing. It is occasionally useful as a binding.
func (w *Window) builtinNop(_ *int) {}

// builtinOpen initiates command mode, primed with an open command.
func (w *Window) builtinOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		"open ",
		"",
		w.runCmd))
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
	if len(w.parent.webViewCache) != 0 && !w.parent.clipboardChanged() {
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
	if len(w.parent.webViewCache) != 0 && !w.parent.primaryChanged() {
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
	w.setState(cmd.NewNormalModeWithSubstate(
		w.State, states.NormalSubstateQuickmark))
}

// builtinQuickmarksTab enters quickmark mode (i.e. a binding mode for
// launching quickmarks), opening in a new tab.
func (w *Window) builtinQuickmarksTab(_ *int) {
	w.setState(cmd.NewNormalModeWithSubstate(
		w.State, states.NormalSubstateQuickmarkTab))
}

// builtinQuickmarksWindow enters quickmark mode (i.e. a binding mode for
// launching quickmarks), opening in a new window.
func (w *Window) builtinQuickmarksWindow(_ *int) {
	w.setState(cmd.NewNormalModeWithSubstate(
		w.State, states.NormalSubstateQuickmarkWindow))
}

// builtinQuickmarksRapid enters quickmark mode (i.e. a binding mode for
// launching quickmarks), opening in a new tab, and remaining in quickmarks
// rapid mode.
func (w *Window) builtinQuickmarksRapid(_ *int) {
	w.setState(cmd.NewNormalModeWithSubstate(
		w.State, states.NormalSubstateQuickmarksRapid))
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
	// TODO with different target areas this will scroll way too much.
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
	height, err := ext.getScrollTargetHeight()
	if err != nil {
		w.logErrorf("Error scrolling: %v", err)
	}
	err = ext.setScrollTargetTop(height)
	if err != nil {
		w.logErrorf("Error scrolling: %v", err)
	}
}

// builtinScrollTotop scrolls to the top of the page.
func (w *Window) builtinScrollToTop(_ *int) {
	err := w.getWebView().setScrollTargetTop(0)
	if err != nil {
		w.logErrorf("Error scrolling %v", err)
	}
}

// builtinScrollUp scrolls up.
func (w *Window) builtinScrollUp(n *int) {
	w.scrollDelta(-w.parent.scrollDelta*getWithDefault(n, 1, 0, 1<<20), true)
}

// search searches for a specific term.
func (w *Window) search(term string) {
	wv := w.getWebView()
	wv.GetFindController().Search(
		term, webkit.FindOptionsWrapAround|webkit.FindOptionsCaseInsensitive)
	wv.searchForward = true
}

// search searches backwards for a specific term.
func (w *Window) backSearch(term string) {
	wv := w.getWebView()
	wv.GetFindController().Search(
		term,
		webkit.FindOptionsWrapAround|
			webkit.FindOptionsBackwards|
			webkit.FindOptionsCaseInsensitive)
	wv.searchForward = false
}

// builtinSearchMode initiates search mode.
func (w *Window) builtinSearchMode(_ *int) {
	w.setState(cmd.NewCommandLineMode(
		w.State, states.CommandLineSubstateSearch, w.search))
}

// builtinSearchModeBackwards initiates search mode in reverse.
func (w *Window) builtinSearchModeBackwards(_ *int) {
	w.setState(cmd.NewCommandLineMode(
		w.State, states.CommandLineSubstateBackSearch, w.backSearch))
}

// builtinSearchNext moves to the next found search element.
func (w *Window) builtinSearchNext(n *int) {
	num := getWithDefault(n, 1, 1, 25)
	wv := w.getWebView()
	fc := wv.GetFindController()
	for i := 0; i < num; i++ {
		if wv.searchForward {
			fc.SearchNext()
		} else {
			fc.SearchPrevious()
		}
	}
}

// builtinSearchPrevious moves to the previous found search element.
func (w *Window) builtinSearchPrevious(n *int) {
	num := getWithDefault(n, 1, 1, 25)
	wv := w.getWebView()
	fc := wv.GetFindController()
	for i := 0; i < num; i++ {
		if wv.searchForward {
			fc.SearchPrevious()
		} else {
			fc.SearchNext()
		}
	}
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
	w.TabGo(num - 1)
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
	w.TabGo(newTab)
}

// builtinTabOpen initiates command mode primed with a tabopen command.
func (w *Window) builtinTabOpen(_ *int) {
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		"tabopen ",
		"",
		w.runCmd))
}

// builtinTabPasteClipboard pastes uris stored in the clipboard into new tabs.
//
// Pastes the tab cache if it isn't empty instead.
func (w *Window) builtinTabPasteClipboard(_ *int) {
	if len(w.parent.webViewCache) != 0 && !w.parent.clipboardChanged() {
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
	if len(w.parent.webViewCache) != 0 && !w.parent.primaryChanged() {
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
	w.TabGo(newTab)
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
	w.setState(cmd.NewPartialCommandLineMode(
		w.State,
		states.CommandLineSubstateCommand,
		"winopen ",
		"",
		w.runCmd))
}

// builtinWindowPasteClipboard pastes uris stored in the clipboard into a new
// window.
//
// Pastes the tab cache if it isn't empty instead.
func (w *Window) builtinWindowPasteClipboard(_ *int) {
	var win *Window
	var err error
	if len(w.parent.webViewCache) != 0 && !w.parent.clipboardChanged() {
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
	if len(w.parent.webViewCache) != 0 && !w.parent.primaryChanged() {
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
	ggtk.GlibMainContextInvoke(func() {
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
	ggtk.GlibMainContextInvoke(func() {
		clip, err := gtk.ClipboardGet(gdk.SELECTION_PRIMARY)
		if err != nil {
			w.logErrorf("Failed to yank to primary selection: %v", err)
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
	split := strings.Split(args[0].(string), "\n")
	ret := make([]string, 0, len(split))
	for _, line := range split {
		if line == "" {
			continue
		}
		splitLine, err := shellwords.Parse(line)
		if err != nil {
			ret = append(ret, line)
		} else {
			ret = append(ret, w.parent.OpenURI(splitLine))
		}
	}
	return ret, nil
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
		curr, err = wv.getScrollTargetTop()
	} else {
		curr, err = wv.getScrollTargetLeft()
	}
	if err != nil {
		w.logErrorf("Error scrolling: %v", err)
		return
	}
	curr += int64(delta)
	if vertical {
		err = wv.setScrollTargetTop(curr)
	} else {
		err = wv.setScrollTargetLeft(curr)
	}
	if err != nil {
		w.logErrorf("Error scrolling: %v", err)
		return
	}
}
