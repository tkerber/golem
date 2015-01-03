package golem

import (
	"fmt"

	"github.com/tkerber/golem/gtk"
	"github.com/tkerber/golem/webkit"
)

// NewTab opens a new tab to a specified URI.
//
// If the URI is blank, the new tab page is used instead.
func (w *Window) NewTab(uri string) (*webView, error) {
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
func (w *Window) newTabWithRequest(req *webkit.UriRequest) (*webView, error) {
	wv, err := w.newTabBlank()
	if err != nil {
		return nil, err
	}
	wv.LoadRequest(req)
	return wv, nil
}

// newTabBlank opens a blank new tab.
func (w *Window) newTabBlank() (*webView, error) {
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
	w.Window.AttachWebView(wv)
	w.Window.TabCount = len(w.webViews)
	go w.UpdateLocation()
	// Note that we do *not* switch tabs here.
	return wv, nil
}

// tabNext goes to the next tab.
func (w *Window) tabNext() {
	w.tabGo((w.currentWebView + 1) % len(w.webViews))
}

// tabPrev goes to the previous tab.
func (w *Window) tabPrev() {
	w.tabGo((w.currentWebView + len(w.webViews) - 1) % len(w.webViews))
}

// tabGo goes to a specified tab.
func (w *Window) tabGo(index int) error {
	if index >= len(w.webViews) || index < 0 {
		return fmt.Errorf("Illegal tab index: %v", index)
	}
	w.wMutex.Lock()
	defer w.wMutex.Unlock()
	w.currentWebView = index
	w.Window.TabNumber = index + 1
	wv := w.getWebView()
	w.reconnectWebViewSignals()
	w.SwitchToWebView(wv)
	w.Window.TabBar.FocusTab(index)
	go w.UpdateLocation()
	return nil
}

// tabsClose closes the tabs from index i to j (slice indexes)
func (w *Window) tabsClose(i, j int) {
	if len(w.webViews) == j-i {
		w.Window.Close()
		return
	}
	w.wMutex.Lock()
	defer w.wMutex.Unlock()
	// I'm not entirely sure why this is necessary. Without it however, closing
	// tabs will occasionally freeze and sometimes crash golem.
	// Probably the closing has to happen between two frame updates. Why?
	// I don't know.
	gtk.GlibMainContextInvoke(func() {
		k := len(w.webViews) - (j - i)
		wvs := make([]*webView, j-i)
		copy(wvs, w.webViews[i:j])
		copy(
			w.webViews[i:k],
			w.webViews[j:])
		for i, _ := range w.webViews[k:] {
			w.webViews[i+k] = nil
		}
		if w.currentWebView > j {
			w.currentWebView -= (j - i)
		}
		w.webViews = w.webViews[:len(w.webViews)-(j-i)]
		w.Window.CloseTabs(i, j)
		for _, wv := range wvs {
			wv.close()
		}
		activeWebView := w.currentWebView >= i && w.currentWebView <= j
		if activeWebView {
			k := i - 1
			if k < 0 {
				k = 0
			}
			w.currentWebView = k
			w.Window.TabNumber = k + 1
			wv := w.getWebView()
			w.reconnectWebViewSignals()
			w.Window.FocusTab(k)
			w.Window.SwitchToWebView(wv)
			w.Window.TabCount = len(w.webViews)
			go w.Window.UpdateLocation()
		}
	})
}

// tabIndex retrieves the index of a particular webView.
//
// A return value of -1 indicates the tab is not contained in the current
// window.
func (w *Window) tabIndex(wv *webView) int {
	for i, wv2 := range w.webViews {
		if wv == wv2 {
			return i
		}
	}
	return -1
}