package main

import (
	"fmt"

	"github.com/tkerber/golem/webkit"
)

// newTab opens a new tab to a specified URI.
//
// If the URI is blank, the new tab page is used instead.
func (w *window) newTab(uri string) (*webView, error) {
	wv, err := w.newTabBlank()
	if err != nil {
		return nil, err
	}
	if uri == "" {
		wv.LoadURI(w.parent.newTabPage)
	} else {
		wv.LoadURI(uri)
	}
	return wv, nil
}

// newTabWithRequests opens a new tab and loads a specified uri request into
// it.
func (w *window) newTabWithRequest(req *webkit.UriRequest) (*webView, error) {
	wv, err := w.newTabBlank()
	if err != nil {
		return nil, err
	}
	wv.LoadRequest(req)
	return wv, nil
}

// newTabBlank opens a blank new tab.
func (w *window) newTabBlank() (*webView, error) {
	wv, err := w.newWebView(w.getWebView().settings)
	if err != nil {
		return nil, err
	}
	tab, err := w.Window.TabBar.AddTab(w.currentWebView + 1)
	if err != nil {
		return nil, err
	}
	wv.setTabUI(tab)
	w.wMutex.Lock()
	defer w.wMutex.Unlock()
	// At the new tab directly after the current one.
	newWebViews := append(w.webViews, nil)
	copy(
		newWebViews[w.currentWebView+2:],
		newWebViews[w.currentWebView+1:len(newWebViews)-1])
	newWebViews[w.currentWebView+1] = wv
	w.webViews = newWebViews
	w.Window.TabCount = len(w.webViews)
	go w.UpdateLocation()
	// Note that we do *not* switch tabs here.
	return wv, nil
}

// tabNext goes to the next tab.
func (w *window) tabNext() {
	w.tabGo((w.currentWebView + 1) % len(w.webViews))
}

// tabPrev goes to the previous tab.
func (w *window) tabPrev() {
	w.tabGo((w.currentWebView + len(w.webViews) - 1) % len(w.webViews))
}

// tabGo goes to a specified tab.
func (w *window) tabGo(index int) error {
	if index >= len(w.webViews) || index < 0 {
		return fmt.Errorf("Illegal tab index: %v", index)
	}
	w.wMutex.Lock()
	defer w.wMutex.Unlock()
	w.currentWebView = index
	w.Window.TabNumber = index + 1
	wv := w.getWebView()
	w.reconnectWebViewSignals()
	w.ReplaceWebView(wv)
	w.Window.TabBar.FocusTab(index)
	go w.UpdateLocation()
	return nil
}

// tabClose closes the current tab.
func (w *window) tabClose(i int) {
	w.wMutex.Lock()
	defer w.wMutex.Unlock()
	wv := w.webViews[i]
	copy(
		w.webViews[i:len(w.webViews)-1],
		w.webViews[i+1:])
	activeWebView := w.currentWebView == i
	if w.currentWebView > i {
		w.currentWebView--
	}
	w.webViews[len(w.webViews)-1] = nil
	w.webViews = w.webViews[:len(w.webViews)-1]
	w.Window.CloseTab(i)
	wv.close()
	if len(w.webViews) == 0 {
		w.Window.Close()
	} else if activeWebView {
		j := w.currentWebView - 1
		if j < 0 {
			j = 0
		}
		w.currentWebView = j
		w.Window.TabNumber = j + 1
		wv := w.getWebView()
		w.Window.FocusTab(j)
		w.reconnectWebViewSignals()
		w.ReplaceWebView(wv)
		w.Window.TabCount = len(w.webViews)
		go w.UpdateLocation()
	}
}

// tabIndex retrieves the index of a particular webView.
//
// A return value of -1 indicates the tab is not contained in the current
// window.
func (w *window) tabIndex(wv *webView) int {
	for i, wv2 := range w.webViews {
		if wv == wv2 {
			return i
		}
	}
	return -1
}
