package golem

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"fmt"
	"html"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/ui"
	ggtk "github.com/tkerber/golem/gtk"
	"github.com/tkerber/golem/webkit"
)

// webView wraps a webkit WebView to do additional bookkeeping.
type webView struct {
	*webkit.WebView
	*webExtension
	id       uint64
	top      int64
	height   int64
	parent   *Golem
	settings *webkit.Settings
	window   *Window
	tabUI    *ui.TabBarTab
	handles  []glib.SignalHandle
}

// newWebView creates a new webView using given settings as a template.
func (w *Window) newWebView(settings *webkit.Settings) (*webView, error) {
	rets := ggtk.GlibMainContextInvoke(
		webkit.NewWebViewWithUserContentManager,
		w.parent.userContentManager)
	if rets[1] != nil {
		return nil, rets[1].(error)
	}
	wv := rets[0].(*webkit.WebView)

	// Each WebView gets it's own settings, to allow toggling settings on a
	// per tab and/or per window basis.
	newSettings := settings.Clone()

	wv.SetSettings(newSettings)

	webExten := webExtensionForWebView(w.parent, wv)

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
		make([]glib.SignalHandle, 0, 4),
	}

	// Attach to the create signal, which creates new tabs on demand.
	handle, err := ret.WebView.Connect("create", func(wv *webkit.WebView, ptr uintptr) {
		// TODO clean this up. It should probably be somewhere in the
		// webkit package.
		boxed := (*C.WebKitNavigationAction)(unsafe.Pointer(ptr))
		req := C.webkit_navigation_action_get_request(boxed)
		cStr := (*C.char)(C.webkit_uri_request_get_uri(req))
		if ret.window == nil {
			ret.window.logError("A tab currently not associated to a " +
				"window attempted to open a new tab. The request was dropped.")
		} else {
			wv, err := ret.window.NewTab(C.GoString(cStr))
			if err != nil {
				ret.window.logError("Failed creation of new tab...")
			} else {
				// Focus our new tab.
				ret.window.tabGo(ret.window.tabIndex(wv))
			}
		}
	})
	if err != nil {
		return nil, err
	}
	ret.handles = append(ret.handles, handle)

	// Attach to decision policies.
	handle, err = ret.WebView.Connect("decide-policy",
		func(
			wv *webkit.WebView,
			decision webkit.PolicyDecision,
			t C.WebKitPolicyDecisionType) bool {

			switch t {
			case C.WEBKIT_POLICY_DECISION_TYPE_NAVIGATION_ACTION:
				fallthrough
			case C.WEBKIT_POLICY_DECISION_TYPE_NEW_WINDOW_ACTION:
				decision := decision.(*webkit.NavigationPolicyDecision)
				action := decision.GetNavigationAction()
				button := action.GetMouseButton()
				modifiers := action.GetModifiers()
				if button == 2 || (modifiers&cmd.ControlMask) != 0 {
					// We don't actually want to open this window directly.
					// we want it in a new tab.
					decision.Ignore()
					if ret.window == nil {
						ret.window.logError("A tab currently not associated " +
							"to a window attempted to open a new tab. The " +
							"request was dropped.")
						return true
					}
					req := action.GetRequest()

					_, err := ret.window.newTabWithRequest(req)
					if err != nil {
						ret.window.logError("Failed creation of new tab...")
					}
					return true
				}
			case C.WEBKIT_POLICY_DECISION_TYPE_RESPONSE:
				decision := decision.(*webkit.ResponsePolicyDecision)
				resp := decision.GetResponse()
				mimetype := resp.GetMimeType()
				switch mimetype {
				case "application/pdf", "application/x-pdf":
					if resp.GetUri() == ret.WebView.GetURI() {
						site, err := Asset("srv/pdf.js/frame.html.fmt")
						if err == nil && w.parent.pdfjsEnabled {
							decision.Ignore()
							ret.WebView.LoadAlternateHtml(
								[]byte(fmt.Sprintf(
									string(site),
									html.EscapeString(resp.GetUri()))),
								resp.GetUri(),
								fmt.Sprintf(
									"golem:///pdf.js/frame.html?%s",
									resp.GetUri))
							return true
						}
					}
					return false
				default:
					return false
				}
			}
			return false
		})
	if err != nil {
		return nil, err
	}
	ret.handles = append(ret.handles, handle)

	// Attach dbus to watch for signals from this extension.
	// There is no real need to disconnect this, dbus disconnects it for us
	// when the web process dies.
	//
	// NOTE: if for any reason we every move away from one process per tab,
	// this no longer holds.
	w.parent.sBus.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		fmt.Sprintf(webExtenWatchMessage, w.parent.profile, ret.id,
			w.parent.profile, ret.id),
	)

	// Add webview to golem and return.
	w.parent.wMutex.Lock()
	defer w.parent.wMutex.Unlock()
	w.parent.webViews[ret.id] = ret
	return ret, nil
}

// GetTop retrieves the scroll distance from the top of the web view.
func (wv *webView) GetTop() int64 {
	return wv.top
}

// GetHeight retrieves the height of the web view.
func (wv *webView) GetHeight() int64 {
	return wv.height
}

// GetWebView retrieves the webkit webview.
func (wv *webView) GetWebView() *webkit.WebView {
	return wv.WebView
}

// IsQuickmarked checks if the current uri is quickmarked.
func (wv *webView) IsQuickmarked() bool {
	return wv.parent.hasQuickmark[wv.GetURI()]
}

// setTabUI sets the tab display for the tab.
func (wv *webView) setTabUI(t *ui.TabBarTab) {
	handle, err := wv.WebView.Connect("notify::title", func(wv *webkit.WebView) {
		t.SetTitle(wv.GetTitle())
	})
	if err == nil {
		wv.handles = append(wv.handles, handle)
	}
	handle, err = wv.WebView.Connect("notify::estimated-load-progress", func(wv *webkit.WebView) {
		t.SetLoadProgress(wv.GetEstimatedLoadProgress())
	})
	if err == nil {
		wv.handles = append(wv.handles, handle)
	}
	wv.tabUI = t
}

// close updates bookkeeping after the web view is closed.
func (wv *webView) close() {
	for _, handle := range wv.handles {
		wv.WebView.HandlerDisconnect(handle)
	}
	wv.parent.wMutex.Lock()
	delete(wv.parent.webViews, wv.id)
	wv.parent.wMutex.Unlock()
	wv.window = nil
	if p, _ := wv.WebView.GetParent(); p != nil {
		cont := &gtk.Container{*p}
		ggtk.GlibMainContextInvoke(cont.Remove, wv.WebView)
	}
	schedGc()
}
