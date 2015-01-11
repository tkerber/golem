package golem

import (
	"fmt"

	"github.com/tkerber/golem/golem/ui"
	"github.com/tkerber/golem/gtk"
	"github.com/tkerber/golem/webkit"
)

// NewTabs opens several new tabs to the specified URIs.
//
// If the URI is blank, the new tab page is used instead.
//
// NewTabs is a glib atomic operation, i.e. it is executed in glibs main
// context.
func (w *Window) NewTabs(uris ...string) ([]*webView, error) {
	var wvs []*webView
	var err error
	gtk.GlibMainContextInvoke(func() {
		wvs = make([]*webView, len(uris))
		wvs, err = w.newTabsWithWebViews(wvs...)
		if err != nil {
			return
		}
		for i, wv := range wvs {
			if uris[i] == "" {
				wv.LoadURI(w.parent.newTabPage)
			} else {
				wv.LoadURI(uris[i])
			}
		}
	})
	return wvs, err
}

// newTabWithRequest opens a new tab and loads a specified uri request into
// it.
//
// newTabWithRequest is a glib atomic operation, i.e. it is executed in glibs
// main context.
func (w *Window) newTabWithRequest(req *webkit.URIRequest) (*webView, error) {
	var wv *webView
	var err error
	gtk.GlibMainContextInvoke(func() {
		var wvs []*webView
		wvs, err = w.newTabsWithWebViews(nil)
		if err != nil {
			return
		}
		wvs[0].LoadRequest(req)
		wv = wvs[0]
	})
	return wv, err
}

// newTabsWithWebViews creates tabs for several web views and attaches them
// after the current tab.
//
// If nil is supplied as a web view, a new web view is created.
//
// newTabsWithWebViews is a glib atomic operation, i.e. it is executed in
// glibs main context.
func (w *Window) newTabsWithWebViews(wvs ...*webView) ([]*webView, error) {
	var err error
	gtk.GlibMainContextInvoke(func() {
		for i, wv := range wvs {
			if wv == nil {
				var wv *webView
				wv, err = w.newWebView(w.getWebView().settings)
				if err != nil {
					return
				}
				wvs[i] = wv
			} else {
				wv.window = w
			}
		}
		var tabs []*ui.TabBarTab
		tabs, err = w.Window.TabBar.AddTabs(
			w.currentWebView+1,
			w.currentWebView+1+len(wvs))
		if err != nil {
			return
		}
		for i := 0; i < len(wvs); i++ {
			wvs[i].tabUI = tabs[i]
			wvs[i].tabUI.SetTitle(wvs[i].GetTitle())
		}
		w.wMutex.Lock()
		defer w.wMutex.Unlock()
		// At the new tab directly after the current one.
		newWebViews := make([]*webView, len(w.webViews)+len(wvs))
		copy(newWebViews[:w.currentWebView+1], w.webViews[:w.currentWebView+1])
		copy(newWebViews[w.currentWebView+1:w.currentWebView+1+len(wvs)], wvs)
		copy(newWebViews[w.currentWebView+1+len(wvs):],
			w.webViews[w.currentWebView+1:])
		w.webViews = newWebViews
		for _, wv := range wvs {
			w.Window.AttachWebView(wv)
		}
		w.Window.TabCount = len(w.webViews)
		go w.UpdateLocation()
	})
	// Note that we do *not* switch tabs here.
	return wvs, err
}

// TabNext goes to the next tab.
//
// TabNext is a glib atomic operation, that is, it is executed in glibs main
// context.
func (w *Window) TabNext() {
	w.TabGo((w.currentWebView + 1) % len(w.webViews))
}

// TabPrev goes to the previous tab.
//
// TabPrev is a glib atomic operation, that is, it is executed in glibs main
// context.
func (w *Window) TabPrev() {
	w.TabGo((w.currentWebView + len(w.webViews) - 1) % len(w.webViews))
}

// TabGo goes to a specified tab.
//
// TabGo is a glib atomic operation, that is, it is executed in glibs main
// context.
func (w *Window) TabGo(index int) error {
	var err error
	gtk.GlibMainContextInvoke(func() {
		if index >= len(w.webViews) || index < 0 {
			err = fmt.Errorf("Illegal tab index: %v", index)
			return
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
	})
	return err
}

// tabsClose closes the tabs from index i to j (slice indexes)
//
// tabsClose is a glib atomic operation, that is, it is executed in glibs main
// context.
func (w *Window) tabsClose(i, j int, cut bool) {
	if len(w.webViews) == j-i {
		gtk.GlibMainContextInvoke(func() {
			if cut {
				for _, wv := range w.webViews {
					wv.detach()
				}
				w.parent.cutWebViews(w.webViews)
				w.webViews = make([]*webView, 0)
			}
			w.Window.Close()
		})
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
		for i := range w.webViews[k:] {
			w.webViews[i+k] = nil
		}
		if w.currentWebView > j {
			w.currentWebView -= (j - i)
		}
		w.webViews = w.webViews[:len(w.webViews)-(j-i)]
		w.Window.CloseTabs(i, j)
		if cut {
			for _, wv := range wvs {
				wv.detach()
			}
			w.parent.cutWebViews(wvs)
		} else {
			for _, wv := range wvs {
				wv.close()
			}
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
