package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <stdlib.h>
// #include <webkit2/webkit2.h>
/*

extern void
cgoURISchemeRequestCallback(WebKitURISchemeRequest *req, gpointer f);

static inline void
go_webkit_web_context_register_uri_scheme(
		WebKitWebContext *c,
		gchar *scheme,
		gpointer callback) {
	webkit_web_context_register_uri_scheme(
		c,
		scheme,
		cgoURISchemeRequestCallback,
		callback,
		NULL);
}

*/
import "C"
import (
	"unsafe"
)

var DefaultWebContext = &WebContext{
	uintptr(unsafe.Pointer(C.webkit_web_context_get_default())),
}

type WebContext struct {
	native uintptr
}

func (c *WebContext) SetWebExtensionsDirectory(to string) {
	cstr := C.CString(to)
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_web_context_set_web_extensions_directory(
		(*C.WebKitWebContext)(unsafe.Pointer(c.native)),
		(*C.gchar)(cstr))
}

func (c *WebContext) RegisterURIScheme(scheme string,
	callback *func(req *URISchemeRequest)) {

	cstr := C.CString(scheme)
	defer C.free(unsafe.Pointer(cstr))
	C.go_webkit_web_context_register_uri_scheme(
		(*C.WebKitWebContext)(unsafe.Pointer(c.native)),
		(*C.gchar)(cstr),
		C.gpointer(unsafe.Pointer(callback)))
}

//export cgoURISchemeRequestCallback
func cgoURISchemeRequestCallback(req *C.WebKitURISchemeRequest, f C.gpointer) {
	goFunc := (*func(req *URISchemeRequest))(unsafe.Pointer(f))
	goReq := &URISchemeRequest{uintptr(unsafe.Pointer(req))}
	(*goFunc)(goReq)
}
