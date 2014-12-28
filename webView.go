package main

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/ui"
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
	tabUI    *ui.TabBarTab
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
		nil,
	}

	// Attach to the create signal, which creates new tabs on demand.
	ret.WebView.Connect("create", func(obj *glib.Object, ptr uintptr) {
		// TODO clean this up. It should probably be somewhere in the
		// webkit package.
		boxed := (*C.WebKitNavigationAction)(unsafe.Pointer(ptr))
		req := C.webkit_navigation_action_get_request(boxed)
		cStr := (*C.char)(C.webkit_uri_request_get_uri(req))
		if ret.window == nil {
			log.Printf("A tab currently not associated to a window " +
				"attempted to open a new tab. The request was dropped.")
		} else {
			wv, err := ret.window.newTab(C.GoString(cStr))
			if err != nil {
				log.Printf("Failed creation of new tab...")
			} else {
				// Focus our new tab.
				ret.window.tabGo(ret.window.tabIndex(wv))
			}
		}
	})

	// Attach to decision policies.
	ret.WebView.Connect("decide-policy",
		func(
			obj *glib.Object,
			decision *glib.Object,
			t C.WebKitPolicyDecisionType) bool {

			switch t {
			case C.WEBKIT_POLICY_DECISION_TYPE_NAVIGATION_ACTION:
				fallthrough
			case C.WEBKIT_POLICY_DECISION_TYPE_NEW_WINDOW_ACTION:
				nav := (*C.WebKitNavigationPolicyDecision)(unsafe.Pointer(decision.Native()))
				action :=
					C.webkit_navigation_policy_decision_get_navigation_action(
						nav)
				button := C.webkit_navigation_action_get_mouse_button(action)
				modifiers := C.webkit_navigation_action_get_modifiers(action)
				if button == 2 || (modifiers&cmd.ControlMask) != 0 {
					// We don't actually want to open this window directly.
					// we want it in a new tab.
					C.webkit_policy_decision_ignore(
						(*C.WebKitPolicyDecision)(unsafe.Pointer(nav)))
					if ret.window == nil {
						log.Printf("A tab currently not associated to a " +
							"window attempted to open a new tab. The " +
							"request was dropped.")
						return true
					}
					cReq := C.webkit_navigation_action_get_request(action)
					req := &webkit.UriRequest{
						&glib.Object{glib.ToGObject(unsafe.Pointer(cReq))},
					}
					req.Object.RefSink()
					runtime.SetFinalizer(req.Object, (*glib.Object).Unref)

					_, err := ret.window.newTabWithRequest(req)
					if err != nil {
						log.Printf("Failed creation of new tab...")
					}
					return true
				}
			case C.WEBKIT_POLICY_DECISION_TYPE_RESPONSE:
			}
			return false
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

// setTabUI sets the tab display for the tab.
func (wv *webView) setTabUI(t *ui.TabBarTab) {
	wv.WebView.Connect("notify::title", func() {
		t.SetTitle(wv.WebView.GetTitle())
	})
	wv.tabUI = t
}

// close updates bookkeeping after the web view is closed.
func (wv *webView) close() {
	wv.parent.wMutex.Lock()
	defer wv.parent.wMutex.Unlock()
	delete(wv.parent.webViews, wv.id)
}
