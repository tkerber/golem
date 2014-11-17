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
import "github.com/conformal/gotk3/gtk"
import "github.com/conformal/gotk3/glib"
import "unsafe"
import "runtime"

// WebView represents a webkit webview widget.
type WebView struct {
	gtk.Widget
}

// NewWebView creates and returns a new webkit webview.
func NewWebView() (*WebView, error) {
	w := C.webkit_web_view_new()
	if w == nil {
		return nil, nilPtrErr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(w))}
	webView := wrapWebView(obj)
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return webView, nil
}

func wrapWebView(obj *glib.Object) *WebView {
	return &WebView{gtk.Widget{glib.InitiallyUnowned{obj}}}
}

// LoadURI requests loading of the speicified URI string.
func (w *WebView) LoadURI(uri string) {
	cURI := (*C.gchar)(C.CString(uri))
	defer C.free(unsafe.Pointer(cURI))
	webViewPtr := (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
	C.webkit_web_view_load_uri(webViewPtr, cURI)
}
