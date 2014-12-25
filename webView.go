package main

import (
	"fmt"

	"github.com/tkerber/golem/webkit"
)

// webView wraps a webkit WebView to do additional bookkeeping.
type webView struct {
	*webkit.WebView
	*webExtension
	id     uint64
	top    int64
	height int64
	parent *golem
}

// newWebView creates a new webView using given settings as a template.
func (g *golem) newWebView(settings *webkit.Settings) (*webView, error) {
	wv, err := webkit.NewWebViewWithUserContentManager(g.userContentManager)
	if err != nil {
		return nil, err
	}

	// Each WebView gets it's own settings, to allow toggling settings on a
	// per tab and/or per window basis.
	newSettings := settings.Clone()

	wv.SetSettings(newSettings)

	webExten := webExtensionForWebView(g.sBus, wv)

	ret := &webView{
		wv,
		webExten,
		wv.GetPageID(),
		0,
		0,
		g,
	}

	// Attach dbus to watch for signals from this extension.
	// There is no real need to disconnect this, dbus disconnects it for us
	// when the web process dies.
	//
	// NOTE: if for any reason we every move away from one process per tab,
	// this no longer holds.
	g.sBus.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		fmt.Sprintf(webExtenWatchMessage, ret.id, ret.id),
	)

	// Add webview to golem and return.
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	g.webViews[ret.id] = ret
	return ret, nil
}

// close updates bookkeeping after the web view is closed.
func (wv *webView) close() {
	wv.parent.wMutex.Lock()
	defer wv.parent.wMutex.Unlock()
	delete(wv.parent.webViews, wv.id)
}
