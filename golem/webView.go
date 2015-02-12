package golem

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"fmt"
	"html"
	"net/url"
	"unsafe"

	"github.com/conformal/gotk3/gdk"
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
	*tabCfg
	id            uint64
	top           int64
	height        int64
	parent        *Golem
	settings      *webkit.Settings
	window        *Window
	tabUI         *ui.TabBarTab
	fullscreen    bool
	searchForward bool
	handles       []glib.SignalHandle
}

// newWebView creates a new webView.
func (w *Window) newWebView() (*webView, error) {
	rets := ggtk.GlibMainContextInvoke(
		webkit.NewWebViewWithUserContentManager,
		w.parent.userContentManager)
	if rets[1] != nil {
		return nil, rets[1].(error)
	}
	wv := rets[0].(*webkit.WebView)

	// Each WebView gets it's own settings, to allow toggling settings on a
	// per tab and/or per window basis.
	newSettings := w.defaultSettings.Clone()

	wv.SetSettings(newSettings)

	webExten := webExtensionForWebView(w.parent, wv)

	ret := &webView{
		wv,
		webExten,
		w.windowCfg.tabCfg.clone(),
		wv.GetPageID(),
		0,
		0,
		w.parent,
		newSettings,
		w,
		nil,
		false,
		true,
		make([]glib.SignalHandle, 0, 4),
	}

	// Attach to the create signal, which creates new tabs on demand.
	handle, err := ret.WebView.Connect("create",
		func(_ interface{}, ptr uintptr) {
			// TODO clean this up. It should probably be somewhere in the
			// webkit package.
			boxed := (*C.WebKitNavigationAction)(unsafe.Pointer(ptr))
			req := C.webkit_navigation_action_get_request(boxed)
			cStr := (*C.char)(C.webkit_uri_request_get_uri(req))
			if ret.window == nil {
				ret.window.logError("A tab currently not associated to a " +
					"window attempted to open a new tab. The request was " +
					"dropped.")
			} else {
				ggtk.GlibMainContextInvoke(func() {
					wvs, err := ret.window.NewTabs(C.GoString(cStr))
					if err != nil {
						ret.window.logError("Failed creation of new tab...")
					} else {
						// Focus our new tab.
						ret.window.TabGo(ret.window.tabIndex(wvs[0]))
					}
				})
			}
		})
	if err != nil {
		return nil, err
	}
	ret.handles = append(ret.handles, handle)

	// Attach to decision policies.
	handle, err = ret.WebView.Connect("decide-policy",
		func(
			_ interface{},
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

					ggtk.GlibMainContextInvoke(func() {
						_, err := ret.window.newTabWithRequest(req)
						if err != nil {
							ret.window.logError("Failed creation of new tab...")
						}
					})
					return true
				}
			case C.WEBKIT_POLICY_DECISION_TYPE_RESPONSE:
				decision := decision.(*webkit.ResponsePolicyDecision)
				resp := decision.GetResponse()
				mimetype := resp.GetMimeType()
				switch mimetype {
				case "application/pdf", "application/x-pdf":
					if resp.GetURI() == ret.WebView.GetURI() {
						site, err := Asset("srv/pdf.js/frame.html.fmt")
						if err == nil && w.parent.pdfjsEnabled {
							decision.Ignore()
							ret.WebView.LoadAlternateHTML(
								[]byte(fmt.Sprintf(
									string(site),
									html.EscapeString(url.QueryEscape(resp.GetURI())))),
								resp.GetURI(),
								fmt.Sprintf(
									"golem-unsafe://pdf.js/frame.html?%s",
									resp.GetURI()))
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
	handle, err = ret.WebView.Connect(
		"button-press-event",
		func(_ interface{}, e *gdk.Event) bool {
			if ret.window == nil {
				(*Window)(nil).logError("Button press registered on non-" +
					"visible webview. Dropping.")
				return false
			}
			return ret.window.handleBackForwardButtons(nil, e)
		})
	if err != nil {
		return nil, err
	}
	ret.handles = append(ret.handles, handle)
	// history handle
	handle, err = ret.WebView.Connect("load-changed",
		func(_ interface{}, e C.WebKitLoadEvent) {
			switch e {
			case C.WEBKIT_LOAD_FINISHED:
				go ret.parent.updateHistory(wv.GetURI(), wv.GetTitle())
			}
		})
	if err == nil {
		ret.handles = append(ret.handles, handle)
	}
	// tab ui handles.
	handle, err = ret.WebView.Connect("notify::title",
		func() {
			if ret.tabUI != nil {
				ret.tabUI.SetTitle(wv.GetTitle())
			}
		})
	if err == nil {
		ret.handles = append(ret.handles, handle)
	}
	handle, err = ret.WebView.Connect("notify::estimated-load-progress",
		func() {
			if ret.tabUI != nil {
				ret.tabUI.SetLoadProgress(wv.GetEstimatedLoadProgress())
			}
		})
	if err == nil {
		ret.handles = append(ret.handles, handle)
	}
	handle, err = ret.WebView.Connect("notify::favicon", ret.faviconChanged)
	if err == nil {
		ret.handles = append(ret.handles, handle)
	}
	// fullscreen handles
	handle, err = wv.Connect("enter-fullscreen", func() bool {
		ret.fullscreen = true
		return false
	})
	if err == nil {
		ret.handles = append(ret.handles, handle)
	}
	handle, err = wv.Connect("leave-fullscreen", func() bool {
		ret.fullscreen = false
		return false
	})
	if err == nil {
		ret.handles = append(ret.handles, handle)
	}

	// Add webview to golem and return.
	w.parent.wMutex.Lock()
	defer w.parent.wMutex.Unlock()
	w.parent.webViews[ret.id] = ret
	return ret, nil
}

// faviconChanged resets the favicon in the tab bar display.
func (wv *webView) faviconChanged() {
	if wv.tabUI != nil {
		favicon, _ := wv.GetFavicon()
		wv.tabUI.SetIcon(favicon)
	}
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
	_, ok := wv.parent.hasQuickmark[wv.GetURI()]
	return ok
}

// IsBookmarked checks if the current uri is bookmarked.
func (wv *webView) IsBookmarked() bool {
	_, ok := wv.parent.isBookmark[wv.GetURI()]
	return ok
}

// detach detaches the webview from the ui.
func (wv *webView) detach() {
	wv.window = nil
	wv.tabUI = nil
	if p, _ := wv.WebView.GetParent(); p != nil {
		cont := &gtk.Container{*p}
		cont.Remove(wv.WebView)
	}
}

// close updates bookkeeping after the web view is closed.
func (wv *webView) close() {
	for _, handle := range wv.handles {
		ggtk.GlibMainContextInvoke(wv.WebView.HandlerDisconnect, handle)
	}
	wv.parent.wMutex.Lock()
	delete(wv.parent.webViews, wv.id)
	wv.parent.wMutex.Unlock()
	ggtk.GlibMainContextInvoke(wv.detach)
	schedGc()
}
