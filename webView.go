package main

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"fmt"
	"log"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/webkit"
)

// webView wraps a webkit WebView to do additional bookkeeping.
type webView struct {
	*webkit.WebView
	*webExtension
	id       uint64
	top      int64
	height   int64
	parent   *golem
	settings *webkit.Settings
	window   *window
}

// newWebView creates a new webView using given settings as a template.
func (w *window) newWebView(settings *webkit.Settings) (*webView, error) {
	wv, err := webkit.NewWebViewWithUserContentManager(
		w.parent.userContentManager)
	if err != nil {
		return nil, err
	}

	// Each WebView gets it's own settings, to allow toggling settings on a
	// per tab and/or per window basis.
	newSettings := settings.Clone()

	wv.SetSettings(newSettings)

	webExten := webExtensionForWebView(w.parent.sBus, wv)

	ret := &webView{
		wv,
		webExten,
		wv.GetPageID(),
		0,
		0,
		w.parent,
		newSettings,
		w,
	}

	// Attach to the create signal, which creates new tabs on demand.
	ret.WebView.Connect("create", func(obj *glib.Object, ptr uintptr) {
		// TODO clean this up. It should probably be somewhere in the
		// webkit package.
		boxed := (*C.WebKitNavigationAction)(unsafe.Pointer(ptr))
		req := C.webkit_navigation_action_get_request(boxed)
		cStr := (*C.char)(C.webkit_uri_request_get_uri(req))
		log.Printf(C.GoString(cStr))
		if ret.window == nil {
			log.Printf("A tab currently not associated to a window " +
				"attempted to open a new tab. The request was dropped.")
		} else {
			ret.window.newTab(C.GoString(cStr))
		}
	})

	// Attach dbus to watch for signals from this extension.
	// There is no real need to disconnect this, dbus disconnects it for us
	// when the web process dies.
	//
	// NOTE: if for any reason we every move away from one process per tab,
	// this no longer holds.
	w.parent.sBus.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		fmt.Sprintf(webExtenWatchMessage, ret.id, ret.id),
	)

	// Add webview to golem and return.
	w.parent.wMutex.Lock()
	defer w.parent.wMutex.Unlock()
	w.parent.webViews[ret.id] = ret
	return ret, nil
}

// close updates bookkeeping after the web view is closed.
func (wv *webView) close() {
	wv.parent.wMutex.Lock()
	defer wv.parent.wMutex.Unlock()
	delete(wv.parent.webViews, wv.id)
}
