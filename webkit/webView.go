package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #cgo pkg-config: gtk+-3.0
// #include <webkit2/webkit2.h>
// #include <gtk/gtk.h>
// #include <stdlib.h>
/*
static GtkWidget* toGtkWidget(void* p) {
	return (GTK_WIDGET(p));
}
*/
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	ggtk "github.com/tkerber/golem/gtk"
)

// init registers a type marshaler for WebViews to glib.
//
// NOTE: This is only to bypass an unref bug/crash related to the type
// marshalers. To bypass is, web views are simply marshalled to "false".
//
// Unfortunately this means that they have to be kept track of manually
// in callbacks (by means of closures).
//
// If I ever think of a better solution, this may be changed, but for now no
// crashes are preferable.
func init() {
	glib.RegisterGValueMarshalers([]glib.TypeMarshaler{
		glib.TypeMarshaler{
			glib.Type(C.webkit_web_view_get_type()),
			func(ptr uintptr) (interface{}, error) {
				return false, nil
			},
		},
	})
}

// WebView represents a WebKitWebView widget.
type WebView struct {
	gtk.Container
	// The settings of the WebView, may be nil if they were never set or
	// retrieved.
	settings *Settings
	// The back forward list of the WebView, may be nil if it was never
	// accessed.
	bfl *BackForwardList
	// The find controller of the WebView, may be nil if it was never accessed.
	findController *FindController
}

// NewWebView creates and returns a new webkit webview.
func NewWebView() (*WebView, error) {
	w := C.webkit_web_view_new()
	if w == nil {
		return nil, errNilPtr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(w))}
	webView := wrapWebView(obj)
	obj.RefSink()
	runtime.SetFinalizer(obj, func(o *glib.Object) {
		ggtk.GlibMainContextInvoke(o.Unref)
	})
	return webView, nil
}

// NewWebViewWithUserContentManager creates a new WebView, using a specific
// UserContentManager.
func NewWebViewWithUserContentManager(
	ucm *UserContentManager) (*WebView, error) {

	w := C.webkit_web_view_new_with_user_content_manager(
		(*C.WebKitUserContentManager)(unsafe.Pointer(ucm.Native())))
	if w == nil {
		return nil, errNilPtr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(w))}
	webView := wrapWebView(obj)
	obj.RefSink()
	runtime.SetFinalizer(obj, func(o *glib.Object) {
		ggtk.GlibMainContextInvoke(o.Unref)
	})
	return webView, nil
}

// wrapWebView wraps a creates web view object in the appropriate classes.
func wrapWebView(obj *glib.Object) *WebView {
	return &WebView{
		gtk.Container{gtk.Widget{glib.InitiallyUnowned{obj}}},
		nil,
		nil,
		nil,
	}
}

// native retrieves (a properly casted) pointer the native C WebKitWebView.
func (w *WebView) native() *C.WebKitWebView {
	return (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
}

// GetPageID gets the ID of the web page corresponding to the web view.
func (w *WebView) GetPageID() uint64 {
	return uint64(C.webkit_web_view_get_page_id(w.native()))
}

// LoadURI requests loading of the speicified URI string.
func (w *WebView) LoadURI(uri string) {
	cURI := (*C.gchar)(C.CString(uri))
	defer C.free(unsafe.Pointer(cURI))
	C.webkit_web_view_load_uri(w.native(), cURI)
}

// LoadRequest loads a specified URI request.
func (w *WebView) LoadRequest(req *URIRequest) {
	C.webkit_web_view_load_request(
		w.native(),
		(*C.WebKitURIRequest)(unsafe.Pointer(req.Native())))
}

// IsLoading checks if a WebView is currently loading.
func (w *WebView) IsLoading() bool {
	return gobool(C.webkit_web_view_is_loading(w.native()))
}

// Reload request the WebView to reload.
func (w *WebView) Reload() {
	C.webkit_web_view_reload(w.native())
}

// ReloadBypassCache request the WebView to reload, bypassing the cache..
func (w *WebView) ReloadBypassCache() {
	C.webkit_web_view_reload_bypass_cache(w.native())
}

// GetEstimatedLoadProgress gets an estimation for the progress of a load
// operation.
func (w *WebView) GetEstimatedLoadProgress() float64 {
	return float64(C.webkit_web_view_get_estimated_load_progress(w.native()))
}

// GetTitle gets the webviews current title.
func (w *WebView) GetTitle() string {
	cstr := C.webkit_web_view_get_title(w.native())
	return C.GoString((*C.char)(cstr))
}

// GetURI gets the currently displayed URI.
func (w *WebView) GetURI() string {
	cstr := C.webkit_web_view_get_uri(w.native())
	return C.GoString((*C.char)(cstr))
}

// GetFavicon retrieves the pointer to the cairo_surface_t of the favicon.
//
// Returns an error if favicon is nil.
func (w *WebView) GetFavicon() (uintptr, error) {
	favicon := C.webkit_web_view_get_favicon(w.native())
	if favicon == nil {
		return 0, errNilPtr
	}
	return uintptr(unsafe.Pointer(favicon)), nil
}

// CanGoBack checks whether it is possible to currently go back.
func (w *WebView) CanGoBack() bool {
	return gobool(C.webkit_web_view_can_go_back(w.native()))
}

// GoBack goes back one step in browser history.
func (w *WebView) GoBack() {
	C.webkit_web_view_go_back(w.native())
}

// CanGoForward checks whether it is possible to currently go forward.
func (w *WebView) CanGoForward() bool {
	return gobool(C.webkit_web_view_can_go_forward(w.native()))
}

// GoForward goes forward one step in browser history.
func (w *WebView) GoForward() {
	C.webkit_web_view_go_forward(w.native())
}

// GoToBackForwardListItem goes to a specified item in the web views BFL.
func (w *WebView) GoToBackForwardListItem(i *BackForwardListItem) {
	C.webkit_web_view_go_to_back_forward_list_item(
		w.native(),
		(*C.WebKitBackForwardListItem)(unsafe.Pointer(i.Native())))
}

// GetBackForwardList gets the views list of back/forward steps in history.
//
// Note that this call is fairly expensive and takes several conversions.
// Keep a reference if you use it more often.
func (w *WebView) GetBackForwardList() *BackForwardList {
	if w.bfl != nil {
		return w.bfl
	}
	bfl := C.webkit_web_view_get_back_forward_list(w.native())
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(bfl))}
	obj.RefSink()
	runtime.SetFinalizer(obj, func(o *glib.Object) {
		ggtk.GlibMainContextInvoke(o.Unref)
	})
	w.bfl = &BackForwardList{obj}
	return w.bfl
}

// SetSettings sets the settings used for this WebView.
func (w *WebView) SetSettings(s *Settings) {
	w.settings = s
	C.webkit_web_view_set_settings(
		w.native(),
		(*C.WebKitSettings)(unsafe.Pointer(s.Native())))
}

// GetSettings retrieves the settings used for this WebView.
func (w *WebView) GetSettings() *Settings {
	if w.settings == nil {
		w.settings = wrapSettings(C.webkit_web_view_get_settings(w.native()))
	}
	return w.settings
}

// LoadAlternateHTML loads html into the web view, with a given uri.
//
// baseURI is used to resolve relative paths in the html.
func (w *WebView) LoadAlternateHTML(
	content []byte,
	contentURI, baseURI string) {

	ccont := (*C.gchar)(C.CString(string(content)))
	defer C.free(unsafe.Pointer(ccont))
	ccuri := (*C.gchar)(C.CString(contentURI))
	defer C.free(unsafe.Pointer(ccuri))
	cburi := (*C.gchar)(C.CString(baseURI))
	defer C.free(unsafe.Pointer(cburi))
	C.webkit_web_view_load_alternate_html(w.native(), ccont, ccuri, cburi)
}

// GetFindController retrieves the find controller used to control searches in
// this web view.
func (w *WebView) GetFindController() *FindController {
	if w.findController == nil {
		cptr := C.webkit_web_view_get_find_controller(w.native())
		w.findController = &FindController{&glib.Object{
			glib.ToGObject(unsafe.Pointer(cptr))}}
		w.findController.Object.RefSink()
		runtime.SetFinalizer(w.findController.Object, (*glib.Object).Unref)
	}
	return w.findController
}
