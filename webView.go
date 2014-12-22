package main

import (
	"fmt"

	"github.com/tkerber/golem/webkit"
)

type webView struct {
	*webkit.WebView
	*webExtension
	id     uint64
	top    int64
	height int64
	parent *golem
}

func (g *golem) newWebView() (*webView, error) {
	wv, err := webkit.NewWebViewWithUserContentManager(g.userContentManager)
	if err != nil {
		return nil, err
	}

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

func (wv *webView) close() {
	wv.parent.wMutex.Lock()
	defer wv.parent.wMutex.Unlock()
	delete(wv.parent.webViews, wv.id)
}
