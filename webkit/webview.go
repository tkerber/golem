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
import "go/build"
import "path/filepath"
import "os"

// WebView represents a webkit webview widget.
type WebView struct {
	gtk.Container
}

func init() {
	context := C.webkit_web_context_get_default()
	// TODO figure out a better way to reference this. (i.e. without the source)
	extenPath := ""
	for src := range build.Default.SrcDirs() {
		p := filepath.Join(src, "github.com", "tkerber", "golem", "web_extension")
		if _, err := os.Stat(p); err != nil {
			extenPath = p
			break
		}
	}
	if extenPath == "" {
		panic("Failed to find source files!")
	}

	cstr := C.CString(extenPath)
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_web_context_set_web_extensions_directory(context, (*C.gchar)(cstr))
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
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return webView, nil
}

func wrapWebView(obj *glib.Object) *WebView {
	return &WebView{gtk.Container{gtk.Widget{glib.InitiallyUnowned{obj}}}}
}

// LoadURI requests loading of the speicified URI string.
func (w *WebView) LoadURI(uri string) {
	cURI := (*C.gchar)(C.CString(uri))
	defer C.free(unsafe.Pointer(cURI))
	webViewPtr := (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
	C.webkit_web_view_load_uri(webViewPtr, cURI)
}

// IsLoading checks if a WebView is currently loading.
func (w *WebView) IsLoading() bool {
	webViewPtr := (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
	if C.webkit_web_view_is_loading(webViewPtr) != 0 {
		return true
	}
	return false
}

// Reload request the WebView to reload.
func (w *WebView) Reload() {
	webViewPtr := (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
	C.webkit_web_view_reload(webViewPtr)
}

// GetEstimatedLoadProgress gets an estimation for the progress of a load
// operation.
func (w *WebView) GetEstimatedLoadProgress() float64 {
	webViewPtr := (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
	return float64(C.webkit_web_view_get_estimated_load_progress(webViewPtr))
}
